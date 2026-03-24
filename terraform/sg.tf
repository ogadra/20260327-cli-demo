resource "aws_security_group" "alb" {
  name_prefix = "bunshin-alb-"
  description = "Security group for ALB"
  vpc_id      = aws_vpc.main.id

  tags = merge(local.common_tags, {
    Name    = "bunshin-alb"
    Service = "alb"
  })

  lifecycle {
    create_before_destroy = true
  }
}

# ALB inbound: HTTP from CloudFront prefix list
data "aws_ec2_managed_prefix_list" "cloudfront" {
  name = "com.amazonaws.global.cloudfront.origin-facing"
}

resource "aws_security_group_rule" "alb_ingress_cloudfront" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  type              = "ingress"
  from_port         = 80
  to_port           = 80
  protocol          = "tcp"
  prefix_list_ids   = [data.aws_ec2_managed_prefix_list.cloudfront.id]
  security_group_id = aws_security_group.alb.id
  description       = "HTTP from CloudFront"
}

# ALB outbound: to nginx
resource "aws_security_group_rule" "alb_egress_nginx" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  type                     = "egress"
  from_port                = local.ecs_services["nginx"].port
  to_port                  = local.ecs_services["nginx"].port
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.nginx.id
  security_group_id        = aws_security_group.alb.id
  description              = "HTTP to nginx"
}

resource "aws_security_group" "nginx" {
  name_prefix = "bunshin-nginx-"
  description = "Security group for nginx ECS tasks"
  vpc_id      = aws_vpc.main.id

  tags = merge(local.common_tags, {
    Name    = "bunshin-nginx"
    Service = "nginx"
  })

  lifecycle {
    create_before_destroy = true
  }
}

# nginx inbound: from ALB
resource "aws_security_group_rule" "nginx_ingress_alb" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  type                     = "ingress"
  from_port                = local.ecs_services["nginx"].port
  to_port                  = local.ecs_services["nginx"].port
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.alb.id
  security_group_id        = aws_security_group.nginx.id
  description              = "HTTP from ALB"
}

# nginx outbound: to broker
resource "aws_security_group_rule" "nginx_egress_broker" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  type                     = "egress"
  from_port                = local.ecs_services["broker"].port
  to_port                  = local.ecs_services["broker"].port
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.broker.id
  security_group_id        = aws_security_group.nginx.id
  description              = "HTTP to broker"
}

# nginx outbound: to runner
resource "aws_security_group_rule" "nginx_egress_runner" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  type                     = "egress"
  from_port                = local.ecs_services["runner"].port
  to_port                  = local.ecs_services["runner"].port
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.runner.id
  security_group_id        = aws_security_group.nginx.id
  description              = "HTTP to runner"
}

resource "aws_security_group" "broker" {
  name_prefix = "bunshin-broker-"
  description = "Security group for broker ECS tasks"
  vpc_id      = aws_vpc.main.id

  tags = merge(local.common_tags, {
    Name    = "bunshin-broker"
    Service = "broker"
  })

  lifecycle {
    create_before_destroy = true
  }
}

# broker inbound: from nginx
resource "aws_security_group_rule" "broker_ingress_nginx" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  type                     = "ingress"
  from_port                = local.ecs_services["broker"].port
  to_port                  = local.ecs_services["broker"].port
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.nginx.id
  security_group_id        = aws_security_group.broker.id
  description              = "HTTP from nginx"
}

# broker inbound: from runner
resource "aws_security_group_rule" "broker_ingress_runner" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  type                     = "ingress"
  from_port                = local.ecs_services["broker"].port
  to_port                  = local.ecs_services["broker"].port
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.runner.id
  security_group_id        = aws_security_group.broker.id
  description              = "HTTP from runner"
}

# broker outbound: to DynamoDB VPC endpoint
resource "aws_security_group_rule" "broker_egress_dynamodb" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  type              = "egress"
  from_port         = 443
  to_port           = 443
  protocol          = "tcp"
  prefix_list_ids   = [aws_vpc_endpoint.dynamodb.prefix_list_id]
  security_group_id = aws_security_group.broker.id
  description       = "HTTPS to DynamoDB VPC endpoint"
}

resource "aws_security_group" "runner" {
  name_prefix = "bunshin-runner-"
  description = "Security group for runner ECS tasks"
  vpc_id      = aws_vpc.main.id

  tags = merge(local.common_tags, {
    Name    = "bunshin-runner"
    Service = "runner"
  })

  lifecycle {
    create_before_destroy = true
  }
}

# runner inbound: from nginx
resource "aws_security_group_rule" "runner_ingress_nginx" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  type                     = "ingress"
  from_port                = local.ecs_services["runner"].port
  to_port                  = local.ecs_services["runner"].port
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.nginx.id
  security_group_id        = aws_security_group.runner.id
  description              = "HTTP from nginx"
}

# runner outbound: to broker
resource "aws_security_group_rule" "runner_egress_broker" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  type                     = "egress"
  from_port                = local.ecs_services["broker"].port
  to_port                  = local.ecs_services["broker"].port
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.broker.id
  security_group_id        = aws_security_group.runner.id
  description              = "HTTP to broker"
}

# trivy:ignore:AVD-AWS-0104 -- runner requires outbound internet access
resource "aws_security_group_rule" "runner_egress_https" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  type              = "egress"
  from_port         = 443
  to_port           = 443
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.runner.id
  description       = "HTTPS to internet"
}

# ECS tasks outbound: to ECR and CloudWatch Logs VPC endpoints
resource "aws_security_group_rule" "ecs_egress_ecr" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  for_each = {
    nginx  = aws_security_group.nginx.id
    broker = aws_security_group.broker.id
    runner = aws_security_group.runner.id
  }

  type                     = "egress"
  from_port                = 443
  to_port                  = 443
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.ecr_endpoint.id
  security_group_id        = each.value
  description              = "HTTPS to ECR and CloudWatch Logs VPC endpoints"
}

# ECS tasks outbound: to S3 Gateway Endpoint for ECR image layers
resource "aws_security_group_rule" "ecs_egress_s3" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  for_each = {
    nginx  = aws_security_group.nginx.id
    broker = aws_security_group.broker.id
    runner = aws_security_group.runner.id
  }

  type              = "egress"
  from_port         = 443
  to_port           = 443
  protocol          = "tcp"
  prefix_list_ids   = [aws_vpc_endpoint.s3.prefix_list_id]
  security_group_id = each.value
  description       = "HTTPS to S3 VPC endpoint"
}

# runner outbound: to Bedrock Runtime VPC endpoint
resource "aws_security_group_rule" "runner_egress_bedrock" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  type                     = "egress"
  from_port                = 443
  to_port                  = 443
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.bedrock_endpoint.id
  security_group_id        = aws_security_group.runner.id
  description              = "HTTPS to Bedrock Runtime VPC endpoint"
}
