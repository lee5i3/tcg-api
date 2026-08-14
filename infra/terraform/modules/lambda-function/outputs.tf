output "function_name" {
  value = aws_lambda_function.this.function_name
}

output "invoke_arn" {
  value = aws_lambda_function.this.invoke_arn
}

output "arn" {
  value = aws_lambda_function.this.arn
}

output "ecr_repository_url" {
  value = aws_ecr_repository.this.repository_url
}
