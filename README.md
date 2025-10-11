# apcdeploy

A declarative CLI tool for managing AWS AppConfig deployments. Manage your AppConfig applications, configuration profiles, and environments as code using a simple YAML configuration file.

Note: This tool only supports AWS AppConfig hosted configuration store. You must create AppConfig resources (application, configuration profile, environment, deployment strategy) before using this tool.

## Features

- **Declarative Configuration**: Define your AppConfig resources in `apcdeploy.yml`
- **Deployment Automation**: Deploy configuration changes with a single command
- **Configuration Retrieval**: Fetch currently deployed configuration from AWS AppConfig
- **Diff Previews**: See exactly what will change before deploying
- **Status Monitoring**: Track deployment progress and completion
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
deployment_strategy: AppConfig.Linear50PercentEvery30Seconds
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

This shows the current deployment state (IN_PROGRESS, COMPLETE, or ROLLED_BACK) and progress percentage.

## Configuration File Reference

### apcdeploy.yml

```yaml
# Required: Name of the AppConfig application
application: my-application

# Required: Name of the configuration profile
configuration_profile: my-config-profile

# Required: Name of the environment
environment: production

# Optional: Deployment strategy (defaults to AppConfig.Linear50PercentEvery30Seconds)
deployment_strategy: AppConfig.AllAtOnce

# Required: Path to your configuration data file (relative or absolute)
data_file: data.json

# Optional: AWS region (uses AWS SDK default if omitted)
region: us-west-2
```

### Supported Content Types

- JSON: `.json` files (validated and auto-formatted)
- YAML: `.yaml` or `.yml` files (validated and auto-formatted)
- Plain Text: `.txt` files or any other extension

For FeatureFlags profiles, metadata fields (`_createdAt`, `_updatedAt`) are automatically ignored during diff and deployment comparisons.

## Commands

### Global Flags

All commands support these global flags:

- `-c, --config`: Config file path (default: `apcdeploy.yml`)
- `-s, --silent`: Suppress verbose output, show only essential information (useful for CI/CD and scripting)

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
apcdeploy run -c apcdeploy.yml [--wait] [--force]
```

Options:

- `--wait`: Wait for deployment to complete
- `--force`: Deploy even if content hasn't changed

### diff

Preview configuration changes:

```bash
apcdeploy diff -c apcdeploy.yml [--exit-nonzero]
```

Options:

- `--exit-nonzero`: Exit with code 1 if differences are found (useful in CI)

### status

Check deployment status:

```bash
apcdeploy status -c apcdeploy.yml
```

### get

Retrieve the currently deployed configuration:

```bash
apcdeploy get -c apcdeploy.yml
```
