package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	brtypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

// ValidationResult holds the outcome of an LLM safety check on a command.
type ValidationResult struct {
	Safe   bool
	Reason string
}

// Validator judges whether a shell command is safe to execute.
type Validator interface {
	Validate(ctx context.Context, command string) (ValidationResult, error)
}

// BedrockConverseClient abstracts the Bedrock Runtime Converse API for dependency injection.
type BedrockConverseClient interface {
	Converse(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error)
}

// systemPrompt instructs the LLM to judge command safety.
const systemPrompt = `You are a security validator for a web-based shell environment.
Your job is to judge whether a given shell command is safe to execute.

Safe commands:
- Read-only operations like ls, cat, head, tail, grep, find, du, df, ps, top, free
- Commands with safe flags like ls -la, uname -a
- Navigation commands like cd, pwd
- Echo and printf for display

Unsafe commands:
- Anything that modifies or deletes files: rm, mv, chmod, chown, truncate
- Package management: apt, yum, pip, npm install
- Network exfiltration: curl, wget, nc, ssh with write/upload
- Process manipulation: kill, reboot, shutdown
- Disk operations: dd, mkfs, mount
- Shell escapes and chaining that hide dangerous operations
- Writing to sensitive paths like /etc, /root, /var
- Commands using backticks, $(), or pipe to shell for code injection

Use the command_safety_judgment tool to report your decision.`

// toolName is the name of the tool use function for command safety judgment.
const toolName = "command_safety_judgment"

// toolSchema defines the JSON schema for the command_safety_judgment tool.
var toolSchema = document.NewLazyDocument(map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"safe": map[string]interface{}{
			"type":        "boolean",
			"description": "true if the command is safe to execute, false otherwise",
		},
		"reason": map[string]interface{}{
			"type":        "string",
			"description": "brief explanation of why the command is safe or unsafe",
		},
	},
	"required": []string{"safe", "reason"},
})

// BedrockValidator validates commands using Bedrock Converse API with tool use.
type BedrockValidator struct {
	client  BedrockConverseClient
	modelID string
}

// NewBedrockValidator creates a BedrockValidator with the given client and model ID.
func NewBedrockValidator(client BedrockConverseClient, modelID string) *BedrockValidator {
	return &BedrockValidator{client: client, modelID: modelID}
}

// Validate calls the Bedrock Converse API to judge whether command is safe to execute.
// It returns an error if the API call fails or the response cannot be parsed.
func (v *BedrockValidator) Validate(ctx context.Context, command string) (ValidationResult, error) {
	input := &bedrockruntime.ConverseInput{
		ModelId: &v.modelID,
		System: []brtypes.SystemContentBlock{
			&brtypes.SystemContentBlockMemberText{Value: systemPrompt},
		},
		Messages: []brtypes.Message{
			{
				Role: brtypes.ConversationRoleUser,
				Content: []brtypes.ContentBlock{
					&brtypes.ContentBlockMemberText{Value: fmt.Sprintf("Judge this command: %s", command)},
				},
			},
		},
		ToolConfig: &brtypes.ToolConfiguration{
			Tools: []brtypes.Tool{
				&brtypes.ToolMemberToolSpec{
					Value: brtypes.ToolSpecification{
						Name:        strPtr(toolName),
						Description: strPtr("Report whether a shell command is safe or unsafe to execute"),
						InputSchema: &brtypes.ToolInputSchemaMemberJson{Value: toolSchema},
					},
				},
			},
			ToolChoice: &brtypes.ToolChoiceMemberAny{Value: brtypes.AnyToolChoice{}},
		},
	}

	output, err := v.client.Converse(ctx, input)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("bedrock converse: %w", err)
	}

	return parseToolUseResult(output)
}

// parseToolUseResult extracts the tool use result from the Converse API output.
func parseToolUseResult(output *bedrockruntime.ConverseOutput) (ValidationResult, error) {
	msg, ok := output.Output.(*brtypes.ConverseOutputMemberMessage)
	if !ok {
		return ValidationResult{}, errors.New("unexpected converse output type")
	}

	for _, block := range msg.Value.Content {
		tu, ok := block.(*brtypes.ContentBlockMemberToolUse)
		if !ok {
			continue
		}

		raw, err := tu.Value.Input.MarshalSmithyDocument()
		if err != nil {
			return ValidationResult{}, fmt.Errorf("marshal tool input: %w", err)
		}

		var result struct {
			Safe   bool   `json:"safe"`
			Reason string `json:"reason"`
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			return ValidationResult{}, fmt.Errorf("parse tool result: %w", err)
		}

		return ValidationResult{Safe: result.Safe, Reason: result.Reason}, nil
	}

	return ValidationResult{}, errors.New("no tool use block in response")
}

// strPtr returns a pointer to the given string.
func strPtr(s string) *string {
	return &s
}
