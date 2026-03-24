# ECS tasks assume role policy
data "aws_iam_policy_document" "ecs_tasks_assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

# ECS task execution roles per service
resource "aws_iam_role" "ecs_task_execution" {
  for_each = local.ecs_services

  name               = "bunshin-${each.key}-task-execution"
  assume_role_policy = data.aws_iam_policy_document.ecs_tasks_assume_role.json

  tags = merge(local.common_tags, {
    Service = each.key
  })
}

# ECR pull permissions scoped to each service repository
data "aws_iam_policy_document" "execution_ecr" {
  for_each = local.ecs_services

  statement {
    effect = "Allow"
    actions = [
      "ecr:GetDownloadUrlForLayer",
      "ecr:BatchGetImage",
      "ecr:BatchCheckLayerAvailability",
    ]
    resources = [aws_ecr_repository.service[each.key].arn]
  }

  statement {
    effect    = "Allow"
    actions   = ["ecr:GetAuthorizationToken"]
    resources = ["*"]
  }
}

resource "aws_iam_role_policy" "execution_ecr" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  for_each = local.ecs_services

  name   = "bunshin-${each.key}-execution-ecr"
  role   = aws_iam_role.ecs_task_execution[each.key].id
  policy = data.aws_iam_policy_document.execution_ecr[each.key].json
}

# CloudWatch Logs permissions scoped to each service log group
data "aws_iam_policy_document" "execution_logs" {
  for_each = local.ecs_services

  statement {
    effect = "Allow"
    actions = [
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]
    resources = ["${aws_cloudwatch_log_group.ecs[each.key].arn}:*"]
  }
}

resource "aws_iam_role_policy" "execution_logs" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  for_each = local.ecs_services

  name   = "bunshin-${each.key}-execution-logs"
  role   = aws_iam_role.ecs_task_execution[each.key].id
  policy = data.aws_iam_policy_document.execution_logs[each.key].json
}

# Task roles per service
resource "aws_iam_role" "task" {
  for_each = local.ecs_services

  name               = "bunshin-${each.key}-task"
  assume_role_policy = data.aws_iam_policy_document.ecs_tasks_assume_role.json

  tags = merge(local.common_tags, {
    Service = each.key
  })
}

# broker: DynamoDB access
data "aws_iam_policy_document" "broker_dynamodb" {
  statement {
    effect = "Allow"
    actions = [
      "dynamodb:GetItem",
      "dynamodb:PutItem",
      "dynamodb:UpdateItem",
      "dynamodb:DeleteItem",
    ]
    resources = [
      aws_dynamodb_table.runners.arn,
    ]
  }

  statement {
    effect = "Allow"
    actions = [
      "dynamodb:Query",
    ]
    resources = [
      "${aws_dynamodb_table.runners.arn}/index/session-index",
      "${aws_dynamodb_table.runners.arn}/index/idle-index",
    ]
  }
}

resource "aws_iam_role_policy" "broker_dynamodb" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  name   = "bunshin-broker-dynamodb"
  role   = aws_iam_role.task["broker"].id
  policy = data.aws_iam_policy_document.broker_dynamodb.json
}

# runner: Bedrock Converse access via inference profile
data "aws_iam_policy_document" "runner_bedrock" {
  statement {
    effect    = "Allow"
    actions   = ["bedrock:InvokeModel", "bedrock:Converse"]
    resources = ["arn:aws:bedrock:ap-northeast-1:${data.aws_caller_identity.current.account_id}:inference-profile/apac.anthropic.claude-sonnet-4-20250514-v1:0"]
  }
}

resource "aws_iam_role_policy" "runner_bedrock" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  name   = "bunshin-runner-bedrock"
  role   = aws_iam_role.task["runner"].id
  policy = data.aws_iam_policy_document.runner_bedrock.json
}
