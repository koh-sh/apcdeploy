# llms.md

Documentation for `apcdeploy`, served by `apcdeploy context` for AI assistants and the humans directing them.

## Overview of apcdeploy

`apcdeploy` is a declarative CLI tool for managing AWS AppConfig deployments. It allows you to manage AppConfig applications, configuration profiles, and environments as "code" using a YAML configuration file (`apcdeploy.yml`).

### Capabilities

- List available AWS AppConfig resources (`ls-resources`)
- Auto-generate configuration files from existing AWS AppConfig resources (`init`)
- Deploy configuration changes (`run`)
- Compare differences between local files and deployed configurations (`diff`)
- Monitor deployment status (`status`)
- Retrieve deployed configurations (`get`)
- Sync local files with deployed configurations (`pull`)
- Validate local configuration data against its schema without deploying (`validate`)
- Stop ongoing deployments (`rollback`)
- Edit deployed configuration directly in `$EDITOR` and deploy (`edit`)
- Print this document (`context`)

### Important Constraints

- **Supports AWS AppConfig hosted configuration store only**
- AppConfig resources (applications, configuration profiles, environments, deployment strategies) must be created in AWS beforehand. This tool manages existing resources and does not create them.
- All commands except `context` require AWS credentials resolved via the standard AWS SDK chain (env vars, shared config, IAM role, etc.).
- Hosted configuration size limit: 2 MB by default, 4 MB max (limit-increase request); `apcdeploy run` / `edit` validate this client-side.

### Supported Content Types

1. **JSON** (`.json` files) — automatic validation and formatting
2. **YAML** (`.yaml` / `.yml` files) — automatic validation and formatting
3. **Plain Text** (`.txt` or other extensions) — deployed as-is

For FeatureFlags profiles (JSON or YAML), the metadata fields `_createdAt` and `_updatedAt` are automatically excluded from diff/idempotency comparisons.

### TTY Requirements

These commands prompt interactively when stdin is a TTY. In non-interactive environments (CI/CD, AI agents) supply the bypass listed below; otherwise the command exits with `interactive mode requires a TTY: …`.

| Command | Trigger | Bypass |
|---|---|---|
| `init` | resource selection prompts | supply all of `--region`, `--app`, `--profile`, `--env` |
| `get` | cost confirmation prompt (Data API is billable) | `--yes` |
| `rollback` | deployment-stop confirmation | `--yes` |
| `edit` | `$EDITOR` invocation | none — there is no non-interactive mode. AI agents should use `pull` → edit file → `run` instead. |

`run`, `diff`, `status`, `pull`, `validate`, `ls-resources`, `context` do not require a TTY.

## Recommended Usage Flows

The flows below show the canonical command sequence for each scenario. Where a step says "edit the data file", AI agents should make programmatic edits (Write/Edit tools); human users typically open the file in `$EDITOR`.

### Initial Setup Flow

Starting from an existing AppConfig profile + environment with no `apcdeploy.yml` yet:

```bash
# 1. Discover resources
apcdeploy ls-resources --region us-west-2 --json   # JSON for AI parsing
# or:
apcdeploy ls-resources --region us-west-2          # human-readable

# 2. Initialize (AI agents: pass all four flags; human users may omit them for prompts)
apcdeploy init --region us-west-2 --app my-app --profile my-profile --env production

# 3. Edit the generated data file (data.json / data.yaml / data.txt) as needed

# 4. Preview, deploy, monitor
apcdeploy diff -c apcdeploy.yml
apcdeploy run -c apcdeploy.yml
apcdeploy status -c apcdeploy.yml
```

### Daily Change Management Flow

```bash
# Edit data file, then:
apcdeploy diff -c apcdeploy.yml
apcdeploy run -c apcdeploy.yml
apcdeploy status -c apcdeploy.yml   # optional, to inspect rollout
```

For CI/CD, gate the deploy step on `apcdeploy diff -c apcdeploy.yml --exit-nonzero --silent`.

### Sync Flow (Changes Made in the AWS Console)

```bash
apcdeploy pull -c apcdeploy.yml    # writes data file only if it differs
apcdeploy diff -c apcdeploy.yml    # should now report no differences
```

## Configuration File Reference

### Structure of apcdeploy.yml

```yaml
# Required: AppConfig application name
application: my-application

# Required: Configuration profile name
configuration_profile: my-config-profile

# Required: Environment name
environment: production

# Optional: Deployment strategy (default: AppConfig.AllAtOnce)
deployment_strategy: AppConfig.Linear

# Required: Path to configuration data file (relative or absolute)
# Relative paths are interpreted from apcdeploy.yml location
data_file: data.json

# Required: AWS region
region: us-west-2
```

### data_file Path Resolution

- **Relative path** (e.g. `data.json`, `config/data.json`): resolved against the directory containing `apcdeploy.yml`.
- **Absolute path** (e.g. `/home/user/configs/data.json`): used as-is.

### Deployment Strategies

`deployment_strategy` accepts the name of any AWS pre-defined strategy (prefixed `AppConfig.`) or a custom strategy you have created in AppConfig. Discover what's available in the target region with:

```bash
apcdeploy ls-resources --region us-west-2 --show-strategies
# or:
aws appconfig list-deployment-strategies
```

Common pre-defined strategies:

| Name | Behavior | Typical use |
|---|---|---|
| `AppConfig.Linear` | 20% every 6 min over 30 min, 10 min bake | production (AWS recommended) |
| `AppConfig.Canary10Percent20Minutes` | exponential 10 % over 20 min | production (AWS recommended) |
| `AppConfig.AllAtOnce` | 100 % immediately, no bake | hotfixes / dev |
| `AppConfig.Linear50PercentEvery30Seconds` | 50 % every 30 s over 1 min | testing / demo |

All strategies (pre-defined and custom) integrate with CloudWatch Alarms for automatic rollback. See [AWS AppConfig pre-defined strategies](https://docs.aws.amazon.com/appconfig/latest/userguide/appconfig-creating-deployment-strategy-predefined.html) and [creating custom strategies](https://docs.aws.amazon.com/appconfig/latest/userguide/appconfig-creating-deployment-strategy.html) for details.

## Command Reference

### ls-resources command

Lists AWS AppConfig resources (applications, configuration profiles, environments, and optionally deployment strategies) in a hierarchical view. Designed for AI agents and scripts to discover resources before running `init`. Read-only; does not require `apcdeploy.yml`.

#### Usage

```bash
apcdeploy ls-resources --region us-east-1
```

#### Flags

| Flag | Description |
|---|---|
| `--region <region>` | AWS region. Uses AWS SDK default if omitted. |
| `--json` | Emit JSON to stdout. The human-readable view goes through Reporter primitives on stderr; combine with `--silent` for clean machine output. |
| `--show-strategies` | Include deployment strategies in the output. Off by default. |

#### Output Format

Human-readable (with `--show-strategies`):

```
Region: us-east-1

Deployment Strategies:
  - AppConfig.Linear (ID: e5f6g7h8)
    Description: AWS Recommended deployment strategy for production environments
    Deployment Duration: 30 minutes
    Final Bake Time: 10 minutes
    Growth Factor: 20.0%
    Growth Type: LINEAR

Applications:
  [1] my-app (ID: abc123)
      Configuration Profiles:
        - my-profile (ID: prof-123)
        - feature-flags (ID: prof-456)
      Environments:
        - dev (ID: env-111)
        - production (ID: env-222)
```

JSON (with `--show-strategies`):

```json
{
  "region": "us-east-1",
  "applications": [
    {
      "name": "my-app",
      "id": "abc123",
      "configuration_profiles": [
        {"name": "my-profile", "id": "prof-123"}
      ],
      "environments": [
        {"name": "production", "id": "env-222"}
      ]
    }
  ],
  "deployment_strategies": [
    {
      "name": "AppConfig.Linear",
      "id": "e5f6g7h8",
      "description": "AWS Recommended deployment strategy for production environments",
      "deployment_duration_in_minutes": 30,
      "final_bake_time_in_minutes": 10,
      "growth_factor": 20,
      "growth_type": "LINEAR"
    }
  ]
}
```

Without `--show-strategies`: the human view omits the `Deployment Strategies:` block; the JSON output keeps the key but emits `"deployment_strategies": []`.

#### Exit Codes

- `0`: success
- `1`: AWS error (credentials, region, throttling, etc.)

#### Examples

```bash
# Human-readable, with deployment strategies (use this before `init`)
apcdeploy ls-resources --region us-east-1 --show-strategies

# JSON for scripts/AI parsing (silent suppresses Reporter chatter on stderr)
apcdeploy ls-resources --region us-west-2 --json --silent > resources.json

# Pipe into jq to extract application names
apcdeploy ls-resources --region us-west-2 --json | jq -r '.applications[].name'
```

### init command

Generates `apcdeploy.yml` and a configuration data file from existing AWS AppConfig resources. Does not create AWS resources — application, configuration profile, and environment must already exist.

#### Usage

```bash
apcdeploy init --region us-west-2 --app my-app --profile my-profile --env production
```

#### Flags

| Flag | Description |
|---|---|
| `--region <region>` | AWS region. Interactive prompt if omitted (requires `account:ListRegions` IAM permission to enumerate regions). |
| `--app <name>` | Application name. Interactive prompt if omitted. |
| `--profile <name>` | Configuration profile name. Interactive prompt if omitted. |
| `--env <name>` | Environment name. Interactive prompt if omitted. |
| `-c, --config <path>` | Output config file path (default: `apcdeploy.yml`). |
| `-o, --output-data <path>` | Output data file path. Auto-determined from content type if omitted (`data.json` / `data.yaml` / `data.txt`). |
| `-f, --force` | Overwrite existing output files without confirmation. |

Partial flag specification is supported: any omitted flag triggers an interactive prompt for that field only. AI agents in non-interactive environments must supply all of `--region`, `--app`, `--profile`, `--env`.

#### Operation Details

1. Resolve region (flag or interactive prompt; selected region's AWS Account ID is displayed)
2. Resolve application (flag or interactive selection from the region's apps)
3. Resolve configuration profile (flag or interactive selection from the app's profiles)
4. Resolve environment (flag or interactive selection from the app's environments)
5. Fetch the latest deployed configuration content and auto-detect Content-Type
6. Generate `apcdeploy.yml` and the data file (extension chosen from content type)

#### Behavior

- **Overwrite protection**: existing `apcdeploy.yml` or data file is preserved unless `-f` is supplied.
- **IAM**: when `--region` is supplied, `account:ListRegions` is **not** required. The permission is only needed for the interactive region picker.
- **AI workflow**: use `apcdeploy ls-resources --region <r> --json` to discover names, then call `init` with all four flags. Fallback if `ls-resources` is unavailable: ask the user to provide application / profile / environment names.

#### Exit Codes

- `0`: success
- `1`: AWS error, file-overwrite refusal, or TTY error when prompts are needed without a TTY

#### Examples

```bash
# AI agent / CI workflow (non-interactive)
apcdeploy init --region us-west-2 --app my-app --profile my-profile --env production

# Fully interactive (human user)
apcdeploy init

# Custom output paths
apcdeploy init --region us-east-1 --app my-app --profile my-profile --env prod \
  -c environments/prod/apcdeploy.yml -o environments/prod/data.json

# Overwrite existing files
apcdeploy init --region us-west-2 --app my-app --profile my-profile --env prod -f
```

### run command

Deploys configuration changes to AWS AppConfig.

#### Usage

```bash
apcdeploy run -c apcdeploy.yml
```

#### Flags

| Flag | Description |
|---|---|
| `--wait-deploy` | Wait until the deployment phase completes (transition to BAKING). Mutually exclusive with `--wait-bake`. |
| `--wait-bake` | Wait until the full deployment completes (DEPLOYING → BAKING → COMPLETE). Mutually exclusive with `--wait-deploy`. |
| `--force` | Deploy even when content is unchanged (bypasses auto-skip). |
| `--timeout <seconds>` | Timeout for deployment wait (default: 1800). Must be greater than 0. Shared across both phases when `--wait-bake` is used. |
| `--description <text>` | Description attached to the configuration version and deployment. Visible in the AppConfig console and in `apcdeploy status` output. Defaults to `"Deployed by apcdeploy"`. Pass `--description ""` to clear. Maximum 1024 Unicode characters (counted by rune); rejected client-side when exceeded. |

#### Operation Details

1. Load `apcdeploy.yml` and the `data_file`
2. Diff check: compare local file with the latest deployed version; if identical, skip (auto-skip, bypass with `--force`)
3. Create a new hosted configuration version
4. Start deployment to the specified environment

The diff check happens before any AWS write, so a no-op `run` produces no version and no deployment.

#### Behavior

**Idempotency**: When the local file matches the deployed version, the command exits successfully without creating a version or deployment. For FeatureFlags profiles, the metadata fields `_createdAt` and `_updatedAt` are excluded from the comparison.

**Wait flag comparison**:

| Option | Wait until | Use |
|---|---|---|
| (none, default) | exits immediately after StartDeployment returns | check progress with `status` |
| `--wait-deploy` | DEPLOYING → BAKING transition | synchronous wait for rollout phase only |
| `--wait-bake` | DEPLOYING → BAKING → COMPLETE | synchronous wait for full lifecycle |

When `--wait-bake` is used, the deploy phase is rendered as a progress bar (AppConfig reports actual rollout %) and the bake phase as a spinner (bake is a monitoring window, not a quantified rollout). Both phases display a `(~N min left)` countdown derived from the strategy's `DeploymentDurationInMinutes` / `FinalBakeTimeInMinutes`. `--timeout` bounds the total wait (shared across both phases).

**For AI assistants**: prefer the no-wait default and poll with `status`. The `--wait-*` modes are intended for human operators.

#### Errors

| Cause | Resolution |
|---|---|
| Another deployment is in progress (DEPLOYING, BAKING, VALIDATING, or ROLLING_BACK) for the same environment | Wait for it to finish, or stop it with `apcdeploy rollback -c apcdeploy.yml --yes`. AppConfig allows only one active deployment per environment. |
| Wait timed out before the deployment reached the requested phase | Raise `--timeout` (e.g. `AppConfig.Linear` needs ≈ 30 min deploy + 10 min bake), or drop `--wait-*` and poll with `apcdeploy status`. |

#### Exit Codes

- `0`: success (including auto-skip when content is unchanged)
- `1`: general error (load, validation, AWS, or wait timeout)

#### Examples

```bash
# Basic deployment (recommended)
apcdeploy run -c apcdeploy.yml

# Deploy even when content is unchanged
apcdeploy run -c apcdeploy.yml --force

# Synchronous wait for full rollout (human operators)
apcdeploy run -c apcdeploy.yml --wait-bake --timeout 3900

# Attach a description
apcdeploy run -c apcdeploy.yml --description "ticket-123: tweak feature flag"
```

### diff command

Displays a unified diff between the local data file and the deployed configuration.

#### Usage

```bash
apcdeploy diff -c apcdeploy.yml
```

#### Flags

| Flag | Description |
|---|---|
| `--exit-nonzero` | Exit with code `1` if differences exist (otherwise `0`). For CI gating. |

#### Behavior

- **Normalization**: JSON and YAML are normalized (indentation and line-break differences are absorbed); plain text is compared byte-for-byte. For FeatureFlags profiles, `_createdAt` and `_updatedAt` are stripped before comparison.
- **In-progress deployments**: when a deployment is `DEPLOYING` or `BAKING`, the diff is taken against that in-flight configuration. If that deployment is later rolled back (`ROLLED_BACK`), the diff output may not match the configuration that ends up deployed.
- **Output**: lines prefixed `-` are present in the deployed version, `+` lines are present locally. When the two match, the command prints `No differences found` and exits 0.
- **No prior deployment**: when the configuration profile + environment has never been deployed, the local data is emitted as an "all-added" unified diff (every line `+` prefixed) — semantically the would-be initial deployment payload. The command exits `0` (unlike `pull` / `edit`, which exit `2` in this state).

#### Exit Codes

- `0`: success — no differences, or `--exit-nonzero` not set
- `1`: differences exist (only when `--exit-nonzero` is set), or general error

#### Examples

```bash
# Pre-deploy check
apcdeploy diff -c apcdeploy.yml

# CI guard: deploy only when changes are present
if apcdeploy diff -c apcdeploy.yml --exit-nonzero --silent; then
  echo "No changes to deploy"
else
  apcdeploy run -c apcdeploy.yml
fi
```

### status command

Displays the latest (or a specified) deployment's state, rollout progress, and metadata.

#### Usage

```bash
apcdeploy status -c apcdeploy.yml
```

#### Flags

| Flag | Description |
|---|---|
| `--deployment <number>` | Show this specific deployment number. Defaults to the latest deployment for the configuration profile + environment. |

#### Behavior

Displayed fields: deployment number, state, percentage complete, configuration version, start time.

State values:

| State | Meaning |
|---|---|
| `VALIDATING` | AppConfig is validating the configuration before the rollout begins (in-flight). |
| `DEPLOYING` | Configuration is being gradually rolled out to targets. |
| `BAKING` | Rollout reached all targets; the strategy is in its validation window. |
| `COMPLETE` | Deployment finished successfully. |
| `ROLLING_BACK` | Deployment is being rolled back, automatically (e.g. a CloudWatch alarm) or via `apcdeploy rollback` (in-flight). |
| `ROLLED_BACK` | Deployment was stopped (or rolled back by AppConfig); previous version is in effect. |
| `REVERTED` | Deployment was reverted to the previous configuration version (terminal). |

#### Errors

| Cause | Resolution |
|---|---|
| No deployment has ever been made for the profile + environment (exit code `2`) | Run `apcdeploy run -c apcdeploy.yml` first to create the initial deployment, then `status` will have something to report. |

#### Exit Codes

- `0`: success
- `1`: AWS error
- `2`: no deployment exists for the profile + environment

#### Examples

```bash
# After deploying, poll for completion (10 s interval)
apcdeploy run -c apcdeploy.yml
while true; do
  apcdeploy status -c apcdeploy.yml --silent
  sleep 10
done

# Inspect a specific past deployment
apcdeploy status -c apcdeploy.yml --deployment 3
```

### get command

Retrieves the deployed configuration via the AWS AppConfig Data API and writes it to stdout. **The Data API is billed per call** — prefer `pull` (which uses no Data API) unless you specifically need the data on stdout.

#### Usage

```bash
apcdeploy get -c apcdeploy.yml --yes > deployed.json
```

#### Flags

| Flag | Description |
|---|---|
| `-y, --yes` | Skip the cost-confirmation prompt. Required in non-interactive environments. |

#### Behavior

- **Output channel**: configuration body to stdout, formatted by Content-Type (JSON / YAML pretty-printed; plain text emitted as-is). Reporter chatter goes to stderr.
- **Billing**: each invocation calls `StartConfigurationSession` + `GetLatestConfiguration` on the AppConfig Data API. AWS bills per call. The cost-confirmation prompt surfaces this; `--yes` bypasses it.
- **Alternative for AI agents**: use `apcdeploy pull` to sync the deployed configuration into the local data file without invoking the Data API.

#### Exit Codes

- `0`: success
- `1`: AWS error, or TTY error when `--yes` is missing in a non-interactive environment

#### Examples

```bash
# Save the deployed configuration to a file
apcdeploy get -c apcdeploy.yml --yes > deployed.json

# Pipe into jq
apcdeploy get -c apcdeploy.yml --yes | jq '.features.new_feature'
```

### pull command

Syncs the local data file with the currently deployed configuration. Does not use the AWS AppConfig Data API (no per-call billing).

#### Usage

```bash
apcdeploy pull -c apcdeploy.yml
```

#### Flags

No command-specific flags. Uses global flags only (`-c, --config`, `-s, --silent`, and `-c` may be repeated for multi-config mode).

#### Operation Details

1. Resolve the latest deployment for the profile + environment (errors if none exists)
2. Fetch the hosted configuration version it points at
3. Compare local vs deployed after normalization (FeatureFlags `_createdAt` / `_updatedAt` stripped)
4. If they differ, overwrite the local data file; otherwise report "already up to date"

Compare-before-write means the local file's mtime is unchanged when content already matches.

#### Behavior

- **Idempotent**: safe to run repeatedly; subsequent runs are no-ops if nothing changed.
- **Content-Type aware**: respects the content type recorded on the hosted configuration version (JSON / YAML / plain text).
- **No Data API charges**: uses control-plane APIs only (`ListDeployments` + `GetHostedConfigurationVersion`).

#### Errors

| Cause | Resolution |
|---|---|
| No prior deployment exists for the profile + environment (exit code `2`) | Run `apcdeploy run` once to create the first deployment, then re-run `pull`. |

#### Exit Codes

- `0`: success (including no-op when local file already matches)
- `1`: general error (AWS, I/O, etc.)
- `2`: no prior deployment exists for the profile + environment

#### Examples

```bash
# Sync after a change made in the AWS Console
apcdeploy pull -c apcdeploy.yml

# Idempotency: a second pull is a no-op
apcdeploy pull -c apcdeploy.yml  # may write
apcdeploy pull -c apcdeploy.yml  # "already up to date"
```

### validate command

Validates the local data file against its schema locally, without creating a configuration version (read-only — no write APIs are called).

#### Usage

```bash
apcdeploy validate -c apcdeploy.yml
```

#### Flags

| Flag | Description |
|---|---|
| `--parallel <n>` | Maximum concurrent targets when `-c` is repeated (`0` = all in parallel) |
| `--continue-on-error` | Run remaining targets after one fails (default: fail-fast) |

#### Operation Details

1. Resolve the profile (reads its type and validators; no deployment is fetched)
2. Read the local data file and determine its content type
3. Select the schema: FeatureFlags use the built-in AWS schema; Freeform JSON uses the profile's JSON_SCHEMA validator fetched from AWS
4. Validate, reporting each violation with its location within the data

#### Behavior

- **Read-only**: never calls `CreateHostedConfigurationVersion` or any write API; safe to run anytime.
- **FeatureFlags**: checks structure against the built-in schema (layer A), then each value against the `constraints` declared in the data — enum / minimum / maximum / pattern / required / type (layer B). Multi-variant flags are checked per variant (`_variants[].attributeValues`).
- **Freeform JSON**: validated against the profile's JSON_SCHEMA validator (the same validator AWS enforces at deploy time, fetched during resource resolution). When the profile has no JSON_SCHEMA validator, only JSON syntax is checked. AWS AppConfig supports JSON Schema draft 4.X for Freeform, so the schema is evaluated as draft-4 (any in-document `$schema` declaring a different draft is ignored).
- **Freeform YAML / text**: syntax only — JSON Schema cannot apply.
- **LAMBDA validators are skipped**: they run only inside AWS, so a passing `validate` does not guarantee a LAMBDA validator will pass at deploy time.
- Local validation approximates AWS: regex flavor and undocumented internals may differ, so in rare cases a schema-passing config can still be rejected at `run`.

#### Exit Codes

- `0`: all targets valid
- `1`: validation failed, or a general error (AWS resolution, I/O)

#### Examples

```bash
# Validate every environment before deploying (quote globs; see Multi-config Mode)
apcdeploy validate -c 'environments/*.yml' --continue-on-error
```

### rollback command

Stops an ongoing deployment by calling AWS AppConfig `StopDeployment`. Only in-flight deployments (`DEPLOYING`, `BAKING`, `VALIDATING`, or `ROLLING_BACK`) can be stopped — terminal deployments cannot be rolled back through this command (use Git-based revert + `apcdeploy run` instead).

#### Usage

```bash
apcdeploy rollback -c apcdeploy.yml --yes
```

#### Flags

| Flag | Description |
|---|---|
| `-y, --yes` | Skip the confirmation prompt. Required in non-interactive environments. |

#### Operation Details

1. Find the current ongoing deployment for the environment (errors if none is in-flight: `DEPLOYING` / `BAKING` / `VALIDATING` / `ROLLING_BACK`)
2. If `--yes` is not set, check TTY and prompt for confirmation (errors before any AWS write if no TTY)
3. Call `StopDeployment`; AppConfig transitions the deployment to `ROLLED_BACK` and reverts to the previous version

#### Behavior

- **Scope**: stops in-flight deployments only. Does **not** use AppConfig's `AllowRevert` feature; does **not** touch completed deployments.
- **Local files unchanged**: `rollback` makes no changes to local files. After it succeeds, local files no longer match the deployed (reverted) configuration.
- **Post-rollback sync**: choose one before continuing:
  - **Accept the reverted state**: `apcdeploy pull -c apcdeploy.yml` to bring local files in line with the previous version.
  - **Fix and redeploy**: edit the local data file, then `apcdeploy diff` → `apcdeploy run`.

For reverting a *completed* deployment, use Git-based rollback:

```bash
git checkout <commit-hash> -- data.json
apcdeploy run -c apcdeploy.yml
```

#### Errors

| Cause | Resolution |
|---|---|
| No ongoing deployment in an in-flight state (`DEPLOYING` / `BAKING` / `VALIDATING` / `ROLLING_BACK`) (exit code `2`) | Nothing to stop. If you intended to revert a completed deployment, use Git revert + `apcdeploy run`. |

#### Exit Codes

- `0`: success
- `1`: AWS error, user declined the prompt, or TTY error when `--yes` is missing in a non-interactive environment
- `2`: no ongoing deployment to stop

#### Examples

```bash
# Stop the current rollout (non-interactive)
apcdeploy rollback -c apcdeploy.yml --yes

# Rollback then sync local files to the reverted state
apcdeploy rollback -c apcdeploy.yml --yes
apcdeploy pull -c apcdeploy.yml

# Rollback, fix the data, and redeploy
apcdeploy rollback -c apcdeploy.yml --yes
# (edit data.json)
apcdeploy diff -c apcdeploy.yml
apcdeploy run -c apcdeploy.yml
```

### edit command

Fetches the deployed configuration, opens it in `$EDITOR`, and deploys the result. Does not use `apcdeploy.yml`.

**AI agents must not run this command.** There is no non-interactive mode (`$EDITOR` is always launched, and a TTY is required). When a user wants this behavior, suggest the equivalent: `apcdeploy pull` → edit the data file programmatically → `apcdeploy run`.

For human users, `edit` is a quick-fix workflow when there is already a prior deployment for the target profile + environment. Target resolution accepts the same flags as `init` (`--region` / `--app` / `--profile` / `--env`); omitted flags fall through to interactive prompts. The deployment strategy of the latest deployment is reused unless overridden with `--deployment-strategy`. `--wait-deploy` / `--wait-bake` / `--timeout` / `--description` behave as in `run` (and `--wait-deploy` / `--wait-bake` are mutually exclusive).

Validation (2 MB size limit, JSON/YAML syntax check), idempotency (skip when normalized content is unchanged), and the in-progress-deployment guard all match `run`. `$EDITOR` defaults to `vi` and is invoked through `sh -c` like `git`'s `GIT_EDITOR`, so avoid shell metacharacters in CI.

```bash
apcdeploy edit --region us-west-2 --app my-app --profile my-profile --env production
```

#### Exit Codes

- `0`: success (including no-op when content is unchanged)
- `1`: general error, validation failure, or editor non-zero exit
- `2`: no prior deployment exists for the profile + environment

### context command

Prints the contents of `llms.md` (this document) to stdout. The content is embedded in the binary at build time, so the command works offline and does not invoke AWS.

#### Usage

```bash
apcdeploy context              # full document to stdout
apcdeploy context | less       # paged
apcdeploy context | grep ...   # search
```

Global flags (`-c`, `--silent`) have no effect on this command.

## Global Flags

Available for all commands:

- `-c, --config <path>`: Configuration file path (default: `apcdeploy.yml`). For `run` / `diff` / `pull` / `validate` it may be repeated, comma-separated, or a quoted glob pattern to operate on several configs in one invocation (see "Multi-config Mode" below); all other commands accept exactly one `-c` and reject multiple.
- `-s, --silent`: Suppress verbose output (see "Silent Mode" below).

## Multi-config Mode (run / diff / pull / validate)

`run`, `diff`, `pull`, and `validate` support running against multiple configurations in one invocation. Pass `-c` repeatedly, comma-separate values, or pass a **quoted** glob pattern:

```bash
apcdeploy diff -c environments/dev.yml -c environments/stg.yml -c environments/prod.yml
apcdeploy run  -c 'environments/*.yml' --parallel 3 --wait-bake
apcdeploy pull -c 'environments/*.yml' --continue-on-error
apcdeploy validate -c 'environments/*.yml' --continue-on-error
```

Behavior:

- Glob patterns are expanded by apcdeploy itself (via Go's `filepath.Glob`), so they **must be quoted** (`-c 'environments/*.yml'`). An unquoted glob is expanded by the shell into positional arguments; apcdeploy rejects positional arguments with an error pointing back to quoting, rather than silently using only the first match. A pattern matching no files is an error; duplicates across patterns are de-duplicated preserving order.
- Comma-separated values (`-c a.yml,b.yml`) are split by the CLI framework (Cobra), not by apcdeploy. Because of this, a path that itself contains a comma will be split incorrectly — repeat `-c` for such paths.
- Each `-c` is loaded and validated up-front; any single load failure aborts the batch before any AWS call.
- Targets are identified by the 4-tuple `region/app/profile/env`. Two configs that resolve to the same identifier produce `ErrDuplicateTarget`.
- Default execution is fully parallel (equivalent to `--parallel 0`). Use `--parallel N` to cap concurrency or `--parallel 1` for strict serial order.
- Default failure mode is fail-fast: queued targets that haven't started yet are reported as `⊘ skipped (fail-fast)`. Use `--continue-on-error` to run every target regardless of failures.
- Each target's `--timeout` is independent (it is per-target, not a global wall-clock cap).
- After all targets settle, a single aggregate line is printed: `N ok, N no-op, N failed [(elapsed)]`. Failed targets are also expanded into an `Errors:` section with optional `Resolution:` hints.
- `diff`'s stdout is buffered per target and emitted in argument order (so the combined diff stream is deterministic regardless of completion order). Each per-target body — including single-target invocations — is prefixed with `=== <region>/<app>/<profile>/<env> ===`. To pipe a body into `patch` / `git apply`, strip the header lines first (e.g. `apcdeploy diff -c x.yml | sed '/^=== /d'`).
- **For AI assistants**: prefer single-target invocations unless the multi-config behavior is explicitly desired. Multi-target output is denser and harder to read step-by-step; failures are aggregated at the end of the run.

## Silent Mode

`--silent` (alias `-s`) suppresses Reporter chatter on stderr (progress, headers, info, warnings). Errors (stderr) and primary payloads — `Data` (e.g. `get`'s body, `ls-resources --json`), `Diff` (`diff`'s output), and the final state line for `status --silent` — are always emitted.

**For AI agents**: do not pass `--silent`. The suppressed progress lines are valuable for debugging when something goes wrong.

```bash
apcdeploy diff -c apcdeploy.yml --silent
apcdeploy ls-resources --region us-west-2 --json --silent > resources.json
```

## Troubleshooting

Cross-cutting issues that span commands. Command-specific errors live in each command's `Errors` section.

### Resource Not Found

```txt
Error: application "my-app" not found in region us-west-2
```

The named resource does not exist in the queried region/account. Check:

- The resource exists in AWS Console.
- `region:` in `apcdeploy.yml` (or `--region` for `init` / `ls-resources` / `edit`) matches the account the resource lives in.
- The resolved AWS credentials point at the correct account (`aws sts get-caller-identity`).

### Authentication Error

```txt
Error: failed to load AWS credentials
```

Resolve via the standard AWS SDK chain:

```bash
aws sts get-caller-identity        # verify which credentials are active
aws configure                      # configure shared profile
# Or: AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY / AWS_REGION env vars
```

### Debugging Tips

- Drop `--silent` to see Reporter progress on stderr.
- Use `diff` before `run` to verify the change set.
- Use `status` to monitor an active deployment without waiting.
- Use `get` (or `pull`) to inspect what is actually deployed.

## Best Practices

### Per-Environment Configuration Files

Keep one `apcdeploy.yml` per environment, side-by-side with its data file:

```
environments/
├── dev/{apcdeploy.yml,data.json}
├── staging/{apcdeploy.yml,data.json}
└── production/{apcdeploy.yml,data.json}
```

Deploy a single environment with `-c`:

```bash
apcdeploy run -c environments/production/apcdeploy.yml
```

Or deploy several at once via multi-config mode (see "Multi-config Mode" above):

```bash
apcdeploy run -c 'environments/*/apcdeploy.yml' --parallel 3
```

### CI/CD Pattern

Gate deployment on `diff --exit-nonzero` so the pipeline is a no-op when nothing changed:

```yaml
- name: Deploy to AppConfig
  run: |
    if apcdeploy diff -c apcdeploy.yml --exit-nonzero --silent; then
      echo "No changes to deploy"
    else
      apcdeploy run -c apcdeploy.yml --silent
    fi
```

`--wait-*` is intentionally omitted here; check rollout progress with a follow-up `status` step if needed.

## Security and Access Control

### Required IAM Permissions

To use `apcdeploy`, the following AWS AppConfig IAM permissions are required:

#### Basic Permissions (required for all commands)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "appconfig:ListApplications",
        "appconfig:ListConfigurationProfiles",
        "appconfig:ListEnvironments",
        "appconfig:ListDeploymentStrategies",
        "appconfig:GetConfigurationProfile"
      ],
      "Resource": "*"
    }
  ]
}
```

Additionally, the interactive region picker on `init` (when `--region` is omitted) calls `account:ListRegions`. Grant it only if you intend to use interactive `init`; non-interactive flows (`--region` supplied) do not need it.

`validate` needs only these basic permissions: it resolves the profile and reads its validators (`GetConfigurationProfile`) without fetching any deployment.

#### Deployment Permissions (run, edit, diff, status, pull, and rollback commands)

```json
{
  "Effect": "Allow",
  "Action": [
    "appconfig:CreateHostedConfigurationVersion",
    "appconfig:StartDeployment",
    "appconfig:GetDeployment",
    "appconfig:GetHostedConfigurationVersion",
    "appconfig:ListDeployments",
    "appconfig:StopDeployment"
  ],
  "Resource": "*"
}
```

#### Data Retrieval Permissions (get command)

```json
{
  "Effect": "Allow",
  "Action": [
    "appconfig:StartConfigurationSession",
    "appconfig:GetLatestConfiguration"
  ],
  "Resource": "*"
}
```

### Security Best Practices

1. **Principle of least privilege**: Grant only necessary permissions
2. **Resource restrictions**: Restrict permissions to specific applications or resources where possible
3. **Credential management**: Do not hardcode AWS credentials; use IAM roles or temporary credentials
4. **Audit logs**: Use CloudTrail to record API calls

## FAQ

### Q1: Can I deploy to multiple environments at once?

Yes. Pass `-c` repeatedly to `run`, `diff`, or `pull` (see "Multi-config Mode"). For example:

```bash
apcdeploy run -c environments/dev.yml -c environments/stg.yml -c environments/prod.yml --parallel 3
```

Single-target invocations still work the same way; multi-target requires `region:` to be set in each yml.

### Q2: Does it support both FeatureFlags and Freeform configuration profiles?

Yes. Content-Type is detected automatically. For FeatureFlags profiles, the metadata fields `_createdAt` and `_updatedAt` are excluded from diff/idempotency comparisons.

## Related Resources

- [AWS AppConfig Official Documentation](https://docs.aws.amazon.com/appconfig/latest/userguide/what-is-appconfig.html)
- [AWS AppConfig Feature Flags Reference](https://docs.aws.amazon.com/appconfig/latest/userguide/appconfig-type-reference-feature-flags.html)
- [AWS AppConfig Quotas](https://docs.aws.amazon.com/appconfig/latest/userguide/appconfig-creating-configuration-and-profile-quotas.html)
- [apcdeploy GitHub Repository](https://github.com/koh-sh/apcdeploy)
