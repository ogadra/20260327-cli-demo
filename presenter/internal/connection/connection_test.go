// Package connection はセッション・接続管理のテストを提供する。
package connection

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// mockDynamoDBAPI は DynamoDBAPI のモック実装。
type mockDynamoDBAPI struct {
	putItemFn    func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	getItemFn    func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	deleteItemFn func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	queryFn      func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

// PutItem はモック PutItem を呼び出す。
func (m *mockDynamoDBAPI) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return m.putItemFn(ctx, params, optFns...)
}

// GetItem はモック GetItem を呼び出す。
func (m *mockDynamoDBAPI) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return m.getItemFn(ctx, params, optFns...)
}

// DeleteItem はモック DeleteItem を呼び出す。
func (m *mockDynamoDBAPI) DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	return m.deleteItemFn(ctx, params, optFns...)
}

// Query はモック Query を呼び出す。
func (m *mockDynamoDBAPI) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	return m.queryFn(ctx, params, optFns...)
}

// mockSessionDynamoDBAPI は SessionDynamoDBAPI のモック実装。
type mockSessionDynamoDBAPI struct {
	getItemFn func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
}

// GetItem はモック GetItem を呼び出す。
func (m *mockSessionDynamoDBAPI) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
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
	if store.nowFn == nil {
		t.Error("nowFn is nil")
	}
	if store.marshalFn == nil {
		t.Error("marshalFn is nil")
	}
}

// TestPut_Success は接続情報の保存が成功するケースを検証する。
func TestPut_Success(t *testing.T) {
	t.Parallel()
	var capturedItem map[string]types.AttributeValue
	mock := &mockDynamoDBAPI{
		putItemFn: func(_ context.Context, params *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			capturedItem = params.Item
			return &dynamodb.PutItemOutput{}, nil
		},
	}
	store := NewStore(mock, "t")
	fixedTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	store.nowFn = func() time.Time { return fixedTime }

	err := store.Put(context.Background(), Connection{
		Room:         "default",
		ConnectionID: "conn-1",
		Role:         "viewer",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ttlRaw, ok := capturedItem["ttl"]
	if !ok {
		t.Fatal("ttl attribute is missing")
	}
	ttlVal, ok := ttlRaw.(*types.AttributeValueMemberN)
	if !ok {
		t.Fatalf("ttl attribute type = %T, want *types.AttributeValueMemberN", ttlRaw)
	}
	expectedTTL := fixedTime.Add(24 * time.Hour).Unix()
	if ttlVal.Value != fmt.Sprintf("%d", expectedTTL) {
		t.Errorf("ttl = %s, want %d", ttlVal.Value, expectedTTL)
	}
}

// TestPut_MarshalError は MarshalMap がエラーを返す場合にエラーを返すことを検証する。
func TestPut_MarshalError(t *testing.T) {
	t.Parallel()
	store := NewStore(&mockDynamoDBAPI{}, "t")
	store.marshalFn = func(_ interface{}) (map[string]types.AttributeValue, error) {
		return nil, errors.New("marshal error")
	}

	err := store.Put(context.Background(), Connection{Room: "default", ConnectionID: "conn-1", Role: "viewer"})
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestPut_PutItemError は PutItem のエラーを検証する。
func TestPut_PutItemError(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		putItemFn: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, errors.New("network error")
		},
	}
	store := NewStore(mock, "t")

	err := store.Put(context.Background(), Connection{Room: "default", ConnectionID: "conn-1", Role: "viewer"})
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestGet_Success は接続情報の取得が成功するケースを検証する。
func TestGet_Success(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, params *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			room := params.Key["room"].(*types.AttributeValueMemberS).Value
			connID := params.Key["connectionId"].(*types.AttributeValueMemberS).Value
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"room":         &types.AttributeValueMemberS{Value: room},
					"connectionId": &types.AttributeValueMemberS{Value: connID},
					"role":         &types.AttributeValueMemberS{Value: "presenter"},
					"ttl":          &types.AttributeValueMemberN{Value: "9999999999"},
				},
			}, nil
		},
	}
	store := NewStore(mock, "t")

	conn, err := store.Get(context.Background(), "default", "conn-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn.Role != "presenter" {
		t.Errorf("role = %q, want %q", conn.Role, "presenter")
	}
}

// TestGet_NotFound は接続が存在しない場合に ErrNotFound を返すことを検証する。
func TestGet_NotFound(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
	}
	store := NewStore(mock, "t")

	_, err := store.Get(context.Background(), "default", "conn-x")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

// TestGet_GetItemError は GetItem のエラーを検証する。
func TestGet_GetItemError(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return nil, errors.New("get error")
		},
	}
	store := NewStore(mock, "t")

	_, err := store.Get(context.Background(), "default", "conn-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestGet_UnmarshalError は GetItem 結果の unmarshal 失敗を検証する。
func TestGet_UnmarshalError(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"room": &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
				},
			}, nil
		},
	}
	store := NewStore(mock, "t")

	_, err := store.Get(context.Background(), "default", "conn-1")
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
}

// TestDelete_Success は接続情報の削除が成功するケースを検証する。
func TestDelete_Success(t *testing.T) {
	t.Parallel()
	var called bool
	mock := &mockDynamoDBAPI{
		deleteItemFn: func(_ context.Context, params *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			called = true
			room := params.Key["room"].(*types.AttributeValueMemberS).Value
			connID := params.Key["connectionId"].(*types.AttributeValueMemberS).Value
			if room != "default" || connID != "conn-1" {
				t.Errorf("unexpected key: room=%q, connectionId=%q", room, connID)
			}
			return &dynamodb.DeleteItemOutput{}, nil
		},
	}
	store := NewStore(mock, "t")

	err := store.Delete(context.Background(), "default", "conn-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("DeleteItem was not called")
	}
}

// TestDelete_Error は DeleteItem のエラーを検証する。
func TestDelete_Error(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		deleteItemFn: func(_ context.Context, _ *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			return nil, errors.New("delete error")
		},
	}
	store := NewStore(mock, "t")

	err := store.Delete(context.Background(), "default", "conn-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestQueryByRoom_Success は room 内の全接続を取得するケースを検証する。
func TestQueryByRoom_Success(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{
						"room":         &types.AttributeValueMemberS{Value: "default"},
						"connectionId": &types.AttributeValueMemberS{Value: "conn-1"},
						"role":         &types.AttributeValueMemberS{Value: "viewer"},
						"ttl":          &types.AttributeValueMemberN{Value: "9999999999"},
					},
					{
						"room":         &types.AttributeValueMemberS{Value: "default"},
						"connectionId": &types.AttributeValueMemberS{Value: "conn-2"},
						"role":         &types.AttributeValueMemberS{Value: "presenter"},
						"ttl":          &types.AttributeValueMemberN{Value: "9999999999"},
					},
				},
			}, nil
		},
	}
	store := NewStore(mock, "t")

	conns, err := store.QueryByRoom(context.Background(), "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 2 {
		t.Fatalf("len = %d, want 2", len(conns))
	}
	if conns[0].ConnectionID != "conn-1" || conns[1].ConnectionID != "conn-2" {
		t.Errorf("unexpected connections: %+v", conns)
	}
}

// TestQueryByRoom_Empty は room に接続がない場合を検証する。
func TestQueryByRoom_Empty(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{Items: nil}, nil
		},
	}
	store := NewStore(mock, "t")

	conns, err := store.QueryByRoom(context.Background(), "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 0 {
		t.Errorf("len = %d, want 0", len(conns))
	}
}

// TestQueryByRoom_QueryError は Query のエラーを検証する。
func TestQueryByRoom_QueryError(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return nil, errors.New("query error")
		},
	}
	store := NewStore(mock, "t")

	_, err := store.QueryByRoom(context.Background(), "default")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestQueryByRoom_Pagination は複数ページにまたがる Query 結果を全件取得できることを検証する。
func TestQueryByRoom_Pagination(t *testing.T) {
	t.Parallel()
	callCount := 0
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, params *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			callCount++
			if callCount == 1 {
				return &dynamodb.QueryOutput{
					Items: []map[string]types.AttributeValue{
						{
							"room":         &types.AttributeValueMemberS{Value: "default"},
							"connectionId": &types.AttributeValueMemberS{Value: "conn-1"},
							"role":         &types.AttributeValueMemberS{Value: "viewer"},
							"ttl":          &types.AttributeValueMemberN{Value: "9999999999"},
						},
					},
					LastEvaluatedKey: map[string]types.AttributeValue{
						"room":         &types.AttributeValueMemberS{Value: "default"},
						"connectionId": &types.AttributeValueMemberS{Value: "conn-1"},
					},
				}, nil
			}
			if params.ExclusiveStartKey == nil {
				t.Error("expected ExclusiveStartKey on second call")
			}
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{
						"room":         &types.AttributeValueMemberS{Value: "default"},
						"connectionId": &types.AttributeValueMemberS{Value: "conn-2"},
						"role":         &types.AttributeValueMemberS{Value: "presenter"},
						"ttl":          &types.AttributeValueMemberN{Value: "9999999999"},
					},
				},
			}, nil
		},
	}
	store := NewStore(mock, "t")

	conns, err := store.QueryByRoom(context.Background(), "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 2 {
		t.Fatalf("len = %d, want 2", len(conns))
	}
	if conns[0].ConnectionID != "conn-1" || conns[1].ConnectionID != "conn-2" {
		t.Errorf("unexpected connections: %+v", conns)
	}
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2", callCount)
	}
}

// TestCountByRoom_Pagination は複数ページにまたがる Count を累積できることを検証する。
func TestCountByRoom_Pagination(t *testing.T) {
	t.Parallel()
	callCount := 0
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, params *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			callCount++
			if callCount == 1 {
				return &dynamodb.QueryOutput{
					Count: 30,
					LastEvaluatedKey: map[string]types.AttributeValue{
						"room":         &types.AttributeValueMemberS{Value: "default"},
						"connectionId": &types.AttributeValueMemberS{Value: "conn-30"},
					},
				}, nil
			}
			if params.ExclusiveStartKey == nil {
				t.Error("expected ExclusiveStartKey on second call")
			}
			return &dynamodb.QueryOutput{Count: 12}, nil
		},
	}
	store := NewStore(mock, "t")

	count, err := store.CountByRoom(context.Background(), "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 42 {
		t.Errorf("count = %d, want 42", count)
	}
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2", callCount)
	}
}

// TestQueryByRoom_PaginationQueryError は2ページ目の Query がエラーを返す場合を検証する。
func TestQueryByRoom_PaginationQueryError(t *testing.T) {
	t.Parallel()
	callCount := 0
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			callCount++
			if callCount == 1 {
				return &dynamodb.QueryOutput{
					Items: []map[string]types.AttributeValue{
						{
							"room":         &types.AttributeValueMemberS{Value: "default"},
							"connectionId": &types.AttributeValueMemberS{Value: "conn-1"},
							"role":         &types.AttributeValueMemberS{Value: "viewer"},
							"ttl":          &types.AttributeValueMemberN{Value: "9999999999"},
						},
					},
					LastEvaluatedKey: map[string]types.AttributeValue{
						"room": &types.AttributeValueMemberS{Value: "default"},
					},
				}, nil
			}
			return nil, errors.New("page 2 error")
		},
	}
	store := NewStore(mock, "t")

	_, err := store.QueryByRoom(context.Background(), "default")
	if err == nil {
		t.Fatal("expected error on second page")
	}
}

// TestQueryByRoom_PaginationUnmarshalError は2ページ目の unmarshal がエラーを返す場合を検証する。
func TestQueryByRoom_PaginationUnmarshalError(t *testing.T) {
	t.Parallel()
	callCount := 0
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			callCount++
			if callCount == 1 {
				return &dynamodb.QueryOutput{
					Items: []map[string]types.AttributeValue{
						{
							"room":         &types.AttributeValueMemberS{Value: "default"},
							"connectionId": &types.AttributeValueMemberS{Value: "conn-1"},
							"role":         &types.AttributeValueMemberS{Value: "viewer"},
							"ttl":          &types.AttributeValueMemberN{Value: "9999999999"},
						},
					},
					LastEvaluatedKey: map[string]types.AttributeValue{
						"room": &types.AttributeValueMemberS{Value: "default"},
					},
				}, nil
			}
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{"room": &types.AttributeValueMemberL{Value: []types.AttributeValue{}}},
				},
			}, nil
		},
	}
	store := NewStore(mock, "t")

	_, err := store.QueryByRoom(context.Background(), "default")
	if err == nil {
		t.Fatal("expected unmarshal error on second page")
	}
}

// TestCountByRoom_PaginationQueryError は2ページ目の Count Query がエラーを返す場合を検証する。
func TestCountByRoom_PaginationQueryError(t *testing.T) {
	t.Parallel()
	callCount := 0
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			callCount++
			if callCount == 1 {
				return &dynamodb.QueryOutput{
					Count: 10,
					LastEvaluatedKey: map[string]types.AttributeValue{
						"room": &types.AttributeValueMemberS{Value: "default"},
					},
				}, nil
			}
			return nil, errors.New("page 2 count error")
		},
	}
	store := NewStore(mock, "t")

	_, err := store.CountByRoom(context.Background(), "default")
	if err == nil {
		t.Fatal("expected error on second page")
	}
}

// TestQueryByRoom_UnmarshalError は Query 結果の unmarshal 失敗を検証する。
func TestQueryByRoom_UnmarshalError(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{"room": &types.AttributeValueMemberL{Value: []types.AttributeValue{}}},
				},
			}, nil
		},
	}
	store := NewStore(mock, "t")

	_, err := store.QueryByRoom(context.Background(), "default")
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
}

// TestCountByRoom_Success は room 内の接続数を取得するケースを検証する。
func TestCountByRoom_Success(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, params *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			if params.Select != types.SelectCount {
				t.Errorf("select = %v, want Count", params.Select)
			}
			return &dynamodb.QueryOutput{Count: 42}, nil
		},
	}
	store := NewStore(mock, "t")

	count, err := store.CountByRoom(context.Background(), "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 42 {
		t.Errorf("count = %d, want 42", count)
	}
}

// TestCountByRoom_Error は Query のエラーを検証する。
func TestCountByRoom_Error(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return nil, errors.New("count error")
		},
	}
	store := NewStore(mock, "t")

	_, err := store.CountByRoom(context.Background(), "default")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestNewSessionStore はコンストラクタの動作を検証する。
func TestNewSessionStore(t *testing.T) {
	t.Parallel()
	mock := &mockSessionDynamoDBAPI{}
	ss := NewSessionStore(mock, "sessions")
	if ss.client != mock {
		t.Error("client mismatch")
	}
	if ss.tableName != "sessions" {
		t.Errorf("tableName = %q, want %q", ss.tableName, "sessions")
	}
}

// TestIsValid_ValidToken は有効なトークンで true を返すことを検証する。
func TestIsValid_ValidToken(t *testing.T) {
	t.Parallel()
	mock := &mockSessionDynamoDBAPI{
		getItemFn: func(_ context.Context, in *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			if in.TableName == nil || *in.TableName != "sessions" {
				t.Fatalf("tableName = %v, want sessions", in.TableName)
			}
			tokenAttr, ok := in.Key["token"].(*types.AttributeValueMemberS)
			if !ok || tokenAttr.Value != "abc123" {
				t.Fatalf("unexpected token key: %#v", in.Key["token"])
			}
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"token":  &types.AttributeValueMemberS{Value: "abc123"},
					"status": &types.AttributeValueMemberS{Value: "valid"},
				},
			}, nil
		},
	}
	ss := NewSessionStore(mock, "sessions")

	valid, err := ss.IsValid(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Error("expected valid")
	}
}

// TestIsValid_EmptyToken は空トークンで false を返すことを検証する。
func TestIsValid_EmptyToken(t *testing.T) {
	t.Parallel()
	ss := NewSessionStore(&mockSessionDynamoDBAPI{}, "sessions")

	valid, err := ss.IsValid(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected invalid for empty token")
	}
}

// TestIsValid_NotFound はトークンが存在しない場合に false を返すことを検証する。
func TestIsValid_NotFound(t *testing.T) {
	t.Parallel()
	mock := &mockSessionDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
	}
	ss := NewSessionStore(mock, "sessions")

	valid, err := ss.IsValid(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected invalid for nonexistent token")
	}
}

// TestIsValid_InvalidStatus は status が "valid" でない場合に false を返すことを検証する。
func TestIsValid_InvalidStatus(t *testing.T) {
	t.Parallel()
	mock := &mockSessionDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"token":  &types.AttributeValueMemberS{Value: "abc123"},
					"status": &types.AttributeValueMemberS{Value: "expired"},
				},
			}, nil
		},
	}
	ss := NewSessionStore(mock, "sessions")

	valid, err := ss.IsValid(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected invalid for expired status")
	}
}

// TestIsValid_MissingStatusAttribute は status 属性がない場合に false を返すことを検証する。
func TestIsValid_MissingStatusAttribute(t *testing.T) {
	t.Parallel()
	mock := &mockSessionDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"token": &types.AttributeValueMemberS{Value: "abc123"},
				},
			}, nil
		},
	}
	ss := NewSessionStore(mock, "sessions")

	valid, err := ss.IsValid(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected invalid for missing status")
	}
}

// TestIsValid_NonStringStatus は status が文字列型でない場合に false を返すことを検証する。
func TestIsValid_NonStringStatus(t *testing.T) {
	t.Parallel()
	mock := &mockSessionDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"token":  &types.AttributeValueMemberS{Value: "abc123"},
					"status": &types.AttributeValueMemberN{Value: "1"},
				},
			}, nil
		},
	}
	ss := NewSessionStore(mock, "sessions")

	valid, err := ss.IsValid(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected invalid for non-string status")
	}
}

// TestIsValid_GetItemError は GetItem のエラーを検証する。
func TestIsValid_GetItemError(t *testing.T) {
	t.Parallel()
	mock := &mockSessionDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return nil, errors.New("dynamo error")
		},
	}
	ss := NewSessionStore(mock, "sessions")

	_, err := ss.IsValid(context.Background(), "abc123")
	if err == nil {
		t.Fatal("expected error")
	}
}
