# Terraform と AWS プロバイダのバージョン制約
terraform {
  required_version = ">= 1.14"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# AWS プロバイダ設定。ローカル開発時は local_override.tf で上書きする
provider "aws" {
  region = "ap-northeast-1"
}

# KMS キーポリシーで参照するアカウント ID の取得
data "aws_caller_identity" "current" {}

# DynamoDB Runners テーブル暗号化用の KMS カスタマーマネージドキー
resource "aws_kms_key" "dynamodb" {
  description         = "KMS key for DynamoDB Runners table encryption"
  enable_key_rotation = true
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "EnableRootAccountAccess"
        Effect    = "Allow"
        Principal = { AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root" }
        Action    = "kms:*"
        Resource  = "*"
      }
    ]
  })
}

# Runner の状態管理テーブル。Session は独立エンティティとしては持たず、
# Runner の属性として currentSessionId を保持する単一テーブル設計。
#
# アクセスパターン:
#   1. runnerId で Runner を取得 -> GetItem
#   2. idle な Runner の一覧を取得 -> idle-index を Query
#   3. sessionId から Runner を特定 -> session-index を Query
#   4. Runner の現在の Session を取得 -> GetItem で currentSessionId を参照
#
# 両 GSI とも sparse index として機能する。
# idle 時は idleBucket のみ存在し、busy 時は currentSessionId のみ存在する。
resource "aws_dynamodb_table" "runners" {
  name         = "Runners"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "runnerId"

  attribute {
    name = "runnerId"
    type = "S"
  }

  attribute {
    name = "currentSessionId"
    type = "S"
  }

  attribute {
    name = "idleBucket"
    type = "S"
  }

  # busy な Runner だけが載る sparse GSI。sessionId から Runner を逆引きする
  global_secondary_index {
    name            = "session-index"
    hash_key        = "currentSessionId"
    projection_type = "ALL"
  }

  # idle な Runner だけが載る sparse GSI。空き Runner の検索に使う
  global_secondary_index {
    name            = "idle-index"
    hash_key        = "idleBucket"
    projection_type = "ALL"
  }

  point_in_time_recovery {
    enabled = true
  }

  # カスタマーマネージド KMS キーによるサーバーサイド暗号化
  server_side_encryption {
    enabled     = true
    kms_key_arn = aws_kms_key.dynamodb.arn
  }
}
