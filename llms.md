# llms.md

This file provides guidelines for AI assistants when using the `apcdeploy` command.

## Overview of apcdeploy

`apcdeploy` is a declarative CLI tool for managing AWS AppConfig deployments. It allows you to manage AppConfig applications, configuration profiles, and environments as "code" using a YAML configuration file (`apcdeploy.yml`).

### Capabilities

- List available AWS AppConfig resources (`ls-resources`)
- Auto-generate configuration files from existing AWS AppConfig resources (`init`)
- Deploy configuration changes (`run`)
- Compare differences between local files and deployed configurations (`diff`)
- Monitor deployment status (`status`)
- Retrieve deployed configurations (`get`)

### Important Constraints

- **Supports AWS AppConfig hosted configuration store only**
- AppConfig resources (applications, configuration profiles, environments, deployment strategies) must be created in AWS beforehand
- This tool is for managing existing resources and does not create resources itself

### Supported Content Types

1. **JSON** (`.json` files)
   - Automatic validation and formatting
   - Metadata fields (`_createdAt`, `_updatedAt`) in FeatureFlags profiles are automatically ignored during diff calculations

2. **YAML** (`.yaml` or `.yml` files)
   - Automatic validation and formatting
   - FeatureFlags profile metadata is automatically ignored

3. **Plain Text** (`.txt` or other extensions)
   - Deployed as-is

### Important: TTY Requirements for AI Agents

**Critical for AI Assistants**: Some commands require interactive terminal (TTY) access. AI agents operating in non-interactive environments must follow these rules:

1. **`init` command**: ALWAYS provide all flags (`--region`, `--app`, `--profile`, `--env`)
   - Without all flags, the command attempts interactive prompts and will fail with TTY error
   - Error message: `interactive mode requires a TTY: please provide --region, --app, --profile, and --env flags`

2. **`get` command**: Use with caution as it incurs AWS API charges
   - When necessary in non-interactive environments, use the `--yes` (or `-y`) flag to skip confirmation
   - Without this flag, the command shows a confirmation prompt and will fail with TTY error
   - Error message: `interactive mode requires a TTY: use --yes to skip confirmation`

3. Other commands (`run`, `diff`, `status`) do not require TTY and work in non-interactive environments

## Recommended Usage Flows

**Note for AI Assistants**: The flows below include interactive commands and text editors. When using these flows via AI agents, always use non-interactive modes (with all flags specified) and programmatic file operations instead of interactive editors like `vim`.

### Initial Setup Flow

Recommended procedure when starting with existing AWS AppConfig resources:

```bash
# 1. Discover available resources (especially useful for AI agents)
apcdeploy ls-resources --region us-west-2

# Or output as JSON for programmatic parsing
apcdeploy ls-resources --region us-west-2 --json

# 2. Initialize (for human users: interactive mode; for AI agents: specify all flags)
# Human users:
apcdeploy init

# AI agents (non-interactive with all flags):
apcdeploy init --region us-west-2 --app my-app --profile my-profile --env production

# 3. Review the generated files
# - apcdeploy.yml: Deployment configuration
# - data.json/data.yaml/data.txt: Current configuration content

# 4. Edit configuration content as needed
# Human users: vim data.json
# AI agents: Use Write or Edit tools to modify the file programmatically

# 5. Preview changes
apcdeploy diff -c apcdeploy.yml

# 6. Execute deployment
apcdeploy run -c apcdeploy.yml

# 7. Check deployment status
apcdeploy status -c apcdeploy.yml
```

### Daily Change Management Flow

```bash
# 1. Edit configuration file
# Human users: vim data.json
# AI agents: Use Write or Edit tools to modify the file programmatically

# 2. Review changes
apcdeploy diff -c apcdeploy.yml

# 3. Automated check in CI/CD (optional)
apcdeploy diff -c apcdeploy.yml --exit-nonzero --silent

# 4. Execute deployment
apcdeploy run -c apcdeploy.yml

# 5. Check status (as needed)
apcdeploy status -c apcdeploy.yml
```

### Troubleshooting Flow

```bash
# Check deployment status details
apcdeploy status -c apcdeploy.yml

# For more detailed information, check AWS Console
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

# Optional: AWS region (uses AWS SDK default if omitted)
region: us-west-2
```

### data_file Path Resolution

- **Relative path**: Interpreted as relative to the directory containing `apcdeploy.yml`
  - Example: `data.json` → `data.json` in the same directory as `apcdeploy.yml`
  - Example: `config/data.json` → `config/data.json` under the `apcdeploy.yml` directory
- **Absolute path**: Used as-is
  - Example: `/home/user/configs/data.json`

### Deployment Strategy Examples

#### How to List Available Deployment Strategies

To see all available deployment strategies (both AWS pre-defined and custom strategies you've created), use the AWS CLI:

```bash
# List all deployment strategies
aws appconfig list-deployment-strategies

# Format output as table for easier reading
aws appconfig list-deployment-strategies --query 'Items[*].[Name,Id,Description]' --output table

# Filter by specific criteria (e.g., only show strategies with names)
aws appconfig list-deployment-strategies --query 'Items[*].[Name,GrowthFactor,FinalBakeTimeInMinutes]' --output table
```

This command returns both:

- **AWS pre-defined strategies**: Start with `AppConfig.` prefix (e.g., `AppConfig.Linear`)
- **Custom strategies**: User-created strategies with custom names

#### Pre-defined Deployment Strategies

Common AWS pre-defined deployment strategies:

- `AppConfig.Linear` (AWS Recommended): Deploy 20% every 6 minutes (30 minutes total), for production environments
- `AppConfig.Canary10Percent20Minutes` (AWS Recommended): Exponentially increase by 10% over 20 minutes, recommended for production deployments
- `AppConfig.AllAtOnce` (Quick): Deploy to all targets immediately
- `AppConfig.Linear50PercentEvery30Seconds` (Testing/Demo): Deploy 50% every 30 seconds (1 minute total), for testing and demo purposes

Each strategy monitors CloudWatch Alarms and automatically rolls back if issues are detected.

#### Using Custom Deployment Strategies

You can create your own deployment strategies in AWS AppConfig and reference them by name in `apcdeploy.yml`:

```yaml
# Example using a custom strategy
deployment_strategy: MyCustomStrategy
```

To create a custom deployment strategy, use the AWS Console or AWS CLI. See [AWS AppConfig Deployment Strategies](https://docs.aws.amazon.com/appconfig/latest/userguide/appconfig-creating-deployment-strategy.html) for details.

Reference: [AWS AppConfig Pre-defined Deployment Strategies](https://docs.aws.amazon.com/appconfig/latest/userguide/appconfig-creating-deployment-strategy-predefined.html)

## Command Reference

### ls-resources command

Lists all AWS AppConfig resources in a hierarchical view. By default, shows only applications, configuration profiles, and environments. Deployment strategies can be optionally included using the `--show-strategies` flag. This command is especially useful for AI agents and automation tools to discover available resources before running the `init` command.

#### Usage

```bash
# List resources in default region
apcdeploy ls-resources

# List resources in specific region
apcdeploy ls-resources --region us-east-1

# Include deployment strategies in output
apcdeploy ls-resources --show-strategies

# Output as JSON (useful for scripts and AI agents)
apcdeploy ls-resources --json

# Suppress progress messages, show only results
apcdeploy ls-resources --silent
```

#### Flags

- `--region <region>`: AWS region (uses AWS SDK default if not specified)
- `--json`: Output in JSON format
- `--show-strategies`: Include deployment strategies in output (default: false)

#### Operation Details

1. **Region determination**: Use specified region or AWS SDK default
2. **List applications**: Fetch all AppConfig applications in the region
3. **List profiles and environments**: For each application, fetch configuration profiles and environments
4. **List deployment strategies** (optional): If `--show-strategies` is set, fetch all deployment strategies in the region (both AWS pre-defined and custom)
5. **Format output**: Display in human-readable format or JSON

#### Output Format

**Human-readable format** (default, without `--show-strategies`):
```
Region: us-east-1

Applications:
  [1] my-app (ID: abc123)
      Configuration Profiles:
        - my-profile (ID: prof-123)
        - feature-flags (ID: prof-456)
      Environments:
        - dev (ID: env-111)
        - production (ID: env-222)

  [2] another-app (ID: xyz789)
      Configuration Profiles:
        - config (ID: prof-789)
      Environments:
        - staging (ID: env-333)
```

**With `--show-strategies` flag**:
```
Region: us-east-1

Deployment Strategies:
  - AppConfig.AllAtOnce (ID: a1b2c3d4)
    Description: Quick deployment to all targets at once
    Deployment Duration: 0 minutes
    Final Bake Time: 0 minutes
    Growth Factor: 100.0%
    Growth Type: LINEAR
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

  [2] another-app (ID: xyz789)
      Configuration Profiles:
        - config (ID: prof-789)
      Environments:
        - staging (ID: env-333)
```

**JSON format** (default `--json`, without `--show-strategies`):
```json
{
  "region": "us-east-1",
  "applications": [
    {
      "name": "my-app",
      "id": "abc123",
      "configuration_profiles": [
        {
          "name": "my-profile",
          "id": "prof-123"
        },
        {
          "name": "feature-flags",
          "id": "prof-456"
        }
      ],
      "environments": [
        {
          "name": "dev",
          "id": "env-111"
        },
        {
          "name": "production",
          "id": "env-222"
        }
      ]
    }
  ],
  "deployment_strategies": []
}
```

**JSON format with `--show-strategies`**:
```json
{
  "region": "us-east-1",
  "applications": [
    {
      "name": "my-app",
      "id": "abc123",
      "configuration_profiles": [
        {
          "name": "my-profile",
          "id": "prof-123"
        }
      ],
      "environments": [
        {
          "name": "production",
          "id": "env-222"
        }
      ]
    }
  ],
  "deployment_strategies": [
    {
      "name": "AppConfig.AllAtOnce",
      "id": "a1b2c3d4",
      "description": "Quick deployment to all targets at once",
      "deployment_duration_in_minutes": 0,
      "final_bake_time_in_minutes": 0,
      "growth_factor": 100,
      "growth_type": "LINEAR"
    },
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

#### Notes

- **For AI Assistants**: This command is specifically designed to help AI agents discover available resources without requiring AWS CLI. Use this command instead of AWS CLI when listing resources for the `init` command.
- **No configuration file required**: This command does not require `apcdeploy.yml`
- **AWS credentials required**: AWS CLI configuration or equivalent credentials are required
- **Read-only operation**: This command only queries AWS resources and does not modify anything
- **No TTY required**: Can be used in non-interactive environments without any issues
- **Deployment strategies**: By default, deployment strategies are not displayed. Use `--show-strategies` flag to include them in the output. This is useful when you need to choose a deployment strategy for the `run` command.

#### Examples

```bash
# Discover resources for init command (AI agent workflow)
# 1. List available resources
apcdeploy ls-resources --region us-west-2 --json

# 2. Parse JSON output to extract resource names (using jq or similar)
APPS=$(apcdeploy ls-resources --region us-west-2 --json | jq -r '.applications[].name')

# 3. Use discovered resource names for init command
apcdeploy init --region us-west-2 --app my-app --profile my-profile --env production

# List resources including deployment strategies
apcdeploy ls-resources --region us-east-1 --show-strategies

# Human workflow - view available resources before interactive init
apcdeploy ls-resources --region us-east-1
apcdeploy init  # Interactive mode will show the same resources

# Use in scripts
apcdeploy ls-resources --region us-west-2 --silent > resources.txt
```

### Global Flags

Available for all commands:

- `-c, --config <path>`: Configuration file path (default: `apcdeploy.yml`)
- `-s, --silent`: Suppress verbose output, show only essential information (useful for CI/CD and scripting)
  - **Note for AI Assistants**: Do not use `--silent` when executing commands via AI agents. Verbose output is essential for debugging and understanding command execution.

### init command

Generates `apcdeploy.yml` and configuration data files from existing AWS AppConfig resources.

#### Usage

```bash
# Interactive mode (for human users)
apcdeploy init

# Non-interactive mode (for AI agents and automation - all flags specified)
apcdeploy init --region us-west-2 --app my-app --profile my-profile --env production

# Specify output destinations
apcdeploy init -c custom-config.yml -o custom-data.json

# Overwrite existing files
apcdeploy init -f
```

#### Flags

- `--region <region>`: AWS region (interactive prompt if omitted)
- `--app <name>`: Application name (interactive prompt if omitted)
- `--profile <name>`: Configuration profile name (interactive prompt if omitted)
- `--env <name>`: Environment name (interactive prompt if omitted)
- `-c, --config <path>`: Output configuration file path (default: `apcdeploy.yml`)
- `-o, --output-data <path>`: Output data file path (auto-determined from content type if omitted: `data.json`, `data.yaml`, `data.txt`)
- `-f, --force`: Overwrite existing files without confirmation

#### Operation Details

**Prerequisites**: This command assumes that AWS AppConfig resources (application, configuration profile, environment) are already created. It does not create resources.

1. **Region Selection** (when `--region` not specified)
   - Select from available AWS regions
   - Display AWS Account ID for the selected region

2. **Application Selection** (when `--app` not specified)
   - Select from AppConfig applications in the specified region

3. **Configuration Profile Selection** (when `--profile` not specified)
   - Select from configuration profiles in the selected application

4. **Environment Selection** (when `--env` not specified)
   - Select from environments in the selected application

5. **Fetch Configuration and Generate**
   - Fetch current configuration content from AWS
   - Auto-detect Content-Type
   - Generate `apcdeploy.yml`
   - Generate configuration data file (extension determined by content type)

#### Notes

- **For AI Assistants (CRITICAL)**: When using this command programmatically or via AI agents, **YOU MUST** always use the non-interactive mode by specifying all required flags (`--region`, `--app`, `--profile`, `--env`). Omitting any flag will cause the command to attempt interactive prompts and fail with a TTY error. **Do not use the `--silent` flag** - verbose output is essential for debugging and understanding command execution. To obtain the necessary values:
  - Use the `ls-resources` command to retrieve available resources:

    ```bash
    # List all resources (human-readable format)
    apcdeploy ls-resources --region us-west-2

    # Or get JSON output for programmatic parsing
    apcdeploy ls-resources --region us-west-2 --json
    ```

  - **Fallback approach**: If `ls-resources` is not available, ask the user to provide the application name, configuration profile name, and environment name
- **All flags are optional**: If not specified, you can select interactively through prompts
- **Partial flag specification**: If only some flags are specified, prompts will appear only for unspecified items
- **Existing file protection**: By default, does not overwrite existing files. Use the `-f` flag to overwrite
- **AWS credentials required**: AWS CLI configuration or equivalent credentials are required
- **IAM permissions**:
  - When selecting region interactively (`--region` not specified), `account:ListRegions` permission is required (to retrieve the list of enabled regions)
  - When specifying region directly with `--region` flag, this permission is not required

#### Examples

```bash
# Fully interactive (for human users)
apcdeploy init

# Specify only region and app, select the rest
apcdeploy init --region us-east-1 --app my-app

# Output to different directory
apcdeploy init -c /path/to/config.yml -o /path/to/data.json

# Use in CI/CD (all specified + silent mode)
apcdeploy init --region us-west-2 --app my-app --profile my-profile --env prod -f --silent

# AI agent workflow example (recommended)
# 1. Get available resources using ls-resources command
apcdeploy ls-resources --region us-west-2

# Or get JSON output for parsing
apcdeploy ls-resources --region us-west-2 --json

# 2. Run init with obtained values
apcdeploy init --region us-west-2 --app my-app --profile my-profile --env production

# 3. (Optional) Modify apcdeploy.yml to use a specific deployment strategy
# Edit the generated apcdeploy.yml to set deployment_strategy field
```

### run command

Deploys configuration changes to AWS AppConfig.

#### Usage

```bash
# Basic deployment
apcdeploy run -c apcdeploy.yml

# Wait for deployment phase to complete
apcdeploy run -c apcdeploy.yml --wait-deploy

# Wait for deployment and baking phase to complete
apcdeploy run -c apcdeploy.yml --wait-bake

# Deploy even when there are no differences
apcdeploy run -c apcdeploy.yml --force

# Specify timeout
apcdeploy run -c apcdeploy.yml --wait-bake --timeout 900
```

#### Flags

- `--wait-deploy`: Wait for deployment phase to complete (until baking starts)
- `--wait-bake`: Wait for complete deployment including baking phase
- `--force`: Deploy even when content is unchanged
- `--timeout <seconds>`: Timeout in seconds for deployment wait (default: 600)

**Important**: `--wait-deploy` and `--wait-bake` are mutually exclusive and cannot be used together.

#### Operation Details

1. **Load configuration file**: Load `apcdeploy.yml` and `data_file`
2. **Resolve resource names**: Resolve application, profile, and environment names to AWS IDs
3. **Diff check**: Compare local file with latest deployed version
   - If content is identical, automatically skips by default (can be overridden with `--force`)
4. **Create version**: Create a new hosted configuration version
5. **Start deployment**: Start deployment to the specified environment
6. **Wait** (optional):
   - `--wait-deploy`: Wait for DEPLOYING → BAKING transition
   - `--wait-bake`: Wait for full lifecycle DEPLOYING → BAKING → COMPLETE

#### Deployment Wait Options Comparison

| Option | Wait Behavior | Completion Condition | Use Case |
|--------|---------------|---------------------|----------|
| None (Recommended) | No wait | Exits immediately after deployment starts | Most cases. Check progress separately with `status` command |
| `--wait-deploy` | Deployment phase only | When entering baking state | Cases where you need to synchronously wait for deployment phase completion |
| `--wait-bake` | Complete deployment | When deployment becomes COMPLETE | Cases where you need to synchronously wait for full deployment completion |

#### Idempotency

- **Auto-skip feature**: If local file content is identical to deployed version, deployment is automatically skipped
- **FeatureFlags special handling**: For FeatureFlags profiles, metadata fields (`_createdAt`, `_updatedAt`) are excluded from comparison
- **Force deploy**: Use the `--force` flag to deploy even when content is unchanged

#### Notes

- **AWS credentials required**: AWS CLI configuration or equivalent credentials are required
- **Existing resources required**: Application, profile, environment, and deployment strategy must exist in AWS
- **In-progress deployments**: If there is an in-progress deployment (DEPLOYING or BAKING state) for the same environment, a new deployment cannot be started. You must wait for the existing deployment to complete or stop it from the AWS Console
- **Timeout settings**: For deployment strategies that take a long time, set `--timeout` appropriately
- **Error handling**: If an error occurs during deployment, it exits with an appropriate error message
- **Recommended usage**: For basically all situations, it is recommended not to use `--wait-deploy` or `--wait-bake` options, and instead check progress separately with the `status` command after deployment starts

#### Examples

```bash
# Basic deployment (recommended)
apcdeploy run -c apcdeploy.yml

# Use in CI/CD pipeline (recommended)
apcdeploy run -c apcdeploy.yml --silent

# Check status after deployment (recommended)
apcdeploy run -c apcdeploy.yml
apcdeploy status -c apcdeploy.yml

# Deploy even when there are no differences
apcdeploy run -c apcdeploy.yml --force

# Only when waiting is needed in specific situations
apcdeploy run -c apcdeploy.yml --wait-deploy
apcdeploy run -c apcdeploy.yml --wait-bake --timeout 1800
```

### diff command

Displays differences between local file and deployed configuration.

#### Usage

```bash
# Display differences
apcdeploy diff -c apcdeploy.yml

# Use in CI (exit code 1 if differences exist)
apcdeploy diff -c apcdeploy.yml --exit-nonzero

# Display only differences in silent mode
apcdeploy diff -c apcdeploy.yml --silent
```

#### Flags

- `--exit-nonzero`: Exit with code 1 if differences exist (useful in CI/CD)

#### Operation Details

1. **Load configuration file**: Load local `apcdeploy.yml` and `data_file`
2. **Fetch deployed configuration**: Fetch latest deployed version from AWS
3. **Normalize**: Normalize both configurations (remove FeatureFlags metadata, unify formatting)
4. **Calculate differences**: Calculate differences in unified diff format
5. **Output**: Display differences (or display message if no differences)

#### Output Format

- **Unified Diff Format**: Display differences in standard diff format
  - Lines starting with `-`: Content to be removed from deployed version
  - Lines starting with `+`: Content to be added
- **No differences**: Display "No differences found" message

#### Normalization Process

- **JSON/YAML format unification**: Absorbs differences in indentation and line breaks
- **FeatureFlags metadata exclusion**: `_createdAt` and `_updatedAt` fields are automatically ignored

#### Notes

- **AWS credentials required**: Required to fetch deployed version
- **Content-Type consideration**: JSON/YAML are normalized, but Plain Text is compared byte-by-byte
- **Exit codes**:
  - 0: No differences, or normal exit
  - 1: When `--exit-nonzero` is specified and differences exist
- **Comparison with in-progress configuration**: If there is a deployment in progress (DEPLOYING) or baking (BAKING), it compares with that configuration. Note that if that deployment is rolled back (ROLLED_BACK), the content displayed by the diff command may differ from the actually deployed content

#### Examples

```bash
# Check before deployment
apcdeploy diff -c apcdeploy.yml

# Change detection in CI/CD
if apcdeploy diff -c apcdeploy.yml --exit-nonzero --silent; then
  echo "No changes to deploy"
else
  echo "Changes detected, deploying..."
  apcdeploy run -c apcdeploy.yml
fi
```

### status command

Displays deployment status.

#### Usage

```bash
# Display latest deployment status
apcdeploy status -c apcdeploy.yml

# Display status of specific deployment number
apcdeploy status -c apcdeploy.yml --deployment 3

# Display only status in silent mode
apcdeploy status -c apcdeploy.yml --silent
```

#### Flags

- `--deployment <number>`: Specify deployment number (defaults to latest deployment if omitted)

#### Operation Details

1. **Load configuration file**: Load `apcdeploy.yml`
2. **Resolve resources**: Resolve application and environment names to AWS IDs
3. **Fetch deployment information**: Fetch information for specified (or latest) deployment from AWS
4. **Display status**: Display deployment state, progress, and detailed information

#### Displayed Information

- **Deployment Number**: Deployment number
- **State**: Deployment state
  - `DEPLOYING`: Deployment in progress
  - `BAKING`: Baking (validation phase)
  - `COMPLETE`: Completed
  - `ROLLED_BACK`: Rolled back
- **Percentage Complete**: Completion percentage (%)
- **Configuration Version**: Configuration version number
- **Started At**: Deployment start time

#### Deployment State Details

- **DEPLOYING**: Configuration is being gradually rolled out to targets
- **BAKING**: Rollout to all targets is complete and in validation period
- **COMPLETE**: Deployment is fully completed
- **ROLLED_BACK**: Issues were detected and rolled back to previous version

#### Notes

- **AWS credentials required**: Required to fetch deployment information
- **No deployment exists**: Error message is displayed
- **Exit codes**: 0 on normal exit, 1 on error

#### Examples

```bash
# Check after deployment
apcdeploy run -c apcdeploy.yml
apcdeploy status -c apcdeploy.yml

# Loop to wait for deployment completion (manual monitoring)
while true; do
  apcdeploy status -c apcdeploy.yml --silent
  sleep 10
done

# Check past deployments
apcdeploy status -c apcdeploy.yml --deployment 1
apcdeploy status -c apcdeploy.yml --deployment 2
apcdeploy status -c apcdeploy.yml --deployment 3
```

### get command

Retrieves deployed configuration and displays to stdout.

#### Usage

```bash
# Retrieve with confirmation prompt
apcdeploy get -c apcdeploy.yml

# Skip confirmation and retrieve
apcdeploy get -c apcdeploy.yml -y

# Redirect to file
apcdeploy get -c apcdeploy.yml -y > deployed.json
```

#### Flags

- `-y, --yes`: Skip confirmation prompt (useful for scripts and automation)
  - **For AI Assistants**: Use this flag when executing in non-interactive environments to avoid TTY errors

#### Operation Details

1. **Confirmation prompt**: Warns that AWS AppConfig Data API is billable (can be skipped with `-y`)
2. **Load configuration file**: Load `apcdeploy.yml`
3. **Resolve resources**: Resolve application, profile, and environment names to AWS IDs
4. **Fetch configuration**: Use AWS AppConfig Data API to fetch latest deployed configuration
5. **Output**: Display configuration content to stdout (formatted according to Content-Type)

#### Important Notes

**About AWS AppConfig Data API billing:**

- This command uses AWS AppConfig Data API
- **Charges are incurred per API call**
- Avoid frequent execution and use only when necessary

#### Output Format

- **JSON**: Output as formatted JSON
- **YAML**: Output as formatted YAML
- **Plain Text**: Output as-is

#### Examples

```bash
# Check currently deployed configuration
apcdeploy get -c apcdeploy.yml -y

# Use in scripts (use -y flag to skip confirmation)
DEPLOYED_CONFIG=$(apcdeploy get -c apcdeploy.yml -y)
echo "$DEPLOYED_CONFIG" | jq '.features.new_feature'

# For AI agents in non-interactive environments
apcdeploy get -c apcdeploy.yml --yes
```

### context command

Outputs context information for AI assistants.

#### Usage

```bash
# Output llms.md content
apcdeploy context
```

#### Operation Details

This command outputs the contents of `llms.md` to stdout. The content is embedded in the binary at build time, so no external files are required.

#### Purpose

The `context` command is designed for AI assistants and LLMs to quickly access comprehensive documentation about the `apcdeploy` tool. When working with AI coding assistants, you can pipe this command's output to provide the assistant with detailed information about:

- Command usage and workflows
- Configuration file formats
- Best practices
- Troubleshooting guidance

#### Examples

```bash
# Output full documentation
apcdeploy context

# Use with AI assistants or documentation viewers
apcdeploy context | less

# Search for specific information
apcdeploy context | grep "deployment strategy"
```

#### Notes

- This command does not interact with AWS
- No AWS credentials are required
- The output is static content embedded at build time
- Global flags like `--config` and `--silent` have no effect on this command

## Silent Mode

The `--silent` (or `-s`) flag suppresses verbose output and displays only essential information.

**Important for AI Assistants**: Do not use `--silent` mode when executing commands via AI agents. Verbose output provides critical information for debugging and understanding command execution, which is essential for AI-driven workflows.

### Behavior

- **Suppressed output**: Progress messages, success messages, warnings
- **Always displayed output**: Error messages (stderr), final results (diff output, status, etc.)

### Use Cases

- **CI/CD pipelines**: Reduce log noise and record only important information
- **Scripts**: Obtain machine-readable output
- **Automation**: Eliminate unnecessary messages (not recommended for AI agents)

### Examples

```bash
# Get only differences (no metadata)
apcdeploy diff -c apcdeploy.yml --silent

# Get only status
apcdeploy status -c apcdeploy.yml --silent

# Quiet deployment in CI/CD
apcdeploy run -c apcdeploy.yml --wait-bake --silent
```

## Troubleshooting

### Common Issues and Solutions

#### 1. TTY Error (Non-Interactive Environment)

**Error Examples:**

```txt
Error: interactive mode requires a TTY: please provide --region, --app, --profile, and --env flags
Error: interactive mode requires a TTY: use --yes to skip confirmation
```

**Cause:**

- Attempting to use interactive prompts in a non-interactive environment (CI/CD, scripts, AI agents)
- Missing required flags for `init` command
- Missing `--yes` flag for `get` command

**Solution:**

For `init` command:
```bash
# ALWAYS provide all flags in non-interactive environments
apcdeploy init --region us-west-2 --app my-app --profile my-profile --env production
```

For `get` command:
```bash
# Use --yes flag in non-interactive environments to skip confirmation
apcdeploy get -c apcdeploy.yml --yes
```

#### 2. Resource Not Found

**Error Example:**

```txt
Error: application "my-app" not found in region us-west-2
```

**Solution:**

- Verify that the resource exists in AWS Console
- Check that the region setting in `apcdeploy.yml` is correct
- Verify that AWS credentials are for the correct account

#### 3. Authentication Error

**Error Example:**

```txt
Error: failed to load AWS credentials
```

**Solution:**

```bash
# Check AWS credentials
aws sts get-caller-identity

# Configure credentials
aws configure

# Or set via environment variables
export AWS_ACCESS_KEY_ID=xxx
export AWS_SECRET_ACCESS_KEY=yyy
export AWS_REGION=us-west-2
```

#### 4. Deployment Skipped

**Cause:**

- Local file content is identical to deployed version

**Solution:**

```bash
# Check differences
apcdeploy diff -c apcdeploy.yml

# Deploy even when there are no differences
apcdeploy run -c apcdeploy.yml --force
```

#### 5. Timeout Error

**Error Example:**

```txt
Error: deployment timeout after 600 seconds
```

**Solution:**

```bash
# Extend timeout
apcdeploy run -c apcdeploy.yml --wait-bake --timeout 1800

# Or deploy without waiting and check status separately
apcdeploy run -c apcdeploy.yml
apcdeploy status -c apcdeploy.yml
```

### Debugging Tips

1. **Run in verbose mode**: Remove `--silent` to see detailed progress
2. **Check differences**: Use `diff` command before deployment to verify changes
3. **Monitor status**: Use `status` command during deployment to check progress
4. **Check deployed configuration**: Use `get` command to verify actually deployed content

## Best Practices

### 1. Version Control

```bash
# Manage apcdeploy.yml and data file with Git
git add apcdeploy.yml data.json
git commit -m "Update feature flag configuration"
```

### 2. Per-Environment Configuration Files

```bash
# Separate by environment directories
environments/
├── dev/
│   ├── apcdeploy.yml
│   └── data.json
├── staging/
│   ├── apcdeploy.yml
│   └── data.json
└── production/
    ├── apcdeploy.yml
    └── data.json

# Explicitly specify when deploying
apcdeploy run -c environments/production/apcdeploy.yml
```

### 3. Use in CI/CD

```yaml
# GitHub Actions example
- name: Deploy to AppConfig
  run: |
    # Check differences
    if apcdeploy diff -c apcdeploy.yml --exit-nonzero --silent; then
      echo "No changes to deploy"
    else
      # Deploy only if there are changes (wait option not recommended)
      apcdeploy run -c apcdeploy.yml --silent
    fi
```

### 4. Pre-Deployment Check

Always verify changes with the `diff` command before deploying:

```bash
# Verify changes
apcdeploy diff -c apcdeploy.yml

# Deploy if no issues
apcdeploy run -c apcdeploy.yml
```

### 5. Rollback Strategy

Easily rollback by managing configuration files with Git:

```bash
# Manage configuration files with Git
git add apcdeploy.yml data.json
git commit -m "Update configuration"

# If there are issues, revert to previous version with Git
git revert HEAD
# Or revert to specific commit
git checkout <commit-hash> -- data.json

# Deploy the rolled-back version
apcdeploy run -c apcdeploy.yml
```

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

#### Deployment Permissions (run, diff, and status commands)

```json
{
  "Effect": "Allow",
  "Action": [
    "appconfig:CreateHostedConfigurationVersion",
    "appconfig:StartDeployment",
    "appconfig:GetDeployment",
    "appconfig:GetHostedConfigurationVersion",
    "appconfig:ListDeployments"
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

## AWS AppConfig Limitations

- **Maximum hosted configuration size**:
  - Default: 2 MB
  - Maximum: 4 MB (can request limit increase)
- For details, see [AWS AppConfig Quotas](https://docs.aws.amazon.com/appconfig/latest/userguide/appconfig-creating-configuration-and-profile-quotas.html)

## FAQ

### Q1: Does the init command overwrite existing configurations?

A: By default, it does not overwrite. Use the `-f, --force` flag to overwrite.

### Q2: Can I cancel a deployment in progress?

A: You can interrupt the `apcdeploy` command itself, but the deployment on the AWS AppConfig side will continue. You need to stop the deployment from the AWS Console or AWS CLI.

### Q3: Can I deploy to multiple environments simultaneously?

A: You need to create separate `apcdeploy.yml` files for each environment and deploy sequentially.

### Q4: Does it support both FeatureFlags and Freeform?

A: Yes, it supports both profile types. Content-Type is automatically detected.

### Q5: How do I perform a rollback?

A: If you are managing configuration files with Git, you can rollback by reverting to a previous version and then executing `apcdeploy run`. Alternatively, you can rollback directly on the AppConfig side using the AWS Console/CLI.

## Related Resources

- [AWS AppConfig Official Documentation](https://docs.aws.amazon.com/appconfig/latest/userguide/what-is-appconfig.html)
- [AWS AppConfig Feature Flags Reference](https://docs.aws.amazon.com/appconfig/latest/userguide/appconfig-type-reference-feature-flags.html)
- [AWS AppConfig Quotas](https://docs.aws.amazon.com/appconfig/latest/userguide/appconfig-creating-configuration-and-profile-quotas.html)
- [apcdeploy GitHub Repository](https://github.com/koh-sh/apcdeploy)
