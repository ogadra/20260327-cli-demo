// Package main は broker サービスのエントリポイントを提供する。
package main

import (
	"fmt"
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

// main は broker の HTTP サーバーを起動する。
func main() {
	r := newRouter()
	addr := ":8080"
	fmt.Fprintf(os.Stdout, "broker listening on %s\n", addr)
	if err := r.Run(addr); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
