package poll

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// metaRecord は META レコードの DynamoDB 表現。
type metaRecord struct {
	PollID     string         `dynamodbav:"pollId"`
	ConnID     string         `dynamodbav:"connectionId"`
	Options    []string       `dynamodbav:"options"`
	MaxChoices int            `dynamodbav:"maxChoices"`
	Votes      map[string]int `dynamodbav:"votes"`
	TTL        int64          `dynamodbav:"ttl"`
}

// Get はアンケートの状態を取得する。存在しない場合は初期化する。
// presenter ロールのみ初期化可能。初期化は条件付き PUT で冪等。
func (s *Store) Get(ctx context.Context, pollID, visitorID string, options []string, maxChoices int, isPresenter bool) (*PollState, error) {
	if isPresenter && options != nil {
		initVotes := make(map[string]int, len(options))
		for _, opt := range options {
			initVotes[opt] = 0
		}
		rec := metaRecord{
			PollID:     pollID,
			ConnID:     metaSK,
			Options:    options,
			MaxChoices: maxChoices,
			Votes:      initVotes,
			TTL:        s.nowFn().Add(ttlDuration).Unix(),
		}
		item, err := s.marshalMapFn(rec)
		if err != nil {
			return nil, fmt.Errorf("marshal meta: %w", err)
		}
		_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
			TableName:           &s.tableName,
			Item:                item,
			ConditionExpression: aws.String("attribute_not_exists(pollId)"),
		})
		if err != nil && !isConditionalCheckFailed(err) {
			return nil, fmt.Errorf("put meta: %w", err)
		}
	}

	meta, err := s.getMeta(ctx, pollID)
	if err != nil {
		return nil, err
	}

	myChoices, err := s.getMyChoices(ctx, pollID, visitorID)
	if err != nil {
		return nil, err
	}

	return &PollState{
		PollID:     meta.PollID,
		Options:    meta.Options,
		MaxChoices: meta.MaxChoices,
		Votes:      meta.Votes,
		MyChoices:  myChoices,
	}, nil
}

// getMeta は META レコードを取得する。
func (s *Store) getMeta(ctx context.Context, pollID string) (*metaRecord, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"pollId":       &types.AttributeValueMemberS{Value: pollID},
			"connectionId": &types.AttributeValueMemberS{Value: metaSK},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get meta: %w", err)
	}
	if out.Item == nil {
		return nil, ErrPollNotFound
	}
	var rec metaRecord
	if err := attributevalue.UnmarshalMap(out.Item, &rec); err != nil {
		return nil, fmt.Errorf("unmarshal meta: %w", err)
	}
	return &rec, nil
}

// getMyChoices は指定した visitorID の投票済み選択肢リストを取得する。
func (s *Store) getMyChoices(ctx context.Context, pollID, visitorID string) ([]string, error) {
	if visitorID == "" {
		return []string{}, nil
	}
	prefix := visitorID + "#"
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              &s.tableName,
		KeyConditionExpression: aws.String("pollId = :pid AND begins_with(connectionId, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pid":    &types.AttributeValueMemberS{Value: pollID},
			":prefix": &types.AttributeValueMemberS{Value: prefix},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("query my choices: %w", err)
	}
	choices := make([]string, 0, len(out.Items))
	for _, item := range out.Items {
		sk, ok := item["connectionId"]
		if !ok {
			continue
		}
		skVal, ok := sk.(*types.AttributeValueMemberS)
		if !ok {
			continue
		}
		parts := strings.SplitN(skVal.Value, "#", 2)
		if len(parts) == 2 {
			choices = append(choices, parts[1])
		}
	}
	return choices, nil
}

// isConditionalCheckFailed は ConditionalCheckFailedException かどうかを判定する。
func isConditionalCheckFailed(err error) bool {
	var cfe *types.ConditionalCheckFailedException
	return errors.As(err, &cfe)
}

// transactionCanceledReasons は TransactionCanceledException のキャンセル理由コードを返す。
// TransactionCanceledException でない場合は nil を返す。
func transactionCanceledReasons(err error) []string {
	var tce *types.TransactionCanceledException
	if !errors.As(err, &tce) {
		return nil
	}
	reasons := make([]string, len(tce.CancellationReasons))
	for i, r := range tce.CancellationReasons {
		if r.Code != nil {
			reasons[i] = *r.Code
		}
	}
	return reasons
}
