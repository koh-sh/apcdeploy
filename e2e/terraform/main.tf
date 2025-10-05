variable "region" {
  description = "AWS region for E2E test resources"
  type        = string
  default     = "ap-northeast-1"
}

variable "app_name" {
  description = "AppConfig application name for E2E tests"
  type        = string
  default     = "apcdeploy-e2e-test"
}

# AppConfig Application
resource "aws_appconfig_application" "e2e_test" {
  name        = var.app_name
  description = "E2E test application for apcdeploy"

  tags = {
    Environment = "test"
    ManagedBy   = "terraform"
    Purpose     = "e2e-testing"
  }
}

# Environments
resource "aws_appconfig_environment" "dev" {
  application_id = aws_appconfig_application.e2e_test.id
  name           = "dev"
  description    = "Development environment for E2E tests"

  tags = {
    Environment = "dev"
  }
}

resource "aws_appconfig_environment" "staging" {
  application_id = aws_appconfig_application.e2e_test.id
  name           = "staging"
  description    = "Staging environment for E2E tests"

  tags = {
    Environment = "staging"
  }
}

# Configuration Profiles

# 1. Freeform JSON profile
resource "aws_appconfig_configuration_profile" "json_freeform" {
  application_id = aws_appconfig_application.e2e_test.id
  name           = "json-freeform"
  description    = "Freeform JSON configuration for testing"
  location_uri   = "hosted"
  type           = "AWS.Freeform"

  tags = {
    ContentType = "application/json"
  }
}

# 2. FeatureFlags profile
resource "aws_appconfig_configuration_profile" "json_featureflags" {
  application_id = aws_appconfig_application.e2e_test.id
  name           = "json-featureflags"
  description    = "FeatureFlags configuration for testing"
  location_uri   = "hosted"
  type           = "AWS.AppConfig.FeatureFlags"

  tags = {
    ContentType = "application/json"
  }
}

# 3. YAML profile
resource "aws_appconfig_configuration_profile" "yaml_config" {
  application_id = aws_appconfig_application.e2e_test.id
  name           = "yaml-config"
  description    = "YAML configuration for testing"
  location_uri   = "hosted"
  type           = "AWS.Freeform"

  tags = {
    ContentType = "application/x-yaml"
  }
}

# 4. Plain text profile
resource "aws_appconfig_configuration_profile" "text_config" {
  application_id = aws_appconfig_application.e2e_test.id
  name           = "text-config"
  description    = "Plain text configuration for testing"
  location_uri   = "hosted"
  type           = "AWS.Freeform"

  tags = {
    ContentType = "text/plain"
  }
}

# 5. Profile for error testing (no validators, used for various error scenarios)
resource "aws_appconfig_configuration_profile" "error_test" {
  application_id = aws_appconfig_application.e2e_test.id
  name           = "error-test"
  description    = "Profile for error scenario testing"
  location_uri   = "hosted"
  type           = "AWS.Freeform"

  tags = {
    ContentType = "application/json"
  }
}

# Deployment Strategies
# Note: AWS provides predefined deployment strategies:
# - AppConfig.AllAtOnce (0% growth, 0 min bake time)
# - AppConfig.Linear50PercentEvery30Seconds (50% growth, 1 min bake time)
# - AppConfig.Canary10Percent20Minutes (10% then 90%, 10 min bake time)

# Custom deployment strategy for testing (instant deployment)
resource "aws_appconfig_deployment_strategy" "e2e_test_strategy" {
  name                           = "E2E-Test-Strategy"
  description                    = "Custom deployment strategy for E2E testing (instant deployment)"
  deployment_duration_in_minutes = 0
  growth_factor                  = 100
  final_bake_time_in_minutes     = 0
  replicate_to                   = "NONE"

  tags = {
    Purpose = "e2e-testing"
  }
}

# Slow deployment strategy for timeout testing
resource "aws_appconfig_deployment_strategy" "e2e_slow_strategy" {
  name                           = "E2E-Slow-Strategy"
  description                    = "Slow deployment strategy for timeout testing (1 min deployment)"
  deployment_duration_in_minutes = 1
  growth_factor                  = 100
  final_bake_time_in_minutes     = 0
  replicate_to                   = "NONE"

  tags = {
    Purpose = "e2e-testing"
  }
}

