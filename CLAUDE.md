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
./apcdeploy init --app my-app --profile my-profile --env production
./apcdeploy diff -c apcdeploy.yml
./apcdeploy run -c apcdeploy.yml --wait
./apcdeploy status -c apcdeploy.yml
./apcdeploy get -c apcdeploy.yml
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

1. **cmd/**: Cobra command definitions and CLI flag parsing
   - `root.go`: Root command with global flags (`--config`)
   - Each command file (`init.go`, `run.go`, `diff.go`, `status.go`, `get.go`) handles CLI concerns only
   - `init.go`: Has a local `--region` flag (required for initial setup)

2. **internal/\<command\>/**: Business logic for each command
   - `executor.go`: Main execution logic
   - `options.go`: Command-specific options struct
   - Executors accept a `reporter.ProgressReporter` for user feedback

### Core Packages

#### internal/aws

AWS AppConfig client wrapper with:

- `Client`: Wraps AWS SDK AppConfig client with polling interval configuration
- `AppConfigAPI`: Interface for AppConfig operations (enables mocking in tests)
- `resolver.go`: Resolves resource names (application, profile, environment) to AWS IDs
- `deployment.go`: Deployment creation and monitoring logic
- Version info is injected at build time via `main.go` variables

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

### Key Workflows

#### Deployment Flow (run command)

1. Load local config (`apcdeploy.yml`) and data file
2. Resolve resource names to AWS IDs (application, profile, environment, deployment strategy)
3. Compare local content with latest deployed version (auto-skip if identical unless `--force`)
4. Create new hosted configuration version
5. Start deployment
6. Optionally poll deployment status if `--wait` is specified

#### Diff Calculation

- For FeatureFlags profiles: Strips `_updatedAt`/`_createdAt` metadata before comparison
- Normalizes both JSON and YAML to consistent formatting
- Uses `github.com/sergi/go-diff/diffmatchpatch` for unified diff output
- Special exit code (1) if differences found with `--exit-nonzero` flag

#### Initialization (init command)

1. Fetch existing AppConfig configuration from AWS
2. Auto-detect ContentType from configuration profile
3. Generate `apcdeploy.yml` with resolved settings
4. Save data file with appropriate extension (`.json`, `.yaml`, `.txt`)

#### Get Flow (get command)

1. Load local config (`apcdeploy.yml`)
2. Resolve resource names to AWS IDs
3. Fetch latest deployed configuration from AppConfig
4. Output configuration to stdout (respects content type formatting)

### Testing Patterns

- All AWS interactions use the `AppConfigAPI` interface defined in `internal/aws/interface.go`
- Mock implementations in `internal/aws/mock/` for unit tests
- Test files follow `*_test.go` naming convention alongside implementation files
- Use `t.Parallel()` where appropriate for faster test execution
- Reporter is mocked in tests via `internal/reporter/testing/mock.go`

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
deployment_strategy: <strategy-name>  # optional, defaults to AppConfig.Linear50PercentEvery30Seconds
data_file: <path>  # relative to apcdeploy.yml or absolute
region: <aws-region>  # optional, uses AWS SDK default if omitted
```

## Go Version and Tools

- Go 1.25.1 (uses `tool` directive in go.mod for managing dev tools)
- golangci-lint v2 with configuration in `.golangci.yml`
- gofumpt for stricter formatting
- tparse for test output formatting
