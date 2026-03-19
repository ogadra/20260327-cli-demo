// Package main は broker サービスのエントリポイントを提供する。
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

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

// listenAndServe はサーバーを起動する関数の型。テスト時に差し替える。
type listenAndServe func(addr string, handler http.Handler) error

// stdout はメインの出力先。テスト時に差し替える。
var stdout io.Writer = os.Stdout

// addr はサーバーのリッスンアドレス。テスト時に差し替える。
var addr = ":8080"

// serve はサーバーを起動する関数。テスト時に差し替える。
var serve listenAndServe = http.ListenAndServe

// fatalf はエラー時の終了処理。テスト時に差し替える。
var fatalf = log.Fatalf

// run はサーバーの起動処理を行う。
func run() error {
	r := newRouter()
	fmt.Fprintf(stdout, "broker listening on %s\n", addr)
	return serve(addr, r)
}

// main は broker の HTTP サーバーを起動する。
func main() {
	if err := run(); err != nil {
		fatalf("server error: %v", err)
	}
}
