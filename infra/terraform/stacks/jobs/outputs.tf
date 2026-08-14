output "ecr_repositories" {
  description = "ECR repository URLs (push targets for tools/scripts/push-images.sh)"
  value = {
    "pokemon-price-updater" = module.pokemon_price_updater.ecr_repository_url
  }
}
