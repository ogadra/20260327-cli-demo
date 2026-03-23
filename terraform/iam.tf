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

resource "aws_iam_role_policy_attachment" "ecs_task_execution" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  role       = aws_iam_role.ecs_task_execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

# broker task role with DynamoDB access
resource "aws_iam_role" "broker_task" {
  name = "bunshin-broker-task"

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
    Service = "broker"
  })
}

resource "aws_iam_role_policy" "broker_dynamodb" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  name = "bunshin-broker-dynamodb"
  role = aws_iam_role.broker_task.id

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

# runner task role
resource "aws_iam_role" "runner_task" {
  name = "bunshin-runner-task"

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
    Service = "runner"
  })
}

# nginx task role
resource "aws_iam_role" "nginx_task" {
  name = "bunshin-nginx-task"

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
    Service = "nginx"
  })
}
