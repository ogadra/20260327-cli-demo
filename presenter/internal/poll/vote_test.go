package poll

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// TestVote_Success は正常な投票を検証する。
func TestVote_Success(t *testing.T) {
	t.Parallel()
	meta := metaItem("q1", []string{"A", "B"}, 2, map[string]int{"A": 0, "B": 0})
	var transactCalled bool
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{}}, nil
		},
		transactWriteItemsFn: func(_ context.Context, params *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
			transactCalled = true
			if len(params.TransactItems) != 2 {
				t.Errorf("expected 2 transact items, got %d", len(params.TransactItems))
			}
			putItem := params.TransactItems[0].Put
			if putItem == nil {
				t.Fatal("expected Put in first transact item")
			}
			sk := putItem.Item["connectionId"].(*types.AttributeValueMemberS).Value
			if sk != "visitor1#A" {
				t.Errorf("expected visitor1#A, got %s", sk)
			}
			return &dynamodb.TransactWriteItemsOutput{}, nil
		},
	}
	s := NewStore(client, "table")
	err := s.Vote(context.Background(), "q1", "visitor1", "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !transactCalled {
		t.Error("expected TransactWriteItems to be called")
	}
}

// TestVote_MaxChoicesExceeded は最大選択数超過を検証する。
func TestVote_MaxChoicesExceeded(t *testing.T) {
	t.Parallel()
	meta := metaItem("q1", []string{"A", "B"}, 1, map[string]int{"A": 1, "B": 0})
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{"connectionId": &types.AttributeValueMemberS{Value: "visitor1#A"}},
				},
			}, nil
		},
	}
	s := NewStore(client, "table")
	err := s.Vote(context.Background(), "q1", "visitor1", "B")
	if err != ErrMaxChoicesExceeded {
		t.Errorf("expected ErrMaxChoicesExceeded, got %v", err)
	}
}

// TestVote_DuplicateVote は重複投票を検証する。
func TestVote_DuplicateVote(t *testing.T) {
	t.Parallel()
	meta := metaItem("q1", []string{"A", "B"}, 2, map[string]int{"A": 1, "B": 0})
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{}}, nil
		},
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
	err := s.Vote(context.Background(), "q1", "visitor1", "A")
	if err != ErrDuplicateVote {
		t.Errorf("expected ErrDuplicateVote, got %v", err)
	}
}

// TestVote_GetMetaError は getMeta エラーを検証する。
func TestVote_GetMetaError(t *testing.T) {
	t.Parallel()
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return nil, fmt.Errorf("get error")
		},
	}
	s := NewStore(client, "table")
	err := s.Vote(context.Background(), "q1", "visitor1", "A")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestVote_QueryMyChoicesError は myChoices クエリエラーを検証する。
func TestVote_QueryMyChoicesError(t *testing.T) {
	t.Parallel()
	meta := metaItem("q1", []string{"A"}, 1, map[string]int{"A": 0})
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return nil, fmt.Errorf("query error")
		},
	}
	s := NewStore(client, "table")
	err := s.Vote(context.Background(), "q1", "visitor1", "A")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestVote_TransactError は TransactWriteItems の非キャンセルエラーを検証する。
func TestVote_TransactError(t *testing.T) {
	t.Parallel()
	meta := metaItem("q1", []string{"A"}, 1, map[string]int{"A": 0})
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{}}, nil
		},
		transactWriteItemsFn: func(_ context.Context, _ *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
			return nil, fmt.Errorf("transact error")
		},
	}
	s := NewStore(client, "table")
	err := s.Vote(context.Background(), "q1", "visitor1", "A")
	if err == nil {
		t.Fatal("expected error")
	}
}
