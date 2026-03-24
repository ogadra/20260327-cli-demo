// Package broadcast は WebSocket 接続へのメッセージ配信を提供する。
package broadcast

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/smithy-go"

	"github.com/ogadra/20260327-cli-demo/presenter/internal/connection"
)

// APIGatewayManagementAPI は API Gateway Management API の narrow interface。
type APIGatewayManagementAPI interface {
	// PostToConnection は指定した接続にメッセージを送信する。
	PostToConnection(ctx context.Context, params *apigatewaymanagementapi.PostToConnectionInput, optFns ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error)
}

// ConnectionQuerier は room 内の接続一覧を取得するインターフェース。
type ConnectionQuerier interface {
	// QueryByRoom は room 内の全接続を取得する。
	QueryByRoom(ctx context.Context, room string) ([]connection.Connection, error)
}

// ConnectionDeleter は接続を削除するインターフェース。
type ConnectionDeleter interface {
	// Delete は接続情報を削除する。
	Delete(ctx context.Context, room, connectionID string) error
}

// Broadcaster は room 内の接続にメッセージを配信する。
type Broadcaster struct {
	apigw   APIGatewayManagementAPI
	querier ConnectionQuerier
	deleter ConnectionDeleter
}

// NewBroadcaster は Broadcaster を生成する。
func NewBroadcaster(apigw APIGatewayManagementAPI, querier ConnectionQuerier, deleter ConnectionDeleter) *Broadcaster {
	return &Broadcaster{
		apigw:   apigw,
		querier: querier,
		deleter: deleter,
	}
}

// isGoneError は API Gateway の GoneException かどうかを判定する。
func isGoneError(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "GoneException"
	}
	return false
}

// Send は room 内の全接続にメッセージを配信する。excludeConnectionID に一致する接続はスキップする。
func (b *Broadcaster) Send(ctx context.Context, room string, payload []byte, excludeConnectionID string) error {
	conns, err := b.querier.QueryByRoom(ctx, room)
	if err != nil {
		return fmt.Errorf("query connections: %w", err)
	}
	for _, conn := range conns {
		if conn.ConnectionID == excludeConnectionID {
			continue
		}
		_, postErr := b.apigw.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
			ConnectionId: &conn.ConnectionID,
			Data:         payload,
		})
		if postErr != nil {
			if isGoneError(postErr) {
				_ = b.deleter.Delete(ctx, conn.Room, conn.ConnectionID)
				continue
			}
			return fmt.Errorf("post to connection %s: %w", conn.ConnectionID, postErr)
		}
	}
	return nil
}

// SendToOne は単一の接続にメッセージを送信する。GoneException 時は接続を自動削除する。
func (b *Broadcaster) SendToOne(ctx context.Context, room, connectionID string, payload []byte) error {
	_, err := b.apigw.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: &connectionID,
		Data:         payload,
	})
	if err != nil {
		if isGoneError(err) {
			_ = b.deleter.Delete(ctx, room, connectionID)
			return nil
		}
		return fmt.Errorf("post to connection %s: %w", connectionID, err)
	}
	return nil
}
