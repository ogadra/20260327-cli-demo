// Package model はドメインモデルと状態遷移ロジックを提供する。
package model

import "fmt"

// RunnerStatus は Runner の状態を表す型。
type RunnerStatus string

const (
	// StatusIdle はアイドル状態を表す。セッション未割当。
	StatusIdle RunnerStatus = "idle"
	// StatusBusy はビジー状態を表す。セッション処理中。
	StatusBusy RunnerStatus = "busy"
	// StatusDead は停止状態を表す。利用不可。
	StatusDead RunnerStatus = "dead"
)

// allStatuses は全ての有効な RunnerStatus を保持する。
var allStatuses = map[RunnerStatus]bool{
	StatusIdle: true,
	StatusBusy: true,
	StatusDead: true,
}

// transitions は各状態から遷移可能な状態の集合を定義する。
var transitions = map[RunnerStatus]map[RunnerStatus]bool{
	StatusIdle: {StatusBusy: true, StatusDead: true},
	StatusBusy: {StatusIdle: true, StatusDead: true},
	StatusDead: {},
}

// Runner は broker が管理する runner のドメインモデル。
type Runner struct {
	// RunnerID は runner の一意識別子であり DynamoDB の PK。
	RunnerID string `dynamodbav:"runnerId"`
	// Status は状態マシンの現在値。
	Status RunnerStatus `dynamodbav:"status"`
	// CurrentSessionID は busy 時のセッション ID。sparse GSI session-index のキー。
	CurrentSessionID string `dynamodbav:"currentSessionId,omitempty"`
	// IdleBucket は idle 時のバケット値。sparse GSI idle-index のキー。
	IdleBucket string `dynamodbav:"idleBucket,omitempty"`
}

// SparseAttributes は状態に応じた sparse 属性値を返す。
// idle 時は idleBucket を設定し currentSessionID をクリアする。
// busy 時は currentSessionID を設定し idleBucket をクリアする。
// その他の状態では両方クリアする。
func SparseAttributes(status RunnerStatus, sessionID string, bucket string) (currentSessionID string, idleBucket string) {
	switch status {
	case StatusIdle:
		return "", bucket
	case StatusBusy:
		return sessionID, ""
	default:
		return "", ""
	}
}

// CanTransitionTo は from から to への状態遷移が許可されているかを返す。
func CanTransitionTo(from, to RunnerStatus) bool {
	allowed, ok := transitions[from]
	if !ok {
		return false
	}
	return allowed[to]
}

// ErrInvalidTransition は無効な状態遷移を示すエラー。
type ErrInvalidTransition struct {
	From RunnerStatus
	To   RunnerStatus
}

// Error は ErrInvalidTransition のエラーメッセージを返す。
func (e *ErrInvalidTransition) Error() string {
	return fmt.Sprintf("invalid transition from %s to %s", e.From, e.To)
}

// ValidateTransition は from から to への状態遷移を検証し、無効な場合はエラーを返す。
func ValidateTransition(from, to RunnerStatus) error {
	if !CanTransitionTo(from, to) {
		return &ErrInvalidTransition{From: from, To: to}
	}
	return nil
}

// IsValidStatus は文字列が有効な RunnerStatus かを返す。
func IsValidStatus(s string) bool {
	return allStatuses[RunnerStatus(s)]
}

// IsTerminal は指定の状態が終端状態かを返す。
func IsTerminal(s RunnerStatus) bool {
	return s == StatusDead
}
