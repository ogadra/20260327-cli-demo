// Package viewercount は接続数通知機能を提供する。
package viewercount

import (
	"context"
	"encoding/json"
	"fmt"
)

// ConnectionCounter は room 内の接続数を取得するインターフェース。
type ConnectionCounter interface {
	// CountByRoom は room 内の接続数を取得する。
	CountByRoom(ctx context.Context, room string) (int, error)
}

// SingleSender は単一接続にメッセージを送信するインターフェース。
type SingleSender interface {
	// SendToOne は単一の接続にメッセージを送信する。
	SendToOne(ctx context.Context, connectionID string, payload []byte) error
}

// Handler は接続数通知ハンドラー。
type Handler struct {
	counter     ConnectionCounter
	sender      SingleSender
	jsonMarshal func(v any) ([]byte, error)
}

// NewHandler は Handler を生成する。
func NewHandler(counter ConnectionCounter, sender SingleSender) *Handler {
	return &Handler{
		counter:     counter,
		sender:      sender,
		jsonMarshal: json.Marshal,
	}
}

// viewerCountMessage は接続数通知メッセージ。
type viewerCountMessage struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// Handle は要求元の接続に現在の接続数を返信する。
func (h *Handler) Handle(ctx context.Context, room, connectionID string) error {
	count, err := h.counter.CountByRoom(ctx, room)
	if err != nil {
		return fmt.Errorf("count connections: %w", err)
	}

	msg := viewerCountMessage{Type: "viewer_count", Count: count}
	payload, err := h.jsonMarshal(msg)
	if err != nil {
		return fmt.Errorf("marshal viewer_count: %w", err)
	}

	if err := h.sender.SendToOne(ctx, connectionID, payload); err != nil {
		return fmt.Errorf("send viewer_count: %w", err)
	}

	return nil
}
