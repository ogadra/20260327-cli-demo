# Deploy role ARNs for GitHub Actions OIDC authentication
output "deploy_role_arns" {
  description = "Map of service name to deploy IAM role ARN"
  value       = { for k, v in aws_iam_role.github_actions_deploy : k => v.arn }
}

# S3 bucket name for front asset deployment
output "front_s3_bucket" {
  description = "S3 bucket name for front static assets"
  value       = aws_s3_bucket.front.bucket
}
