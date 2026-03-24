# Route53 hosted zone for the parent domain
data "aws_route53_zone" "main" {
  name = join(".", slice(split(".", var.domain_name), 1, length(split(".", var.domain_name))))
}

# DNS alias record pointing the custom domain to the CloudFront distribution
resource "aws_route53_record" "cloudfront_alias" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  zone_id = data.aws_route53_zone.main.zone_id
  name    = var.domain_name
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.main.domain_name
    zone_id                = aws_cloudfront_distribution.main.hosted_zone_id
    evaluate_target_health = false
  }
}

# DNS AAAA alias record for IPv6 reachability
resource "aws_route53_record" "cloudfront_alias_aaaa" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  zone_id = data.aws_route53_zone.main.zone_id
  name    = var.domain_name
  type    = "AAAA"

  alias {
    name                   = aws_cloudfront_distribution.main.domain_name
    zone_id                = aws_cloudfront_distribution.main.hosted_zone_id
    evaluate_target_health = false
  }
}
