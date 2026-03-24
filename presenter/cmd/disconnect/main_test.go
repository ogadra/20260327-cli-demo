package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// mockConnectionManager は connectionManager のモック。
type mockConnectionManager struct {
	deleteFn      func(ctx context.Context, room, connectionID string) error
	countByRoomFn func(ctx context.Context, room string) (int, error)
}

// Delete はモックの Delete を呼び出す。
func (m *mockConnectionManager) Delete(ctx context.Context, room, connectionID string) error {
	return m.deleteFn(ctx, room, connectionID)
}

// CountByRoom はモックの CountByRoom を呼び出す。
func (m *mockConnectionManager) CountByRoom(ctx context.Context, room string) (int, error) {
	return m.countByRoomFn(ctx, room)
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
func newRequest(connectionID string) events.APIGatewayWebsocketProxyRequest {
	return events.APIGatewayWebsocketProxyRequest{
		RequestContext: events.APIGatewayWebsocketProxyRequestContext{
			ConnectionID: connectionID,
		},
	}
}

// TestHandle_Success は正常な切断処理を検証する。
func TestHandle_Success(t *testing.T) {
	t.Parallel()
	var deletedID string
	h := &disconnectHandler{
		connStore: &mockConnectionManager{
			deleteFn: func(_ context.Context, _, connectionID string) error {
				deletedID = connectionID
				return nil
			},
			countByRoomFn: func(_ context.Context, _ string) (int, error) {
				return 0, nil
			},
		},
		broadcaster: &mockBroadcaster{
			sendFn: func(_ context.Context, _ string, _ []byte, _ string) error {
				return nil
			},
		},
	}
	resp, err := h.handle(context.Background(), newRequest("conn1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if deletedID != "conn1" {
		t.Errorf("expected conn1, got %s", deletedID)
	}
}

// TestHandle_DeleteError は接続削除エラー時に 500 を返すことを検証する。
func TestHandle_DeleteError(t *testing.T) {
	t.Parallel()
	h := &disconnectHandler{
		connStore: &mockConnectionManager{
			deleteFn: func(_ context.Context, _, _ string) error {
				return fmt.Errorf("delete error")
			},
		},
		broadcaster: &mockBroadcaster{},
	}
	resp, err := h.handle(context.Background(), newRequest("conn1"))
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
	h := &disconnectHandler{
		connStore: &mockConnectionManager{
			deleteFn: func(_ context.Context, _, _ string) error {
				return nil
			},
			countByRoomFn: func(_ context.Context, _ string) (int, error) {
				return 0, fmt.Errorf("count error")
			},
		},
		broadcaster: &mockBroadcaster{},
	}
	resp, err := h.handle(context.Background(), newRequest("conn1"))
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
	h := &disconnectHandler{
		connStore: &mockConnectionManager{
			deleteFn: func(_ context.Context, _, _ string) error {
				return nil
			},
			countByRoomFn: func(_ context.Context, _ string) (int, error) {
				return 5, nil
			},
		},
		broadcaster: &mockBroadcaster{
			sendFn: func(_ context.Context, _ string, _ []byte, _ string) error {
				return fmt.Errorf("broadcast error")
			},
		},
	}
	resp, err := h.handle(context.Background(), newRequest("conn1"))
	if err == nil {
		t.Fatal("expected error")
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// TestHandle_MarshalError は JSON マーシャルエラー時に 500 を返すことを検証する。
func TestHandle_MarshalError(t *testing.T) {
	origMarshal := jsonMarshal
	defer func() { jsonMarshal = origMarshal }()
	jsonMarshal = func(_ any) ([]byte, error) {
		return nil, fmt.Errorf("marshal error")
	}
	h := &disconnectHandler{
		connStore: &mockConnectionManager{
			deleteFn: func(_ context.Context, _, _ string) error {
				return nil
			},
			countByRoomFn: func(_ context.Context, _ string) (int, error) {
				return 1, nil
			},
		},
		broadcaster: &mockBroadcaster{},
	}
	resp, err := h.handle(context.Background(), newRequest("conn1"))
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
