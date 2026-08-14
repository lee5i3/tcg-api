output "table_name" {
  description = "DynamoDB catalog table name"
  value       = aws_dynamodb_table.catalog.name
}

output "table_arn" {
  description = "DynamoDB catalog table ARN"
  value       = aws_dynamodb_table.catalog.arn
}
