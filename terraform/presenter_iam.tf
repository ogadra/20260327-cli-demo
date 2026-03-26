# --- Presenter WebSocket Lambda Role ---

# Lambda assume role policy for presenter functions
data "aws_iam_policy_document" "presenter_lambda_assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

# IAM role for WebSocket Lambda functions (connect, disconnect, message)
resource "aws_iam_role" "presenter_ws_lambda" {
  name               = "bunshin-presenter-ws-lambda"
  assume_role_policy = data.aws_iam_policy_document.presenter_lambda_assume_role.json

  tags = merge(local.common_tags, {
    Service = "presenter"
  })
}

# DynamoDB access for WebSocket Lambda: ws_connections, sessions, poll_votes
data "aws_iam_policy_document" "presenter_ws_dynamodb" {
  statement {
    effect = "Allow"
    actions = [
      "dynamodb:PutItem",
      "dynamodb:GetItem",
      "dynamodb:DeleteItem",
      "dynamodb:Query",
      "dynamodb:UpdateItem",
    ]
    resources = [
      aws_dynamodb_table.presenter_ws_connections.arn,
      aws_dynamodb_table.presenter_sessions.arn,
      aws_dynamodb_table.presenter_poll_votes.arn,
      aws_dynamodb_table.presenter_room_state.arn,
    ]
  }
}

resource "aws_iam_role_policy" "presenter_ws_dynamodb" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  name   = "bunshin-presenter-ws-dynamodb"
  role   = aws_iam_role.presenter_ws_lambda.id
  policy = data.aws_iam_policy_document.presenter_ws_dynamodb.json
}

# API Gateway Management API access for broadcasting messages
data "aws_iam_policy_document" "presenter_ws_apigw" {
  statement {
    effect    = "Allow"
    actions   = ["execute-api:ManageConnections"]
    resources = ["${aws_apigatewayv2_api.presenter_websocket.execution_arn}/*"]
  }
}

resource "aws_iam_role_policy" "presenter_ws_apigw" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  name   = "bunshin-presenter-ws-apigw"
  role   = aws_iam_role.presenter_ws_lambda.id
  policy = data.aws_iam_policy_document.presenter_ws_apigw.json
}

# CloudWatch Logs for WebSocket Lambda
resource "aws_iam_role_policy_attachment" "presenter_ws_logs" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  role       = aws_iam_role.presenter_ws_lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

# --- Presenter Auth Lambda Role ---

# IAM role for login Lambda function
resource "aws_iam_role" "presenter_auth_lambda" {
  name               = "bunshin-presenter-auth-lambda"
  assume_role_policy = data.aws_iam_policy_document.presenter_lambda_assume_role.json

  tags = merge(local.common_tags, {
    Service = "presenter"
  })
}

# DynamoDB access for auth Lambda: sessions table only
data "aws_iam_policy_document" "presenter_auth_dynamodb" {
  statement {
    effect = "Allow"
    actions = [
      "dynamodb:PutItem",
      "dynamodb:GetItem",
    ]
    resources = [aws_dynamodb_table.presenter_sessions.arn]
  }
}

resource "aws_iam_role_policy" "presenter_auth_dynamodb" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  name   = "bunshin-presenter-auth-dynamodb"
  role   = aws_iam_role.presenter_auth_lambda.id
  policy = data.aws_iam_policy_document.presenter_auth_dynamodb.json
}

# Secrets Manager access for password hash retrieval
data "aws_iam_policy_document" "presenter_auth_secrets" {
  statement {
    effect    = "Allow"
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [aws_secretsmanager_secret.presenter_password_hash.arn]
  }
}

resource "aws_iam_role_policy" "presenter_auth_secrets" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  name   = "bunshin-presenter-auth-secrets"
  role   = aws_iam_role.presenter_auth_lambda.id
  policy = data.aws_iam_policy_document.presenter_auth_secrets.json
}

# CloudWatch Logs for auth Lambda
resource "aws_iam_role_policy_attachment" "presenter_auth_logs" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  role       = aws_iam_role.presenter_auth_lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}
