package poll

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// TestUnvote_Success は正常な投票取消を検証する。
func TestUnvote_Success(t *testing.T) {
	t.Parallel()
	var transactCalled bool
	client := &mockDynamoDBAPI{
		transactWriteItemsFn: func(_ context.Context, params *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
			transactCalled = true
			if len(params.TransactItems) != 2 {
				t.Errorf("expected 2 transact items, got %d", len(params.TransactItems))
			}
			deleteItem := params.TransactItems[0].Delete
			if deleteItem == nil {
				t.Fatal("expected Delete in first transact item")
			}
			sk := deleteItem.Key["connectionId"].(*types.AttributeValueMemberS).Value
			if sk != "visitor1#A" {
				t.Errorf("expected visitor1#A, got %s", sk)
			}
			return &dynamodb.TransactWriteItemsOutput{}, nil
		},
	}
	s := NewStore(client, "table")
	err := s.Unvote(context.Background(), "q1", "visitor1", "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !transactCalled {
		t.Error("expected TransactWriteItems to be called")
	}
}

// TestUnvote_VoteNotFound は投票が存在しない場合のエラーを検証する。
func TestUnvote_VoteNotFound(t *testing.T) {
	t.Parallel()
	client := &mockDynamoDBAPI{
		transactWriteItemsFn: func(_ context.Context, _ *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
			return nil, &types.TransactionCanceledException{
				CancellationReasons: []types.CancellationReason{
					{Code: stringPtr("ConditionalCheckFailed")},
					{Code: stringPtr("None")},
				},
			}
		},
	}
	s := NewStore(client, "table")
	err := s.Unvote(context.Background(), "q1", "visitor1", "A")
	if err != ErrVoteNotFound {
		t.Errorf("expected ErrVoteNotFound, got %v", err)
	}
}

// TestUnvote_TransactError は TransactWriteItems の非キャンセルエラーを検証する。
func TestUnvote_TransactError(t *testing.T) {
	t.Parallel()
	client := &mockDynamoDBAPI{
		transactWriteItemsFn: func(_ context.Context, _ *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
			return nil, fmt.Errorf("transact error")
		},
	}
	s := NewStore(client, "table")
	err := s.Unvote(context.Background(), "q1", "visitor1", "A")
	if err == nil {
		t.Fatal("expected error")
	}
}
