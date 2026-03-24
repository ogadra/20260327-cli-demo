# Terraform と AWS プロバイダのバージョン制約
terraform {
  required_version = ">= 1.14"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.37"
    }
  }
}

# AWS プロバイダ設定。環境ごとの tfvars で profile を切り替える
provider "aws" {
  region  = "ap-northeast-1"
  profile = var.aws_profile
}
