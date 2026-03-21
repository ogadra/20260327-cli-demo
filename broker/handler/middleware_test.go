// Package handler は broker の HTTP ハンドラーのテストを提供する。
package handler

import (
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestRequestIDMiddleware_SetsHeaderAndContext はミドルウェアがヘッダーとコンテキストに RequestID をセットすることを検証する。
func TestRequestIDMiddleware_SetsHeaderAndContext(t *testing.T) {
	t.Parallel()
	r := gin.New()
	r.Use(RequestIDMiddleware(func() (string, error) {
		return "test-request-id", nil
	}))

	var gotCtxID string
	r.GET("/test", func(c *gin.Context) {
		v, _ := c.Get(requestIDKey)
		gotCtxID = v.(string)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("X-Request-Id"); got != "test-request-id" {
		t.Errorf("X-Request-Id header = %q, want %q", got, "test-request-id")
	}
	if gotCtxID != "test-request-id" {
		t.Errorf("context requestId = %q, want %q", gotCtxID, "test-request-id")
	}
}

// TestRequestIDMiddleware_IDFnError は ID 生成関数がエラーを返す場合に 500 を返すことを検証する。
func TestRequestIDMiddleware_IDFnError(t *testing.T) {
	t.Parallel()
	r := gin.New()
	r.Use(RequestIDMiddleware(func() (string, error) {
		return "", errors.New("rand error")
	}))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

// TestDefaultIDFn は DefaultIDFn が 32 文字の hex 文字列を返すことを検証する。
func TestDefaultIDFn(t *testing.T) {
	t.Parallel()
	id, err := DefaultIDFn()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(id) != 32 {
		t.Errorf("len(id) = %d, want 32", len(id))
	}
}

// TestDefaultIDFn_Unique は DefaultIDFn が一意の値を返すことを検証する。
func TestDefaultIDFn_Unique(t *testing.T) {
	t.Parallel()
	id1, err := DefaultIDFn()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	id2, err := DefaultIDFn()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id1 == id2 {
		t.Error("expected unique request IDs")
	}
}

// TestDefaultIDFnWithReader_Success は正常な io.Reader から hex ID を生成できることを検証する。
func TestDefaultIDFnWithReader_Success(t *testing.T) {
	t.Parallel()
	id, err := defaultIDFnWithReader(rand.Reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(id) != 32 {
		t.Errorf("len(id) = %d, want 32", len(id))
	}
}

// TestDefaultIDFnWithReader_Error は io.Reader がエラーを返す場合にエラーを返すことを検証する。
func TestDefaultIDFnWithReader_Error(t *testing.T) {
	t.Parallel()
	_, err := defaultIDFnWithReader(&errorReader{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// errorReader は常にエラーを返す io.Reader。
type errorReader struct{}

// Read は常にエラーを返す。
func (e *errorReader) Read(_ []byte) (int, error) {
	return 0, errors.New("rand read error")
}

// ensure errorReader implements io.Reader.
var _ io.Reader = (*errorReader)(nil)

// ensure rand.Reader is used by default.
var _ io.Reader = rand.Reader

// TestRequestIDMiddleware_NilIDFn は idFn が nil の場合にデフォルト関数が使われることを検証する。
func TestRequestIDMiddleware_NilIDFn(t *testing.T) {
	t.Parallel()
	r := gin.New()
	r.Use(RequestIDMiddleware(nil))
	r.GET("/test", func(c *gin.Context) {
		v, exists := c.Get(requestIDKey)
		if !exists {
			t.Error("requestId not set in context")
			return
		}
		id, ok := v.(string)
		if !ok || len(id) != 32 {
			t.Errorf("expected 32 char hex id, got %q", id)
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("X-Request-Id"); len(got) != 32 {
		t.Errorf("X-Request-Id header len = %d, want 32", len(got))
	}
}
