# Cache policy for S3 static assets: no caching, no cookies/query strings
# 将来的に変更するのでマネージドポリシーではなく自前定義
resource "aws_cloudfront_cache_policy" "static_assets" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  name        = "bunshin-static-assets"
  min_ttl     = 0
  default_ttl = 0
  max_ttl     = 0

  parameters_in_cache_key_and_forwarded_to_origin {
    cookies_config {
      cookie_behavior = "none"
    }
    headers_config {
      header_behavior = "none"
    }
    query_strings_config {
      query_string_behavior = "none"
    }
    enable_accept_encoding_gzip   = false
    enable_accept_encoding_brotli = false
  }
}

# Origin request policy for API: forward all viewer headers and cookies to ALB
resource "aws_cloudfront_origin_request_policy" "api_all_viewer" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  name = "bunshin-api-all-viewer"

  cookies_config {
    cookie_behavior = "all"
  }
  headers_config {
    header_behavior = "allViewer"
  }
  query_strings_config {
    query_string_behavior = "all"
  }
}

# CloudFront Origin Access Control for S3
resource "aws_cloudfront_origin_access_control" "front" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  name                              = "bunshin-front"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

# AWS managed cache policy: no caching for API Gateway origins
data "aws_cloudfront_cache_policy" "caching_disabled" {
  name = "Managed-CachingDisabled"
}

# AWS managed origin request policy: forward all viewer headers except Host
data "aws_cloudfront_origin_request_policy" "all_viewer_except_host" {
  name = "Managed-AllViewerExceptHostHeader"
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

  # WebSocket API Gateway origin for presenter slide sync
  origin {
    domain_name = "${aws_apigatewayv2_api.presenter_websocket.id}.execute-api.${data.aws_region.current.id}.amazonaws.com"
    origin_id   = local.presenter_cf_origin_id.websocket

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  # HTTP API Gateway origin for presenter login
  origin {
    domain_name = "${aws_apigatewayv2_api.presenter_login.id}.execute-api.${data.aws_region.current.id}.amazonaws.com"
    origin_id   = local.presenter_cf_origin_id.login

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
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

    cache_policy_id = aws_cloudfront_cache_policy.static_assets.id

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
    origin_request_policy_id = aws_cloudfront_origin_request_policy.api_all_viewer.id
  }

  # /ws behavior: forward to WebSocket API Gateway
  ordered_cache_behavior {
    path_pattern             = "/ws"
    allowed_methods          = ["GET", "HEAD"]
    cached_methods           = ["GET", "HEAD"]
    target_origin_id         = local.presenter_cf_origin_id.websocket
    viewer_protocol_policy   = "https-only"
    compress                 = false
    cache_policy_id          = data.aws_cloudfront_cache_policy.caching_disabled.id
    origin_request_policy_id = data.aws_cloudfront_origin_request_policy.all_viewer_except_host.id
  }

  # /login behavior: forward to HTTP API Gateway
  ordered_cache_behavior {
    path_pattern             = "/login"
    allowed_methods          = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
    cached_methods           = ["GET", "HEAD"]
    target_origin_id         = local.presenter_cf_origin_id.login
    viewer_protocol_policy   = "https-only"
    compress                 = false
    cache_policy_id          = data.aws_cloudfront_cache_policy.caching_disabled.id
    origin_request_policy_id = data.aws_cloudfront_origin_request_policy.all_viewer_except_host.id
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
