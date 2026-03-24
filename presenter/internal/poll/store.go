// Package poll はアンケート投票機能を提供する。
package poll

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamoDBAPI は poll_votes テーブル用の narrow interface。
type DynamoDBAPI interface {
	// PutItem は DynamoDB にアイテムを書き込む。
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	// GetItem は DynamoDB からアイテムを取得する。
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	// DeleteItem は DynamoDB のアイテムを削除する。
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	// UpdateItem は DynamoDB のアイテムを更新する。
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	// Query は DynamoDB のクエリを実行する。
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

// Store は poll_votes テーブルの操作を提供する。
type Store struct {
	client       DynamoDBAPI
	tableName    string
	nowFn        func() time.Time
	marshalMapFn func(in interface{}) (map[string]types.AttributeValue, error)
}

// NewStore は Store を生成する。
func NewStore(client DynamoDBAPI, tableName string) *Store {
	return &Store{
		client:       client,
		tableName:    tableName,
		nowFn:        time.Now,
		marshalMapFn: attributevalue.MarshalMap,
	}
}

// PollState はアンケートの現在状態を表す。
type PollState struct {
	PollID     string         `json:"pollId"`
	Options    []string       `json:"options"`
	MaxChoices int            `json:"maxChoices"`
	Votes      map[string]int `json:"votes"`
	MyChoices  []string       `json:"myChoices"`
}

// PollError はアンケート操作エラーのレスポンスを表す。
type PollError struct {
	PollID    string         `json:"pollId"`
	Error     string         `json:"error"`
	Votes     map[string]int `json:"votes"`
	MyChoices []string       `json:"myChoices"`
}

// ErrMaxChoicesExceeded は最大選択数を超過した場合のエラー。
var ErrMaxChoicesExceeded = fmt.Errorf("max choices exceeded")

// ErrDuplicateVote は重複投票の場合のエラー。
var ErrDuplicateVote = fmt.Errorf("duplicate vote")

// ErrVoteNotFound は投票が存在しない場合のエラー。
var ErrVoteNotFound = fmt.Errorf("vote not found")

// ErrPollNotFound はアンケートが存在しない場合のエラー。
var ErrPollNotFound = fmt.Errorf("poll not found")

// metaSK は META レコードのソートキー。
const metaSK = "META"

// ttlDuration は TTL の有効期間。
const ttlDuration = 24 * time.Hour
