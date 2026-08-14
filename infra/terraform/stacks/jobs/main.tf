# Stack: jobs — scheduled Lambda jobs (currently pokemon-price-updater).
# Depends on the database stack (remote state); independent of the api stack.

terraform {
  required_version = ">= 1.6"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # backend "s3" {}   # key e.g. tcg/jobs/terraform.tfstate
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "tcg"
      Stack       = "jobs"
      ManagedBy   = "terraform"
    }
  }
}

data "terraform_remote_state" "database" {
  backend = "local"
  config = {
    path = var.database_state_path
  }
}

module "pokemon_price_updater" {
  source           = "../../modules/lambda-function"
  function_name    = "${var.name_prefix}-pokemon-price-updater"
  image_tag        = var.image_tag
  ecr_force_delete = var.ecr_force_delete
  table_arn        = data.terraform_remote_state.database.outputs.table_arn
  environment_variables = {
    TABLE_NAME    = data.terraform_remote_state.database.outputs.table_name
    PRICE_API_URL = var.price_api_url
    GAME_KEY      = "pokemon"
  }
  memory_mb          = 256
  timeout_seconds    = 300 # walks the whole Pokémon catalog
  log_retention_days = var.log_retention_days
}

resource "aws_cloudwatch_event_rule" "price_update" {
  name                = "${var.name_prefix}-pokemon-price-update"
  description         = "Refresh Pokémon card prices"
  schedule_expression = var.price_update_schedule
  state               = var.price_updater_enabled ? "ENABLED" : "DISABLED"
}

resource "aws_cloudwatch_event_target" "price_update" {
  rule = aws_cloudwatch_event_rule.price_update.name
  arn  = module.pokemon_price_updater.arn
}

resource "aws_lambda_permission" "price_update_events" {
  statement_id  = "AllowEventBridgeInvoke"
  action        = "lambda:InvokeFunction"
  function_name = module.pokemon_price_updater.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.price_update.arn
}
