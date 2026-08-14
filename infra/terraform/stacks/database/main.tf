# Stack: database — the single-table catalog. Apply this stack first; the
# api and jobs stacks read its outputs via terraform_remote_state.
# Item layout and GSI usage: libs/card-catalog-store/dynamo.go and
# docs/data-model.md.

terraform {
  required_version = ">= 1.6"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # Remote state: fill in and uncomment for team/CI use (key must be unique
  # per stack, e.g. tcg/database/terraform.tfstate).
  # backend "s3" {}
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "tcg"
      Stack       = "database"
      ManagedBy   = "terraform"
    }
  }
}

resource "aws_dynamodb_table" "catalog" {
  name                        = "${var.name_prefix}-catalog"
  billing_mode                = "PAY_PER_REQUEST"
  hash_key                    = "PK"
  range_key                   = "SK"
  deletion_protection_enabled = var.enable_deletion_protection

  attribute {
    name = "PK"
    type = "S"
  }
  attribute {
    name = "SK"
    type = "S"
  }
  attribute {
    name = "GSI1PK"
    type = "S"
  }
  attribute {
    name = "GSI1SK"
    type = "S"
  }
  attribute {
    name = "GSI2PK"
    type = "S"
  }
  attribute {
    name = "GSI3PK"
    type = "S"
  }

  # Games listing + per-game card name search.
  global_secondary_index {
    name            = "GSI1"
    hash_key        = "GSI1PK"
    range_key       = "GSI1SK"
    projection_type = "ALL"
  }

  # Card lookup by GUID.
  global_secondary_index {
    name            = "GSI2"
    hash_key        = "GSI2PK"
    projection_type = "ALL"
  }

  # Card lookup by TCGplayer product id (sparse).
  global_secondary_index {
    name            = "GSI3"
    hash_key        = "GSI3PK"
    projection_type = "ALL"
  }

  point_in_time_recovery {
    enabled = true
  }

  server_side_encryption {
    enabled = true
  }
}
