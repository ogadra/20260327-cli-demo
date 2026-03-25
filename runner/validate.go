package main

import (
	"regexp"
	"strings"
)

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

// shellMetaChars matches shell operators that could be used to chain commands.
var shellMetaChars = regexp.MustCompile(`[;|&<>\n\r` + "`" + `]|\$\(`)

// classifyCommand returns the classification of a command for audit logging.
// It returns "whitelisted" if the trimmed command exactly matches a bare
// whitelisted command, or if it is a "nix run nixpkgs#..." invocation
// without shell metacharacters. Otherwise it returns "validated".
func classifyCommand(cmd string) string {
	trimmed := strings.TrimSpace(cmd)
	if whitelistedCommands[trimmed] {
		return "whitelisted"
	}
	if strings.HasPrefix(trimmed, "nix run nixpkgs#") && !shellMetaChars.MatchString(trimmed) {
		return "whitelisted"
	}
	return "validated"
}
