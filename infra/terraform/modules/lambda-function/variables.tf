variable "function_name" {
  description = "Full function name; also the ECR repository name"
  type        = string
}

variable "image_tag" {
  description = "Image tag to deploy from the function's ECR repository"
  type        = string
  default     = "latest"
}

variable "ecr_force_delete" {
  description = "Allow deleting the ECR repository even when it holds images (dev)"
  type        = bool
  default     = false
}

variable "table_arn" {
  description = "DynamoDB table ARN the function may access (indexes included)"
  type        = string
}

variable "environment_variables" {
  description = "Environment variables for the function"
  type        = map(string)
  default     = {}
}

variable "memory_mb" {
  description = "Memory (and proportional CPU)"
  type        = number
  default     = 256
}

variable "timeout_seconds" {
  description = "Invocation timeout"
  type        = number
  default     = 10
}

variable "log_retention_days" {
  description = "CloudWatch log retention"
  type        = number
  default     = 14
}
