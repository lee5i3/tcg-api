# Infrastructure (Terraform)

Split into four **independently-appliable stacks** under `stacks/`, each with
its own state; shared building blocks live in `modules/`. **There is one
environment — production.** Resource names carry no environment suffix
(`tcg-catalog`, `tcg-api-card-routes`, …), the DynamoDB table has deletion
protection on by default, and ECR repositories refuse force-deletion.

| Stack | Provisions | Depends on |
| --- | --- | --- |
| `database` | The single-table DynamoDB catalog (GSI1–3, PITR, SSE) | — |
| `api` | `api-game-routes` / `api-set-routes` / `api-card-routes` (ECR + container-image Lambdas) and the API Gateway HTTP API | database |
| `jobs` | `pokemon-price-updater` (ECR + Lambda) + its EventBridge schedule | database |
| `sites` | `www` (marketing), `app`, `admin` — three static SvelteKit sites, each a private S3 bucket + CloudFront distribution (app/admin proxy `/v1/*` same-origin) | api |

Cross-stack wiring uses `terraform_remote_state`; with the default local
backend the paths (`../database/terraform.tfstate`, `../api/terraform.tfstate`)
just work. **For CI (`.github/workflows/infra.yaml`) or any team use,
configure the S3 backend in each stack's `backend "s3" {}` block first** —
local state does not survive a CI runner — and switch the remote-state data
sources accordingly.

## Order of operations (fresh AWS account)

```sh
cd infra/terraform/stacks/database
cp terraform.tfvars.example terraform.tfvars
terraform init && terraform apply

cd ../api
cp terraform.tfvars.example terraform.tfvars   # set api_token!
terraform init
# ECR repos must exist before the images, images before the Lambdas:
terraform apply \
  -target=module.api_game_routes.aws_ecr_repository.this \
  -target=module.api_set_routes.aws_ecr_repository.this \
  -target=module.api_card_routes.aws_ecr_repository.this
../../../../tools/scripts/push-images.sh latest
terraform apply

cd ../jobs      # same dance for the updater's repo, or let build.yaml push
terraform init && terraform apply

cd ../sites
cp terraform.tfvars.example terraform.tfvars
terraform init && terraform apply

# Publish the sites (repeat per site: marketing, app, admin)
npx nx build app
aws s3 sync ../../../../apps/web/app/dist "s3://$(terraform output -raw app_bucket)" --delete
aws cloudfront create-invalidation --distribution-id "$(terraform output -raw app_distribution_id)" --paths '/*'

# Marketing is prerendered — bake the real app URL into its CTAs at build time:
PUBLIC_APP_URL="$(terraform output -raw app_url)" npx nx build www
aws s3 sync ../../../../apps/web/www/dist "s3://$(terraform output -raw www_bucket)" --delete
aws cloudfront create-invalidation --distribution-id "$(terraform output -raw www_distribution_id)" --paths '/*'
```

Terraform auto-loads `terraform.tfvars` — no `-var-file` needed.

## Day-to-day

- **Lambda code**: push a new tag (`tools/scripts/push-images.sh v42` or the
  `build.yaml` workflow, which pushes on every merge to main) and
  `terraform apply -var image_tag=v42` in `stacks/api` (and/or `stacks/jobs`).
  Only the touched stack rolls.
- **Sites**: rebuild, `aws s3 sync`, invalidate — no Terraform involved.
- **CI applies**: the `infra.yaml` workflow (workflow_dispatch: pick stack +
  plan/apply); needs the S3 backend plus AWS OIDC credentials
  (`AWS_ROLE_ARN` secret, `TF_API_TOKEN_VALUE` secret, `AWS_REGION` var).
- The route table in `stacks/api/apigateway.tf` is mirrored by
  `tools/local-gateway/main.go` — keep them in sync.

## Auth & price feed

- `api_token` (api stack) → Lambdas' `API_TOKEN`; admin logs in with it via
  `POST /v1/auth/check`.
- The price updater (jobs stack) stays `DISABLED` until `price_api_url` is
  set and `price_updater_enabled = true`.
