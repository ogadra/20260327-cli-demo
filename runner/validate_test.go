package main

import "testing"

// TestClassifyWhitelisted verifies that bare whitelisted commands with no arguments
// are classified as "whitelisted".
func TestClassifyWhitelisted(t *testing.T) {
	cases := []struct {
		cmd  string
		name string
	}{
		{"ls", "bare ls"},
		{"pwd", "bare pwd"},
		{"date", "bare date"},
		{"whoami", "bare whoami"},
		{"env", "bare env"},
		{"tree", "bare tree"},
		{"uname", "bare uname"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyCommand(tc.cmd)
			if got != "whitelisted" {
				t.Errorf("classifyCommand(%q) = %q, want %q", tc.cmd, got, "whitelisted")
			}
		})
	}
}

// TestClassifyWhitelistedWithSurroundingSpaces verifies that leading and trailing
// whitespace is ignored when matching whitelisted commands.
func TestClassifyWhitelistedWithSurroundingSpaces(t *testing.T) {
	cases := []struct {
		cmd  string
		name string
	}{
		{"  ls  ", "ls with spaces"},
		{"\tpwd\n", "pwd with tabs and newlines"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyCommand(tc.cmd)
			if got != "whitelisted" {
				t.Errorf("classifyCommand(%q) = %q, want %q", tc.cmd, got, "whitelisted")
			}
		})
	}
}

// TestClassifyWhitelistedWithArgs verifies that whitelisted commands with arguments
// are classified as "validated" because arguments can be abused.
func TestClassifyWhitelistedWithArgs(t *testing.T) {
	cases := []struct {
		cmd  string
		name string
	}{
		{"ls -la /tmp", "ls with flags"},
		{"uname -a", "uname with flag"},
		{"tree .", "tree with path"},
		{"env FOO=bar", "env with assignment"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyCommand(tc.cmd)
			if got != "validated" {
				t.Errorf("classifyCommand(%q) = %q, want %q", tc.cmd, got, "validated")
			}
		})
	}
}

// TestClassifyValidated verifies that commands not in the whitelist
// are classified as "validated".
func TestClassifyValidated(t *testing.T) {
	cases := []struct {
		cmd  string
		name string
	}{
		{"rm -rf /", "rm command"},
		{"apt-get install vim", "apt-get command"},
		{"curl https://example.com", "curl command"},
		{"python3 script.py", "python3 command"},
		{"go build ./...", "go command"},
		{"docker run hello", "docker command"},
		{"make all", "make command"},
		{"echo hello", "echo with args"},
		{"cat /etc/passwd", "cat with path"},
		{"cd /tmp", "cd with path"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyCommand(tc.cmd)
			if got != "validated" {
				t.Errorf("classifyCommand(%q) = %q, want %q", tc.cmd, got, "validated")
			}
		})
	}
}

// TestClassifyNixRunWhitelisted verifies that nix run nixpkgs# commands
// are classified as "whitelisted" when they contain no shell metacharacters.
func TestClassifyNixRunWhitelisted(t *testing.T) {
	cases := []struct {
		cmd  string
		name string
	}{
		{"nix run nixpkgs#hello", "bare nix run nixpkgs"},
		{"nix run nixpkgs#jq -- --help", "nix run with trailing args"},
		{"  nix run nixpkgs#hello  ", "nix run with surrounding spaces"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyCommand(tc.cmd)
			if got != "whitelisted" {
				t.Errorf("classifyCommand(%q) = %q, want %q", tc.cmd, got, "whitelisted")
			}
		})
	}
}

// TestClassifyNixRunRejected verifies that nix run commands targeting non-nixpkgs
// flake refs or containing shell metacharacters are classified as "validated".
func TestClassifyNixRunRejected(t *testing.T) {
	cases := []struct {
		cmd  string
		name string
	}{
		{"nix run github:user/repo#pkg", "non-nixpkgs flake ref"},
		{"nix run nixpkgs#hello; rm -rf /", "semicolon chaining"},
		{"nix run nixpkgs#hello && echo pwned", "ampersand chaining"},
		{"nix run nixpkgs#hello | cat", "pipe operator"},
		{"nix run nixpkgs#$(echo evil)", "command substitution dollar"},
		{"nix run nixpkgs#`echo evil`", "command substitution backtick"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyCommand(tc.cmd)
			if got != "validated" {
				t.Errorf("classifyCommand(%q) = %q, want %q", tc.cmd, got, "validated")
			}
		})
	}
}

// TestClassifyChainedCommands verifies that chained commands using shell operators
// are classified as "validated" because the full string does not match a bare command.
func TestClassifyChainedCommands(t *testing.T) {
	cases := []struct {
		cmd  string
		name string
	}{
		{"ls && rm -rf /", "ls chained with rm"},
		{"pwd; cat /etc/shadow", "pwd chained with cat"},
		{"ls | xargs rm", "ls piped to rm"},
		{"date || echo pwned", "date chained with echo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyCommand(tc.cmd)
			if got != "validated" {
				t.Errorf("classifyCommand(%q) = %q, want %q", tc.cmd, got, "validated")
			}
		})
	}
}

// TestClassifyEmptyCommand verifies that an empty command string
// is classified as "validated".
func TestClassifyEmptyCommand(t *testing.T) {
	got := classifyCommand("")
	if got != "validated" {
		t.Errorf("classifyCommand(%q) = %q, want %q", "", got, "validated")
	}
}

// TestClassifyWhitespaceOnly verifies that a whitespace-only command string
// is classified as "validated".
func TestClassifyWhitespaceOnly(t *testing.T) {
	got := classifyCommand("   ")
	if got != "validated" {
		t.Errorf("classifyCommand(%q) = %q, want %q", "   ", got, "validated")
	}
}
