package main

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	brtypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

// mockBedrockClient is a test double for BedrockConverseClient that returns
// preconfigured output or error from Converse.
type mockBedrockClient struct {
	output *bedrockruntime.ConverseOutput
	err    error
}

// Converse returns the preconfigured output and error.
func (m *mockBedrockClient) Converse(_ context.Context, _ *bedrockruntime.ConverseInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
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
// is propagated as a Validate error.
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
}

// TestValidateNoToolUseBlock verifies that a response without any tool use block
// returns an error.
func TestValidateNoToolUseBlock(t *testing.T) {
	client := &mockBedrockClient{
		output: &bedrockruntime.ConverseOutput{
			Output: &brtypes.ConverseOutputMemberMessage{
				Value: brtypes.Message{
					Role: brtypes.ConversationRoleAssistant,
					Content: []brtypes.ContentBlock{
						&brtypes.ContentBlockMemberText{Value: "I think it is safe"},
					},
				},
			},
		},
	}
	v := NewBedrockValidator(client, "test-model")

	_, err := v.Validate(context.Background(), "ls")
	if err == nil {
		t.Fatal("expected error for missing tool use block, got nil")
	}
}

// TestValidateInvalidJSON verifies that an unparseable tool input
// returns an error.
func TestValidateInvalidJSON(t *testing.T) {
	client := &mockBedrockClient{
		output: &bedrockruntime.ConverseOutput{
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
		},
	}
	v := NewBedrockValidator(client, "test-model")

	_, err := v.Validate(context.Background(), "ls")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// TestValidateUnexpectedOutputType verifies that an unexpected output type
// from Converse returns an error.
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
}

// TestValidateMarshalError verifies that a document that fails to marshal
// returns an error from Validate.
func TestValidateMarshalError(t *testing.T) {
	// A function type cannot be marshaled to JSON, causing MarshalSmithyDocument to fail.
	unmarshalable := func() {}
	client := &mockBedrockClient{
		output: &bedrockruntime.ConverseOutput{
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
		},
	}
	v := NewBedrockValidator(client, "test-model")

	_, err := v.Validate(context.Background(), "ls")
	if err == nil {
		t.Fatal("expected error for marshal failure, got nil")
	}
}
