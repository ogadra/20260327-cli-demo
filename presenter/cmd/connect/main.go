// Package main は WebSocket $connect ルートの Lambda ハンドラーを提供する。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

// connectHandler は $connect イベントを処理するハンドラー。
type connectHandler struct {
	connStore    connectionStore
	sessionStore sessionValidator
	broadcaster  messageBroadcaster
}

// connectionStore は接続の永続化インターフェース。
type connectionStore interface {
	// Put は接続情報を保存する。
	Put(ctx context.Context, conn connection.Connection) error
	// CountByRoom は room 内の接続数を取得する。
	CountByRoom(ctx context.Context, room string) (int, error)
}

// sessionValidator はセッショントークンの検証インターフェース。
type sessionValidator interface {
	// IsValid はトークンが有効かどうかを検証する。
	IsValid(ctx context.Context, token string) (bool, error)
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

// extractCookie は cookie ヘッダーから指定した名前の値を取得する。
func extractCookie(cookieHeader, name string) string {
	header := http.Header{}
	header.Add("Cookie", cookieHeader)
	request := http.Request{Header: header}
	cookie, err := request.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// handle は $connect イベントを処理する。
func (h *connectHandler) handle(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	connectionID := req.RequestContext.ConnectionID
	cookieHeader := req.Headers["cookie"]
	if cookieHeader == "" {
		cookieHeader = req.Headers["Cookie"]
	}
	token := extractCookie(cookieHeader, "slide_auth")

	role := "viewer"
	valid, err := h.sessionStore.IsValid(ctx, token)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("validate session: %w", err)
	}
	if valid {
		role = "presenter"
	}

	conn := connection.Connection{
		Room:         room,
		ConnectionID: connectionID,
		Role:         role,
	}
	if err := h.connStore.Put(ctx, conn); err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("put connection: %w", err)
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
	sessTable := os.Getenv("SESSIONS_TABLE")
	apigwEndpoint := os.Getenv("APIGW_ENDPOINT")

	apigwClient := apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
		if apigwEndpoint != "" {
			o.BaseEndpoint = &apigwEndpoint
		}
	})

	connStore := connection.NewStore(ddbClient, connTable)
	sessStore := connection.NewSessionStore(ddbClient, sessTable)
	b := broadcast.NewBroadcaster(apigwClient, connStore, connStore)

	h := &connectHandler{
		connStore:    connStore,
		sessionStore: sessStore,
		broadcaster:  b,
	}

	startLambda(h.handle)
	return nil
}

// main は connect Lambda のエントリポイント。
func main() {
	if err := runFn(); err != nil {
		fatalf("connect: %v", err)
	}
}
