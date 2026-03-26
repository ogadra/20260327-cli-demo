// Package handson はハンズオン指示配信機能を提供する。
package handson

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

// Handler はハンズオン指示ハンドラー。
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

// ErrNotPresenter は presenter ロール以外がハンズオン指示を試みた場合のエラー。
var ErrNotPresenter = fmt.Errorf("only presenter can send hands_on")

// handsOnMessage はハンズオン指示メッセージ。
type handsOnMessage struct {
	Type        string `json:"type"`
	Instruction string `json:"instruction"`
	Placeholder string `json:"placeholder"`
}

// Handle はハンズオン指示を処理する。presenter ロールのみ許可する。
func (h *Handler) Handle(ctx context.Context, room, connectionID, instruction, placeholder string) error {
	conn, err := h.connGetter.Get(ctx, room, connectionID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}
	if conn == nil {
		return fmt.Errorf("get connection: nil connection")
	}
	if conn.Role != "presenter" {
		return ErrNotPresenter
	}

	msg := handsOnMessage{Type: "hands_on", Instruction: instruction, Placeholder: placeholder}
	payload, err := h.jsonMarshal(msg)
	if err != nil {
		return fmt.Errorf("marshal hands_on: %w", err)
	}

	if err := h.broadcaster.Send(ctx, room, payload, connectionID); err != nil {
		return fmt.Errorf("broadcast hands_on: %w", err)
	}

	return nil
}
