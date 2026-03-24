variable "domain_name" {
  description = "Custom domain name for CloudFront distribution"
  type        = string
}

variable "acm_certificate_arn" {
  description = "ARN of ACM certificate in us-east-1 for custom domain"
  type        = string
}

variable "aws_profile" {
  description = "AWS CLI profile name for the target account"
  type        = string
}

variable "runner_desired_count" {
  description = "Desired number of runner ECS tasks"
  type        = number

  validation {
    condition     = var.runner_desired_count >= 0
    error_message = "runner_desired_count must be non-negative."
  }
}
