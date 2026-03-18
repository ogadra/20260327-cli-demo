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

func TestNewShell(t *testing.T) {
	s, err := NewShell()
	if err != nil {
		t.Fatalf("NewShell() error: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
}

func TestExecuteBasic(t *testing.T) {
	s := newTestShell(t)
	stdout, exitCode, stderr, err := s.Execute("echo hello")
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if strings.TrimSpace(stdout) != "hello" {
		t.Errorf("stdout = %q, want %q", stdout, "hello\n")
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

func TestExecuteExitCode(t *testing.T) {
	s := newTestShell(t)
	_, exitCode, _, err := s.Execute("false")
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if exitCode != 1 {
		t.Errorf("exitCode = %d, want 1", exitCode)
	}
}

func TestExecuteStderr(t *testing.T) {
	s := newTestShell(t)
	stdout, exitCode, stderr, err := s.Execute("echo err >&2")
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "err") {
		t.Errorf("stderr = %q, want to contain %q", stderr, "err")
	}
}

func TestExecuteStdoutAndStderr(t *testing.T) {
	s := newTestShell(t)
	stdout, exitCode, stderr, err := s.Execute("echo out && echo err >&2")
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if !strings.Contains(stdout, "out") {
		t.Errorf("stdout = %q, want to contain %q", stdout, "out")
	}
	if !strings.Contains(stderr, "err") {
		t.Errorf("stderr = %q, want to contain %q", stderr, "err")
	}
}

func TestSessionPersistenceCd(t *testing.T) {
	s := newTestShell(t)
	_, _, _, err := s.Execute("cd /tmp")
	if err != nil {
		t.Fatalf("Execute(cd) error: %v", err)
	}
	stdout, exitCode, _, err := s.Execute("pwd")
	if err != nil {
		t.Fatalf("Execute(pwd) error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if strings.TrimSpace(stdout) != "/tmp" {
		t.Errorf("stdout = %q, want %q", stdout, "/tmp\n")
	}
}

func TestSessionPersistenceEnv(t *testing.T) {
	s := newTestShell(t)
	_, _, _, err := s.Execute("export FOO=bar")
	if err != nil {
		t.Fatalf("Execute(export) error: %v", err)
	}
	stdout, exitCode, _, err := s.Execute("echo $FOO")
	if err != nil {
		t.Fatalf("Execute(echo) error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if strings.TrimSpace(stdout) != "bar" {
		t.Errorf("stdout = %q, want %q", stdout, "bar\n")
	}
}

func TestSessionPersistenceAlias(t *testing.T) {
	s := newTestShell(t)
	_, _, _, err := s.Execute("alias greet='echo hi'")
	if err != nil {
		t.Fatalf("Execute(alias) error: %v", err)
	}
	_, _, _, err = s.Execute("shopt -s expand_aliases")
	if err != nil {
		t.Fatalf("Execute(shopt) error: %v", err)
	}
	stdout, exitCode, _, err := s.Execute("greet")
	if err != nil {
		t.Fatalf("Execute(greet) error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if strings.TrimSpace(stdout) != "hi" {
		t.Errorf("stdout = %q, want %q", stdout, "hi\n")
	}
}

func TestExecuteMultilineOutput(t *testing.T) {
	s := newTestShell(t)
	stdout, exitCode, _, err := s.Execute("printf 'line1\nline2\nline3\n'")
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("got %d lines, want 3: %q", len(lines), stdout)
	}
}

func TestExecuteEmptyCommand(t *testing.T) {
	s := newTestShell(t)
	_, exitCode, _, err := s.Execute("")
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
}

func TestExecuteStream(t *testing.T) {
	s := newTestShell(t)
	ch := make(chan string, 10)
	exitCode, _, err := s.ExecuteStream("printf 'a\nb\nc\n'", ch)
	if err != nil {
		t.Fatalf("ExecuteStream() error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}

	var lines []string
	for line := range ch {
		lines = append(lines, line)
	}
	if len(lines) != 3 {
		t.Errorf("got %d lines, want 3: %v", len(lines), lines)
	}
	expected := []string{"a", "b", "c"}
	for i, want := range expected {
		if i < len(lines) && lines[i] != want {
			t.Errorf("lines[%d] = %q, want %q", i, lines[i], want)
		}
	}
}

func TestExecuteStreamExitCode(t *testing.T) {
	s := newTestShell(t)
	ch := make(chan string, 10)
	exitCode, _, err := s.ExecuteStream("false", ch)
	if err != nil {
		t.Fatalf("ExecuteStream() error: %v", err)
	}
	if exitCode != 1 {
		t.Errorf("exitCode = %d, want 1", exitCode)
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

func TestNewShellStdinPipeError(t *testing.T) {
	_, err := newShellFromCommander(&fakeCommander{stdinErr: errFake})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "stdin pipe") {
		t.Errorf("error = %q, want to contain %q", err, "stdin pipe")
	}
}

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

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

func TestExecuteWriteError(t *testing.T) {
	s := &Shell{
		stdin:  failWriter{},
		stdout: bufio.NewScanner(strings.NewReader("")),
		cmd:    &fakeCommander{},
	}
	_, _, _, err := s.Execute("echo hello")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "write command") {
		t.Errorf("error = %q, want to contain %q", err, "write command")
	}
}

func TestExecuteStreamWriteError(t *testing.T) {
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

func TestExecuteUnexpectedEOF(t *testing.T) {
	s := &Shell{
		stdin:  nopWriteCloser{&strings.Builder{}},
		stdout: bufio.NewScanner(strings.NewReader("some output\n")),
		cmd:    &fakeCommander{},
	}
	_, _, _, err := s.Execute("echo hello")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected end of stdout") {
		t.Errorf("error = %q, want to contain %q", err, "unexpected end of stdout")
	}
}

func TestExecuteStreamUnexpectedEOF(t *testing.T) {
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

func TestExecuteScanError(t *testing.T) {
	s := &Shell{
		stdin:  nopWriteCloser{&strings.Builder{}},
		stdout: bufio.NewScanner(errReader{}),
		cmd:    &fakeCommander{},
	}
	_, _, _, err := s.Execute("echo hello")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "scan stdout") {
		t.Errorf("error = %q, want to contain %q", err, "scan stdout")
	}
}

func TestExecuteStreamScanError(t *testing.T) {
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

// markerCapturingWriter captures what Execute writes to stdin so we can
// extract the marker and produce a fake stdout response with an invalid exit code.
type markerCapturingWriter struct {
	buf strings.Builder
}

func (w *markerCapturingWriter) Write(p []byte) (int, error) { return w.buf.Write(p) }
func (w *markerCapturingWriter) Close() error                { return nil }

func extractMarker(written string) string {
	for _, line := range strings.Split(written, "\n") {
		if strings.HasPrefix(line, "echo '__MRK_") {
			// line is: echo '__MRK_<nano>_END__'${__ec}
			start := strings.Index(line, "'") + 1
			end := strings.LastIndex(line, "'")
			if start > 0 && end > start {
				return line[start:end]
			}
		}
	}
	return ""
}

func TestExecuteInvalidExitCode(t *testing.T) {
	// We need to capture the marker that Execute generates, then provide
	// a stdout that has that marker followed by a non-numeric string.
	// Use a pipe: Execute writes to stdin (captured), reads from stdout (we control).
	stdinCapture := &markerCapturingWriter{}
	stdoutR, stdoutW := io.Pipe()

	s := &Shell{
		stdin:  stdinCapture,
		stdout: bufio.NewScanner(stdoutR),
		cmd:    &fakeCommander{},
	}

	errCh := make(chan error, 1)
	go func() {
		_, _, _, err := s.Execute("echo hello")
		errCh <- err
	}()

	// Wait a bit for Execute to write the script to stdin, then extract the marker
	// and write a response with invalid exit code to stdout.
	// Since stdinCapture is synchronous, by the time Execute calls stdout.Scan(),
	// the script is already written.
	// We need to give Execute a moment to write, then provide the response.
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

func TestExecuteStreamInvalidExitCode(t *testing.T) {
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
