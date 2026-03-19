// Package model はドメインモデルを提供する。
package model

// RunnerStatus は Runner の状態を表す型。
type RunnerStatus string

const (
	// StatusIdle はアイドル状態を表す。セッション未割当。
	StatusIdle RunnerStatus = "idle"
	// StatusBusy はビジー状態を表す。セッション処理中。
	StatusBusy RunnerStatus = "busy"
)

// allStatuses は全ての有効な RunnerStatus を保持する。
var allStatuses = map[RunnerStatus]bool{
	StatusIdle: true,
	StatusBusy: true,
}

// Runner は broker が管理する runner のドメインモデル。
// runner は使い捨てであり、セッション終了時または異常終了時はレコードごと削除する。
type Runner struct {
	// RunnerID は runner の一意識別子であり DynamoDB の PK。
	RunnerID string `dynamodbav:"runnerId"`
	// Status は現在の状態。
	Status RunnerStatus `dynamodbav:"status"`
	// CurrentSessionID は busy 時のセッション ID。sparse GSI session-index のキー。
	CurrentSessionID string `dynamodbav:"currentSessionId,omitempty"`
	// IdleBucket は idle 時のバケット値。sparse GSI idle-index のキー。
	IdleBucket string `dynamodbav:"idleBucket,omitempty"`
}

// SparseAttributes は状態に応じた sparse 属性値を返す。
// idle 時は idleBucket を設定し currentSessionID をクリアする。
// busy 時は currentSessionID を設定し idleBucket をクリアする。
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

// IsValidStatus は文字列が有効な RunnerStatus かを返す。
func IsValidStatus(s string) bool {
	return allStatuses[RunnerStatus(s)]
}
