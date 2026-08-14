# Developer entry points. `make` lists targets; `make dev` runs everything.
.DEFAULT_GOAL := help

.PHONY: help dev down logs ps test lint build clean images

help: ## List available targets
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

dev: ## Start the entire dev environment (marketing :8082, app :8080, admin :8081, API :3000, DynamoDB :8000)
	docker compose up --build

down: ## Stop the dev environment and remove its containers
	docker compose down

logs: ## Tail dev-environment logs
	docker compose logs -f

ps: ## Show dev-environment status
	docker compose ps

test: ## Run every project's tests (Go + Vitest via NX)
	npx nx run-many -t test

lint: ## Run every project's linters (go vet/gofmt, svelte-check, terraform fmt)
	npx nx run-many -t lint

build: ## Build every project (Lambda compile checks + static site bundles)
	npx nx run-many -t build

images: ## Build and push all Lambda images to ECR (production)
	tools/scripts/push-images.sh

clean: ## Stop the dev environment and drop volumes + built site output
	docker compose down --volumes --remove-orphans
	rm -rf apps/web/app/dist apps/web/admin/dist apps/web/www/dist
