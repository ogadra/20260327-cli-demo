// Package model はドメインモデルのテストを提供する。
package model

import "testing"

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
