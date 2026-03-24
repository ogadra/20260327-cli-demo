package viewercount

import (
	"context"
	"fmt"
	"testing"
)

// mockConnectionCounter は ConnectionCounter のモック。
type mockConnectionCounter struct {
	countByRoomFn func(ctx context.Context, room string) (int, error)
}

// CountByRoom はモックの CountByRoom を呼び出す。
func (m *mockConnectionCounter) CountByRoom(ctx context.Context, room string) (int, error) {
	return m.countByRoomFn(ctx, room)
}

// mockSingleSender は SingleSender のモック。
type mockSingleSender struct {
	sendToOneFn func(ctx context.Context, connectionID string, payload []byte) error
}

// SendToOne はモックの SendToOne を呼び出す。
func (m *mockSingleSender) SendToOne(ctx context.Context, connectionID string, payload []byte) error {
	return m.sendToOneFn(ctx, connectionID, payload)
}

// TestNewHandler は Handler の生成を検証する。
func TestNewHandler(t *testing.T) {
	t.Parallel()
	c := &mockConnectionCounter{}
	s := &mockSingleSender{}
	h := NewHandler(c, s)
	if h.counter != c {
		t.Error("counter mismatch")
	}
	if h.sender != s {
		t.Error("sender mismatch")
	}
	if h.jsonMarshal == nil {
		t.Error("jsonMarshal should not be nil")
	}
}

// TestHandle_Success は正常な接続数通知を検証する。
func TestHandle_Success(t *testing.T) {
	t.Parallel()
	var capturedID string
	var capturedPayload []byte
	h := NewHandler(
		&mockConnectionCounter{
			countByRoomFn: func(_ context.Context, _ string) (int, error) {
				return 42, nil
			},
		},
		&mockSingleSender{
			sendToOneFn: func(_ context.Context, connectionID string, payload []byte) error {
				capturedID = connectionID
				capturedPayload = payload
				return nil
			},
		},
	)
	err := h.Handle(context.Background(), "default", "conn1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedID != "conn1" {
		t.Errorf("expected conn1, got %s", capturedID)
	}
	expected := `{"type":"viewer_count","count":42}`
	if string(capturedPayload) != expected {
		t.Errorf("expected %s, got %s", expected, string(capturedPayload))
	}
}

// TestHandle_CountError は接続数取得エラーを検証する。
func TestHandle_CountError(t *testing.T) {
	t.Parallel()
	h := NewHandler(
		&mockConnectionCounter{
			countByRoomFn: func(_ context.Context, _ string) (int, error) {
				return 0, fmt.Errorf("count error")
			},
		},
		&mockSingleSender{},
	)
	err := h.Handle(context.Background(), "default", "conn1")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestHandle_MarshalError は JSON マーシャルエラーを検証する。
func TestHandle_MarshalError(t *testing.T) {
	t.Parallel()
	h := NewHandler(
		&mockConnectionCounter{
			countByRoomFn: func(_ context.Context, _ string) (int, error) {
				return 1, nil
			},
		},
		&mockSingleSender{},
	)
	h.jsonMarshal = func(_ any) ([]byte, error) {
		return nil, fmt.Errorf("marshal error")
	}
	err := h.Handle(context.Background(), "default", "conn1")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestHandle_SendError は送信エラーを検証する。
func TestHandle_SendError(t *testing.T) {
	t.Parallel()
	h := NewHandler(
		&mockConnectionCounter{
			countByRoomFn: func(_ context.Context, _ string) (int, error) {
				return 1, nil
			},
		},
		&mockSingleSender{
			sendToOneFn: func(_ context.Context, _ string, _ []byte) error {
				return fmt.Errorf("send error")
			},
		},
	)
	err := h.Handle(context.Background(), "default", "conn1")
	if err == nil {
		t.Fatal("expected error")
	}
}
