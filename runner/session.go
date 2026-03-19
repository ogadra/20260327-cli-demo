package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
)

// SessionManager manages multiple persistent bash sessions keyed by session ID.
// It is safe for concurrent use.
type SessionManager struct {
	mu       sync.Mutex
	sessions map[string]executor
	genID    func() (string, error)   // ID generator; defaults to generateID
	newShell func() (executor, error) // shell factory; defaults to newDefaultShell
}

// newDefaultShell wraps NewShell to satisfy the executor factory signature.
func newDefaultShell() (executor, error) {
	return NewShell()
}

// NewSessionManager creates an empty SessionManager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]executor),
		genID:    generateID,
		newShell: newDefaultShell,
	}
}

// generateID returns a cryptographically random 16-byte hex string of 32 characters.
// crypto/rand.Read always returns len(b) and a nil error on supported platforms,
// so the error return exists only to satisfy the genID function signature for testability.
func generateID() (string, error) {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b), nil
}

// Create starts a new bash session and returns its ID and the executor.
func (m *SessionManager) Create() (string, executor, error) {
	id, err := m.genID()
	if err != nil {
		return "", nil, err
	}

	shell, err := m.newShell()
	if err != nil {
		return "", nil, fmt.Errorf("create session: %w", err)
	}

	m.mu.Lock()
	if _, exists := m.sessions[id]; exists {
		m.mu.Unlock()
		_ = shell.Close()
		return "", nil, fmt.Errorf("create session: duplicate session id %q", id)
	}
	m.sessions[id] = shell
	m.mu.Unlock()

	return id, shell, nil
}

// Get returns the executor for the given session ID.
// Returns an error if the session does not exist.
func (m *SessionManager) Get(id string) (executor, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	shell, ok := m.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	return shell, nil
}

// Delete closes and removes the session with the given ID.
// Returns an error if the session does not exist.
func (m *SessionManager) Delete(id string) error {
	m.mu.Lock()
	shell, ok := m.sessions[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("session not found: %s", id)
	}
	delete(m.sessions, id)
	m.mu.Unlock()

	return shell.Close()
}

// CloseAll closes all sessions and clears the map.
// Returns the first error encountered, but attempts to close all sessions.
func (m *SessionManager) CloseAll() error {
	m.mu.Lock()
	sessions := m.sessions
	m.sessions = make(map[string]executor)
	m.mu.Unlock()

	var firstErr error
	for _, shell := range sessions {
		if err := shell.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
