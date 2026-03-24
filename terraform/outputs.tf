# GitHub Actions secrets to configure for deploy workflows
output "github_actions_secrets" {
  description = "Map of GitHub Actions secret names to their values"
  value = merge(
    { for k, v in aws_iam_role.github_actions_deploy : "DEPLOY_${upper(k)}_ROLE_ARN" => v.arn },
    { "FRONT_S3_BUCKET" = aws_s3_bucket.front.bucket }
  )
}
