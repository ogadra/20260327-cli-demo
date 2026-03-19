package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// sessionIDHeader is the HTTP header name used to pass the session ID.
const sessionIDHeader = "X-Session-Id"

// executeRequest is the JSON body for POST /api/execute.
type executeRequest struct {
	Command string `json:"command" binding:"required"`
}

// sessionResponse is the JSON body returned by POST /api/session.
type sessionResponse struct {
	SessionID string `json:"sessionId"`
}

// sseEvent represents a single Server-Sent Event sent during command execution.
type sseEvent struct {
	Type     string `json:"type"`
	Data     string `json:"data,omitempty"`
	ExitCode *int   `json:"exitCode,omitempty"`
}

// newHandler creates a gin.Engine with all API routes registered.
// The returned engine handles POST /api/session, DELETE /api/session, and POST /api/execute.
func newHandler(sm *SessionManager) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.HandleMethodNotAllowed = true
	r.POST("/api/session", handleCreateSession(sm))
	r.DELETE("/api/session", handleDeleteSession(sm))
	r.POST("/api/execute", handleExecute(sm))
	return r
}

// handleCreateSession returns a gin handler for POST /api/session.
// It creates a new session and returns the session ID as JSON.
func handleCreateSession(sm *SessionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _, err := sm.Create()
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, sessionResponse{SessionID: id})
	}
}

// handleDeleteSession returns a gin handler for DELETE /api/session.
// It deletes the session specified by X-Session-Id header.
func handleDeleteSession(sm *SessionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(sessionIDHeader)
		if id == "" {
			c.String(http.StatusBadRequest, "missing X-Session-Id header")
			return
		}
		if err := sm.Delete(id); err != nil {
			c.String(http.StatusNotFound, err.Error())
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// handleExecute returns a gin handler for POST /api/execute.
// It executes the command in the session specified by X-Session-Id and streams
// the result as Server-Sent Events.
func handleExecute(sm *SessionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(sessionIDHeader)
		if id == "" {
			c.String(http.StatusBadRequest, "missing X-Session-Id header")
			return
		}

		shell, err := sm.Get(id)
		if err != nil {
			c.String(http.StatusNotFound, err.Error())
			return
		}

		var req executeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.String(http.StatusBadRequest, "invalid request: %s", err.Error())
			return
		}

		class := classifyCommand(req.Command)

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

		if execErr != nil {
			log.Printf("[AUDIT] command=%q class=%s error=%v", req.Command, class, execErr)
		} else {
			log.Printf("[AUDIT] command=%q class=%s exitCode=%d", req.Command, class, exitCode)
		}
	}
}

// writeSSE marshals an sseEvent to JSON and writes it as a Server-Sent Event line.
// sseEvent contains only string and *int fields, so json.Marshal cannot fail.
func writeSSE(w http.ResponseWriter, event sseEvent) {
	data, _ := json.Marshal(event)
	fmt.Fprintf(w, "data: %s\n\n", data)
}
