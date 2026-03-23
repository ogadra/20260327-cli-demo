variable "domain_name" {
  description = "Custom domain name for CloudFront distribution"
  type        = string
  default     = ""
}

variable "acm_certificate_arn" {
  description = "ARN of ACM certificate in us-east-1 for custom domain. Required if domain_name is set."
  type        = string
  default     = ""
}
