# GitHub OIDC provider for GitHub Actions keyless authentication
resource "aws_iam_openid_connect_provider" "github" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  url             = "https://token.actions.githubusercontent.com"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = ["ffffffffffffffffffffffffffffffffffffffff"]

  tags = merge(local.common_tags, {
    Service = "cd"
  })
}

locals {
  # ECS services that need deploy roles with ECR push and ECS update permissions
  ecs_deploy_services = toset(["nginx", "broker", "runner"])
}

# IAM roles for GitHub Actions deployment workflows per service
resource "aws_iam_role" "github_actions_deploy" {
  for_each = toset(concat(tolist(local.ecs_deploy_services), ["front"]))

  name = "bunshin-deploy-${each.key}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Federated = aws_iam_openid_connect_provider.github.arn
      }
      Action = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
          "token.actions.githubusercontent.com:sub" = "repo:ogadra/20260327-cli-demo:ref:refs/heads/main"
        }
      }
    }]
  })

  tags = merge(local.common_tags, {
    Service = each.key
  })
}

# ECR push permissions scoped to each service repository
resource "aws_iam_role_policy" "deploy_ecr" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  for_each = local.ecs_deploy_services

  name = "bunshin-deploy-${each.key}-ecr"
  role = aws_iam_role.github_actions_deploy[each.key].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ecr:GetDownloadUrlForLayer",
          "ecr:BatchGetImage",
          "ecr:BatchCheckLayerAvailability",
          "ecr:PutImage",
          "ecr:InitiateLayerUpload",
          "ecr:UploadLayerPart",
          "ecr:CompleteLayerUpload",
        ]
        Resource = aws_ecr_repository.service[each.key].arn
      },
      {
        Effect   = "Allow"
        Action   = "ecr:GetAuthorizationToken"
        Resource = "*"
      },
    ]
  })
}

# ECS deploy permissions scoped to each service
resource "aws_iam_role_policy" "deploy_ecs" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  for_each = local.ecs_deploy_services

  name = "bunshin-deploy-${each.key}-ecs"
  role = aws_iam_role.github_actions_deploy[each.key].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "ecs:UpdateService",
        "ecs:DescribeServices",
      ]
      Resource = "arn:aws:ecs:*:*:service/bunshin/bunshin-${each.key}"
    }]
  })
}

# S3 permissions for front deployment
data "aws_iam_policy_document" "deploy_front_s3" {
  statement {
    effect    = "Allow"
    actions   = ["s3:ListBucket"]
    resources = [aws_s3_bucket.front.arn]
  }
  statement {
    effect = "Allow"
    actions = [
      "s3:PutObject",
      "s3:DeleteObject",
    ]
    resources = ["${aws_s3_bucket.front.arn}/*"]
  }
}

resource "aws_iam_role_policy" "deploy_front_s3" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  name   = "bunshin-deploy-front-s3"
  role   = aws_iam_role.github_actions_deploy["front"].id
  policy = data.aws_iam_policy_document.deploy_front_s3.json
}
