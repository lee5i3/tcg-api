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

variable "api_state_path" {
  description = "Path to the api stack's state file (local backend)"
  type        = string
  default     = "../api/terraform.tfstate"
}
