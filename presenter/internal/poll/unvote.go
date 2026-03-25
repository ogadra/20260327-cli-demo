package poll

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Unvote は投票を取り消す。投票が存在しない場合は ErrVoteNotFound を返す。
// 投票レコードの削除とカウンター更新を TransactWriteItems でアトミックに実行する。
func (s *Store) Unvote(ctx context.Context, pollID, visitorID, choice string) error {
	sk := visitorID + "#" + choice
	_, err := s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Delete: &types.Delete{
					TableName: &s.tableName,
					Key: map[string]types.AttributeValue{
						"pollId":       &types.AttributeValueMemberS{Value: pollID},
						"connectionId": &types.AttributeValueMemberS{Value: sk},
					},
					ConditionExpression: aws.String("attribute_exists(connectionId)"),
				},
			},
			{
				Update: &types.Update{
					TableName: &s.tableName,
					Key: map[string]types.AttributeValue{
						"pollId":       &types.AttributeValueMemberS{Value: pollID},
						"connectionId": &types.AttributeValueMemberS{Value: metaSK},
					},
					UpdateExpression: aws.String("SET votes.#choice = if_not_exists(votes.#choice, :zero) - :one"),
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
			return ErrVoteNotFound
		}
		return fmt.Errorf("transact unvote: %w", err)
	}

	return nil
}
