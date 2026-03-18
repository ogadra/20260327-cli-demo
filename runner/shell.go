package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// commander abstracts process lifecycle for testing.
type commander interface {
	Start() error
	Wait() error
	StdinPipe() (io.WriteCloser, error)
	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)
}

// execCommander wraps exec.Cmd to implement commander.
type execCommander struct {
	cmd *exec.Cmd
}

func (c *execCommander) Start() error                       { return c.cmd.Start() }
func (c *execCommander) Wait() error                        { return c.cmd.Wait() }
func (c *execCommander) StdinPipe() (io.WriteCloser, error) { return c.cmd.StdinPipe() }
func (c *execCommander) StdoutPipe() (io.ReadCloser, error) { return c.cmd.StdoutPipe() }
func (c *execCommander) StderrPipe() (io.ReadCloser, error) { return c.cmd.StderrPipe() }

// Shell manages a single persistent bash session.
type Shell struct {
	cmd       commander
	stdin     io.WriteCloser
	stdout    *bufio.Scanner
	stderrBuf bytes.Buffer
	stderrMu  sync.Mutex
	mu        sync.Mutex
}

// newShellFromCommander creates a Shell from the given commander.
func newShellFromCommander(cmd commander) (*Shell, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start bash: %w", err)
	}

	s := &Shell{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewScanner(stdoutPipe),
	}

	go s.readStderr(stderrPipe)

	return s, nil
}

// NewShell starts a new persistent bash session.
func NewShell() (*Shell, error) {
	return newShellFromCommander(&execCommander{
		cmd: exec.Command("bash", "--norc", "--noprofile"),
	})
}

func (s *Shell) readStderr(r io.Reader) {
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			s.stderrMu.Lock()
			s.stderrBuf.Write(buf[:n])
			s.stderrMu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

func (s *Shell) resetStderr() {
	s.stderrMu.Lock()
	s.stderrBuf.Reset()
	s.stderrMu.Unlock()
}

func (s *Shell) getStderr() string {
	s.stderrMu.Lock()
	defer s.stderrMu.Unlock()
	return s.stderrBuf.String()
}

// Execute runs a command in the persistent shell and returns stdout, exit code, stderr, and any error.
func (s *Shell) Execute(command string) (string, int, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.resetStderr()

	marker := fmt.Sprintf("__MRK_%d_END__", time.Now().UnixNano())
	script := fmt.Sprintf("%s\n__ec=$?\necho '%s'${__ec}\n", command, marker)

	if _, err := io.WriteString(s.stdin, script); err != nil {
		return "", -1, "", fmt.Errorf("write command: %w", err)
	}

	var stdoutLines []string
	for s.stdout.Scan() {
		line := s.stdout.Text()
		if strings.HasPrefix(line, marker) {
			ecStr := line[len(marker):]
			exitCode, err := strconv.Atoi(ecStr)
			if err != nil {
				return "", -1, "", fmt.Errorf("parse exit code %q: %w", ecStr, err)
			}
			time.Sleep(50 * time.Millisecond)
			stderr := s.getStderr()
			stdout := strings.Join(stdoutLines, "\n")
			if len(stdoutLines) > 0 {
				stdout += "\n"
			}
			return stdout, exitCode, stderr, nil
		}
		stdoutLines = append(stdoutLines, line)
	}

	if err := s.stdout.Err(); err != nil {
		return "", -1, "", fmt.Errorf("scan stdout: %w", err)
	}
	return "", -1, "", fmt.Errorf("unexpected end of stdout")
}

// ExecuteStream runs a command and sends each stdout line to the provided channel.
// Returns exit code, stderr, and any error. The channel is closed when done.
func (s *Shell) ExecuteStream(command string, stdoutCh chan<- string) (int, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer close(stdoutCh)

	s.resetStderr()

	marker := fmt.Sprintf("__MRK_%d_END__", time.Now().UnixNano())
	script := fmt.Sprintf("%s\n__ec=$?\necho '%s'${__ec}\n", command, marker)

	if _, err := io.WriteString(s.stdin, script); err != nil {
		return -1, "", fmt.Errorf("write command: %w", err)
	}

	for s.stdout.Scan() {
		line := s.stdout.Text()
		if strings.HasPrefix(line, marker) {
			ecStr := line[len(marker):]
			exitCode, err := strconv.Atoi(ecStr)
			if err != nil {
				return -1, "", fmt.Errorf("parse exit code %q: %w", ecStr, err)
			}
			time.Sleep(50 * time.Millisecond)
			stderr := s.getStderr()
			return exitCode, stderr, nil
		}
		stdoutCh <- line
	}

	if err := s.stdout.Err(); err != nil {
		return -1, "", fmt.Errorf("scan stdout: %w", err)
	}
	return -1, "", fmt.Errorf("unexpected end of stdout")
}

// Close terminates the persistent shell session.
func (s *Shell) Close() error {
	_, err := io.WriteString(s.stdin, "exit\n")
	if err != nil {
		return fmt.Errorf("write exit: %w", err)
	}
	return s.cmd.Wait()
}
