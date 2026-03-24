// Package connection は WebSocket 接続の永続化層を提供する。
package connection

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamoDBAPI は ws_connections テーブル用の narrow interface。
type DynamoDBAPI interface {
	// PutItem は DynamoDB にアイテムを書き込む。
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	// GetItem は DynamoDB からアイテムを取得する。
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	// DeleteItem は DynamoDB のアイテムを削除する。
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	// Query は DynamoDB のクエリを実行する。
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

// SessionDynamoDBAPI はセッションテーブル用の narrow interface。
type SessionDynamoDBAPI interface {
	// GetItem は DynamoDB からアイテムを取得する。
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
}

// Connection は WebSocket 接続情報を表す。
type Connection struct {
	Room         string `dynamodbav:"room"`
	ConnectionID string `dynamodbav:"connectionId"`
	Role         string `dynamodbav:"role"`
	TTL          int64  `dynamodbav:"ttl"`
}

// ErrNotFound は指定された接続が存在しない場合に返される。
var ErrNotFound = fmt.Errorf("connection not found")

// Store は ws_connections テーブルの操作を提供する。
type Store struct {
	client    DynamoDBAPI
	tableName string
	nowFn     func() time.Time
	marshalFn func(in interface{}) (map[string]types.AttributeValue, error)
}

// NewStore は Store を生成する。
func NewStore(client DynamoDBAPI, tableName string) *Store {
	return &Store{
		client:    client,
		tableName: tableName,
		nowFn:     time.Now,
		marshalFn: attributevalue.MarshalMap,
	}
}

// Put は接続情報を保存する。TTL は現在時刻から 24 時間後に設定される。
func (s *Store) Put(ctx context.Context, conn Connection) error {
	conn.TTL = s.nowFn().Add(24 * time.Hour).Unix()
	item, err := s.marshalFn(conn)
	if err != nil {
		return fmt.Errorf("marshal connection: %w", err)
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &s.tableName,
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put item: %w", err)
	}
	return nil
}

// Get は指定した room と connectionID の接続情報を取得する。
func (s *Store) Get(ctx context.Context, room, connectionID string) (*Connection, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"room":         &types.AttributeValueMemberS{Value: room},
			"connectionId": &types.AttributeValueMemberS{Value: connectionID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get item: %w", err)
	}
	if out.Item == nil {
		return nil, ErrNotFound
	}
	var conn Connection
	if err := attributevalue.UnmarshalMap(out.Item, &conn); err != nil {
		return nil, fmt.Errorf("unmarshal connection: %w", err)
	}
	return &conn, nil
}

// Delete は接続情報を削除する。条件なしで冪等。
func (s *Store) Delete(ctx context.Context, room, connectionID string) error {
	_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"room":         &types.AttributeValueMemberS{Value: room},
			"connectionId": &types.AttributeValueMemberS{Value: connectionID},
		},
	})
	if err != nil {
		return fmt.Errorf("delete item: %w", err)
	}
	return nil
}

// QueryByRoom は room 内の全接続を取得する。
func (s *Store) QueryByRoom(ctx context.Context, room string) ([]Connection, error) {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              &s.tableName,
		KeyConditionExpression: aws.String("#r = :room"),
		ExpressionAttributeNames: map[string]string{
			"#r": "room",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":room": &types.AttributeValueMemberS{Value: room},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("query by room: %w", err)
	}
	conns := make([]Connection, 0, len(out.Items))
	for _, item := range out.Items {
		var conn Connection
		if err := attributevalue.UnmarshalMap(item, &conn); err != nil {
			return nil, fmt.Errorf("unmarshal connection: %w", err)
		}
		conns = append(conns, conn)
	}
	return conns, nil
}

// CountByRoom は room 内の接続数を取得する。
func (s *Store) CountByRoom(ctx context.Context, room string) (int, error) {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              &s.tableName,
		KeyConditionExpression: aws.String("#r = :room"),
		ExpressionAttributeNames: map[string]string{
			"#r": "room",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":room": &types.AttributeValueMemberS{Value: room},
		},
		Select: types.SelectCount,
	})
	if err != nil {
		return 0, fmt.Errorf("count by room: %w", err)
	}
	return int(out.Count), nil
}

// SessionStore はセッショントークンの検証を提供する。
type SessionStore struct {
	client    SessionDynamoDBAPI
	tableName string
}

// NewSessionStore は SessionStore を生成する。
func NewSessionStore(client SessionDynamoDBAPI, tableName string) *SessionStore {
	return &SessionStore{
		client:    client,
		tableName: tableName,
	}
}

// IsValid はトークンが有効かどうかを検証する。
func (ss *SessionStore) IsValid(ctx context.Context, token string) (bool, error) {
	if token == "" {
		return false, nil
	}
	out, err := ss.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &ss.tableName,
		Key: map[string]types.AttributeValue{
			"token": &types.AttributeValueMemberS{Value: token},
		},
	})
	if err != nil {
		return false, fmt.Errorf("get session: %w", err)
	}
	if out.Item == nil {
		return false, nil
	}
	statusAttr, ok := out.Item["status"]
	if !ok {
		return false, nil
	}
	statusVal, ok := statusAttr.(*types.AttributeValueMemberS)
	if !ok {
		return false, nil
	}
	return statusVal.Value == "valid", nil
}
