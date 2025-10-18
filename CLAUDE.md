# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`apcdeploy` is a CLI tool for managing AWS AppConfig deployments. It enables developers to manage AppConfig resources (applications, configuration profiles, environments) as code through a declarative YAML configuration file (`apcdeploy.yml`).

## Development Rules

When implementing new features or fixing bugs, follow these absolute rules:

- **TDD (Test-Driven Development)**: Write tests before implementation
- **Code consistency**: Match existing code style and patterns
- **CI validation**: Ensure `make ci` passes before considering work complete
- **Test coverage**: Maintain or improve test coverage (never decrease it)

## Common Commands

### Development

- **Build**: `make build` or `go build`
- **Run tests**: `make test` (uses tparse for formatted output)
- **Run single test**: `go test -run TestName ./path/to/package`
- **Lint**: `make lint` (uses golangci-lint v2)
- **Fix lint issues**: `make lint-fix`
- **Format code**: `make fmt` (uses gofumpt)
- **Generate coverage**: `make cov` (creates cover.html)
- **Full CI workflow**: `make ci` (fmt, modernize, lint, test, build)
- **Install dev tools**: `make tool-install`

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
./apcdeploy context  # Output llms.md for AI assistants

# Silent mode (suppress verbose output)
./apcdeploy ls-resources --region us-east-1 --silent
./apcdeploy diff -c apcdeploy.yml --silent
./apcdeploy status -c apcdeploy.yml --silent
```

### E2E Testing

E2E tests require AWS credentials and use Terraform to provision resources:

- **Setup resources**: `make e2e-setup` (provisions AWS resources via Terraform)
- **Run tests**: `make e2e-run` (executes e2e test script)
- **Clean up**: `make e2e-clean` (destroys test resources)
- **Full workflow**: `make e2e-full` (setup, test, cleanup in one command)

## Architecture

### Command Structure (Cobra-based)

All commands follow the pattern: `cmd/<command>.go` → `internal/<command>/executor.go`

**Exception**: The `context` command is a simple utility that only outputs embedded content (`llms.md`). It does not follow the standard command structure and has no corresponding `internal/context/` directory. The implementation is entirely contained in `cmd/context.go`, with the content embedded in `main.go` and passed via `cmd.SetLLMsContent()`.

1. **cmd/**: Cobra command definitions and CLI flag parsing
   - `root.go`: Root command with global flags (`--config`, `--silent`)
   - Each command file (`init.go`, `run.go`, `diff.go`, `status.go`, `get.go`, `ls_resources.go`) handles CLI concerns only
   - `context.go`: Simple command that outputs embedded `llms.md` content for AI assistants (no internal package)
   - `init.go`: Supports interactive mode for resource selection; all flags are optional
   - `ls_resources.go`: Lists AppConfig resources; does not require `apcdeploy.yml`; all flags are optional

2. **internal/\<command\>/**: Business logic for each command
   - `executor.go`: Main execution logic using Factory pattern for testability
   - `options.go`: Command-specific options struct
   - `workflow.go` (init command): Handles initialization workflow including interactive prompts
   - Executors accept a `reporter.ProgressReporter` for user feedback

### Core Packages

#### internal/aws

AWS AppConfig client wrapper with:

- `Client`: Wraps AWS SDK AppConfig client with polling interval configuration
- `AppConfigAPI`: Interface for AppConfig operations (enables mocking in tests)
- `list.go`: **Centralized list operations with pagination handling** - All AWS List APIs should use these methods
- `resolver.go`: Resolves resource names (application, profile, environment) to AWS IDs
- `deployment.go`: Deployment creation and monitoring logic
- Version info is injected at build time via `main.go` variables

**IMPORTANT - AWS List API Usage:**

Always use the centralized list methods in `list.go` instead of calling AWS SDK List APIs directly:

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
- `normalize.go`: Normalizes JSON/YAML for consistent comparisons (removes FeatureFlags metadata)
- `generator.go`: Generates `apcdeploy.yml` from AWS resources during init

#### internal/reporter

Progress reporting interface used across all commands:

- `ProgressReporter`: Interface with `Progress()`, `Success()`, `Warning()` methods
- `internal/cli/reporter.go`: Console implementation with colored output using lipgloss
- `internal/cli/silent_reporter.go`: Silent implementation that suppresses all output (for `--silent` flag)
- `internal/cli/factory.go`: Factory function `GetReporter()` to select appropriate reporter

#### internal/prompt

Interactive prompt interface for user input:

- `Prompter`: Interface with `Select()`, `Input()`, and `CheckTTY()` methods for interactive operations
- `internal/prompt/huh.go`: Implementation using `huh` library for terminal UI
- `internal/prompt/tty.go`: TTY detection utility that checks if stdin is a terminal
- `internal/prompt/testing/mock.go`: Mock implementation for unit tests
- TTY checking prevents interactive prompts from hanging in non-interactive environments (CI/CD, scripts)

#### internal/lsresources

Resource listing functionality for discovering AppConfig resources:

- `executor.go`: Orchestrates the resource listing workflow using Factory pattern
- `lister.go`: Core logic for fetching AppConfig resources (applications, profiles, environments, deployment strategies)
- `formatter.go`: Formats output in human-readable or JSON format
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
3. Fetch existing AppConfig configuration from AWS
4. Auto-detect ContentType from configuration profile
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

#### Resource Listing Flow (ls-resources command)

1. Create AWS client with specified region (or use SDK default)
2. Fetch all deployment strategies (always fetched, optionally displayed)
3. Fetch all applications in the region
4. For each application:
   - Fetch all configuration profiles
   - Fetch all environments
   - Sort profiles and environments by name
5. Sort applications by name for consistent output
6. Format output:
   - Human-readable format: Hierarchical text view with optional deployment strategies section
   - JSON format: Structured JSON with all resource details
7. Output to stdout

Key characteristics:
- No configuration file required (operates independently)
- Read-only operation (no AWS resource modifications)
- Supports silent mode for script-friendly output
- Deployment strategies fetched but hidden by default (use `--show-strategies` to display)
- All resources sorted alphabetically for consistent output

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

## Silent Mode

The `--silent` (or `-s`) flag is a global flag that suppresses verbose output and shows only essential information.

### Behavior

- **Suppressed**: Progress messages, success messages, warnings
- **Always shown**: Error messages (via stderr), final results (diff output, status, etc.)
- **Use cases**: CI/CD pipelines, scripting, machine-readable output

### Implementation

- Silent mode is implemented via the `reporter.ProgressReporter` interface
- `internal/cli/factory.go` provides `GetReporter()` to select the appropriate reporter
- When `--silent` is set, `SilentReporter` is used, which has no-op implementations for all methods
- Each command's Options struct includes a `Silent` field for conditional display logic
- Commands like `diff` and `status` use `opts.Silent` to choose between verbose and silent display functions

### Examples

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
- Recommended for production deployments requiring full validation

```bash
./apcdeploy run -c apcdeploy.yml --wait-bake
```

These flags are mutually exclusive and cannot be used together. Either flag can be combined with `--timeout` to specify maximum wait duration (default: 600 seconds).

## Go Version and Tools

- Go 1.25.1 (uses `tool` directive in go.mod for managing dev tools)
- golangci-lint v2 with configuration in `.golangci.yml`
- gofumpt for stricter formatting
- tparse for test output formatting
