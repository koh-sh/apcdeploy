package aws

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
)

// CheckOngoingDeployment checks if there is an ongoing deployment
func (c *Client) CheckOngoingDeployment(ctx context.Context, applicationID, environmentID string) (bool, *types.DeploymentSummary, error) {
	deployments, err := c.ListAllDeployments(ctx, applicationID, environmentID)
	if err != nil {
		return false, nil, wrapAWSError(err, "failed to list deployments")
	}

	// Check for ongoing (in-flight) deployments. The transitional states
	// are DEPLOYING, BAKING, VALIDATING and ROLLING_BACK. The terminal
	// states COMPLETE, ROLLED_BACK and REVERTED are deliberately excluded:
	// they persist in the deployment history indefinitely, so treating one
	// as "ongoing" would permanently block run/edit (false "already in
	// progress") and make rollback try to stop a finished deployment.
	for _, deployment := range deployments {
		if deployment.State == types.DeploymentStateDeploying ||
			deployment.State == types.DeploymentStateBaking ||
			deployment.State == types.DeploymentStateValidating ||
			deployment.State == types.DeploymentStateRollingBack {
			return true, &deployment, nil
		}
	}

	return false, nil, nil
}

// CreateHostedConfigurationVersion creates a new hosted configuration version
func (c *Client) CreateHostedConfigurationVersion(
	ctx context.Context,
	applicationID, profileID string,
	content []byte,
	contentType, description string,
) (int32, error) {
	input := &appconfig.CreateHostedConfigurationVersionInput{
		ApplicationId:          aws.String(applicationID),
		ConfigurationProfileId: aws.String(profileID),
		Content:                content,
		ContentType:            aws.String(contentType),
	}

	if description != "" {
		input.Description = aws.String(description)
	}

	output, err := c.appConfig.CreateHostedConfigurationVersion(ctx, input)
	if err != nil {
		return 0, wrapAWSError(err, "failed to create hosted configuration version")
	}

	return output.VersionNumber, nil
}

// StartDeployment starts a new deployment
func (c *Client) StartDeployment(
	ctx context.Context,
	applicationID, environmentID, profileID, strategyID string,
	versionNumber int32,
	description string,
) (int32, error) {
	versionStr := strconv.FormatInt(int64(versionNumber), 10)

	input := &appconfig.StartDeploymentInput{
		ApplicationId:          aws.String(applicationID),
		EnvironmentId:          aws.String(environmentID),
		ConfigurationProfileId: aws.String(profileID),
		ConfigurationVersion:   aws.String(versionStr),
		DeploymentStrategyId:   aws.String(strategyID),
	}

	if description != "" {
		input.Description = aws.String(description)
	}

	output, err := c.appConfig.StartDeployment(ctx, input)
	if err != nil {
		return 0, wrapAWSError(err, "failed to start deployment")
	}

	return output.DeploymentNumber, nil
}

// StopDeployment stops an in-progress deployment
func (c *Client) StopDeployment(
	ctx context.Context,
	applicationID, environmentID string,
	deploymentNumber int32,
) error {
	input := &appconfig.StopDeploymentInput{
		ApplicationId:    aws.String(applicationID),
		EnvironmentId:    aws.String(environmentID),
		DeploymentNumber: &deploymentNumber,
	}

	_, err := c.appConfig.StopDeployment(ctx, input)
	if err != nil {
		return wrapAWSError(err, "failed to stop deployment")
	}

	return nil
}

// rolledBackError formats the error returned when a deployment has reached
// the ROLLED_BACK state. It pulls the most recent rollback description from
// the event log when available, falling back to a generic message otherwise.
// Shared by waitForDeploymentWithCondition and WaitForBakingComplete so the
// rollback message is consistent across phases.
func rolledBackError(eventLog []types.DeploymentEvent) error {
	if reason := ExtractRollbackReason(eventLog); reason != "" {
		return fmt.Errorf("deployment was rolled back: %s", reason)
	}
	return fmt.Errorf("deployment was rolled back")
}

// ExtractRollbackReason returns the most recent non-empty rollback description
// from a deployment event log, or "" if none is present.
func ExtractRollbackReason(eventLog []types.DeploymentEvent) string {
	// Look for rollback events in reverse order (most recent first)
	for i := len(eventLog) - 1; i >= 0; i-- {
		event := eventLog[i]
		// Check for rollback-related events
		if event.EventType == types.DeploymentEventTypeRollbackStarted ||
			event.EventType == types.DeploymentEventTypeRollbackCompleted {
			if event.Description != nil && *event.Description != "" {
				return *event.Description
			}
		}
	}
	return ""
}

// normalizeWaitError decides what a polling wait loop reports once it stops.
//
// The loop owns a context carrying the caller-supplied timeout, so once that
// context is done the reason it is done outranks whatever the last poll
// produced (an in-flight GetDeployment fails with a transport-level error the
// moment the context is cancelled, which says nothing useful):
//
//   - deadline reached — the loop's own timeout expired, so keep the
//     "<phase> timed out after <timeout>" message.
//   - anything else — in practice the parent context cancelled by Ctrl+C, so
//     return ctx.Err() and let cmd/root.go's errors.Is(err, context.Canceled)
//     branch report "cancelled by user" and exit 130 instead of a fabricated
//     timeout that never elapsed.
//
// With a live context err passes through unchanged, which is how genuine
// deployment failures (rollback, unexpected state) keep their message.
func normalizeWaitError(ctx context.Context, err error, phase string, timeout time.Duration) error {
	ctxErr := ctx.Err()
	if ctxErr == nil {
		return err
	}
	if errors.Is(ctxErr, context.DeadlineExceeded) {
		return fmt.Errorf("%s timed out after %v", phase, timeout)
	}
	return ctxErr
}

// DeploymentTickFunc is invoked on each polling tick of a deployment wait
// loop, with the current state, percentage-complete reported by AWS, and the
// configured deployment duration (DeploymentDurationInMinutes converted to a
// time.Duration). totalDuration is zero when the deployment strategy reports
// no deploy phase (e.g. AppConfig.AllAtOnce). Callers use it to drive live
// progress UI and surface remaining-time estimates; nil is allowed.
type DeploymentTickFunc func(state types.DeploymentState, percent float64, totalDuration time.Duration)

// waitForDeploymentWithCondition is a generic wait function that polls deployment status
// until the provided checkComplete function returns true or an error occurs
func (c *Client) waitForDeploymentWithCondition(
	ctx context.Context,
	applicationID, environmentID string,
	deploymentNumber int32,
	timeout time.Duration,
	checkComplete func(types.DeploymentState) bool,
	onTick DeploymentTickFunc,
) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Use configured polling interval, default to 5s if not set
	pollingInterval := c.PollingInterval
	if pollingInterval == 0 {
		pollingInterval = 5 * time.Second
	}
	ticker := time.NewTicker(pollingInterval)
	defer ticker.Stop()

	checkDeployment := func() (bool, error) {
		input := &appconfig.GetDeploymentInput{
			ApplicationId:    aws.String(applicationID),
			EnvironmentId:    aws.String(environmentID),
			DeploymentNumber: &deploymentNumber,
		}

		output, err := c.appConfig.GetDeployment(ctx, input)
		if err != nil {
			return false, wrapAWSError(err, "failed to get deployment status")
		}

		if onTick != nil {
			var pct float64
			if output.PercentageComplete != nil {
				pct = float64(*output.PercentageComplete)
			}
			totalDuration := time.Duration(output.DeploymentDurationInMinutes) * time.Minute
			onTick(output.State, pct, totalDuration)
		}

		// Check deployment state
		switch output.State {
		case types.DeploymentStateComplete, types.DeploymentStateBaking:
			// Check if deployment is complete based on the provided condition
			if checkComplete(output.State) {
				return true, nil
			}
			// Still in progress (e.g., BAKING when waiting for COMPLETE)
			return false, nil

		case types.DeploymentStateRolledBack:
			return false, rolledBackError(output.EventLog)

		case types.DeploymentStateDeploying, types.DeploymentStateValidating:
			// Still in progress
			return false, nil

		case types.DeploymentStateRollingBack:
			// Auto-rollback in progress; keep polling until ROLLED_BACK
			// so that rolledBackError can surface the reason from the event log.
			return false, nil

		case types.DeploymentStateReverted:
			// Terminal state: deployment was explicitly reverted to a previous version.
			return false, fmt.Errorf("deployment was reverted")

		default:
			// Unexpected state
			return false, fmt.Errorf("unexpected deployment state: %s", output.State)
		}
	}

	// Check immediately first
	complete, err := checkDeployment()
	if err != nil {
		return normalizeWaitError(ctx, err, "deployment", timeout)
	}
	if complete {
		return nil
	}

	// Then check periodically
	for {
		select {
		case <-ctx.Done():
			return normalizeWaitError(ctx, ctx.Err(), "deployment", timeout)
		case <-ticker.C:
			complete, err := checkDeployment()
			if err != nil {
				return normalizeWaitError(ctx, err, "deployment", timeout)
			}
			if complete {
				return nil
			}
		}
	}
}

// WaitForDeploymentPhase waits for a deployment to reach a specific phase.
// If waitForBaking is false, it waits until the deployment enters BAKING state (deploy phase complete).
// If waitForBaking is true, it waits until the deployment reaches COMPLETE state (baking phase complete).
// onTick is invoked on each polling tick with the current state and percentage; nil is allowed.
func (c *Client) WaitForDeploymentPhase(
	ctx context.Context,
	applicationID, environmentID string,
	deploymentNumber int32,
	waitForBaking bool,
	timeout time.Duration,
	onTick DeploymentTickFunc,
) error {
	return c.waitForDeploymentWithCondition(
		ctx,
		applicationID,
		environmentID,
		deploymentNumber,
		timeout,
		func(state types.DeploymentState) bool {
			// Always complete if deployment is COMPLETE
			if state == types.DeploymentStateComplete {
				return true
			}
			// If waiting for deploy phase only, BAKING means deploy is complete
			if state == types.DeploymentStateBaking && !waitForBaking {
				return true
			}
			return false
		},
		onTick,
	)
}

// BakeTickFunc is invoked on each polling tick of WaitForBakingComplete with
// the elapsed time since the wait started locally and the configured bake
// duration (FinalBakeTimeInMinutes). When total is zero the deployment has
// no bake window. Callers use it to drive live UI (typically a spinner with
// a "(~N min left)" countdown label); nil is allowed.
//
// Bake is fundamentally a wait, not a quantified rollout — AppConfig does
// not surface a bake-phase percentage. Callers that want a "% done" feeling
// can derive elapsed/total themselves, but the canonical UX is a spinner
// (no bar) since nothing is being deployed during bake.
type BakeTickFunc func(elapsed, total time.Duration)

// WaitForBakingComplete waits for an in-bake deployment to reach COMPLETE.
// onTick is invoked on each polling tick with (elapsed, total), where
// elapsed is the duration since this function was called locally and total
// is the deployment's FinalBakeTimeInMinutes. AppConfig does not surface a
// bake-phase percentage of its own, so callers that want a "% done" feeling
// must derive it from elapsed/total themselves.
//
// When the bake duration is zero, onTick is still invoked once on COMPLETE
// with (0, 0) so the caller can finalize its UI uniformly.
//
// The deployment is expected to already be in BAKING (or COMPLETE) state.
// DEPLOYING or other states are treated as unexpected and yield an error.
func (c *Client) WaitForBakingComplete(
	ctx context.Context,
	applicationID, environmentID string,
	deploymentNumber int32,
	timeout time.Duration,
	onTick BakeTickFunc,
) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	pollingInterval := c.PollingInterval
	if pollingInterval == 0 {
		pollingInterval = 5 * time.Second
	}
	ticker := time.NewTicker(pollingInterval)
	defer ticker.Stop()

	bakeStart := time.Now()

	checkDeployment := func() (bool, error) {
		input := &appconfig.GetDeploymentInput{
			ApplicationId:    aws.String(applicationID),
			EnvironmentId:    aws.String(environmentID),
			DeploymentNumber: &deploymentNumber,
		}

		output, err := c.appConfig.GetDeployment(ctx, input)
		if err != nil {
			return false, wrapAWSError(err, "failed to get deployment status")
		}

		bakeDuration := time.Duration(output.FinalBakeTimeInMinutes) * time.Minute

		switch output.State {
		case types.DeploymentStateComplete:
			if onTick != nil {
				onTick(bakeDuration, bakeDuration)
			}
			return true, nil

		case types.DeploymentStateBaking:
			if onTick != nil {
				onTick(time.Since(bakeStart), bakeDuration)
			}
			return false, nil

		case types.DeploymentStateRolledBack:
			return false, rolledBackError(output.EventLog)

		case types.DeploymentStateRollingBack:
			// Auto-rollback in progress; keep polling until ROLLED_BACK
			// so that rolledBackError can surface the reason from the event log.
			return false, nil

		case types.DeploymentStateReverted:
			// Terminal state: deployment was explicitly reverted to a previous version.
			return false, fmt.Errorf("deployment was reverted")

		default:
			return false, fmt.Errorf("unexpected deployment state during bake wait: %s", output.State)
		}
	}

	complete, err := checkDeployment()
	if err != nil {
		return normalizeWaitError(ctx, err, "bake phase", timeout)
	}
	if complete {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return normalizeWaitError(ctx, ctx.Err(), "bake phase", timeout)
		case <-ticker.C:
			complete, err := checkDeployment()
			if err != nil {
				return normalizeWaitError(ctx, err, "bake phase", timeout)
			}
			if complete {
				return nil
			}
		}
	}
}

// DeploymentInfo contains information about a deployment
type DeploymentInfo struct {
	DeploymentNumber     int32
	ConfigurationVersion string
	DeploymentStrategyID string
	State                types.DeploymentState
	Description          string
}

// GetLatestDeployment retrieves the latest deployment for the specified configuration profile
// This function skips ROLLED_BACK deployments and returns the last successful or in-progress deployment
func GetLatestDeployment(ctx context.Context, client *Client, applicationID, environmentID, profileID string) (*DeploymentInfo, error) {
	return getLatestDeploymentInternal(ctx, client, applicationID, environmentID, profileID, true)
}

// GetLatestDeploymentIncludingRollback retrieves the latest deployment for the specified configuration profile
// This function includes ROLLED_BACK deployments and returns the absolute latest deployment
func GetLatestDeploymentIncludingRollback(ctx context.Context, client *Client, applicationID, environmentID, profileID string) (*DeploymentInfo, error) {
	return getLatestDeploymentInternal(ctx, client, applicationID, environmentID, profileID, false)
}

// getLatestDeploymentInternal is the internal implementation for retrieving the latest deployment.
//
// Deployments are sorted descending by DeploymentNumber before scanning so that the
// newest deployment is examined first. Any GetDeployment error causes an immediate
// return — if we cannot fetch details for a deployment we have no way to determine
// whether it is the true latest for the requested profile, so silently skipping it
// could cause callers (run / diff / pull) to compare against an older version.
func getLatestDeploymentInternal(ctx context.Context, client *Client, applicationID, environmentID, profileID string, skipRolledBack bool) (*DeploymentInfo, error) {
	deployments, err := client.ListAllDeployments(ctx, applicationID, environmentID)
	if err != nil {
		return nil, wrapAWSError(err, "failed to list deployments")
	}

	// Sort newest-first so we examine the highest deployment number first.
	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].DeploymentNumber > deployments[j].DeploymentNumber
	})

	// Find the latest deployment for this configuration profile.
	// We need to get full deployment details to access ConfigurationProfileId.
	for i := range deployments {
		summary := &deployments[i]

		getInput := &appconfig.GetDeploymentInput{
			ApplicationId:    aws.String(applicationID),
			EnvironmentId:    aws.String(environmentID),
			DeploymentNumber: &summary.DeploymentNumber,
		}

		deployment, err := client.appConfig.GetDeployment(ctx, getInput)
		if err != nil {
			// Fail fast: if any GetDeployment call errors we cannot be certain
			// which deployment is truly the latest for this profile, so surfacing
			// the error is safer than silently returning an older deployment.
			return nil, wrapAWSError(err, "failed to fetch deployment details")
		}

		// Skip deployments that belong to a different configuration profile.
		if aws.ToString(deployment.ConfigurationProfileId) != profileID {
			continue
		}

		// Skip ROLLED_BACK deployments if requested.
		if skipRolledBack && deployment.State == types.DeploymentStateRolledBack {
			continue
		}

		return &DeploymentInfo{
			DeploymentNumber:     summary.DeploymentNumber,
			ConfigurationVersion: aws.ToString(deployment.ConfigurationVersion),
			DeploymentStrategyID: aws.ToString(deployment.DeploymentStrategyId),
			State:                deployment.State,
			Description:          aws.ToString(deployment.Description),
		}, nil
	}

	return nil, nil
}

// GetHostedConfigurationVersion retrieves the content of a specific hosted configuration version
func GetHostedConfigurationVersion(ctx context.Context, client *Client, applicationID, profileID, versionNumber string) ([]byte, error) {
	version, err := strconv.ParseInt(versionNumber, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid version number: %s", versionNumber)
	}
	input := &appconfig.GetHostedConfigurationVersionInput{
		ApplicationId:          aws.String(applicationID),
		ConfigurationProfileId: aws.String(profileID),
		VersionNumber:          aws.Int32(int32(version)),
	}

	output, err := client.appConfig.GetHostedConfigurationVersion(ctx, input)
	if err != nil {
		return nil, wrapAWSError(err, "failed to get hosted configuration version")
	}

	return output.Content, nil
}

// DeploymentDetails contains detailed information about a deployment
type DeploymentDetails struct {
	DeploymentNumber       int32
	ConfigurationProfileID string
	ConfigurationVersion   string
	DeploymentStrategyID   string
	DeploymentStrategyName string
	State                  types.DeploymentState
	Description            string
	EventLog               []types.DeploymentEvent
	StartedAt              *time.Time
	CompletedAt            *time.Time
	PercentageComplete     float32
	GrowthFactor           float32
	// DeploymentDurationInMinutes is the strategy's configured deploy
	// phase duration (integer minutes — AppConfig API constraint).
	// AppConfig schedules the rollout to fit within this window, so
	// the actual deploy phase always takes exactly this many minutes
	// modulo a few seconds of state-machine overhead.
	DeploymentDurationInMinutes int32
	// FinalBakeTimeInMinutes is the strategy's configured bake phase
	// duration (integer minutes — same constraint).
	FinalBakeTimeInMinutes int32
}

// BakeTimeStartedAt returns the OccurredAt timestamp of the
// BAKE_TIME_STARTED event in details.EventLog, or nil if no such event
// is present (yet). AppConfig records this when the deploy phase ends
// and the bake phase begins; subtracting StartedAt from it yields the
// AWS-recorded deploy phase elapsed (jitter included).
func BakeTimeStartedAt(details *DeploymentDetails) *time.Time {
	if details == nil {
		return nil
	}
	for _, e := range details.EventLog {
		if e.EventType == types.DeploymentEventTypeBakeTimeStarted && e.OccurredAt != nil {
			t := *e.OccurredAt
			return &t
		}
	}
	return nil
}

// GetDeploymentDetails retrieves detailed information about a specific deployment
func GetDeploymentDetails(ctx context.Context, client *Client, applicationID, environmentID string, deploymentNumber int32) (*DeploymentDetails, error) {
	input := &appconfig.GetDeploymentInput{
		ApplicationId:    aws.String(applicationID),
		EnvironmentId:    aws.String(environmentID),
		DeploymentNumber: &deploymentNumber,
	}

	output, err := client.appConfig.GetDeployment(ctx, input)
	if err != nil {
		return nil, wrapAWSError(err, "failed to get deployment details")
	}

	var percentageComplete float32
	if output.PercentageComplete != nil {
		percentageComplete = *output.PercentageComplete
	}

	var growthFactor float32
	if output.GrowthFactor != nil {
		growthFactor = *output.GrowthFactor
	}

	details := &DeploymentDetails{
		DeploymentNumber:            output.DeploymentNumber,
		ConfigurationProfileID:      aws.ToString(output.ConfigurationProfileId),
		ConfigurationVersion:        aws.ToString(output.ConfigurationVersion),
		DeploymentStrategyID:        aws.ToString(output.DeploymentStrategyId),
		State:                       output.State,
		Description:                 aws.ToString(output.Description),
		EventLog:                    output.EventLog,
		StartedAt:                   output.StartedAt,
		CompletedAt:                 output.CompletedAt,
		PercentageComplete:          percentageComplete,
		GrowthFactor:                growthFactor,
		DeploymentDurationInMinutes: output.DeploymentDurationInMinutes,
		FinalBakeTimeInMinutes:      output.FinalBakeTimeInMinutes,
	}

	return details, nil
}
