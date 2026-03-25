// Package healthcheck は runner の死活監視のテストを提供する。
package healthcheck

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestHTTPChecker_ImplementsChecker は HTTPChecker が Checker インターフェースを満たすことを検証する。
func TestHTTPChecker_ImplementsChecker(t *testing.T) {
	t.Parallel()
	var _ Checker = (*HTTPChecker)(nil)
}

// TestNewHTTPChecker はコンストラクタが非 nil を返すことを検証する。
func TestNewHTTPChecker(t *testing.T) {
	t.Parallel()
	c := NewHTTPChecker(http.DefaultClient)
	if c == nil {
		t.Fatal("expected non-nil checker")
	}
	if c.client != http.DefaultClient {
		t.Error("client mismatch")
	}
}

// TestHTTPChecker_Check_Healthy は runner が 200 を返す場合に nil を返すことを検証する。
func TestHTTPChecker_Check_Healthy(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/health")
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want %q", r.Method, http.MethodGet)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	checker := NewHTTPChecker(srv.Client())
	err := checker.Check(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestHTTPChecker_Check_Unhealthy は runner が 500 を返す場合にエラーを返すことを検証する。
func TestHTTPChecker_Check_Unhealthy(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	checker := NewHTTPChecker(srv.Client())
	err := checker.Check(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error for unhealthy runner")
	}
	if !strings.Contains(err.Error(), "unexpected status 500") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "unexpected status 500")
	}
}

// TestHTTPChecker_Check_Unreachable は runner に接続できない場合にエラーを返すことを検証する。
func TestHTTPChecker_Check_Unreachable(t *testing.T) {
	t.Parallel()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	client := &http.Client{Timeout: 1 * time.Second}
	checker := NewHTTPChecker(client)
	err = checker.Check(context.Background(), "http://"+addr)
	if err == nil {
		t.Fatal("expected error for unreachable runner")
	}
	if !strings.Contains(err.Error(), "request failed") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "request failed")
	}
}

// TestNewHTTPChecker_NilClient は nil を渡した場合に http.DefaultClient にフォールバックすることを検証する。
func TestNewHTTPChecker_NilClient(t *testing.T) {
	t.Parallel()
	c := NewHTTPChecker(nil)
	if c == nil {
		t.Fatal("expected non-nil checker")
	}
	if c.client == nil {
		t.Fatal("expected non-nil client, got nil")
	}
	if c.client != http.DefaultClient {
		t.Error("expected http.DefaultClient as fallback")
	}
}

// TestHTTPChecker_Check_InvalidURL は不正な URL の場合にエラーを返すことを検証する。
func TestHTTPChecker_Check_InvalidURL(t *testing.T) {
	t.Parallel()
	checker := NewHTTPChecker(http.DefaultClient)
	err := checker.Check(context.Background(), "://invalid")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	if !strings.Contains(err.Error(), "create request") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "create request")
	}
}

// TestHTTPChecker_Check_ContextCanceled はコンテキストキャンセル時にエラーを返すことを検証する。
func TestHTTPChecker_Check_ContextCanceled(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	checker := NewHTTPChecker(srv.Client())
	err := checker.Check(ctx, srv.URL)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}
