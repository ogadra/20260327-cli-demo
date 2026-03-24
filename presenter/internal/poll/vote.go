package poll

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Vote は投票を記録する。maxChoices 超過と重複投票を防止する。
func (s *Store) Vote(ctx context.Context, pollID, visitorID, choice string) error {
	meta, err := s.getMeta(ctx, pollID)
	if err != nil {
		return err
	}

	myChoices, err := s.getMyChoices(ctx, pollID, visitorID)
	if err != nil {
		return err
	}
	if len(myChoices) >= meta.MaxChoices {
		return ErrMaxChoicesExceeded
	}

	sk := visitorID + "#" + choice
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &s.tableName,
		Item: map[string]types.AttributeValue{
			"pollId":       &types.AttributeValueMemberS{Value: pollID},
			"connectionId": &types.AttributeValueMemberS{Value: sk},
			"ttl":          &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", s.nowFn().Add(ttlDuration).Unix())},
		},
		ConditionExpression: aws.String("attribute_not_exists(connectionId)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return ErrDuplicateVote
		}
		return fmt.Errorf("put vote: %w", err)
	}

	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"pollId":       &types.AttributeValueMemberS{Value: pollID},
			"connectionId": &types.AttributeValueMemberS{Value: metaSK},
		},
		UpdateExpression: aws.String("ADD votes.#choice :one"),
		ExpressionAttributeNames: map[string]string{
			"#choice": choice,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":one": &types.AttributeValueMemberN{Value: "1"},
		},
	})
	if err != nil {
		return fmt.Errorf("update votes: %w", err)
	}

	return nil
}
