package poll

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// mockDynamoDBAPI は DynamoDBAPI のモック。
type mockDynamoDBAPI struct {
	putItemFn            func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	getItemFn            func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	deleteItemFn         func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	updateItemFn         func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	queryFn              func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	transactWriteItemsFn func(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
}

// PutItem はモックの PutItem を呼び出す。
func (m *mockDynamoDBAPI) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return m.putItemFn(ctx, params, optFns...)
}

// GetItem はモックの GetItem を呼び出す。
func (m *mockDynamoDBAPI) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return m.getItemFn(ctx, params, optFns...)
}

// DeleteItem はモックの DeleteItem を呼び出す。
func (m *mockDynamoDBAPI) DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	return m.deleteItemFn(ctx, params, optFns...)
}

// UpdateItem はモックの UpdateItem を呼び出す。
func (m *mockDynamoDBAPI) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	return m.updateItemFn(ctx, params, optFns...)
}

// Query はモックの Query を呼び出す。
func (m *mockDynamoDBAPI) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	return m.queryFn(ctx, params, optFns...)
}

// TransactWriteItems はモックの TransactWriteItems を呼び出す。
func (m *mockDynamoDBAPI) TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	return m.transactWriteItemsFn(ctx, params, optFns...)
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
