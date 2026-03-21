package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// fatalf is the function called on fatal errors. It defaults to log.Fatalf
// and can be replaced in tests to avoid os.Exit.
var fatalf = log.Fatalf

// main starts the HTTP server on :3000 with graceful shutdown on SIGTERM/SIGINT.
// The empty host binds to all interfaces, which is intentional for use inside a Docker container.
func main() {
	if err := start(":3000"); err != nil {
		fatalf("server error: %v", err)
	}
}

// start binds to the given address, registers signal handlers, and runs the
// server until a termination signal is received. It returns any error from
// the server lifecycle.
func start(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(sig)

	cfg := serverConfig{
		sm:              NewSessionManager(),
		shutdownTimeout: 10 * time.Second,
	}

	return run(ln, sig, cfg)
}

// serverConfig holds all dependencies needed to start and shut down the server.
// Fields default to production values when created via main.
type serverConfig struct {
	sm              *SessionManager
	validator       Validator
	shutdownTimeout time.Duration
}

// run starts the HTTP server on the given listener and blocks until a signal is
// received on sigCh, then performs graceful shutdown.
// Separating this from main allows tests to inject a signal channel and listener.
func run(ln net.Listener, sigCh <-chan os.Signal, cfg serverConfig) error {
	srv := &http.Server{
		Handler: newHandler(cfg.sm, cfg.validator),
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Printf("server listening on %s", ln.Addr())
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve: %w", err)
	case <-sigCh:
	}

	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.shutdownTimeout)
	defer cancel()

	var firstErr error
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v", err)
		firstErr = err
	}

	if err := cfg.sm.CloseAll(); err != nil {
		log.Printf("session cleanup error: %v", err)
		if firstErr == nil {
			firstErr = err
		}
	}

	log.Println("shutdown complete")
	return firstErr
}
