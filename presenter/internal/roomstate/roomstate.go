// Package roomstate は room の現在のスライド状態を永続化する。
package roomstate

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamoDBAPI は room_state テーブル用の narrow interface。
type DynamoDBAPI interface {
	// PutItem は DynamoDB にアイテムを書き込む。
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	// GetItem は DynamoDB からアイテムを取得する。
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
}

// State は room の現在のスライド状態を表す。
type State struct {
	Room string `dynamodbav:"room"`
	Page int    `dynamodbav:"page"`
}

// Store は room_state テーブルの操作を提供する。
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

// PutState は room の現在のスライドページを保存する。
func (s *Store) PutState(ctx context.Context, room string, page int) error {
	state := State{Room: room, Page: page}
	item, err := s.marshalFn(state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &s.tableName,
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put state: %w", err)
	}
	return nil
}

// GetState は room の現在のスライドページを取得する。状態が未保存の場合は 0 を返す。
func (s *Store) GetState(ctx context.Context, room string) (int, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"room": &types.AttributeValueMemberS{Value: room},
		},
	})
	if err != nil {
		return 0, fmt.Errorf("get state: %w", err)
	}
	if out.Item == nil {
		return 0, nil
	}
	pageAttr, ok := out.Item["page"]
	if !ok {
		return 0, nil
	}
	pageVal, ok := pageAttr.(*types.AttributeValueMemberN)
	if !ok {
		return 0, nil
	}
	var page int
	if _, err := fmt.Sscanf(pageVal.Value, "%d", &page); err != nil {
		return 0, fmt.Errorf("parse page: %w", err)
	}
	return page, nil
}
