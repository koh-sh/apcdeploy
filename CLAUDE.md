# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`apcdeploy` is a CLI tool for managing AWS AppConfig deployments. It enables developers to manage AppConfig resources (applications, configuration profiles, environments) as code through a declarative YAML configuration file (`apcdeploy.yml`).

## Development Rules

When implementing new features or fixing bugs, follow these absolute rules:

- **TDD (Test-Driven Development)**: Write tests before implementation
- **Code consistency**: Match existing code style and patterns
- **CI validation**: Ensure `mise run ci` passes before considering work complete
- **Test coverage**: Maintain or improve test coverage (never decrease it)

## Common Commands

Dev tools (Go toolchain, golangci-lint, gofumpt, tparse, octocov, goreleaser, terraform) are managed by [mise](https://mise.jdx.dev/) via `.mise.toml`. Run `mise install` once to provision them.

### Development

- **Install dev tools**: `mise install`
- **Build**: `mise run build` (or `go build`)
- **Run tests**: `mise run test` (uses tparse for formatted output)
- **Run single test**: `go test -run TestName ./path/to/package`
- **Lint**: `mise run lint` (uses golangci-lint v2)
- **Fix lint issues**: `mise run lint-fix`
- **Format code**: `mise run fmt` (uses gofumpt)
- **Run go fix (modernize)**: `mise run fix`
- **Generate coverage**: `mise run cov` (creates cover.html)
- **Full CI workflow**: `mise run ci` (fmt, fix, lint-fix, build, cov)
- **Upgrade managed tools**: `mise run upgrade-tools`

### Testing the CLI

```bash
# List available resources
./apcdeploy ls-resources --region us-east-1
./apcdeploy ls-resources --region us-east-1 --json
./apcdeploy ls-resources --region us-east-1 --show-strategies

# Interactive mode (recommended for init)
./apcdeploy init

# Non-interactive mode with flags
./apcdeploy init --region us-east-1 --app my-app --profile my-profile --env production

# Other commands
./apcdeploy diff -c apcdeploy.yml
./apcdeploy run -c apcdeploy.yml --wait-bake  # Wait for full deployment
./apcdeploy run -c apcdeploy.yml --wait-deploy  # Wait for deploy phase only
./apcdeploy status -c apcdeploy.yml
./apcdeploy get -c apcdeploy.yml
./apcdeploy pull -c apcdeploy.yml  # Pull latest deployed configuration to local data file
./apcdeploy rollback -c apcdeploy.yml  # Stop ongoing deployment (rollback)
./apcdeploy rollback -c apcdeploy.yml --yes  # Skip confirmation
./apcdeploy edit  # Edit deployed configuration directly in $EDITOR (no apcdeploy.yml)
./apcdeploy edit --region us-east-1 --app my-app --profile my-profile --env prod
./apcdeploy context  # Output llms.md for AI assistants

# Silent mode (suppress verbose output)
./apcdeploy ls-resources --region us-east-1 --json --silent  # silent without --json yields no stdout
./apcdeploy diff -c apcdeploy.yml --silent
./apcdeploy status -c apcdeploy.yml --silent
```

### E2E Testing

E2E tests require AWS credentials and use Terraform to provision resources:

- **Setup resources**: `mise run e2e-setup` (provisions AWS resources via Terraform)
- **Run tests**: `mise run e2e-run` (executes e2e test script)
- **Clean up**: `mise run e2e-clean` (destroys test resources)
- **Full workflow**: `mise run e2e-full` (setup, test, cleanup in one command)

## Architecture

### Command Structure (Cobra-based)

All commands follow the pattern: `cmd/<command>.go` → `internal/<command>/executor.go`

**Exception**: The `context` command is a simple utility that only outputs embedded content (`llms.md`). It does not follow the standard command structure and has no corresponding `internal/context/` directory. The implementation is entirely contained in `cmd/context.go`, with the content embedded in `main.go` and passed via `cmd.SetLLMsContent()`.

1. **cmd/**: Cobra command definitions and CLI flag parsing
   - `root.go`: Root command with global flags (`--config`, `--silent`); also hosts the `--description` shared helpers (`validateDescription` for the 1024-rune client-side limit and `resolveDescription` for the `defaultDescription` fallback) used by `run` and `edit`
   - Each command file (`init.go`, `run.go`, `diff.go`, `status.go`, `get.go`, `pull.go`, `rollback.go`, `ls_resources.go`, `edit.go`) handles CLI concerns only
   - `context.go`: Simple command that outputs embedded `llms.md` content for AI assistants (no internal package)
   - `init.go`: Supports interactive mode for resource selection; all flags are optional
   - `ls_resources.go`: Lists AppConfig resources; does not require `apcdeploy.yml`; all flags are optional
   - `rollback.go`: Stops an ongoing deployment; supports confirmation prompt
   - `edit.go`: Opens `$EDITOR` on the deployed configuration and deploys; does not require `apcdeploy.yml`; all flags are optional

2. **internal/\<command\>/**: Business logic for each command
   - `executor.go`: Main execution logic using Factory pattern for testability
   - `options.go`: Command-specific options struct
   - `workflow.go` (init, edit commands): Handles multi-step workflow including interactive prompts and resource resolution
   - Executors accept a `reporter.Reporter` for user feedback

### Core Packages

#### internal/aws

AWS AppConfig client wrapper with:

- `Client`: Wraps AWS SDK AppConfig client with polling interval configuration
- `AppConfigAPI`: Interface for AppConfig operations (enables mocking in tests)
- `client_list_paginated.go`: **Centralized list operations with pagination handling** - All AWS List APIs should use these methods
- `resolver.go`: Resolves resource names (application, profile, environment) to AWS IDs
- `deployment.go`: Deployment creation, monitoring, and rollback logic (includes `StopDeployment` method)
- `config_fetcher.go`: Provides `GetLatestDeployedConfiguration` to retrieve deployed configuration from the latest deployment; exposes `ErrNoDeployment` sentinel for callers (`pull`, `edit`) that need to detect "no prior deployment" via `errors.Is`
- Version info is injected at build time via `main.go` variables

**IMPORTANT - AWS List API Usage:**

Always use the centralized list methods in `client_list_paginated.go` instead of calling AWS SDK List APIs directly:

- `ListAllApplications()` - Lists all applications with pagination
- `ListAllConfigurationProfiles(appID)` - Lists all profiles with pagination
- `ListAllEnvironments(appID)` - Lists all environments with pagination
- `ListAllDeploymentStrategies()` - Lists all deployment strategies with pagination
- `ListAllDeployments(appID, envID)` - Lists all deployments with pagination
- `ListAllHostedConfigurationVersions(appID, profileID)` - Lists all versions with pagination

These methods automatically handle pagination to ensure all resources are retrieved, even in environments with many resources. Direct SDK calls without pagination can silently truncate results when the resource count exceeds AWS API page limits.

#### internal/config

Configuration file management:

- `types.go`: Defines `Config` struct (application, profile, environment, deployment strategy, data file path)
- `loader.go`: Loads and validates `apcdeploy.yml`, resolves relative paths
- `data.go`: Loads data files (JSON/YAML/text) and detects content type
- `normalize.go`: Normalizes JSON/YAML for consistent comparisons (removes FeatureFlags metadata); also exposes `HasContentChanged` and `NormalizeByExtension` for diff detection shared by `run`, `pull`, and `edit`
- `validate.go`: Validates configuration data before deployment (size limit + JSON/YAML syntax checks); shared by `run` and `edit`
- `generator.go`: Generates `apcdeploy.yml` from AWS resources during init

#### internal/reporter

The single output abstraction used by every command. See [Output Contract](.claude/rules/output-contract.md) for the full contract.

- `Reporter`: Interface with `Step`, `Success`, `Info`, `Warn`, `Error`, `Header`, `Box`, `Table`, `Spin`, `Checklist`, `Progress`, `Data`, `Diff`
- `internal/cli/reporter.go`: TTY-aware console implementation using lipgloss styles + bubbles spinner frames
- `internal/cli/silent_reporter.go`: Silent variant that suppresses everything except `Error` / `Data` / `Diff`
- `internal/cli/style.go`: Centralized lipgloss styles (the only place ANSI/color is defined)
- `internal/cli/factory.go`: `GetReporter(silent bool) reporter.Reporter` selects the appropriate implementation
- `internal/cli/tty.go`: TTY detection used to degrade animations and color in non-interactive environments

Executors MUST NOT call `fmt.Fprint*` directly; all output flows through `Reporter`. Executors MUST NOT branch on `opts.Silent` — Reporter selection in `cmd/root.go` handles silent semantics.

#### internal/prompt

Interactive prompt interface for user input:

- `Prompter`: Interface with `Select()`, `Input()`, and `CheckTTY()` methods for interactive operations
- `internal/prompt/huh.go`: Implementation using `huh` library for terminal UI
- `internal/prompt/tty.go`: TTY detection utility that checks if stdin is a terminal
- `internal/prompt/testing/mock.go`: Mock implementation for unit tests
- TTY checking prevents interactive prompts from hanging in non-interactive environments (CI/CD, scripts)

#### internal/edit

Edit command implementation:

- `executor.go`: Orchestrates the edit workflow using a `WorkflowFactory` for testability
- `workflow.go`: Resolves the target resources (region/app/profile/env) via flags or interactive prompts, fetches the latest deployed configuration, launches the editor, validates, and deploys
- `editor.go`: Launches `$EDITOR` (falls back to `vi`) against a temp file whose extension is derived from the content type; cleans up the temp file after use
- `options.go`: Command-specific options struct (`Region`, `Application`, `Profile`, `Environment`, `DeploymentStrategy`, `WaitDeploy`, `WaitBake`, `Timeout`, `Description`)
- Reuses `init.InteractiveSelector` for interactive resource selection
- No configuration file required; operates independently of `apcdeploy.yml`
- Validation parity with `run`: same size limit and JSON/YAML syntax checks
- Deployment strategy defaults to the strategy of the most recent deployment when `--deployment-strategy` is omitted

#### internal/lsresources

Resource listing functionality for discovering AppConfig resources:

- `executor.go`: Orchestrates the resource listing workflow using Factory pattern
- `lister.go`: Core logic for fetching AppConfig resources (applications, profiles, environments, deployment strategies)
- `formatter.go`: `FormatJSON` returns the JSON payload; `RenderHumanReadable` emits the human view through Reporter primitives (`Header` / `Table` / `Info`)
- `types.go`: Defines data structures (`ResourcesTree`, `Application`, `ConfigurationProfile`, `Environment`, `DeploymentStrategy`)
- `options.go`: Command-specific options struct (`Region`, `JSON`, `ShowStrategies`, `Silent`)
- Factory pattern enables dependency injection for testing (custom `ClientFactory`)
- No configuration file required; operates independently of `apcdeploy.yml`

### Key Workflows

#### Deployment Flow (run command)

1. Load local config (`apcdeploy.yml`) and data file
2. Resolve resource names to AWS IDs (application, profile, environment, deployment strategy)
3. Compare local content with latest deployed version (auto-skip if identical unless `--force`)
4. Create new hosted configuration version
5. Start deployment
6. Optionally wait for deployment:
   - `--wait-deploy`: Wait until deployment phase completes (enters BAKING state)
   - `--wait-bake`: Wait for complete deployment (DEPLOYING → BAKING → COMPLETE)

#### Diff Calculation

- For FeatureFlags profiles: Strips `_updatedAt`/`_createdAt` metadata before comparison
- Normalizes both JSON and YAML to consistent formatting
- Uses `github.com/sergi/go-diff/diffmatchpatch` for unified diff output
- Special exit code (1) if differences found with `--exit-nonzero` flag

#### Initialization (init command)

1. TTY check: If any flags are omitted (requiring interactive prompts), verify stdin is a terminal
   - Returns `ErrNoTTY` with helpful message suggesting to provide all flags if not a TTY
2. If flags are omitted, use interactive prompts to select:
   - AWS region (with account ID detection)
   - Application
   - Configuration profile
   - Environment
3. Fetch latest deployed configuration from AWS using `GetLatestDeployedConfiguration`
   - Gets the latest deployment for the selected profile/environment
   - Retrieves the configuration content from that deployment
   - If no deployment exists, creates config file without data file
4. Auto-detect ContentType from the hosted configuration version
5. Generate `apcdeploy.yml` with resolved settings
6. Save data file with appropriate extension (`.json`, `.yaml`, `.txt`)

Interactive mode uses `huh` library for terminal UI prompts. TTY checking prevents the command from hanging in non-interactive environments (CI/CD pipelines, scripts).

#### Get Flow (get command)

1. Load local config (`apcdeploy.yml`)
2. Resolve resource names to AWS IDs
3. Check TTY availability (if confirmation prompt required)
   - Returns `ErrNoTTY` with helpful message suggesting `--yes` flag if not a TTY
4. Show confirmation prompt (unless `--yes` flag is used)
5. Fetch latest deployed configuration from AppConfig
6. Output configuration to stdout (respects content type formatting)

#### Pull Flow (pull command)

1. Load local config (`apcdeploy.yml`)
2. Resolve resource names to AWS IDs (application, profile, environment)
3. Get latest deployed configuration using `GetLatestDeployedConfiguration`
   - Fetches the latest deployment for the configuration profile
   - Retrieves configuration content, content type, and deployment metadata
   - Returns an error wrapping `aws.ErrNoDeployment` if no deployment exists
4. Compare local and remote content after normalization
   - For FeatureFlags profiles: Removes `_updatedAt`/`_createdAt` metadata before comparison
   - If no differences found, skip update and report "already up to date"
5. Update local data file only if changes detected
   - Automatically detects content type from the hosted configuration version
   - Overwrites existing data file (force=true)

Key characteristics:
- **Idempotent**: Only updates file when changes exist; safe to run repeatedly
- Does NOT use AppConfig Data API (no per-call charges)
- Useful when configuration changes are made directly in AWS Console
- Syncs local files with currently deployed state
- Supports silent mode for script-friendly output

#### Resource Listing Flow (ls-resources command)

1. Create AWS client with specified region (or use SDK default)
2. Fetch all deployment strategies (always fetched, optionally displayed)
3. Fetch all applications in the region
4. For each application:
   - Fetch all configuration profiles
   - Fetch all environments
   - Sort profiles and environments by name
5. Sort applications by name for consistent output
6. Emit output:
   - Human-readable mode: Reporter primitives (`Header` per region/app, `Table` for strategies/profiles/environments) on stderr — suppressed under `--silent`
   - JSON mode: Encoded payload written to stdout via `Reporter.Data` (always shown, even under `--silent`)

Key characteristics:
- No configuration file required (operates independently)
- Read-only operation (no AWS resource modifications)
- Use `--json` for script consumption; the human-readable view is for terminals only and is suppressed entirely under `--silent`
- Deployment strategies fetched but hidden by default (use `--show-strategies` to display)
- All resources sorted alphabetically for consistent output

#### Edit Flow (edit command)

1. Parse flags; run TTY check if any of `--region`/`--app`/`--profile`/`--env` are omitted
2. Resolve region (flag or interactive prompt), then create the AWS client
3. Select application/profile/environment via flag or interactive prompt
4. Resolve resource names to AWS IDs
5. Fetch the latest deployed configuration using `GetLatestDeployedConfiguration`
   - Error wrapping `aws.ErrNoDeployment` if no prior deployment exists (shared with `pull`)
6. Determine deployment strategy:
   - If `--deployment-strategy` is provided, resolve it to an ID
   - Otherwise reuse the strategy of the latest deployment (from `DeployedConfigInfo.DeploymentStrategyID`)
7. Check for ongoing deployments; abort if one is in progress
8. Launch `$EDITOR` (defaults to `vi`) on a temp file. Extension is derived from the content type (`.json`/`.yaml`/`.txt`)
9. Validate the edited content (size + JSON/YAML syntax) with the same rules as `run`
10. Normalize and compare; skip deployment if content is unchanged
11. Create a new hosted configuration version and start deployment
12. Optionally wait for `--wait-deploy` or `--wait-bake`

Key characteristics:
- **No `apcdeploy.yml` required**: Targets are specified via flags or interactive prompts
- **Requires TTY**: Interactive selection (if any) and `$EDITOR` both need a terminal
- **Strategy inheritance**: Omit `--deployment-strategy` to reuse the previous deployment's strategy
- **Validation parity with `run`**: Same checks apply before AWS mutations

#### Rollback Flow (rollback command)

1. Load local config (`apcdeploy.yml`)
2. Resolve resource names to AWS IDs (application, profile, environment)
3. Find ongoing deployment:
   - Automatically detects the current ongoing deployment (DEPLOYING or BAKING state)
   - Returns `ErrNoOngoingDeployment` if no ongoing deployment exists
4. Get deployment details for confirmation display
5. Prompt for confirmation (unless `--yes` flag is used):
   - Check TTY availability before interactive prompt
   - Returns `ErrNoTTY` with helpful message suggesting `--yes` flag if not a TTY
   - Display deployment status and ask for confirmation
   - Returns `ErrUserDeclined` if user declines
6. Stop deployment using AWS AppConfig StopDeployment API
7. Report success

Key characteristics:
- **Does NOT support AllowRevert**: Only stops in-progress deployments
- Maintains local files as source of truth (no AWS version history dependency)
- Automatically detects and stops the current ongoing deployment
- Requires confirmation by default for safety
- Supports silent mode for script-friendly output

### Testing Patterns

- **Table-driven tests**: All tests should use table-driven test pattern for consistency
- All AWS interactions use the `AppConfigAPI` interface defined in `internal/aws/interface.go`
- Mock implementations in `internal/aws/mock/` for unit tests
- Test files follow `*_test.go` naming convention alongside implementation files
- Use `t.Parallel()` where appropriate for faster test execution
- Reporter is mocked in tests via `internal/reporter/testing/mock.go`
- Prompter is mocked in tests via `internal/prompt/testing/mock.go`
- Factory pattern enables dependency injection for testing (see `internal/init/executor.go`)

### Important Constants

Defined in `internal/config/constants.go`:

- Default deployment strategy
- Supported content types (JSON, YAML, text)
- Magic numbers for normalization (indentation, JSON formatting)

### Configuration File Format

`apcdeploy.yml`:

```yaml
application: <name>
configuration_profile: <name>
environment: <name>
deployment_strategy: <strategy-name>  # optional, defaults to AppConfig.AllAtOnce
data_file: <path>  # relative to apcdeploy.yml or absolute
region: <aws-region>  # optional, uses AWS SDK default if omitted
```

## Output Contract

Every command produces output through `internal/reporter`. The full contract — channels (stdout vs stderr), output kinds (Step/Success/Info/Warn/Error/Header/Box/Table/Spin/Checklist/Progress/Data/Diff), `--silent` semantics, TTY degradation, and rules for adding new commands — lives in [.claude/rules/output-contract.md](.claude/rules/output-contract.md).

Quick reference:

- stdout = machine-readable payload (one per command, e.g. `get` body, `diff` body, `ls-resources --json` payload, `status --silent` state).
- stderr = human-readable progress, structure, errors.
- `--silent` (`-s`) suppresses everything except `Error`, `Data`, `Diff`. Executors MUST NOT branch on `opts.Silent`.

```bash
# Show only the diff without metadata
./apcdeploy diff -c apcdeploy.yml --silent

# Show only the deployment status
./apcdeploy status -c apcdeploy.yml --silent

# Suppress progress messages during deployment
./apcdeploy run -c apcdeploy.yml --wait-bake --silent
```

## Deployment Wait Options

The `run` command supports two wait modes for monitoring deployment progress:

### `--wait-deploy`

Waits until the deployment phase completes (when the deployment enters BAKING state):
- Monitors deployment progress through the DEPLOYING phase
- Returns successfully once baking begins
- Useful for CI/CD pipelines that only need to confirm the rollout started

```bash
./apcdeploy run -c apcdeploy.yml --wait-deploy
```

### `--wait-bake`

Waits for complete deployment including the baking phase:
- Monitors the full deployment lifecycle: DEPLOYING → BAKING → COMPLETE
- Returns only when deployment is fully complete
- Deploy phase renders a progress bar (AppConfig reports a real rollout %); bake phase renders a spinner (no quantified progress — bake is a monitoring wait, not a deployment activity).
- Both phases show a `(~N min left)` countdown derived from the locally observed elapsed time vs the strategy's `DeploymentDurationInMinutes` / `FinalBakeTimeInMinutes`.
- Recommended for production deployments requiring full validation

```bash
./apcdeploy run -c apcdeploy.yml --wait-bake
```

These flags are mutually exclusive and cannot be used together. Either flag can be combined with `--timeout` to specify the total maximum wait duration across both phases (default: 1800 seconds).

## Go Version and Tools

Dev tools are managed by [mise](https://mise.jdx.dev/) via `.mise.toml` (GitHub-releases backend). Run `mise install` to provision them.

- Go 1.26.2
- golangci-lint v2 with configuration in `.golangci.yml`
- gofumpt for stricter formatting
- tparse for test output formatting
- octocov for coverage reporting
- goreleaser for release builds
- terraform for E2E resource provisioning
