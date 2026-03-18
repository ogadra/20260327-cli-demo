package main

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"testing"
)

func newTestShell(t *testing.T) *Shell {
	t.Helper()
	s, err := NewShell()
	if err != nil {
		t.Fatalf("NewShell() error: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// execStream is a test helper that runs ExecuteStream and collects stdout lines.
func execStream(t *testing.T, s *Shell, command string) ([]string, int, string) {
	t.Helper()
	ch := make(chan string, 100)
	exitCode, stderr, err := s.ExecuteStream(command, ch)
	if err != nil {
		t.Fatalf("ExecuteStream(%q) error: %v", command, err)
	}
	var lines []string
	for line := range ch {
		lines = append(lines, line)
	}
	return lines, exitCode, stderr
}

func TestNewShell(t *testing.T) {
	s, err := NewShell()
	if err != nil {
		t.Fatalf("NewShell() error: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
}

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

func TestStreamExitCode(t *testing.T) {
	s := newTestShell(t)
	_, exitCode, _ := execStream(t, s, "false")
	if exitCode != 1 {
		t.Errorf("exitCode = %d, want 1", exitCode)
	}
}

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

func TestStreamEmptyCommand(t *testing.T) {
	s := newTestShell(t)
	_, exitCode, _ := execStream(t, s, "")
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
}

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

func (f *fakeCommander) StdinPipe() (io.WriteCloser, error) {
	if f.stdinErr != nil {
		return nil, f.stdinErr
	}
	return f.stdinW, nil
}

func (f *fakeCommander) StdoutPipe() (io.ReadCloser, error) {
	if f.stdoutErr != nil {
		return nil, f.stdoutErr
	}
	return f.stdoutR, nil
}

func (f *fakeCommander) StderrPipe() (io.ReadCloser, error) {
	if f.stderrErr != nil {
		return nil, f.stderrErr
	}
	return f.stderrR, nil
}

func (f *fakeCommander) Start() error { return f.startErr }
func (f *fakeCommander) Wait() error  { return f.waitErr }

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

func TestNewShellStdinPipeError(t *testing.T) {
	_, err := newShellFromCommander(&fakeCommander{stdinErr: errFake})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "stdin pipe") {
		t.Errorf("error = %q, want to contain %q", err, "stdin pipe")
	}
}

func TestNewShellStdoutPipeError(t *testing.T) {
	_, err := newShellFromCommander(&fakeCommander{
		stdinW:    nopWriteCloser{&strings.Builder{}},
		stdoutErr: errFake,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "stdout pipe") {
		t.Errorf("error = %q, want to contain %q", err, "stdout pipe")
	}
}

func TestNewShellStderrPipeError(t *testing.T) {
	_, err := newShellFromCommander(&fakeCommander{
		stdinW:    nopWriteCloser{&strings.Builder{}},
		stdoutR:   io.NopCloser(strings.NewReader("")),
		stderrErr: errFake,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "stderr pipe") {
		t.Errorf("error = %q, want to contain %q", err, "stderr pipe")
	}
}

func TestNewShellStartError(t *testing.T) {
	_, err := newShellFromCommander(&fakeCommander{
		stdinW:   nopWriteCloser{&strings.Builder{}},
		stdoutR:  io.NopCloser(strings.NewReader("")),
		stderrR:  io.NopCloser(strings.NewReader("")),
		startErr: errFake,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "start bash") {
		t.Errorf("error = %q, want to contain %q", err, "start bash")
	}
}

// failWriter always returns an error on Write.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errFake }
func (failWriter) Close() error              { return nil }

func TestStreamWriteError(t *testing.T) {
	s := &Shell{
		stdin:  failWriter{},
		stdout: bufio.NewScanner(strings.NewReader("")),
		cmd:    &fakeCommander{},
	}
	ch := make(chan string, 10)
	_, _, err := s.ExecuteStream("echo hello", ch)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "write command") {
		t.Errorf("error = %q, want to contain %q", err, "write command")
	}
}

func TestStreamUnexpectedEOF(t *testing.T) {
	s := &Shell{
		stdin:  nopWriteCloser{&strings.Builder{}},
		stdout: bufio.NewScanner(strings.NewReader("some output\n")),
		cmd:    &fakeCommander{},
	}
	ch := make(chan string, 10)
	_, _, err := s.ExecuteStream("echo hello", ch)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected end of stdout") {
		t.Errorf("error = %q, want to contain %q", err, "unexpected end of stdout")
	}
}

// errReader returns an error on Read (for scanner error path).
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errFake }

func TestStreamScanError(t *testing.T) {
	s := &Shell{
		stdin:  nopWriteCloser{&strings.Builder{}},
		stdout: bufio.NewScanner(errReader{}),
		cmd:    &fakeCommander{},
	}
	ch := make(chan string, 10)
	_, _, err := s.ExecuteStream("echo hello", ch)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "scan stdout") {
		t.Errorf("error = %q, want to contain %q", err, "scan stdout")
	}
}

// markerCapturingWriter captures what ExecuteStream writes to stdin so we can
// extract the marker and produce a fake stdout response with an invalid exit code.
type markerCapturingWriter struct {
	buf strings.Builder
}

func (w *markerCapturingWriter) Write(p []byte) (int, error) { return w.buf.Write(p) }
func (w *markerCapturingWriter) Close() error                { return nil }

func extractMarker(written string) string {
	for _, line := range strings.Split(written, "\n") {
		if strings.HasPrefix(line, "echo '__MRK_") {
			start := strings.Index(line, "'") + 1
			end := strings.LastIndex(line, "'")
			if start > 0 && end > start {
				return line[start:end]
			}
		}
	}
	return ""
}

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
		_, _, err := s.ExecuteStream("echo hello", ch)
		errCh <- err
	}()

	go func() {
		for {
			marker := extractMarker(stdinCapture.buf.String())
			if marker != "" {
				stdoutW.Write([]byte(marker + "notanumber\n"))
				stdoutW.Close()
				return
			}
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
