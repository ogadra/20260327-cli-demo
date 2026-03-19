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
		{StatusIdle, StatusReserved, true},
		{StatusIdle, StatusBusy, false},
		{StatusIdle, StatusDraining, false},
		{StatusIdle, StatusDead, true},
		// reserved -> *
		{StatusReserved, StatusIdle, false},
		{StatusReserved, StatusReserved, false},
		{StatusReserved, StatusBusy, true},
		{StatusReserved, StatusDraining, false},
		{StatusReserved, StatusDead, true},
		// busy -> *
		{StatusBusy, StatusIdle, true},
		{StatusBusy, StatusReserved, false},
		{StatusBusy, StatusBusy, false},
		{StatusBusy, StatusDraining, true},
		{StatusBusy, StatusDead, true},
		// draining -> *
		{StatusDraining, StatusIdle, false},
		{StatusDraining, StatusReserved, false},
		{StatusDraining, StatusBusy, false},
		{StatusDraining, StatusDraining, false},
		{StatusDraining, StatusDead, true},
		// dead -> *
		{StatusDead, StatusIdle, false},
		{StatusDead, StatusReserved, false},
		{StatusDead, StatusBusy, false},
		{StatusDead, StatusDraining, false},
		{StatusDead, StatusDead, false},
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
	if err := ValidateTransition(StatusIdle, StatusReserved); err != nil {
		t.Errorf("ValidateTransition(idle, reserved) returned unexpected error: %v", err)
	}

	err := ValidateTransition(StatusIdle, StatusBusy)
	if err == nil {
		t.Fatal("ValidateTransition(idle, busy) should return error")
	}

	invalidErr, ok := err.(*ErrInvalidTransition)
	if !ok {
		t.Fatalf("expected *ErrInvalidTransition, got %T", err)
	}
	if invalidErr.From != StatusIdle || invalidErr.To != StatusBusy {
		t.Errorf("ErrInvalidTransition = {%s, %s}, want {idle, busy}", invalidErr.From, invalidErr.To)
	}

	expected := "invalid transition from idle to busy"
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
			name:           "reserved clears both",
			status:         StatusReserved,
			sessionID:      "sess-1",
			bucket:         "bucket-0",
			wantSessionID:  "",
			wantIdleBucket: "",
		},
		{
			name:           "draining clears both",
			status:         StatusDraining,
			sessionID:      "sess-1",
			bucket:         "bucket-0",
			wantSessionID:  "",
			wantIdleBucket: "",
		},
		{
			name:           "dead clears both",
			status:         StatusDead,
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
	validStatuses := []string{"idle", "reserved", "busy", "draining", "dead"}
	for _, s := range validStatuses {
		if !IsValidStatus(s) {
			t.Errorf("IsValidStatus(%q) = false, want true", s)
		}
	}

	invalidStatuses := []string{"", "unknown", "IDLE", "running", "stopped"}
	for _, s := range invalidStatuses {
		if IsValidStatus(s) {
			t.Errorf("IsValidStatus(%q) = true, want false", s)
		}
	}
}

// TestIsTerminal は終端状態と非終端状態を検証する。
func TestIsTerminal(t *testing.T) {
	if !IsTerminal(StatusDead) {
		t.Error("IsTerminal(dead) = false, want true")
	}

	nonTerminal := []RunnerStatus{StatusIdle, StatusReserved, StatusBusy, StatusDraining}
	for _, s := range nonTerminal {
		if IsTerminal(s) {
			t.Errorf("IsTerminal(%s) = true, want false", s)
		}
	}
}
