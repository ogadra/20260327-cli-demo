package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// saveAndRestore は全てのパッケージレベル変数を退避し、テスト終了時に復元する。
func saveAndRestore(t *testing.T) {
	t.Helper()
	origStdout := stdout
	origAddr := addr
	origServe := serve
	origFatalf := fatalf
	t.Cleanup(func() {
		stdout = origStdout
		addr = origAddr
		serve = origServe
		fatalf = origFatalf
	})
}

// TestHealthEndpoint は GET /health が 200 OK を返すことを検証する。
func TestHealthEndpoint(t *testing.T) {
	r := newRouter()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

// TestRunSuccess は run がサーバー起動成功時に nil を返すことを検証する。
func TestRunSuccess(t *testing.T) {
	saveAndRestore(t)

	var buf bytes.Buffer
	stdout = &buf
	addr = ":9999"
	serve = func(a string, handler http.Handler) error {
		return nil
	}

	err := run()

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if !strings.Contains(buf.String(), "broker listening on :9999") {
		t.Errorf("expected listening message, got %q", buf.String())
	}
}

// TestRunServerError は run がサーバー起動失敗時にエラーを返すことを検証する。
func TestRunServerError(t *testing.T) {
	saveAndRestore(t)

	stdout = io.Discard
	serve = func(a string, handler http.Handler) error {
		return errors.New("bind failed")
	}

	err := run()

	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "bind failed") {
		t.Errorf("expected bind failed error, got %v", err)
	}
}

// TestMainSuccess は main がサーバー起動成功時に正常終了することを検証する。
func TestMainSuccess(t *testing.T) {
	saveAndRestore(t)

	stdout = io.Discard
	serve = func(a string, handler http.Handler) error {
		return nil
	}

	main()
}

// TestMainServerError は main がサーバー起動失敗時に fatalf を呼ぶことを検証する。
func TestMainServerError(t *testing.T) {
	saveAndRestore(t)

	stdout = io.Discard
	serve = func(a string, handler http.Handler) error {
		return errors.New("port in use")
	}

	var got string
	fatalf = func(format string, args ...any) {
		got = fmt.Sprintf(format, args...)
	}

	main()

	if !strings.Contains(got, "port in use") {
		t.Errorf("expected fatalf called with error, got %q", got)
	}
}
