package poll

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// mockDynamoDBAPI は DynamoDBAPI のモック。
type mockDynamoDBAPI struct {
	putItemFn    func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	getItemFn    func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	deleteItemFn func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	updateItemFn func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	queryFn      func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

// PutItem はモックの PutItem を呼び出す。未設定時は明示エラーを返す。
func (m *mockDynamoDBAPI) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	if m.putItemFn == nil {
		return nil, fmt.Errorf("unexpected PutItem call: mock not configured")
	}
	return m.putItemFn(ctx, params, optFns...)
}

// GetItem はモックの GetItem を呼び出す。未設定時は明示エラーを返す。
func (m *mockDynamoDBAPI) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	if m.getItemFn == nil {
		return nil, fmt.Errorf("unexpected GetItem call: mock not configured")
	}
	return m.getItemFn(ctx, params, optFns...)
}

// DeleteItem はモックの DeleteItem を呼び出す。未設定時は明示エラーを返す。
func (m *mockDynamoDBAPI) DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	if m.deleteItemFn == nil {
		return nil, fmt.Errorf("unexpected DeleteItem call: mock not configured")
	}
	return m.deleteItemFn(ctx, params, optFns...)
}

// UpdateItem はモックの UpdateItem を呼び出す。未設定時は明示エラーを返す。
func (m *mockDynamoDBAPI) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	if m.updateItemFn == nil {
		return nil, fmt.Errorf("unexpected UpdateItem call: mock not configured")
	}
	return m.updateItemFn(ctx, params, optFns...)
}

// Query はモックの Query を呼び出す。未設定時は明示エラーを返す。
func (m *mockDynamoDBAPI) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	if m.queryFn == nil {
		return nil, fmt.Errorf("unexpected Query call: mock not configured")
	}
	return m.queryFn(ctx, params, optFns...)
}

// TestNewStore は Store の生成を検証する。
func TestNewStore(t *testing.T) {
	t.Parallel()
	client := &mockDynamoDBAPI{}
	s := NewStore(client, "test-table")
	if s.client != client {
		t.Error("client mismatch")
	}
	if s.tableName != "test-table" {
		t.Error("tableName mismatch")
	}
	if s.nowFn == nil {
		t.Error("nowFn should not be nil")
	}
	if s.marshalMapFn == nil {
		t.Error("marshalMapFn should not be nil")
	}
}

// TestNewStore_NilClient は client が nil の場合に panic することを検証する。
func TestNewStore_NilClient(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil client")
		}
	}()
	NewStore(nil, "table")
}

// TestNewStore_EmptyTableName は tableName が空の場合に panic することを検証する。
func TestNewStore_EmptyTableName(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty table name")
		}
	}()
	NewStore(&mockDynamoDBAPI{}, "")
}
