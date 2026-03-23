variable "domain_name" {
  description = "Custom domain name for CloudFront distribution"
  type        = string
}

variable "acm_certificate_arn" {
  description = "ARN of ACM certificate in us-east-1 for custom domain"
  type        = string
}
