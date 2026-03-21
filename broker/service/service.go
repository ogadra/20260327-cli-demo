// Package service はブローカーのビジネスロジックを提供する。
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/ogadra/20260327-cli-demo/broker/model"
	"github.com/ogadra/20260327-cli-demo/broker/store"
)

// randReader はランダムバイト生成に使う io.Reader。テスト時に差し替える。
var randReader io.Reader = rand.Reader

// Service はブローカーのビジネスロジックを定義するインターフェース。
type Service interface {
	// CreateSession は idle runner を確保しセッションを作成する。
	CreateSession(ctx context.Context) (*CreateSessionResult, error)
	// CloseSession はセッションを終了し紐づく runner を削除する。
	CloseSession(ctx context.Context, sessionID string) error
	// ResolveSession はセッション ID から runner のプライベート URL を返す。
	ResolveSession(ctx context.Context, sessionID string) (string, error)
	// RegisterRunner は runner を idle として登録する。
	RegisterRunner(ctx context.Context, runnerID, privateURL string) error
	// DeregisterRunner は runner を削除する。
	DeregisterRunner(ctx context.Context, runnerID string) error
}

// CreateSessionResult はセッション作成の結果を表す。
type CreateSessionResult struct {
	// SessionID は作成されたセッション ID。
	SessionID string
	// Runner は確保された runner。
	Runner *model.Runner
}

// BrokerService は Service の実装。
type BrokerService struct {
	repo      store.Repository
	sessionFn func() (string, error)
}

// Option は BrokerService のオプション関数。
type Option func(*BrokerService)

// WithSessionFn はセッション ID 生成関数を差し替えるオプション。
func WithSessionFn(fn func() (string, error)) Option {
	return func(s *BrokerService) {
		if fn != nil {
			s.sessionFn = fn
		}
	}
}

// NewBrokerService は BrokerService を生成する。
func NewBrokerService(repo store.Repository, opts ...Option) *BrokerService {
	s := &BrokerService{
		repo:      repo,
		sessionFn: defaultSessionFn,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// defaultSessionFn は crypto/rand で 16 バイトのランダム値を生成し hex 32 文字の文字列を返す。
func defaultSessionFn() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(randReader, b); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// CreateSession は idle runner を確保しセッションを作成する。
func (s *BrokerService) CreateSession(ctx context.Context) (*CreateSessionResult, error) {
	sessionID, err := s.sessionFn()
	if err != nil {
		return nil, err
	}
	runner, err := s.repo.AcquireIdle(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return &CreateSessionResult{SessionID: sessionID, Runner: runner}, nil
}

// CloseSession はセッションを終了し紐づく runner を削除する。
func (s *BrokerService) CloseSession(ctx context.Context, sessionID string) error {
	runner, err := s.repo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, runner.RunnerID)
}

// ResolveSession はセッション ID から runner のプライベート URL を返す。
func (s *BrokerService) ResolveSession(ctx context.Context, sessionID string) (string, error) {
	runner, err := s.repo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return "", err
	}
	return runner.PrivateURL, nil
}

// RegisterRunner は runner を idle として登録する。
func (s *BrokerService) RegisterRunner(ctx context.Context, runnerID, privateURL string) error {
	return s.repo.Register(ctx, runnerID, privateURL)
}

// DeregisterRunner は runner を削除する。
func (s *BrokerService) DeregisterRunner(ctx context.Context, runnerID string) error {
	return s.repo.Delete(ctx, runnerID)
}
