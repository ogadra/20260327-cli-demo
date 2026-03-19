package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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
