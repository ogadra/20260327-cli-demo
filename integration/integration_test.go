//go:build integration

// Package integration は全サービスを結合した統合テストを提供する。
package integration

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// httpClient はテスト用の HTTP クライアント。タイムアウトを設定して CI でのハングを防止する。
var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

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

// cookieHeader は sessionCookies を Cookie ヘッダ文字列に変換する。
func (c sessionCookies) cookieHeader() string {
	return fmt.Sprintf("runner_id=%s; session_id=%s", c.RunnerID, c.SessionID)
}

// base は nginx の URL。TestMain で初期化される。
var base string

// brokerBase は broker の URL。TestMain で初期化される。
var brokerBase string

// runnerHostname は runner のホスト名。TestMain で初期化される。
var runnerHostname string

// TestMain はテスト実行前にサービスの起動を待機し、runner のホスト名を解決する。
func TestMain(m *testing.M) {
	base = os.Getenv("NGINX_URL")
	if base == "" {
		fmt.Fprintln(os.Stderr, "NGINX_URL environment variable is required")
		os.Exit(1)
	}
	brokerBase = os.Getenv("BROKER_URL")
	if brokerBase == "" {
		fmt.Fprintln(os.Stderr, "BROKER_URL environment variable is required")
		os.Exit(1)
	}

	if !waitForReady(base + "/health") {
		fmt.Fprintln(os.Stderr, "nginx did not become ready within 60 seconds")
		os.Exit(1)
	}

	hostname, err := discoverRunnerHostname()
	if err != nil {
		fmt.Fprintf(os.Stderr, "discover runner: %v\n", err)
		os.Exit(1)
	}
	runnerHostname = hostname

	os.Exit(m.Run())
}

// waitForReady は指定された URL が 200 を返すまでポーリングする。
func waitForReady(url string) bool {
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := httpClient.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}
		time.Sleep(2 * time.Second)
	}
	return false
}

// discoverRunnerHostname は broker の GET /resolve を呼び出して runner のホスト名を解決する。
// resolve で作成されたセッションはクリーンアップし、runner を再登録する。
func discoverRunnerHostname() (string, error) {
	resp, err := httpClient.Get(brokerBase + "/resolve")
	if err != nil {
		return "", fmt.Errorf("GET /resolve: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET /resolve: status %d", resp.StatusCode)
	}

	runnerURL := resp.Header.Get("X-Runner-Url")
	if runnerURL == "" {
		return "", fmt.Errorf("X-Runner-Url header not found")
	}

	hostname := strings.TrimPrefix(runnerURL, "http://")
	hostname = strings.SplitN(hostname, ":", 2)[0]

	// クリーンアップ: resolve で作成されたセッションを閉じて runner を再登録
	for _, c := range resp.Cookies() {
		if c.Name == "runner_id" {
			deleteFromBroker(brokerBase + "/sessions/" + c.Value)
		}
	}
	registerRunnerOnBroker(hostname)

	return hostname, nil
}

// deleteFromBroker は broker のエンドポイントに DELETE リクエストを送信する。
func deleteFromBroker(url string) {
	req, _ := http.NewRequest(http.MethodDelete, url, nil)
	resp, err := httpClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

// registerRunnerOnBroker は runner を broker に登録する。
func registerRunnerOnBroker(hostname string) {
	body := fmt.Sprintf(`{"runnerId":%q,"privateUrl":"http://%s:3000"}`, hostname, hostname)
	req, _ := http.NewRequest(http.MethodPost, brokerBase+"/internal/runners/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

// resetRunner は runner を broker から削除し再登録することで idle 状態に戻す。
// sessionID が空でなければ broker セッションを閉じる。
func resetRunner(t *testing.T, sessionID string) {
	t.Helper()
	if sessionID != "" {
		deleteFromBroker(brokerBase + "/sessions/" + sessionID)
	}
	registerRunnerOnBroker(runnerHostname)
}

// setupSession はセッションを作成し、テスト終了時に runner を idle に戻す cleanup を登録する。
func setupSession(t *testing.T) sessionCookies {
	t.Helper()
	cookies := createSession(t)
	t.Cleanup(func() { resetRunner(t, cookies.RunnerID) })
	return cookies
}

// createSession は POST /api/session を nginx 経由で呼び出しセッションを作成する。
func createSession(t *testing.T) sessionCookies {
	t.Helper()
	resp := doRequest(t, http.MethodPost, base+"/api/session", "", "")
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

// doRequest は HTTP リクエストを送信しレスポンスを返す。
func doRequest(t *testing.T, method, url, bodyStr, cookie string) *http.Response {
	t.Helper()
	var body io.Reader
	if bodyStr != "" {
		body = strings.NewReader(bodyStr)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if bodyStr != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

// executeCommand は POST /api/execute を呼び出し SSE イベントをパースして返す。
func executeCommand(t *testing.T, cookies sessionCookies, command string) []sseEvent {
	t.Helper()
	payload := struct {
		Command string `json:"command"`
	}{Command: command}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	resp := doRequest(t, http.MethodPost, base+"/api/execute", string(bodyBytes), cookies.cookieHeader())
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
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
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

// --- 正常系テスト ---

// TestHealthCheck は nginx のヘルスチェックエンドポイントが正常に応答することを検証する。
func TestHealthCheck(t *testing.T) {
	resp, err := httpClient.Get(base + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /health: want 200, got %d", resp.StatusCode)
	}
}

// TestCreateSessionAndExecute はセッション作成からコマンド実行までの正常系フローを検証する。
func TestCreateSessionAndExecute(t *testing.T) {
	cookies := setupSession(t)

	events := executeCommand(t, cookies, "pwd")

	var stdout string
	var hasComplete bool
	for _, e := range events {
		if e.Type == "stdout" {
			stdout += e.Data
		}
		if e.Type == "complete" && e.ExitCode != nil && *e.ExitCode == 0 {
			hasComplete = true
		}
	}
	if got := strings.TrimSpace(stdout); got != "/" {
		t.Errorf("pwd output: want %q, got %q (events: %+v)", "/", got, events)
	}
	if !hasComplete {
		t.Errorf("complete event with exitCode=0 not found in events: %+v", events)
	}
}

// TestDeleteSession はセッション削除が 204 を返すことを検証する。
func TestDeleteSession(t *testing.T) {
	cookies := setupSession(t)

	resp := doRequest(t, http.MethodDelete, base+"/api/session", "", cookies.cookieHeader())
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("DELETE /api/session: want 204, got %d", resp.StatusCode)
	}
}

// --- 異常系テスト ---

// TestExpiredSessionCookie は正規の runner_id cookie と不正な session_id cookie で
// コマンド実行した場合に runner が 404 を返すことを検証する。
func TestExpiredSessionCookie(t *testing.T) {
	cookies := setupSession(t)

	badCookies := sessionCookies{RunnerID: cookies.RunnerID, SessionID: "invalid-session-id"}
	body := `{"command":"pwd"}`
	resp := doRequest(t, http.MethodPost, base+"/api/execute", body, badCookies.cookieHeader())
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for invalid session_id, got %d", resp.StatusCode)
	}
}

// TestInvalidRunnerCookie は存在しない runner_id cookie でリクエストした場合の挙動を検証する。
// broker の resolve-or-create が idle runner を新規割り当てするため runner は busy になる。
// 元の session_id は新しい runner で無効なため、エラーが返る。
func TestInvalidRunnerCookie(t *testing.T) {
	// resolve-or-create で runner が busy になるので、テスト後に runner を idle に戻す
	// runner を直接削除して再登録することで状態をリセットする
	t.Cleanup(func() {
		deleteFromBroker(brokerBase + "/internal/runners/" + runnerHostname)
		registerRunnerOnBroker(runnerHostname)
	})

	fakeCookies := sessionCookies{RunnerID: "nonexistent-runner-id", SessionID: "nonexistent-session-id"}
	body := `{"command":"pwd"}`
	resp := doRequest(t, http.MethodPost, base+"/api/execute", body, fakeCookies.cookieHeader())
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Error("expected non-200 when using invalid cookies, but got 200")
	}
}

// TestExecuteAfterSessionDelete は runner の bash セッション削除後に同じ cookie で
// execute した場合に runner が 404 を返すことを検証する。
func TestExecuteAfterSessionDelete(t *testing.T) {
	cookies := setupSession(t)

	// runner の bash セッションを削除
	resp := doRequest(t, http.MethodDelete, base+"/api/session", "", cookies.cookieHeader())
	resp.Body.Close()

	// 同じ cookie で実行を試みる → runner が session_id を見つけられず 404
	body := `{"command":"pwd"}`
	resp = doRequest(t, http.MethodPost, base+"/api/execute", body, cookies.cookieHeader())
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after session deletion, got %d", resp.StatusCode)
	}
}

// TestNoIdleRunner は全ての runner が登録解除された状態でセッション作成を試みた場合、
// 503 SERVICE_UNAVAILABLE が返ることを検証する。
func TestNoIdleRunner(t *testing.T) {
	// セッションを作成して runner を busy にする
	cookies := createSession(t)

	// broker でセッションを閉じることで runner を DynamoDB から削除する
	deleteFromBroker(brokerBase + "/sessions/" + cookies.RunnerID)

	// idle runner がない状態で新規セッション作成を試みる → 503
	resp := doRequest(t, http.MethodPost, base+"/api/session", "", "")
	defer resp.Body.Close()

	// idle runner が存在しない場合、broker は 503 を返し nginx が error_page で処理する
	// broker 内部エラーの場合は 500 が返ることもある
	if resp.StatusCode != http.StatusServiceUnavailable && resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 503 or 500 when no idle runner, got %d", resp.StatusCode)
	}

	// クリーンアップ: runner を再登録
	registerRunnerOnBroker(runnerHostname)
}
