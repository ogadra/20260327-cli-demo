# trivy:ignore:AVD-AWS-0017 -- KMS encryption is not required for this use case
resource "aws_cloudwatch_log_group" "ecs" {
  # checkov:skip=CKV_AWS_158:KMS encryption is not required for this use case
  # checkov:skip=CKV_AWS_338:30 days retention is sufficient for ephemeral runner logs
  for_each = local.ecs_services

  name              = "/ecs/bunshin-${each.key}"
  retention_in_days = 30

  tags = merge(local.common_tags, {
    Service = each.key
  })
}

# trivy:ignore:AVD-AWS-0017 -- KMS encryption is not required for this use case
resource "aws_cloudwatch_log_group" "presenter_ws" {
  # checkov:skip=CKV_AWS_158:KMS encryption is not required for this use case
  # checkov:skip=CKV_AWS_338:30 days retention is sufficient for presenter Lambda logs
  for_each = local.presenter_ws_handlers

  name              = "/aws/lambda/bunshin-presenter-ws-${each.key}"
  retention_in_days = 30

  tags = merge(local.common_tags, {
    Service = "presenter"
  })
}

# trivy:ignore:AVD-AWS-0017 -- KMS encryption is not required for this use case
resource "aws_cloudwatch_log_group" "presenter_login" {
  # checkov:skip=CKV_AWS_158:KMS encryption is not required for this use case
  # checkov:skip=CKV_AWS_338:30 days retention is sufficient for presenter Lambda logs
  name              = "/aws/lambda/bunshin-presenter-login"
  retention_in_days = 30

  tags = merge(local.common_tags, {
    Service = "presenter"
  })
}
