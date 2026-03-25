package slidesync

import (
	"context"
	"fmt"
	"testing"

	"github.com/ogadra/20260327-cli-demo/presenter/internal/connection"
)

// mockConnectionGetter は ConnectionGetter のモック。
type mockConnectionGetter struct {
	getFn func(ctx context.Context, room, connectionID string) (*connection.Connection, error)
}

// Get はモックの Get を呼び出す。
func (m *mockConnectionGetter) Get(ctx context.Context, room, connectionID string) (*connection.Connection, error) {
	return m.getFn(ctx, room, connectionID)
}

// mockBroadcaster は Broadcaster のモック。
type mockBroadcaster struct {
	sendFn func(ctx context.Context, room string, payload []byte, excludeConnectionID string) error
}

// Send はモックの Send を呼び出す。
func (m *mockBroadcaster) Send(ctx context.Context, room string, payload []byte, excludeConnectionID string) error {
	return m.sendFn(ctx, room, payload, excludeConnectionID)
}

// TestNewHandler は Handler の生成を検証する。
func TestNewHandler(t *testing.T) {
	t.Parallel()
	g := &mockConnectionGetter{}
	b := &mockBroadcaster{}
	h := NewHandler(g, b)
	if h.connGetter != g {
		t.Error("connGetter mismatch")
	}
	if h.broadcaster != b {
		t.Error("broadcaster mismatch")
	}
	if h.jsonMarshal == nil {
		t.Error("jsonMarshal should not be nil")
	}
}

// TestHandle_Success は presenter ロールでの正常なスライド同期を検証する。
func TestHandle_Success(t *testing.T) {
	t.Parallel()
	var capturedExclude string
	var capturedPayload []byte
	h := NewHandler(
		&mockConnectionGetter{
			getFn: func(_ context.Context, _, _ string) (*connection.Connection, error) {
				return &connection.Connection{Role: "presenter"}, nil
			},
		},
		&mockBroadcaster{
			sendFn: func(_ context.Context, _ string, payload []byte, exclude string) error {
				capturedExclude = exclude
				capturedPayload = payload
				return nil
			},
		},
	)
	err := h.Handle(context.Background(), "default", "conn1", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedExclude != "conn1" {
		t.Errorf("expected conn1 excluded, got %s", capturedExclude)
	}
	expected := `{"type":"slide_sync","page":3}`
	if string(capturedPayload) != expected {
		t.Errorf("expected %s, got %s", expected, string(capturedPayload))
	}
}

// TestHandle_NotPresenter は viewer ロールがスライド同期を拒否されることを検証する。
func TestHandle_NotPresenter(t *testing.T) {
	t.Parallel()
	h := NewHandler(
		&mockConnectionGetter{
			getFn: func(_ context.Context, _, _ string) (*connection.Connection, error) {
				return &connection.Connection{Role: "viewer"}, nil
			},
		},
		&mockBroadcaster{},
	)
	err := h.Handle(context.Background(), "default", "conn1", 3)
	if err != ErrNotPresenter {
		t.Errorf("expected ErrNotPresenter, got %v", err)
	}
}

// TestHandle_GetError は接続情報取得エラーを検証する。
func TestHandle_GetError(t *testing.T) {
	t.Parallel()
	h := NewHandler(
		&mockConnectionGetter{
			getFn: func(_ context.Context, _, _ string) (*connection.Connection, error) {
				return nil, fmt.Errorf("get error")
			},
		},
		&mockBroadcaster{},
	)
	err := h.Handle(context.Background(), "default", "conn1", 3)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestHandle_MarshalError は JSON マーシャルエラーを検証する。
func TestHandle_MarshalError(t *testing.T) {
	t.Parallel()
	h := NewHandler(
		&mockConnectionGetter{
			getFn: func(_ context.Context, _, _ string) (*connection.Connection, error) {
				return &connection.Connection{Role: "presenter"}, nil
			},
		},
		&mockBroadcaster{},
	)
	h.jsonMarshal = func(_ any) ([]byte, error) {
		return nil, fmt.Errorf("marshal error")
	}
	err := h.Handle(context.Background(), "default", "conn1", 3)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestHandle_BroadcastError はブロードキャストエラーを検証する。
func TestHandle_BroadcastError(t *testing.T) {
	t.Parallel()
	h := NewHandler(
		&mockConnectionGetter{
			getFn: func(_ context.Context, _, _ string) (*connection.Connection, error) {
				return &connection.Connection{Role: "presenter"}, nil
			},
		},
		&mockBroadcaster{
			sendFn: func(_ context.Context, _ string, _ []byte, _ string) error {
				return fmt.Errorf("broadcast error")
			},
		},
	)
	err := h.Handle(context.Background(), "default", "conn1", 3)
	if err == nil {
		t.Fatal("expected error")
	}
}
