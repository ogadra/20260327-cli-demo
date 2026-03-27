// Package store はリポジトリ層のテストを提供する。
package store

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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
	if repo.marshalFn == nil {
		t.Error("marshalFn is nil")
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

// TestRegister_MarshalError は MarshalMap がエラーを返す場合にエラーを返すことを検証する。
func TestRegister_MarshalError(t *testing.T) {
	t.Parallel()
	repo := NewDynamoRepository(&mockDynamoDBAPI{}, "t")
	repo.bucketFn = func() string { return "bucket-0" }
	repo.marshalFn = func(_ interface{}) (map[string]types.AttributeValue, error) {
		return nil, errors.New("marshal error")
	}

	err := repo.Register(context.Background(), "r1", "http://10.0.0.1:8080")
	if err == nil {
		t.Fatal("expected error")
	}
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
			v, ok := params.Item["privateUrl"].(*types.AttributeValueMemberS)
			if !ok || v.Value != "http://10.0.0.1:8080" {
				t.Errorf("privateUrl was not marshaled correctly")
			}
			return &dynamodb.PutItemOutput{}, nil
		},
	}
	repo := NewDynamoRepository(mock, "t")
	repo.bucketFn = func() string { return "bucket-0" }

	err := repo.Register(context.Background(), "r1", "http://10.0.0.1:8080")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("PutItem was not called")
	}
}

// TestRegister_AlreadyExists は登録済み runner の再登録が同一 privateURL なら冪等に成功することを検証する。
func TestRegister_AlreadyExists(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		putItemFn: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, &types.ConditionalCheckFailedException{Message: aws.String("exists")}
		},
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"runnerId":   &types.AttributeValueMemberS{Value: "r1"},
					"privateUrl": &types.AttributeValueMemberS{Value: "http://10.0.0.1:8080"},
					"idleBucket": &types.AttributeValueMemberS{Value: "bucket-0"},
				},
			}, nil
		},
	}
	repo := NewDynamoRepository(mock, "t")
	repo.bucketFn = func() string { return "bucket-0" }

	err := repo.Register(context.Background(), "r1", "http://10.0.0.1:8080")
	if err != nil {
		t.Fatalf("expected nil for idempotent register, got: %v", err)
	}
}

// TestRegister_ConflictPrivateURL は同一 runnerID で異なる privateURL の登録が ErrConflict を返すことを検証する。
func TestRegister_ConflictPrivateURL(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		putItemFn: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, &types.ConditionalCheckFailedException{Message: aws.String("exists")}
		},
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"runnerId":   &types.AttributeValueMemberS{Value: "r1"},
					"privateUrl": &types.AttributeValueMemberS{Value: "http://10.0.0.1:8080"},
					"idleBucket": &types.AttributeValueMemberS{Value: "bucket-0"},
				},
			}, nil
		},
	}
	repo := NewDynamoRepository(mock, "t")
	repo.bucketFn = func() string { return "bucket-0" }

	err := repo.Register(context.Background(), "r1", "http://10.0.0.2:9090")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got: %v", err)
	}
}

// TestRegister_ConflictFindByIDError は条件失敗後の FindByID エラーを検証する。
func TestRegister_ConflictFindByIDError(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		putItemFn: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, &types.ConditionalCheckFailedException{Message: aws.String("exists")}
		},
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return nil, errors.New("get error")
		},
	}
	repo := NewDynamoRepository(mock, "t")
	repo.bucketFn = func() string { return "bucket-0" }

	err := repo.Register(context.Background(), "r1", "http://10.0.0.1:8080")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRegister_EmptyPrivateURL は privateURL が空の場合にエラーを返すことを検証する。
func TestRegister_EmptyPrivateURL(t *testing.T) {
	t.Parallel()
	repo := NewDynamoRepository(&mockDynamoDBAPI{}, "t")

	err := repo.Register(context.Background(), "r1", "")
	if err == nil {
		t.Fatal("expected error for empty privateURL")
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

	err := repo.Register(context.Background(), "r1", "http://10.0.0.1:8080")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestBucketCount は BucketCount がバケット数を返すことを検証する。
func TestBucketCount(t *testing.T) {
	t.Parallel()
	repo := NewDynamoRepository(&mockDynamoDBAPI{}, "t")
	if repo.BucketCount() != bucketCount {
		t.Errorf("BucketCount() = %d, want %d", repo.BucketCount(), bucketCount)
	}
}

// TestAcquireIdle_Success は指定バケットから idle runner の確保が成功するケースを検証する。
func TestAcquireIdle_Success(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, params *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			bucket := params.ExpressionAttributeValues[":b"].(*types.AttributeValueMemberS).Value
			if bucket != "bucket-2" {
				t.Errorf("bucket = %q, want %q", bucket, "bucket-2")
			}
			if *params.Limit != idleQueryLimit {
				t.Errorf("Limit = %d, want %d", *params.Limit, idleQueryLimit)
			}
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{
						"runnerId":   &types.AttributeValueMemberS{Value: "r1"},
						"idleBucket": &types.AttributeValueMemberS{Value: "bucket-2"},
					},
				},
			}, nil
		},
		updateItemFn: func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			return &dynamodb.UpdateItemOutput{}, nil
		},
	}
	repo := NewDynamoRepository(mock, "t")

	runner, err := repo.AcquireIdle(context.Background(), "sess-1", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner.CurrentSessionID != "sess-1" {
		t.Errorf("currentSessionId = %q, want %q", runner.CurrentSessionID, "sess-1")
	}
	if runner.IdleBucket != "" {
		t.Errorf("idleBucket = %q, want empty", runner.IdleBucket)
	}
}

// TestAcquireIdle_NoIdleRunner はバケットが空の場合に ErrNoIdleRunner を返すことを検証する。
func TestAcquireIdle_NoIdleRunner(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{Items: nil}, nil
		},
	}
	repo := NewDynamoRepository(mock, "t")

	_, err := repo.AcquireIdle(context.Background(), "sess-1", 0)
	if !errors.Is(err, ErrNoIdleRunner) {
		t.Fatalf("expected ErrNoIdleRunner, got: %v", err)
	}
}

// TestAcquireIdle_QueryError は Query エラー時にエラーを返すことを検証する。
func TestAcquireIdle_QueryError(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return nil, errors.New("query error")
		},
	}
	repo := NewDynamoRepository(mock, "t")

	_, err := repo.AcquireIdle(context.Background(), "sess-1", 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

// assertBucket は QueryInput のバケットバインディングを検証するヘルパー。
func assertBucket(t *testing.T, params *dynamodb.QueryInput, want string) {
	t.Helper()
	got, ok := params.ExpressionAttributeValues[":b"].(*types.AttributeValueMemberS)
	if !ok || got.Value != want {
		t.Fatalf("bucket binding = %v, want %q", params.ExpressionAttributeValues[":b"], want)
	}
}

// TestAcquireIdle_RetryWithinBatch は同一バッチ内で競合時に次の候補を試行することを検証する。
func TestAcquireIdle_RetryWithinBatch(t *testing.T) {
	t.Parallel()
	queryCount := 0
	updateCount := 0
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, params *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			assertBucket(t, params, "bucket-0")
			queryCount++
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{
						"runnerId":   &types.AttributeValueMemberS{Value: "r1"},
						"idleBucket": &types.AttributeValueMemberS{Value: "bucket-0"},
					},
					{
						"runnerId":   &types.AttributeValueMemberS{Value: "r2"},
						"idleBucket": &types.AttributeValueMemberS{Value: "bucket-0"},
					},
				},
			}, nil
		},
		updateItemFn: func(_ context.Context, params *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			updateCount++
			rid := params.Key["runnerId"].(*types.AttributeValueMemberS).Value
			// r2 のみ成功可能。シャッフル順序に関係なく r1 は必ず競合する。
			if rid != "r2" {
				return nil, &types.ConditionalCheckFailedException{Message: aws.String("conflict")}
			}
			return &dynamodb.UpdateItemOutput{}, nil
		},
	}
	repo := NewDynamoRepository(mock, "t")

	runner, err := repo.AcquireIdle(context.Background(), "sess-1", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner.RunnerID != "r2" {
		t.Errorf("runnerID = %q, want %q", runner.RunnerID, "r2")
	}
	if queryCount != 1 {
		t.Errorf("query count = %d, want 1", queryCount)
	}
	if updateCount < 1 || updateCount > 2 {
		t.Errorf("update count = %d, want 1 or 2", updateCount)
	}
}

// TestAcquireIdle_AllConflictThenEmpty はバッチ内全候補が競合し次クエリが空の場合に ErrNoIdleRunner を返すことを検証する。
func TestAcquireIdle_AllConflictThenEmpty(t *testing.T) {
	t.Parallel()
	queryCount := 0
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, params *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			assertBucket(t, params, "bucket-0")
			queryCount++
			if queryCount == 1 {
				return &dynamodb.QueryOutput{
					Items: []map[string]types.AttributeValue{
						{
							"runnerId":   &types.AttributeValueMemberS{Value: "r1"},
							"idleBucket": &types.AttributeValueMemberS{Value: "bucket-0"},
						},
					},
				}, nil
			}
			return &dynamodb.QueryOutput{Items: nil}, nil
		},
		updateItemFn: func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			return nil, &types.ConditionalCheckFailedException{Message: aws.String("conflict")}
		},
	}
	repo := NewDynamoRepository(mock, "t")

	_, err := repo.AcquireIdle(context.Background(), "sess-1", 0)
	if !errors.Is(err, ErrNoIdleRunner) {
		t.Fatalf("expected ErrNoIdleRunner, got: %v", err)
	}
	if queryCount != 2 {
		t.Errorf("query count = %d, want 2", queryCount)
	}
}

// TestAcquireIdle_StaleGSI は GSI が stale な項目を返し続ける場合に無限ループせず ErrNoIdleRunner を返すことを検証する。
func TestAcquireIdle_StaleGSI(t *testing.T) {
	t.Parallel()
	queryCount := 0
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, params *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			assertBucket(t, params, "bucket-0")
			queryCount++
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{
						"runnerId":   &types.AttributeValueMemberS{Value: "r-stale"},
						"idleBucket": &types.AttributeValueMemberS{Value: "bucket-0"},
					},
				},
			}, nil
		},
		updateItemFn: func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			return nil, &types.ConditionalCheckFailedException{Message: aws.String("conflict")}
		},
	}
	repo := NewDynamoRepository(mock, "t")

	_, err := repo.AcquireIdle(context.Background(), "sess-1", 0)
	if !errors.Is(err, ErrNoIdleRunner) {
		t.Fatalf("expected ErrNoIdleRunner, got: %v", err)
	}
	if queryCount != 2 {
		t.Errorf("query count = %d, want 2 (initial batch tried + stale detection)", queryCount)
	}
}

// TestAcquireIdle_UpdateError は UpdateItem の予期せぬエラーを検証する。
func TestAcquireIdle_UpdateError(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{
						"runnerId":   &types.AttributeValueMemberS{Value: "r1"},
						"idleBucket": &types.AttributeValueMemberS{Value: "bucket-0"},
					},
				},
			}, nil
		},
		updateItemFn: func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			return nil, errors.New("update error")
		},
	}
	repo := NewDynamoRepository(mock, "t")

	_, err := repo.AcquireIdle(context.Background(), "sess-1", 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestAcquireIdle_UnmarshalError は Query 結果の unmarshal 失敗を検証する。
func TestAcquireIdle_UnmarshalError(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{
						"runnerId": &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
					},
				},
			}, nil
		},
	}
	repo := NewDynamoRepository(mock, "t")

	_, err := repo.AcquireIdle(context.Background(), "sess-1", 0)
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
}

// TestAcquireIdle_ShuffleDistribution はバッチ内の候補がランダムに選ばれることを検証する。
func TestAcquireIdle_ShuffleDistribution(t *testing.T) {
	t.Parallel()
	assigned := map[string]int{}
	for range 1000 {
		mock := &mockDynamoDBAPI{
			queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
				return &dynamodb.QueryOutput{
					Items: []map[string]types.AttributeValue{
						{
							"runnerId":   &types.AttributeValueMemberS{Value: "r1"},
							"idleBucket": &types.AttributeValueMemberS{Value: "bucket-0"},
						},
						{
							"runnerId":   &types.AttributeValueMemberS{Value: "r2"},
							"idleBucket": &types.AttributeValueMemberS{Value: "bucket-0"},
						},
						{
							"runnerId":   &types.AttributeValueMemberS{Value: "r3"},
							"idleBucket": &types.AttributeValueMemberS{Value: "bucket-0"},
						},
					},
				}, nil
			},
			updateItemFn: func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
				return &dynamodb.UpdateItemOutput{}, nil
			},
		}
		repo := NewDynamoRepository(mock, "t")
		runner, err := repo.AcquireIdle(context.Background(), "sess-1", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assigned[runner.RunnerID]++
	}
	for _, id := range []string{"r1", "r2", "r3"} {
		if assigned[id] == 0 {
			t.Errorf("runner %q was never selected in 100 iterations", id)
		}
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

// TestFindBySessionID_UnmarshalError は Query 結果の unmarshal 失敗を検証する。
func TestFindBySessionID_UnmarshalError(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{
						"runnerId": &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
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

// TestFindByID_Success は runner ID で runner が見つかるケースを検証する。
func TestFindByID_Success(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"runnerId":   &types.AttributeValueMemberS{Value: "r1"},
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
	if !runner.IsIdle() {
		t.Error("expected runner to be idle")
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

// TestFindByID_UnmarshalError は GetItem 結果の unmarshal 失敗を検証する。
func TestFindByID_UnmarshalError(t *testing.T) {
	t.Parallel()
	mock := &mockDynamoDBAPI{
		getItemFn: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"runnerId": &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
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

// TestDynamoRepository_ImplementsRepository は DynamoRepository が Repository インターフェースを満たすことを検証する。
func TestDynamoRepository_ImplementsRepository(t *testing.T) {
	t.Parallel()
	var _ Repository = (*DynamoRepository)(nil)
}
