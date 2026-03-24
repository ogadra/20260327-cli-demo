# Deploy role ARNs for GitHub Actions OIDC authentication
output "deploy_broker_role_arn" {
  description = "IAM role ARN for broker deployment workflow"
  value       = aws_iam_role.github_actions_deploy["broker"].arn
}

output "deploy_front_role_arn" {
  description = "IAM role ARN for front deployment workflow"
  value       = aws_iam_role.github_actions_deploy["front"].arn
}

output "deploy_nginx_role_arn" {
  description = "IAM role ARN for nginx deployment workflow"
  value       = aws_iam_role.github_actions_deploy["nginx"].arn
}

output "deploy_runner_role_arn" {
  description = "IAM role ARN for runner deployment workflow"
  value       = aws_iam_role.github_actions_deploy["runner"].arn
}

# S3 bucket name for front asset deployment
output "front_s3_bucket" {
  description = "S3 bucket name for front static assets"
  value       = aws_s3_bucket.front.bucket
}
