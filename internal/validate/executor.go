package validate

import (
	"context"
	"fmt"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/batch"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// Executor handles the validate operation orchestration
type Executor struct {
	reporter      reporter.Reporter
	clientFactory func(context.Context, string) (*aws.Client, error)
}

// NewExecutor creates a new validate executor
func NewExecutor(rep reporter.Reporter) *Executor {
	return &Executor{
		reporter:      rep,
		clientFactory: aws.NewClient,
	}
}

// NewExecutorWithFactory creates a new validate executor with a custom client factory
// This is useful for testing with mock clients
func NewExecutorWithFactory(rep reporter.Reporter, factory func(context.Context, string) (*aws.Client, error)) *Executor {
	return &Executor{
		reporter:      rep,
		clientFactory: factory,
	}
}

// RunOnTarget validates one pre-loaded target's data file against the profile's
// schema, without creating a configuration version (read-only). It is invoked
// from cmd/validate.go via batch.Orchestrator (single-target runs flow through
// the orchestrator with a single goroutine). The supplied TargetReporter is
// always driven to a terminal state (Done / Fail).
//
// Schema selection for Freeform JSON: the profile's JSON_SCHEMA validator is
// fetched from AWS (the same validator AWS enforces at deploy time, retrieved
// for free during resource resolution). FeatureFlags use the built-in schema.
// YAML/text and LAMBDA validators are not schema-checked — only syntax is
// verified.
func (e *Executor) RunOnTarget(ctx context.Context, t *batch.Target, tr reporter.TargetReporter) error {
	awsClient, err := e.clientFactory(ctx, t.Config.Region)
	if err != nil {
		tr.Fail(err)
		return fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	cfg := t.Config

	tr.SetPhase("preparing", "")

	resolver := aws.NewResolver(awsClient)
	resources, err := resolver.ResolveAll(ctx, cfg.Application, cfg.ConfigurationProfile, cfg.Environment, "")
	if err != nil {
		tr.Fail(err)
		return fmt.Errorf("failed to resolve resources: %w", err)
	}

	data, err := config.LoadDataFile(cfg.DataFile)
	if err != nil {
		tr.Fail(err)
		return fmt.Errorf("failed to load local configuration file: %w", err)
	}

	contentType := config.DetermineContentType(resources.Profile.Type, cfg.DataFile)

	schema, summary := resolveSchema(resources.Profile, contentType)

	if err := config.ValidateConfigData(data, resources.Profile.Type, contentType, schema); err != nil {
		tr.Fail(err)
		return fmt.Errorf("validation failed: %w", err)
	}

	tr.Done(summary)
	return nil
}

// resolveSchema selects the JSON schema (if any) to validate against and returns
// a human-readable summary describing what was checked. The schema is the
// profile's JSON_SCHEMA validator, already fetched from AWS during resource
// resolution — the same validator AWS enforces when a version is created.
func resolveSchema(profile *aws.ProfileInfo, contentType string) ([]byte, string) {
	if profile.Type == config.ProfileTypeFeatureFlags {
		return nil, "valid (feature flags schema)"
	}

	// Freeform: schema validation only applies to JSON.
	if contentType != config.ContentTypeJSON {
		return nil, "valid (syntax only)"
	}

	if content, ok := profile.JSONSchemaContent(); ok {
		return []byte(content), "valid (remote schema)"
	}

	if profile.HasLambdaValidator() {
		return nil, "valid (syntax only; lambda validator not checked)"
	}
	return nil, "valid (syntax only; no schema)"
}
