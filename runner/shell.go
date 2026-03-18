// Package main implements a sandbox command execution server.
//
// shell.go: Shell manages a single persistent bash session for command execution.
//
// # Architecture
//
//	┌─────────────────────────────────────────────────────────┐
//	│                        Shell                            │
//	│─────────────────────────────────────────────────────────│
//	│  cmd        commander          ← interface              │
//	│  stdin      io.WriteCloser     ← interface              │
//	│  stdout     *bufio.Scanner     ← 具体型                 │
//	│  stderrBuf  bytes.Buffer       ← 具体型                 │
//	│  stderrMu   sync.Mutex         ← 具体型                 │
//	│  mu         sync.Mutex         ← 具体型                 │
//	├─────────────────────────────────────────────────────────┤
//	│  ExecuteStream()  Close()                               │
//	└────────┬──────────────┬─────────────────────────────────┘
//	         │              │
//	         ▼              ▼
//	┌─────────────┐  ┌──────────────┐  ┌─────────────────────┐
//	│  commander  │  │io.WriteCloser│  │  *bufio.Scanner     │
//	│  (interface)│  │ (interface)  │  │  (具体型)            │
//	├─────────────┤  └──────┬───────┘  └──────────┬──────────┘
//	│ Start()     │         │                     │
//	│ Wait()      │    stdin に直接          stdout に直接
//	│ StdinPipe() │    Write する            Scan する
//	│ StdoutPipe()│
//	│ StderrPipe()│
//	└──────┬──────┘
//	       │
//	       │ 実装
//	       ▼
//	┌──────────────────┐     ┌──────────────────┐
//	│  execCommander   │     │  fakeCommander   │
//	│  (本番)          │     │  (テスト用)       │
//	├──────────────────┤     ├──────────────────┤
//	│  cmd *exec.Cmd   │     │  各種エラー注入   │
//	└──────────────────┘     └──────────────────┘
//
// commander は初期化時 (newShellFromCommander) にパイプ取得とプロセス起動に使われる。
// 実行時は stdin/stdout を直接操作し、commander の Wait() は Close() でのみ呼ばれる。
//
// # Marker Protocol
//
// 各コマンドはユニークなマーカーで囲まれ、出力の境界を検出する:
//
//	<command>
//	__ec=$?
//	echo '__MRK_<nanoseconds>_END__'${__ec}
//
// stdout を行単位でスキャンし、マーカー行が現れたら接尾辞から exit code をパースする。
// stderr は goroutine で非同期に蓄積され、マーカー検出後に短い遅延を挟んで取得する。
package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// commander abstracts process lifecycle for testing.
// In production, [execCommander] wraps [exec.Cmd].
// In tests, fakeCommander injects errors into pipe creation and process lifecycle.
type commander interface {
	// Start starts the process.
	Start() error
	// Wait waits for the process to exit and returns its exit status.
	Wait() error
	// StdinPipe returns a pipe connected to the process's standard input.
	StdinPipe() (io.WriteCloser, error)
	// StdoutPipe returns a pipe connected to the process's standard output.
	StdoutPipe() (io.ReadCloser, error)
	// StderrPipe returns a pipe connected to the process's standard error.
	StderrPipe() (io.ReadCloser, error)
}

// execCommander wraps [exec.Cmd] to implement [commander].
type execCommander struct {
	cmd *exec.Cmd // underlying OS process
}

// Start starts the bash process.
func (c *execCommander) Start() error { return c.cmd.Start() }

// Wait waits for the bash process to exit.
func (c *execCommander) Wait() error { return c.cmd.Wait() }

// StdinPipe returns a pipe to the bash process's stdin.
func (c *execCommander) StdinPipe() (io.WriteCloser, error) { return c.cmd.StdinPipe() }

// StdoutPipe returns a pipe to the bash process's stdout.
func (c *execCommander) StdoutPipe() (io.ReadCloser, error) { return c.cmd.StdoutPipe() }

// StderrPipe returns a pipe to the bash process's stderr.
func (c *execCommander) StderrPipe() (io.ReadCloser, error) { return c.cmd.StderrPipe() }

// Shell manages a single persistent bash session.
// Use [NewShell] to create an instance. Must be closed with [Shell.Close] when done.
type Shell struct {
	cmd          commander      // process lifecycle (Start/Wait/pipes)
	stdin        io.WriteCloser // pipe to bash stdin; commands are written here
	stdout       *bufio.Scanner // line scanner over bash stdout; used for marker detection
	stderrBuf    bytes.Buffer   // accumulates stderr output from the readStderr goroutine
	stderrMu     sync.Mutex     // guards stderrBuf
	stderrMarker string         // current marker string to detect in stderr
	stderrDone   chan struct{}  // closed when stderr marker is detected
	mu           sync.Mutex     // serializes command execution (one command at a time)
	broken       error          // non-nil if session is desynchronized (e.g. context canceled mid-stream)
}

// newShellFromCommander creates a [Shell] from the given [commander].
// It obtains stdin/stdout/stderr pipes, starts the process, and launches
// a goroutine to accumulate stderr.
func newShellFromCommander(cmd commander) (*Shell, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdoutPipe.Close()
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdoutPipe.Close()
		stderrPipe.Close()
		return nil, fmt.Errorf("start bash: %w", err)
	}

	scanner := bufio.NewScanner(stdoutPipe)
	// TODO: Replace bufio.Scanner with a delimiter-based reader to remove the
	// per-line size cap entirely. See https://github.com/ogadra/20260327-cli-demo/issues/2
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // up to 1 MiB per line

	s := &Shell{
		cmd:    cmd,
		stdin:  stdin,
		stdout: scanner,
	}

	go s.readStderr(stderrPipe)

	return s, nil
}

// NewShell starts a new persistent bash session with "bash --norc --noprofile".
// It returns an error if the bash process fails to start.
// The caller must call [Shell.Close] to terminate the session.
func NewShell() (*Shell, error) {
	return newShellFromCommander(&execCommander{
		cmd: exec.Command("bash", "--norc", "--noprofile"),
	})
}

// readStderr continuously reads from the stderr pipe and appends to stderrBuf.
// When a stderr marker is set, it detects the marker line in the accumulated
// output and signals stderrDone by closing the channel. The marker line itself
// is stripped from the buffer.
// It runs as a goroutine for the lifetime of the bash process.
// Returns when the pipe is closed (i.e. bash exits).
func (s *Shell) readStderr(r io.Reader) {
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			s.stderrMu.Lock()
			s.stderrBuf.Write(buf[:n])
			s.checkStderrMarker()
			s.stderrMu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

// checkStderrMarker checks if the stderr buffer contains the current marker.
// If found, it strips the marker line from the buffer and signals stderrDone.
// Must be called with stderrMu held.
func (s *Shell) checkStderrMarker() {
	if s.stderrMarker == "" {
		return
	}
	content := s.stderrBuf.String()
	idx := strings.Index(content, s.stderrMarker)
	if idx < 0 {
		return
	}
	// Strip the marker line (marker + trailing newline) from the buffer.
	markerEnd := idx + len(s.stderrMarker)
	if markerEnd < len(content) && content[markerEnd] == '\n' {
		markerEnd++
	}
	s.stderrBuf.Reset()
	s.stderrBuf.WriteString(content[:idx] + content[markerEnd:])
	s.stderrMarker = ""
	close(s.stderrDone)
}

// resetStderr clears the stderr buffer. Called at the start of each command.
func (s *Shell) resetStderr() {
	s.stderrMu.Lock()
	s.stderrBuf.Reset()
	s.stderrMu.Unlock()
}

// getStderr returns the current contents of the stderr buffer.
func (s *Shell) getStderr() string {
	s.stderrMu.Lock()
	defer s.stderrMu.Unlock()
	return s.stderrBuf.String()
}

// ExecuteStream runs a command in the persistent bash session.
// Each stdout line is sent to stdoutCh as it arrives. The channel is closed when
// the command completes (or on error).
//
// Returns the exit code, accumulated stderr, and any error.
// Calls are serialized: concurrent calls block until the previous one completes.
func (s *Shell) ExecuteStream(ctx context.Context, command string, stdoutCh chan<- string) (int, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer close(stdoutCh)

	if s.broken != nil {
		return -1, "", s.broken
	}

	s.resetStderr()

	marker := fmt.Sprintf("__MRK_%d_END__", time.Now().UnixNano())

	s.stderrMu.Lock()
	s.stderrMarker = marker
	s.stderrDone = make(chan struct{})
	s.stderrMu.Unlock()

	script := fmt.Sprintf("%s\n__ec=$?\nbuiltin echo '%s' >&2\nbuiltin echo ''\nbuiltin echo '%s'${__ec}\n", command, marker, marker)

	if err := ctx.Err(); err != nil {
		return -1, "", fmt.Errorf("context: %w", err)
	}

	if _, err := io.WriteString(s.stdin, script); err != nil {
		return -1, "", fmt.Errorf("write command: %w", err)
	}

	// sendLine sends a line to stdoutCh, respecting context cancellation.
	sendLine := func(line string) error {
		select {
		case stdoutCh <- line:
			return nil
		case <-ctx.Done():
			return fmt.Errorf("context: %w", ctx.Err())
		}
	}

	// The script emits an empty line before the marker to ensure a newline
	// boundary after commands that produce output without a trailing newline
	// (e.g. printf "no newline"). We track whether the previous line was
	// empty so we can suppress it if it immediately precedes the marker.
	var pendingEmpty bool
	var sendErr error
	for s.stdout.Scan() {
		line := s.stdout.Text()
		if strings.HasPrefix(line, marker) {
			ecStr := line[len(marker):]
			exitCode, err := strconv.Atoi(ecStr)
			if err != nil {
				return -1, "", fmt.Errorf("parse exit code %q: %w", ecStr, err)
			}
			<-s.stderrDone
			stderr := s.getStderr()
			if sendErr != nil {
				// Marker was consumed successfully, so the session
				// is still synchronized despite the send failure.
				s.broken = nil
				return -1, "", sendErr
			}
			return exitCode, stderr, nil
		}
		if sendErr != nil {
			continue // drain until marker
		}
		if pendingEmpty {
			if err := sendLine(""); err != nil {
				sendErr = fmt.Errorf("shell session desynchronized: %w", err)
				s.broken = sendErr
				continue
			}
			pendingEmpty = false
		}
		if line == "" {
			pendingEmpty = true
		} else {
			if err := sendLine(line); err != nil {
				sendErr = fmt.Errorf("shell session desynchronized: %w", err)
				s.broken = sendErr
			}
		}
	}

	if err := s.stdout.Err(); err != nil {
		s.broken = fmt.Errorf("shell session desynchronized: scan stdout: %w", err)
		return -1, "", fmt.Errorf("scan stdout: %w", err)
	}
	s.broken = fmt.Errorf("shell session desynchronized: unexpected end of stdout")
	return -1, "", fmt.Errorf("unexpected end of stdout")
}

// Close terminates the persistent shell session by sending "exit" to bash
// and waiting for the process to finish.
// It acquires mu to ensure no concurrent [Shell.ExecuteStream] is writing to stdin.
func (s *Shell) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := io.WriteString(s.stdin, "exit\n")
	if err != nil {
		return fmt.Errorf("write exit: %w", err)
	}
	return s.cmd.Wait()
}
