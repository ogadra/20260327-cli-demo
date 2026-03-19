package main

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// --- Integration test helpers ---

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
	return parseIntegrationSSEEvents(t, buf.String())
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

// parseIntegrationSSEEvents parses a raw SSE response body into a slice of sseEvent.
func parseIntegrationSSEEvents(t *testing.T, body string) []sseEvent {
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

// --- Integration tests ---

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

// TestIntegrationConcurrentExecute verifies that concurrent execute requests
// on the same session are serialized by the shell mutex and all complete
// successfully without data races or interleaved output.
func TestIntegrationConcurrentExecute(t *testing.T) {
	sm := NewSessionManager()
	defer sm.CloseAll()

	ts := httptest.NewServer(newHandler(sm))
	defer ts.Close()

	sid := createSession(t, ts)

	const n = 5
	var wg sync.WaitGroup
	wg.Add(n)
	results := make([][]sseEvent, n)
	errs := make([]error, n)

	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			body := strings.NewReader(`{"command":"pwd"}`)
			req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/execute", body)
			req.Header.Set(sessionIDHeader, sid)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				errs[i] = err
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				errs[i] = &unexpectedStatusError{got: resp.StatusCode}
				return
			}
			var buf strings.Builder
			if _, err := io.Copy(&buf, resp.Body); err != nil {
				errs[i] = err
				return
			}
			results[i] = parseIntegrationSSEEvents(t, buf.String())
		}(i)
	}
	wg.Wait()

	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("request %d error: %v", i, errs[i])
		}
		last := results[i][len(results[i])-1]
		if last.Type != "complete" || last.ExitCode == nil || *last.ExitCode != 0 {
			t.Fatalf("request %d: last event = %+v, want complete with exitCode=0", i, last)
		}
	}
}

// unexpectedStatusError is returned when an HTTP response has an unexpected status code.
type unexpectedStatusError struct {
	got int
}

// Error returns a human-readable description of the unexpected status code.
func (e *unexpectedStatusError) Error() string {
	return "unexpected status: " + http.StatusText(e.got)
}
