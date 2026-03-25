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
	meta := metaItem("q1", []string{"A", "B"}, 2, map[string]int{"A": 1, "B": 0})
	var transactCalled bool
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
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
	meta := metaItem("q1", []string{"A", "B"}, 2, map[string]int{"A": 1, "B": 0})
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
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
	meta := metaItem("q1", []string{"A", "B"}, 2, map[string]int{"A": 1, "B": 0})
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
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
	meta := metaItem("q1", []string{"A", "B"}, 2, map[string]int{"A": 0, "B": 0})
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
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

// TestSwitch_InvalidFromChoice は from が未定義の選択肢の場合にエラーを返すことを検証する。
func TestSwitch_InvalidFromChoice(t *testing.T) {
	t.Parallel()
	meta := metaItem("q1", []string{"A", "B"}, 2, map[string]int{"A": 0, "B": 0})
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
	}
	s := NewStore(client, "table")
	err := s.Switch(context.Background(), "q1", "visitor1", "C", "B")
	if err != ErrInvalidChoice {
		t.Errorf("expected ErrInvalidChoice, got %v", err)
	}
}

// TestSwitch_InvalidToChoice は to が未定義の選択肢の場合にエラーを返すことを検証する。
func TestSwitch_InvalidToChoice(t *testing.T) {
	t.Parallel()
	meta := metaItem("q1", []string{"A", "B"}, 2, map[string]int{"A": 0, "B": 0})
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
	}
	s := NewStore(client, "table")
	err := s.Switch(context.Background(), "q1", "visitor1", "A", "C")
	if err != ErrInvalidChoice {
		t.Errorf("expected ErrInvalidChoice, got %v", err)
	}
}

// TestSwitch_GetMetaError は getMeta エラーを検証する。
func TestSwitch_GetMetaError(t *testing.T) {
	t.Parallel()
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return nil, fmt.Errorf("get error")
		},
	}
	s := NewStore(client, "table")
	err := s.Switch(context.Background(), "q1", "visitor1", "A", "B")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestSwitch_PollNotFound は poll が存在しない場合のエラーを検証する。
func TestSwitch_PollNotFound(t *testing.T) {
	t.Parallel()
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
	}
	s := NewStore(client, "table")
	err := s.Switch(context.Background(), "q1", "visitor1", "A", "B")
	if err != ErrPollNotFound {
		t.Errorf("expected ErrPollNotFound, got %v", err)
	}
}
