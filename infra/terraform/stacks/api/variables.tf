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

variable "api_token" {
  description = "Bearer token required on write routes. Empty disables write auth — never leave empty outside local experiments."
  type        = string
  default     = ""
  sensitive   = true
}

variable "user_jwt_secret" {
  description = "HS256 secret for app user session tokens (auth-routes). Generate with: openssl rand -hex 32"
  type        = string
  sensitive   = true
}

variable "app_url" {
  description = "Public base URL of the app site (e.g. the CloudFront app_url output from the sites stack). Required for social login: it is the OAuth redirect_uri host and the post-login destination. Set it after the first sites apply, then re-apply this stack."
  type        = string
  default     = ""
}

variable "google_client_id" {
  description = "Google OAuth client id (empty disables Google sign-in)"
  type        = string
  default     = ""
}

variable "google_client_secret" {
  description = "Google OAuth client secret"
  type        = string
  default     = ""
  sensitive   = true
}

variable "facebook_client_id" {
  description = "Facebook app id (empty disables Facebook sign-in)"
  type        = string
  default     = ""
}

variable "facebook_client_secret" {
  description = "Facebook app secret"
  type        = string
  default     = ""
  sensitive   = true
}

variable "apple_client_id" {
  description = "Apple Services ID (empty disables Apple sign-in)"
  type        = string
  default     = ""
}

variable "apple_client_secret" {
  description = "Apple client-secret JWT (ES256-signed; expires — rotate within 6 months)"
  type        = string
  default     = ""
  sensitive   = true
}

variable "image_tag" {
  description = "Container image tag to deploy (pushed by tools/scripts/push-images.sh)"
  type        = string
  default     = "latest"
}

variable "ecr_force_delete" {
  description = "Allow deleting ECR repositories that still hold images (this is production — leave off)"
  type        = bool
  default     = false
}

variable "cors_allowed_origins" {
  description = "Origins allowed to call the API from a browser (direct API URL only; the sites proxy same-origin)"
  type        = list(string)
  default     = ["*"]
}

variable "lambda_memory_mb" {
  description = "Memory (and proportional CPU) per function"
  type        = number
  default     = 256
}

variable "log_retention_days" {
  description = "CloudWatch log retention"
  type        = number
  default     = 14
}
