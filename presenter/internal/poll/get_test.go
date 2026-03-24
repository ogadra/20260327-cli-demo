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

// metaItem はテスト用の META アイテムを生成するヘルパー。
func metaItem(pollID string, options []string, maxChoices int, votes map[string]int) map[string]types.AttributeValue {
	rec := metaRecord{
		PollID:     pollID,
		ConnID:     metaSK,
		Options:    options,
		MaxChoices: maxChoices,
		Votes:      votes,
		TTL:        time.Now().Add(ttlDuration).Unix(),
	}
	item, _ := attributevalue.MarshalMap(rec)
	return item
}

// TestGet_PresenterInit は presenter が新規 poll を初期化することを検証する。
func TestGet_PresenterInit(t *testing.T) {
	t.Parallel()
	var putCalled bool
	meta := metaItem("q1", []string{"A", "B"}, 1, map[string]int{"A": 0, "B": 0})
	client := &mockDynamoDBAPI{
		putItemFn: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			putCalled = true
			return &dynamodb.PutItemOutput{}, nil
		},
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{}}, nil
		},
	}
	s := NewStore(client, "table")
	state, err := s.Get(context.Background(), "q1", "visitor1", []string{"A", "B"}, 1, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !putCalled {
		t.Error("expected PutItem to be called")
	}
	if state.PollID != "q1" {
		t.Errorf("expected q1, got %s", state.PollID)
	}
	if len(state.Options) != 2 {
		t.Errorf("expected 2 options, got %d", len(state.Options))
	}
}

// TestGet_PresenterInitAlreadyExists は既存 poll への初期化が冪等であることを検証する。
func TestGet_PresenterInitAlreadyExists(t *testing.T) {
	t.Parallel()
	meta := metaItem("q1", []string{"A", "B"}, 1, map[string]int{"A": 5, "B": 3})
	client := &mockDynamoDBAPI{
		putItemFn: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, &types.ConditionalCheckFailedException{Message: stringPtr("exists")}
		},
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{}}, nil
		},
	}
	s := NewStore(client, "table")
	state, err := s.Get(context.Background(), "q1", "", []string{"A", "B"}, 1, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Votes["A"] != 5 {
		t.Errorf("expected existing votes, got %v", state.Votes)
	}
}

// TestGet_PresenterNilOptions は presenter が options=nil の場合に初期化をスキップすることを検証する。
func TestGet_PresenterNilOptions(t *testing.T) {
	t.Parallel()
	meta := metaItem("q1", []string{"A"}, 1, map[string]int{"A": 0})
	client := &mockDynamoDBAPI{
		putItemFn: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			t.Error("PutItem should not be called")
			return nil, nil
		},
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{}}, nil
		},
	}
	s := NewStore(client, "table")
	_, err := s.Get(context.Background(), "q1", "", nil, 0, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestGet_ViewerSkipsInit は viewer が初期化をスキップすることを検証する。
func TestGet_ViewerSkipsInit(t *testing.T) {
	t.Parallel()
	meta := metaItem("q1", []string{"A"}, 1, map[string]int{"A": 0})
	client := &mockDynamoDBAPI{
		putItemFn: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			t.Error("PutItem should not be called for viewer")
			return nil, nil
		},
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{}}, nil
		},
	}
	s := NewStore(client, "table")
	_, err := s.Get(context.Background(), "q1", "visitor1", []string{"A"}, 1, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestGet_MarshalMetaError は META レコードのマーシャルエラーを検証する。
func TestGet_MarshalMetaError(t *testing.T) {
	t.Parallel()
	client := &mockDynamoDBAPI{}
	s := NewStore(client, "table")
	s.marshalMapFn = func(_ interface{}) (map[string]types.AttributeValue, error) {
		return nil, fmt.Errorf("marshal error")
	}
	_, err := s.Get(context.Background(), "q1", "", []string{"A"}, 1, true)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestGet_PutItemError は PutItem の非条件エラーを検証する。
func TestGet_PutItemError(t *testing.T) {
	t.Parallel()
	client := &mockDynamoDBAPI{
		putItemFn: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, fmt.Errorf("dynamo error")
		},
	}
	s := NewStore(client, "table")
	_, err := s.Get(context.Background(), "q1", "", []string{"A"}, 1, true)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestGet_GetMetaError は getMeta のエラーを検証する。
func TestGet_GetMetaError(t *testing.T) {
	t.Parallel()
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return nil, fmt.Errorf("get error")
		},
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{}}, nil
		},
	}
	s := NewStore(client, "table")
	_, err := s.Get(context.Background(), "q1", "", nil, 0, false)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestGet_PollNotFound は poll が存在しない場合のエラーを検証する。
func TestGet_PollNotFound(t *testing.T) {
	t.Parallel()
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
	}
	s := NewStore(client, "table")
	_, err := s.Get(context.Background(), "q1", "", nil, 0, false)
	if err != ErrPollNotFound {
		t.Errorf("expected ErrPollNotFound, got %v", err)
	}
}

// TestGet_UnmarshalMetaError は META レコードのアンマーシャルエラーを検証する。
func TestGet_UnmarshalMetaError(t *testing.T) {
	t.Parallel()
	badItem := map[string]types.AttributeValue{
		"pollId":       &types.AttributeValueMemberS{Value: "q1"},
		"connectionId": &types.AttributeValueMemberS{Value: metaSK},
		"maxChoices":   &types.AttributeValueMemberS{Value: "not-a-number"},
	}
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: badItem}, nil
		},
	}
	s := NewStore(client, "table")
	_, err := s.Get(context.Background(), "q1", "", nil, 0, false)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestGet_QueryMyChoicesError は myChoices クエリのエラーを検証する。
func TestGet_QueryMyChoicesError(t *testing.T) {
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
	_, err := s.Get(context.Background(), "q1", "visitor1", nil, 0, false)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestGet_WithMyChoices は myChoices が正しく取得されることを検証する。
func TestGet_WithMyChoices(t *testing.T) {
	t.Parallel()
	meta := metaItem("q1", []string{"A", "B", "C"}, 2, map[string]int{"A": 1, "B": 1, "C": 0})
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{"connectionId": &types.AttributeValueMemberS{Value: "visitor1#A"}},
					{"connectionId": &types.AttributeValueMemberS{Value: "visitor1#B"}},
				},
			}, nil
		},
	}
	s := NewStore(client, "table")
	state, err := s.Get(context.Background(), "q1", "visitor1", nil, 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(state.MyChoices) != 2 {
		t.Fatalf("expected 2 choices, got %d", len(state.MyChoices))
	}
	if state.MyChoices[0] != "A" || state.MyChoices[1] != "B" {
		t.Errorf("expected [A B], got %v", state.MyChoices)
	}
}

// TestGet_EmptyVisitorID は visitorID が空の場合に myChoices が空配列であることを検証する。
func TestGet_EmptyVisitorID(t *testing.T) {
	t.Parallel()
	meta := metaItem("q1", []string{"A"}, 1, map[string]int{"A": 0})
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
	}
	s := NewStore(client, "table")
	state, err := s.Get(context.Background(), "q1", "", nil, 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(state.MyChoices) != 0 {
		t.Errorf("expected empty myChoices, got %v", state.MyChoices)
	}
}

// TestGetMyChoices_MissingConnectionIdKey は connectionId キーが存在しないアイテムをスキップすることを検証する。
func TestGetMyChoices_MissingConnectionIdKey(t *testing.T) {
	t.Parallel()
	meta := metaItem("q1", []string{"A"}, 1, map[string]int{"A": 0})
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{"otherKey": &types.AttributeValueMemberS{Value: "value"}},
				},
			}, nil
		},
	}
	s := NewStore(client, "table")
	state, err := s.Get(context.Background(), "q1", "visitor1", nil, 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(state.MyChoices) != 0 {
		t.Errorf("expected empty myChoices, got %v", state.MyChoices)
	}
}

// TestGetMyChoices_NonStringConnectionId は connectionId が文字列型でないアイテムをスキップすることを検証する。
func TestGetMyChoices_NonStringConnectionId(t *testing.T) {
	t.Parallel()
	meta := metaItem("q1", []string{"A"}, 1, map[string]int{"A": 0})
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{"connectionId": &types.AttributeValueMemberN{Value: "123"}},
				},
			}, nil
		},
	}
	s := NewStore(client, "table")
	state, err := s.Get(context.Background(), "q1", "visitor1", nil, 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(state.MyChoices) != 0 {
		t.Errorf("expected empty myChoices, got %v", state.MyChoices)
	}
}

// TestGetMyChoices_NoHashInSK は # を含まないソートキーをスキップすることを検証する。
func TestGetMyChoices_NoHashInSK(t *testing.T) {
	t.Parallel()
	meta := metaItem("q1", []string{"A"}, 1, map[string]int{"A": 0})
	client := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: meta}, nil
		},
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{"connectionId": &types.AttributeValueMemberS{Value: "nohash"}},
				},
			}, nil
		},
	}
	s := NewStore(client, "table")
	state, err := s.Get(context.Background(), "q1", "visitor1", nil, 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(state.MyChoices) != 0 {
		t.Errorf("expected empty myChoices, got %v", state.MyChoices)
	}
}

// TestIsConditionalCheckFailed_True は ConditionalCheckFailedException が正しく判定されることを検証する。
func TestIsConditionalCheckFailed_True(t *testing.T) {
	t.Parallel()
	err := &types.ConditionalCheckFailedException{Message: stringPtr("test")}
	if !isConditionalCheckFailed(err) {
		t.Error("expected true")
	}
}

// TestIsConditionalCheckFailed_False は非 ConditionalCheckFailedException が false を返すことを検証する。
func TestIsConditionalCheckFailed_False(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf("other error")
	if isConditionalCheckFailed(err) {
		t.Error("expected false")
	}
}

// stringPtr は文字列ポインタを返すヘルパー。
func stringPtr(s string) *string {
	return &s
}
