data "aws_region" "current" {}

resource "aws_ecs_cluster" "main" {
  name = "bunshin"

  setting {
    name  = "containerInsights"
    value = "enabled"
  }

  tags = merge(local.common_tags, {
    Service = "ecs"
  })
}

resource "aws_ecs_task_definition" "nginx" {
  # checkov:skip=CKV_AWS_336:nginx requires writable tmp directories
  family                   = "bunshin-nginx"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 256
  memory                   = 512
  execution_role_arn       = aws_iam_role.ecs_task_execution["nginx"].arn
  task_role_arn            = aws_iam_role.task["nginx"].arn

  runtime_platform {
    cpu_architecture        = "ARM64"
    operating_system_family = "LINUX"
  }

  container_definitions = jsonencode([{
    name      = "nginx"
    image     = "${aws_ecr_repository.service["nginx"].repository_url}:latest"
    essential = true

    portMappings = [{
      containerPort = local.ecs_services["nginx"].port
      protocol      = "tcp"
    }]

    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = aws_cloudwatch_log_group.ecs["nginx"].name
        "awslogs-region"        = data.aws_region.current.id
        "awslogs-stream-prefix" = "nginx"
      }
    }
  }])

  tags = merge(local.common_tags, {
    Service = "nginx"
  })
}

# broker task definition
resource "aws_ecs_task_definition" "broker" {
  family                   = "bunshin-broker"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 256
  memory                   = 512
  execution_role_arn       = aws_iam_role.ecs_task_execution["broker"].arn
  task_role_arn            = aws_iam_role.task["broker"].arn

  runtime_platform {
    cpu_architecture        = "ARM64"
    operating_system_family = "LINUX"
  }

  container_definitions = jsonencode([{
    name                   = "broker"
    image                  = "${aws_ecr_repository.service["broker"].repository_url}:latest"
    essential              = true
    readonlyRootFilesystem = true

    portMappings = [{
      containerPort = local.ecs_services["broker"].port
      protocol      = "tcp"
    }]

    environment = [
      { name = "AWS_REGION", value = data.aws_region.current.id },
    ]

    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = aws_cloudwatch_log_group.ecs["broker"].name
        "awslogs-region"        = data.aws_region.current.id
        "awslogs-stream-prefix" = "broker"
      }
    }
  }])

  tags = merge(local.common_tags, {
    Service = "broker"
  })
}

resource "aws_ecs_task_definition" "runner" {
  # checkov:skip=CKV_AWS_336:runner executes user commands and requires writable filesystem
  family                   = "bunshin-runner"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 256
  memory                   = 512
  execution_role_arn       = aws_iam_role.ecs_task_execution["runner"].arn
  task_role_arn            = aws_iam_role.task["runner"].arn

  runtime_platform {
    cpu_architecture        = "X86_64"
    operating_system_family = "LINUX"
  }

  container_definitions = jsonencode([{
    name      = "runner"
    image     = "${aws_ecr_repository.service["runner"].repository_url}:latest"
    essential = true

    portMappings = [{
      containerPort = local.ecs_services["runner"].port
      protocol      = "tcp"
    }]

    environment = [
      { name = "RUNNER_PORT", value = tostring(local.ecs_services["runner"].port) },
      { name = "BROKER_URL", value = "http://${aws_service_discovery_service.broker.name}.${aws_service_discovery_private_dns_namespace.internal.name}:${local.ecs_services["broker"].port}" },
    ]

    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = aws_cloudwatch_log_group.ecs["runner"].name
        "awslogs-region"        = data.aws_region.current.id
        "awslogs-stream-prefix" = "runner"
      }
    }
  }])

  tags = merge(local.common_tags, {
    Service = "runner"
  })
}

resource "aws_ecs_service" "nginx" {
  name                              = "bunshin-nginx"
  cluster                           = aws_ecs_cluster.main.id
  task_definition                   = aws_ecs_task_definition.nginx.arn
  desired_count                     = 6
  launch_type                       = "FARGATE"
  health_check_grace_period_seconds = 60

  network_configuration {
    subnets         = aws_subnet.private[*].id
    security_groups = [aws_security_group.nginx.id]
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.nginx.arn
    container_name   = "nginx"
    container_port   = local.ecs_services["nginx"].port
  }

  tags = merge(local.common_tags, {
    Service = "nginx"
  })
}

# broker ECS service with Cloud Map service discovery
resource "aws_ecs_service" "broker" {
  name            = "bunshin-broker"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.broker.arn
  desired_count   = 6
  launch_type     = "FARGATE"

  network_configuration {
    subnets         = aws_subnet.private[*].id
    security_groups = [aws_security_group.broker.id]
  }

  service_registries {
    registry_arn = aws_service_discovery_service.broker.arn
  }

  tags = merge(local.common_tags, {
    Service = "broker"
  })
}

# runner ECS service
resource "aws_ecs_service" "runner" {
  name            = "bunshin-runner"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.runner.arn
  desired_count   = 300
  launch_type     = "FARGATE"

  network_configuration {
    subnets         = aws_subnet.private[*].id
    security_groups = [aws_security_group.runner.id]
  }

  tags = merge(local.common_tags, {
    Service = "runner"
  })
}
