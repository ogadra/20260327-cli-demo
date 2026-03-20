// Package model はドメインモデルを提供する。
package model

// Runner は broker が管理する runner のドメインモデル。
// runner は使い捨てであり、セッション終了時または異常終了時はレコードごと削除する。
// 状態は sparse 属性で暗黙的に表現する。
// IdleBucket が存在すれば idle、CurrentSessionID が存在すれば busy。
type Runner struct {
	// RunnerID は runner の一意識別子であり DynamoDB の PK。
	RunnerID string `dynamodbav:"runnerId"`
	// CurrentSessionID は busy 時のセッション ID。sparse GSI session-index のキー。
	CurrentSessionID string `dynamodbav:"currentSessionId,omitempty"`
	// IdleBucket は idle 時のバケット値。sparse GSI idle-index のキー。
	IdleBucket string `dynamodbav:"idleBucket,omitempty"`
}

// IsIdle は runner が idle 状態かを返す。
func (r *Runner) IsIdle() bool {
	return r.IdleBucket != ""
}

// IsBusy は runner が busy 状態かを返す。
func (r *Runner) IsBusy() bool {
	return r.CurrentSessionID != ""
}
