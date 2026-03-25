package poll

import (
	"context"
	"fmt"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Vote は投票を記録する。maxChoices 超過、重複投票、無効な選択肢を防止する。
// 投票レコードの追加とカウンター更新を TransactWriteItems でアトミックに実行する。
func (s *Store) Vote(ctx context.Context, pollID, visitorID, choice string) error {
	meta, err := s.getMeta(ctx, pollID)
	if err != nil {
		return err
	}

	if !slices.Contains(meta.Options, choice) {
		return ErrInvalidChoice
	}

	myChoices, err := s.getMyChoices(ctx, pollID, visitorID)
	if err != nil {
		return err
	}
	if len(myChoices) >= meta.MaxChoices {
		return ErrMaxChoicesExceeded
	}

	sk := visitorID + "#" + choice
	_, err = s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Put: &types.Put{
					TableName: &s.tableName,
					Item: map[string]types.AttributeValue{
						"pollId":       &types.AttributeValueMemberS{Value: pollID},
						"connectionId": &types.AttributeValueMemberS{Value: sk},
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
					UpdateExpression: aws.String("SET votes.#choice = if_not_exists(votes.#choice, :zero) + :one"),
					ExpressionAttributeNames: map[string]string{
						"#choice": choice,
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
		if len(reasons) > 0 && reasons[0] == "ConditionalCheckFailed" {
			return ErrDuplicateVote
		}
		return fmt.Errorf("transact vote: %w", err)
	}

	return nil
}
