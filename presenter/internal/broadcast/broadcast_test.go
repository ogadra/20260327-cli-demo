// Package broadcast はブロードキャスト機能のテストを提供する。
package broadcast

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/smithy-go"

	"github.com/ogadra/20260327-cli-demo/presenter/internal/connection"
)

// mockAPIGatewayManagementAPI は APIGatewayManagementAPI のモック。
type mockAPIGatewayManagementAPI struct {
	postToConnectionFn func(ctx context.Context, params *apigatewaymanagementapi.PostToConnectionInput, optFns ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error)
}

// PostToConnection はモックの PostToConnection を呼び出す。
func (m *mockAPIGatewayManagementAPI) PostToConnection(ctx context.Context, params *apigatewaymanagementapi.PostToConnectionInput, optFns ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error) {
	return m.postToConnectionFn(ctx, params, optFns...)
}

// mockConnectionQuerier は ConnectionQuerier のモック。
type mockConnectionQuerier struct {
	queryByRoomFn func(ctx context.Context, room string) ([]connection.Connection, error)
}

// QueryByRoom はモックの QueryByRoom を呼び出す。
func (m *mockConnectionQuerier) QueryByRoom(ctx context.Context, room string) ([]connection.Connection, error) {
	return m.queryByRoomFn(ctx, room)
}

// mockConnectionDeleter は ConnectionDeleter のモック。
type mockConnectionDeleter struct {
	deleteFn func(ctx context.Context, room, connectionID string) error
}

// Delete はモックの Delete を呼び出す。
func (m *mockConnectionDeleter) Delete(ctx context.Context, room, connectionID string) error {
	return m.deleteFn(ctx, room, connectionID)
}

// goneError は GoneException を模倣する smithy.APIError。
type goneError struct{}

// Error はエラーメッセージを返す。
func (e *goneError) Error() string { return "GoneException" }

// ErrorCode はエラーコードを返す。
func (e *goneError) ErrorCode() string { return "GoneException" }

// ErrorMessage はエラーメッセージを返す。
func (e *goneError) ErrorMessage() string { return "gone" }

// ErrorFault はエラーフォルトを返す。
func (e *goneError) ErrorFault() smithy.ErrorFault { return smithy.FaultClient }

// TestNewBroadcaster は Broadcaster の生成を検証する。
func TestNewBroadcaster(t *testing.T) {
	t.Parallel()
	apigw := &mockAPIGatewayManagementAPI{}
	querier := &mockConnectionQuerier{}
	deleter := &mockConnectionDeleter{}
	b := NewBroadcaster(apigw, querier, deleter)
	if b.apigw != apigw {
		t.Error("apigw mismatch")
	}
	if b.querier != querier {
		t.Error("querier mismatch")
	}
	if b.deleter != deleter {
		t.Error("deleter mismatch")
	}
}

// TestSend_Success は全接続への正常送信を検証する。
func TestSend_Success(t *testing.T) {
	t.Parallel()
	var posted sync.Map
	apigw := &mockAPIGatewayManagementAPI{
		postToConnectionFn: func(_ context.Context, params *apigatewaymanagementapi.PostToConnectionInput, _ ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error) {
			posted.Store(*params.ConnectionId, true)
			return &apigatewaymanagementapi.PostToConnectionOutput{}, nil
		},
	}
	querier := &mockConnectionQuerier{
		queryByRoomFn: func(_ context.Context, _ string) ([]connection.Connection, error) {
			return []connection.Connection{
				{Room: "default", ConnectionID: "conn1"},
				{Room: "default", ConnectionID: "conn2"},
			}, nil
		},
	}
	deleter := &mockConnectionDeleter{}
	b := NewBroadcaster(apigw, querier, deleter)
	err := b.Send(context.Background(), "default", []byte(`{"type":"test"}`), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, ok1 := posted.Load("conn1")
	_, ok2 := posted.Load("conn2")
	if !ok1 || !ok2 {
		t.Error("expected both connections to receive message")
	}
}

// TestSend_ExcludeConnection は除外接続がスキップされることを検証する。
func TestSend_ExcludeConnection(t *testing.T) {
	t.Parallel()
	var posted sync.Map
	apigw := &mockAPIGatewayManagementAPI{
		postToConnectionFn: func(_ context.Context, params *apigatewaymanagementapi.PostToConnectionInput, _ ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error) {
			posted.Store(*params.ConnectionId, true)
			return &apigatewaymanagementapi.PostToConnectionOutput{}, nil
		},
	}
	querier := &mockConnectionQuerier{
		queryByRoomFn: func(_ context.Context, _ string) ([]connection.Connection, error) {
			return []connection.Connection{
				{Room: "default", ConnectionID: "conn1"},
				{Room: "default", ConnectionID: "conn2"},
				{Room: "default", ConnectionID: "conn3"},
			}, nil
		},
	}
	deleter := &mockConnectionDeleter{}
	b := NewBroadcaster(apigw, querier, deleter)
	err := b.Send(context.Background(), "default", []byte(`{"type":"test"}`), "conn2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, ok1 := posted.Load("conn1")
	_, ok3 := posted.Load("conn3")
	if !ok1 || !ok3 {
		t.Error("expected conn1 and conn3 to receive message")
	}
	_, ok2 := posted.Load("conn2")
	if ok2 {
		t.Error("expected conn2 to be excluded")
	}
}

// TestSend_QueryError はクエリ失敗時のエラーを検証する。
func TestSend_QueryError(t *testing.T) {
	t.Parallel()
	querier := &mockConnectionQuerier{
		queryByRoomFn: func(_ context.Context, _ string) ([]connection.Connection, error) {
			return nil, fmt.Errorf("dynamo error")
		},
	}
	b := NewBroadcaster(&mockAPIGatewayManagementAPI{}, querier, &mockConnectionDeleter{})
	err := b.Send(context.Background(), "default", []byte(`{}`), "")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestSend_GoneException は GoneException 時に接続が自動削除されることを検証する。
func TestSend_GoneException(t *testing.T) {
	t.Parallel()
	var deleted sync.Map
	apigw := &mockAPIGatewayManagementAPI{
		postToConnectionFn: func(_ context.Context, params *apigatewaymanagementapi.PostToConnectionInput, _ ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error) {
			if *params.ConnectionId == "gone-conn" {
				return nil, &goneError{}
			}
			return &apigatewaymanagementapi.PostToConnectionOutput{}, nil
		},
	}
	querier := &mockConnectionQuerier{
		queryByRoomFn: func(_ context.Context, _ string) ([]connection.Connection, error) {
			return []connection.Connection{
				{Room: "default", ConnectionID: "gone-conn"},
				{Room: "default", ConnectionID: "ok-conn"},
			}, nil
		},
	}
	deleter := &mockConnectionDeleter{
		deleteFn: func(_ context.Context, _, connectionID string) error {
			deleted.Store(connectionID, true)
			return nil
		},
	}
	b := NewBroadcaster(apigw, querier, deleter)
	err := b.Send(context.Background(), "default", []byte(`{}`), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, ok := deleted.Load("gone-conn")
	if !ok {
		t.Error("expected gone-conn to be deleted")
	}
}

// TestSend_GoneExceptionDeleteError は GoneException 時の削除失敗が無視されることを検証する。
func TestSend_GoneExceptionDeleteError(t *testing.T) {
	t.Parallel()
	apigw := &mockAPIGatewayManagementAPI{
		postToConnectionFn: func(_ context.Context, _ *apigatewaymanagementapi.PostToConnectionInput, _ ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error) {
			return nil, &goneError{}
		},
	}
	querier := &mockConnectionQuerier{
		queryByRoomFn: func(_ context.Context, _ string) ([]connection.Connection, error) {
			return []connection.Connection{
				{Room: "default", ConnectionID: "conn1"},
			}, nil
		},
	}
	deleter := &mockConnectionDeleter{
		deleteFn: func(_ context.Context, _, _ string) error {
			return fmt.Errorf("delete error")
		},
	}
	b := NewBroadcaster(apigw, querier, deleter)
	err := b.Send(context.Background(), "default", []byte(`{}`), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestSend_PostError は PostToConnection の非 Gone エラーを検証する。
func TestSend_PostError(t *testing.T) {
	t.Parallel()
	apigw := &mockAPIGatewayManagementAPI{
		postToConnectionFn: func(_ context.Context, _ *apigatewaymanagementapi.PostToConnectionInput, _ ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error) {
			return nil, fmt.Errorf("internal error")
		},
	}
	querier := &mockConnectionQuerier{
		queryByRoomFn: func(_ context.Context, _ string) ([]connection.Connection, error) {
			return []connection.Connection{
				{Room: "default", ConnectionID: "conn1"},
			}, nil
		},
	}
	b := NewBroadcaster(apigw, querier, &mockConnectionDeleter{})
	err := b.Send(context.Background(), "default", []byte(`{}`), "")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestSend_EmptyConnections は接続が空の場合を検証する。
func TestSend_EmptyConnections(t *testing.T) {
	t.Parallel()
	querier := &mockConnectionQuerier{
		queryByRoomFn: func(_ context.Context, _ string) ([]connection.Connection, error) {
			return []connection.Connection{}, nil
		},
	}
	b := NewBroadcaster(&mockAPIGatewayManagementAPI{}, querier, &mockConnectionDeleter{})
	err := b.Send(context.Background(), "default", []byte(`{}`), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestSendToOne_Success は単一接続への正常送信を検証する。
func TestSendToOne_Success(t *testing.T) {
	t.Parallel()
	var capturedID string
	apigw := &mockAPIGatewayManagementAPI{
		postToConnectionFn: func(_ context.Context, params *apigatewaymanagementapi.PostToConnectionInput, _ ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error) {
			capturedID = *params.ConnectionId
			return &apigatewaymanagementapi.PostToConnectionOutput{}, nil
		},
	}
	b := NewBroadcaster(apigw, &mockConnectionQuerier{}, &mockConnectionDeleter{})
	err := b.SendToOne(context.Background(), "default", "conn1", []byte(`{"type":"test"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedID != "conn1" {
		t.Errorf("expected conn1, got %s", capturedID)
	}
}

// TestSendToOne_Error は単一接続への送信失敗を検証する。
func TestSendToOne_Error(t *testing.T) {
	t.Parallel()
	apigw := &mockAPIGatewayManagementAPI{
		postToConnectionFn: func(_ context.Context, _ *apigatewaymanagementapi.PostToConnectionInput, _ ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error) {
			return nil, fmt.Errorf("post error")
		},
	}
	b := NewBroadcaster(apigw, &mockConnectionQuerier{}, &mockConnectionDeleter{})
	err := b.SendToOne(context.Background(), "default", "conn1", []byte(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestSendToOne_GoneException は GoneException 時に接続が自動削除されることを検証する。
func TestSendToOne_GoneException(t *testing.T) {
	t.Parallel()
	var deletedConn string
	apigw := &mockAPIGatewayManagementAPI{
		postToConnectionFn: func(_ context.Context, _ *apigatewaymanagementapi.PostToConnectionInput, _ ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error) {
			return nil, &goneError{}
		},
	}
	deleter := &mockConnectionDeleter{
		deleteFn: func(_ context.Context, _, connectionID string) error {
			deletedConn = connectionID
			return nil
		},
	}
	b := NewBroadcaster(apigw, &mockConnectionQuerier{}, deleter)
	err := b.SendToOne(context.Background(), "default", "gone-conn", []byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedConn != "gone-conn" {
		t.Errorf("expected gone-conn to be deleted, got %s", deletedConn)
	}
}

// TestSendToOne_GoneExceptionDeleteError は GoneException 時の削除失敗が無視されることを検証する。
func TestSendToOne_GoneExceptionDeleteError(t *testing.T) {
	t.Parallel()
	apigw := &mockAPIGatewayManagementAPI{
		postToConnectionFn: func(_ context.Context, _ *apigatewaymanagementapi.PostToConnectionInput, _ ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error) {
			return nil, &goneError{}
		},
	}
	deleter := &mockConnectionDeleter{
		deleteFn: func(_ context.Context, _, _ string) error {
			return fmt.Errorf("delete error")
		},
	}
	b := NewBroadcaster(apigw, &mockConnectionQuerier{}, deleter)
	err := b.SendToOne(context.Background(), "default", "conn1", []byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestIsGoneError_True は GoneException が正しく判定されることを検証する。
func TestIsGoneError_True(t *testing.T) {
	t.Parallel()
	if !isGoneError(&goneError{}) {
		t.Error("expected true for GoneException")
	}
}

// TestIsGoneError_False は非 GoneException が正しく判定されることを検証する。
func TestIsGoneError_False(t *testing.T) {
	t.Parallel()
	if isGoneError(fmt.Errorf("other error")) {
		t.Error("expected false for non-GoneException")
	}
}

// TestIsGoneError_OtherAPIError は GoneException 以外の APIError が false を返すことを検証する。
func TestIsGoneError_OtherAPIError(t *testing.T) {
	t.Parallel()
	err := &otherAPIError{}
	if isGoneError(err) {
		t.Error("expected false for non-Gone APIError")
	}
}

// otherAPIError は GoneException 以外の APIError。
type otherAPIError struct{}

// Error はエラーメッセージを返す。
func (e *otherAPIError) Error() string { return "other" }

// ErrorCode はエラーコードを返す。
func (e *otherAPIError) ErrorCode() string { return "OtherException" }

// ErrorMessage はエラーメッセージを返す。
func (e *otherAPIError) ErrorMessage() string { return "other" }

// ErrorFault はエラーフォルトを返す。
func (e *otherAPIError) ErrorFault() smithy.ErrorFault { return smithy.FaultClient }
