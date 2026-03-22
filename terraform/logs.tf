# trivy:ignore:AVD-AWS-0017 -- KMS encryption is not required for this use case
resource "aws_cloudwatch_log_group" "nginx" {
  # checkov:skip=CKV_AWS_158:KMS encryption is not required for this use case
  # checkov:skip=CKV_AWS_338:30 days retention is sufficient for ephemeral runner logs
  name              = "/ecs/bunshin-nginx"
  retention_in_days = 30

  tags = merge(local.common_tags, {
    Service = "nginx"
  })
}

# trivy:ignore:AVD-AWS-0017 -- KMS encryption is not required for this use case
resource "aws_cloudwatch_log_group" "broker" {
  # checkov:skip=CKV_AWS_158:KMS encryption is not required for this use case
  # checkov:skip=CKV_AWS_338:30 days retention is sufficient for ephemeral runner logs
  name              = "/ecs/bunshin-broker"
  retention_in_days = 30

  tags = merge(local.common_tags, {
    Service = "broker"
  })
}

# trivy:ignore:AVD-AWS-0017 -- KMS encryption is not required for this use case
resource "aws_cloudwatch_log_group" "runner" {
  # checkov:skip=CKV_AWS_158:KMS encryption is not required for this use case
  # checkov:skip=CKV_AWS_338:30 days retention is sufficient for ephemeral runner logs
  name              = "/ecs/bunshin-runner"
  retention_in_days = 30

  tags = merge(local.common_tags, {
    Service = "runner"
  })
}
