# --- Presenter WebSocket API ---

# WebSocket API for slide sync, poll, and viewer count
resource "aws_apigatewayv2_api" "presenter_websocket" {
  name                       = "bunshin-presenter-ws"
  protocol_type              = "WEBSOCKET"
  route_selection_expression = "$request.body.action"

  tags = merge(local.common_tags, {
    Service = "presenter"
  })
}

# WebSocket API stage with auto deploy
# trivy:ignore:AVD-AWS-0001 -- Access logging is not needed for presenter WebSocket API
resource "aws_apigatewayv2_stage" "presenter_websocket" {
  # checkov:skip=CKV2_AWS_51:Client certificate authentication is not needed for WebSocket API behind CloudFront
  # checkov:skip=CKV_AWS_76:Access logging is not needed for presenter WebSocket API
  api_id      = aws_apigatewayv2_api.presenter_websocket.id
  name        = "ws"
  auto_deploy = true

  tags = merge(local.common_tags, {
    Service = "presenter"
  })
}

# WebSocket Lambda integrations for each handler
resource "aws_apigatewayv2_integration" "presenter_ws" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  for_each = local.presenter_ws_handlers

  api_id                    = aws_apigatewayv2_api.presenter_websocket.id
  integration_type          = "AWS_PROXY"
  integration_method        = "POST"
  integration_uri           = aws_lambda_function.presenter_ws[each.key].invoke_arn
  content_handling_strategy = "CONVERT_TO_TEXT"
}

# WebSocket routes for $connect, $disconnect, $default
resource "aws_apigatewayv2_route" "presenter_ws" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  # checkov:skip=CKV_AWS_309:Public WebSocket endpoint for browser slide sync
  for_each = local.presenter_ws_handlers

  api_id    = aws_apigatewayv2_api.presenter_websocket.id
  route_key = each.value.route_key
  target    = "integrations/${aws_apigatewayv2_integration.presenter_ws[each.key].id}"
}

# --- Presenter Login HTTP API ---

# HTTP API for presenter authentication
resource "aws_apigatewayv2_api" "presenter_login" {
  name          = "bunshin-presenter-login"
  protocol_type = "HTTP"

  tags = merge(local.common_tags, {
    Service = "presenter"
  })
}

# Login API stage with auto deploy
# trivy:ignore:AVD-AWS-0001 -- Access logging is not needed for presenter login API
resource "aws_apigatewayv2_stage" "presenter_login" {
  # checkov:skip=CKV_AWS_76:Access logging is not needed for presenter login API
  api_id      = aws_apigatewayv2_api.presenter_login.id
  name        = "$default"
  auto_deploy = true

  tags = merge(local.common_tags, {
    Service = "presenter"
  })
}

# Login Lambda integration
resource "aws_apigatewayv2_integration" "presenter_login" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  api_id                 = aws_apigatewayv2_api.presenter_login.id
  integration_type       = "AWS_PROXY"
  integration_method     = "POST"
  integration_uri        = aws_lambda_function.presenter_login.invoke_arn
  payload_format_version = "2.0"
}

# GET /login route
resource "aws_apigatewayv2_route" "presenter_login_get" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  # checkov:skip=CKV_AWS_309:Public login endpoint for presenter authentication
  api_id    = aws_apigatewayv2_api.presenter_login.id
  route_key = "GET /login"
  target    = "integrations/${aws_apigatewayv2_integration.presenter_login.id}"
}

# POST /login route
resource "aws_apigatewayv2_route" "presenter_login_post" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  # checkov:skip=CKV_AWS_309:Public login endpoint for presenter authentication
  api_id    = aws_apigatewayv2_api.presenter_login.id
  route_key = "POST /login"
  target    = "integrations/${aws_apigatewayv2_integration.presenter_login.id}"
}
