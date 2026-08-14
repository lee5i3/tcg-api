#!/usr/bin/env bash
# Build and push every Lambda container image to its ECR repository.
# There is one environment — production.
#
#   tools/scripts/push-images.sh [tag]
#
# tag defaults to "latest" (must match the image_tag Terraform deploys).
# Repositories are created by the api/jobs Terraform stacks — on a fresh
# account apply their ECR repos first (or -target them), push, then apply
# the rest.
set -euo pipefail
cd "$(dirname "$0")/../.."

TAG="${1:-latest}"
PREFIX="${NAME_PREFIX:-tcg}"
REGION="${AWS_REGION:-$(aws configure get region)}"
ACCOUNT="$(aws sts get-caller-identity --query Account --output text)"
REGISTRY="$ACCOUNT.dkr.ecr.$REGION.amazonaws.com"

# function name → Dockerfile directory
declare -A FUNCTIONS=(
  ["api-game-routes"]="apps/api/game-routes"
  ["api-set-routes"]="apps/api/set-routes"
  ["api-card-routes"]="apps/api/card-routes"
  ["api-auth-routes"]="apps/api/auth-routes"
  ["pokemon-price-updater"]="apps/jobs/pokemon-price-updater"
)

aws ecr get-login-password --region "$REGION" |
  docker login --username AWS --password-stdin "$REGISTRY"

for fn in "${!FUNCTIONS[@]}"; do
  repo="$REGISTRY/$PREFIX-$fn"
  echo "==> $fn → $repo:$TAG"
  docker build --platform linux/arm64 \
    -f "${FUNCTIONS[$fn]}/Dockerfile" \
    -t "$repo:$TAG" .
  docker push "$repo:$TAG"
done

echo "done — deploy with: terraform apply (image_tag=$TAG)"
