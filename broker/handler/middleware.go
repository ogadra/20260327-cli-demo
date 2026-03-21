// Package handler は broker の HTTP ハンドラーを提供する。
package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/gin-gonic/gin"
)

// requestIDKey は gin.Context に格納する RequestID のキー。
const requestIDKey = "requestId"

// RequestIDMiddleware はリクエストごとにユニークな ID を生成しコンテキストとレスポンスヘッダーにセットするミドルウェアを返す。
// idFn が nil の場合は DefaultIDFn を使用する。
func RequestIDMiddleware(idFn func() (string, error)) gin.HandlerFunc {
	if idFn == nil {
		idFn = DefaultIDFn
	}
	return func(c *gin.Context) {
		id, err := idFn()
		if err != nil {
			c.AbortWithStatus(500)
			return
		}
		c.Set(requestIDKey, id)
		c.Header("X-Request-Id", id)
		c.Next()
	}
}

// DefaultIDFn は crypto/rand で 16 バイトのランダム値を生成し hex 32 文字の文字列を返す。
func DefaultIDFn() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate request id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
