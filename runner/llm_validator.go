package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
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
