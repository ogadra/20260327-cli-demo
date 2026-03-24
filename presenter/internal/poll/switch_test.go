package poll

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// TestSwitch_Success は正常な投票変更を検証する。
func TestSwitch_Success(t *testing.T) {
	t.Parallel()
	var deletedSK, putSK string
	var updateCalled bool
	client := &mockDynamoDBAPI{
		deleteItemFn: func(_ context.Context, params *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			deletedSK = params.Key["connectionId"].(*types.AttributeValueMemberS).Value
			return &dynamodb.DeleteItemOutput{}, nil
		},
		putItemFn: func(_ context.Context, params *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			putSK = params.Item["connectionId"].(*types.AttributeValueMemberS).Value
			return &dynamodb.PutItemOutput{}, nil
		},
		updateItemFn: func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			updateCalled = true
			return &dynamodb.UpdateItemOutput{}, nil
		},
	}
	s := NewStore(client, "table")
	err := s.Switch(context.Background(), "q1", "visitor1", "A", "B")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedSK != "visitor1#A" {
		t.Errorf("expected visitor1#A deleted, got %s", deletedSK)
	}
	if putSK != "visitor1#B" {
		t.Errorf("expected visitor1#B put, got %s", putSK)
	}
	if !updateCalled {
		t.Error("expected UpdateItem to be called")
	}
}

// TestSwitch_FromNotFound は旧選択肢が存在しない場合のエラーを検証する。
func TestSwitch_FromNotFound(t *testing.T) {
	t.Parallel()
	client := &mockDynamoDBAPI{
		deleteItemFn: func(_ context.Context, _ *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			return nil, &types.ConditionalCheckFailedException{Message: stringPtr("not found")}
		},
	}
	s := NewStore(client, "table")
	err := s.Switch(context.Background(), "q1", "visitor1", "A", "B")
	if err != ErrVoteNotFound {
		t.Errorf("expected ErrVoteNotFound, got %v", err)
	}
}

// TestSwitch_DeleteError は旧選択肢削除の非条件エラーを検証する。
func TestSwitch_DeleteError(t *testing.T) {
	t.Parallel()
	client := &mockDynamoDBAPI{
		deleteItemFn: func(_ context.Context, _ *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			return nil, fmt.Errorf("delete error")
		},
	}
	s := NewStore(client, "table")
	err := s.Switch(context.Background(), "q1", "visitor1", "A", "B")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestSwitch_DuplicateToWithRollback は新選択肢の重複で旧選択肢が復元されることを検証する。
func TestSwitch_DuplicateToWithRollback(t *testing.T) {
	t.Parallel()
	var rollbackCalled bool
	putCount := 0
	client := &mockDynamoDBAPI{
		deleteItemFn: func(_ context.Context, _ *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			return &dynamodb.DeleteItemOutput{}, nil
		},
		putItemFn: func(_ context.Context, params *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			putCount++
			if putCount == 1 {
				return nil, &types.ConditionalCheckFailedException{Message: stringPtr("duplicate")}
			}
			rollbackCalled = true
			return &dynamodb.PutItemOutput{}, nil
		},
	}
	s := NewStore(client, "table")
	err := s.Switch(context.Background(), "q1", "visitor1", "A", "B")
	if err != ErrDuplicateVote {
		t.Errorf("expected ErrDuplicateVote, got %v", err)
	}
	if !rollbackCalled {
		t.Error("expected rollback PutItem to be called")
	}
}

// TestSwitch_PutError は新選択肢追加の非条件エラーを検証する。
func TestSwitch_PutError(t *testing.T) {
	t.Parallel()
	client := &mockDynamoDBAPI{
		deleteItemFn: func(_ context.Context, _ *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			return &dynamodb.DeleteItemOutput{}, nil
		},
		putItemFn: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, fmt.Errorf("put error")
		},
	}
	s := NewStore(client, "table")
	err := s.Switch(context.Background(), "q1", "visitor1", "A", "B")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestSwitch_UpdateError は votes 更新エラーを検証する。
func TestSwitch_UpdateError(t *testing.T) {
	t.Parallel()
	client := &mockDynamoDBAPI{
		deleteItemFn: func(_ context.Context, _ *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			return &dynamodb.DeleteItemOutput{}, nil
		},
		putItemFn: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return &dynamodb.PutItemOutput{}, nil
		},
		updateItemFn: func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			return nil, fmt.Errorf("update error")
		},
	}
	s := NewStore(client, "table")
	err := s.Switch(context.Background(), "q1", "visitor1", "A", "B")
	if err == nil {
		t.Fatal("expected error")
	}
}
