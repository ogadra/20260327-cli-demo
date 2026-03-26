# Presenter WebSocket 接続管理テーブル。room 単位で connectionId を管理する
#
# アクセスパターン:
#   1. room の全接続を取得 -> Query by room
#   2. connectionId で接続を削除 -> DeleteItem
#   3. room の接続数を取得 -> Query by room (count)
#
# trivy:ignore:AVD-AWS-0024 -- PITR is not required for ephemeral WebSocket connections
# trivy:ignore:AVD-AWS-0025 -- AWS managed encryption is sufficient for this use case
resource "aws_dynamodb_table" "presenter_ws_connections" {
  # checkov:skip=CKV_AWS_28:PITR is not required for ephemeral WebSocket connections
  # checkov:skip=CKV_AWS_119:AWS managed encryption is sufficient for this use case
  name         = "bunshin-presenter-ws-connections"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "room"
  range_key    = "connectionId"

  attribute {
    name = "room"
    type = "S"
  }

  attribute {
    name = "connectionId"
    type = "S"
  }

  ttl {
    attribute_name = "ttl"
    enabled        = true
  }

  tags = merge(local.common_tags, {
    Service     = "presenter"
    Environment = "shared"
  })
}

# Presenter アンケート投票テーブル。pollId 単位で connectionId ごとの投票を管理する
#
# アクセスパターン:
#   1. pollId の全投票を取得 -> Query by pollId
#   2. connectionId の投票を取得/更新/削除 -> GetItem/UpdateItem/DeleteItem
#
# trivy:ignore:AVD-AWS-0024 -- PITR is not required for ephemeral poll data
# trivy:ignore:AVD-AWS-0025 -- AWS managed encryption is sufficient for this use case
resource "aws_dynamodb_table" "presenter_poll_votes" {
  # checkov:skip=CKV_AWS_28:PITR is not required for ephemeral poll data
  # checkov:skip=CKV_AWS_119:AWS managed encryption is sufficient for this use case
  name         = "bunshin-presenter-poll-votes"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "pollId"
  range_key    = "connectionId"

  attribute {
    name = "pollId"
    type = "S"
  }

  attribute {
    name = "connectionId"
    type = "S"
  }

  ttl {
    attribute_name = "ttl"
    enabled        = true
  }

  tags = merge(local.common_tags, {
    Service     = "presenter"
    Environment = "shared"
  })
}

# Presenter 認証セッションテーブル。token でプレゼンター認証状態を管理する
#
# アクセスパターン:
#   1. token でセッションを取得 -> GetItem
#   2. token でセッションを作成 -> PutItem
#
# trivy:ignore:AVD-AWS-0024 -- PITR is not required for ephemeral session data
# trivy:ignore:AVD-AWS-0025 -- AWS managed encryption is sufficient for this use case
resource "aws_dynamodb_table" "presenter_sessions" {
  # checkov:skip=CKV_AWS_28:PITR is not required for ephemeral session data
  # checkov:skip=CKV_AWS_119:AWS managed encryption is sufficient for this use case
  name         = "bunshin-presenter-sessions"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "token"

  attribute {
    name = "token"
    type = "S"
  }

  ttl {
    attribute_name = "ttl"
    enabled        = true
  }

  tags = merge(local.common_tags, {
    Service     = "presenter"
    Environment = "shared"
  })
}

# Presenter room 状態テーブル。room ごとの現在のスライド位置を保持する
#
# アクセスパターン:
#   1. room の現在のページを取得 -> GetItem by room
#   2. room の現在のページを更新 -> PutItem by room
#
# trivy:ignore:AVD-AWS-0024 -- PITR is not required for room state
# trivy:ignore:AVD-AWS-0025 -- AWS managed encryption is sufficient for this use case
resource "aws_dynamodb_table" "presenter_room_state" {
  # checkov:skip=CKV_AWS_28:PITR is not required for room state
  # checkov:skip=CKV_AWS_119:AWS managed encryption is sufficient for this use case
  name         = "bunshin-presenter-room-state"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "room"

  attribute {
    name = "room"
    type = "S"
  }

  tags = merge(local.common_tags, {
    Service     = "presenter"
    Environment = "shared"
  })
}
