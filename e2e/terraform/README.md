# E2E Test Infrastructure

This directory manages AWS AppConfig resources required for apcdeploy E2E tests.

## Resources Created

### Application
- **apcdeploy-e2e-test**: Application for E2E testing

### Environments
- **dev**: Development environment (used for basic tests)
- **staging**: Staging environment (used for multi-environment tests)

### Configuration Profiles
- **json-freeform**: Freeform JSON profile (basic workflow tests)
- **json-featureflags**: FeatureFlags profile (metadata normalization tests)
- **yaml-config**: YAML profile (YAML format tests)
- **text-config**: Plain text profile (text format tests)
- **error-test**: Error testing profile (various error scenarios)
- **json-validated**: Freeform JSON with a JSON_SCHEMA validator (`validate` remote-schema tests)
- **json-lambda**: Freeform JSON with a LAMBDA validator (`validate` lambda-skip tests)

### Deployment Strategies
- **E2E-Test-Strategy**: Custom strategy (instant deployment, no bake time)
- **E2E-Slow-Strategy**: Custom strategy (1 min deployment; rollback and timeout tests)

The tests also rely on the AWS predefined `AppConfig.AllAtOnce` strategy,
which AWS provides in every account (not created here). MiniStack does not
provide predefined strategies.

### Lambda Validator (supporting the json-lambda profile)
- A minimal Lambda function, IAM role, and AppConfig invoke permission. The
  function is never invoked by tests — `validate` skips LAMBDA validators — but
  AppConfig requires a real, invokable function to attach the validator.

## Setup

```bash
# Initialize
terraform init

# Preview changes
terraform plan

# Create resources
terraform apply

# Destroy resources
terraform destroy
```

### Local (MiniStack)

The same configuration also provisions the resources on the MiniStack
emulator for local E2E runs. Use the mise tasks from the repository root
rather than running Terraform directly, so the endpoint, dummy credentials,
and the separate `local` workspace are applied:

```bash
mise run e2e-local-full   # up -> setup -> run -> clean
```

Every Terraform task pins `TF_WORKSPACE` (`default` for real AWS, `local`
for MiniStack), so the two states never mix.

### Provider lock file

`.terraform.lock.hcl` is committed so every run (local, CI, real AWS) uses
the same provider versions. When upgrading providers, keep hashes for all
platforms the suite runs on:

```bash
terraform init -upgrade
terraform providers lock \
  -platform=linux_arm64 -platform=linux_amd64 \
  -platform=darwin_arm64 -platform=darwin_amd64
```

## Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `region` | `ap-northeast-1` | AWS region |
| `app_name` | `apcdeploy-e2e-test` | AppConfig application name |

## Customization

To use a different region or application name:

```bash
terraform apply -var="region=us-west-2" -var="app_name=my-e2e-test"
```

Or create a `terraform.tfvars` file:

```hcl
region   = "us-west-2"
app_name = "my-e2e-test"
```

## Notes

- This infrastructure is dedicated to E2E testing only
- Deployment history accumulates during test execution but is managed automatically by AWS
- Deployment history is deleted when resources are destroyed
