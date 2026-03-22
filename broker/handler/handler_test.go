// Package handler は broker の HTTP ハンドラーのテストを提供する。
package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ogadra/20260327-cli-demo/broker/service"
	"github.com/ogadra/20260327-cli-demo/broker/store"
)

// mockService は service.Service のモック実装。
type mockService struct {
	closeSessionFn     func(ctx context.Context, sessionID string) error
	resolveSessionFn   func(ctx context.Context, sessionID string) (*service.ResolveResult, error)
	registerRunnerFn   func(ctx context.Context, runnerID, privateURL string) error
	deregisterRunnerFn func(ctx context.Context, runnerID string) error
}

// CloseSession はモック CloseSession を呼び出す。
func (m *mockService) CloseSession(ctx context.Context, sessionID string) error {
	return m.closeSessionFn(ctx, sessionID)
}

// ResolveSession はモック ResolveSession を呼び出す。
func (m *mockService) ResolveSession(ctx context.Context, sessionID string) (*service.ResolveResult, error) {
	return m.resolveSessionFn(ctx, sessionID)
}

// RegisterRunner はモック RegisterRunner を呼び出す。
func (m *mockService) RegisterRunner(ctx context.Context, runnerID, privateURL string) error {
	return m.registerRunnerFn(ctx, runnerID, privateURL)
}

// DeregisterRunner はモック DeregisterRunner を呼び出す。
func (m *mockService) DeregisterRunner(ctx context.Context, runnerID string) error {
	return m.deregisterRunnerFn(ctx, runnerID)
}

// newTestRouter はテスト用のルーターを構築する。
func newTestRouter(h *Handler) *gin.Engine {
	r := gin.New()
	r.Use(RequestIDMiddleware(func() (string, error) {
		return "test-req-id", nil
	}))
	r.DELETE("/sessions/:sessionId", h.DeleteSession)
	r.GET("/resolve", h.GetResolve)
	r.POST("/internal/runners/register", h.PostRegister)
	r.DELETE("/internal/runners/:runnerId", h.DeleteRunner)
	return r
}

// TestDeleteSession_Success はセッション終了の成功を検証する。
func TestDeleteSession_Success(t *testing.T) {
	t.Parallel()
	h := NewHandler(&mockService{
		closeSessionFn: func(_ context.Context, sessionID string) error {
			if sessionID != "sess-abc" {
				t.Errorf("sessionID = %q, want %q", sessionID, "sess-abc")
			}
			return nil
		},
	})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/sessions/sess-abc", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

// TestDeleteSession_NotFound はセッションが見つからない場合に 404 を返すことを検証する。
func TestDeleteSession_NotFound(t *testing.T) {
	t.Parallel()
	h := NewHandler(&mockService{
		closeSessionFn: func(_ context.Context, _ string) error {
			return store.ErrNotFound
		},
	})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/sessions/sess-missing", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// TestDeleteSession_InternalError は内部エラー時に 500 を返すことを検証する。
func TestDeleteSession_InternalError(t *testing.T) {
	t.Parallel()
	h := NewHandler(&mockService{
		closeSessionFn: func(_ context.Context, _ string) error {
			return errors.New("unexpected")
		},
	})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/sessions/sess-abc", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

// TestGetResolve_ExistingSession は既存セッションの解決成功を検証する。
func TestGetResolve_ExistingSession(t *testing.T) {
	t.Parallel()
	h := NewHandler(&mockService{
		resolveSessionFn: func(_ context.Context, sessionID string) (*service.ResolveResult, error) {
			if sessionID != "sess-abc" {
				t.Errorf("sessionID = %q, want %q", sessionID, "sess-abc")
			}
			return &service.ResolveResult{SessionID: "sess-abc", RunnerURL: "http://10.0.0.1:8080", Created: false}, nil
		},
	})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/resolve", nil)
	req.AddCookie(&http.Cookie{Name: "runner_id", Value: "sess-abc"})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("X-Runner-Url"); got != "http://10.0.0.1:8080" {
		t.Errorf("X-Runner-Url = %q, want %q", got, "http://10.0.0.1:8080")
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == "runner_id" {
			t.Error("should not set runner_id cookie for existing session")
		}
	}
}

// TestGetResolve_MissingCookie_CreatesSession は cookie がない場合にセッションを新規作成することを検証する。
func TestGetResolve_MissingCookie_CreatesSession(t *testing.T) {
	t.Parallel()
	h := NewHandler(&mockService{
		resolveSessionFn: func(_ context.Context, sessionID string) (*service.ResolveResult, error) {
			if sessionID != "" {
				t.Errorf("sessionID = %q, want empty", sessionID)
			}
			return &service.ResolveResult{SessionID: "new-sess", RunnerURL: "http://10.0.0.2:8080", Created: true}, nil
		},
	})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/resolve", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("X-Runner-Url"); got != "http://10.0.0.2:8080" {
		t.Errorf("X-Runner-Url = %q, want %q", got, "http://10.0.0.2:8080")
	}
	found := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == "runner_id" && c.Value == "new-sess" {
			found = true
		}
	}
	if !found {
		t.Error("expected runner_id cookie for new session")
	}
}

// TestGetResolve_NoIdleRunner は idle runner がない場合に 503 を返すことを検証する。
func TestGetResolve_NoIdleRunner(t *testing.T) {
	t.Parallel()
	h := NewHandler(&mockService{
		resolveSessionFn: func(_ context.Context, _ string) (*service.ResolveResult, error) {
			return nil, store.ErrNoIdleRunner
		},
	})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/resolve", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

// TestGetResolve_InternalError は内部エラー時に 500 を返すことを検証する。
func TestGetResolve_InternalError(t *testing.T) {
	t.Parallel()
	h := NewHandler(&mockService{
		resolveSessionFn: func(_ context.Context, _ string) (*service.ResolveResult, error) {
			return nil, errors.New("unexpected")
		},
	})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/resolve", nil)
	req.AddCookie(&http.Cookie{Name: "runner_id", Value: "sess-abc"})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

// TestPostRegister_Success は runner 登録の成功を検証する。
func TestPostRegister_Success(t *testing.T) {
	t.Parallel()
	h := NewHandler(&mockService{
		registerRunnerFn: func(_ context.Context, runnerID, privateURL string) error {
			if runnerID != "r1" {
				t.Errorf("runnerID = %q, want %q", runnerID, "r1")
			}
			if privateURL != "http://10.0.0.1:8080" {
				t.Errorf("privateURL = %q, want %q", privateURL, "http://10.0.0.1:8080")
			}
			return nil
		},
	})
	r := newTestRouter(h)

	body := strings.NewReader(`{"runnerId":"r1","privateUrl":"http://10.0.0.1:8080"}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/runners/register", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

// TestPostRegister_InvalidBody はリクエストボディが不正な場合に 400 を返すことを検証する。
func TestPostRegister_InvalidBody(t *testing.T) {
	t.Parallel()
	h := NewHandler(&mockService{})
	r := newTestRouter(h)

	body := strings.NewReader(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/runners/register", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// TestPostRegister_InternalError は内部エラー時に 500 を返すことを検証する。
func TestPostRegister_InternalError(t *testing.T) {
	t.Parallel()
	h := NewHandler(&mockService{
		registerRunnerFn: func(_ context.Context, _, _ string) error {
			return errors.New("unexpected")
		},
	})
	r := newTestRouter(h)

	body := strings.NewReader(`{"runnerId":"r1","privateUrl":"http://10.0.0.1:8080"}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/runners/register", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

// TestDeleteRunner_Success は runner 削除の成功を検証する。
func TestDeleteRunner_Success(t *testing.T) {
	t.Parallel()
	h := NewHandler(&mockService{
		deregisterRunnerFn: func(_ context.Context, runnerID string) error {
			if runnerID != "r1" {
				t.Errorf("runnerID = %q, want %q", runnerID, "r1")
			}
			return nil
		},
	})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/internal/runners/r1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

// TestDeleteRunner_InternalError は内部エラー時に 500 を返すことを検証する。
func TestDeleteRunner_InternalError(t *testing.T) {
	t.Parallel()
	h := NewHandler(&mockService{
		deregisterRunnerFn: func(_ context.Context, _ string) error {
			return errors.New("unexpected")
		},
	})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/internal/runners/r1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

// TestWriteError_IncludesRequestID はエラーレスポンスに requestId が含まれることを検証する。
func TestWriteError_IncludesRequestID(t *testing.T) {
	t.Parallel()
	h := NewHandler(&mockService{
		resolveSessionFn: func(_ context.Context, _ string) (*service.ResolveResult, error) {
			return nil, errors.New("unexpected")
		},
	})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/resolve", nil)
	req.AddCookie(&http.Cookie{Name: "runner_id", Value: "sess-missing"})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "test-req-id") {
		t.Errorf("body = %q, want to contain %q", rec.Body.String(), "test-req-id")
	}
}

// TestNewHandler は NewHandler のコンストラクタを検証する。
func TestNewHandler(t *testing.T) {
	t.Parallel()
	svc := &mockService{}
	h := NewHandler(svc)
	if h.svc != svc {
		t.Error("svc mismatch")
	}
}

// TestNewHandler_NilPanics は NewHandler に nil を渡すと panic することを検証する。
func TestNewHandler_NilPanics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil service")
		}
	}()
	NewHandler(nil)
}

// TestPostRegister_InvalidURL は不正な URL 形式の場合に 400 を返すことを検証する。
func TestPostRegister_InvalidURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		url  string
	}{
		{"no scheme", "10.0.0.1:8080"},
		{"ftp scheme", "ftp://10.0.0.1:8080"},
		{"no host", "http://"},
		{"relative path", "/runner"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := NewHandler(&mockService{
				registerRunnerFn: func(_ context.Context, _, _ string) error {
					t.Fatal("service should not be called for invalid URL")
					return nil
				},
			})
			r := newTestRouter(h)

			body := strings.NewReader(`{"runnerId":"r1","privateUrl":"` + tt.url + `"}`)
			req := httptest.NewRequest(http.MethodPost, "/internal/runners/register", body)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d for url %q", rec.Code, http.StatusBadRequest, tt.url)
			}
		})
	}
}

// TestValidateRunnerURL は validateRunnerURL の境界値を検証する。
func TestValidateRunnerURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid http", "http://10.0.0.1:3000", false},
		{"valid https", "https://runner.local:3000", false},
		{"valid http no port", "http://runner.local", false},
		{"no scheme", "10.0.0.1:3000", true},
		{"ftp scheme", "ftp://10.0.0.1:3000", true},
		{"empty host", "http://", true},
		{"relative", "/path", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateRunnerURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRunnerURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

// TestGetResolve_CookieSecure はセッション新規作成時の runner_id cookie が Secure=true であることを検証する。
func TestGetResolve_CookieSecure(t *testing.T) {
	t.Parallel()
	h := NewHandler(&mockService{
		resolveSessionFn: func(_ context.Context, _ string) (*service.ResolveResult, error) {
			return &service.ResolveResult{SessionID: "new-sess", RunnerURL: "http://10.0.0.1:8080", Created: true}, nil
		},
	})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/resolve", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	for _, c := range rec.Result().Cookies() {
		if c.Name == "runner_id" {
			if !c.Secure {
				t.Error("expected Secure=true on runner_id cookie")
			}
			if !c.HttpOnly {
				t.Error("expected HttpOnly=true on runner_id cookie")
			}
			if c.SameSite != http.SameSiteStrictMode {
				t.Errorf("expected SameSite=Strict on runner_id cookie, got %v", c.SameSite)
			}
			return
		}
	}
	t.Error("runner_id cookie not found")
}
