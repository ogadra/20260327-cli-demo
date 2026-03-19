// Package store はリポジトリ層のテストを提供する。
package store

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/ogadra/20260327-cli-demo/broker/model"
)

// mockDynamoDBAPI は DynamoDBAPI のモック実装。
type mockDynamoDBAPI struct {
	putItemFn    func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	getItemFn    func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	updateItemFn func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
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

// UpdateItem はモック UpdateItem を呼び出す。
func (m *mockDynamoDBAPI) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	return m.updateItemFn(ctx, params, optFns...)
}

// DeleteItem はモック DeleteItem を呼び出す。
func (m *mockDynamoDBAPI) DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	return m.deleteItemFn(ctx, params, optFns...)
}

// Query はモック Query を呼び出す。
func (m *mockDynamoDBAPI) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	return m.queryFn(ctx, params, optFns...)
}

// TestNewDynamoRepository はコンストラクタの動作を検証する。
func TestNewDynamoRepository(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{}
	repo := NewDynamoRepository(mock, "test-table")
	if repo.client != mock {
		t.Error("client mismatch")
	}
	if repo.tableName != "test-table" {
		t.Errorf("tableName = %q, want %q", repo.tableName, "test-table")
	}
	if repo.bucketFn == nil {
		t.Error("bucketFn is nil")
	}
}

// TestDefaultBucketFn はデフォルトバケット関数がバケット範囲内の値を返すことを検証する。
func TestDefaultBucketFn(t *testing.T) {
	t.Parallel()
	seen := map[string]struct{}{}
	for range 1000 {
		b := defaultBucketFn()
		seen[b] = struct{}{}
	}
	for i := range bucketCount {
		key := "bucket-" + itoa(i)
		if _, ok := seen[key]; !ok {
			t.Errorf("bucket %q never seen in 1000 iterations", key)
		}
	}
}

// itoa は整数を文字列に変換するヘルパー。
func itoa(i int) string {
	return string(rune('0' + i))
}

// TestRegister_Success は新規登録の成功ケースを検証する。
func TestRegister_Success(t *testing.T) {
	t.Parallel()
	called := false
	mock := &mockDynamoDBAPI{
		putItemFn: func(_ context.Context, params *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			called = true
			if *params.ConditionExpression != "attribute_not_exists(runnerId)" {
				t.Errorf("unexpected condition: %s", *params.ConditionExpression)
			}
			return &dynamodb.PutItemOutput{}, nil
		},
	}
	repo := NewDynamoRepository(mock, "t")
	repo.bucketFn = func() string { return "bucket-0" }

	err := repo.Register(context.Background(), "r1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("PutItem was not called")
	}
}

// TestRegister_AlreadyExists は登録済み runner の再登録が冪等に成功することを検証する。
func TestRegister_AlreadyExists(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		putItemFn: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, &types.ConditionalCheckFailedException{Message: aws.String("exists")}
		},
	}
	repo := NewDynamoRepository(mock, "t")
	repo.bucketFn = func() string { return "bucket-0" }

	err := repo.Register(context.Background(), "r1")
	if err != nil {
		t.Fatalf("expected nil for idempotent register, got: %v", err)
	}
}

// TestRegister_PutItemError は PutItem の予期せぬエラーを検証する。
func TestRegister_PutItemError(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		putItemFn: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, errors.New("network error")
		},
	}
	repo := NewDynamoRepository(mock, "t")
	repo.bucketFn = func() string { return "bucket-0" }

	err := repo.Register(context.Background(), "r1")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestFindIdle_Success は idle runner が見つかるケースを検証する。
func TestFindIdle_Success(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, params *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{
						"runnerId":   &types.AttributeValueMemberS{Value: "r1"},
						"status":     &types.AttributeValueMemberS{Value: "idle"},
						"idleBucket": &types.AttributeValueMemberS{Value: "bucket-0"},
					},
				},
			}, nil
		},
	}
	repo := NewDynamoRepository(mock, "t")

	runner, err := repo.FindIdle(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner.RunnerID != "r1" {
		t.Errorf("runnerID = %q, want %q", runner.RunnerID, "r1")
	}
}

// TestFindIdle_NoIdleRunner は全バケット空の場合に ErrNoIdleRunner を返すことを検証する。
func TestFindIdle_NoIdleRunner(t *testing.T) {
	t.Parallel()
	queryCount := 0
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			queryCount++
			return &dynamodb.QueryOutput{Items: nil}, nil
		},
	}
	repo := NewDynamoRepository(mock, "t")

	_, err := repo.FindIdle(context.Background())
	if !errors.Is(err, ErrNoIdleRunner) {
		t.Fatalf("expected ErrNoIdleRunner, got: %v", err)
	}
	if queryCount != bucketCount {
		t.Errorf("query count = %d, want %d", queryCount, bucketCount)
	}
}

// TestFindIdle_QueryError は Query エラー時にエラーを返すことを検証する。
func TestFindIdle_QueryError(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return nil, errors.New("query error")
		},
	}
	repo := NewDynamoRepository(mock, "t")

	_, err := repo.FindIdle(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestFindIdle_FallbackBucket は最初のバケットが空で次のバケットで見つかるケースを検証する。
func TestFindIdle_FallbackBucket(t *testing.T) {
	t.Parallel()
	callCount := 0
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, params *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			callCount++
			if callCount == 1 {
				return &dynamodb.QueryOutput{Items: nil}, nil
			}
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{
						"runnerId":   &types.AttributeValueMemberS{Value: "r2"},
						"status":     &types.AttributeValueMemberS{Value: "idle"},
						"idleBucket": &types.AttributeValueMemberS{Value: "bucket-1"},
					},
				},
			}, nil
		},
	}
	repo := NewDynamoRepository(mock, "t")

	runner, err := repo.FindIdle(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner.RunnerID != "r2" {
		t.Errorf("runnerID = %q, want %q", runner.RunnerID, "r2")
	}
	if callCount != 2 {
		t.Errorf("query call count = %d, want 2", callCount)
	}
}

// TestAssignSession_Success は正常な session 割当を検証する。
func TestAssignSession_Success(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		updateItemFn: func(_ context.Context, params *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			if params.Key["runnerId"].(*types.AttributeValueMemberS).Value != "r1" {
				t.Errorf("unexpected runnerId")
			}
			return &dynamodb.UpdateItemOutput{}, nil
		},
	}
	repo := NewDynamoRepository(mock, "t")

	err := repo.AssignSession(context.Background(), "r1", "sess-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestAssignSession_ConditionFailed は既に busy の runner への割当が ErrConditionFailed を返すことを検証する。
func TestAssignSession_ConditionFailed(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		updateItemFn: func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			return nil, &types.ConditionalCheckFailedException{Message: aws.String("not idle")}
		},
	}
	repo := NewDynamoRepository(mock, "t")

	err := repo.AssignSession(context.Background(), "r1", "sess-1")
	if !errors.Is(err, ErrConditionFailed) {
		t.Fatalf("expected ErrConditionFailed, got: %v", err)
	}
}

// TestAssignSession_UpdateError は UpdateItem の予期せぬエラーを検証する。
func TestAssignSession_UpdateError(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		updateItemFn: func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			return nil, errors.New("update error")
		},
	}
	repo := NewDynamoRepository(mock, "t")

	err := repo.AssignSession(context.Background(), "r1", "sess-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestFindBySessionID_Success は session ID で runner が見つかるケースを検証する。
func TestFindBySessionID_Success(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, params *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			if *params.IndexName != "session-index" {
				t.Errorf("unexpected index: %s", *params.IndexName)
			}
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{
						"runnerId":         &types.AttributeValueMemberS{Value: "r1"},
						"status":           &types.AttributeValueMemberS{Value: "busy"},
						"currentSessionId": &types.AttributeValueMemberS{Value: "sess-1"},
					},
				},
			}, nil
		},
	}
	repo := NewDynamoRepository(mock, "t")

	runner, err := repo.FindBySessionID(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner.RunnerID != "r1" {
		t.Errorf("runnerID = %q, want %q", runner.RunnerID, "r1")
	}
}

// TestFindBySessionID_NotFound は session が見つからない場合に ErrNotFound を返すことを検証する。
func TestFindBySessionID_NotFound(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{Items: nil}, nil
		},
	}
	repo := NewDynamoRepository(mock, "t")

	_, err := repo.FindBySessionID(context.Background(), "sess-x")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

// TestFindBySessionID_QueryError は Query エラー時にエラーを返すことを検証する。
func TestFindBySessionID_QueryError(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return nil, errors.New("query error")
		},
	}
	repo := NewDynamoRepository(mock, "t")

	_, err := repo.FindBySessionID(context.Background(), "sess-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestFindByID_Success は runner ID で runner が見つかるケースを検証する。
func TestFindByID_Success(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, params *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"runnerId":   &types.AttributeValueMemberS{Value: "r1"},
					"status":     &types.AttributeValueMemberS{Value: "idle"},
					"idleBucket": &types.AttributeValueMemberS{Value: "bucket-0"},
				},
			}, nil
		},
	}
	repo := NewDynamoRepository(mock, "t")

	runner, err := repo.FindByID(context.Background(), "r1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner.RunnerID != "r1" {
		t.Errorf("runnerID = %q, want %q", runner.RunnerID, "r1")
	}
	if runner.Status != model.StatusIdle {
		t.Errorf("status = %q, want %q", runner.Status, model.StatusIdle)
	}
}

// TestFindByID_NotFound は runner が存在しない場合に ErrNotFound を返すことを検証する。
func TestFindByID_NotFound(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
	}
	repo := NewDynamoRepository(mock, "t")

	_, err := repo.FindByID(context.Background(), "r-missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

// TestFindByID_GetItemError は GetItem の予期せぬエラーを検証する。
func TestFindByID_GetItemError(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return nil, errors.New("get error")
		},
	}
	repo := NewDynamoRepository(mock, "t")

	_, err := repo.FindByID(context.Background(), "r1")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestDelete_Success は正常な削除を検証する。
func TestDelete_Success(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		deleteItemFn: func(_ context.Context, params *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			if params.Key["runnerId"].(*types.AttributeValueMemberS).Value != "r1" {
				t.Errorf("unexpected runnerId")
			}
			return &dynamodb.DeleteItemOutput{}, nil
		},
	}
	repo := NewDynamoRepository(mock, "t")

	err := repo.Delete(context.Background(), "r1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestDelete_Error は DeleteItem の予期せぬエラーを検証する。
func TestDelete_Error(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		deleteItemFn: func(_ context.Context, _ *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			return nil, errors.New("delete error")
		},
	}
	repo := NewDynamoRepository(mock, "t")

	err := repo.Delete(context.Background(), "r1")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestFindIdle_UnmarshalError は Query 結果の unmarshal 失敗を検証する。
func TestFindIdle_UnmarshalError(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{
						"runnerId": &types.AttributeValueMemberN{Value: "123"},
						"status":   &types.AttributeValueMemberBOOL{Value: true},
					},
				},
			}, nil
		},
	}
	repo := NewDynamoRepository(mock, "t")

	_, err := repo.FindIdle(context.Background())
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
}

// TestFindBySessionID_UnmarshalError は Query 結果の unmarshal 失敗を検証する。
func TestFindBySessionID_UnmarshalError(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{
						"runnerId": &types.AttributeValueMemberN{Value: "123"},
						"status":   &types.AttributeValueMemberBOOL{Value: true},
					},
				},
			}, nil
		},
	}
	repo := NewDynamoRepository(mock, "t")

	_, err := repo.FindBySessionID(context.Background(), "sess-1")
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
}

// TestFindByID_UnmarshalError は GetItem 結果の unmarshal 失敗を検証する。
func TestFindByID_UnmarshalError(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"runnerId": &types.AttributeValueMemberN{Value: "123"},
					"status":   &types.AttributeValueMemberBOOL{Value: true},
				},
			}, nil
		},
	}
	repo := NewDynamoRepository(mock, "t")

	_, err := repo.FindByID(context.Background(), "r1")
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
}

// TestDynamoRepository_ImplementsRepository は DynamoRepository が Repository インターフェースを満たすことを検証する。
func TestDynamoRepository_ImplementsRepository(t *testing.T) {
	t.Parallel()
	var _ Repository = (*DynamoRepository)(nil)
}
