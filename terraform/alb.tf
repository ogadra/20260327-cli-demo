# trivy:ignore:AVD-AWS-0054 -- ALB access logs are optional for initial deployment
# trivy:ignore:AVD-AWS-0053 -- ALB is intentionally internet-facing, accessed via CloudFront
resource "aws_lb" "main" {
  # checkov:skip=CKV_AWS_91:ALB access logs are optional for initial deployment
  drop_invalid_header_fields = true
  # checkov:skip=CKV2_AWS_28:WAF is out of scope for initial deployment
  # checkov:skip=CKV_AWS_150:Deletion protection is not needed for initial deployment
  # checkov:skip=CKV2_AWS_20:HTTPS terminates at CloudFront, ALB uses HTTP
  name               = "bunshin"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb.id]
  subnets            = aws_subnet.public[*].id

  tags = merge(local.common_tags, {
    Service = "alb"
  })
}

# trivy:ignore:AVD-AWS-0053 -- target group uses HTTP as HTTPS terminates at CloudFront
resource "aws_lb_target_group" "nginx" {
  # checkov:skip=CKV_AWS_378:HTTPS terminates at CloudFront, ALB uses HTTP
  name        = "bunshin-nginx"
  port        = local.ecs_services["nginx"].port
  protocol    = "HTTP"
  vpc_id      = aws_vpc.main.id
  target_type = "ip"

  health_check {
    path                = "/health"
    protocol            = "HTTP"
    matcher             = "200"
    interval            = 30
    timeout             = 5
    healthy_threshold   = 2
    unhealthy_threshold = 3
  }

  deregistration_delay = 30

  tags = merge(local.common_tags, {
    Service = "nginx"
  })
}

# trivy:ignore:AVD-AWS-0054 -- ALB uses HTTP as HTTPS terminates at CloudFront
resource "aws_lb_listener" "http" {
  # checkov:skip=CKV_AWS_2:HTTPS terminates at CloudFront, ALB uses HTTP
  # checkov:skip=CKV_AWS_103:HTTPS terminates at CloudFront, ALB uses HTTP
  load_balancer_arn = aws_lb.main.arn
  port              = local.ecs_services["nginx"].port
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.nginx.arn
  }

  tags = merge(local.common_tags, {
    Service = "alb"
  })
}
