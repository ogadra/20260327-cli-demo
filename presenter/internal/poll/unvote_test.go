package poll

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// TestUnvote_Success は正常な投票取消を検証する。
func TestUnvote_Success(t *testing.T) {
	t.Parallel()
	var deletedSK string
	var updateCalled bool
	client := &mockDynamoDBAPI{
		deleteItemFn: func(_ context.Context, params *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			deletedSK = params.Key["connectionId"].(*types.AttributeValueMemberS).Value
			return &dynamodb.DeleteItemOutput{}, nil
		},
		updateItemFn: func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			updateCalled = true
			return &dynamodb.UpdateItemOutput{}, nil
		},
	}
	s := NewStore(client, "table")
	err := s.Unvote(context.Background(), "q1", "visitor1", "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedSK != "visitor1#A" {
		t.Errorf("expected visitor1#A, got %s", deletedSK)
	}
	if !updateCalled {
		t.Error("expected UpdateItem to be called")
	}
}

// TestUnvote_VoteNotFound は投票が存在しない場合のエラーを検証する。
func TestUnvote_VoteNotFound(t *testing.T) {
	t.Parallel()
	client := &mockDynamoDBAPI{
		deleteItemFn: func(_ context.Context, _ *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			return nil, &types.ConditionalCheckFailedException{Message: stringPtr("not found")}
		},
	}
	s := NewStore(client, "table")
	err := s.Unvote(context.Background(), "q1", "visitor1", "A")
	if err != ErrVoteNotFound {
		t.Errorf("expected ErrVoteNotFound, got %v", err)
	}
}

// TestUnvote_DeleteError は DeleteItem の非条件エラーを検証する。
func TestUnvote_DeleteError(t *testing.T) {
	t.Parallel()
	client := &mockDynamoDBAPI{
		deleteItemFn: func(_ context.Context, _ *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			return nil, fmt.Errorf("delete error")
		},
	}
	s := NewStore(client, "table")
	err := s.Unvote(context.Background(), "q1", "visitor1", "A")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestUnvote_UpdateError は UpdateItem エラーを検証する。
func TestUnvote_UpdateError(t *testing.T) {
	t.Parallel()
	client := &mockDynamoDBAPI{
		deleteItemFn: func(_ context.Context, _ *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			return &dynamodb.DeleteItemOutput{}, nil
		},
		updateItemFn: func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			return nil, fmt.Errorf("update error")
		},
	}
	s := NewStore(client, "table")
	err := s.Unvote(context.Background(), "q1", "visitor1", "A")
	if err == nil {
		t.Fatal("expected error")
	}
}
