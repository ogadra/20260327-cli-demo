//go:build integration

// Package store は統合テスト用ヘルパーを提供する。
package store

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// setupIntegrationTable は統合テスト用の DynamoDB テーブルを作成し、クリーンアップ関数を返す。
func setupIntegrationTable(t *testing.T) (*dynamodb.Client, string) {
	t.Helper()

	endpoint := os.Getenv("DYNAMODB_ENDPOINT")
	if endpoint == "" {
		t.Skip("DYNAMODB_ENDPOINT not set, skipping integration test")
	}

	tableName := fmt.Sprintf("runners-%d", time.Now().UnixNano())

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("ap-northeast-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("dummy", "dummy", "")),
	)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	_, err = client.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		TableName: &tableName,
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("runnerId"), KeyType: types.KeyTypeHash},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("runnerId"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("idleBucket"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("currentSessionId"), AttributeType: types.ScalarAttributeTypeS},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("idle-index"),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("idleBucket"), KeyType: types.KeyTypeHash},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(5),
					WriteCapacityUnits: aws.Int64(5),
				},
			},
			{
				IndexName: aws.String("session-index"),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("currentSessionId"), KeyType: types.KeyTypeHash},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(5),
					WriteCapacityUnits: aws.Int64(5),
				},
			},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
	})
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	t.Cleanup(func() {
		_, _ = client.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{
			TableName: &tableName,
		})
	})

	return client, tableName
}
