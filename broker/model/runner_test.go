// Package model はドメインモデルと状態遷移ロジックのテストを提供する。
package model

import "testing"

// TestCanTransitionTo は全状態ペアの遷移可否を検証する。
func TestCanTransitionTo(t *testing.T) {
	tests := []struct {
		from   RunnerStatus
		to     RunnerStatus
		expect bool
	}{
		// idle -> *
		{StatusIdle, StatusIdle, false},
		{StatusIdle, StatusBusy, true},
		// busy -> *
		{StatusBusy, StatusIdle, true},
		{StatusBusy, StatusBusy, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			got := CanTransitionTo(tt.from, tt.to)
			if got != tt.expect {
				t.Errorf("CanTransitionTo(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.expect)
			}
		})
	}
}

// TestCanTransitionToUnknownStatus は未知の状態からの遷移が拒否されることを検証する。
func TestCanTransitionToUnknownStatus(t *testing.T) {
	unknown := RunnerStatus("unknown")
	if CanTransitionTo(unknown, StatusIdle) {
		t.Error("CanTransitionTo from unknown status should return false")
	}
}

// TestValidateTransition は有効な遷移でエラーなし、無効な遷移でエラーありを検証する。
func TestValidateTransition(t *testing.T) {
	if err := ValidateTransition(StatusIdle, StatusBusy); err != nil {
		t.Errorf("ValidateTransition(idle, busy) returned unexpected error: %v", err)
	}

	err := ValidateTransition(StatusIdle, StatusIdle)
	if err == nil {
		t.Fatal("ValidateTransition(idle, idle) should return error")
	}

	invalidErr, ok := err.(*ErrInvalidTransition)
	if !ok {
		t.Fatalf("expected *ErrInvalidTransition, got %T", err)
	}
	if invalidErr.From != StatusIdle || invalidErr.To != StatusIdle {
		t.Errorf("ErrInvalidTransition = {%s, %s}, want {idle, idle}", invalidErr.From, invalidErr.To)
	}

	expected := "invalid transition from idle to idle"
	if invalidErr.Error() != expected {
		t.Errorf("Error() = %q, want %q", invalidErr.Error(), expected)
	}
}

// TestSparseAttributes は状態ごとの sparse 属性値を検証する。
func TestSparseAttributes(t *testing.T) {
	tests := []struct {
		name           string
		status         RunnerStatus
		sessionID      string
		bucket         string
		wantSessionID  string
		wantIdleBucket string
	}{
		{
			name:           "idle sets bucket only",
			status:         StatusIdle,
			sessionID:      "sess-1",
			bucket:         "bucket-0",
			wantSessionID:  "",
			wantIdleBucket: "bucket-0",
		},
		{
			name:           "busy sets sessionID only",
			status:         StatusBusy,
			sessionID:      "sess-1",
			bucket:         "bucket-0",
			wantSessionID:  "sess-1",
			wantIdleBucket: "",
		},
		{
			name:           "unknown status clears both",
			status:         RunnerStatus("unknown"),
			sessionID:      "sess-1",
			bucket:         "bucket-0",
			wantSessionID:  "",
			wantIdleBucket: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSessionID, gotBucket := SparseAttributes(tt.status, tt.sessionID, tt.bucket)
			if gotSessionID != tt.wantSessionID {
				t.Errorf("currentSessionID = %q, want %q", gotSessionID, tt.wantSessionID)
			}
			if gotBucket != tt.wantIdleBucket {
				t.Errorf("idleBucket = %q, want %q", gotBucket, tt.wantIdleBucket)
			}
		})
	}
}

// TestIsValidStatus は有効な状態文字列と無効な状態文字列を検証する。
func TestIsValidStatus(t *testing.T) {
	validStatuses := []string{"idle", "busy"}
	for _, s := range validStatuses {
		if !IsValidStatus(s) {
			t.Errorf("IsValidStatus(%q) = false, want true", s)
		}
	}

	invalidStatuses := []string{"", "unknown", "IDLE", "dead", "reserved", "draining", "stopped"}
	for _, s := range invalidStatuses {
		if IsValidStatus(s) {
			t.Errorf("IsValidStatus(%q) = true, want false", s)
		}
	}
}
