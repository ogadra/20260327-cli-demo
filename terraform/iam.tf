# ECS task execution role shared by all services
resource "aws_iam_role" "ecs_task_execution" {
  name = "bunshin-ecs-task-execution"

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
    Service = "ecs"
  })
}

# ECR pull permissions for task execution role
resource "aws_iam_role_policy" "execution_ecr" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  name = "bunshin-execution-ecr"
  role = aws_iam_role.ecs_task_execution.id

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
        Resource = [for s in local.ecs_service_names : aws_ecr_repository.service[s].arn]
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
  name = "bunshin-execution-logs"
  role = aws_iam_role.ecs_task_execution.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "logs:CreateLogStream",
        "logs:PutLogEvents",
      ]
      Resource = [for s in local.ecs_service_names : "${aws_cloudwatch_log_group.ecs[s].arn}:*"]
    }]
  })
}

# Task roles per service
resource "aws_iam_role" "task" {
  for_each = local.ecs_service_names

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
