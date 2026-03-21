package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// saveAndRestore は全てのパッケージレベル変数を退避し、テスト終了時に復元する。
func saveAndRestore(t *testing.T) {
	t.Helper()
	origStdout := stdout
	origAddr := addr
	origShutdownTimeout := shutdownTimeout
	origFatalf := fatalf
	origSignalNotify := signalNotify
	t.Cleanup(func() {
		stdout = origStdout
		addr = origAddr
		shutdownTimeout = origShutdownTimeout
		fatalf = origFatalf
		signalNotify = origSignalNotify
	})
}

// TestHealthEndpoint は GET /health が 200 OK を返すことを検証する。
func TestHealthEndpoint(t *testing.T) {
	r := newRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

// TestRunGracefulShutdown は run がシグナル受信時にグレースフルシャットダウンすることを検証する。
func TestRunGracefulShutdown(t *testing.T) {
	saveAndRestore(t)

	var buf bytes.Buffer
	stdout = &buf
	addr = ":0"
	shutdownTimeout = 1 * time.Second

	sigCh := make(chan os.Signal, 1)
	signalNotify = func(c chan<- os.Signal, _ ...os.Signal) {
		go func() {
			sig := <-sigCh
			c <- sig
		}()
	}

	done := make(chan error, 1)
	go func() {
		done <- run()
	}()

	time.Sleep(100 * time.Millisecond)
	sigCh <- os.Interrupt

	err := <-done
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if !strings.Contains(buf.String(), "broker listening on") {
		t.Errorf("expected listening message, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "shutting down") {
		t.Errorf("expected shutdown message, got %q", buf.String())
	}
}

// TestRunListenError は run がリッスン失敗時にエラーを返すことを検証する。
func TestRunListenError(t *testing.T) {
	saveAndRestore(t)

	stdout = io.Discard
	shutdownTimeout = 1 * time.Second

	// ポートを占有してリッスンエラーを発生させる
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	addr = srv.Listener.Addr().String()

	signalNotify = func(c chan<- os.Signal, _ ...os.Signal) {}

	err := run()
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// TestMainSuccess は main がサーバー起動後シグナルで正常終了することを検証する。
func TestMainSuccess(t *testing.T) {
	saveAndRestore(t)

	stdout = io.Discard
	addr = ":0"
	shutdownTimeout = 1 * time.Second

	sigCh := make(chan os.Signal, 1)
	signalNotify = func(c chan<- os.Signal, _ ...os.Signal) {
		go func() {
			sig := <-sigCh
			c <- sig
		}()
	}

	done := make(chan struct{})
	go func() {
		main()
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	sigCh <- os.Interrupt
	<-done
}

// TestMainServerError は main がサーバー起動失敗時に fatalf を呼ぶことを検証する。
func TestMainServerError(t *testing.T) {
	saveAndRestore(t)

	stdout = io.Discard
	shutdownTimeout = 1 * time.Second

	// ポートを占有してリッスンエラーを発生させる
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	addr = srv.Listener.Addr().String()

	signalNotify = func(c chan<- os.Signal, _ ...os.Signal) {}

	var got string
	fatalf = func(format string, args ...any) {
		got = fmt.Sprintf(format, args...)
	}

	main()

	if !strings.Contains(got, "server error") {
		t.Errorf("expected fatalf called with error, got %q", got)
	}
}
