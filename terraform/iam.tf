# ECS task execution roles per service
resource "aws_iam_role" "ecs_task_execution" {
  for_each = local.ecs_services

  name = "bunshin-${each.key}-task-execution"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ecs-tasks.amazonaws.com"
      }
    }]
  })

  tags = merge(local.common_tags, {
    Service = each.key
  })
}

# ECR pull permissions scoped to each service repository
resource "aws_iam_role_policy" "execution_ecr" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  for_each = local.ecs_services

  name = "bunshin-${each.key}-execution-ecr"
  role = aws_iam_role.ecs_task_execution[each.key].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ecr:GetDownloadUrlForLayer",
          "ecr:BatchGetImage",
          "ecr:BatchCheckLayerAvailability",
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

# CloudWatch Logs permissions scoped to each service log group
resource "aws_iam_role_policy" "execution_logs" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  for_each = local.ecs_services

  name = "bunshin-${each.key}-execution-logs"
  role = aws_iam_role.ecs_task_execution[each.key].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "logs:CreateLogStream",
        "logs:PutLogEvents",
      ]
      Resource = "${aws_cloudwatch_log_group.ecs[each.key].arn}:*"
    }]
  })
}

# Task roles per service
resource "aws_iam_role" "task" {
  for_each = local.ecs_services

  name = "bunshin-${each.key}-task"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ecs-tasks.amazonaws.com"
      }
    }]
  })

  tags = merge(local.common_tags, {
    Service = each.key
  })
}

# broker: DynamoDB access
resource "aws_iam_role_policy" "broker_dynamodb" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  name = "bunshin-broker-dynamodb"
  role = aws_iam_role.task["broker"].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:UpdateItem",
        "dynamodb:DeleteItem",
        "dynamodb:Query",
      ]
      Resource = [
        aws_dynamodb_table.runners.arn,
        "${aws_dynamodb_table.runners.arn}/index/*",
      ]
    }]
  })
}

# runner: Bedrock InvokeModel access
resource "aws_iam_role_policy" "runner_bedrock" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  name = "bunshin-runner-bedrock"
  role = aws_iam_role.task["runner"].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = "bedrock:InvokeModel"
      Resource = "arn:aws:bedrock:ap-northeast-1::foundation-model/*"
    }]
  })
}
