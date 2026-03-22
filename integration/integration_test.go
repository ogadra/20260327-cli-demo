//go:build integration

// Package integration は全サービスを結合した統合テストを提供する。
package integration

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// sseEvent は SSE ストリームから受信する単一イベントを表す。
type sseEvent struct {
	Type     string `json:"type"`
	Data     string `json:"data,omitempty"`
	ExitCode *int   `json:"exitCode,omitempty"`
}

// sessionCookies はセッション管理に必要な cookie を保持する。
type sessionCookies struct {
	RunnerID  string
	SessionID string
}

// nginxURL はテスト対象の nginx の URL を返す。
func nginxURL(t *testing.T) string {
	t.Helper()
	u := os.Getenv("NGINX_URL")
	if u == "" {
		t.Fatal("NGINX_URL environment variable is required")
	}
	return u
}

// waitForNginx は nginx の /health エンドポイントが応答するまでポーリングする。
// 最大 60 秒間待機し、タイムアウトした場合はテストを失敗させる。
func waitForNginx(t *testing.T, baseURL string) {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatal("nginx did not become ready within 60 seconds")
}

// createSession は POST /api/session を呼び出しセッションを作成する。
// レスポンスヘッダから runner_id と session_id の cookie を抽出して返す。
func createSession(t *testing.T, baseURL string) sessionCookies {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/session", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/session: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("POST /api/session: want 204, got %d", resp.StatusCode)
	}
	var cookies sessionCookies
	for _, c := range resp.Cookies() {
		switch c.Name {
		case "runner_id":
			cookies.RunnerID = c.Value
		case "session_id":
			cookies.SessionID = c.Value
		}
	}
	if cookies.RunnerID == "" {
		t.Fatal("runner_id cookie not found in response")
	}
	if cookies.SessionID == "" {
		t.Fatal("session_id cookie not found in response")
	}
	return cookies
}

// executeCommand は POST /api/execute を呼び出し SSE イベントをパースして返す。
// Cookie ヘッダは手動で設定する。Secure フラグが HTTP 環境の cookie jar と非互換のため。
func executeCommand(t *testing.T, baseURL string, cookies sessionCookies, command string) []sseEvent {
	t.Helper()
	body := fmt.Sprintf(`{"command":%q}`, command)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/execute", strings.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", fmt.Sprintf("runner_id=%s; session_id=%s", cookies.RunnerID, cookies.SessionID))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/execute: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/execute: want 200, got %d", resp.StatusCode)
	}
	return parseSSEEvents(t, resp)
}

// parseSSEEvents は HTTP レスポンスから SSE イベントを読み取りパースする。
func parseSSEEvents(t *testing.T, resp *http.Response) []sseEvent {
	t.Helper()
	var events []sseEvent
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		var event sseEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			t.Fatalf("parse SSE event %q: %v", data, err)
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read SSE stream: %v", err)
	}
	return events
}

// deleteSession は DELETE /api/session を呼び出しセッションを削除する。
func deleteSession(t *testing.T, baseURL string, cookies sessionCookies) {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, baseURL+"/api/session", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Cookie", fmt.Sprintf("runner_id=%s; session_id=%s", cookies.RunnerID, cookies.SessionID))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE /api/session: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE /api/session: want 204, got %d", resp.StatusCode)
	}
}

// TestHealthCheck は nginx のヘルスチェックエンドポイントが正常に応答することを検証する。
func TestHealthCheck(t *testing.T) {
	base := nginxURL(t)
	waitForNginx(t, base)

	resp, err := http.Get(base + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /health: want 200, got %d", resp.StatusCode)
	}
}

// TestSmoke_CreateSessionAndExecute はセッション作成からコマンド実行までの正常系フローを検証する。
// ホワイトリストコマンド pwd を使用する。LLM バリデータが未設定の環境でも動作させるため。
func TestSmoke_CreateSessionAndExecute(t *testing.T) {
	base := nginxURL(t)
	waitForNginx(t, base)

	cookies := createSession(t, base)

	events := executeCommand(t, base, cookies, "pwd")

	var hasStdout, hasComplete bool
	for _, e := range events {
		if e.Type == "stdout" && strings.TrimSpace(e.Data) != "" {
			hasStdout = true
		}
		if e.Type == "complete" && e.ExitCode != nil && *e.ExitCode == 0 {
			hasComplete = true
		}
	}
	if !hasStdout {
		t.Errorf("stdout event not found in events: %+v", events)
	}
	if !hasComplete {
		t.Errorf("complete event with exitCode=0 not found in events: %+v", events)
	}

	deleteSession(t, base, cookies)
}
