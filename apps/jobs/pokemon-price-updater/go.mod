module github.com/lee5i3/tcg-api/apps/jobs/pokemon-price-updater

go 1.26

require (
	github.com/aws/aws-lambda-go v1.47.0
	github.com/lee5i3/tcg-api/libs/card-catalog-store v0.0.0
	github.com/lee5i3/tcg-api/libs/httpapi v0.0.0
)

require (
	github.com/aws/aws-sdk-go-v2 v1.36.0 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.29.0 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.53 // indirect
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.18.0 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.24 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.31 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.31 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.39.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodbstreams v1.24.18 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.10.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.24.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.28.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.8 // indirect
	github.com/aws/smithy-go v1.22.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
)

replace (
	github.com/lee5i3/tcg-api/libs/card-catalog-store => ../../../libs/card-catalog-store
	github.com/lee5i3/tcg-api/libs/httpapi => ../../../libs/httpapi
)
