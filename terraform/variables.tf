variable "domain_name" {
  description = "Domain name for ALB ACM certificate lookup"
  type        = string
}

variable "aws_profile" {
  description = "AWS CLI profile name for the target account"
  type        = string
}

variable "proxy_secret" {
  description = "Secret header value for Cloudflare Workers proxy verification via WAF"
  type        = string
  sensitive   = true
}

variable "runner_desired_count" {
  description = "Desired number of runner ECS tasks"
  type        = number

  validation {
    condition     = var.runner_desired_count >= 0
    error_message = "runner_desired_count must be non-negative."
  }
}
