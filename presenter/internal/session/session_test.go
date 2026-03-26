// Package session はセッション管理のテストを提供する。
package session

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
}

// PutItem はモック PutItem を呼び出す。
func (m *mockDynamoDBAPI) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return m.putItemFn(ctx, params, optFns...)
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

// TestCreate_Success はセッション作成が成功するケースを検証する。
func TestCreate_Success(t *testing.T) {
	t.Parallel()
	var capturedItem map[string]types.AttributeValue
	var capturedTableName string
	mock := &mockDynamoDBAPI{
		putItemFn: func(_ context.Context, params *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			capturedItem = params.Item
			capturedTableName = *params.TableName
			return &dynamodb.PutItemOutput{}, nil
		},
	}
	store := NewStore(mock, "sessions")

	err := store.Create(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedTableName != "sessions" {
		t.Errorf("tableName = %q, want %q", capturedTableName, "sessions")
	}
	tokenAttr, ok := capturedItem["token"].(*types.AttributeValueMemberS)
	if !ok {
		t.Fatalf("token attribute type = %T, want *types.AttributeValueMemberS", capturedItem["token"])
	}
	if tokenAttr.Value != "abc123" {
		t.Errorf("token = %q, want %q", tokenAttr.Value, "abc123")
	}
	statusAttr, ok := capturedItem["status"].(*types.AttributeValueMemberS)
	if !ok {
		t.Fatalf("status attribute type = %T, want *types.AttributeValueMemberS", capturedItem["status"])
	}
	if statusAttr.Value != "valid" {
		t.Errorf("status = %q, want %q", statusAttr.Value, "valid")
	}
	if _, hasTTL := capturedItem["ttl"]; hasTTL {
		t.Error("session should not have TTL attribute")
	}
}

// TestCreate_MarshalError は MarshalMap がエラーを返す場合にエラーを返すことを検証する。
func TestCreate_MarshalError(t *testing.T) {
	t.Parallel()
	store := NewStore(&mockDynamoDBAPI{}, "t")
	store.marshalFn = func(_ interface{}) (map[string]types.AttributeValue, error) {
		return nil, errors.New("marshal error")
	}

	err := store.Create(context.Background(), "abc123")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestCreate_PutItemError は PutItem のエラーを検証する。
func TestCreate_PutItemError(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		putItemFn: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, errors.New("network error")
		},
	}
	store := NewStore(mock, "t")

	err := store.Create(context.Background(), "abc123")
	if err == nil {
		t.Fatal("expected error")
	}
}
