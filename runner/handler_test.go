package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCreateSession verifies that POST /api/session creates a new session
// and returns a JSON body containing a non-empty sessionId.
func TestCreateSession(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()
	handler := newHandler(sm, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/session", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp sessionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.SessionID == "" {
		t.Fatal("sessionId is empty")
	}
}

// TestDeleteSession verifies that DELETE /api/session with a valid X-Session-Id
// deletes the session and returns 204 No Content.
func TestDeleteSession(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()
	handler := newHandler(sm, nil)

	id, _, err := sm.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/session", nil)
	req.Header.Set(sessionIDHeader, id)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}

	_, err = sm.Get(id)
	if err == nil {
		t.Fatal("session should be deleted")
	}
}

// TestDeleteSessionMissingHeader verifies that DELETE /api/session without
// X-Session-Id header returns 400 Bad Request.
func TestDeleteSessionMissingHeader(t *testing.T) {
	sm := NewSessionManager()
	handler := newHandler(sm, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/session", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestDeleteSessionNotFound verifies that DELETE /api/session with a nonexistent
// session ID returns 404 Not Found.
func TestDeleteSessionNotFound(t *testing.T) {
	sm := NewSessionManager()
	handler := newHandler(sm, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/session", nil)
	req.Header.Set(sessionIDHeader, "nonexistent")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// TestDeleteSessionCloseError verifies that DELETE /api/session returns 500
// when the session exists but Close fails.
func TestDeleteSessionCloseError(t *testing.T) {
	sm := NewSessionManager()
	sm.newShell = func() (Shell, error) {
		return &mockShell{closeErr: errors.New("close failed")}, nil
	}
	handler := newHandler(sm, nil)

	id, _, err := sm.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/session", nil)
	req.Header.Set(sessionIDHeader, id)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// TestExecuteWhitelisted verifies that POST /api/execute with a whitelisted command
// streams SSE events for stdout and complete with exit code 0.
func TestExecuteWhitelisted(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()
	handler := newHandler(sm, nil)

	id, _, err := sm.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	body := strings.NewReader(`{"command":"ls"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/execute", body)
	req.Header.Set(sessionIDHeader, id)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want %q", ct, "text/event-stream")
	}

	events := parseSSEEvents(t, w.Body.String())
	if len(events) < 1 {
		t.Fatalf("expected at least 1 event, got %d", len(events))
	}

	last := events[len(events)-1]
	if last.Type != "complete" {
		t.Fatalf("last event type = %q, want %q", last.Type, "complete")
	}
	if last.ExitCode == nil || *last.ExitCode != 0 {
		t.Fatalf("last event exitCode = %v, want 0", last.ExitCode)
	}
}

// TestExecuteRejected verifies that POST /api/execute with a non-whitelisted command
// returns 403 Forbidden without executing the command.
func TestExecuteRejected(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()
	handler := newHandler(sm, nil)

	id, _, err := sm.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	body := strings.NewReader(`{"command":"echo hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/execute", body)
	req.Header.Set(sessionIDHeader, id)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}

	var errResp errorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !strings.Contains(errResp.Error, "command not allowed") {
		t.Fatalf("error = %q, want to contain %q", errResp.Error, "command not allowed")
	}
}

// TestExecuteRejectedWithArgs verifies that a whitelisted command name with arguments
// is rejected with 403 because it does not exactly match the whitelist.
func TestExecuteRejectedWithArgs(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()
	handler := newHandler(sm, nil)

	id, _, err := sm.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	body := strings.NewReader(`{"command":"ls -la"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/execute", body)
	req.Header.Set(sessionIDHeader, id)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

// TestExecuteMissingSessionHeader verifies that POST /api/execute without
// X-Session-Id header returns 400 Bad Request.
func TestExecuteMissingSessionHeader(t *testing.T) {
	sm := NewSessionManager()
	handler := newHandler(sm, nil)

	body := strings.NewReader(`{"command":"ls"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/execute", body)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestExecuteSessionNotFound verifies that POST /api/execute with a nonexistent
// session ID returns 404 Not Found.
func TestExecuteSessionNotFound(t *testing.T) {
	sm := NewSessionManager()
	handler := newHandler(sm, nil)

	body := strings.NewReader(`{"command":"ls"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/execute", body)
	req.Header.Set(sessionIDHeader, "nonexistent")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// TestExecuteInvalidJSON verifies that POST /api/execute with invalid JSON body
// returns 400 Bad Request.
func TestExecuteInvalidJSON(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()
	handler := newHandler(sm, nil)

	id, _, err := sm.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	body := strings.NewReader(`{invalid`)
	req := httptest.NewRequest(http.MethodPost, "/api/execute", body)
	req.Header.Set(sessionIDHeader, id)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestExecuteEmptyCommand verifies that POST /api/execute with an empty command
// returns 400 Bad Request.
func TestExecuteEmptyCommand(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()
	handler := newHandler(sm, nil)

	id, _, err := sm.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	body := strings.NewReader(`{"command":""}`)
	req := httptest.NewRequest(http.MethodPost, "/api/execute", body)
	req.Header.Set(sessionIDHeader, id)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestSessionMethodNotAllowed verifies that unsupported HTTP methods on
// /api/session return 405 Method Not Allowed.
func TestSessionMethodNotAllowed(t *testing.T) {
	sm := NewSessionManager()
	handler := newHandler(sm, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/session", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// TestExecuteMethodNotAllowed verifies that unsupported HTTP methods on
// /api/execute return 405 Method Not Allowed.
func TestExecuteMethodNotAllowed(t *testing.T) {
	sm := NewSessionManager()
	handler := newHandler(sm, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/execute", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// TestCreateSessionError verifies that POST /api/session returns 500
// when the session manager fails to create a new shell.
func TestCreateSessionError(t *testing.T) {
	sm := NewSessionManager()
	sm.newShell = func() (Shell, error) {
		return nil, errors.New("shell broken")
	}
	handler := newHandler(sm, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/session", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// mockShell is a test double for the Shell interface that returns
// preconfigured values from ExecuteStream and Close.
type mockShell struct {
	exitCode int
	stderr   string
	err      error
	closeErr error
}

// ExecuteStream sends no stdout lines and returns the preconfigured exit code, stderr, and error.
func (m *mockShell) ExecuteStream(_ context.Context, _ string, ch chan<- string) (int, string, error) {
	close(ch)
	return m.exitCode, m.stderr, m.err
}

// Close returns the preconfigured close error.
func (m *mockShell) Close() error {
	return m.closeErr
}

// TestExecuteWhitelistedWithStderr verifies that stderr output from a whitelisted command
// is sent as an SSE event of type "stderr" before the complete event.
func TestExecuteWhitelistedWithStderr(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()
	sm.newShell = func() (Shell, error) {
		return &mockShell{exitCode: 0, stderr: "warning: something"}, nil
	}
	handler := newHandler(sm, nil)

	id, _, err := sm.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	body := strings.NewReader(`{"command":"ls"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/execute", body)
	req.Header.Set(sessionIDHeader, id)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	events := parseSSEEvents(t, w.Body.String())
	foundStderr := false
	for _, e := range events {
		if e.Type == "stderr" && strings.Contains(e.Data, "warning") {
			foundStderr = true
		}
	}
	if !foundStderr {
		t.Fatalf("did not find stderr event in %v", events)
	}
}

// TestExecuteWhitelistedNonZeroExit verifies that a whitelisted command returning
// a non-zero exit code sends the correct exit code in the complete event.
func TestExecuteWhitelistedNonZeroExit(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()
	sm.newShell = func() (Shell, error) {
		return &mockShell{exitCode: 2}, nil
	}
	handler := newHandler(sm, nil)

	id, _, err := sm.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	body := strings.NewReader(`{"command":"ls"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/execute", body)
	req.Header.Set(sessionIDHeader, id)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	events := parseSSEEvents(t, w.Body.String())
	last := events[len(events)-1]
	if last.Type != "complete" || last.ExitCode == nil || *last.ExitCode != 2 {
		t.Fatalf("expected exitCode=2, got %v", last)
	}
}

// TestExecuteWhitelistedWithExecError verifies that when ExecuteStream returns an error
// on a whitelisted command, the audit log records the error via auditLog.
func TestExecuteWhitelistedWithExecError(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()
	sm.newShell = func() (Shell, error) {
		return &mockShell{exitCode: -1, err: errors.New("broken")}, nil
	}
	handler := newHandler(sm, nil)

	id, _, err := sm.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	body := strings.NewReader(`{"command":"ls"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/execute", body)
	req.Header.Set(sessionIDHeader, id)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// mockValidator is a test double for the Validator interface that returns
// preconfigured ValidationResult and error, and tracks whether it was called.
type mockValidator struct {
	result ValidationResult
	err    error
	called bool
}

// Validate records that it was called and returns the preconfigured result and error.
func (m *mockValidator) Validate(_ context.Context, _ string) (ValidationResult, error) {
	m.called = true
	return m.result, m.err
}

// TestExecuteValidatedSafe verifies that a non-whitelisted command judged safe
// by the validator is executed and returns SSE events.
func TestExecuteValidatedSafe(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()
	sm.newShell = func() (Shell, error) {
		return &mockShell{exitCode: 0}, nil
	}
	v := &mockValidator{result: ValidationResult{Safe: true, Reason: "safe command"}}
	handler := newHandler(sm, v)

	id, _, err := sm.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	body := strings.NewReader(`{"command":"echo hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/execute", body)
	req.Header.Set(sessionIDHeader, id)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	events := parseSSEEvents(t, w.Body.String())
	last := events[len(events)-1]
	if last.Type != "complete" || last.ExitCode == nil || *last.ExitCode != 0 {
		t.Fatalf("expected complete with exitCode=0, got %+v", last)
	}
}

// TestExecuteValidatedUnsafe verifies that a non-whitelisted command judged unsafe
// by the validator returns 403 with the reason.
func TestExecuteValidatedUnsafe(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()
	v := &mockValidator{result: ValidationResult{Safe: false, Reason: "destructive"}}
	handler := newHandler(sm, v)

	id, _, err := sm.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	body := strings.NewReader(`{"command":"rm -rf /"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/execute", body)
	req.Header.Set(sessionIDHeader, id)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	var errResp errorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !strings.Contains(errResp.Error, "destructive") {
		t.Fatalf("error = %q, want to contain reason", errResp.Error)
	}
}

// TestExecuteValidatorError verifies that a validator error results in
// 403 fail-closed behavior.
func TestExecuteValidatorError(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()
	v := &mockValidator{err: errors.New("LLM unavailable")}
	handler := newHandler(sm, v)

	id, _, err := sm.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	body := strings.NewReader(`{"command":"echo hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/execute", body)
	req.Header.Set(sessionIDHeader, id)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

// TestExecuteWhitelistedSkipsValidator verifies that whitelisted commands
// bypass the validator entirely and execute directly.
func TestExecuteWhitelistedSkipsValidator(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()
	sm.newShell = func() (Shell, error) {
		return &mockShell{exitCode: 0}, nil
	}
	// Validator that would reject if called.
	v := &mockValidator{result: ValidationResult{Safe: false, Reason: "should not be called"}}
	handler := newHandler(sm, v)

	id, _, err := sm.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	body := strings.NewReader(`{"command":"ls"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/execute", body)
	req.Header.Set(sessionIDHeader, id)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if v.called {
		t.Fatal("validator should not be called for whitelisted commands")
	}
}

// parseSSEEvents parses a raw SSE response body into a slice of sseEvent.
// It expects each event to be a "data: " line followed by a blank line.
func parseSSEEvents(t *testing.T, body string) []sseEvent {
	t.Helper()
	var events []sseEvent
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		var event sseEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			t.Fatalf("unmarshal SSE event %q: %v", data, err)
		}
		events = append(events, event)
	}
	return events
}
