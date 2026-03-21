// Package handler は broker の HTTP ハンドラーを提供する。
package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ogadra/20260327-cli-demo/broker/model"
	"github.com/ogadra/20260327-cli-demo/broker/service"
	"github.com/ogadra/20260327-cli-demo/broker/store"
)

// Handler は broker の HTTP ハンドラー。
type Handler struct {
	svc service.Service
}

// NewHandler は Handler を生成する。svc が nil の場合は panic する。
func NewHandler(svc service.Service) *Handler {
	if svc == nil {
		panic("handler: nil service")
	}
	return &Handler{svc: svc}
}

// registerRequest は POST /internal/runners/register のリクエストボディ。
type registerRequest struct {
	// RunnerID は runner の一意識別子。
	RunnerID string `json:"runnerId" binding:"required"`
	// PrivateURL は runner のプライベート URL。
	PrivateURL string `json:"privateUrl" binding:"required"`
}

// PostSessions は POST /sessions を処理しセッションを作成する。
func (h *Handler) PostSessions(c *gin.Context) {
	result, err := h.svc.CreateSession(c.Request.Context())
	if err != nil {
		if errors.Is(err, store.ErrNoIdleRunner) {
			writeError(c, http.StatusServiceUnavailable, model.CodeNoIdleRunner, "no idle runner available")
			return
		}
		writeError(c, http.StatusInternalServerError, model.CodeInternalError, "failed to create session")
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("runner_id", result.SessionID, 0, "/", "", true, true)
	c.JSON(http.StatusCreated, gin.H{"sessionId": result.SessionID})
}

// DeleteSession は DELETE /sessions/:sessionId を処理しセッションを終了する。
func (h *Handler) DeleteSession(c *gin.Context) {
	sessionID := c.Param("sessionId")
	err := h.svc.CloseSession(c.Request.Context(), sessionID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(c, http.StatusNotFound, model.CodeSessionNotFound, "session not found")
			return
		}
		writeError(c, http.StatusInternalServerError, model.CodeInternalError, "failed to close session")
		return
	}
	c.Status(http.StatusNoContent)
}

// GetResolve は GET /resolve を処理し runner_id cookie からセッションを解決する。
func (h *Handler) GetResolve(c *gin.Context) {
	sessionID, err := c.Cookie("runner_id")
	if err != nil {
		writeError(c, http.StatusUnauthorized, model.CodeInvalidRequest, "runner_id cookie is required")
		return
	}
	url, err := h.svc.ResolveSession(c.Request.Context(), sessionID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(c, http.StatusNotFound, model.CodeSessionNotFound, "session not found")
			return
		}
		writeError(c, http.StatusInternalServerError, model.CodeInternalError, "failed to resolve session")
		return
	}
	c.Header("X-Runner-Url", url)
	c.Status(http.StatusOK)
}

// PostRegister は POST /internal/runners/register を処理し runner を登録する。
func (h *Handler) PostRegister(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, model.CodeInvalidRequest, "invalid request body")
		return
	}
	err := h.svc.RegisterRunner(c.Request.Context(), req.RunnerID, req.PrivateURL)
	if err != nil {
		writeError(c, http.StatusInternalServerError, model.CodeInternalError, "failed to register runner")
		return
	}
	c.Status(http.StatusCreated)
}

// DeleteRunner は DELETE /internal/runners/:runnerId を処理し runner を削除する。
func (h *Handler) DeleteRunner(c *gin.Context) {
	runnerID := c.Param("runnerId")
	err := h.svc.DeregisterRunner(c.Request.Context(), runnerID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, model.CodeInternalError, "failed to deregister runner")
		return
	}
	c.Status(http.StatusNoContent)
}

// writeError はエラーレスポンスを JSON で返す。
func writeError(c *gin.Context, status int, code, message string) {
	reqID, _ := c.Get(requestIDKey)
	rid, _ := reqID.(string)
	c.JSON(status, model.ErrorResponse{Code: code, Message: message, RequestID: rid})
}
