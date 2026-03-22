package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestRegisterSuccess verifies that register returns nil
// when the broker responds with 201 on the first attempt.
func TestRegisterSuccess(t *testing.T) {
	var gotBody registerRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotBody)
		w.WriteHeader(http.StatusCreated)
	}))
	defer ts.Close()

	var logged string
	deps := registerDeps{
		brokerURL: ts.URL,
		identity:  Identity{RunnerID: "r1", PrivateURL: "http://10.0.0.1:3000"},
		httpPost:  defaultHTTPPost,
		sleep:     func(d time.Duration) {},
		logf:      func(f string, a ...any) { logged = f },
	}

	err := register(context.Background(), deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotBody.RunnerID != "r1" {
		t.Errorf("runnerId = %q, want %q", gotBody.RunnerID, "r1")
	}
	if gotBody.PrivateURL != "http://10.0.0.1:3000" {
		t.Errorf("privateUrl = %q, want %q", gotBody.PrivateURL, "http://10.0.0.1:3000")
	}
	if logged != "registered with broker: id=%s url=%s" {
		t.Errorf("unexpected log format: %q", logged)
	}
}

// TestRegisterRetryThenSuccess verifies that register retries
// when the broker returns a non-201 status and succeeds on the second attempt.
func TestRegisterRetryThenSuccess(t *testing.T) {
	var callCount atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if callCount.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer ts.Close()

	deps := registerDeps{
		brokerURL: ts.URL,
		identity:  Identity{RunnerID: "r1", PrivateURL: "http://10.0.0.1:3000"},
		httpPost:  defaultHTTPPost,
		sleep:     func(d time.Duration) {},
		logf:      func(f string, a ...any) {},
	}

	err := register(context.Background(), deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount.Load() != 2 {
		t.Errorf("call count = %d, want 2", callCount.Load())
	}
}

// TestRegisterHTTPErrorRetry verifies that register retries
// when httpPost returns an error, then succeeds.
func TestRegisterHTTPErrorRetry(t *testing.T) {
	var callCount atomic.Int32
	deps := registerDeps{
		brokerURL: "http://broker:8080",
		identity:  Identity{RunnerID: "r1", PrivateURL: "http://10.0.0.1:3000"},
		httpPost: func(ctx context.Context, url string, contentType string, body io.Reader) (*http.Response, error) {
			if callCount.Add(1) == 1 {
				return nil, errors.New("connection refused")
			}
			return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(nil)}, nil
		},
		sleep: func(d time.Duration) {},
		logf:  func(f string, a ...any) {},
	}

	err := register(context.Background(), deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount.Load() != 2 {
		t.Errorf("call count = %d, want 2", callCount.Load())
	}
}

// TestRegisterContextCanceledBeforeSleep verifies that register
// returns the context error when context is canceled before the sleep.
func TestRegisterContextCanceledBeforeSleep(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	deps := registerDeps{
		brokerURL: "http://broker:8080",
		identity:  Identity{RunnerID: "r1", PrivateURL: "http://10.0.0.1:3000"},
		httpPost: func(ctx context.Context, url string, contentType string, body io.Reader) (*http.Response, error) {
			cancel()
			return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(nil)}, nil
		},
		sleep: func(d time.Duration) {},
		logf:  func(f string, a ...any) {},
	}

	err := register(ctx, deps)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

// TestRegisterContextCanceledAfterSleep verifies that register
// returns the context error when context is canceled during the sleep.
func TestRegisterContextCanceledAfterSleep(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var callCount atomic.Int32

	deps := registerDeps{
		brokerURL: "http://broker:8080",
		identity:  Identity{RunnerID: "r1", PrivateURL: "http://10.0.0.1:3000"},
		httpPost: func(ctx context.Context, url string, contentType string, body io.Reader) (*http.Response, error) {
			callCount.Add(1)
			return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(nil)}, nil
		},
		sleep: func(d time.Duration) {
			cancel()
		},
		logf: func(f string, a ...any) {},
	}

	err := register(ctx, deps)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

// TestRegisterRequestBody verifies the exact JSON structure sent to the broker.
func TestRegisterRequestBody(t *testing.T) {
	var gotContentType string
	var gotURL string
	var gotBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		gotURL = r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer ts.Close()

	deps := registerDeps{
		brokerURL: ts.URL,
		identity:  Identity{RunnerID: "test-runner", PrivateURL: "http://10.0.0.5:3000"},
		httpPost:  defaultHTTPPost,
		sleep:     func(d time.Duration) {},
		logf:      func(f string, a ...any) {},
	}

	register(context.Background(), deps)

	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", gotContentType, "application/json")
	}
	if gotURL != "/internal/runners/register" {
		t.Errorf("URL = %q, want %q", gotURL, "/internal/runners/register")
	}

	var req registerRequest
	json.Unmarshal(gotBody, &req)
	if req.RunnerID != "test-runner" {
		t.Errorf("runnerId = %q, want %q", req.RunnerID, "test-runner")
	}
	if req.PrivateURL != "http://10.0.0.5:3000" {
		t.Errorf("privateUrl = %q, want %q", req.PrivateURL, "http://10.0.0.5:3000")
	}
}

// TestDefaultHTTPPostSuccess verifies that defaultHTTPPost sends a POST request
// with the correct content type and body.
func TestDefaultHTTPPostSuccess(t *testing.T) {
	var gotMethod string
	var gotCT string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotCT = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusCreated)
	}))
	defer ts.Close()

	resp, err := defaultHTTPPost(context.Background(), ts.URL, "application/json", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotCT != "application/json" {
		t.Errorf("Content-Type = %q, want %q", gotCT, "application/json")
	}
}

// TestDefaultHTTPPostInvalidURL verifies that defaultHTTPPost returns an error
// when the URL is malformed.
func TestDefaultHTTPPostInvalidURL(t *testing.T) {
	_, err := defaultHTTPPost(context.Background(), "://bad", "application/json", nil)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

// TestDefaultHTTPPostCanceledContext verifies that defaultHTTPPost returns an error
// when the context is already canceled.
func TestDefaultHTTPPostCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := defaultHTTPPost(ctx, "http://127.0.0.1:1/unreachable", "application/json", nil)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}
