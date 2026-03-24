package poll

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Unvote は投票を取り消す。投票が存在しない場合は ErrVoteNotFound を返す。
func (s *Store) Unvote(ctx context.Context, pollID, visitorID, choice string) error {
	sk := visitorID + "#" + choice
	_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"pollId":       &types.AttributeValueMemberS{Value: pollID},
			"connectionId": &types.AttributeValueMemberS{Value: sk},
		},
		ConditionExpression: aws.String("attribute_exists(connectionId)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return ErrVoteNotFound
		}
		return fmt.Errorf("delete vote: %w", err)
	}

	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"pollId":       &types.AttributeValueMemberS{Value: pollID},
			"connectionId": &types.AttributeValueMemberS{Value: metaSK},
		},
		UpdateExpression: aws.String("ADD votes.#choice :negone"),
		ExpressionAttributeNames: map[string]string{
			"#choice": choice,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":negone": &types.AttributeValueMemberN{Value: "-1"},
		},
	})
	if err != nil {
		return fmt.Errorf("update votes: %w", err)
	}

	return nil
}
