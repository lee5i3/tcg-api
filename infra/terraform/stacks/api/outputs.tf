output "api_endpoint" {
  description = "Direct base URL of the HTTP API (services; browsers go through the site distributions)"
  value       = aws_apigatewayv2_api.catalog.api_endpoint
}

output "ecr_repositories" {
  description = "ECR repository URLs (push targets for tools/scripts/push-images.sh)"
  value = {
    "api-game-routes" = module.api_game_routes.ecr_repository_url
    "api-set-routes"  = module.api_set_routes.ecr_repository_url
    "api-card-routes" = module.api_card_routes.ecr_repository_url
    "api-auth-routes" = module.api_auth_routes.ecr_repository_url
  }
}
