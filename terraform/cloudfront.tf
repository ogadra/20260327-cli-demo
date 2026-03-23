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
  # checkov:skip=CKV2_AWS_42:Custom domain is configured separately via variables
  # checkov:skip=CKV_AWS_374:Geo restriction is not needed
  # checkov:skip=CKV_AWS_174:Default certificate uses CloudFront-managed TLS
  # checkov:skip=CKV_AWS_68:WAF is out of scope for initial deployment
  # checkov:skip=CKV2_AWS_32:Response headers policy is not needed for initial deployment
  enabled             = true
  default_root_object = "index.html"
  aliases             = var.domain_name != "" ? [var.domain_name] : []

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

    forwarded_values {
      query_string = false
      cookies {
        forward = "none"
      }
    }

    function_association {
      event_type   = "viewer-request"
      function_arn = aws_cloudfront_function.spa_rewrite.arn
    }

    compress = true
  }

  # /api/* behavior: forward to ALB
  ordered_cache_behavior {
    path_pattern           = "/api/*"
    allowed_methods        = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
    cached_methods         = ["GET", "HEAD"]
    target_origin_id       = "alb"
    viewer_protocol_policy = "redirect-to-https"

    forwarded_values {
      query_string = true
      headers      = ["*"]
      cookies {
        forward = "all"
      }
    }

    compress = false
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = var.acm_certificate_arn == ""
    acm_certificate_arn            = var.acm_certificate_arn != "" ? var.acm_certificate_arn : null
    ssl_support_method             = var.acm_certificate_arn != "" ? "sni-only" : null
    minimum_protocol_version       = var.acm_certificate_arn != "" ? "TLSv1.2_2021" : "TLSv1"
  }

  tags = merge(local.common_tags, {
    Service = "cdn"
  })
}
