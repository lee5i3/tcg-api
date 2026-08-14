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

variable "enable_deletion_protection" {
  description = "Protect the table from accidental deletion (this is production — leave on)"
  type        = bool
  default     = true
}
