// Package main は WebSocket $disconnect ルートの Lambda ハンドラーを提供する。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/ogadra/20260327-cli-demo/presenter/internal/broadcast"
	"github.com/ogadra/20260327-cli-demo/presenter/internal/connection"
)

// fatalf はエラー時の終了処理。テスト時に差し替える。
var fatalf = log.Fatalf

// startLambda は lambda.Start のラッパー。テスト時に差し替える。
var startLambda = lambda.Start

// runFn は run のラッパー。テスト時に差し替える。
var runFn = run

// loadConfig は AWS 設定読み込みのラッパー。テスト時に差し替える。
var loadConfig = config.LoadDefaultConfig

// jsonMarshal は JSON エンコードのラッパー。テスト時に差し替える。
var jsonMarshal = json.Marshal

// room は WebSocket 接続のグループ識別子。
const room = "default"

// disconnectHandler は $disconnect イベントを処理するハンドラー。
type disconnectHandler struct {
	connStore   connectionManager
	broadcaster messageBroadcaster
}

// connectionManager は接続の削除とカウントのインターフェース。
type connectionManager interface {
	// Delete は接続情報を削除する。
	Delete(ctx context.Context, room, connectionID string) error
	// CountByRoom は room 内の接続数を取得する。
	CountByRoom(ctx context.Context, room string) (int, error)
}

// messageBroadcaster はメッセージ配信インターフェース。
type messageBroadcaster interface {
	// Send は room 内の全接続にメッセージを配信する。
	Send(ctx context.Context, room string, payload []byte, excludeConnectionID string) error
}

// viewerCountMessage は接続数通知メッセージ。
type viewerCountMessage struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// handle は $disconnect イベントを処理する。
func (h *disconnectHandler) handle(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	connectionID := req.RequestContext.ConnectionID

	if err := h.connStore.Delete(ctx, room, connectionID); err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("delete connection: %w", err)
	}

	count, err := h.connStore.CountByRoom(ctx, room)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("count connections: %w", err)
	}

	msg := viewerCountMessage{Type: "viewer_count", Count: count}
	payload, err := jsonMarshal(msg)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("marshal viewer_count: %w", err)
	}

	if err := h.broadcaster.Send(ctx, room, payload, ""); err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("broadcast viewer_count: %w", err)
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

// run は依存を初期化し Lambda ハンドラーを起動する。
func run() error {
	ctx := context.Background()
	cfg, err := loadConfig(ctx)
	if err != nil {
		return fmt.Errorf("load aws config: %w", err)
	}

	ddbClient := dynamodb.NewFromConfig(cfg)
	connTable := os.Getenv("CONNECTIONS_TABLE")
	if connTable == "" {
		return fmt.Errorf("CONNECTIONS_TABLE environment variable is required")
	}
	apigwEndpoint := os.Getenv("APIGW_ENDPOINT")

	apigwClient := apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
		if apigwEndpoint != "" {
			o.BaseEndpoint = &apigwEndpoint
		}
	})

	connStore := connection.NewStore(ddbClient, connTable)
	b := broadcast.NewBroadcaster(apigwClient, connStore, connStore)

	h := &disconnectHandler{
		connStore:   connStore,
		broadcaster: b,
	}

	startLambda(h.handle)
	return nil
}

// main は disconnect Lambda のエントリポイント。
func main() {
	if err := runFn(); err != nil {
		fatalf("disconnect: %v", err)
	}
}
