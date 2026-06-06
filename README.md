# apcdeploy

A declarative CLI tool for managing AWS AppConfig deployments. Roll out configuration changes to existing AppConfig applications, configuration profiles, and environments using a simple YAML configuration file.

Note: This tool only supports AWS AppConfig hosted configuration store. You must create AppConfig resources (application, configuration profile, environment, deployment strategy) before using this tool.

Why not Terraform or CDK? AppConfig's hosted configuration versions are append-only — every deploy produces a new immutable version, and the deployment itself has its own lifecycle (deploy → bake → complete or rollback). That stateful, monotonically-incrementing shape doesn't fit the desired-state model Terraform and CDK assume. Just as importantly, the resource shells (application / profile / environment) and the configuration values inside them often have different owners — the platform team manages the shells in IaC; the application team ships value changes at a faster cadence. `apcdeploy` separates those layers: your IaC owns the resources, this tool owns the value rollout.

![apcdeploy demo](vhs/demo.gif)

## Features

- **Resource Discovery**: List all AWS AppConfig resources (applications, profiles, environments) in a region
- **Declarative Configuration**: Define your AppConfig resources in `apcdeploy.yml`
- **Configuration Retrieval**: Fetch deployed configuration (`get`) or sync local files with deployed state (`pull`)
- **Deployment Rollback**: Stop ongoing deployments with a single command
- **Diff Previews**: See exactly what will change before deploying
- **Multiple Content Types**: Support for both Feature Flags and Freeform configuration profiles
- **Idempotent Deployments**: Automatically skips deployment when local content matches deployed version

## Installation

### Homebrew

```bash
brew install koh-sh/tap/apcdeploy
```

### mise

```bash
mise use -g github:koh-sh/apcdeploy
```

### Pre-built Binary

Download the latest release from the [Releases page](https://github.com/koh-sh/apcdeploy/releases).

## Quick Start Tutorial

> [!TIP]
> Working with an AI coding assistant like Claude Code? Have it run `apcdeploy context` to access comprehensive guidelines and usage details for this tool.

This tutorial walks you through the complete workflow: initializing from existing AWS AppConfig resources, making changes, and deploying.

### Prerequisites

- AWS credentials configured (via `~/.aws/credentials`, environment variables, or IAM role)
- An existing AppConfig application, configuration profile, and environment in AWS

### Step 1: Initialize from Existing Resources

Generate an `apcdeploy.yml` file from your existing AppConfig resources using the interactive mode:

```bash
apcdeploy init
```

The command will guide you through:
1. Selecting an AWS region
2. Choosing an application
3. Selecting a configuration profile
4. Choosing an environment

Alternatively, you can skip the interactive prompts by providing flags:

```bash
apcdeploy init \
  --region us-west-2 \
  --app my-application \
  --profile my-config-profile \
  --env production
```

This creates two files:

- `apcdeploy.yml`: Your deployment configuration
- `data.json` (or `.yaml`/`.txt`): Your current configuration content

Example `apcdeploy.yml`:

```yaml
application: my-application
configuration_profile: my-config-profile
environment: production
deployment_strategy: AppConfig.AllAtOnce  # optional, uses default if omitted
data_file: data.json
region: us-west-2
```

### Step 2: Make Changes

Edit your configuration file (`data.json`, `data.yaml`, etc.):

```bash
# Edit your configuration
vim data.json
```

Example change:

```json
{
  "database": {
    "host": "db.example.com",
    "port": 5432,
    "max_connections": 100
  },
  "features": {
    "cache_enabled": true,
    "debug_mode": false
  }
}
```

For AWS AppConfig Feature Flags format, see the [AWS documentation](https://docs.aws.amazon.com/appconfig/latest/userguide/appconfig-type-reference-feature-flags.html).

### Step 3: Preview Changes

See what will be deployed before actually deploying:

```bash
apcdeploy diff -c apcdeploy.yml
```

This shows a unified diff of changes between your local file and the currently deployed version.

### Step 4: Deploy

Deploy your changes to AWS AppConfig:

```bash
apcdeploy run -c apcdeploy.yml
```

### Step 5: Check Deployment Status

Check the status of your latest deployment:

```bash
apcdeploy status -c apcdeploy.yml
```

This shows the current deployment state (VALIDATING, DEPLOYING, BAKING, COMPLETE, ROLLING_BACK, ROLLED_BACK, or REVERTED) and progress percentage.

## Configuration File Reference

### apcdeploy.yml

```yaml
# Required: Name of the AppConfig application
application: my-application

# Required: Name of the configuration profile
configuration_profile: my-config-profile

# Required: Name of the environment
environment: production

# Optional: Deployment strategy (defaults to AppConfig.AllAtOnce)
deployment_strategy: AppConfig.AllAtOnce

# Required: Path to your configuration data file (relative or absolute)
data_file: data.json

# Required: AWS region
region: us-west-2
```

### Supported Content Types

- JSON: `.json` files (validated and auto-formatted)
- YAML: `.yaml` or `.yml` files (validated and auto-formatted)
- Plain Text: `.txt` files or any other extension

For FeatureFlags profiles, metadata fields (`_createdAt`, `_updatedAt`) are automatically ignored during diff and deployment comparisons.

## Commands

For detailed usage information and advanced features, see [llms.md](./llms.md).

### Global Flags

All commands support these global flags:

- `-c, --config`: Config file path (default: `apcdeploy.yml`). May be passed multiple times for `run` / `diff` / `pull` / `validate` to operate on several configs in one invocation
- `-s, --silent`: Suppress verbose output, show only essential information (useful for CI/CD and scripting)

### ls-resources

List all AWS AppConfig resources in a region:

```bash
apcdeploy ls-resources --region us-west-2
```

This command helps you discover available applications, configuration profiles, and environments before running `init`. It's especially useful for automation and AI-assisted workflows.

Options:

- `--region`: AWS region (uses AWS SDK default if not specified)
- `--json`: Output in JSON format (useful for scripts and automation)
- `--show-strategies`: Include deployment strategies in output

Example with JSON output:

```bash
apcdeploy ls-resources --region us-west-2 --json
```

This command does not require an `apcdeploy.yml` file and is read-only.

### init

Initialize a new `apcdeploy.yml` from existing AWS resources:

```bash
apcdeploy init
```

Flags are optional. If omitted, you will be prompted interactively to select from available resources.

```bash
apcdeploy init \
  --region <region> \
  --app <application> \
  --profile <profile> \
  --env <environment>
```

Options:

- `--region`: AWS region
- `--app`: Application name
- `--profile`: Configuration profile name
- `--env`: Environment name
- `-c, --config`: Output config file path (default: `apcdeploy.yml`)
- `-o, --output-data`: Output data file path (auto-detected from content type if omitted)
- `-f, --force`: Overwrite existing files

### run

Deploy configuration changes:

```bash
apcdeploy run -c apcdeploy.yml [--wait-deploy|--wait-bake] [--force]
```

Pass `-c` multiple times to deploy several configs in one invocation. You can also pass a comma-separated list or a quoted glob pattern (quote it so the shell does not expand it). Comma-separation is handled by the CLI framework, so repeat `-c` for any path that itself contains a comma.

```bash
apcdeploy run -c environments/dev.yml -c environments/stg.yml -c environments/prod.yml --wait-bake
apcdeploy run -c 'environments/*.yml' --wait-bake
```

Options:

- `--wait-deploy`: Wait for deployment phase to complete (until baking starts)
- `--wait-bake`: Wait for complete deployment including baking phase
- `--timeout`: Per-target timeout in seconds for deployment wait (default: 1800)
- `--force`: Deploy even if content hasn't changed
- `--description`: Description attached to the configuration version and deployment (max 1024 Unicode characters, counted by rune). Defaults to `"Deployed by apcdeploy"`; pass `--description ""` to clear it.
- `--parallel`: Maximum concurrent targets when `-c` is repeated (default: 0, meaning all in parallel)
- `--continue-on-error`: Run remaining targets after one fails (default: fail-fast)

Note: `--wait-deploy` and `--wait-bake` are mutually exclusive.

### edit

Edit the currently deployed configuration directly in `$EDITOR` and deploy:

```bash
apcdeploy edit \
  --region <region> \
  --app <application> \
  --profile <profile> \
  --env <environment>
```

Flags are optional. If omitted, you will be prompted interactively to select from available resources. The deployment strategy of the most recent deployment is reused unless `--deployment-strategy` is specified.

Options:

- `--region`: AWS region
- `--app`: Application name
- `--profile`: Configuration profile name
- `--env`: Environment name
- `--deployment-strategy`: Deployment strategy name (defaults to the strategy of the latest deployment)
- `--wait-deploy`: Wait for deployment phase to complete (until baking starts)
- `--wait-bake`: Wait for complete deployment including baking phase
- `--timeout`: Timeout in seconds for deployment wait (default: 1800)
- `--description`: Description attached to the configuration version and deployment (max 1024 Unicode characters, counted by rune). Defaults to `"Deployed by apcdeploy"`; pass `--description ""` to clear it.

**Note:** This command does not use `apcdeploy.yml`. `$EDITOR` is invoked through `sh -c` (matching `git`'s `GIT_EDITOR` behavior), so its value is shell-evaluated — avoid passing values that contain shell metacharacters in CI environments. Exits with code 2 when no prior deployment exists.

### diff

Preview configuration changes:

```bash
apcdeploy diff -c apcdeploy.yml [--exit-nonzero]
```

Pass `-c` multiple times to diff several configs in one invocation. Every diff body — single- or multi-target — is prefixed with `=== <region>/<app>/<profile>/<env> ===`. Strip the header line if you want to pipe the body into `patch` / `git apply`.

```bash
apcdeploy diff -c environments/dev.yml -c environments/stg.yml -c environments/prod.yml
```

Options:

- `--exit-nonzero`: Exit with code 1 if differences are found (useful in CI)
- `--parallel`: Maximum concurrent targets when `-c` is repeated (default: 0, meaning all in parallel)
- `--continue-on-error`: Run remaining targets after one fails (default: fail-fast)

### status

Check deployment status:

```bash
apcdeploy status -c apcdeploy.yml
```

Options:

- `--deployment`: Deployment number to check (defaults to latest)

Exits with code 2 when no deployment exists for the profile + environment.

### get

Retrieve the currently deployed configuration:

```bash
apcdeploy get -c apcdeploy.yml
```

**Note:** This command uses AWS AppConfig Data API which incurs charges per API call. You will be prompted for confirmation before proceeding.

Options:

- `-y, --yes`: Skip confirmation prompt (for scripts and automation)

### pull

Pull the latest deployed configuration and update your local data file:

```bash
apcdeploy pull -c apcdeploy.yml
```

Pass `-c` multiple times to pull several configurations in one invocation:

```bash
apcdeploy pull -c environments/dev.yml -c environments/stg.yml -c environments/prod.yml
```

Options:

- `--parallel`: Maximum concurrent targets when `-c` is repeated (default: 0, meaning all in parallel)
- `--continue-on-error`: Run remaining targets after one fails (default: fail-fast)

This command is useful when configuration changes are made directly in the AWS Console and you want to sync your local files with the deployed state. Exits with code 2 when no prior deployment exists.

### validate

Validate your local data file against its schema without deploying:

```bash
apcdeploy validate -c apcdeploy.yml
```

Pass `-c` multiple times to validate several configurations in one invocation:

```bash
apcdeploy validate -c environments/dev.yml -c environments/stg.yml -c environments/prod.yml
```

Options:

- `--parallel`: Maximum concurrent targets when `-c` is repeated (default: 0, meaning all in parallel)
- `--continue-on-error`: Run remaining targets after one fails (default: fail-fast)

This read-only check does not create a configuration version. FeatureFlags are validated against the built-in AWS schema and their declared constraints; Freeform JSON is validated against the profile's JSON_SCHEMA validator (syntax only when none is present). YAML and text are syntax-checked only, and LAMBDA validators are skipped.

### rollback

Stop an ongoing deployment:

```bash
apcdeploy rollback -c apcdeploy.yml
```

This command stops an in-flight deployment (DEPLOYING, BAKING, VALIDATING, or ROLLING_BACK state) by calling the AWS AppConfig StopDeployment API. It automatically detects the current ongoing deployment and stops it.

**Note:** This only stops deployments currently in progress. It does not revert previously completed deployments. Exits with code 2 when no deployment is currently in flight.

Options:

- `-y, --yes`: Skip confirmation prompt (for scripts and automation)

### context

Output context information for AI assistants:

```bash
apcdeploy context
```
