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

# IAM role for GitHub Actions deployment workflows
resource "aws_iam_role" "github_actions_deploy" {
  name = "bunshin-github-actions-deploy"

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
    Service = "cd"
  })
}

# ECR push permissions for all service repositories
resource "aws_iam_role_policy" "github_actions_ecr" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  name = "bunshin-github-actions-ecr"
  role = aws_iam_role.github_actions_deploy.id

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
        Resource = [for repo in aws_ecr_repository.service : repo.arn]
      },
      {
        Effect   = "Allow"
        Action   = "ecr:GetAuthorizationToken"
        Resource = "*"
      },
    ]
  })
}

# ECS deploy permissions
resource "aws_iam_role_policy" "github_actions_ecs" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  name = "bunshin-github-actions-ecs"
  role = aws_iam_role.github_actions_deploy.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "ecs:UpdateService",
        "ecs:DescribeServices",
      ]
      Resource = [
        aws_ecs_service.nginx.id,
        aws_ecs_service.broker.id,
        aws_ecs_service.runner.id,
      ]
    }]
  })
}

# S3 and CloudFront permissions for front deployment
resource "aws_iam_role_policy" "github_actions_front" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  name = "bunshin-github-actions-front"
  role = aws_iam_role.github_actions_deploy.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:DeleteObject",
          "s3:ListBucket",
        ]
        Resource = [
          aws_s3_bucket.front.arn,
          "${aws_s3_bucket.front.arn}/*",
        ]
      },
      {
        Effect   = "Allow"
        Action   = "cloudfront:CreateInvalidation"
        Resource = aws_cloudfront_distribution.main.arn
      },
    ]
  })
}
