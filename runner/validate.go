package main

import "strings"

// whitelistedCommands is the set of commands that are allowed without LLM validation.
// Only bare commands with no arguments are whitelisted, because arguments can be
// abused to execute arbitrary code or exfiltrate data.
var whitelistedCommands = map[string]bool{
	"ls":     true,
	"pwd":    true,
	"date":   true,
	"whoami": true,
	"env":    true,
	"tree":   true,
	"uname":  true,
}

// classifyCommand returns the classification of a command for audit logging.
// It returns "whitelisted" only if the entire command after trimming whitespace
// exactly matches a whitelisted command name with no arguments.
// Otherwise it returns "validated".
func classifyCommand(cmd string) string {
	if whitelistedCommands[strings.TrimSpace(cmd)] {
		return "whitelisted"
	}
	return "validated"
}
