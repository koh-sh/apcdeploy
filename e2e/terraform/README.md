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

### Deployment Strategies
- **E2E-Test-Strategy**: Custom strategy (fast deployment + 1 min bake time)
- **AppConfig.AllAtOnce**: Built-in strategy (immediate deployment)
- **AppConfig.Linear50PercentEvery30Seconds**: Built-in strategy (gradual deployment)

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
