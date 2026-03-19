package main

import (
	"bufio"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// newTestShell creates a [Shell] for testing and registers cleanup.
func newTestShell(t *testing.T) *Shell {
	t.Helper()
	s, err := NewShell()
	if err != nil {
		t.Fatalf("NewShell() error: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("Close() error: %v", err)
		}
	})
	return s
}

// execStream is a test helper that runs [Shell.ExecuteStream] and collects stdout lines.
// It drains the channel concurrently to avoid deadlocks when output exceeds the buffer.
func execStream(t *testing.T, s *Shell, command string) ([]string, int, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ch := make(chan string, 100)
	var lines []string
	done := make(chan struct{})
	go func() {
		defer close(done)
		for line := range ch {
			lines = append(lines, line)
		}
	}()
	exitCode, stderr, err := s.ExecuteStream(ctx, command, ch)
	if err != nil {
		t.Fatalf("ExecuteStream(%q) error: %v", command, err)
	}
	<-done
	return lines, exitCode, stderr
}

// TestNewShell verifies that a Shell can be created and closed successfully.
func TestNewShell(t *testing.T) {
	s, err := NewShell()
	if err != nil {
		t.Fatalf("NewShell() error: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
}

// TestStreamBasic verifies basic command execution with stdout and exit code.
func TestStreamBasic(t *testing.T) {
	s := newTestShell(t)
	lines, exitCode, stderr := execStream(t, s, "echo hello")
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if len(lines) != 1 || lines[0] != "hello" {
		t.Errorf("lines = %v, want [hello]", lines)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

// TestStreamExitCode verifies that a failing command returns exit code 1.
func TestStreamExitCode(t *testing.T) {
	s := newTestShell(t)
	_, exitCode, _ := execStream(t, s, "false")
	if exitCode != 1 {
		t.Errorf("exitCode = %d, want 1", exitCode)
	}
}

// TestStreamStderr verifies that stderr output is captured separately.
func TestStreamStderr(t *testing.T) {
	s := newTestShell(t)
	lines, exitCode, stderr := execStream(t, s, "echo err >&2")
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if len(lines) != 0 {
		t.Errorf("lines = %v, want empty", lines)
	}
	if !strings.Contains(stderr, "err") {
		t.Errorf("stderr = %q, want to contain %q", stderr, "err")
	}
}

// TestStreamStdoutAndStderr verifies that stdout and stderr are separated correctly.
func TestStreamStdoutAndStderr(t *testing.T) {
	s := newTestShell(t)
	lines, exitCode, stderr := execStream(t, s, "echo out && echo err >&2")
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if len(lines) != 1 || lines[0] != "out" {
		t.Errorf("lines = %v, want [out]", lines)
	}
	if !strings.Contains(stderr, "err") {
		t.Errorf("stderr = %q, want to contain %q", stderr, "err")
	}
}

// TestSessionPersistenceCd verifies that cd persists across commands.
func TestSessionPersistenceCd(t *testing.T) {
	s := newTestShell(t)
	execStream(t, s, "cd /tmp")
	lines, exitCode, _ := execStream(t, s, "pwd")
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if len(lines) != 1 || lines[0] != "/tmp" {
		t.Errorf("lines = %v, want [/tmp]", lines)
	}
}

// TestSessionPersistenceEnv verifies that exported env vars persist across commands.
func TestSessionPersistenceEnv(t *testing.T) {
	s := newTestShell(t)
	execStream(t, s, "export FOO=bar")
	lines, exitCode, _ := execStream(t, s, "echo $FOO")
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if len(lines) != 1 || lines[0] != "bar" {
		t.Errorf("lines = %v, want [bar]", lines)
	}
}

// TestSessionPersistenceAlias verifies that aliases persist across commands.
func TestSessionPersistenceAlias(t *testing.T) {
	s := newTestShell(t)
	execStream(t, s, "alias greet='echo hi'")
	execStream(t, s, "shopt -s expand_aliases")
	lines, exitCode, _ := execStream(t, s, "greet")
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if len(lines) != 1 || lines[0] != "hi" {
		t.Errorf("lines = %v, want [hi]", lines)
	}
}

// TestStreamMultilineOutput verifies that multiple output lines are collected correctly.
func TestStreamMultilineOutput(t *testing.T) {
	s := newTestShell(t)
	lines, exitCode, _ := execStream(t, s, "printf 'line1\nline2\nline3\n'")
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if len(lines) != 3 {
		t.Errorf("got %d lines, want 3: %v", len(lines), lines)
	}
	expected := []string{"line1", "line2", "line3"}
	for i, want := range expected {
		if i < len(lines) && lines[i] != want {
			t.Errorf("lines[%d] = %q, want %q", i, lines[i], want)
		}
	}
}

// TestStreamEmptyCommand verifies that an empty command succeeds with exit code 0.
func TestStreamEmptyCommand(t *testing.T) {
	s := newTestShell(t)
	_, exitCode, _ := execStream(t, s, "")
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
}

// TestStreamOutputWithEmptyLines verifies that empty lines in command output are preserved.
func TestStreamOutputWithEmptyLines(t *testing.T) {
	s := newTestShell(t)
	lines, exitCode, _ := execStream(t, s, `printf 'a\n\nb\n'`)
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	expected := []string{"a", "", "b"}
	if len(lines) != len(expected) {
		t.Fatalf("got %d lines %v, want %v", len(lines), lines, expected)
	}
	for i, want := range expected {
		if lines[i] != want {
			t.Errorf("lines[%d] = %q, want %q", i, lines[i], want)
		}
	}
}

// TestStreamLongLine verifies that lines exceeding the default 64KiB scanner buffer work.
func TestStreamLongLine(t *testing.T) {
	s := newTestShell(t)
	lines, exitCode, _ := execStream(t, s, `python3 -c "print('A'*100000)" 2>/dev/null || printf '%0100000d\n' 0`)
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	gotLen := 0
	if len(lines) > 0 {
		gotLen = len(lines[0])
	}
	if len(lines) != 1 || gotLen != 100000 {
		t.Errorf("got %d lines, first line length = %d, want 1 line of 100000 chars", len(lines), gotLen)
	}
}

// TestStreamOver100Lines verifies that commands producing more than 100 stdout
// lines (exceeding the execStream helper's channel buffer) do not deadlock.
func TestStreamOver100Lines(t *testing.T) {
	s := newTestShell(t)
	lines, exitCode, _ := execStream(t, s, `for i in $(seq 1 200); do echo "line $i"; done`)
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if len(lines) != 200 {
		t.Errorf("got %d lines, want 200", len(lines))
	}
}

// TestStreamStderrLargeOutput verifies that large stderr output is fully captured
// via the stderr marker protocol (not lost by a fixed sleep).
func TestStreamStderrLargeOutput(t *testing.T) {
	s := newTestShell(t)
	_, exitCode, stderr := execStream(t, s, `for i in $(seq 1 500); do echo "stderr line $i" >&2; done`)
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	lines := strings.Split(strings.TrimSpace(stderr), "\n")
	if len(lines) != 500 {
		t.Errorf("got %d stderr lines, want 500", len(lines))
	}
}

// TestStreamNoTrailingNewline verifies that commands producing output without
// a trailing newline do not hang the marker detection.
func TestStreamNoTrailingNewline(t *testing.T) {
	s := newTestShell(t)
	lines, exitCode, _ := execStream(t, s, `printf "no newline"`)
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if len(lines) != 1 || lines[0] != "no newline" {
		t.Errorf("lines = %v, want [no newline]", lines)
	}
}

// TestClose verifies that Close terminates the bash session cleanly.
func TestClose(t *testing.T) {
	s, err := NewShell()
	if err != nil {
		t.Fatalf("NewShell() error: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
}

// --- Error injection tests using fakeCommander ---

var errFake = errors.New("fake error")

// fakeCommander is a test double for [commander] that injects errors
// into pipe creation and process lifecycle methods.
type fakeCommander struct {
	stdinErr  error
	stdoutErr error
	stderrErr error
	startErr  error
	waitErr   error
	stdinW    io.WriteCloser
	stdoutR   io.ReadCloser
	stderrR   io.ReadCloser
}

// StdinPipe returns the configured writer or error.
func (f *fakeCommander) StdinPipe() (io.WriteCloser, error) {
	if f.stdinErr != nil {
		return nil, f.stdinErr
	}
	return f.stdinW, nil
}

// StdoutPipe returns the configured reader or error.
func (f *fakeCommander) StdoutPipe() (io.ReadCloser, error) {
	if f.stdoutErr != nil {
		return nil, f.stdoutErr
	}
	return f.stdoutR, nil
}

// StderrPipe returns the configured reader or error.
func (f *fakeCommander) StderrPipe() (io.ReadCloser, error) {
	if f.stderrErr != nil {
		return nil, f.stderrErr
	}
	return f.stderrR, nil
}

// Start returns the configured error.
func (f *fakeCommander) Start() error { return f.startErr }

// Wait returns the configured error.
func (f *fakeCommander) Wait() error { return f.waitErr }

// nopWriteCloser wraps an [io.Writer] with a no-op Close method.
type nopWriteCloser struct{ io.Writer }

// Close is a no-op.
func (nopWriteCloser) Close() error { return nil }

// trackingCloser wraps an [io.Writer] and records whether Close was called.
type trackingCloser struct {
	io.Writer
	closed bool
}

// Close records the call and returns nil.
func (t *trackingCloser) Close() error { t.closed = true; return nil }

// trackingReadCloser wraps an [io.Reader] and records whether Close was called.
type trackingReadCloser struct {
	io.Reader
	closed bool
}

// Read delegates to the wrapped reader.
func (t *trackingReadCloser) Read(p []byte) (int, error) { return t.Reader.Read(p) }

// Close records the call and returns nil.
func (t *trackingReadCloser) Close() error { t.closed = true; return nil }

// TestNewShellStdinPipeError verifies that a stdin pipe error is reported correctly.
func TestNewShellStdinPipeError(t *testing.T) {
	_, err := newShellFromCommander(&fakeCommander{stdinErr: errFake})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "stdin pipe") {
		t.Errorf("error = %q, want to contain %q", err, "stdin pipe")
	}
}

// TestNewShellStdoutPipeError verifies that a stdout pipe error is reported
// and previously acquired pipes are closed (no FD leak).
func TestNewShellStdoutPipeError(t *testing.T) {
	stdinTracker := &trackingCloser{Writer: &strings.Builder{}}
	_, err := newShellFromCommander(&fakeCommander{
		stdinW:    stdinTracker,
		stdoutErr: errFake,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "stdout pipe") {
		t.Errorf("error = %q, want to contain %q", err, "stdout pipe")
	}
	if !stdinTracker.closed {
		t.Error("stdin pipe was not closed on stdout pipe error (FD leak)")
	}
}

// TestNewShellStderrPipeError verifies that a stderr pipe error is reported
// and previously acquired pipes are closed (no FD leak).
func TestNewShellStderrPipeError(t *testing.T) {
	stdinTracker := &trackingCloser{Writer: &strings.Builder{}}
	stdoutTracker := &trackingReadCloser{Reader: strings.NewReader("")}
	_, err := newShellFromCommander(&fakeCommander{
		stdinW:    stdinTracker,
		stdoutR:   stdoutTracker,
		stderrErr: errFake,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "stderr pipe") {
		t.Errorf("error = %q, want to contain %q", err, "stderr pipe")
	}
	if !stdinTracker.closed {
		t.Error("stdin pipe was not closed on stderr pipe error (FD leak)")
	}
	if !stdoutTracker.closed {
		t.Error("stdout pipe was not closed on stderr pipe error (FD leak)")
	}
}

// TestNewShellStartError verifies that a process start error is reported
// and all acquired pipes are closed (no FD leak).
func TestNewShellStartError(t *testing.T) {
	stdinTracker := &trackingCloser{Writer: &strings.Builder{}}
	stdoutTracker := &trackingReadCloser{Reader: strings.NewReader("")}
	stderrTracker := &trackingReadCloser{Reader: strings.NewReader("")}
	_, err := newShellFromCommander(&fakeCommander{
		stdinW:   stdinTracker,
		stdoutR:  stdoutTracker,
		stderrR:  stderrTracker,
		startErr: errFake,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "start bash") {
		t.Errorf("error = %q, want to contain %q", err, "start bash")
	}
	if !stdinTracker.closed {
		t.Error("stdin pipe was not closed on start error (FD leak)")
	}
	if !stdoutTracker.closed {
		t.Error("stdout pipe was not closed on start error (FD leak)")
	}
	if !stderrTracker.closed {
		t.Error("stderr pipe was not closed on start error (FD leak)")
	}
}

// TestStreamContextCanceled verifies that a pre-canceled context returns immediately.
func TestStreamContextCanceled(t *testing.T) {
	s := newTestShell(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	ch := make(chan string, 100)
	_, _, err := s.ExecuteStream(ctx, `echo hello`, ch)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "context") {
		t.Errorf("error = %q, want to contain %q", err, "context")
	}
}

// TestStreamContextCanceledDuringSend verifies that cancellation during stdout
// line sending returns a context error instead of deadlocking.
func TestStreamContextCanceledDuringSend(t *testing.T) {
	s := newTestShell(t)
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan string)
	errCh := make(chan error, 1)
	go func() {
		_, _, err := s.ExecuteStream(ctx, `echo line1; echo line2; echo line3`, ch)
		errCh <- err
	}()
	<-ch
	cancel()
	for range ch {
	}
	err := <-errCh
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "context") {
		t.Errorf("error = %q, want to contain %q", err, "context")
	}
}

// TestStreamContextCanceledDuringEmptyLine verifies that cancellation during
// an empty line send returns a context error.
func TestStreamContextCanceledDuringEmptyLine(t *testing.T) {
	s := newTestShell(t)
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan string)
	errCh := make(chan error, 1)
	go func() {
		_, _, err := s.ExecuteStream(ctx, `printf 'a\n\nb\n'`, ch)
		errCh <- err
	}()
	<-ch // read "a"
	cancel()
	for range ch {
	}
	err := <-errCh
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "context") {
		t.Errorf("error = %q, want to contain %q", err, "context")
	}
}

// TestStreamSlowConsumer verifies that ExecuteStream works with an unbuffered channel.
func TestStreamSlowConsumer(t *testing.T) {
	s := newTestShell(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ch := make(chan string)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range ch {
		}
	}()
	exitCode, _, err := s.ExecuteStream(ctx, `for i in $(seq 1 10); do echo "line $i"; done`, ch)
	<-done
	if err != nil {
		t.Fatalf("ExecuteStream error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
}

// TestCheckStderrMarkerNoMarkerSet verifies that checkStderrMarker is a no-op
// when no marker is set.
func TestCheckStderrMarkerNoMarkerSet(t *testing.T) {
	s := &Shell{}
	s.stderrMu.Lock()
	s.stderrBuf.WriteString("some output")
	s.checkStderrMarker()
	s.stderrMu.Unlock()
	if got := s.getStderr(); got != "some output" {
		t.Errorf("stderr = %q, want %q", got, "some output")
	}
}

// TestCheckStderrMarkerNoTrailingNewline verifies that checkStderrMarker handles
// a marker at the end of the buffer without a trailing newline.
func TestCheckStderrMarkerNoTrailingNewline(t *testing.T) {
	s := &Shell{}
	s.stderrMarker = "__MRK_TEST__"
	s.stderrDone = make(chan struct{})

	s.stderrMu.Lock()
	s.stderrBuf.WriteString("some error\n__MRK_TEST__")
	s.checkStderrMarker()
	s.stderrMu.Unlock()

	select {
	case <-s.stderrDone:
	default:
		t.Fatal("stderrDone was not closed")
	}

	got := s.getStderr()
	if got != "some error\n" {
		t.Errorf("stderr = %q, want %q", got, "some error\n")
	}
}

// failWriter always returns an error on Write.
type failWriter struct{}

// Write always returns errFake.
func (failWriter) Write([]byte) (int, error) { return 0, errFake }

// Close is a no-op.
func (failWriter) Close() error { return nil }

// TestStreamWriteError verifies that a stdin write failure is reported.
func TestStreamWriteError(t *testing.T) {
	s := &Shell{
		stdin:  failWriter{},
		stdout: bufio.NewScanner(strings.NewReader("")),
		cmd:    &fakeCommander{},
	}
	ch := make(chan string, 10)
	_, _, err := s.ExecuteStream(context.Background(), "echo hello", ch)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "write command") {
		t.Errorf("error = %q, want to contain %q", err, "write command")
	}
}

// TestStreamWriteErrorMarksBroken verifies that a stdin write failure marks
// the session as broken, preventing subsequent commands.
func TestStreamWriteErrorMarksBroken(t *testing.T) {
	s := &Shell{
		stdin:  failWriter{},
		stdout: bufio.NewScanner(strings.NewReader("")),
		cmd:    &fakeCommander{},
	}
	ch := make(chan string, 10)
	s.ExecuteStream(context.Background(), "echo hello", ch)

	// Next call should fail with desynchronized error.
	ch2 := make(chan string, 10)
	_, _, err := s.ExecuteStream(context.Background(), "echo hello", ch2)
	if err == nil {
		t.Fatal("expected broken session error, got nil")
	}
	if !strings.Contains(err.Error(), "desynchronized") {
		t.Errorf("error = %q, want to contain %q", err, "desynchronized")
	}
}

// TestStreamUnexpectedEOF verifies that stdout closing without a marker is reported.
func TestStreamUnexpectedEOF(t *testing.T) {
	s := &Shell{
		stdin:  nopWriteCloser{&strings.Builder{}},
		stdout: bufio.NewScanner(strings.NewReader("some output\n")),
		cmd:    &fakeCommander{},
	}
	ch := make(chan string, 10)
	_, _, err := s.ExecuteStream(context.Background(), "echo hello", ch)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected end of stdout") {
		t.Errorf("error = %q, want to contain %q", err, "unexpected end of stdout")
	}
}

// errReader returns an error on Read (for scanner error path).
type errReader struct{}

// Read always returns errFake.
func (errReader) Read([]byte) (int, error) { return 0, errFake }

// TestStreamScanError verifies that a scanner read error is reported.
func TestStreamScanError(t *testing.T) {
	s := &Shell{
		stdin:  nopWriteCloser{&strings.Builder{}},
		stdout: bufio.NewScanner(errReader{}),
		cmd:    &fakeCommander{},
	}
	ch := make(chan string, 10)
	_, _, err := s.ExecuteStream(context.Background(), "echo hello", ch)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "scan stdout") {
		t.Errorf("error = %q, want to contain %q", err, "scan stdout")
	}
}

// markerCapturingWriter captures what [Shell.ExecuteStream] writes to stdin
// so we can extract the marker and produce a fake stdout response.
// All methods are safe for concurrent use.
type markerCapturingWriter struct {
	mu  sync.Mutex
	buf strings.Builder
}

// Write appends data to the internal buffer.
func (w *markerCapturingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

// Close is a no-op.
func (w *markerCapturingWriter) Close() error { return nil }

// String returns the current buffer contents.
func (w *markerCapturingWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

// extractMarker parses a marker string from the captured stdin script.
func extractMarker(written string) string {
	for _, line := range strings.Split(written, "\n") {
		if strings.HasPrefix(line, "builtin echo '__MRK_") {
			start := strings.Index(line, "'") + 1
			end := strings.LastIndex(line, "'")
			if start > 0 && end > start {
				return line[start:end]
			}
		}
	}
	return ""
}

// TestStreamInvalidExitCode verifies that an unparseable exit code after the
// marker is reported as an error.
func TestStreamInvalidExitCode(t *testing.T) {
	stdinCapture := &markerCapturingWriter{}
	stdoutR, stdoutW := io.Pipe()

	s := &Shell{
		stdin:  stdinCapture,
		stdout: bufio.NewScanner(stdoutR),
		cmd:    &fakeCommander{},
	}

	errCh := make(chan error, 1)
	ch := make(chan string, 10)
	go func() {
		_, _, err := s.ExecuteStream(context.Background(), "echo hello", ch)
		errCh <- err
	}()

	go func() {
		timeout := time.After(5 * time.Second)
		for {
			select {
			case <-timeout:
				stdoutW.Close()
				return
			default:
			}
			marker := extractMarker(stdinCapture.String())
			if marker != "" {
				stdoutW.Write([]byte(marker + "notanumber\n"))
				stdoutW.Close()
				return
			}
			time.Sleep(time.Millisecond)
		}
	}()

	err := <-errCh
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "parse exit code") {
		t.Errorf("error = %q, want to contain %q", err, "parse exit code")
	}
}

// TestStreamDrainMarkerOnContextCancel verifies that when a context is canceled
// mid-stream, ExecuteStream drains stdout until the marker before returning,
// keeping the session usable for subsequent calls.
func TestStreamDrainMarkerOnContextCancel(t *testing.T) {
	s := newTestShell(t)
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan string)
	errCh := make(chan error, 1)
	go func() {
		_, _, err := s.ExecuteStream(ctx, `echo line1; echo line2; echo line3`, ch)
		errCh <- err
	}()
	<-ch // read first line
	cancel()
	for range ch {
	}
	err := <-errCh
	if err == nil {
		t.Fatal("expected error from canceled context, got nil")
	}
	if !strings.Contains(err.Error(), "context") {
		t.Errorf("error = %q, want to contain %q", err, "context")
	}

	// Session should still be usable because the marker was drained.
	lines, exitCode, _ := execStream(t, s, `echo recovered`)
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if len(lines) != 1 || lines[0] != "recovered" {
		t.Errorf("lines = %v, want [recovered]", lines)
	}
}

// TestStreamBrokenOnUnexpectedEOF verifies that if stdout closes before the
// marker is found (e.g. bash crashes), the session is marked as broken.
func TestStreamBrokenOnUnexpectedEOF(t *testing.T) {
	s := &Shell{
		stdin:  nopWriteCloser{&strings.Builder{}},
		stdout: bufio.NewScanner(strings.NewReader("some output\n")),
		cmd:    &fakeCommander{},
	}
	ch := make(chan string, 10)
	_, _, err := s.ExecuteStream(context.Background(), "echo hello", ch)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Session should be broken.
	ch2 := make(chan string, 10)
	_, _, err = s.ExecuteStream(context.Background(), "echo hello", ch2)
	if err == nil {
		t.Fatal("expected broken session error, got nil")
	}
	if !strings.Contains(err.Error(), "desynchronized") {
		t.Errorf("error = %q, want to contain %q", err, "desynchronized")
	}
}

// TestCloseWriteError verifies that a stdin write failure during Close is reported.
func TestCloseWriteError(t *testing.T) {
	s := &Shell{
		stdin: failWriter{},
		cmd:   &fakeCommander{},
	}
	err := s.Close()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "write exit") {
		t.Errorf("error = %q, want to contain %q", err, "write exit")
	}
}

// TestCloseWaitError verifies that a process wait error during Close is propagated.
func TestCloseWaitError(t *testing.T) {
	s := &Shell{
		stdin: nopWriteCloser{&strings.Builder{}},
		cmd:   &fakeCommander{waitErr: errFake},
	}
	err := s.Close()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errFake) {
		t.Errorf("error = %v, want %v", err, errFake)
	}
}

// TestCloseCallsWaitOnWriteError verifies that Close calls Wait even when
// writing "exit" to stdin fails, so the process is properly reaped.
func TestCloseCallsWaitOnWriteError(t *testing.T) {
	cmd := &fakeCommander{}
	tc := &trackingCommander{inner: cmd}
	s := &Shell{
		stdin: failWriter{},
		cmd:   tc,
	}
	s.Close()
	if !tc.waitCalled {
		t.Error("Wait() was not called after write exit failure")
	}
}

// TestCloseWriteAndWaitError verifies that Close returns a combined error
// when both writing "exit" and waiting for the process fail.
func TestCloseWriteAndWaitError(t *testing.T) {
	s := &Shell{
		stdin: failWriter{},
		cmd:   &fakeCommander{waitErr: errFake},
	}
	err := s.Close()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "write exit") {
		t.Errorf("error = %q, want to contain %q", err, "write exit")
	}
	if !strings.Contains(err.Error(), "wait") {
		t.Errorf("error = %q, want to contain %q", err, "wait")
	}
}

// trackingCommander wraps a [commander] and records whether Wait was called.
type trackingCommander struct {
	inner      commander
	waitCalled bool
}

func (t *trackingCommander) Start() error                       { return t.inner.Start() }
func (t *trackingCommander) Wait() error                        { t.waitCalled = true; return t.inner.Wait() }
func (t *trackingCommander) StdinPipe() (io.WriteCloser, error) { return t.inner.StdinPipe() }
func (t *trackingCommander) StdoutPipe() (io.ReadCloser, error) { return t.inner.StdoutPipe() }
func (t *trackingCommander) StderrPipe() (io.ReadCloser, error) { return t.inner.StderrPipe() }
