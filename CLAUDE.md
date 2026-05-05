# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`apcdeploy` is a CLI tool for managing AWS AppConfig deployments. It enables developers to manage AppConfig resources (applications, configuration profiles, environments) as code through a declarative YAML configuration file (`apcdeploy.yml`).

## Documentation map

This repo ships three reader-facing docs plus a rules file. They target **different audiences** — do not treat them as interchangeable when sourcing facts:

| File | Audience | Purpose |
|---|---|---|
| `README.md` | apcdeploy users (humans) | install, quickstart, command summary |
| `llms.md` | apcdeploy users (AI agents) — served by `apcdeploy context` | detailed command specs, gotchas, AI-specific guidance |
| `CLAUDE.md` (this file) | apcdeploy **developers** (incl. Claude Code) | architecture, conventions, dev workflow |
| `.claude/rules/documentation.md` | doc editors | conventions for the three docs above |

`llms.md` is **not** a substitute for `CLAUDE.md`: it documents how to *use* the CLI, not how the project is built. When in doubt about where a fact belongs, consult `.claude/rules/documentation.md`.

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

# Multi-config (run / diff / pull): pass -c repeatedly
./apcdeploy diff -c environments/dev.yml -c environments/stg.yml -c environments/prod.yml
./apcdeploy run  -c environments/*.yml --parallel 3 --wait-bake
./apcdeploy pull -c environments/*.yml --continue-on-error
```

### E2E Testing

E2E tests require AWS credentials and use Terraform to provision resources:

- **Setup resources**: `mise run e2e-setup` (provisions AWS resources via Terraform)
- **Run tests**: `mise run e2e-run` (executes e2e test script)
- **Clean up**: `mise run e2e-clean` (destroys test resources)
- **Full workflow**: `mise run e2e-full` (setup, test, cleanup in one command)

The runner (`e2e/e2e-test.sh`) drives per-scenario case files in
`e2e/cases/` (S1–S9 success scenarios, E1–E5 error scenarios) using shared
helpers in `e2e/lib/` (`common.sh` for env / output / traps, `assert.sh`
for assertions, `apc.sh` for CLI wrappers and fixture helpers). Run a
subset by passing section IDs (`./e2e/e2e-test.sh S1 S3`); set
`E2E_KEEP_TMP=1` to keep per-section tempdirs for debugging.

## Architecture

### Command Structure (Cobra-based)

Every command follows: `cmd/<command>.go` (CLI parsing only) → `internal/<command>/executor.go` (business logic, Factory pattern, accepts `reporter.Reporter`). `init` and `edit` additionally own a `workflow.go` for their multi-step interactive flow.

- `cmd/root.go` hosts the global flags (`--config`, `--silent`) and the `--description` shared helpers (`validateDescription` for the 1024-rune client-side limit, `resolveDescription` for the `defaultDescription` fallback) used by `run` and `edit`.
- **Commands that do NOT require `apcdeploy.yml`**: `init`, `ls-resources`, `edit`.
- **Exception**: the `context` command is a self-contained utility that outputs embedded `llms.md` via `cmd.SetLLMsContent()` from `main.go`. There is no `internal/context/` package.

### Core Packages

Package-level index. Read the source for details — this only points you at the right place and surfaces non-obvious conventions.

- `internal/aws`: AWS AppConfig SDK wrapper. Owns `AppConfigAPI` (mocked in `internal/aws/mock/`), the name→ID resolver, deployment lifecycle (incl. `StopDeployment`), and `ErrNoDeployment` sentinel consumed by `pull` / `edit` via `errors.Is`.
- `internal/config`: Loads / validates `apcdeploy.yml` and data files (JSON/YAML/text). Owns normalization (`HasContentChanged`, `NormalizeByExtension`, FeatureFlags metadata stripping) and pre-deploy validation (size + syntax) shared by `run` / `edit`.
- `internal/reporter` + `internal/cli`: The single output abstraction. See [Output Contract](.claude/rules/output-contract.md). Executors MUST NOT call `fmt.Fprint*` and MUST NOT branch on `opts.Silent` — Reporter selection in `cmd/root.go` handles silent semantics.
- `internal/prompt`: Interactive prompt interface (`Select` / `Input` / `CheckTTY`). TTY check prevents hangs in non-interactive environments.
- `internal/batch`: Multi-config orchestration for `run` / `diff` / `pull`. Pre-loads/validates every `-c` target before any AWS work, runs under a worker pool with fail-fast or `ContinueOnError`. Each executor exposes both `Execute` (single-config) and `RunOnTarget` (orchestrator), sharing a `runOnTargetWith*` body so output is bit-identical between paths.
- `internal/deploywait`: Helpers between `internal/aws` and `internal/reporter` that adapt deployment polling ticks into `Targets.SetPhase` / `SetProgress`. Lives in its own package to avoid `aws ↔ reporter` reverse dependencies.
- `internal/apcerrors`: Maps AWS API error codes to short user-facing resolution hints via `Resolution(err)`. Add new hints to `resolutionHints` rather than inlining at call sites — see `.claude/rules/output-contract.md` § "Resolution hints".
- `internal/display`: Presentation helpers shared by `status` and `rollback` (deployment-status block rendering). Routes everything through `Reporter` so silent mode behaves uniformly — callers MUST NOT branch on `opts.Silent`.
- Per-command packages (`internal/init`, `internal/run`, `internal/diff`, `internal/edit`, `internal/lsresources`, `internal/pull`, `internal/get`, `internal/status`, `internal/rollback`): each owns an `executor.go` (Factory pattern for testability) plus `options.go`. `init` and `edit` additionally own a `workflow.go` for their multi-step interactive flow.

**IMPORTANT — AWS List API usage:** Always go through the centralized helpers in `internal/aws/client_list_paginated.go` (`ListAllApplications`, `ListAllConfigurationProfiles`, `ListAllEnvironments`, `ListAllDeploymentStrategies`, `ListAllDeployments`, `ListAllHostedConfigurationVersions`). Direct SDK `List*` calls outside that file silently truncate when results exceed AWS page limits.

### Cross-cutting conventions

A few rules don't live cleanly in any one package's source. Spelled out so they survive code drift:

- **TTY discipline**: every command that may prompt (`init`, `edit`, `rollback`, `get`) checks TTY before prompting and returns `prompt.ErrNoTTY` with a hint pointing at the bypass flag (`--yes` or "supply all flags") when stdin is not a terminal. The check sits in the executor, not in the prompter.
- **`pull` / `edit` "no prior deployment"**: both surface failures as errors that wrap `aws.ErrNoDeployment`. New callers of `GetLatestDeployedConfiguration` should preserve that wrap so `errors.Is` works.
- **`edit` strategy inheritance**: omit `--deployment-strategy` to reuse the previous deployment's strategy via `DeployedConfigInfo.DeploymentStrategyID`. Do not introduce a hard-coded fallback.
- **`run` / `edit` validation parity**: pre-deploy size + JSON/YAML syntax checks live in `internal/config/validate.go`. New deploy-shape commands MUST reuse it rather than re-implementing.
- **`run` / `pull` no-op detection**: content comparison is normalized via `internal/config/normalize.go` (FeatureFlags `_updatedAt` / `_createdAt` stripped) and gated on `HasContentChanged`. Both commands skip the AWS write when unchanged.
- **`rollback` semantics**: stops the *current* ongoing deployment only (no `AllowRevert`). Returns `ErrNoOngoingDeployment` when nothing is in flight.
- **Wait flags**: `--wait-deploy` and `--wait-bake` are mutually exclusive on `run` and `edit`. Both honor `--timeout` (default 1800s, applied across phases).

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

Constants for default deployment strategy, supported content types, and normalization tunings live in `internal/config/constants.go`. Reach for it before introducing new magic numbers.

## Output Contract

The full contract lives in [.claude/rules/output-contract.md](.claude/rules/output-contract.md). Three rules to remember while writing executor code:

- stdout = machine-readable payload only (one per command). stderr = everything else.
- `--silent` (`-s`) suppresses progress; preserves `Error` / `Data` / `Diff`.
- Executors MUST NOT call `fmt.Fprint*` directly and MUST NOT branch on `opts.Silent` — Reporter selection in `cmd/root.go` is the single source of truth.

