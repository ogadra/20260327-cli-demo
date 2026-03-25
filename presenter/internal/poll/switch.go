package poll

import (
	"context"
	"fmt"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Switch は投票を変更する。旧選択肢の削除、新選択肢の追加、カウンター更新を
// TransactWriteItems でアトミックに実行する。
// from または to が meta.Options に含まれない場合は ErrInvalidChoice を返す。
func (s *Store) Switch(ctx context.Context, pollID, visitorID, from, to string) error {
	meta, err := s.getMeta(ctx, pollID)
	if err != nil {
		return err
	}
	if !slices.Contains(meta.Options, from) || !slices.Contains(meta.Options, to) {
		return ErrInvalidChoice
	}

	fromSK := visitorID + "#" + from
	toSK := visitorID + "#" + to

	_, err = s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Delete: &types.Delete{
					TableName: &s.tableName,
					Key: map[string]types.AttributeValue{
						"pollId":       &types.AttributeValueMemberS{Value: pollID},
						"connectionId": &types.AttributeValueMemberS{Value: fromSK},
					},
					ConditionExpression: aws.String("attribute_exists(connectionId)"),
				},
			},
			{
				Put: &types.Put{
					TableName: &s.tableName,
					Item: map[string]types.AttributeValue{
						"pollId":       &types.AttributeValueMemberS{Value: pollID},
						"connectionId": &types.AttributeValueMemberS{Value: toSK},
						"ttl":          &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", s.nowFn().Add(ttlDuration).Unix())},
					},
					ConditionExpression: aws.String("attribute_not_exists(connectionId)"),
				},
			},
			{
				Update: &types.Update{
					TableName: &s.tableName,
					Key: map[string]types.AttributeValue{
						"pollId":       &types.AttributeValueMemberS{Value: pollID},
						"connectionId": &types.AttributeValueMemberS{Value: metaSK},
					},
					UpdateExpression: aws.String("SET votes.#from = if_not_exists(votes.#from, :zero) - :one, votes.#to = if_not_exists(votes.#to, :zero) + :one"),
					ExpressionAttributeNames: map[string]string{
						"#from": from,
						"#to":   to,
					},
					ExpressionAttributeValues: map[string]types.AttributeValue{
						":zero": &types.AttributeValueMemberN{Value: "0"},
						":one":  &types.AttributeValueMemberN{Value: "1"},
					},
				},
			},
		},
	})
	if err != nil {
		reasons := transactionCanceledReasons(err)
		if len(reasons) >= 2 {
			if reasons[0] == "ConditionalCheckFailed" {
				return ErrVoteNotFound
			}
			if reasons[1] == "ConditionalCheckFailed" {
				return ErrDuplicateVote
			}
		}
		return fmt.Errorf("transact switch: %w", err)
	}

	return nil
}
