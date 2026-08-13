.DEFAULT_GOAL := help

.PHONY: help dev up down logs build test vet fmt check

help: ## List targets
	@grep -E '^[a-z]+:.*##' $(MAKEFILE_LIST) | awk -F ':.*## ' '{printf "  %-8s %s\n", $$1, $$2}'

dev: ## Run the stack in watch mode (rebuilds the API on code changes)
	docker compose up --build --watch

up: ## Run the stack detached
	docker compose up -d --build

down: ## Stop the stack
	docker compose down

logs: ## Tail the API logs
	docker compose logs -f app

build: ## Compile all packages
	go build ./...

test: ## Run tests
	go test ./...

vet: ## Run go vet
	go vet ./...

fmt: ## Format Go sources
	gofmt -w cmd internal

proto: ## Regenerate gRPC code from proto/ (needs buf, protoc-gen-go, protoc-gen-go-grpc in PATH)
	buf generate
	buf lint

check: build vet test ## Build + vet + test
	@gofmt -l cmd internal | tee /dev/stderr | test -z "$$(cat)" || (echo "gofmt needed"; exit 1)
