# --- Presenter Lambda Functions ---
#
# Lambda code is deployed via CI/CD using aws lambda update-function-code
# with --s3-bucket and --s3-key. Terraform manages only the function
# definition. The S3 object is uploaded by CI before the first apply.

# S3 bucket for presenter Lambda deployment artifacts
# trivy:ignore:AVD-AWS-0089 -- S3 bucket logging is optional for deployment artifacts
# trivy:ignore:AVD-AWS-0132 -- S3 bucket encryption uses AWS managed key
# trivy:ignore:AVD-AWS-0090 -- Versioning is not needed for Lambda deployment artifacts
resource "aws_s3_bucket" "presenter_lambda" {
  # checkov:skip=CKV_AWS_18:S3 bucket logging is optional for deployment artifacts
  # checkov:skip=CKV_AWS_145:AWS managed encryption is sufficient
  # checkov:skip=CKV_AWS_144:Cross-region replication is not needed
  # checkov:skip=CKV2_AWS_62:Event notifications are not needed
  # checkov:skip=CKV2_AWS_61:Lifecycle configuration is not needed for deployment artifacts
  # checkov:skip=CKV_AWS_21:Versioning is not needed for deployment artifacts
  bucket = format(
    "bunshin-presenter-lambda-%s-%s-an",
    data.aws_caller_identity.current.account_id,
    data.aws_region.current.id,
  )
  bucket_namespace = "account-regional"

  tags = merge(local.common_tags, {
    Service = "presenter"
  })
}

resource "aws_s3_bucket_public_access_block" "presenter_lambda" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  bucket = aws_s3_bucket.presenter_lambda.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# --- Presenter WebSocket Lambda Functions ---

# trivy:ignore:AVD-AWS-0066 -- X-Ray tracing is not required for presenter Lambda
resource "aws_lambda_function" "presenter_ws" {
  # checkov:skip=CKV_AWS_115:Reserved concurrency is not needed for on-demand presenter Lambda
  # checkov:skip=CKV_AWS_116:DLQ is not needed for synchronous WebSocket handlers
  # checkov:skip=CKV_AWS_117:VPC is not needed for presenter Lambda accessing only DynamoDB and API Gateway
  # checkov:skip=CKV_AWS_173:Environment variables do not contain sensitive data
  # checkov:skip=CKV_AWS_50:X-Ray tracing is not required for presenter Lambda
  # checkov:skip=CKV_AWS_272:Code signing is not required for presenter Lambda
  for_each = local.presenter_ws_handlers

  s3_bucket     = aws_s3_bucket.presenter_lambda.id
  s3_key        = "presenter-ws-${each.key}.zip"
  function_name = "bunshin-presenter-ws-${each.key}"
  role          = aws_iam_role.presenter_ws_lambda.arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]
  timeout       = 10

  environment {
    variables = {
      WS_CONNECTIONS_TABLE   = aws_dynamodb_table.presenter_ws_connections.name
      SESSIONS_TABLE         = aws_dynamodb_table.presenter_sessions.name
      POLL_VOTES_TABLE       = aws_dynamodb_table.presenter_poll_votes.name
      WEBSOCKET_API_ENDPOINT = "${aws_apigatewayv2_api.presenter_websocket.id}.execute-api.ap-northeast-1.amazonaws.com/${aws_apigatewayv2_stage.presenter_websocket.name}"
    }
  }

  depends_on = [aws_cloudwatch_log_group.presenter_ws]

  tags = merge(local.common_tags, {
    Service = "presenter"
  })

  lifecycle {
    ignore_changes = [s3_key, s3_object_version, source_code_hash]
  }
}

# API Gateway invoke permissions for WebSocket handlers
resource "aws_lambda_permission" "presenter_ws" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  for_each = local.presenter_ws_handlers

  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.presenter_ws[each.key].function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.presenter_websocket.execution_arn}/*/${each.value.route_key}"
}

# --- Presenter Login Lambda Function ---

# trivy:ignore:AVD-AWS-0066 -- X-Ray tracing is not required for presenter Lambda
resource "aws_lambda_function" "presenter_login" {
  # checkov:skip=CKV_AWS_115:Reserved concurrency is not needed for on-demand presenter Lambda
  # checkov:skip=CKV_AWS_116:DLQ is not needed for synchronous HTTP handler
  # checkov:skip=CKV_AWS_117:VPC is not needed for presenter Lambda accessing only DynamoDB and Secrets Manager
  # checkov:skip=CKV_AWS_173:Environment variables do not contain sensitive data
  # checkov:skip=CKV_AWS_50:X-Ray tracing is not required for presenter Lambda
  # checkov:skip=CKV_AWS_272:Code signing is not required for presenter Lambda
  s3_bucket     = aws_s3_bucket.presenter_lambda.id
  s3_key        = "presenter-login.zip"
  function_name = "bunshin-presenter-login"
  role          = aws_iam_role.presenter_auth_lambda.arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]
  timeout       = 10

  environment {
    variables = {
      SESSIONS_TABLE = aws_dynamodb_table.presenter_sessions.name
      SECRET_ARN     = aws_secretsmanager_secret.presenter_password_hash.arn
    }
  }

  depends_on = [aws_cloudwatch_log_group.presenter_login]

  tags = merge(local.common_tags, {
    Service = "presenter"
  })

  lifecycle {
    ignore_changes = [s3_key, s3_object_version, source_code_hash]
  }
}

# API Gateway invoke permission for login handler
resource "aws_lambda_permission" "presenter_login" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.presenter_login.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.presenter_login.execution_arn}/*/*/login"
}
