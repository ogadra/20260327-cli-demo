package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Shell abstracts command execution and lifecycle for handler and session manager.
// In production, *bashShell implements this interface.
type Shell interface {
	ExecuteStream(ctx context.Context, command string, stdoutCh chan<- string) (int, string, error)
	Close() error
}

// sessionIDCookie is the cookie name used to pass the session ID.
const sessionIDCookie = "session_id"

// errMissingSessionCookie is the error message returned when the session_id cookie is absent.
const errMissingSessionCookie = "missing session_id cookie"

// executeRequest is the JSON body for POST /api/execute.
type executeRequest struct {
	Command string `json:"command" binding:"required"`
}

// errorResponse is the JSON body returned for error responses.
type errorResponse struct {
	Error string `json:"error"`
}

// sseEvent represents a single Server-Sent Event sent during command execution.
type sseEvent struct {
	Type     string `json:"type"`
	Data     string `json:"data,omitempty"`
	ExitCode *int   `json:"exitCode,omitempty"`
}

// newHandler creates a gin.Engine with all API routes registered.
// The returned engine handles GET /health, POST /api/session, DELETE /api/session, and POST /api/execute.
// The Validator v is used to judge non-whitelisted commands via LLM; it may be nil
// in which case all non-whitelisted commands are rejected with 403.
func newHandler(sm *SessionManager, v Validator) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"})
	r.HandleMethodNotAllowed = true
	r.GET("/health", handleHealth())
	r.POST("/api/session", handleCreateSession(sm))
	r.DELETE("/api/session", handleDeleteSession(sm))
	r.POST("/api/execute", handleExecute(sm, v))
	return r
}

// handleHealth returns a gin handler for GET /health.
// It returns 200 OK with body "ok\n" to indicate the runner is reachable.
func handleHealth() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(http.StatusOK, "ok\n")
	}
}

// handleCreateSession returns a gin handler for POST /api/session.
// It creates a new session and sets the session_id cookie.
func handleCreateSession(sm *SessionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _, err := sm.Create()
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
			return
		}
		c.SetSameSite(http.SameSiteStrictMode)
		c.SetCookie(sessionIDCookie, id, 0, "/", "", true, true)
		c.Status(http.StatusNoContent)
	}
}

// handleDeleteSession returns a gin handler for DELETE /api/session.
// It deletes the session specified by session_id cookie.
func handleDeleteSession(sm *SessionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := c.Cookie(sessionIDCookie)
		if err != nil || id == "" {
			c.JSON(http.StatusBadRequest, errorResponse{Error: errMissingSessionCookie})
			return
		}
		if err := sm.Delete(id); err != nil {
			if errors.Is(err, ErrSessionNotFound) {
				c.JSON(http.StatusNotFound, errorResponse{Error: err.Error()})
			} else {
				c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
			}
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// handleExecute returns a gin handler for POST /api/execute.
// It classifies the command: whitelisted commands execute immediately,
// validated commands are checked by the Validator before execution,
// and commands that fail validation are rejected with 403.
func handleExecute(sm *SessionManager, v Validator) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := c.Cookie(sessionIDCookie)
		if err != nil || id == "" {
			c.JSON(http.StatusBadRequest, errorResponse{Error: errMissingSessionCookie})
			return
		}

		// Get only returns ErrSessionNotFound; no other error paths exist.
		shell, err := sm.Get(id)
		if err != nil {
			c.JSON(http.StatusNotFound, errorResponse{Error: err.Error()})
			return
		}

		var req executeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Error: fmt.Sprintf("invalid request: %s", err.Error())})
			return
		}

		class := classifyCommand(req.Command)
		remote := clientIP(c)
		auditLog(id, remote, class, req.Command, nil, nil)

		if class == "validated" {
			if v == nil {
				c.JSON(http.StatusForbidden, errorResponse{Error: "command not allowed"})
				return
			}
			result, err := v.Validate(c.Request.Context(), req.Command)
			if err != nil {
				var unavail *ValidationUnavailableError
				if errors.As(err, &unavail) {
					auditLog(id, remote, "validation-skipped", req.Command, nil, err)
				} else {
					auditLog(id, remote, class, req.Command, nil, err)
					c.JSON(http.StatusForbidden, errorResponse{Error: "command not allowed"})
					return
				}
			} else if !result.Safe {
				auditLog(id, remote, "rejected", req.Command, nil, fmt.Errorf("reason: %s", result.Reason))
				c.JSON(http.StatusForbidden, errorResponse{Error: fmt.Sprintf("command not allowed: %s", result.Reason)})
				return
			}
		}

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		ch := make(chan string, 100)
		done := make(chan struct{})
		go func() {
			defer close(done)
			for line := range ch {
				writeSSE(c.Writer, sseEvent{Type: "stdout", Data: line})
				c.Writer.Flush()
			}
		}()

		exitCode, stderr, execErr := shell.ExecuteStream(c.Request.Context(), req.Command, ch)
		<-done

		if stderr != "" {
			writeSSE(c.Writer, sseEvent{Type: "stderr", Data: stderr})
			c.Writer.Flush()
		}

		writeSSE(c.Writer, sseEvent{Type: "complete", ExitCode: &exitCode})
		c.Writer.Flush()

		auditLog(id, remote, class, req.Command, &exitCode, execErr)
	}
}

// auditLog writes a structured audit log line.
// exitCode and err are optional and only appended when non-nil.
func auditLog(session, remote, class, command string, exitCode *int, err error) {
	msg := fmt.Sprintf("[AUDIT] session=%s remote=%s class=%s command=%q", session, remote, class, command)
	if exitCode != nil {
		msg += fmt.Sprintf(" exitCode=%d", *exitCode)
	}
	if err != nil {
		msg += fmt.Sprintf(" error=%v", err)
	}
	log.Print(msg)
}

// writeSSE marshals an sseEvent to JSON and writes it as a Server-Sent Event line.
// sseEvent contains only string and *int fields, so json.Marshal cannot fail.
func writeSSE(w http.ResponseWriter, event sseEvent) {
	data, _ := json.Marshal(event)
	fmt.Fprintf(w, "data: %s\n\n", data)
}

// clientIP extracts the client address from the request.
// It prefers the CloudFront-Viewer-Address header set by CloudFront,
// which contains the viewer IP and source port in ip:port format.
// The source port is preserved because it helps identify clients
// behind MAP-E or DS-Lite where multiple households share a single
// global IP and are distinguished by port ranges.
// If the header is absent, it falls back to Gin's c.ClientIP.
func clientIP(c *gin.Context) string {
	if addr := c.GetHeader("CloudFront-Viewer-Address"); addr != "" {
		return addr
	}
	return c.ClientIP()
}
