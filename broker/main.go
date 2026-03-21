// Package main は broker サービスのエントリポイントを提供する。
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gin-gonic/gin"
	"github.com/ogadra/20260327-cli-demo/broker/handler"
	"github.com/ogadra/20260327-cli-demo/broker/service"
	"github.com/ogadra/20260327-cli-demo/broker/store"
)

// newRouter は broker の HTTP ルーティングを構成した gin.Engine を返す。
// h が nil の場合はヘルスチェックのみ登録する。
func newRouter(h *handler.Handler) *gin.Engine {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "ok\n")
	})
	if h != nil {
		r.Use(handler.RequestIDMiddleware(handler.DefaultIDFn))
		r.POST("/sessions", h.PostSessions)
		r.DELETE("/sessions/:sessionId", h.DeleteSession)
		r.GET("/resolve", h.GetResolve)
		r.POST("/internal/runners/register", h.PostRegister)
		r.DELETE("/internal/runners/:runnerId", h.DeleteRunner)
	}
	return r
}

// stdout はメインの出力先。テスト時に差し替える。
var stdout io.Writer = os.Stdout

// addr はサーバーのリッスンアドレス。テスト時に差し替える。
var addr = ":8080"

// shutdownTimeout はグレースフルシャットダウンのタイムアウト。テスト時に差し替える。
var shutdownTimeout = 5 * time.Second

// fatalf はエラー時の終了処理。テスト時に差し替える。
var fatalf = log.Fatalf

// signalNotify は os/signal.Notify のラッパー。テスト時に差し替える。
var signalNotify = signal.Notify

// initHandler は DynamoDB クライアントを初期化し Handler を生成する関数。テスト時に差し替える。
var initHandler = defaultInitHandler

// loadAWSConfig は AWS SDK の設定をロードする関数。テスト時に差し替える。
var loadAWSConfig = config.LoadDefaultConfig

// defaultInitHandler は環境変数から DynamoDB クライアントを構築し Handler を返す。
func defaultInitHandler() (*handler.Handler, error) {
	endpoint := os.Getenv("DYNAMODB_ENDPOINT")
	region := os.Getenv("AWS_REGION")
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	var missing []string
	for _, pair := range []struct{ name, val string }{
		{"DYNAMODB_ENDPOINT", endpoint},
		{"AWS_REGION", region},
		{"AWS_ACCESS_KEY_ID", accessKey},
		{"AWS_SECRET_ACCESS_KEY", secretKey},
	} {
		if pair.val == "" {
			missing = append(missing, pair.name)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	ctx := context.Background()
	cfg, err := loadAWSConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})
	repo := store.NewDynamoRepository(client, "Runners")
	svc := service.NewBrokerService(repo)
	return handler.NewHandler(svc), nil
}

// run はサーバーの起動とグレースフルシャットダウンを行う。
func run() error {
	h, err := initHandler()
	if err != nil {
		return fmt.Errorf("init handler: %w", err)
	}
	r := newRouter(h)

	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	fmt.Fprintf(stdout, "broker listening on %s\n", addr)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signalNotify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		fmt.Fprintf(stdout, "received signal %s, shutting down...\n", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return srv.Shutdown(ctx)
}

// main は broker の HTTP サーバーを起動する。
func main() {
	if err := run(); err != nil {
		fatalf("server error: %v", err)
	}
}
