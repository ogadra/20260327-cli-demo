// Package slidesync はスライドページ同期機能を提供する。
package slidesync

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ogadra/20260327-cli-demo/presenter/internal/connection"
)

// ConnectionGetter は接続情報の取得インターフェース。
type ConnectionGetter interface {
	// Get は指定した room と connectionID の接続情報を取得する。
	Get(ctx context.Context, room, connectionID string) (*connection.Connection, error)
}

// Broadcaster はメッセージ配信インターフェース。
type Broadcaster interface {
	// Send は room 内の全接続にメッセージを配信する。
	Send(ctx context.Context, room string, payload []byte, excludeConnectionID string) error
}

// Handler はスライド同期ハンドラー。
type Handler struct {
	connGetter  ConnectionGetter
	broadcaster Broadcaster
	jsonMarshal func(v any) ([]byte, error)
}

// NewHandler は Handler を生成する。
func NewHandler(connGetter ConnectionGetter, broadcaster Broadcaster) *Handler {
	return &Handler{
		connGetter:  connGetter,
		broadcaster: broadcaster,
		jsonMarshal: json.Marshal,
	}
}

// ErrNotPresenter は presenter ロール以外がスライド同期を試みた場合のエラー。
var ErrNotPresenter = fmt.Errorf("only presenter can sync slides")

// slideSyncMessage はスライド同期メッセージ。
type slideSyncMessage struct {
	Type string `json:"type"`
	Page int    `json:"page"`
}

// Handle はスライドページ同期を処理する。presenter ロールのみ許可する。
func (h *Handler) Handle(ctx context.Context, room, connectionID string, page int) error {
	conn, err := h.connGetter.Get(ctx, room, connectionID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}
	if conn.Role != "presenter" {
		return ErrNotPresenter
	}

	msg := slideSyncMessage{Type: "slide_sync", Page: page}
	payload, err := h.jsonMarshal(msg)
	if err != nil {
		return fmt.Errorf("marshal slide_sync: %w", err)
	}

	if err := h.broadcaster.Send(ctx, room, payload, connectionID); err != nil {
		return fmt.Errorf("broadcast slide_sync: %w", err)
	}

	return nil
}
