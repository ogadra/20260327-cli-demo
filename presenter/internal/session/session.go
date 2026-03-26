// Package session はプレゼンターセッションの永続化層を提供する。
package session

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamoDBAPI はセッションテーブル用の narrow interface。
type DynamoDBAPI interface {
	// PutItem は DynamoDB にアイテムを書き込む。
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

// sessionItem はセッションの DynamoDB アイテムを表す。
type sessionItem struct {
	Token  string `dynamodbav:"token"`
	Status string `dynamodbav:"status"`
}

// Store はセッションテーブルの操作を提供する。
type Store struct {
	client    DynamoDBAPI
	tableName string
	marshalFn func(in interface{}) (map[string]types.AttributeValue, error)
}

// NewStore は Store を生成する。
func NewStore(client DynamoDBAPI, tableName string) *Store {
	return &Store{
		client:    client,
		tableName: tableName,
		marshalFn: attributevalue.MarshalMap,
	}
}

// Create はセッションを作成する。セッションは無期限で有効。DynamoDB TTL によるレコード自動削除は行わない。
func (s *Store) Create(ctx context.Context, token string) error {
	item := sessionItem{
		Token:  token,
		Status: "valid",
	}
	av, err := s.marshalFn(item)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &s.tableName,
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("put session: %w", err)
	}
	return nil
}
