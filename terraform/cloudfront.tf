# AWS Managed cache policies and origin request policies
data "aws_cloudfront_cache_policy" "caching_optimized" {
  name = "Managed-CachingOptimized"
}

data "aws_cloudfront_cache_policy" "caching_disabled" {
  name = "Managed-CachingDisabled"
}

data "aws_cloudfront_origin_request_policy" "all_viewer" {
  name = "Managed-AllViewer"
}

# CloudFront Origin Access Control for S3
resource "aws_cloudfront_origin_access_control" "front" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  name                              = "bunshin-front"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

# CloudFront function for SPA routing
resource "aws_cloudfront_function" "spa_rewrite" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  name    = "bunshin-spa-rewrite"
  runtime = "cloudfront-js-2.0"
  publish = true
  code    = <<-EOF
    function handler(event) {
      var request = event.request;
      if (!request.uri.includes('.')) {
        request.uri = '/index.html';
      }
      return request;
    }
  EOF
}

# checkov:skip=CKV_AWS_310:CloudFront origin failover is not needed
# checkov:skip=CKV2_AWS_47:WAF is out of scope for initial deployment
# trivy:ignore:AVD-AWS-0010 -- CloudFront access logs are optional for initial deployment
# trivy:ignore:AVD-AWS-0012 -- CloudFront access logs are optional for initial deployment
# trivy:ignore:AVD-AWS-0011 -- WAF is out of scope for initial deployment
resource "aws_cloudfront_distribution" "main" {
  # checkov:skip=CKV_AWS_310:CloudFront origin failover is not needed
  # checkov:skip=CKV2_AWS_47:WAF is out of scope for initial deployment
  # checkov:skip=CKV_AWS_86:CloudFront access logs are optional for initial deployment
  # checkov:skip=CKV2_AWS_42:Custom domain is configured via variables
  # checkov:skip=CKV_AWS_374:Geo restriction is not needed
  # checkov:skip=CKV_AWS_68:WAF is out of scope for initial deployment
  # checkov:skip=CKV2_AWS_32:Response headers policy is not needed for initial deployment
  enabled             = true
  default_root_object = "index.html"
  price_class         = "PriceClass_200"
  aliases             = [var.domain_name]

  # S3 origin for static assets
  origin {
    domain_name              = aws_s3_bucket.front.bucket_regional_domain_name
    origin_id                = "s3"
    origin_access_control_id = aws_cloudfront_origin_access_control.front.id
  }

  # ALB origin for API
  origin {
    domain_name = aws_lb.main.dns_name
    origin_id   = "alb"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "http-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  # Default behavior: S3 static assets
  default_cache_behavior {
    allowed_methods        = ["GET", "HEAD", "OPTIONS"]
    cached_methods         = ["GET", "HEAD"]
    target_origin_id       = "s3"
    viewer_protocol_policy = "redirect-to-https"
    compress               = true

    cache_policy_id = data.aws_cloudfront_cache_policy.caching_optimized.id

    function_association {
      event_type   = "viewer-request"
      function_arn = aws_cloudfront_function.spa_rewrite.arn
    }
  }

  # /api/* behavior: forward to ALB with no caching
  ordered_cache_behavior {
    path_pattern             = "/api/*"
    allowed_methods          = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
    cached_methods           = ["GET", "HEAD"]
    target_origin_id         = "alb"
    viewer_protocol_policy   = "redirect-to-https"
    compress                 = false
    cache_policy_id          = data.aws_cloudfront_cache_policy.caching_disabled.id
    origin_request_policy_id = data.aws_cloudfront_origin_request_policy.all_viewer.id
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn      = var.acm_certificate_arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.2_2021"
  }

  tags = merge(local.common_tags, {
    Service = "cdn"
  })
}
