terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

variable "dynamodb_endpoint" {
  description = "DynamoDB endpoint URL (for local development)"
  type        = string
  default     = null
}

provider "aws" {
  region = "ap-northeast-1"

  dynamic "endpoints" {
    for_each = var.dynamodb_endpoint != null ? [var.dynamodb_endpoint] : []
    content {
      dynamodb = endpoints.value
    }
  }

  skip_credentials_validation = var.dynamodb_endpoint != null
  skip_metadata_api_check     = var.dynamodb_endpoint != null
  skip_requesting_account_id  = var.dynamodb_endpoint != null

  access_key = var.dynamodb_endpoint != null ? "dummy" : null
  secret_key = var.dynamodb_endpoint != null ? "dummy" : null
}

resource "aws_dynamodb_table" "runners" {
  name         = "Runners"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "runnerId"

  attribute {
    name = "runnerId"
    type = "S"
  }

  attribute {
    name = "status"
    type = "S"
  }

  global_secondary_index {
    name            = "status-index"
    hash_key        = "status"
    projection_type = "ALL"
  }
}

resource "aws_dynamodb_table" "sessions" {
  name         = "Sessions"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "sessionId"

  attribute {
    name = "sessionId"
    type = "S"
  }

  attribute {
    name = "runnerId"
    type = "S"
  }

  global_secondary_index {
    name            = "runner-index"
    hash_key        = "runnerId"
    projection_type = "ALL"
  }
}
