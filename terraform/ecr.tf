# trivy:ignore:AVD-AWS-0033 -- AWS managed encryption is sufficient
resource "aws_ecr_repository" "service" {
  # checkov:skip=CKV_AWS_136:AWS managed encryption is sufficient
  for_each = local.services

  name                 = "bunshin/${each.key}"
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = merge(local.common_tags, {
    Service = each.key
  })
}

resource "aws_ecr_lifecycle_policy" "service" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  for_each = local.services

  repository = aws_ecr_repository.service[each.key].name

  policy = jsonencode({
    rules = [{
      rulePriority = 1
      description  = "Keep last 3 images"
      selection = {
        tagStatus   = "any"
        countType   = "imageCountMoreThan"
        countNumber = 3
      }
      action = {
        type = "expire"
      }
    }]
  })
}
