# Stack: api — the three HTTP Lambda functions (container images + ECR) and
# the API Gateway HTTP API. Depends on the database stack (remote state).

terraform {
  required_version = ">= 1.6"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # backend "s3" {}   # key e.g. tcg/api/terraform.tfstate
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "tcg"
      Stack       = "api"
      ManagedBy   = "terraform"
    }
  }
}

# Outputs of the database stack. Defaults to its local state file; point
# database_state_path elsewhere (or switch the backend block) for remote state.
data "terraform_remote_state" "database" {
  backend = "local"
  config = {
    path = var.database_state_path
  }
}

locals {
  table_arn  = data.terraform_remote_state.database.outputs.table_arn
  table_name = data.terraform_remote_state.database.outputs.table_name

  lambda_env = {
    TABLE_NAME = local.table_name
    API_TOKEN  = var.api_token
  }
}

module "api_game_routes" {
  source                = "../../modules/lambda-function"
  function_name         = "${var.name_prefix}-api-game-routes"
  image_tag             = var.image_tag
  ecr_force_delete      = var.ecr_force_delete
  table_arn             = local.table_arn
  environment_variables = local.lambda_env
  memory_mb             = var.lambda_memory_mb
  log_retention_days    = var.log_retention_days
}

module "api_set_routes" {
  source                = "../../modules/lambda-function"
  function_name         = "${var.name_prefix}-api-set-routes"
  image_tag             = var.image_tag
  ecr_force_delete      = var.ecr_force_delete
  table_arn             = local.table_arn
  environment_variables = local.lambda_env
  memory_mb             = var.lambda_memory_mb
  log_retention_days    = var.log_retention_days
}

module "api_card_routes" {
  source                = "../../modules/lambda-function"
  function_name         = "${var.name_prefix}-api-card-routes"
  image_tag             = var.image_tag
  ecr_force_delete      = var.ecr_force_delete
  table_arn             = local.table_arn
  environment_variables = local.lambda_env
  memory_mb             = var.lambda_memory_mb
  log_retention_days    = var.log_retention_days
}

# End-user accounts (app login/register + social sign-in) + the admin token
# check. Social providers activate only when their credentials are set.
module "api_auth_routes" {
  source           = "../../modules/lambda-function"
  function_name    = "${var.name_prefix}-api-auth-routes"
  image_tag        = var.image_tag
  ecr_force_delete = var.ecr_force_delete
  table_arn        = local.table_arn
  environment_variables = merge(local.lambda_env, {
    USER_JWT_SECRET        = var.user_jwt_secret
    APP_URL                = var.app_url
    GOOGLE_CLIENT_ID       = var.google_client_id
    GOOGLE_CLIENT_SECRET   = var.google_client_secret
    FACEBOOK_CLIENT_ID     = var.facebook_client_id
    FACEBOOK_CLIENT_SECRET = var.facebook_client_secret
    APPLE_CLIENT_ID        = var.apple_client_id
    APPLE_CLIENT_SECRET    = var.apple_client_secret
  })
  memory_mb          = var.lambda_memory_mb
  log_retention_days = var.log_retention_days
}
