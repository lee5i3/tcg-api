# API Gateway HTTP API fronting the three Lambdas. Each route key maps to
# exactly one function; the same map drives routes, integrations, and invoke
# permissions. Write-route auth (bearer token) happens inside the Lambdas
# (libs/httpapi). Mirrored by tools/local-gateway/main.go — keep in sync.
locals {
  routes = {
    "GET /healthz"                            = "game-routes"
    "GET /v1/games"                           = "game-routes"
    "POST /v1/games"                          = "game-routes"
    "POST /v1/auth/register"                  = "auth-routes"
    "POST /v1/auth/login"                     = "auth-routes"
    "GET /v1/auth/me"                         = "auth-routes"
    "POST /v1/auth/check"                     = "auth-routes"
    "GET /v1/auth/providers"                  = "auth-routes"
    "GET /v1/auth/oauth/{provider}/start"     = "auth-routes"
    "GET /v1/auth/oauth/{provider}/callback"  = "auth-routes"
    "POST /v1/auth/oauth/{provider}/callback" = "auth-routes"
    "GET /v1/games/{game}/sets"               = "set-routes"
    "POST /v1/games/{game}/sets"              = "set-routes"
    "PUT /v1/games/{game}/sets/{set}"         = "set-routes"
    "DELETE /v1/games/{game}/sets/{set}"      = "set-routes"
    "GET /v1/games/{game}/sets/{set}/cards"   = "card-routes"
    "GET /v1/games/{game}/cards"              = "card-routes"
    "POST /v1/games/{game}/cards"             = "card-routes"
    "GET /v1/games/{game}/cards/{card}"       = "card-routes"
    "PUT /v1/games/{game}/cards/{card}"       = "card-routes"
    "DELETE /v1/games/{game}/cards/{card}"    = "card-routes"
  }

  lambdas = {
    "game-routes" = module.api_game_routes
    "set-routes"  = module.api_set_routes
    "card-routes" = module.api_card_routes
    "auth-routes" = module.api_auth_routes
  }
}

resource "aws_apigatewayv2_api" "catalog" {
  name          = "${var.name_prefix}-api"
  protocol_type = "HTTP"

  cors_configuration {
    allow_origins = var.cors_allowed_origins
    allow_methods = ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allow_headers = ["authorization", "content-type"]
    max_age       = 3600
  }
}

resource "aws_apigatewayv2_integration" "lambda" {
  for_each               = local.lambdas
  api_id                 = aws_apigatewayv2_api.catalog.id
  integration_type       = "AWS_PROXY"
  integration_uri        = each.value.invoke_arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_route" "routes" {
  for_each  = local.routes
  api_id    = aws_apigatewayv2_api.catalog.id
  route_key = each.key
  target    = "integrations/${aws_apigatewayv2_integration.lambda[each.value].id}"
}

resource "aws_cloudwatch_log_group" "api_access" {
  name              = "/aws/apigateway/${var.name_prefix}-api"
  retention_in_days = var.log_retention_days
}

resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.catalog.id
  name        = "$default"
  auto_deploy = true

  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.api_access.arn
    format = jsonencode({
      requestId    = "$context.requestId"
      routeKey     = "$context.routeKey"
      status       = "$context.status"
      latencyMs    = "$context.responseLatency"
      errorMessage = "$context.integrationErrorMessage"
    })
  }

  default_route_settings {
    throttling_burst_limit = 200
    throttling_rate_limit  = 100
  }
}

resource "aws_lambda_permission" "apigw" {
  for_each      = local.lambdas
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = each.value.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.catalog.execution_arn}/*/*"
}
