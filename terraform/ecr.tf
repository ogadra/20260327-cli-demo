# trivy:ignore:AVD-AWS-0033 -- AWS managed encryption is sufficient
resource "aws_ecr_repository" "nginx" {
  # checkov:skip=CKV_AWS_136:AWS managed encryption is sufficient
  name                 = "bunshin/nginx"
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = merge(local.common_tags, {
    Service = "nginx"
  })
}

# trivy:ignore:AVD-AWS-0033 -- AWS managed encryption is sufficient
resource "aws_ecr_repository" "broker" {
  # checkov:skip=CKV_AWS_136:AWS managed encryption is sufficient
  name                 = "bunshin/broker"
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = merge(local.common_tags, {
    Service = "broker"
  })
}

# trivy:ignore:AVD-AWS-0033 -- AWS managed encryption is sufficient
resource "aws_ecr_repository" "runner" {
  # checkov:skip=CKV_AWS_136:AWS managed encryption is sufficient
  name                 = "bunshin/runner"
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = merge(local.common_tags, {
    Service = "runner"
  })
}

resource "aws_ecr_lifecycle_policy" "nginx" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  repository = aws_ecr_repository.nginx.name

  policy = jsonencode({
    rules = [{
      rulePriority = 1
      description  = "Keep last 10 images"
      selection = {
        tagStatus   = "any"
        countType   = "imageCountMoreThan"
        countNumber = 10
      }
      action = {
        type = "expire"
      }
    }]
  })
}

resource "aws_ecr_lifecycle_policy" "broker" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  repository = aws_ecr_repository.broker.name

  policy = jsonencode({
    rules = [{
      rulePriority = 1
      description  = "Keep last 10 images"
      selection = {
        tagStatus   = "any"
        countType   = "imageCountMoreThan"
        countNumber = 10
      }
      action = {
        type = "expire"
      }
    }]
  })
}

resource "aws_ecr_lifecycle_policy" "runner" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  repository = aws_ecr_repository.runner.name

  policy = jsonencode({
    rules = [{
      rulePriority = 1
      description  = "Keep last 10 images"
      selection = {
        tagStatus   = "any"
        countType   = "imageCountMoreThan"
        countNumber = 10
      }
      action = {
        type = "expire"
      }
    }]
  })
}
