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
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// newRouter は broker の HTTP ルーティングを構成した gin.Engine を返す。
func newRouter() *gin.Engine {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "ok\n")
	})
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

// run はサーバーの起動とグレースフルシャットダウンを行う。
func run() error {
	r := newRouter()

	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	errCh := make(chan error, 1)
	go func() {
		fmt.Fprintf(stdout, "broker listening on %s\n", addr)
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
