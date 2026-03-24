package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/ogadra/20260327-cli-demo/presenter/internal/connection"
)

// mockConnectionStore は connectionStore のモック。
type mockConnectionStore struct {
	putFn         func(ctx context.Context, conn connection.Connection) error
	countByRoomFn func(ctx context.Context, room string) (int, error)
}

// Put はモックの Put を呼び出す。
func (m *mockConnectionStore) Put(ctx context.Context, conn connection.Connection) error {
	return m.putFn(ctx, conn)
}

// CountByRoom はモックの CountByRoom を呼び出す。
func (m *mockConnectionStore) CountByRoom(ctx context.Context, room string) (int, error) {
	return m.countByRoomFn(ctx, room)
}

// mockSessionValidator は sessionValidator のモック。
type mockSessionValidator struct {
	isValidFn func(ctx context.Context, token string) (bool, error)
}

// IsValid はモックの IsValid を呼び出す。
func (m *mockSessionValidator) IsValid(ctx context.Context, token string) (bool, error) {
	return m.isValidFn(ctx, token)
}

// mockBroadcaster は messageBroadcaster のモック。
type mockBroadcaster struct {
	sendFn func(ctx context.Context, room string, payload []byte, excludeConnectionID string) error
}

// Send はモックの Send を呼び出す。
func (m *mockBroadcaster) Send(ctx context.Context, room string, payload []byte, excludeConnectionID string) error {
	return m.sendFn(ctx, room, payload, excludeConnectionID)
}

// newRequest はテスト用の WebSocket プロキシリクエストを生成する。
func newRequest(connectionID, cookieHeader string) events.APIGatewayWebsocketProxyRequest {
	headers := map[string]string{}
	if cookieHeader != "" {
		headers["cookie"] = cookieHeader
	}
	return events.APIGatewayWebsocketProxyRequest{
		RequestContext: events.APIGatewayWebsocketProxyRequestContext{
			ConnectionID: connectionID,
		},
		Headers: headers,
	}
}

// TestHandle_ViewerConnect は未認証接続が viewer ロールで保存されることを検証する。
func TestHandle_ViewerConnect(t *testing.T) {
	t.Parallel()
	var savedConn connection.Connection
	h := &connectHandler{
		connStore: &mockConnectionStore{
			putFn: func(_ context.Context, conn connection.Connection) error {
				savedConn = conn
				return nil
			},
			countByRoomFn: func(_ context.Context, _ string) (int, error) {
				return 1, nil
			},
		},
		sessionStore: &mockSessionValidator{
			isValidFn: func(_ context.Context, _ string) (bool, error) {
				return false, nil
			},
		},
		broadcaster: &mockBroadcaster{
			sendFn: func(_ context.Context, _ string, _ []byte, _ string) error {
				return nil
			},
		},
	}
	resp, err := h.handle(context.Background(), newRequest("conn1", ""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if savedConn.Role != "viewer" {
		t.Errorf("expected viewer role, got %s", savedConn.Role)
	}
	if savedConn.ConnectionID != "conn1" {
		t.Errorf("expected conn1, got %s", savedConn.ConnectionID)
	}
}

// TestHandle_PresenterConnect は認証済み接続が presenter ロールで保存されることを検証する。
func TestHandle_PresenterConnect(t *testing.T) {
	t.Parallel()
	var savedRole string
	h := &connectHandler{
		connStore: &mockConnectionStore{
			putFn: func(_ context.Context, conn connection.Connection) error {
				savedRole = conn.Role
				return nil
			},
			countByRoomFn: func(_ context.Context, _ string) (int, error) {
				return 2, nil
			},
		},
		sessionStore: &mockSessionValidator{
			isValidFn: func(_ context.Context, token string) (bool, error) {
				return token == "valid-token", nil
			},
		},
		broadcaster: &mockBroadcaster{
			sendFn: func(_ context.Context, _ string, _ []byte, _ string) error {
				return nil
			},
		},
	}
	resp, err := h.handle(context.Background(), newRequest("conn2", "slide_auth=valid-token"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if savedRole != "presenter" {
		t.Errorf("expected presenter role, got %s", savedRole)
	}
}

// TestHandle_CookieHeaderCapital は Cookie ヘッダーが大文字の場合でもトークンを取得できることを検証する。
func TestHandle_CookieHeaderCapital(t *testing.T) {
	t.Parallel()
	var capturedToken string
	h := &connectHandler{
		connStore: &mockConnectionStore{
			putFn: func(_ context.Context, _ connection.Connection) error {
				return nil
			},
			countByRoomFn: func(_ context.Context, _ string) (int, error) {
				return 1, nil
			},
		},
		sessionStore: &mockSessionValidator{
			isValidFn: func(_ context.Context, token string) (bool, error) {
				capturedToken = token
				return false, nil
			},
		},
		broadcaster: &mockBroadcaster{
			sendFn: func(_ context.Context, _ string, _ []byte, _ string) error {
				return nil
			},
		},
	}
	req := events.APIGatewayWebsocketProxyRequest{
		RequestContext: events.APIGatewayWebsocketProxyRequestContext{
			ConnectionID: "conn3",
		},
		Headers: map[string]string{
			"Cookie": "slide_auth=capital-token",
		},
	}
	_, err := h.handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedToken != "capital-token" {
		t.Errorf("expected capital-token, got %s", capturedToken)
	}
}

// TestHandle_SessionValidateError はセッション検証エラー時に 500 を返すことを検証する。
func TestHandle_SessionValidateError(t *testing.T) {
	t.Parallel()
	h := &connectHandler{
		connStore: &mockConnectionStore{},
		sessionStore: &mockSessionValidator{
			isValidFn: func(_ context.Context, _ string) (bool, error) {
				return false, fmt.Errorf("session error")
			},
		},
		broadcaster: &mockBroadcaster{},
	}
	resp, err := h.handle(context.Background(), newRequest("conn1", ""))
	if err == nil {
		t.Fatal("expected error")
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// TestHandle_PutError は接続保存エラー時に 500 を返すことを検証する。
func TestHandle_PutError(t *testing.T) {
	t.Parallel()
	h := &connectHandler{
		connStore: &mockConnectionStore{
			putFn: func(_ context.Context, _ connection.Connection) error {
				return fmt.Errorf("put error")
			},
		},
		sessionStore: &mockSessionValidator{
			isValidFn: func(_ context.Context, _ string) (bool, error) {
				return false, nil
			},
		},
		broadcaster: &mockBroadcaster{},
	}
	resp, err := h.handle(context.Background(), newRequest("conn1", ""))
	if err == nil {
		t.Fatal("expected error")
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// TestHandle_CountError は接続数取得エラー時に 500 を返すことを検証する。
func TestHandle_CountError(t *testing.T) {
	t.Parallel()
	h := &connectHandler{
		connStore: &mockConnectionStore{
			putFn: func(_ context.Context, _ connection.Connection) error {
				return nil
			},
			countByRoomFn: func(_ context.Context, _ string) (int, error) {
				return 0, fmt.Errorf("count error")
			},
		},
		sessionStore: &mockSessionValidator{
			isValidFn: func(_ context.Context, _ string) (bool, error) {
				return false, nil
			},
		},
		broadcaster: &mockBroadcaster{},
	}
	resp, err := h.handle(context.Background(), newRequest("conn1", ""))
	if err == nil {
		t.Fatal("expected error")
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// TestHandle_BroadcastError はブロードキャストエラー時に 500 を返すことを検証する。
func TestHandle_BroadcastError(t *testing.T) {
	t.Parallel()
	h := &connectHandler{
		connStore: &mockConnectionStore{
			putFn: func(_ context.Context, _ connection.Connection) error {
				return nil
			},
			countByRoomFn: func(_ context.Context, _ string) (int, error) {
				return 1, nil
			},
		},
		sessionStore: &mockSessionValidator{
			isValidFn: func(_ context.Context, _ string) (bool, error) {
				return false, nil
			},
		},
		broadcaster: &mockBroadcaster{
			sendFn: func(_ context.Context, _ string, _ []byte, _ string) error {
				return fmt.Errorf("broadcast error")
			},
		},
	}
	resp, err := h.handle(context.Background(), newRequest("conn1", ""))
	if err == nil {
		t.Fatal("expected error")
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// TestExtractCookie_Found はクッキーが存在する場合に値を返すことを検証する。
func TestExtractCookie_Found(t *testing.T) {
	t.Parallel()
	val := extractCookie("slide_auth=abc123; other=xyz", "slide_auth")
	if val != "abc123" {
		t.Errorf("expected abc123, got %s", val)
	}
}

// TestExtractCookie_NotFound はクッキーが存在しない場合に空文字を返すことを検証する。
func TestExtractCookie_NotFound(t *testing.T) {
	t.Parallel()
	val := extractCookie("other=xyz", "slide_auth")
	if val != "" {
		t.Errorf("expected empty, got %s", val)
	}
}

// TestExtractCookie_Empty は空のクッキーヘッダーで空文字を返すことを検証する。
func TestExtractCookie_Empty(t *testing.T) {
	t.Parallel()
	val := extractCookie("", "slide_auth")
	if val != "" {
		t.Errorf("expected empty, got %s", val)
	}
}

// TestHandle_MarshalError は JSON マーシャルエラー時に 500 を返すことを検証する。
func TestHandle_MarshalError(t *testing.T) {
	origMarshal := jsonMarshal
	defer func() { jsonMarshal = origMarshal }()
	jsonMarshal = func(_ any) ([]byte, error) {
		return nil, fmt.Errorf("marshal error")
	}
	h := &connectHandler{
		connStore: &mockConnectionStore{
			putFn: func(_ context.Context, _ connection.Connection) error {
				return nil
			},
			countByRoomFn: func(_ context.Context, _ string) (int, error) {
				return 1, nil
			},
		},
		sessionStore: &mockSessionValidator{
			isValidFn: func(_ context.Context, _ string) (bool, error) {
				return false, nil
			},
		},
		broadcaster: &mockBroadcaster{},
	}
	resp, err := h.handle(context.Background(), newRequest("conn1", ""))
	if err == nil {
		t.Fatal("expected error")
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// TestRun_Success は run が正常に完了することを検証する。
func TestRun_Success(t *testing.T) {
	origStart := startLambda
	origLoadConfig := loadConfig
	defer func() {
		startLambda = origStart
		loadConfig = origLoadConfig
	}()

	t.Setenv("CONNECTIONS_TABLE", "conn-table")
	t.Setenv("SESSIONS_TABLE", "sess-table")

	startLambda = func(handler any) {}
	loadConfig = func(_ context.Context, _ ...func(*config.LoadOptions) error) (aws.Config, error) {
		return aws.Config{}, nil
	}

	if err := run(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRun_SuccessWithEndpoint は APIGW_ENDPOINT が設定されている場合に run が正常に完了することを検証する。
func TestRun_SuccessWithEndpoint(t *testing.T) {
	origStart := startLambda
	origLoadConfig := loadConfig
	defer func() {
		startLambda = origStart
		loadConfig = origLoadConfig
	}()

	t.Setenv("CONNECTIONS_TABLE", "conn-table")
	t.Setenv("SESSIONS_TABLE", "sess-table")
	t.Setenv("APIGW_ENDPOINT", "https://example.com")

	startLambda = func(handler any) {}
	loadConfig = func(_ context.Context, _ ...func(*config.LoadOptions) error) (aws.Config, error) {
		return aws.Config{}, nil
	}

	if err := run(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRun_MissingConnectionsTable は CONNECTIONS_TABLE 未設定時にエラーを返すことを検証する。
func TestRun_MissingConnectionsTable(t *testing.T) {
	origStart := startLambda
	origLoadConfig := loadConfig
	defer func() {
		startLambda = origStart
		loadConfig = origLoadConfig
	}()

	t.Setenv("CONNECTIONS_TABLE", "")

	startLambda = func(handler any) {}
	loadConfig = func(_ context.Context, _ ...func(*config.LoadOptions) error) (aws.Config, error) {
		return aws.Config{}, nil
	}

	if err := run(); err == nil {
		t.Fatal("expected error")
	}
}

// TestRun_MissingSessionsTable は SESSIONS_TABLE 未設定時にエラーを返すことを検証する。
func TestRun_MissingSessionsTable(t *testing.T) {
	origStart := startLambda
	origLoadConfig := loadConfig
	defer func() {
		startLambda = origStart
		loadConfig = origLoadConfig
	}()

	t.Setenv("CONNECTIONS_TABLE", "conn-table")
	t.Setenv("SESSIONS_TABLE", "")

	startLambda = func(handler any) {}
	loadConfig = func(_ context.Context, _ ...func(*config.LoadOptions) error) (aws.Config, error) {
		return aws.Config{}, nil
	}

	if err := run(); err == nil {
		t.Fatal("expected error")
	}
}

// TestRun_ConfigError は AWS 設定読み込みエラーで run がエラーを返すことを検証する。
func TestRun_ConfigError(t *testing.T) {
	origStart := startLambda
	origLoadConfig := loadConfig
	defer func() {
		startLambda = origStart
		loadConfig = origLoadConfig
	}()

	startLambda = func(handler any) {}
	loadConfig = func(_ context.Context, _ ...func(*config.LoadOptions) error) (aws.Config, error) {
		return aws.Config{}, fmt.Errorf("config error")
	}

	if err := run(); err == nil {
		t.Fatal("expected error")
	}
}

// TestMain_success は main 関数が正常に完了することを検証する。
func TestMain_success(t *testing.T) {
	origFatalf := fatalf
	origRun := runFn
	defer func() {
		fatalf = origFatalf
		runFn = origRun
	}()

	fatalf = func(format string, args ...any) {
		t.Fatalf("unexpected fatalf: "+format, args...)
	}
	runFn = func() error {
		return nil
	}

	main()
}

// TestMain_error は run がエラーを返した場合に fatalf が呼ばれることを検証する。
func TestMain_error(t *testing.T) {
	origFatalf := fatalf
	origRun := runFn
	defer func() {
		fatalf = origFatalf
		runFn = origRun
	}()

	var called bool
	fatalf = func(format string, args ...any) {
		called = true
	}
	runFn = func() error {
		return fmt.Errorf("test error")
	}

	main()

	if !called {
		t.Fatal("fatalf was not called")
	}
}
