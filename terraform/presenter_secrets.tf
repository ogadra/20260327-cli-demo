# Presenter 認証用パスワードハッシュ。bcrypt ハッシュを保持する
# trivy:ignore:AVD-AWS-0098 -- AWS managed encryption is sufficient for password hash
resource "aws_secretsmanager_secret" "presenter_password_hash" {
  # checkov:skip=CKV2_AWS_57:Auto rotation not applicable for static password hash
  # checkov:skip=CKV_AWS_149:AWS managed encryption is sufficient for password hash
  name = "bunshin-presenter-password-hash"

  tags = merge(local.common_tags, {
    Service = "presenter"
  })
}
