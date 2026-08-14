# Stack: sites — the three static SvelteKit sites, each on its own private
# S3 bucket + CloudFront distribution:
#   www — the product's front door (pricing, screenshots, CTAs)
#   app       — the public catalog browser
#   admin     — catalog management (token login), separate from the app
# app and admin proxy /v1/* + /healthz to the API same-origin (remote state
# from the api stack). Deploy build output:
#   npx nx build app   && aws s3 sync apps/web/app/dist   s3://<app_bucket> --delete
#   npx nx build admin && aws s3 sync apps/web/admin/dist s3://<admin_bucket> --delete
#   npx nx build www && aws s3 sync apps/web/www/dist s3://<www_bucket> --delete
# then invalidate each distribution ('/*').

terraform {
  required_version = ">= 1.6"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # backend "s3" {}   # key e.g. tcg/sites/terraform.tfstate
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "tcg"
      Stack       = "sites"
      ManagedBy   = "terraform"
    }
  }
}

data "terraform_remote_state" "api" {
  backend = "local"
  config = {
    path = var.api_state_path
  }
}

locals {
  # CloudFront origins want a bare domain, not a URL.
  api_origin_domain = replace(data.terraform_remote_state.api.outputs.api_endpoint, "https://", "")
}

# SPA routing, shared by all sites: extensionless paths serve /index.html.
resource "aws_cloudfront_function" "spa_rewrite" {
  name    = "${var.name_prefix}-spa-rewrite"
  runtime = "cloudfront-js-2.0"
  publish = true
  code    = <<-EOT
    function handler(event) {
      var request = event.request;
      if (!request.uri.includes('.')) {
        request.uri = '/index.html';
      }
      return request;
    }
  EOT
}

module "www_site" {
  source                   = "../../modules/spa-site"
  site_name                = "${var.name_prefix}-www"
  api_origin_domain        = local.api_origin_domain
  spa_rewrite_function_arn = aws_cloudfront_function.spa_rewrite.arn
}

module "app_site" {
  source                   = "../../modules/spa-site"
  site_name                = "${var.name_prefix}-app"
  api_origin_domain        = local.api_origin_domain
  spa_rewrite_function_arn = aws_cloudfront_function.spa_rewrite.arn
}

module "admin_site" {
  source                   = "../../modules/spa-site"
  site_name                = "${var.name_prefix}-admin"
  api_origin_domain        = local.api_origin_domain
  spa_rewrite_function_arn = aws_cloudfront_function.spa_rewrite.arn
}
