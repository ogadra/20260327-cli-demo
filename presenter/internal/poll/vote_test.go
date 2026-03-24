package poll

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// TestVote_Success は正常な投票を検証する。
func TestVote_Success(t *testing.T) {
	t.Parallel()
	meta := metaItem("q1", []string{"A", "B"}, 2, map[string]int{"A": 0, "B": 0})
	var putSK string
	var updateCalled bool
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{}}, nil
		},
		putItemFn: func(_ context.Context, params *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			sk := params.Item["connectionId"].(*types.AttributeValueMemberS).Value
			putSK = sk
			return &dynamodb.PutItemOutput{}, nil
		},
		updateItemFn: func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			updateCalled = true
			return &dynamodb.UpdateItemOutput{}, nil
		},
	}
	s := NewStore(client, "table")
	err := s.Vote(context.Background(), "q1", "visitor1", "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if putSK != "visitor1#A" {
		t.Errorf("expected visitor1#A, got %s", putSK)
	}
	if !updateCalled {
		t.Error("expected UpdateItem to be called")
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
		putItemFn: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, &types.ConditionalCheckFailedException{Message: stringPtr("exists")}
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

// TestVote_PutItemError は PutItem の非条件エラーを検証する。
func TestVote_PutItemError(t *testing.T) {
	t.Parallel()
	meta := metaItem("q1", []string{"A"}, 1, map[string]int{"A": 0})
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{}}, nil
		},
		putItemFn: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, fmt.Errorf("put error")
		},
	}
	s := NewStore(client, "table")
	err := s.Vote(context.Background(), "q1", "visitor1", "A")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestVote_UpdateItemError は UpdateItem エラーを検証する。
func TestVote_UpdateItemError(t *testing.T) {
	t.Parallel()
	meta := metaItem("q1", []string{"A"}, 1, map[string]int{"A": 0})
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{}}, nil
		},
		putItemFn: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return &dynamodb.PutItemOutput{}, nil
		},
		updateItemFn: func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			return nil, fmt.Errorf("update error")
		},
	}
	s := NewStore(client, "table")
	err := s.Vote(context.Background(), "q1", "visitor1", "A")
	if err == nil {
		t.Fatal("expected error")
	}
}

// metaItem はテスト用の META アイテムを生成するヘルパー。vote_test 用の再定義を避けるため get_test.go に配置済み。
// ここでは get_test.go の metaItem を使用する。

// 以下は vote_test.go で未使用の import を回避するために attributevalue を参照する。
var _ = attributevalue.MarshalMap
var _ = time.Now
