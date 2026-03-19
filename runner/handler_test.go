package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
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

// TestDeleteSessionCloseError verifies that DELETE /api/session returns 500
// when the session exists but Close fails.
func TestDeleteSessionCloseError(t *testing.T) {
	sm := NewSessionManager()
	sm.newShell = func() (Shell, error) {
		return &mockShell{closeErr: errors.New("close failed")}, nil
	}
	handler := newHandler(sm)

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
	handler := newHandler(sm)

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

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}

	if !strings.Contains(w.Body.String(), "command not allowed") {
		t.Fatalf("body = %q, want to contain %q", w.Body.String(), "command not allowed")
	}
}

// TestExecuteRejectedWithArgs verifies that a whitelisted command name with arguments
// is rejected with 403 because it does not exactly match the whitelist.
func TestExecuteRejectedWithArgs(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()
	handler := newHandler(sm)

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
	handler := newHandler(sm)

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
	handler := newHandler(sm)

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
	sm.newShell = func() (Shell, error) {
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
	handler := newHandler(sm)

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
	handler := newHandler(sm)

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
	handler := newHandler(sm)

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

// --- Integration tests ---
// The following tests exercise the full HTTP stack using httptest.NewServer
// with real bash sessions to verify end-to-end behavior.

// TestIntegrationExecuteSSEResponse verifies that executing a whitelisted command
// through the full HTTP stack returns valid SSE events including stdout output
// and a complete event with exitCode 0.
func TestIntegrationExecuteSSEResponse(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()

	ts := httptest.NewServer(newHandler(sm))
	defer ts.Close()

	sid := createSession(t, ts)
	events := executeCommand(t, ts, sid, "pwd")

	if len(events) < 2 {
		t.Fatalf("expected at least 2 events (stdout + complete), got %d", len(events))
	}

	stdout := firstStdoutData(t, events)
	if stdout == "" {
		t.Fatal("expected non-empty stdout from pwd")
	}

	last := events[len(events)-1]
	if last.Type != "complete" || last.ExitCode == nil || *last.ExitCode != 0 {
		t.Fatalf("last event = %+v, want complete with exitCode=0", last)
	}
}

// TestIntegrationSessionPersistence verifies that state persists across
// multiple execute calls within the same session by checking that pwd
// returns the same directory consistently.
func TestIntegrationSessionPersistence(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()

	ts := httptest.NewServer(newHandler(sm))
	defer ts.Close()

	sid := createSession(t, ts)

	events1 := executeCommand(t, ts, sid, "pwd")
	events2 := executeCommand(t, ts, sid, "pwd")

	dir1 := firstStdoutData(t, events1)
	dir2 := firstStdoutData(t, events2)
	if dir1 != dir2 {
		t.Fatalf("session state not persistent: pwd returned %q then %q", dir1, dir2)
	}
}

// TestIntegrationRejectedCommand verifies that a non-whitelisted command
// returns 403 Forbidden through the full HTTP stack.
func TestIntegrationRejectedCommand(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()

	ts := httptest.NewServer(newHandler(sm))
	defer ts.Close()

	sid := createSession(t, ts)

	body := strings.NewReader(`{"command":"echo hello"}`)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/execute", body)
	req.Header.Set(sessionIDHeader, sid)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/execute error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

// TestIntegrationExecuteAfterDelete verifies that executing a command on a
// deleted session returns 404 Not Found.
func TestIntegrationExecuteAfterDelete(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()

	ts := httptest.NewServer(newHandler(sm))
	defer ts.Close()

	sid := createSession(t, ts)

	// Delete the session.
	delReq, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/session", nil)
	delReq.Header.Set(sessionIDHeader, sid)
	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatalf("DELETE error: %v", err)
	}
	delResp.Body.Close()

	// Execute on the deleted session.
	body := strings.NewReader(`{"command":"ls"}`)
	execReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/execute", body)
	execReq.Header.Set(sessionIDHeader, sid)
	execResp, err := http.DefaultClient.Do(execReq)
	if err != nil {
		t.Fatalf("POST /api/execute error: %v", err)
	}
	defer execResp.Body.Close()

	if execResp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", execResp.StatusCode, http.StatusNotFound)
	}
}

// TestIntegrationSessionIsolation verifies that two sessions have independent
// bash processes by confirming both return the same initial working directory
// without interfering with each other.
func TestIntegrationSessionIsolation(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()

	ts := httptest.NewServer(newHandler(sm))
	defer ts.Close()

	sid1 := createSession(t, ts)
	sid2 := createSession(t, ts)

	events1 := executeCommand(t, ts, sid1, "pwd")
	events2 := executeCommand(t, ts, sid2, "pwd")

	dir1 := firstStdoutData(t, events1)
	dir2 := firstStdoutData(t, events2)
	if dir1 == "" || dir2 == "" {
		t.Fatalf("expected non-empty pwd output, got %q and %q", dir1, dir2)
	}
	if dir1 != dir2 {
		t.Fatalf("expected same initial pwd, got %q and %q", dir1, dir2)
	}
}

// TestIntegrationCreateDeleteLifecycle verifies the full lifecycle of
// creating a session, executing a command, and deleting the session.
func TestIntegrationCreateDeleteLifecycle(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()

	ts := httptest.NewServer(newHandler(sm))
	defer ts.Close()

	// Create.
	sid := createSession(t, ts)

	// Execute.
	events := executeCommand(t, ts, sid, "ls")
	last := events[len(events)-1]
	if last.Type != "complete" || last.ExitCode == nil || *last.ExitCode != 0 {
		t.Fatalf("expected complete with exitCode=0, got %+v", last)
	}

	// Delete.
	delReq, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/session", nil)
	delReq.Header.Set(sessionIDHeader, sid)
	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatalf("DELETE error: %v", err)
	}
	delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", delResp.StatusCode, http.StatusNoContent)
	}

	// Verify session is gone.
	body := strings.NewReader(`{"command":"ls"}`)
	execReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/execute", body)
	execReq.Header.Set(sessionIDHeader, sid)
	execResp, err := http.DefaultClient.Do(execReq)
	if err != nil {
		t.Fatalf("POST /api/execute error: %v", err)
	}
	defer execResp.Body.Close()
	if execResp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d after delete", execResp.StatusCode, http.StatusNotFound)
	}
}

// createSession is a test helper that creates a new session via the HTTP API
// and returns its ID.
func createSession(t *testing.T, ts *httptest.Server) string {
	t.Helper()
	resp, err := http.Post(ts.URL+"/api/session", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/session error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create session status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var sr sessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	return sr.SessionID
}

// executeCommand is a test helper that executes a whitelisted command via the
// HTTP API and returns the parsed SSE events.
func executeCommand(t *testing.T, ts *httptest.Server, sessionID, command string) []sseEvent {
	t.Helper()
	body := strings.NewReader(`{"command":"` + command + `"}`)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/execute", body)
	req.Header.Set(sessionIDHeader, sessionID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/execute error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("execute status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var buf strings.Builder
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		t.Fatalf("read body error: %v", err)
	}
	return parseSSEEvents(t, buf.String())
}

// firstStdoutData is a test helper that returns the Data field of the first
// stdout event in the given events slice.
func firstStdoutData(t *testing.T, events []sseEvent) string {
	t.Helper()
	for _, e := range events {
		if e.Type == "stdout" {
			return e.Data
		}
	}
	t.Fatal("no stdout event found")
	return ""
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
