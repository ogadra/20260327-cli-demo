// Package store は Runner の永続化層を提供する。
package store

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/ogadra/20260327-cli-demo/broker/model"
)

// bucketCount は IdleBucket のバケット数。
// idle runner を複数バケットに分散させ、DynamoDB のホットパーティションを防ぐ。
const bucketCount = 4

// DynamoRepository は DynamoDB を使った Repository の実装。
type DynamoRepository struct {
	client    DynamoDBAPI
	tableName string
	bucketFn  func() string
}

// NewDynamoRepository は DynamoRepository を生成する。
func NewDynamoRepository(client DynamoDBAPI, tableName string) *DynamoRepository {
	return &DynamoRepository{
		client:    client,
		tableName: tableName,
		bucketFn:  defaultBucketFn,
	}
}

// defaultBucketFn はランダムなバケット値を返す。
func defaultBucketFn() string {
	return fmt.Sprintf("bucket-%d", rand.IntN(bucketCount))
}

// Register は runner を idle 状態で登録する。attribute_not_exists で冪等性を確保する。
func (r *DynamoRepository) Register(ctx context.Context, runnerID string) error {
	item, err := attributevalue.MarshalMap(model.Runner{
		RunnerID:   runnerID,
		IdleBucket: r.bucketFn(),
	})
	if err != nil {
		return fmt.Errorf("marshal runner: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           &r.tableName,
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(runnerId)"),
	})
	if err != nil {
		var condErr *types.ConditionalCheckFailedException
		if isConditionalCheckFailed(err, &condErr) {
			return nil
		}
		return fmt.Errorf("put item: %w", err)
	}
	return nil
}

// FindIdle は idle 状態の runner を1台返す。ランダムバケットから検索し、見つからなければ他バケットも試行する。
func (r *DynamoRepository) FindIdle(ctx context.Context) (*model.Runner, error) {
	start := rand.IntN(bucketCount)
	for i := range bucketCount {
		bucket := fmt.Sprintf("bucket-%d", (start+i)%bucketCount)
		runner, err := r.queryIdleBucket(ctx, bucket)
		if err != nil {
			return nil, err
		}
		if runner != nil {
			return runner, nil
		}
	}
	return nil, ErrNoIdleRunner
}

// queryIdleBucket は指定バケットから idle runner を1台取得する。
func (r *DynamoRepository) queryIdleBucket(ctx context.Context, bucket string) (*model.Runner, error) {
	out, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              &r.tableName,
		IndexName:              aws.String("idle-index"),
		KeyConditionExpression: aws.String("idleBucket = :b"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":b": &types.AttributeValueMemberS{Value: bucket},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, fmt.Errorf("query idle-index: %w", err)
	}
	if len(out.Items) == 0 {
		return nil, nil
	}
	var runner model.Runner
	if err := attributevalue.UnmarshalMap(out.Items[0], &runner); err != nil {
		return nil, fmt.Errorf("unmarshal runner: %w", err)
	}
	return &runner, nil
}

// AssignSession は runner に session を紐づけ idle から busy に遷移させる。idleBucket が存在する場合のみ成功する。
func (r *DynamoRepository) AssignSession(ctx context.Context, runnerID, sessionID string) error {
	_, err := r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &r.tableName,
		Key: map[string]types.AttributeValue{
			"runnerId": &types.AttributeValueMemberS{Value: runnerID},
		},
		UpdateExpression:    aws.String("SET currentSessionId = :sid REMOVE idleBucket"),
		ConditionExpression: aws.String("attribute_exists(idleBucket)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":sid": &types.AttributeValueMemberS{Value: sessionID},
		},
	})
	if err != nil {
		var condErr *types.ConditionalCheckFailedException
		if isConditionalCheckFailed(err, &condErr) {
			return ErrConditionFailed
		}
		return fmt.Errorf("update item: %w", err)
	}
	return nil
}

// FindBySessionID は session ID から runner を検索する。
func (r *DynamoRepository) FindBySessionID(ctx context.Context, sessionID string) (*model.Runner, error) {
	out, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              &r.tableName,
		IndexName:              aws.String("session-index"),
		KeyConditionExpression: aws.String("currentSessionId = :sid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":sid": &types.AttributeValueMemberS{Value: sessionID},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, fmt.Errorf("query session-index: %w", err)
	}
	if len(out.Items) == 0 {
		return nil, ErrNotFound
	}
	var runner model.Runner
	if err := attributevalue.UnmarshalMap(out.Items[0], &runner); err != nil {
		return nil, fmt.Errorf("unmarshal runner: %w", err)
	}
	return &runner, nil
}

// FindByID は runner ID から runner を検索する。
func (r *DynamoRepository) FindByID(ctx context.Context, runnerID string) (*model.Runner, error) {
	out, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &r.tableName,
		Key: map[string]types.AttributeValue{
			"runnerId": &types.AttributeValueMemberS{Value: runnerID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get item: %w", err)
	}
	if out.Item == nil {
		return nil, ErrNotFound
	}
	var runner model.Runner
	if err := attributevalue.UnmarshalMap(out.Item, &runner); err != nil {
		return nil, fmt.Errorf("unmarshal runner: %w", err)
	}
	return &runner, nil
}

// Delete は runner レコードを削除する。条件なしで冪等。
func (r *DynamoRepository) Delete(ctx context.Context, runnerID string) error {
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &r.tableName,
		Key: map[string]types.AttributeValue{
			"runnerId": &types.AttributeValueMemberS{Value: runnerID},
		},
	})
	if err != nil {
		return fmt.Errorf("delete item: %w", err)
	}
	return nil
}

// isConditionalCheckFailed は err が ConditionalCheckFailedException かどうかを判定するヘルパー。
func isConditionalCheckFailed(err error, target **types.ConditionalCheckFailedException) bool {
	var condErr *types.ConditionalCheckFailedException
	if errors.As(err, &condErr) {
		*target = condErr
		return true
	}
	return false
}
