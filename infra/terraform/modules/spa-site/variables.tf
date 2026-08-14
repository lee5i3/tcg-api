variable "site_name" {
  description = "Site name; drives bucket, OAC, and distribution naming"
  type        = string
}

variable "api_origin_domain" {
  description = "Bare domain of the API Gateway HTTP API origin"
  type        = string
}

variable "spa_rewrite_function_arn" {
  description = "CloudFront Function that rewrites extensionless paths to /index.html"
  type        = string
}
