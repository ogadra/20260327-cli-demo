# trivy:ignore:AVD-AWS-0017 -- KMS encryption is not required for this use case
resource "aws_cloudwatch_log_group" "ecs" {
  # checkov:skip=CKV_AWS_158:KMS encryption is not required for this use case
  for_each = local.ecs_services

  name                        = "/ecs/bunshin-${each.key}"
  retention_in_days           = 365
  deletion_protection_enabled = true

  tags = merge(local.common_tags, {
    Service = each.key
  })
}
