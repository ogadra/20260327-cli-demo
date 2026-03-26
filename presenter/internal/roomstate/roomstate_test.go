// Package roomstate は room 状態管理のテストを提供する。
package roomstate

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// mockDynamoDBAPI は DynamoDBAPI のモック実装。
type mockDynamoDBAPI struct {
	putItemFn func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	getItemFn func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
}

// PutItem はモック PutItem を呼び出す。
func (m *mockDynamoDBAPI) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return m.putItemFn(ctx, params, optFns...)
}

// GetItem はモック GetItem を呼び出す。
func (m *mockDynamoDBAPI) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return m.getItemFn(ctx, params, optFns...)
}

// TestNewStore はコンストラクタの動作を検証する。
func TestNewStore(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{}
	store := NewStore(mock, "test-table")
	if store.client != mock {
		t.Error("client mismatch")
	}
	if store.tableName != "test-table" {
		t.Errorf("tableName = %q, want %q", store.tableName, "test-table")
	}
	if store.marshalFn == nil {
		t.Error("marshalFn is nil")
	}
}

// TestPutState_Success は room 状態の保存が成功するケースを検証する。
func TestPutState_Success(t *testing.T) {
	t.Parallel()
	var capturedItem map[string]types.AttributeValue
	mock := &mockDynamoDBAPI{
		putItemFn: func(_ context.Context, params *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			capturedItem = params.Item
			return &dynamodb.PutItemOutput{}, nil
		},
	}
	store := NewStore(mock, "t")

	err := store.PutState(context.Background(), "default", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	room := capturedItem["room"].(*types.AttributeValueMemberS).Value
	if room != "default" {
		t.Errorf("room = %q, want default", room)
	}
	page := capturedItem["page"].(*types.AttributeValueMemberN).Value
	if page != "5" {
		t.Errorf("page = %q, want 5", page)
	}
}

// TestPutState_MarshalError は marshal 失敗を検証する。
func TestPutState_MarshalError(t *testing.T) {
	t.Parallel()
	store := NewStore(&mockDynamoDBAPI{}, "t")
	store.marshalFn = func(_ interface{}) (map[string]types.AttributeValue, error) {
		return nil, errors.New("marshal error")
	}

	err := store.PutState(context.Background(), "default", 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestPutState_PutItemError は PutItem のエラーを検証する。
func TestPutState_PutItemError(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		putItemFn: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, errors.New("put error")
		},
	}
	store := NewStore(mock, "t")

	err := store.PutState(context.Background(), "default", 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestGetState_Success は room 状態の取得が成功するケースを検証する。
func TestGetState_Success(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, params *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			room := params.Key["room"].(*types.AttributeValueMemberS).Value
			if room != "default" {
				t.Errorf("room = %q, want default", room)
			}
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"room": &types.AttributeValueMemberS{Value: "default"},
					"page": &types.AttributeValueMemberN{Value: "3"},
				},
			}, nil
		},
	}
	store := NewStore(mock, "t")

	page, err := store.GetState(context.Background(), "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page != 3 {
		t.Errorf("page = %d, want 3", page)
	}
}

// TestGetState_NotFound は状態未保存時に 0 を返すことを検証する。
func TestGetState_NotFound(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
	}
	store := NewStore(mock, "t")

	page, err := store.GetState(context.Background(), "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page != 0 {
		t.Errorf("page = %d, want 0", page)
	}
}

// TestGetState_MissingPageAttribute は page 属性がない場合に 0 を返すことを検証する。
func TestGetState_MissingPageAttribute(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"room": &types.AttributeValueMemberS{Value: "default"},
				},
			}, nil
		},
	}
	store := NewStore(mock, "t")

	page, err := store.GetState(context.Background(), "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page != 0 {
		t.Errorf("page = %d, want 0", page)
	}
}

// TestGetState_NonNumberPage は page が数値型でない場合に 0 を返すことを検証する。
func TestGetState_NonNumberPage(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"room": &types.AttributeValueMemberS{Value: "default"},
					"page": &types.AttributeValueMemberS{Value: "not-a-number"},
				},
			}, nil
		},
	}
	store := NewStore(mock, "t")

	page, err := store.GetState(context.Background(), "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page != 0 {
		t.Errorf("page = %d, want 0", page)
	}
}

// TestGetState_InvalidNumber は page の数値パースが失敗する場合を検証する。
func TestGetState_InvalidNumber(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"room": &types.AttributeValueMemberS{Value: "default"},
					"page": &types.AttributeValueMemberN{Value: "abc"},
				},
			}, nil
		},
	}
	store := NewStore(mock, "t")

	_, err := store.GetState(context.Background(), "default")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestGetState_GetItemError は GetItem のエラーを検証する。
func TestGetState_GetItemError(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return nil, errors.New("get error")
		},
	}
	store := NewStore(mock, "t")

	_, err := store.GetState(context.Background(), "default")
	if err == nil {
		t.Fatal("expected error")
	}
}
