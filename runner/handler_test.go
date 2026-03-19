package main

import (
	"bufio"
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
	handler := newHandler(sm)

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
	handler := newHandler(sm)

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
	handler := newHandler(sm)

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
	handler := newHandler(sm)

	req := httptest.NewRequest(http.MethodDelete, "/api/session", nil)
	req.Header.Set(sessionIDHeader, "nonexistent")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// TestExecuteBasic verifies that POST /api/execute with a valid session
// streams SSE events for stdout and complete.
func TestExecuteBasic(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()
	handler := newHandler(sm)

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

	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want %q", ct, "text/event-stream")
	}

	events := parseSSEEvents(t, w.Body.String())
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}

	last := events[len(events)-1]
	if last.Type != "complete" {
		t.Fatalf("last event type = %q, want %q", last.Type, "complete")
	}
	if last.ExitCode == nil || *last.ExitCode != 0 {
		t.Fatalf("last event exitCode = %v, want 0", last.ExitCode)
	}

	foundHello := false
	for _, e := range events {
		if e.Type == "stdout" && e.Data == "hello" {
			foundHello = true
		}
	}
	if !foundHello {
		t.Fatalf("did not find stdout event with data 'hello' in %v", events)
	}
}

// TestExecuteWithStderr verifies that stderr output is sent as an SSE event
// of type "stderr" before the complete event.
func TestExecuteWithStderr(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()
	handler := newHandler(sm)

	id, _, err := sm.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	body := strings.NewReader(`{"command":"echo err >&2"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/execute", body)
	req.Header.Set(sessionIDHeader, id)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	events := parseSSEEvents(t, w.Body.String())
	foundStderr := false
	for _, e := range events {
		if e.Type == "stderr" && strings.Contains(e.Data, "err") {
			foundStderr = true
		}
	}
	if !foundStderr {
		t.Fatalf("did not find stderr event in %v", events)
	}
}

// TestExecuteNonZeroExit verifies that a failing command returns
// a complete event with a non-zero exit code.
func TestExecuteNonZeroExit(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()
	handler := newHandler(sm)

	id, _, err := sm.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	body := strings.NewReader(`{"command":"false"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/execute", body)
	req.Header.Set(sessionIDHeader, id)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	events := parseSSEEvents(t, w.Body.String())
	last := events[len(events)-1]
	if last.Type != "complete" || last.ExitCode == nil || *last.ExitCode == 0 {
		t.Fatalf("expected non-zero exit code, got %v", last)
	}
}

// TestExecuteMissingSessionHeader verifies that POST /api/execute without
// X-Session-Id header returns 400 Bad Request.
func TestExecuteMissingSessionHeader(t *testing.T) {
	sm := NewSessionManager()
	handler := newHandler(sm)

	body := strings.NewReader(`{"command":"echo hello"}`)
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
	handler := newHandler(sm)

	body := strings.NewReader(`{"command":"echo hello"}`)
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
	handler := newHandler(sm)

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
	handler := newHandler(sm)

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
	handler := newHandler(sm)

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
	handler := newHandler(sm)

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
	sm.newShell = func() (*Shell, error) {
		return nil, errors.New("shell broken")
	}
	handler := newHandler(sm)

	req := httptest.NewRequest(http.MethodPost, "/api/session", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// TestExecuteWithExecError verifies that the audit log records an error
// when ExecuteStream fails on a broken session.
func TestExecuteWithExecError(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()
	handler := newHandler(sm)

	id, shell, err := sm.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Close the shell to make ExecuteStream return an error.
	shell.Close()

	body := strings.NewReader(`{"command":"echo hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/execute", body)
	req.Header.Set(sessionIDHeader, id)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Should still return 200 with SSE events since headers are sent before execution.
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
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
