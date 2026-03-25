package poll

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// TestSwitch_Success は正常な投票変更を検証する。
func TestSwitch_Success(t *testing.T) {
	t.Parallel()
	var transactCalled bool
	client := &mockDynamoDBAPI{
		transactWriteItemsFn: func(_ context.Context, params *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
			transactCalled = true
			if len(params.TransactItems) != 3 {
				t.Errorf("expected 3 transact items, got %d", len(params.TransactItems))
			}
			deleteSK := params.TransactItems[0].Delete.Key["connectionId"].(*types.AttributeValueMemberS).Value
			if deleteSK != "visitor1#A" {
				t.Errorf("expected visitor1#A deleted, got %s", deleteSK)
			}
			putSK := params.TransactItems[1].Put.Item["connectionId"].(*types.AttributeValueMemberS).Value
			if putSK != "visitor1#B" {
				t.Errorf("expected visitor1#B put, got %s", putSK)
			}
			return &dynamodb.TransactWriteItemsOutput{}, nil
		},
	}
	s := NewStore(client, "table")
	err := s.Switch(context.Background(), "q1", "visitor1", "A", "B")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !transactCalled {
		t.Error("expected TransactWriteItems to be called")
	}
}

// TestSwitch_FromNotFound は旧選択肢が存在しない場合のエラーを検証する。
func TestSwitch_FromNotFound(t *testing.T) {
	t.Parallel()
	client := &mockDynamoDBAPI{
		transactWriteItemsFn: func(_ context.Context, _ *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
			return nil, &types.TransactionCanceledException{
				CancellationReasons: []types.CancellationReason{
					{Code: stringPtr("ConditionalCheckFailed")},
					{Code: stringPtr("None")},
					{Code: stringPtr("None")},
				},
			}
		},
	}
	s := NewStore(client, "table")
	err := s.Switch(context.Background(), "q1", "visitor1", "A", "B")
	if err != ErrVoteNotFound {
		t.Errorf("expected ErrVoteNotFound, got %v", err)
	}
}

// TestSwitch_DuplicateTo は新選択肢が重複している場合のエラーを検証する。
func TestSwitch_DuplicateTo(t *testing.T) {
	t.Parallel()
	client := &mockDynamoDBAPI{
		transactWriteItemsFn: func(_ context.Context, _ *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
			return nil, &types.TransactionCanceledException{
				CancellationReasons: []types.CancellationReason{
					{Code: stringPtr("None")},
					{Code: stringPtr("ConditionalCheckFailed")},
					{Code: stringPtr("None")},
				},
			}
		},
	}
	s := NewStore(client, "table")
	err := s.Switch(context.Background(), "q1", "visitor1", "A", "B")
	if err != ErrDuplicateVote {
		t.Errorf("expected ErrDuplicateVote, got %v", err)
	}
}

// TestSwitch_TransactError は TransactWriteItems の非キャンセルエラーを検証する。
func TestSwitch_TransactError(t *testing.T) {
	t.Parallel()
	client := &mockDynamoDBAPI{
		transactWriteItemsFn: func(_ context.Context, _ *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
			return nil, fmt.Errorf("transact error")
		},
	}
	s := NewStore(client, "table")
	err := s.Switch(context.Background(), "q1", "visitor1", "A", "B")
	if err == nil {
		t.Fatal("expected error")
	}
}
