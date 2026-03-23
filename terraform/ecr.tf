# trivy:ignore:AVD-AWS-0033 -- AWS managed encryption is sufficient
# trivy:ignore:AVD-AWS-0031 -- mutable tags required for latest-based deployment
resource "aws_ecr_repository" "service" {
  # checkov:skip=CKV_AWS_136:AWS managed encryption is sufficient
  # checkov:skip=CKV_AWS_51:mutable tags required for latest-based deployment
  for_each = local.ecs_services

  name                 = "bunshin/${each.key}"
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = merge(local.common_tags, {
    Service = each.key
  })
}

resource "aws_ecr_lifecycle_policy" "service" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  for_each = local.ecs_services

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
