// Package main は WebSocket $default ルートの Lambda ハンドラーを提供する。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/ogadra/20260327-cli-demo/presenter/internal/broadcast"
	"github.com/ogadra/20260327-cli-demo/presenter/internal/connection"
	"github.com/ogadra/20260327-cli-demo/presenter/internal/handson"
	"github.com/ogadra/20260327-cli-demo/presenter/internal/poll"
	"github.com/ogadra/20260327-cli-demo/presenter/internal/slidesync"
	"github.com/ogadra/20260327-cli-demo/presenter/internal/viewercount"
)

// fatalf はエラー時の終了処理。テスト時に差し替える。
var fatalf = log.Fatalf

// startLambda は lambda.Start のラッパー。テスト時に差し替える。
var startLambda = lambda.Start

// runFn は run のラッパー。テスト時に差し替える。
var runFn = run

// loadConfig は AWS 設定読み込みのラッパー。テスト時に差し替える。
var loadConfig = config.LoadDefaultConfig

// jsonMarshal は JSON エンコードのラッパー。テスト時に差し替える。
var jsonMarshal = json.Marshal

// room は WebSocket 接続のグループ識別子。
const room = "default"

// slideSyncDispatcher はスライド同期ハンドラーインターフェース。
type slideSyncDispatcher interface {
	// Handle はスライドページ同期を処理する。
	Handle(ctx context.Context, room, connectionID string, page int) error
}

// handsOnDispatcher はハンズオン指示ハンドラーインターフェース。
type handsOnDispatcher interface {
	// Handle はハンズオン指示を処理する。
	Handle(ctx context.Context, room, connectionID, instruction, placeholder string) error
}

// viewerCountDispatcher は接続数通知ハンドラーインターフェース。
type viewerCountDispatcher interface {
	// Handle は接続数を返信する。
	Handle(ctx context.Context, room, connectionID string) error
}

// pollGetter はアンケート状態取得インターフェース。
type pollGetter interface {
	// Get はアンケートの状態を取得する。
	Get(ctx context.Context, pollID, visitorID string, options []string, maxChoices int, isPresenter bool) (*poll.PollState, error)
}

// pollVoter は投票インターフェース。
type pollVoter interface {
	// Vote は投票を記録する。
	Vote(ctx context.Context, pollID, visitorID, choice string) error
}

// pollUnvoter は投票取消インターフェース。
type pollUnvoter interface {
	// Unvote は投票を取り消す。
	Unvote(ctx context.Context, pollID, visitorID, choice string) error
}

// pollSwitcher は投票変更インターフェース。
type pollSwitcher interface {
	// Switch は投票を変更する。
	Switch(ctx context.Context, pollID, visitorID, from, to string) error
}

// connectionGetter は接続情報取得インターフェース。
type connectionGetter interface {
	// Get は接続情報を取得する。
	Get(ctx context.Context, room, connectionID string) (*connection.Connection, error)
}

// messageBroadcaster はメッセージ配信インターフェース。
type messageBroadcaster interface {
	// Send は room 内の全接続にメッセージを配信する。
	Send(ctx context.Context, room string, payload []byte, excludeConnectionID string) error
}

// singleSender は単一接続への送信インターフェース。
type singleSender interface {
	// SendToOne は単一の接続にメッセージを送信する。
	SendToOne(ctx context.Context, room, connectionID string, payload []byte) error
}

// handleDepsFactory はリクエストごとの依存を生成するファクトリー。
type handleDepsFactory func(domainName, stage string) handleDeps

// handleDeps はリクエストごとに生成される依存。
type handleDeps struct {
	slideSync    slideSyncDispatcher
	handsOn      handsOnDispatcher
	viewerCount  viewerCountDispatcher
	broadcaster  messageBroadcaster
	singleSender singleSender
}

// messageHandler は $default イベントを処理するハンドラー。
type messageHandler struct {
	pollGet    pollGetter
	pollVote   pollVoter
	pollUnvote pollUnvoter
	pollSwitch pollSwitcher
	connGetter connectionGetter
	newDeps    handleDepsFactory
}

// incomingMessage はクライアントからの受信メッセージ。
type incomingMessage struct {
	Type        string   `json:"type"`
	Page        int      `json:"page"`
	Instruction string   `json:"instruction"`
	Placeholder string   `json:"placeholder"`
	PollID      string   `json:"pollId"`
	Options     []string `json:"options"`
	MaxChoices  int      `json:"maxChoices"`
	Choice      string   `json:"choice"`
	From        string   `json:"from"`
	To          string   `json:"to"`
}

// pollStateResponse はアンケート状態レスポンス。
type pollStateResponse struct {
	Type       string         `json:"type"`
	PollID     string         `json:"pollId"`
	Options    []string       `json:"options"`
	MaxChoices int            `json:"maxChoices"`
	Votes      map[string]int `json:"votes"`
	MyChoices  []string       `json:"myChoices"`
}

// pollErrorResponse はアンケートエラーレスポンス。
type pollErrorResponse struct {
	Type      string         `json:"type"`
	PollID    string         `json:"pollId"`
	Error     string         `json:"error"`
	Votes     map[string]int `json:"votes"`
	MyChoices []string       `json:"myChoices"`
}

// errorResponse はエラーレスポンス。
type errorResponse struct {
	Type  string `json:"type"`
	Error string `json:"error"`
}

// handle は $default イベントを処理する。
func (h *messageHandler) handle(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	connectionID := req.RequestContext.ConnectionID
	deps := h.newDeps(req.RequestContext.DomainName, req.RequestContext.Stage)

	var msg incomingMessage
	if err := json.Unmarshal([]byte(req.Body), &msg); err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 400}, fmt.Errorf("unmarshal body: %w", err)
	}

	switch msg.Type {
	case "slide_sync":
		return h.handleSlideSync(ctx, connectionID, msg, deps)
	case "hands_on":
		return h.handleHandsOn(ctx, connectionID, msg, deps)
	case "viewer_count":
		return h.handleViewerCount(ctx, connectionID, deps)
	case "poll_get":
		return h.handlePollGet(ctx, connectionID, msg, deps)
	case "poll_vote":
		return h.handlePollVote(ctx, connectionID, msg, deps)
	case "poll_unvote":
		return h.handlePollUnvote(ctx, connectionID, msg, deps)
	case "poll_switch":
		return h.handlePollSwitch(ctx, connectionID, msg, deps)
	default:
		return h.sendError(ctx, connectionID, "unknown message type", deps)
	}
}

// handleSlideSync はスライド同期メッセージを処理する。
func (h *messageHandler) handleSlideSync(ctx context.Context, connectionID string, msg incomingMessage, deps handleDeps) (events.APIGatewayProxyResponse, error) {
	if err := deps.slideSync.Handle(ctx, room, connectionID, msg.Page); err != nil {
		return h.sendError(ctx, connectionID, err.Error(), deps)
	}
	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

// handleHandsOn はハンズオン指示メッセージを処理する。
func (h *messageHandler) handleHandsOn(ctx context.Context, connectionID string, msg incomingMessage, deps handleDeps) (events.APIGatewayProxyResponse, error) {
	if err := deps.handsOn.Handle(ctx, room, connectionID, msg.Instruction, msg.Placeholder); err != nil {
		return h.sendError(ctx, connectionID, err.Error(), deps)
	}
	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

// handleViewerCount は接続数通知メッセージを処理する。
func (h *messageHandler) handleViewerCount(ctx context.Context, connectionID string, deps handleDeps) (events.APIGatewayProxyResponse, error) {
	if err := deps.viewerCount.Handle(ctx, room, connectionID); err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("viewer_count: %w", err)
	}
	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

// handlePollGet はアンケート状態取得メッセージを処理する。
func (h *messageHandler) handlePollGet(ctx context.Context, connectionID string, msg incomingMessage, deps handleDeps) (events.APIGatewayProxyResponse, error) {
	if msg.PollID == "" {
		return h.sendError(ctx, connectionID, "pollId is required", deps)
	}

	conn, err := h.connGetter.Get(ctx, room, connectionID)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("get connection: %w", err)
	}
	isPresenter := conn.Role == "presenter"

	state, err := h.pollGet.Get(ctx, msg.PollID, connectionID, msg.Options, msg.MaxChoices, isPresenter)
	if err != nil {
		if isPollBusinessError(err) {
			return h.sendError(ctx, connectionID, err.Error(), deps)
		}
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("poll_get: %w", err)
	}

	return h.sendPollState(ctx, connectionID, state, deps)
}

// handlePollVote は投票メッセージを処理する。
func (h *messageHandler) handlePollVote(ctx context.Context, connectionID string, msg incomingMessage, deps handleDeps) (events.APIGatewayProxyResponse, error) {
	if msg.PollID == "" || msg.Choice == "" {
		return h.sendError(ctx, connectionID, "pollId and choice are required", deps)
	}
	err := h.pollVote.Vote(ctx, msg.PollID, connectionID, msg.Choice)
	if err != nil {
		return h.handlePollError(ctx, connectionID, msg.PollID, connectionID, err, deps)
	}
	return h.refreshAndBroadcastPoll(ctx, msg.PollID, connectionID, deps)
}

// handlePollUnvote は投票取消メッセージを処理する。
func (h *messageHandler) handlePollUnvote(ctx context.Context, connectionID string, msg incomingMessage, deps handleDeps) (events.APIGatewayProxyResponse, error) {
	if msg.PollID == "" || msg.Choice == "" {
		return h.sendError(ctx, connectionID, "pollId and choice are required", deps)
	}
	err := h.pollUnvote.Unvote(ctx, msg.PollID, connectionID, msg.Choice)
	if err != nil {
		return h.handlePollError(ctx, connectionID, msg.PollID, connectionID, err, deps)
	}
	return h.refreshAndBroadcastPoll(ctx, msg.PollID, connectionID, deps)
}

// handlePollSwitch は投票変更メッセージを処理する。
func (h *messageHandler) handlePollSwitch(ctx context.Context, connectionID string, msg incomingMessage, deps handleDeps) (events.APIGatewayProxyResponse, error) {
	if msg.PollID == "" || msg.From == "" || msg.To == "" {
		return h.sendError(ctx, connectionID, "pollId, from, and to are required", deps)
	}
	if msg.From == msg.To {
		return h.sendError(ctx, connectionID, "from and to must be different", deps)
	}
	err := h.pollSwitch.Switch(ctx, msg.PollID, connectionID, msg.From, msg.To)
	if err != nil {
		return h.handlePollError(ctx, connectionID, msg.PollID, connectionID, err, deps)
	}
	return h.refreshAndBroadcastPoll(ctx, msg.PollID, connectionID, deps)
}

// handlePollError はアンケート操作エラーを処理する。業務エラーは poll_error を送信元に返す。
// ErrPollNotFound の場合は状態再取得をスキップしてエラーメッセージのみ返す。
func (h *messageHandler) handlePollError(ctx context.Context, connectionID, pollID, visitorID string, err error, deps handleDeps) (events.APIGatewayProxyResponse, error) {
	if !isPollBusinessError(err) {
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("poll operation: %w", err)
	}

	if errors.Is(err, poll.ErrPollNotFound) {
		return h.sendError(ctx, connectionID, err.Error(), deps)
	}

	state, getErr := h.pollGet.Get(ctx, pollID, visitorID, nil, 0, false)
	if getErr != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("get poll state for error response: %w", getErr)
	}

	resp := pollErrorResponse{
		Type:      "poll_error",
		PollID:    pollID,
		Error:     err.Error(),
		Votes:     state.Votes,
		MyChoices: state.MyChoices,
	}
	payload, marshalErr := jsonMarshal(resp)
	if marshalErr != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("marshal poll_error: %w", marshalErr)
	}

	if sendErr := deps.singleSender.SendToOne(ctx, room, connectionID, payload); sendErr != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("send poll_error: %w", sendErr)
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

// isPollBusinessError はアンケート業務エラーかどうかを判定する。
func isPollBusinessError(err error) bool {
	return errors.Is(err, poll.ErrMaxChoicesExceeded) ||
		errors.Is(err, poll.ErrDuplicateVote) ||
		errors.Is(err, poll.ErrVoteNotFound) ||
		errors.Is(err, poll.ErrInvalidChoice) ||
		errors.Is(err, poll.ErrPollNotFound)
}

// refreshAndBroadcastPoll は最新のアンケート状態を取得してブロードキャストする。
func (h *messageHandler) refreshAndBroadcastPoll(ctx context.Context, pollID, visitorID string, deps handleDeps) (events.APIGatewayProxyResponse, error) {
	state, err := h.pollGet.Get(ctx, pollID, visitorID, nil, 0, false)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("refresh poll state: %w", err)
	}
	return h.broadcastPollState(ctx, state, deps)
}

// sendPollState はアンケート状態を要求元の接続にのみ送信する。
func (h *messageHandler) sendPollState(ctx context.Context, connectionID string, state *poll.PollState, deps handleDeps) (events.APIGatewayProxyResponse, error) {
	resp := pollStateResponse{
		Type:       "poll_state",
		PollID:     state.PollID,
		Options:    state.Options,
		MaxChoices: state.MaxChoices,
		Votes:      state.Votes,
		MyChoices:  state.MyChoices,
	}
	payload, err := jsonMarshal(resp)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("marshal poll_state: %w", err)
	}

	if err := deps.singleSender.SendToOne(ctx, room, connectionID, payload); err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("send poll_state: %w", err)
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

// broadcastPollState はアンケート状態を全接続にブロードキャストする。
func (h *messageHandler) broadcastPollState(ctx context.Context, state *poll.PollState, deps handleDeps) (events.APIGatewayProxyResponse, error) {
	resp := pollStateResponse{
		Type:       "poll_state",
		PollID:     state.PollID,
		Options:    state.Options,
		MaxChoices: state.MaxChoices,
		Votes:      state.Votes,
		MyChoices:  state.MyChoices,
	}
	payload, err := jsonMarshal(resp)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("marshal poll_state: %w", err)
	}

	if err := deps.broadcaster.Send(ctx, room, payload, ""); err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("broadcast poll_state: %w", err)
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

// sendError はエラーレスポンスを送信元に返す。
func (h *messageHandler) sendError(ctx context.Context, connectionID, errMsg string, deps handleDeps) (events.APIGatewayProxyResponse, error) {
	resp := errorResponse{Type: "error", Error: errMsg}
	payload, err := jsonMarshal(resp)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("marshal error: %w", err)
	}

	if err := deps.singleSender.SendToOne(ctx, room, connectionID, payload); err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("send error: %w", err)
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

// newAPIGWEndpoint は requestContext の domainName と stage からエンドポイント URL を構築する。
func newAPIGWEndpoint(domainName, stage string) string {
	return fmt.Sprintf("https://%s/%s", domainName, stage)
}

// run は依存を初期化し Lambda ハンドラーを起動する。
func run() error {
	ctx := context.Background()
	cfg, err := loadConfig(ctx)
	if err != nil {
		return fmt.Errorf("load aws config: %w", err)
	}

	ddbClient := dynamodb.NewFromConfig(cfg)
	connTable := os.Getenv("CONNECTIONS_TABLE")
	if connTable == "" {
		return fmt.Errorf("CONNECTIONS_TABLE environment variable is required")
	}
	pollTable := os.Getenv("POLL_VOTES_TABLE")
	if pollTable == "" {
		return fmt.Errorf("POLL_VOTES_TABLE environment variable is required")
	}

	connStore := connection.NewStore(ddbClient, connTable)
	pollStore := poll.NewStore(ddbClient, pollTable)

	h := &messageHandler{
		pollGet:    pollStore,
		pollVote:   pollStore,
		pollUnvote: pollStore,
		pollSwitch: pollStore,
		connGetter: connStore,
		newDeps: func(domainName, stage string) handleDeps {
			endpoint := newAPIGWEndpoint(domainName, stage)
			apigwClient := apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
				o.BaseEndpoint = &endpoint
			})
			b := broadcast.NewBroadcaster(apigwClient, connStore, connStore)
			return handleDeps{
				slideSync:    slidesync.NewHandler(connStore, b),
				handsOn:      handson.NewHandler(connStore, b),
				viewerCount:  viewercount.NewHandler(connStore, &viewerCountAdapter{sender: b}),
				broadcaster:  b,
				singleSender: b,
			}
		},
	}

	startLambda(h.handle)
	return nil
}

// viewerCountAdapter は singleSender を viewercount.SingleSender に適合させるアダプター。
type viewerCountAdapter struct {
	sender singleSender
}

// SendToOne は viewercount.SingleSender を満たす。room 引数を無視し固定の room 定数で委譲する。
func (a *viewerCountAdapter) SendToOne(ctx context.Context, _, connectionID string, payload []byte) error {
	return a.sender.SendToOne(ctx, room, connectionID, payload)
}

// main は message Lambda のエントリポイント。
func main() {
	if err := runFn(); err != nil {
		fatalf("message: %v", err)
	}
}
