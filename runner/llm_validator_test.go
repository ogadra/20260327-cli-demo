package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	brtypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

// mockBedrockClient is a test double for BedrockConverseClient.
// When outputs has multiple entries, each Converse call returns the next one in
// sequence. When outputs is nil, the single output and err fields are used.
type mockBedrockClient struct {
	output  *bedrockruntime.ConverseOutput
	err     error
	outputs []*bedrockruntime.ConverseOutput
	errs    []error
	calls   int
}

// Converse returns the preconfigured output and error, supporting sequential
// responses when outputs is set.
func (m *mockBedrockClient) Converse(_ context.Context, _ *bedrockruntime.ConverseInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
	if m.outputs != nil {
		i := m.calls
		m.calls++
		if i >= len(m.outputs) || i >= len(m.errs) {
			return nil, fmt.Errorf("mockBedrockClient: sequence out of range (i=%d, outputs=%d, errs=%d)", i, len(m.outputs), len(m.errs))
		}
		return m.outputs[i], m.errs[i]
	}
	m.calls++
	return m.output, m.err
}

// makeToolUseOutput builds a ConverseOutput containing a tool use block with the given JSON map.
func makeToolUseOutput(data map[string]interface{}) *bedrockruntime.ConverseOutput {
	return &bedrockruntime.ConverseOutput{
		Output: &brtypes.ConverseOutputMemberMessage{
			Value: brtypes.Message{
				Role: brtypes.ConversationRoleAssistant,
				Content: []brtypes.ContentBlock{
					&brtypes.ContentBlockMemberToolUse{
						Value: brtypes.ToolUseBlock{
							ToolUseId: strPtr("tool-1"),
							Name:      strPtr(toolName),
							Input:     document.NewLazyDocument(data),
						},
					},
				},
			},
		},
	}
}

// noToolUseOutput builds a ConverseOutput containing only a text block.
func noToolUseOutput() *bedrockruntime.ConverseOutput {
	return &bedrockruntime.ConverseOutput{
		Output: &brtypes.ConverseOutputMemberMessage{
			Value: brtypes.Message{
				Role: brtypes.ConversationRoleAssistant,
				Content: []brtypes.ContentBlock{
					&brtypes.ContentBlockMemberText{Value: "I think it is safe"},
				},
			},
		},
	}
}

// TestValidateSafe verifies that a safe judgment from the LLM produces
// ValidationResult with Safe=true and the correct reason.
func TestValidateSafe(t *testing.T) {
	client := &mockBedrockClient{
		output: makeToolUseOutput(map[string]interface{}{
			"safe":   true,
			"reason": "read-only listing",
		}),
	}
	v := NewBedrockValidator(client, "test-model")

	result, err := v.Validate(context.Background(), "ls -la")
	if err != nil {
		t.Fatalf("Validate error: %v", err)
	}
	if !result.Safe {
		t.Fatalf("Safe = false, want true")
	}
	if result.Reason != "read-only listing" {
		t.Fatalf("Reason = %q, want %q", result.Reason, "read-only listing")
	}
}

// TestValidateUnsafe verifies that an unsafe judgment from the LLM produces
// ValidationResult with Safe=false and the correct reason.
func TestValidateUnsafe(t *testing.T) {
	client := &mockBedrockClient{
		output: makeToolUseOutput(map[string]interface{}{
			"safe":   false,
			"reason": "destructive delete operation",
		}),
	}
	v := NewBedrockValidator(client, "test-model")

	result, err := v.Validate(context.Background(), "rm -rf /")
	if err != nil {
		t.Fatalf("Validate error: %v", err)
	}
	if result.Safe {
		t.Fatalf("Safe = true, want false")
	}
	if result.Reason != "destructive delete operation" {
		t.Fatalf("Reason = %q, want %q", result.Reason, "destructive delete operation")
	}
}

// TestValidateAPIError verifies that an API error from the Bedrock client
// is propagated immediately without retry.
func TestValidateAPIError(t *testing.T) {
	client := &mockBedrockClient{
		err: errors.New("service unavailable"),
	}
	v := NewBedrockValidator(client, "test-model")

	_, err := v.Validate(context.Background(), "ls")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, client.err) {
		t.Fatalf("error = %v, want wrapping %v", err, client.err)
	}
	if client.calls != 1 {
		t.Fatalf("calls = %d, want 1 since API errors are not retried", client.calls)
	}
}

// TestValidateNoToolUseBlockRetries verifies that a response without any tool
// use block triggers retries and eventually returns an error.
func TestValidateNoToolUseBlockRetries(t *testing.T) {
	client := &mockBedrockClient{
		output: noToolUseOutput(),
	}
	v := NewBedrockValidator(client, "test-model")

	_, err := v.Validate(context.Background(), "ls")
	if err == nil {
		t.Fatal("expected error for missing tool use block, got nil")
	}
	if !strings.Contains(err.Error(), "retries exhausted") {
		t.Fatalf("error = %v, want containing 'retries exhausted'", err)
	}
	if client.calls != maxRetries+1 {
		t.Fatalf("calls = %d, want %d", client.calls, maxRetries+1)
	}
}

// TestValidateInvalidJSONRetries verifies that an unparseable tool input
// triggers retries and eventually returns an error.
func TestValidateInvalidJSONRetries(t *testing.T) {
	badOutput := &bedrockruntime.ConverseOutput{
		Output: &brtypes.ConverseOutputMemberMessage{
			Value: brtypes.Message{
				Role: brtypes.ConversationRoleAssistant,
				Content: []brtypes.ContentBlock{
					&brtypes.ContentBlockMemberToolUse{
						Value: brtypes.ToolUseBlock{
							ToolUseId: strPtr("tool-1"),
							Name:      strPtr(toolName),
							Input:     document.NewLazyDocument("not-a-json-object"),
						},
					},
				},
			},
		},
	}
	client := &mockBedrockClient{output: badOutput}
	v := NewBedrockValidator(client, "test-model")

	_, err := v.Validate(context.Background(), "ls")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if client.calls != maxRetries+1 {
		t.Fatalf("calls = %d, want %d", client.calls, maxRetries+1)
	}
}

// TestValidateUnexpectedOutputType verifies that an unexpected output type
// from Converse triggers retries and eventually returns an error.
func TestValidateUnexpectedOutputType(t *testing.T) {
	client := &mockBedrockClient{
		output: &bedrockruntime.ConverseOutput{
			Output: nil,
		},
	}
	v := NewBedrockValidator(client, "test-model")

	_, err := v.Validate(context.Background(), "ls")
	if err == nil {
		t.Fatal("expected error for unexpected output type, got nil")
	}
	if client.calls != maxRetries+1 {
		t.Fatalf("calls = %d, want %d", client.calls, maxRetries+1)
	}
}

// TestValidateMarshalError verifies that a document that fails to marshal
// triggers retries and eventually returns an error.
func TestValidateMarshalError(t *testing.T) {
	unmarshalable := func() {}
	badOutput := &bedrockruntime.ConverseOutput{
		Output: &brtypes.ConverseOutputMemberMessage{
			Value: brtypes.Message{
				Role: brtypes.ConversationRoleAssistant,
				Content: []brtypes.ContentBlock{
					&brtypes.ContentBlockMemberToolUse{
						Value: brtypes.ToolUseBlock{
							ToolUseId: strPtr("tool-1"),
							Name:      strPtr(toolName),
							Input:     document.NewLazyDocument(unmarshalable),
						},
					},
				},
			},
		},
	}
	client := &mockBedrockClient{output: badOutput}
	v := NewBedrockValidator(client, "test-model")

	_, err := v.Validate(context.Background(), "ls")
	if err == nil {
		t.Fatal("expected error for marshal failure, got nil")
	}
	if client.calls != maxRetries+1 {
		t.Fatalf("calls = %d, want %d", client.calls, maxRetries+1)
	}
}

// TestValidateRetrySucceeds verifies that a parse failure on the first
// attempt followed by a valid response on the second attempt succeeds.
func TestValidateRetrySucceeds(t *testing.T) {
	goodOutput := makeToolUseOutput(map[string]interface{}{
		"safe":   true,
		"reason": "ok",
	})
	client := &mockBedrockClient{
		outputs: []*bedrockruntime.ConverseOutput{noToolUseOutput(), goodOutput},
		errs:    []error{nil, nil},
	}
	v := NewBedrockValidator(client, "test-model")

	result, err := v.Validate(context.Background(), "ls")
	if err != nil {
		t.Fatalf("Validate error: %v", err)
	}
	if !result.Safe {
		t.Fatalf("Safe = false, want true")
	}
	if client.calls != 2 {
		t.Fatalf("calls = %d, want 2", client.calls)
	}
}

// TestValidateRetryAPIError verifies that when the first attempt fails to
// parse but the second attempt returns an API error, the API error is returned
// immediately.
func TestValidateRetryAPIError(t *testing.T) {
	client := &mockBedrockClient{
		outputs: []*bedrockruntime.ConverseOutput{
			noToolUseOutput(),
			nil,
		},
		errs: []error{nil, errors.New("api down")},
	}
	v := NewBedrockValidator(client, "test-model")

	_, err := v.Validate(context.Background(), "ls")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "api down") {
		t.Fatalf("error = %v, want containing 'api down'", err)
	}
	if client.calls != 2 {
		t.Fatalf("calls = %d, want 2", client.calls)
	}
}

// TestValidateNilOutputRetries verifies that a nil ConverseOutput triggers
// retries and returns an error instead of panicking.
func TestValidateNilOutputRetries(t *testing.T) {
	client := &mockBedrockClient{
		output: nil,
	}
	v := NewBedrockValidator(client, "test-model")

	_, err := v.Validate(context.Background(), "ls")
	if err == nil {
		t.Fatal("expected error for nil output, got nil")
	}
	if client.calls != maxRetries+1 {
		t.Fatalf("calls = %d, want %d", client.calls, maxRetries+1)
	}
}

// TestValidateWrongToolNameIgnored verifies that a tool use block with the
// wrong name is ignored and treated as no matching tool use.
func TestValidateWrongToolNameIgnored(t *testing.T) {
	wrongNameOutput := &bedrockruntime.ConverseOutput{
		Output: &brtypes.ConverseOutputMemberMessage{
			Value: brtypes.Message{
				Role: brtypes.ConversationRoleAssistant,
				Content: []brtypes.ContentBlock{
					&brtypes.ContentBlockMemberToolUse{
						Value: brtypes.ToolUseBlock{
							ToolUseId: strPtr("tool-1"),
							Name:      strPtr("wrong_tool"),
							Input: document.NewLazyDocument(map[string]interface{}{
								"safe":   true,
								"reason": "should be ignored",
							}),
						},
					},
				},
			},
		},
	}
	client := &mockBedrockClient{output: wrongNameOutput}
	v := NewBedrockValidator(client, "test-model")

	_, err := v.Validate(context.Background(), "ls")
	if err == nil {
		t.Fatal("expected error for wrong tool name, got nil")
	}
	if !strings.Contains(err.Error(), "retries exhausted") {
		t.Fatalf("error = %v, want containing 'retries exhausted'", err)
	}
}

// TestValidateMaxRetriesMeansInitialPlusRetries verifies that maxRetries=2
// results in 1 initial attempt + 2 retries = 3 total calls.
func TestValidateMaxRetriesMeansInitialPlusRetries(t *testing.T) {
	client := &mockBedrockClient{
		output: noToolUseOutput(),
	}
	v := NewBedrockValidator(client, "test-model")

	_, _ = v.Validate(context.Background(), "ls")

	// maxRetries=2 should mean initial + 2 retries = 3 total calls.
	if client.calls != maxRetries+1 {
		t.Fatalf("calls = %d, want %d (initial + %d retries)", client.calls, maxRetries+1, maxRetries)
	}
}
