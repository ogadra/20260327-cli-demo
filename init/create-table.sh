#!/bin/sh
# Create the Runners table in DynamoDB Local.
# Schema must match terraform/main.tf aws_dynamodb_table.runners.
set -eu

ENDPOINT="http://dynamodb-local:8000"

OUTPUT=$(aws dynamodb create-table \
  --endpoint-url "$ENDPOINT" \
  --table-name Runners \
  --billing-mode PAY_PER_REQUEST \
  --attribute-definitions \
    AttributeName=runnerId,AttributeType=S \
    AttributeName=currentSessionId,AttributeType=S \
    AttributeName=idleBucket,AttributeType=S \
  --key-schema AttributeName=runnerId,KeyType=HASH \
  --global-secondary-indexes \
    'IndexName=session-index,KeySchema=[{AttributeName=currentSessionId,KeyType=HASH}],Projection={ProjectionType=ALL}' \
    'IndexName=idle-index,KeySchema=[{AttributeName=idleBucket,KeyType=HASH}],Projection={ProjectionType=ALL}' \
  --region ap-northeast-1 2>&1) && echo "$OUTPUT" || {
  echo "$OUTPUT" >&2
  echo "$OUTPUT" | grep -q "ResourceInUseException"
}
