# trivy:ignore:AVD-AWS-0054 -- ALB access logs are optional for initial deployment
# trivy:ignore:AVD-AWS-0053 -- ALB is intentionally internet-facing, protected by WAF
resource "aws_lb" "main" {
  # checkov:skip=CKV_AWS_91:ALB access logs are optional for initial deployment
  drop_invalid_header_fields = true
  # checkov:skip=CKV2_AWS_76:Log4j WAF rule is not needed, backend does not use Java
  # checkov:skip=CKV_AWS_150:Deletion protection is not needed for initial deployment
  name               = "bunshin"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb.id]
  subnets            = aws_subnet.public[*].id

  tags = merge(local.common_tags, {
    Service = "alb"
  })
}

# ACM certificate for ALB HTTPS listener
data "aws_acm_certificate" "alb" {
  domain   = var.domain_name
  statuses = ["ISSUED"]
}

# trivy:ignore:AVD-AWS-0053 -- target group uses HTTP, HTTPS terminates at ALB
resource "aws_lb_target_group" "nginx" {
  # checkov:skip=CKV_AWS_378:HTTPS terminates at ALB, target uses HTTP
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

# HTTPS listener with ACM certificate
resource "aws_lb_listener" "https" {
  load_balancer_arn = aws_lb.main.arn
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS13-1-2-2021-06"
  certificate_arn   = data.aws_acm_certificate.alb.arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.nginx.arn
  }

  tags = merge(local.common_tags, {
    Service = "alb"
  })
}

# HTTP listener redirects to HTTPS
resource "aws_lb_listener" "http" {
  # checkov:skip=CKV_AWS_2:HTTP listener is used for redirect to HTTPS only
  # checkov:skip=CKV_AWS_103:HTTP listener is used for redirect to HTTPS only
  load_balancer_arn = aws_lb.main.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type = "redirect"

    redirect {
      port        = "443"
      protocol    = "HTTPS"
      status_code = "HTTP_301"
    }
  }

  tags = merge(local.common_tags, {
    Service = "alb"
  })
}
