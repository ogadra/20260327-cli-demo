# WAF Web ACL for ALB: default block, allow only requests with valid proxy secret
resource "aws_wafv2_web_acl" "alb" {
  # checkov:skip=CKV_AWS_192:Log4j protection is not needed, backend does not use Java
  # checkov:skip=CKV2_AWS_31:WAF logging is not needed for initial deployment
  name  = "bunshin-alb"
  scope = "REGIONAL"

  default_action {
    block {}
  }

  # Allow requests with matching X-Proxy-Secret header from Cloudflare Workers
  rule {
    name     = "allow-proxy-secret"
    priority = 1

    action {
      allow {}
    }

    statement {
      byte_match_statement {
        search_string = var.proxy_secret

        field_to_match {
          single_header {
            name = "x-proxy-secret"
          }
        }

        text_transformation {
          priority = 0
          type     = "NONE"
        }

        positional_constraint = "EXACTLY"
      }
    }

    visibility_config {
      sampled_requests_enabled   = true
      cloudwatch_metrics_enabled = true
      metric_name                = "bunshin-allow-proxy-secret"
    }
  }

  visibility_config {
    sampled_requests_enabled   = true
    cloudwatch_metrics_enabled = true
    metric_name                = "bunshin-alb-waf"
  }

  tags = merge(local.common_tags, {
    Service = "waf"
  })
}

# Associate WAF Web ACL with ALB
resource "aws_wafv2_web_acl_association" "alb" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  resource_arn = aws_lb.main.arn
  web_acl_arn  = aws_wafv2_web_acl.alb.arn
}
