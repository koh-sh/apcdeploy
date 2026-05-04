// Package deploywait holds helpers shared by deploy-shape commands
// (currently `run` and `edit`) that wait on AppConfig deployments and
// surface progress through Reporter.Targets.
//
// The package sits between `internal/aws` (deployment polling /
// timestamps) and `internal/reporter` (Targets primitive). Placing it
// in either parent would introduce a reverse dependency: `aws` would
// gain a UI dependency on Reporter, or `cli`/`reporter` would gain an
// AWS-domain dependency on DeploymentState. Both are worse than a
// dedicated neutral package.
package deploywait

import (
	"context"
	"time"

	"github.com/koh-sh/apcdeploy/internal/aws"
)

// AWSElapsedForDeploy returns the AWS-recorded deploy phase elapsed:
// BAKE_TIME_STARTED.OccurredAt - StartedAt. AppConfig records both
// timestamps at microsecond precision when the corresponding state
// transitions occur, so this is the authoritative "actual deploy
// time" — including a few seconds of state-machine overhead that AWS
// does not document but consistently exhibits (the public
// GetDeployment example shows ~3.5 s on top of a 15-minute strategy).
//
// Falls back to time.Since(fallback) when the fetch fails, the
// EventLog hasn't surfaced BAKE_TIME_STARTED yet, or StartedAt is
// nil. Best-effort: wall-clock is the only sensible signal when
// AWS-side data is unavailable.
func AWSElapsedForDeploy(ctx context.Context, client *aws.Client, appID, envID string, deploymentNumber int32, fallback time.Time) time.Duration {
	details, err := aws.GetDeploymentDetails(ctx, client, appID, envID, deploymentNumber)
	if err == nil && details.StartedAt != nil {
		if bakeStart := aws.BakeTimeStartedAt(details); bakeStart != nil {
			return bakeStart.Sub(*details.StartedAt)
		}
	}
	return time.Since(fallback)
}

// AWSElapsedForBake returns the AWS-recorded total elapsed:
// CompletedAt - StartedAt. AWS does not emit a separate
// BAKE_TIME_COMPLETED event, so this is the only way to get the
// authoritative AWS-side total (deploy + bake monitoring + completion
// overhead).
//
// Falls back to time.Since(fallback) on fetch failure or when either
// timestamp is nil.
func AWSElapsedForBake(ctx context.Context, client *aws.Client, appID, envID string, deploymentNumber int32, fallback time.Time) time.Duration {
	details, err := aws.GetDeploymentDetails(ctx, client, appID, envID, deploymentNumber)
	if err == nil && details.StartedAt != nil && details.CompletedAt != nil {
		return details.CompletedAt.Sub(*details.StartedAt)
	}
	return time.Since(fallback)
}
