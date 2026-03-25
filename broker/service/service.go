// Package service はブローカーのビジネスロジックを提供する。
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/ogadra/20260327-cli-demo/broker/healthcheck"
	"github.com/ogadra/20260327-cli-demo/broker/model"
	"github.com/ogadra/20260327-cli-demo/broker/store"
)

// maxAcquireRetries はヘルスチェック失敗時の AcquireIdle リトライ回数上限。
const maxAcquireRetries = 3

// randReader はランダムバイト生成に使う io.Reader。テスト時に差し替える。
var randReader io.Reader = rand.Reader

// logPrintf はログ出力関数。テスト時に差し替える。
var logPrintf = log.Printf

// Service はブローカーのビジネスロジックを定義するインターフェース。
type Service interface {
	// CloseSession はセッションを終了し紐づく runner を削除する。
	CloseSession(ctx context.Context, sessionID string) error
	// ResolveSession はセッション ID から runner を解決し、見つからなければ新規作成する。
	ResolveSession(ctx context.Context, sessionID string) (*ResolveResult, error)
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
	// Reassigned はセッションが再割当てされたかどうかを示す。
	// dead runner 検出により既存セッションが新しい runner に再割当てされた場合に true。
	Reassigned bool
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
	checker   healthcheck.Checker
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

// WithChecker はヘルスチェッカーを差し替えるオプション。
func WithChecker(c healthcheck.Checker) Option {
	return func(s *BrokerService) {
		s.checker = c
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

// createSession は idle runner を確保しヘルスチェックを行いセッションを作成する。
// 不健全な runner は削除して最大 maxAcquireRetries 回リトライする。
func (s *BrokerService) createSession(ctx context.Context) (*CreateSessionResult, error) {
	for range maxAcquireRetries {
		sessionID, err := s.sessionFn()
		if err != nil {
			return nil, err
		}
		runner, err := s.repo.AcquireIdle(ctx, sessionID)
		if err != nil {
			return nil, err
		}
		if s.checker == nil {
			return &CreateSessionResult{SessionID: sessionID, Runner: runner}, nil
		}
		if err := s.checker.Check(ctx, runner.PrivateURL); err == nil {
			return &CreateSessionResult{SessionID: sessionID, Runner: runner}, nil
		}
		logPrintf("healthcheck failed for runner %s, deleting stale record", runner.RunnerID)
		_ = s.repo.Delete(ctx, runner.RunnerID)
	}
	return nil, store.ErrNoIdleRunner
}

// CloseSession はセッションを終了し紐づく runner を削除する。
func (s *BrokerService) CloseSession(ctx context.Context, sessionID string) error {
	runner, err := s.repo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, runner.RunnerID)
}

// ResolveSession はセッション ID から runner を解決し、見つからなければ新規作成する。
// sessionID が空の場合は検索をスキップして即座に新規作成する。
// 既存 runner が不健全な場合は削除して新規 runner を割り当てる。
func (s *BrokerService) ResolveSession(ctx context.Context, sessionID string) (*ResolveResult, error) {
	reassigned := false
	if sessionID != "" {
		runner, err := s.repo.FindBySessionID(ctx, sessionID)
		if err == nil {
			if s.checker == nil {
				return &ResolveResult{SessionID: sessionID, RunnerURL: runner.PrivateURL, Created: false}, nil
			}
			if checkErr := s.checker.Check(ctx, runner.PrivateURL); checkErr == nil {
				return &ResolveResult{SessionID: sessionID, RunnerURL: runner.PrivateURL, Created: false}, nil
			}
			logPrintf("healthcheck failed for runner %s, deleting stale record", runner.RunnerID)
			_ = s.repo.Delete(ctx, runner.RunnerID)
			reassigned = true
		} else if !errors.Is(err, store.ErrNotFound) {
			return nil, err
		}
	}
	result, err := s.createSession(ctx)
	if err != nil {
		return nil, err
	}
	return &ResolveResult{
		SessionID:  result.SessionID,
		RunnerURL:  result.Runner.PrivateURL,
		Created:    true,
		Reassigned: reassigned,
	}, nil
}

// RegisterRunner は runner を idle として登録する。
func (s *BrokerService) RegisterRunner(ctx context.Context, runnerID, privateURL string) error {
	return s.repo.Register(ctx, runnerID, privateURL)
}

// DeregisterRunner は runner を削除する。
func (s *BrokerService) DeregisterRunner(ctx context.Context, runnerID string) error {
	return s.repo.Delete(ctx, runnerID)
}
