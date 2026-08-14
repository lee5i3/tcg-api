variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "name_prefix" {
  description = "Prefix for resource names"
  type        = string
  default     = "tcg"
}

variable "database_state_path" {
  description = "Path to the database stack's state file (local backend)"
  type        = string
  default     = "../database/terraform.tfstate"
}

variable "image_tag" {
  description = "Container image tag to deploy"
  type        = string
  default     = "latest"
}

variable "ecr_force_delete" {
  description = "Allow deleting ECR repositories that still hold images (this is production — leave off)"
  type        = bool
  default     = false
}

variable "price_api_url" {
  description = "Price feed: GET ?ids=1,2 → {\"prices\":{\"<id>\":{\"<variant>\":price}}}"
  type        = string
  default     = ""
}

variable "price_update_schedule" {
  description = "EventBridge schedule for pokemon-price-updater"
  type        = string
  default     = "rate(24 hours)"
}

variable "price_updater_enabled" {
  description = "Whether the schedule fires (needs price_api_url)"
  type        = bool
  default     = false
}

variable "log_retention_days" {
  description = "CloudWatch log retention"
  type        = number
  default     = 14
}
