# S3 bucket for front static assets
# trivy:ignore:AVD-AWS-0089 -- S3 bucket logging is optional for initial deployment
# trivy:ignore:AVD-AWS-0132 -- S3 bucket encryption uses AWS managed key
# trivy:ignore:AVD-AWS-0090 -- Versioning is not needed for build output
resource "aws_s3_bucket" "front" {
  # checkov:skip=CKV_AWS_18:S3 bucket logging is optional for initial deployment
  # checkov:skip=CKV_AWS_145:AWS managed encryption is sufficient
  # checkov:skip=CKV_AWS_144:Cross-region replication is not needed
  # checkov:skip=CKV2_AWS_62:Event notifications are not needed
  # checkov:skip=CKV2_AWS_61:Lifecycle configuration is not needed for static assets
  # checkov:skip=CKV_AWS_21:Versioning is not needed for build output
  bucket           = "bunshin-front"
  bucket_namespace = "account-regional"

  tags = merge(local.common_tags, {
    Service = "front"
  })
}

resource "aws_s3_bucket_public_access_block" "front" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  bucket = aws_s3_bucket.front.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_policy" "front" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  bucket = aws_s3_bucket.front.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid       = "AllowCloudFrontOAC"
      Effect    = "Allow"
      Principal = { Service = "cloudfront.amazonaws.com" }
      Action    = "s3:GetObject"
      Resource  = "${aws_s3_bucket.front.arn}/*"
      Condition = {
        StringEquals = {
          "AWS:SourceArn" = aws_cloudfront_distribution.main.arn
        }
      }
    }]
  })

  depends_on = [aws_s3_bucket_public_access_block.front]
}
