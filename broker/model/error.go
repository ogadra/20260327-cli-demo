// Package model はドメインモデルを提供する。
package model

const (
	// CodeNoIdleRunner は idle 状態の runner が存在しない場合のエラーコード。
	CodeNoIdleRunner = "NO_IDLE_RUNNER"
	// CodeSessionNotFound は指定されたセッションが見つからない場合のエラーコード。
	CodeSessionNotFound = "SESSION_NOT_FOUND"
	// CodeRunnerNotFound は指定された runner が見つからない場合のエラーコード。
	CodeRunnerNotFound = "RUNNER_NOT_FOUND"
	// CodeInvalidRequest はリクエストが不正な場合のエラーコード。
	CodeInvalidRequest = "INVALID_REQUEST"
	// CodeInternalError は内部エラーのエラーコード。
	CodeInternalError = "INTERNAL_ERROR"
)

// ErrorResponse は API エラーレスポンスの構造体。
type ErrorResponse struct {
	// Code はエラーコード。
	Code string `json:"code"`
	// Message はエラーメッセージ。
	Message string `json:"message"`
	// RequestID はリクエスト ID。
	RequestID string `json:"requestId"`
}
