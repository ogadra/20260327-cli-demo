package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// TestCreateAndGet verifies that Create returns a valid session ID and shell,
// and that Get retrieves the same shell instance by ID.
func TestCreateAndGet(t *testing.T) {
	m := NewSessionManager()
	defer m.CloseAll()

	id, shell, err := m.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if id == "" {
		t.Fatal("Create() returned empty id")
	}
	if shell == nil {
		t.Fatal("Create() returned nil shell")
	}

	got, err := m.Get(id)
	if err != nil {
		t.Fatalf("Get(%q) error: %v", id, err)
	}
	if got != shell {
		t.Fatal("Get() returned different shell instance")
	}
}

// TestGetNotFound verifies that Get returns ErrSessionNotFound for a nonexistent session ID.
func TestGetNotFound(t *testing.T) {
	m := NewSessionManager()

	_, err := m.Get("nonexistent")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Get(nonexistent) error = %v, want ErrSessionNotFound", err)
	}
}

// TestDelete verifies that Delete removes a session and that subsequent Get fails.
func TestDelete(t *testing.T) {
	m := NewSessionManager()

	id, _, err := m.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := m.Delete(id); err != nil {
		t.Fatalf("Delete(%q) error: %v", id, err)
	}

	_, err = m.Get(id)
	if err == nil {
		t.Fatal("Get() after Delete() should return error")
	}
}

// TestDeleteNotFound verifies that Delete returns ErrSessionNotFound for a nonexistent session ID.
func TestDeleteNotFound(t *testing.T) {
	m := NewSessionManager()

	err := m.Delete("nonexistent")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Delete(nonexistent) error = %v, want ErrSessionNotFound", err)
	}
}

// TestCloseAll verifies that CloseAll closes all sessions and clears the map.
func TestCloseAll(t *testing.T) {
	m := NewSessionManager()

	ids := make([]string, 3)
	for i := range ids {
		id, _, err := m.Create()
		if err != nil {
			t.Fatalf("Create() error: %v", err)
		}
		ids[i] = id
	}

	if err := m.CloseAll(); err != nil {
		t.Fatalf("CloseAll() error: %v", err)
	}

	for _, id := range ids {
		_, err := m.Get(id)
		if err == nil {
			t.Fatalf("Get(%q) after CloseAll() should return error", id)
		}
	}
}

// TestSessionExecute verifies that a command can be executed through a session-managed shell
// and that stdout and exit code are returned correctly.
func TestSessionExecute(t *testing.T) {
	m := NewSessionManager()
	defer m.CloseAll()

	_, shell, err := m.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

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

	exitCode, _, err := shell.ExecuteStream(ctx, "echo hello", ch)
	if err != nil {
		t.Fatalf("ExecuteStream error: %v", err)
	}
	<-done

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if len(lines) != 1 || lines[0] != "hello\n" {
		t.Fatalf("lines = %v, want [hello\\n]", lines)
	}
}

// TestSessionIsolation verifies that environment variables set in one session
// are not visible in another session.
func TestSessionIsolation(t *testing.T) {
	m := NewSessionManager()
	defer m.CloseAll()

	_, shell1, err := m.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	_, shell2, err := m.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Set variable in session 1
	ch1 := make(chan string, 100)
	done1 := make(chan struct{})
	go func() {
		defer close(done1)
		for range ch1 {
		}
	}()
	_, _, err = shell1.ExecuteStream(ctx, "export TESTVAR=session1", ch1)
	if err != nil {
		t.Fatalf("ExecuteStream error: %v", err)
	}
	<-done1

	// Session 2 should not see it
	ch2 := make(chan string, 100)
	var lines2 []string
	done2 := make(chan struct{})
	go func() {
		defer close(done2)
		for line := range ch2 {
			lines2 = append(lines2, line)
		}
	}()
	exitCode, _, err := shell2.ExecuteStream(ctx, "echo ${TESTVAR:-unset}", ch2)
	if err != nil {
		t.Fatalf("ExecuteStream error: %v", err)
	}
	<-done2

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if len(lines2) != 1 || lines2[0] != "unset\n" {
		t.Fatalf("session2 saw TESTVAR: lines = %v, want [unset\\n]", lines2)
	}
}

// TestConcurrentCreate verifies that concurrent Create calls produce unique session IDs
// without data races.
func TestConcurrentCreate(t *testing.T) {
	m := NewSessionManager()
	defer m.CloseAll()

	const n = 10
	ids := make([]string, n)
	errs := make([]error, n)

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			id, _, err := m.Create()
			ids[i] = id
			errs[i] = err
		}(i)
	}
	wg.Wait()

	seen := make(map[string]bool)
	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("Create() [%d] error: %v", i, errs[i])
		}
		if seen[ids[i]] {
			t.Fatalf("duplicate session id: %s", ids[i])
		}
		seen[ids[i]] = true
	}
}

// TestCreateIDGeneratorError verifies that Create propagates errors from the ID generator.
func TestCreateIDGeneratorError(t *testing.T) {
	m := NewSessionManager()
	m.genID = func() (string, error) {
		return "", errors.New("rand broken")
	}

	_, _, err := m.Create()
	if err == nil {
		t.Fatal("Create() should return error when genID fails")
	}
}

// TestCreateDuplicateID verifies that Create returns an error and closes the new shell
// when genID returns an ID that already exists in the session map.
func TestCreateDuplicateID(t *testing.T) {
	m := NewSessionManager()
	defer m.CloseAll()

	m.genID = func() (string, error) {
		return "fixed-id", nil
	}

	_, _, err := m.Create()
	if err != nil {
		t.Fatalf("first Create() error: %v", err)
	}

	_, _, err = m.Create()
	if err == nil {
		t.Fatal("second Create() with duplicate ID should return error")
	}

	// Original session should still be accessible.
	_, err = m.Get("fixed-id")
	if err != nil {
		t.Fatalf("Get() after duplicate Create() error: %v", err)
	}
}

// TestCreateNewShellError verifies that Create propagates errors from the shell factory.
func TestCreateNewShellError(t *testing.T) {
	m := NewSessionManager()
	m.newShell = func() (Shell, error) {
		return nil, errors.New("shell broken")
	}

	_, _, err := m.Create()
	if err == nil {
		t.Fatal("Create() should return error when newShell fails")
	}
}

// TestCloseAllWithError verifies that CloseAll returns an error when a session's Close fails,
// such as when the shell has already been closed.
func TestCloseAllWithError(t *testing.T) {
	m := NewSessionManager()

	// Create a real session so CloseAll has something to iterate
	id, _, err := m.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Close it manually so CloseAll will get an error from Close()
	shell, _ := m.Get(id)
	shell.Close()

	err = m.CloseAll()
	if err == nil {
		t.Fatal("CloseAll() should return error when a session Close fails")
	}
}

// TestDeleteThenCreate verifies that a new session can be created after deleting an existing one,
// and that the new session receives a different ID.
func TestDeleteThenCreate(t *testing.T) {
	m := NewSessionManager()
	defer m.CloseAll()

	id1, _, err := m.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := m.Delete(id1); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	id2, _, err := m.Create()
	if err != nil {
		t.Fatalf("Create() after Delete() error: %v", err)
	}

	if id1 == id2 {
		t.Fatal("new session should have different id")
	}

	_, err = m.Get(id2)
	if err != nil {
		t.Fatalf("Get(%q) error: %v", id2, err)
	}
}
