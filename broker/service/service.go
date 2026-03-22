// Package service はブローカーのビジネスロジックを提供する。
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
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
	// ResolveOrCreateSession はセッション ID から runner を解決し、見つからなければ新規作成する。
	ResolveOrCreateSession(ctx context.Context, sessionID string) (*ResolveResult, error)
	// RegisterRunner は runner を idle として登録する。
	RegisterRunner(ctx context.Context, runnerID, privateURL string) error
	// DeregisterRunner は runner を削除する。
	DeregisterRunner(ctx context.Context, runnerID string) error
}

// ResolveResult はセッション解決または作成の結果を表す。
type ResolveResult struct {
	// SessionID はセッション ID。新規作成時は新しい ID、既存時は入力と同じ値。
	SessionID string
	// RunnerURL は runner のプライベート URL。
	RunnerURL string
	// Created は新規作成されたかどうかを示す。
	Created bool
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

// ResolveOrCreateSession はセッション ID から runner を解決し、見つからなければ新規作成する。
// sessionID が空の場合は検索をスキップして即座に新規作成する。
func (s *BrokerService) ResolveOrCreateSession(ctx context.Context, sessionID string) (*ResolveResult, error) {
	if sessionID != "" {
		runner, err := s.repo.FindBySessionID(ctx, sessionID)
		if err == nil {
			return &ResolveResult{SessionID: sessionID, RunnerURL: runner.PrivateURL, Created: false}, nil
		}
		if !errors.Is(err, store.ErrNotFound) {
			return nil, err
		}
	}
	result, err := s.CreateSession(ctx)
	if err != nil {
		return nil, err
	}
	return &ResolveResult{SessionID: result.SessionID, RunnerURL: result.Runner.PrivateURL, Created: true}, nil
}

// RegisterRunner は runner を idle として登録する。
func (s *BrokerService) RegisterRunner(ctx context.Context, runnerID, privateURL string) error {
	return s.repo.Register(ctx, runnerID, privateURL)
}

// DeregisterRunner は runner を削除する。
func (s *BrokerService) DeregisterRunner(ctx context.Context, runnerID string) error {
	return s.repo.Delete(ctx, runnerID)
}
