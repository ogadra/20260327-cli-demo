package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/ogadra/20260327-cli-demo/presenter/internal/connection"
	"github.com/ogadra/20260327-cli-demo/presenter/internal/poll"
)

// mockSlideSyncDispatcher はスライド同期ハンドラーのモック。
type mockSlideSyncDispatcher struct {
	handleFn func(ctx context.Context, room, connectionID string, page int) error
}

// Handle はモックの Handle を呼び出す。
func (m *mockSlideSyncDispatcher) Handle(ctx context.Context, room, connectionID string, page int) error {
	return m.handleFn(ctx, room, connectionID, page)
}

// mockViewerCountDispatcher は接続数通知ハンドラーのモック。
type mockViewerCountDispatcher struct {
	handleFn func(ctx context.Context, room, connectionID string) error
}

// Handle はモックの Handle を呼び出す。
func (m *mockViewerCountDispatcher) Handle(ctx context.Context, room, connectionID string) error {
	return m.handleFn(ctx, room, connectionID)
}

// mockPollGetter はアンケート状態取得のモック。
type mockPollGetter struct {
	getFn func(ctx context.Context, pollID, visitorID string, options []string, maxChoices int, isPresenter bool) (*poll.PollState, error)
}

// Get はモックの Get を呼び出す。
func (m *mockPollGetter) Get(ctx context.Context, pollID, visitorID string, options []string, maxChoices int, isPresenter bool) (*poll.PollState, error) {
	return m.getFn(ctx, pollID, visitorID, options, maxChoices, isPresenter)
}

// mockPollVoter は投票のモック。
type mockPollVoter struct {
	voteFn func(ctx context.Context, pollID, visitorID, choice string) error
}

// Vote はモックの Vote を呼び出す。
func (m *mockPollVoter) Vote(ctx context.Context, pollID, visitorID, choice string) error {
	return m.voteFn(ctx, pollID, visitorID, choice)
}

// mockPollUnvoter は投票取消のモック。
type mockPollUnvoter struct {
	unvoteFn func(ctx context.Context, pollID, visitorID, choice string) error
}

// Unvote はモックの Unvote を呼び出す。
func (m *mockPollUnvoter) Unvote(ctx context.Context, pollID, visitorID, choice string) error {
	return m.unvoteFn(ctx, pollID, visitorID, choice)
}

// mockPollSwitcher は投票変更のモック。
type mockPollSwitcher struct {
	switchFn func(ctx context.Context, pollID, visitorID, from, to string) error
}

// Switch はモックの Switch を呼び出す。
func (m *mockPollSwitcher) Switch(ctx context.Context, pollID, visitorID, from, to string) error {
	return m.switchFn(ctx, pollID, visitorID, from, to)
}

// mockConnectionGetter は接続情報取得のモック。
type mockConnectionGetter struct {
	getFn func(ctx context.Context, room, connectionID string) (*connection.Connection, error)
}

// Get はモックの Get を呼び出す。
func (m *mockConnectionGetter) Get(ctx context.Context, room, connectionID string) (*connection.Connection, error) {
	return m.getFn(ctx, room, connectionID)
}

// mockMessageBroadcaster はメッセージ配信のモック。
type mockMessageBroadcaster struct {
	sendFn func(ctx context.Context, room string, payload []byte, excludeConnectionID string) error
}

// Send はモックの Send を呼び出す。
func (m *mockMessageBroadcaster) Send(ctx context.Context, room string, payload []byte, excludeConnectionID string) error {
	return m.sendFn(ctx, room, payload, excludeConnectionID)
}

// mockSingleSender は単一送信のモック。
type mockSingleSender struct {
	sendToOneFn func(ctx context.Context, room, connectionID string, payload []byte) error
}

// SendToOne はモックの SendToOne を呼び出す。
func (m *mockSingleSender) SendToOne(ctx context.Context, room, connectionID string, payload []byte) error {
	return m.sendToOneFn(ctx, room, connectionID, payload)
}

// newRequest はテスト用の APIGatewayWebsocketProxyRequest を生成する。
func newRequest(connectionID, body string) events.APIGatewayWebsocketProxyRequest {
	return events.APIGatewayWebsocketProxyRequest{
		RequestContext: events.APIGatewayWebsocketProxyRequestContext{
			ConnectionID: connectionID,
		},
		Body: body,
	}
}

// defaultPollState はテスト用のデフォルト PollState を返す。
func defaultPollState() *poll.PollState {
	return &poll.PollState{
		PollID:     "q1",
		Options:    []string{"A", "B"},
		MaxChoices: 1,
		Votes:      map[string]int{"A": 1, "B": 0},
		MyChoices:  []string{"A"},
	}
}

// TestHandle_SlideSync はスライド同期メッセージの正常処理を検証する。
func TestHandle_SlideSync(t *testing.T) {
	t.Parallel()
	var capturedPage int
	h := &messageHandler{
		slideSync: &mockSlideSyncDispatcher{
			handleFn: func(_ context.Context, _, _ string, page int) error {
				capturedPage = page
				return nil
			},
		},
	}
	req := newRequest("conn1", `{"type":"slide_sync","page":3}`)
	resp, err := h.handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if capturedPage != 3 {
		t.Errorf("expected page 3, got %d", capturedPage)
	}
}

// TestHandle_SlideSyncError はスライド同期エラー時のエラーレスポンス送信を検証する。
func TestHandle_SlideSyncError(t *testing.T) {
	t.Parallel()
	var sentPayload []byte
	h := &messageHandler{
		slideSync: &mockSlideSyncDispatcher{
			handleFn: func(_ context.Context, _, _ string, _ int) error {
				return fmt.Errorf("not presenter")
			},
		},
		singleSender: &mockSingleSender{
			sendToOneFn: func(_ context.Context, _, _ string, payload []byte) error {
				sentPayload = payload
				return nil
			},
		},
	}
	req := newRequest("conn1", `{"type":"slide_sync","page":1}`)
	resp, err := h.handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var errResp errorResponse
	if unmarshalErr := json.Unmarshal(sentPayload, &errResp); unmarshalErr != nil {
		t.Fatalf("unmarshal error response: %v", unmarshalErr)
	}
	if errResp.Type != "error" {
		t.Errorf("expected type error, got %s", errResp.Type)
	}
}

// TestHandle_ViewerCount は接続数通知メッセージの正常処理を検証する。
func TestHandle_ViewerCount(t *testing.T) {
	t.Parallel()
	called := false
	h := &messageHandler{
		viewerCount: &mockViewerCountDispatcher{
			handleFn: func(_ context.Context, _, _ string) error {
				called = true
				return nil
			},
		},
	}
	req := newRequest("conn1", `{"type":"viewer_count"}`)
	resp, err := h.handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !called {
		t.Error("expected viewerCount handler to be called")
	}
}

// TestHandle_ViewerCountError は接続数通知エラーを検証する。
func TestHandle_ViewerCountError(t *testing.T) {
	t.Parallel()
	h := &messageHandler{
		viewerCount: &mockViewerCountDispatcher{
			handleFn: func(_ context.Context, _, _ string) error {
				return fmt.Errorf("count error")
			},
		},
	}
	req := newRequest("conn1", `{"type":"viewer_count"}`)
	_, err := h.handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestHandle_PollGet はアンケート取得の正常処理を検証する。
func TestHandle_PollGet(t *testing.T) {
	t.Parallel()
	var broadcastPayload []byte
	h := &messageHandler{
		connGetter: &mockConnectionGetter{
			getFn: func(_ context.Context, _, _ string) (*connection.Connection, error) {
				return &connection.Connection{Role: "presenter"}, nil
			},
		},
		pollGet: &mockPollGetter{
			getFn: func(_ context.Context, _, _ string, _ []string, _ int, isPresenter bool) (*poll.PollState, error) {
				if !isPresenter {
					t.Error("expected isPresenter to be true")
				}
				return defaultPollState(), nil
			},
		},
		broadcaster: &mockMessageBroadcaster{
			sendFn: func(_ context.Context, _ string, payload []byte, _ string) error {
				broadcastPayload = payload
				return nil
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_get","pollId":"q1","options":["A","B"],"maxChoices":1,"visitorId":"v1"}`)
	resp, err := h.handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var stateResp pollStateResponse
	if unmarshalErr := json.Unmarshal(broadcastPayload, &stateResp); unmarshalErr != nil {
		t.Fatalf("unmarshal poll state: %v", unmarshalErr)
	}
	if stateResp.Type != "poll_state" {
		t.Errorf("expected type poll_state, got %s", stateResp.Type)
	}
}

// TestHandle_PollGetViewer はビューアーからのアンケート取得を検証する。
func TestHandle_PollGetViewer(t *testing.T) {
	t.Parallel()
	h := &messageHandler{
		connGetter: &mockConnectionGetter{
			getFn: func(_ context.Context, _, _ string) (*connection.Connection, error) {
				return &connection.Connection{Role: "viewer"}, nil
			},
		},
		pollGet: &mockPollGetter{
			getFn: func(_ context.Context, _, _ string, _ []string, _ int, isPresenter bool) (*poll.PollState, error) {
				if isPresenter {
					t.Error("expected isPresenter to be false")
				}
				return defaultPollState(), nil
			},
		},
		broadcaster: &mockMessageBroadcaster{
			sendFn: func(_ context.Context, _ string, _ []byte, _ string) error {
				return nil
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_get","pollId":"q1","visitorId":"v1"}`)
	resp, err := h.handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// TestHandle_PollGetConnError は接続情報取得エラーを検証する。
func TestHandle_PollGetConnError(t *testing.T) {
	t.Parallel()
	h := &messageHandler{
		connGetter: &mockConnectionGetter{
			getFn: func(_ context.Context, _, _ string) (*connection.Connection, error) {
				return nil, fmt.Errorf("conn error")
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_get","pollId":"q1"}`)
	_, err := h.handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestHandle_PollGetError はアンケート取得エラーを検証する。
func TestHandle_PollGetError(t *testing.T) {
	t.Parallel()
	h := &messageHandler{
		connGetter: &mockConnectionGetter{
			getFn: func(_ context.Context, _, _ string) (*connection.Connection, error) {
				return &connection.Connection{Role: "viewer"}, nil
			},
		},
		pollGet: &mockPollGetter{
			getFn: func(_ context.Context, _, _ string, _ []string, _ int, _ bool) (*poll.PollState, error) {
				return nil, fmt.Errorf("get error")
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_get","pollId":"q1","visitorId":"v1"}`)
	_, err := h.handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestHandle_PollVote は投票の正常処理を検証する。
func TestHandle_PollVote(t *testing.T) {
	t.Parallel()
	var broadcastCalled bool
	h := &messageHandler{
		pollVote: &mockPollVoter{
			voteFn: func(_ context.Context, _, _, _ string) error {
				return nil
			},
		},
		pollGet: &mockPollGetter{
			getFn: func(_ context.Context, _, _ string, _ []string, _ int, _ bool) (*poll.PollState, error) {
				return defaultPollState(), nil
			},
		},
		broadcaster: &mockMessageBroadcaster{
			sendFn: func(_ context.Context, _ string, _ []byte, _ string) error {
				broadcastCalled = true
				return nil
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_vote","pollId":"q1","visitorId":"v1","choice":"A"}`)
	resp, err := h.handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !broadcastCalled {
		t.Error("expected broadcast to be called")
	}
}

// TestHandle_PollVoteDuplicate は重複投票時の poll_error レスポンスを検証する。
func TestHandle_PollVoteDuplicate(t *testing.T) {
	t.Parallel()
	var sentPayload []byte
	h := &messageHandler{
		pollVote: &mockPollVoter{
			voteFn: func(_ context.Context, _, _, _ string) error {
				return poll.ErrDuplicateVote
			},
		},
		pollGet: &mockPollGetter{
			getFn: func(_ context.Context, _, _ string, _ []string, _ int, _ bool) (*poll.PollState, error) {
				return defaultPollState(), nil
			},
		},
		singleSender: &mockSingleSender{
			sendToOneFn: func(_ context.Context, _, _ string, payload []byte) error {
				sentPayload = payload
				return nil
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_vote","pollId":"q1","visitorId":"v1","choice":"A"}`)
	resp, err := h.handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var errResp pollErrorResponse
	if unmarshalErr := json.Unmarshal(sentPayload, &errResp); unmarshalErr != nil {
		t.Fatalf("unmarshal: %v", unmarshalErr)
	}
	if errResp.Type != "poll_error" {
		t.Errorf("expected type poll_error, got %s", errResp.Type)
	}
}

// TestHandle_PollVoteMaxExceeded は最大選択数超過時の poll_error レスポンスを検証する。
func TestHandle_PollVoteMaxExceeded(t *testing.T) {
	t.Parallel()
	h := &messageHandler{
		pollVote: &mockPollVoter{
			voteFn: func(_ context.Context, _, _, _ string) error {
				return poll.ErrMaxChoicesExceeded
			},
		},
		pollGet: &mockPollGetter{
			getFn: func(_ context.Context, _, _ string, _ []string, _ int, _ bool) (*poll.PollState, error) {
				return defaultPollState(), nil
			},
		},
		singleSender: &mockSingleSender{
			sendToOneFn: func(_ context.Context, _, _ string, _ []byte) error {
				return nil
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_vote","pollId":"q1","visitorId":"v1","choice":"C"}`)
	resp, err := h.handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// TestHandle_PollVoteInternalError は投票の内部エラーを検証する。
func TestHandle_PollVoteInternalError(t *testing.T) {
	t.Parallel()
	h := &messageHandler{
		pollVote: &mockPollVoter{
			voteFn: func(_ context.Context, _, _, _ string) error {
				return fmt.Errorf("internal error")
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_vote","pollId":"q1","visitorId":"v1","choice":"A"}`)
	_, err := h.handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestHandle_PollUnvote は投票取消の正常処理を検証する。
func TestHandle_PollUnvote(t *testing.T) {
	t.Parallel()
	h := &messageHandler{
		pollUnvote: &mockPollUnvoter{
			unvoteFn: func(_ context.Context, _, _, _ string) error {
				return nil
			},
		},
		pollGet: &mockPollGetter{
			getFn: func(_ context.Context, _, _ string, _ []string, _ int, _ bool) (*poll.PollState, error) {
				return defaultPollState(), nil
			},
		},
		broadcaster: &mockMessageBroadcaster{
			sendFn: func(_ context.Context, _ string, _ []byte, _ string) error {
				return nil
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_unvote","pollId":"q1","visitorId":"v1","choice":"A"}`)
	resp, err := h.handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// TestHandle_PollUnvoteNotFound は投票取消の ErrVoteNotFound を検証する。
func TestHandle_PollUnvoteNotFound(t *testing.T) {
	t.Parallel()
	h := &messageHandler{
		pollUnvote: &mockPollUnvoter{
			unvoteFn: func(_ context.Context, _, _, _ string) error {
				return poll.ErrVoteNotFound
			},
		},
		pollGet: &mockPollGetter{
			getFn: func(_ context.Context, _, _ string, _ []string, _ int, _ bool) (*poll.PollState, error) {
				return defaultPollState(), nil
			},
		},
		singleSender: &mockSingleSender{
			sendToOneFn: func(_ context.Context, _, _ string, _ []byte) error {
				return nil
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_unvote","pollId":"q1","visitorId":"v1","choice":"A"}`)
	resp, err := h.handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// TestHandle_PollUnvoteInternalError は投票取消の内部エラーを検証する。
func TestHandle_PollUnvoteInternalError(t *testing.T) {
	t.Parallel()
	h := &messageHandler{
		pollUnvote: &mockPollUnvoter{
			unvoteFn: func(_ context.Context, _, _, _ string) error {
				return fmt.Errorf("internal error")
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_unvote","pollId":"q1","visitorId":"v1","choice":"A"}`)
	_, err := h.handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestHandle_PollSwitch は投票変更の正常処理を検証する。
func TestHandle_PollSwitch(t *testing.T) {
	t.Parallel()
	h := &messageHandler{
		pollSwitch: &mockPollSwitcher{
			switchFn: func(_ context.Context, _, _, _, _ string) error {
				return nil
			},
		},
		pollGet: &mockPollGetter{
			getFn: func(_ context.Context, _, _ string, _ []string, _ int, _ bool) (*poll.PollState, error) {
				return defaultPollState(), nil
			},
		},
		broadcaster: &mockMessageBroadcaster{
			sendFn: func(_ context.Context, _ string, _ []byte, _ string) error {
				return nil
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_switch","pollId":"q1","visitorId":"v1","from":"A","to":"B"}`)
	resp, err := h.handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// TestHandle_PollSwitchVoteNotFound は投票変更の ErrVoteNotFound を検証する。
func TestHandle_PollSwitchVoteNotFound(t *testing.T) {
	t.Parallel()
	h := &messageHandler{
		pollSwitch: &mockPollSwitcher{
			switchFn: func(_ context.Context, _, _, _, _ string) error {
				return poll.ErrVoteNotFound
			},
		},
		pollGet: &mockPollGetter{
			getFn: func(_ context.Context, _, _ string, _ []string, _ int, _ bool) (*poll.PollState, error) {
				return defaultPollState(), nil
			},
		},
		singleSender: &mockSingleSender{
			sendToOneFn: func(_ context.Context, _, _ string, _ []byte) error {
				return nil
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_switch","pollId":"q1","visitorId":"v1","from":"A","to":"B"}`)
	resp, err := h.handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// TestHandle_PollSwitchInternalError は投票変更の内部エラーを検証する。
func TestHandle_PollSwitchInternalError(t *testing.T) {
	t.Parallel()
	h := &messageHandler{
		pollSwitch: &mockPollSwitcher{
			switchFn: func(_ context.Context, _, _, _, _ string) error {
				return fmt.Errorf("internal error")
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_switch","pollId":"q1","visitorId":"v1","from":"A","to":"B"}`)
	_, err := h.handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestHandle_UnknownType は不明メッセージタイプ時のエラーレスポンスを検証する。
func TestHandle_UnknownType(t *testing.T) {
	t.Parallel()
	var sentPayload []byte
	h := &messageHandler{
		singleSender: &mockSingleSender{
			sendToOneFn: func(_ context.Context, _, _ string, payload []byte) error {
				sentPayload = payload
				return nil
			},
		},
	}
	req := newRequest("conn1", `{"type":"unknown"}`)
	resp, err := h.handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var errResp errorResponse
	if unmarshalErr := json.Unmarshal(sentPayload, &errResp); unmarshalErr != nil {
		t.Fatalf("unmarshal: %v", unmarshalErr)
	}
	if errResp.Error != "unknown message type" {
		t.Errorf("expected unknown message type, got %s", errResp.Error)
	}
}

// TestHandle_InvalidJSON は不正 JSON のエラーを検証する。
func TestHandle_InvalidJSON(t *testing.T) {
	t.Parallel()
	h := &messageHandler{}
	req := newRequest("conn1", `invalid json`)
	_, err := h.handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestHandle_PollVoteRefreshError は投票後の状態取得エラーを検証する。
func TestHandle_PollVoteRefreshError(t *testing.T) {
	t.Parallel()
	h := &messageHandler{
		pollVote: &mockPollVoter{
			voteFn: func(_ context.Context, _, _, _ string) error {
				return nil
			},
		},
		pollGet: &mockPollGetter{
			getFn: func(_ context.Context, _, _ string, _ []string, _ int, _ bool) (*poll.PollState, error) {
				return nil, fmt.Errorf("get error")
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_vote","pollId":"q1","visitorId":"v1","choice":"A"}`)
	_, err := h.handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestHandle_PollVoteBroadcastError は投票後のブロードキャストエラーを検証する。
func TestHandle_PollVoteBroadcastError(t *testing.T) {
	t.Parallel()
	h := &messageHandler{
		pollVote: &mockPollVoter{
			voteFn: func(_ context.Context, _, _, _ string) error {
				return nil
			},
		},
		pollGet: &mockPollGetter{
			getFn: func(_ context.Context, _, _ string, _ []string, _ int, _ bool) (*poll.PollState, error) {
				return defaultPollState(), nil
			},
		},
		broadcaster: &mockMessageBroadcaster{
			sendFn: func(_ context.Context, _ string, _ []byte, _ string) error {
				return fmt.Errorf("broadcast error")
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_vote","pollId":"q1","visitorId":"v1","choice":"A"}`)
	_, err := h.handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestHandle_PollErrorGetStateError は業務エラー後の状態取得失敗を検証する。
func TestHandle_PollErrorGetStateError(t *testing.T) {
	t.Parallel()
	h := &messageHandler{
		pollVote: &mockPollVoter{
			voteFn: func(_ context.Context, _, _, _ string) error {
				return poll.ErrDuplicateVote
			},
		},
		pollGet: &mockPollGetter{
			getFn: func(_ context.Context, _, _ string, _ []string, _ int, _ bool) (*poll.PollState, error) {
				return nil, fmt.Errorf("get error")
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_vote","pollId":"q1","visitorId":"v1","choice":"A"}`)
	_, err := h.handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestHandle_PollErrorSendError は poll_error 送信失敗を検証する。
func TestHandle_PollErrorSendError(t *testing.T) {
	t.Parallel()
	h := &messageHandler{
		pollVote: &mockPollVoter{
			voteFn: func(_ context.Context, _, _, _ string) error {
				return poll.ErrDuplicateVote
			},
		},
		pollGet: &mockPollGetter{
			getFn: func(_ context.Context, _, _ string, _ []string, _ int, _ bool) (*poll.PollState, error) {
				return defaultPollState(), nil
			},
		},
		singleSender: &mockSingleSender{
			sendToOneFn: func(_ context.Context, _, _ string, _ []byte) error {
				return fmt.Errorf("send error")
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_vote","pollId":"q1","visitorId":"v1","choice":"A"}`)
	_, err := h.handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestHandle_SendErrorMarshalError はエラーレスポンスの marshal 失敗を検証する。
// jsonMarshal を差し替えるため非 parallel。
func TestHandle_SendErrorMarshalError(t *testing.T) {
	orig := jsonMarshal
	defer func() { jsonMarshal = orig }()
	jsonMarshal = func(_ any) ([]byte, error) {
		return nil, fmt.Errorf("marshal error")
	}
	h := &messageHandler{
		slideSync: &mockSlideSyncDispatcher{
			handleFn: func(_ context.Context, _, _ string, _ int) error {
				return fmt.Errorf("slide error")
			},
		},
	}
	req := newRequest("conn1", `{"type":"slide_sync","page":1}`)
	_, err := h.handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestHandle_PollErrorMarshalError は poll_error の marshal 失敗を検証する。
// jsonMarshal を差し替えるため非 parallel。
func TestHandle_PollErrorMarshalError(t *testing.T) {
	orig := jsonMarshal
	defer func() { jsonMarshal = orig }()
	jsonMarshal = func(_ any) ([]byte, error) {
		return nil, fmt.Errorf("marshal error")
	}
	h := &messageHandler{
		pollVote: &mockPollVoter{
			voteFn: func(_ context.Context, _, _, _ string) error {
				return poll.ErrDuplicateVote
			},
		},
		pollGet: &mockPollGetter{
			getFn: func(_ context.Context, _, _ string, _ []string, _ int, _ bool) (*poll.PollState, error) {
				return defaultPollState(), nil
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_vote","pollId":"q1","visitorId":"v1","choice":"A"}`)
	_, err := h.handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestHandle_PollStateMarshalError は poll_state の marshal 失敗を検証する。
// jsonMarshal を差し替えるため非 parallel。
func TestHandle_PollStateMarshalError(t *testing.T) {
	orig := jsonMarshal
	defer func() { jsonMarshal = orig }()
	jsonMarshal = func(_ any) ([]byte, error) {
		return nil, fmt.Errorf("marshal error")
	}
	h := &messageHandler{
		pollVote: &mockPollVoter{
			voteFn: func(_ context.Context, _, _, _ string) error {
				return nil
			},
		},
		pollGet: &mockPollGetter{
			getFn: func(_ context.Context, _, _ string, _ []string, _ int, _ bool) (*poll.PollState, error) {
				return defaultPollState(), nil
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_vote","pollId":"q1","visitorId":"v1","choice":"A"}`)
	_, err := h.handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestHandle_SendErrorSendToOneError はエラーレスポンスの SendToOne 失敗を検証する。
func TestHandle_SendErrorSendToOneError(t *testing.T) {
	t.Parallel()
	h := &messageHandler{
		slideSync: &mockSlideSyncDispatcher{
			handleFn: func(_ context.Context, _, _ string, _ int) error {
				return fmt.Errorf("slide error")
			},
		},
		singleSender: &mockSingleSender{
			sendToOneFn: func(_ context.Context, _, _ string, _ []byte) error {
				return fmt.Errorf("send error")
			},
		},
	}
	req := newRequest("conn1", `{"type":"slide_sync","page":1}`)
	_, err := h.handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestHandle_PollGetBroadcastError はアンケート取得後のブロードキャストエラーを検証する。
func TestHandle_PollGetBroadcastError(t *testing.T) {
	t.Parallel()
	h := &messageHandler{
		connGetter: &mockConnectionGetter{
			getFn: func(_ context.Context, _, _ string) (*connection.Connection, error) {
				return &connection.Connection{Role: "viewer"}, nil
			},
		},
		pollGet: &mockPollGetter{
			getFn: func(_ context.Context, _, _ string, _ []string, _ int, _ bool) (*poll.PollState, error) {
				return defaultPollState(), nil
			},
		},
		broadcaster: &mockMessageBroadcaster{
			sendFn: func(_ context.Context, _ string, _ []byte, _ string) error {
				return fmt.Errorf("broadcast error")
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_get","pollId":"q1","visitorId":"v1"}`)
	_, err := h.handle(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestIsPollBusinessError は業務エラー判定を検証する。
func TestIsPollBusinessError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"ErrMaxChoicesExceeded", poll.ErrMaxChoicesExceeded, true},
		{"ErrDuplicateVote", poll.ErrDuplicateVote, true},
		{"ErrVoteNotFound", poll.ErrVoteNotFound, true},
		{"generic error", fmt.Errorf("other"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isPollBusinessError(tt.err); got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

// TestRun_Success は run 関数の正常処理を検証する。
func TestRun_Success(t *testing.T) {
	origLoadConfig := loadConfig
	origStartLambda := startLambda
	defer func() {
		loadConfig = origLoadConfig
		startLambda = origStartLambda
	}()

	loadConfig = func(_ context.Context, _ ...func(*config.LoadOptions) error) (aws.Config, error) {
		return aws.Config{}, nil
	}
	startLambda = func(_ interface{}) {}

	t.Setenv("CONNECTIONS_TABLE", "conn-table")
	t.Setenv("POLL_VOTES_TABLE", "poll-table")
	t.Setenv("APIGW_ENDPOINT", "")

	if err := run(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRun_SuccessWithEndpoint は APIGW_ENDPOINT 設定時の run 関数を検証する。
func TestRun_SuccessWithEndpoint(t *testing.T) {
	origLoadConfig := loadConfig
	origStartLambda := startLambda
	defer func() {
		loadConfig = origLoadConfig
		startLambda = origStartLambda
	}()

	loadConfig = func(_ context.Context, _ ...func(*config.LoadOptions) error) (aws.Config, error) {
		return aws.Config{}, nil
	}
	startLambda = func(_ interface{}) {}

	t.Setenv("CONNECTIONS_TABLE", "conn-table")
	t.Setenv("POLL_VOTES_TABLE", "poll-table")
	t.Setenv("APIGW_ENDPOINT", "https://example.com")

	if err := run(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRun_ConfigError は AWS 設定読み込みエラーを検証する。
func TestRun_ConfigError(t *testing.T) {
	origLoadConfig := loadConfig
	defer func() { loadConfig = origLoadConfig }()

	loadConfig = func(_ context.Context, _ ...func(*config.LoadOptions) error) (aws.Config, error) {
		return aws.Config{}, fmt.Errorf("config error")
	}

	if err := run(); err == nil {
		t.Fatal("expected error")
	}
}

// TestMain_Success は main 関数の正常処理を検証する。
func TestMain_Success(t *testing.T) {
	origRunFn := runFn
	defer func() { runFn = origRunFn }()

	runFn = func() error { return nil }
	main()
}

// TestViewerCountAdapter_SendToOne は viewerCountAdapter の動作を検証する。
func TestViewerCountAdapter_SendToOne(t *testing.T) {
	t.Parallel()
	var capturedRoom, capturedConnID string
	var capturedPayload []byte
	mock := &mockSingleSender{
		sendToOneFn: func(_ context.Context, r, connID string, payload []byte) error {
			capturedRoom = r
			capturedConnID = connID
			capturedPayload = payload
			return nil
		},
	}
	adapter := &viewerCountAdapter{sender: mock}
	err := adapter.SendToOne(context.Background(), "ignored-room", "conn1", []byte("test"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedRoom != "default" {
		t.Errorf("expected room default, got %s", capturedRoom)
	}
	if capturedConnID != "conn1" {
		t.Errorf("expected conn1, got %s", capturedConnID)
	}
	if string(capturedPayload) != "test" {
		t.Errorf("expected test, got %s", string(capturedPayload))
	}
}

// TestViewerCountAdapter_SendToOneError は viewerCountAdapter のエラーを検証する。
func TestViewerCountAdapter_SendToOneError(t *testing.T) {
	t.Parallel()
	mock := &mockSingleSender{
		sendToOneFn: func(_ context.Context, _, _ string, _ []byte) error {
			return fmt.Errorf("send error")
		},
	}
	adapter := &viewerCountAdapter{sender: mock}
	err := adapter.SendToOne(context.Background(), "ignored-room", "conn1", []byte("test"))
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestHandlePollGet_MissingPollID は pollId 未指定時のエラーを検証する。
func TestHandlePollGet_MissingPollID(t *testing.T) {
	t.Parallel()
	var sentPayload []byte
	h := &messageHandler{
		singleSender: &mockSingleSender{
			sendToOneFn: func(_ context.Context, _, _ string, payload []byte) error {
				sentPayload = payload
				return nil
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_get"}`)
	resp, err := h.handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !contains(string(sentPayload), "pollId is required") {
		t.Errorf("expected validation error, got %s", string(sentPayload))
	}
}

// TestHandlePollVote_MissingFields は pollId/choice 未指定時のエラーを検証する。
func TestHandlePollVote_MissingFields(t *testing.T) {
	t.Parallel()
	var sentPayload []byte
	h := &messageHandler{
		singleSender: &mockSingleSender{
			sendToOneFn: func(_ context.Context, _, _ string, payload []byte) error {
				sentPayload = payload
				return nil
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_vote","pollId":"q1"}`)
	resp, err := h.handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !contains(string(sentPayload), "required") {
		t.Errorf("expected validation error, got %s", string(sentPayload))
	}
}

// TestHandlePollUnvote_MissingFields は pollId/choice 未指定時のエラーを検証する。
func TestHandlePollUnvote_MissingFields(t *testing.T) {
	t.Parallel()
	var sentPayload []byte
	h := &messageHandler{
		singleSender: &mockSingleSender{
			sendToOneFn: func(_ context.Context, _, _ string, payload []byte) error {
				sentPayload = payload
				return nil
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_unvote","pollId":"q1"}`)
	resp, err := h.handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !contains(string(sentPayload), "required") {
		t.Errorf("expected validation error, got %s", string(sentPayload))
	}
}

// TestHandlePollSwitch_MissingFields は pollId/from/to 未指定時のエラーを検証する。
func TestHandlePollSwitch_MissingFields(t *testing.T) {
	t.Parallel()
	var sentPayload []byte
	h := &messageHandler{
		singleSender: &mockSingleSender{
			sendToOneFn: func(_ context.Context, _, _ string, payload []byte) error {
				sentPayload = payload
				return nil
			},
		},
	}
	req := newRequest("conn1", `{"type":"poll_switch","pollId":"q1","from":"A"}`)
	resp, err := h.handle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !contains(string(sentPayload), "required") {
		t.Errorf("expected validation error, got %s", string(sentPayload))
	}
}

// contains は文字列に部分文字列が含まれるかを判定するヘルパー。
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

// containsSubstr は strings.Contains の代替。strings パッケージを import せずに使う。
func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestMain_Error は main 関数のエラー処理を検証する。
func TestMain_Error(t *testing.T) {
	origRunFn := runFn
	origFatalf := fatalf
	defer func() {
		runFn = origRunFn
		fatalf = origFatalf
	}()

	runFn = func() error { return fmt.Errorf("run error") }
	var called bool
	fatalf = func(_ string, _ ...interface{}) { called = true }
	main()
	if !called {
		t.Error("expected fatalf to be called")
	}
}
