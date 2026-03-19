package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestMainSuccess verifies that main completes without calling fatalf
// when start succeeds. It sends SIGTERM to trigger graceful shutdown.
func TestMainSuccess(t *testing.T) {
	orig := fatalf
	defer func() { fatalf = orig }()

	fatalf = func(format string, args ...any) {
		t.Fatalf("unexpected fatalf: "+format, args...)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		main()
	}()

	time.Sleep(200 * time.Millisecond)

	proc, _ := os.FindProcess(os.Getpid())
	proc.Signal(syscall.SIGTERM)

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("main did not return within 10 seconds")
	}
}

// TestMainError verifies that main calls fatalf when start returns an error,
// such as when the default port :3000 is already in use.
func TestMainError(t *testing.T) {
	// Occupy :3000 to make start fail.
	ln, err := net.Listen("tcp", ":3000")
	if err != nil {
		t.Skipf("cannot bind :3000 for test: %v", err)
	}
	defer ln.Close()

	orig := fatalf
	defer func() { fatalf = orig }()

	var called bool
	fatalf = func(format string, args ...any) {
		called = true
		// Log but don't exit.
		log.Printf("captured fatalf: "+format, args...)
	}

	main()

	if !called {
		t.Fatal("fatalf should have been called when start fails")
	}
}

// TestRunGracefulShutdown verifies that run starts the server and shuts down
// gracefully when a signal is sent on the injected channel.
func TestRunGracefulShutdown(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	addr := ln.Addr().String()

	sigCh := make(chan os.Signal, 1)
	cfg := serverConfig{
		sm:              NewSessionManager(),
		shutdownTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ln, sigCh, cfg)
	}()

	// Wait for the server to start accepting connections.
	waitForServer(t, addr)

	sigCh <- os.Interrupt

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run did not return within 10 seconds")
	}
}

// TestRunServeError verifies that run returns an error when the listener is
// closed before serving, causing Serve to fail immediately.
func TestRunServeError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	ln.Close()

	sigCh := make(chan os.Signal, 1)
	cfg := serverConfig{
		sm:              NewSessionManager(),
		shutdownTimeout: 10 * time.Second,
	}

	err = run(ln, sigCh, cfg)
	if err == nil {
		t.Fatal("run should return error when listener is closed")
	}
}

// TestRunCloseAllError verifies that run returns the CloseAll error
// when a session was manually closed before shutdown.
func TestRunCloseAllError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	addr := ln.Addr().String()

	sm := NewSessionManager()
	cfg := serverConfig{
		sm:              sm,
		shutdownTimeout: 10 * time.Second,
	}

	sigCh := make(chan os.Signal, 1)
	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ln, sigCh, cfg)
	}()

	waitForServer(t, addr)

	// Create a session and close it manually so CloseAll will fail.
	id, _, err := sm.Create()
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	shell, _ := sm.Get(id)
	shell.Close()

	sigCh <- os.Interrupt

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("run should return error when CloseAll fails")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run did not return within 10 seconds")
	}
}

// TestIntegrationCreateExecuteDelete verifies the full lifecycle of creating a session,
// executing a command, and deleting the session through the HTTP API using httptest.
func TestIntegrationCreateExecuteDelete(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()

	ts := httptest.NewServer(newHandler(sm))
	defer ts.Close()

	// Create session.
	resp, err := http.Post(ts.URL+"/api/session", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/session error: %v", err)
	}
	defer resp.Body.Close()

	var sr sessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if sr.SessionID == "" {
		t.Fatal("empty session ID")
	}

	// Execute whitelisted command.
	body := strings.NewReader(`{"command":"pwd"}`)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/execute", body)
	req.Header.Set(sessionIDHeader, sr.SessionID)
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/execute error: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp2.StatusCode, http.StatusOK)
	}

	// Delete session.
	req3, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/session", nil)
	req3.Header.Set(sessionIDHeader, sr.SessionID)
	resp3, err := http.DefaultClient.Do(req3)
	if err != nil {
		t.Fatalf("DELETE /api/session error: %v", err)
	}
	defer resp3.Body.Close()

	if resp3.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp3.StatusCode, http.StatusNoContent)
	}
}

// TestRunShutdownTimeout verifies that run returns an error when the shutdown
// context times out due to an in-flight connection that does not complete in time.
func TestRunShutdownTimeout(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	addr := ln.Addr().String()

	sm := NewSessionManager()
	cfg := serverConfig{
		sm:              sm,
		shutdownTimeout: 1 * time.Nanosecond,
	}

	sigCh := make(chan os.Signal, 1)
	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ln, sigCh, cfg)
	}()

	waitForServer(t, addr)

	// Open a kept-alive connection so the server has active connections at shutdown.
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	sigCh <- os.Interrupt

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("run should return error when shutdown times out")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run did not return within 10 seconds")
	}
}

// TestStartAndShutdown verifies that start binds to the given address and
// shuts down when SIGTERM is delivered to the process.
func TestStartAndShutdown(t *testing.T) {
	errCh := make(chan error, 1)
	go func() {
		errCh <- start("127.0.0.1:0")
	}()

	// start uses signal.Notify internally, so we send SIGTERM to the process.
	// Give the server a moment to bind.
	time.Sleep(200 * time.Millisecond)

	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("FindProcess error: %v", err)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("Signal error: %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("start returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("start did not return within 10 seconds")
	}
}

// TestStartListenError verifies that start returns an error when the address
// is already in use.
func TestStartListenError(t *testing.T) {
	// Bind a port first to cause a conflict.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()

	err = start(ln.Addr().String())
	if err == nil {
		t.Fatal("start should return error when address is already in use")
	}
}

// waitForServer polls the given address until it accepts a connection or times out.
func waitForServer(t *testing.T, addr string) {
	t.Helper()
	for i := 0; i < 50; i++ {
		resp, err := http.Post(fmt.Sprintf("http://%s/api/session", addr), "application/json", nil)
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("server did not start within 5 seconds")
}
