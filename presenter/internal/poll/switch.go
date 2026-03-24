package poll

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Switch は投票を変更する。旧選択肢を削除し新選択肢を追加する。
// 新選択肢の追加が重複で失敗した場合は旧選択肢を復元する。
func (s *Store) Switch(ctx context.Context, pollID, visitorID, from, to string) error {
	fromSK := visitorID + "#" + from
	toSK := visitorID + "#" + to

	_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"pollId":       &types.AttributeValueMemberS{Value: pollID},
			"connectionId": &types.AttributeValueMemberS{Value: fromSK},
		},
		ConditionExpression: aws.String("attribute_exists(connectionId)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return ErrVoteNotFound
		}
		return fmt.Errorf("delete old vote: %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &s.tableName,
		Item: map[string]types.AttributeValue{
			"pollId":       &types.AttributeValueMemberS{Value: pollID},
			"connectionId": &types.AttributeValueMemberS{Value: toSK},
			"ttl":          &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", s.nowFn().Add(ttlDuration).Unix())},
		},
		ConditionExpression: aws.String("attribute_not_exists(connectionId)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			if rbErr := s.rollbackDelete(ctx, pollID, fromSK); rbErr != nil {
				return fmt.Errorf("rollback after duplicate vote failed: %w", rbErr)
			}
			return ErrDuplicateVote
		}
		return fmt.Errorf("put new vote: %w", err)
	}

	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"pollId":       &types.AttributeValueMemberS{Value: pollID},
			"connectionId": &types.AttributeValueMemberS{Value: metaSK},
		},
		UpdateExpression: aws.String("ADD votes.#from :negone, votes.#to :one"),
		ExpressionAttributeNames: map[string]string{
			"#from": from,
			"#to":   to,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":negone": &types.AttributeValueMemberN{Value: "-1"},
			":one":    &types.AttributeValueMemberN{Value: "1"},
		},
	})
	if err != nil {
		return fmt.Errorf("update votes: %w", err)
	}

	return nil
}

// rollbackDelete は削除した投票レコードを復元する。
func (s *Store) rollbackDelete(ctx context.Context, pollID, sk string) error {
	_, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &s.tableName,
		Item: map[string]types.AttributeValue{
			"pollId":       &types.AttributeValueMemberS{Value: pollID},
			"connectionId": &types.AttributeValueMemberS{Value: sk},
			"ttl":          &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", s.nowFn().Add(ttlDuration).Unix())},
		},
	})
	if err != nil {
		return fmt.Errorf("rollback put: %w", err)
	}
	return nil
}
