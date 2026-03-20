// Package model はドメインモデルのテストを提供する。
package model

import (
	"encoding/json"
	"testing"
)

// TestErrorResponse_JSON は ErrorResponse の JSON マーシャルを検証する。
func TestErrorResponse_JSON(t *testing.T) {
	t.Parallel()
	resp := ErrorResponse{
		Code:      CodeNoIdleRunner,
		Message:   "no idle runner available",
		RequestID: "req-123",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got ErrorResponse
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if got.Code != CodeNoIdleRunner {
		t.Errorf("Code = %q, want %q", got.Code, CodeNoIdleRunner)
	}
	if got.Message != "no idle runner available" {
		t.Errorf("Message = %q, want %q", got.Message, "no idle runner available")
	}
	if got.RequestID != "req-123" {
		t.Errorf("RequestID = %q, want %q", got.RequestID, "req-123")
	}
}

// TestErrorResponse_JSONKeys は JSON キー名が仕様通りであることを検証する。
func TestErrorResponse_JSONKeys(t *testing.T) {
	t.Parallel()
	resp := ErrorResponse{
		Code:      CodeInternalError,
		Message:   "something went wrong",
		RequestID: "req-456",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	for _, key := range []string{"code", "message", "requestId"} {
		if _, ok := m[key]; !ok {
			t.Errorf("missing JSON key %q", key)
		}
	}
}

// TestErrorCodeConstants はエラーコード定数が空でないことを検証する。
func TestErrorCodeConstants(t *testing.T) {
	t.Parallel()
	codes := []string{
		CodeNoIdleRunner,
		CodeSessionNotFound,
		CodeInvalidRequest,
		CodeInternalError,
	}
	for i, code := range codes {
		if code == "" {
			t.Errorf("codes[%d] is empty", i)
		}
	}
}
