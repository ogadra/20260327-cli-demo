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
#
# trivy:ignore:AVD-AWS-0024 -- PITR is not required for ephemeral runner state
# trivy:ignore:AVD-AWS-0025 -- AWS managed encryption is sufficient for this use case
resource "aws_dynamodb_table" "runners" {
  # checkov:skip=CKV_AWS_28:PITR is not required for ephemeral runner state
  # checkov:skip=CKV_AWS_119:AWS managed encryption is sufficient for this use case
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
    projection_type = "ALL"

    key_schema {
      attribute_name = "currentSessionId"
      key_type       = "HASH"
    }
  }

  # idle な Runner だけが載る sparse GSI。空き Runner の検索に使う
  global_secondary_index {
    name            = "idle-index"
    projection_type = "ALL"

    key_schema {
      attribute_name = "idleBucket"
      key_type       = "HASH"
    }
  }

  tags = {
    Project     = "Bunshin"
    Environment = "shared"
    Service     = "broker"
    ManagedBy   = "terraform"
  }
}
