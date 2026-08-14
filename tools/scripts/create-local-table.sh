#!/usr/bin/env bash
# Create the tcg-catalog table on DynamoDB Local (docker compose up -d).
# Mirrors the schema Terraform provisions in AWS (infra/terraform/dynamodb.tf).
set -euo pipefail

ENDPOINT="${DYNAMODB_ENDPOINT:-http://localhost:8000}"
TABLE="${TABLE_NAME:-tcg-catalog}"

AWS_ACCESS_KEY_ID=local AWS_SECRET_ACCESS_KEY=local aws dynamodb create-table \
  --endpoint-url "$ENDPOINT" \
  --region us-east-1 \
  --table-name "$TABLE" \
  --billing-mode PAY_PER_REQUEST \
  --attribute-definitions \
    AttributeName=PK,AttributeType=S \
    AttributeName=SK,AttributeType=S \
    AttributeName=GSI1PK,AttributeType=S \
    AttributeName=GSI1SK,AttributeType=S \
    AttributeName=GSI2PK,AttributeType=S \
    AttributeName=GSI3PK,AttributeType=S \
  --key-schema AttributeName=PK,KeyType=HASH AttributeName=SK,KeyType=RANGE \
  --global-secondary-indexes \
    'IndexName=GSI1,KeySchema=[{AttributeName=GSI1PK,KeyType=HASH},{AttributeName=GSI1SK,KeyType=RANGE}],Projection={ProjectionType=ALL}' \
    'IndexName=GSI2,KeySchema=[{AttributeName=GSI2PK,KeyType=HASH}],Projection={ProjectionType=ALL}' \
    'IndexName=GSI3,KeySchema=[{AttributeName=GSI3PK,KeyType=HASH}],Projection={ProjectionType=ALL}'

echo "created $TABLE at $ENDPOINT"
