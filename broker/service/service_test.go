// Package service はブローカーのビジネスロジックのテストを提供する。
package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ogadra/20260327-cli-demo/broker/model"
	"github.com/ogadra/20260327-cli-demo/broker/store"
)

// errorReader は常にエラーを返す io.Reader。
type errorReader struct{}

// Read は常にエラーを返す。
func (e *errorReader) Read(_ []byte) (int, error) {
	return 0, errors.New("rand read error")
}

// mockRepository は store.Repository のモック実装。
type mockRepository struct {
	registerFn        func(ctx context.Context, runnerID, privateURL string) error
	acquireIdleFn     func(ctx context.Context, sessionID string) (*model.Runner, error)
	findBySessionIDFn func(ctx context.Context, sessionID string) (*model.Runner, error)
	findByIDFn        func(ctx context.Context, runnerID string) (*model.Runner, error)
	deleteFn          func(ctx context.Context, runnerID string) error
}

// Register はモック Register を呼び出す。
func (m *mockRepository) Register(ctx context.Context, runnerID, privateURL string) error {
	return m.registerFn(ctx, runnerID, privateURL)
}

// AcquireIdle はモック AcquireIdle を呼び出す。
func (m *mockRepository) AcquireIdle(ctx context.Context, sessionID string) (*model.Runner, error) {
	return m.acquireIdleFn(ctx, sessionID)
}

// FindBySessionID はモック FindBySessionID を呼び出す。
func (m *mockRepository) FindBySessionID(ctx context.Context, sessionID string) (*model.Runner, error) {
	return m.findBySessionIDFn(ctx, sessionID)
}

// FindByID はモック FindByID を呼び出す。
func (m *mockRepository) FindByID(ctx context.Context, runnerID string) (*model.Runner, error) {
	return m.findByIDFn(ctx, runnerID)
}

// Delete はモック Delete を呼び出す。
func (m *mockRepository) Delete(ctx context.Context, runnerID string) error {
	return m.deleteFn(ctx, runnerID)
}

// TestBrokerService_ImplementsService は BrokerService が Service インターフェースを満たすことを検証する。
func TestBrokerService_ImplementsService(t *testing.T) {
	t.Parallel()
	var _ Service = (*BrokerService)(nil)
}

// TestNewBrokerService はコンストラクタの動作を検証する。
func TestNewBrokerService(t *testing.T) {
	t.Parallel()
	repo := &mockRepository{}
	svc := NewBrokerService(repo)
	if svc.repo != repo {
		t.Error("repo mismatch")
	}
	if svc.sessionFn == nil {
		t.Error("sessionFn is nil")
	}
}

// TestNewBrokerService_WithSessionFn は WithSessionFn オプションで sessionFn が差し替わることを検証する。
func TestNewBrokerService_WithSessionFn(t *testing.T) {
	t.Parallel()
	called := false
	fn := func() (string, error) {
		called = true
		return "test-session", nil
	}
	svc := NewBrokerService(&mockRepository{}, WithSessionFn(fn))
	got, err := svc.sessionFn()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "test-session" {
		t.Errorf("sessionFn() = %q, want %q", got, "test-session")
	}
	if !called {
		t.Error("custom sessionFn was not called")
	}
}

// TestWithSessionFn_Nil は WithSessionFn に nil を渡してもデフォルト関数が維持されることを検証する。
func TestWithSessionFn_Nil(t *testing.T) {
	t.Parallel()
	svc := NewBrokerService(&mockRepository{}, WithSessionFn(nil))
	if svc.sessionFn == nil {
		t.Fatal("sessionFn should not be nil when WithSessionFn(nil) is passed")
	}
	id, err := svc.sessionFn()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(id) != 32 {
		t.Errorf("len(id) = %d, want 32", len(id))
	}
}

// TestDefaultSessionFn はデフォルトセッション ID 生成関数が 32 文字の hex 文字列を返すことを検証する。
func TestDefaultSessionFn(t *testing.T) {
	t.Parallel()
	id, err := defaultSessionFn()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(id) != 32 {
		t.Errorf("len(id) = %d, want 32", len(id))
	}
}

// TestDefaultSessionFn_Unique はデフォルトセッション ID 生成関数が一意の値を返すことを検証する。
func TestDefaultSessionFn_Unique(t *testing.T) {
	t.Parallel()
	id1, err := defaultSessionFn()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	id2, err := defaultSessionFn()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id1 == id2 {
		t.Error("expected unique session IDs")
	}
}

// TestDefaultSessionFn_RandReadError は rand.Reader がエラーを返す場合に defaultSessionFn がエラーを返すことを検証する。
func TestDefaultSessionFn_RandReadError(t *testing.T) {
	orig := randReader
	t.Cleanup(func() { randReader = orig })
	randReader = &errorReader{}

	_, err := defaultSessionFn()
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestCreateSession_Success はセッション作成の成功ケースを検証する。
func TestCreateSession_Success(t *testing.T) {
	t.Parallel()
	repo := &mockRepository{
		acquireIdleFn: func(_ context.Context, sessionID string) (*model.Runner, error) {
			if sessionID != "fixed-session" {
				t.Errorf("sessionID = %q, want %q", sessionID, "fixed-session")
			}
			return &model.Runner{
				RunnerID:         "r1",
				CurrentSessionID: sessionID,
				PrivateURL:       "http://10.0.0.1:8080",
			}, nil
		},
	}
	svc := NewBrokerService(repo, WithSessionFn(func() (string, error) {
		return "fixed-session", nil
	}))

	result, err := svc.createSession(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SessionID != "fixed-session" {
		t.Errorf("SessionID = %q, want %q", result.SessionID, "fixed-session")
	}
	if result.Runner.RunnerID != "r1" {
		t.Errorf("RunnerID = %q, want %q", result.Runner.RunnerID, "r1")
	}
}

// TestCreateSession_SessionFnError はセッション ID 生成のエラーを検証する。
func TestCreateSession_SessionFnError(t *testing.T) {
	t.Parallel()
	svc := NewBrokerService(&mockRepository{}, WithSessionFn(func() (string, error) {
		return "", errors.New("rand error")
	}))

	_, err := svc.createSession(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestCreateSession_AcquireIdleError は AcquireIdle のエラーを検証する。
func TestCreateSession_AcquireIdleError(t *testing.T) {
	t.Parallel()
	repo := &mockRepository{
		acquireIdleFn: func(_ context.Context, _ string) (*model.Runner, error) {
			return nil, store.ErrNoIdleRunner
		},
	}
	svc := NewBrokerService(repo, WithSessionFn(func() (string, error) {
		return "sess-1", nil
	}))

	_, err := svc.createSession(context.Background())
	if !errors.Is(err, store.ErrNoIdleRunner) {
		t.Fatalf("expected ErrNoIdleRunner, got: %v", err)
	}
}

// TestCloseSession_Success はセッション終了の成功ケースを検証する。
func TestCloseSession_Success(t *testing.T) {
	t.Parallel()
	deleteCalled := false
	repo := &mockRepository{
		findBySessionIDFn: func(_ context.Context, sessionID string) (*model.Runner, error) {
			return &model.Runner{RunnerID: "r1", CurrentSessionID: sessionID}, nil
		},
		deleteFn: func(_ context.Context, runnerID string) error {
			deleteCalled = true
			if runnerID != "r1" {
				t.Errorf("runnerID = %q, want %q", runnerID, "r1")
			}
			return nil
		},
	}
	svc := NewBrokerService(repo)

	err := svc.CloseSession(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleteCalled {
		t.Error("Delete was not called")
	}
}

// TestCloseSession_FindError は FindBySessionID のエラーを検証する。
func TestCloseSession_FindError(t *testing.T) {
	t.Parallel()
	repo := &mockRepository{
		findBySessionIDFn: func(_ context.Context, _ string) (*model.Runner, error) {
			return nil, store.ErrNotFound
		},
	}
	svc := NewBrokerService(repo)

	err := svc.CloseSession(context.Background(), "sess-missing")
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

// TestCloseSession_DeleteError は Delete のエラーを検証する。
func TestCloseSession_DeleteError(t *testing.T) {
	t.Parallel()
	repo := &mockRepository{
		findBySessionIDFn: func(_ context.Context, _ string) (*model.Runner, error) {
			return &model.Runner{RunnerID: "r1"}, nil
		},
		deleteFn: func(_ context.Context, _ string) error {
			return errors.New("delete error")
		},
	}
	svc := NewBrokerService(repo)

	err := svc.CloseSession(context.Background(), "sess-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRegisterRunner_Success は runner 登録の成功ケースを検証する。
func TestRegisterRunner_Success(t *testing.T) {
	t.Parallel()
	repo := &mockRepository{
		registerFn: func(_ context.Context, runnerID, privateURL string) error {
			if runnerID != "r1" {
				t.Errorf("runnerID = %q, want %q", runnerID, "r1")
			}
			if privateURL != "http://10.0.0.1:8080" {
				t.Errorf("privateURL = %q, want %q", privateURL, "http://10.0.0.1:8080")
			}
			return nil
		},
	}
	svc := NewBrokerService(repo)

	err := svc.RegisterRunner(context.Background(), "r1", "http://10.0.0.1:8080")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRegisterRunner_Error は Register のエラーを検証する。
func TestRegisterRunner_Error(t *testing.T) {
	t.Parallel()
	repo := &mockRepository{
		registerFn: func(_ context.Context, _, _ string) error {
			return errors.New("register error")
		},
	}
	svc := NewBrokerService(repo)

	err := svc.RegisterRunner(context.Background(), "r1", "http://10.0.0.1:8080")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestDeregisterRunner_Success は runner 削除の成功ケースを検証する。
func TestDeregisterRunner_Success(t *testing.T) {
	t.Parallel()
	repo := &mockRepository{
		deleteFn: func(_ context.Context, runnerID string) error {
			if runnerID != "r1" {
				t.Errorf("runnerID = %q, want %q", runnerID, "r1")
			}
			return nil
		},
	}
	svc := NewBrokerService(repo)

	err := svc.DeregisterRunner(context.Background(), "r1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestDeregisterRunner_Error は Delete のエラーを検証する。
func TestDeregisterRunner_Error(t *testing.T) {
	t.Parallel()
	repo := &mockRepository{
		deleteFn: func(_ context.Context, _ string) error {
			return errors.New("delete error")
		},
	}
	svc := NewBrokerService(repo)

	err := svc.DeregisterRunner(context.Background(), "r1")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestResolveSession_ExistingSession は既存セッションの解決を検証する。
func TestResolveSession_ExistingSession(t *testing.T) {
	t.Parallel()
	repo := &mockRepository{
		findBySessionIDFn: func(_ context.Context, sessionID string) (*model.Runner, error) {
			return &model.Runner{RunnerID: "r1", PrivateURL: "http://10.0.0.1:8080"}, nil
		},
	}
	svc := NewBrokerService(repo)

	result, err := svc.ResolveSession(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Created {
		t.Error("expected Created=false for existing session")
	}
	if result.SessionID != "sess-1" {
		t.Errorf("SessionID = %q, want %q", result.SessionID, "sess-1")
	}
	if result.RunnerURL != "http://10.0.0.1:8080" {
		t.Errorf("RunnerURL = %q, want %q", result.RunnerURL, "http://10.0.0.1:8080")
	}
}

// TestResolveSession_NotFound_CreatesNew はセッションが見つからない場合に新規作成することを検証する。
func TestResolveSession_NotFound_CreatesNew(t *testing.T) {
	t.Parallel()
	repo := &mockRepository{
		findBySessionIDFn: func(_ context.Context, _ string) (*model.Runner, error) {
			return nil, store.ErrNotFound
		},
		acquireIdleFn: func(_ context.Context, sessionID string) (*model.Runner, error) {
			return &model.Runner{
				RunnerID:         "r2",
				CurrentSessionID: sessionID,
				PrivateURL:       "http://10.0.0.2:8080",
			}, nil
		},
	}
	svc := NewBrokerService(repo, WithSessionFn(func() (string, error) {
		return "new-session", nil
	}))

	result, err := svc.ResolveSession(context.Background(), "sess-missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Created {
		t.Error("expected Created=true for new session")
	}
	if result.SessionID != "new-session" {
		t.Errorf("SessionID = %q, want %q", result.SessionID, "new-session")
	}
	if result.RunnerURL != "http://10.0.0.2:8080" {
		t.Errorf("RunnerURL = %q, want %q", result.RunnerURL, "http://10.0.0.2:8080")
	}
}

// TestResolveSession_FindInternalError は FindBySessionID の内部エラーを検証する。
func TestResolveSession_FindInternalError(t *testing.T) {
	t.Parallel()
	repo := &mockRepository{
		findBySessionIDFn: func(_ context.Context, _ string) (*model.Runner, error) {
			return nil, errors.New("db error")
		},
	}
	svc := NewBrokerService(repo)

	_, err := svc.ResolveSession(context.Background(), "sess-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestResolveSession_CreateError は新規作成時のエラーを検証する。
func TestResolveSession_CreateError(t *testing.T) {
	t.Parallel()
	repo := &mockRepository{
		findBySessionIDFn: func(_ context.Context, _ string) (*model.Runner, error) {
			return nil, store.ErrNotFound
		},
		acquireIdleFn: func(_ context.Context, _ string) (*model.Runner, error) {
			return nil, store.ErrNoIdleRunner
		},
	}
	svc := NewBrokerService(repo, WithSessionFn(func() (string, error) {
		return "new-session", nil
	}))

	_, err := svc.ResolveSession(context.Background(), "sess-missing")
	if !errors.Is(err, store.ErrNoIdleRunner) {
		t.Fatalf("expected ErrNoIdleRunner, got: %v", err)
	}
}

// TestResolveSession_EmptySessionID は空のセッション ID で FindBySessionID をスキップして新規作成されることを検証する。
func TestResolveSession_EmptySessionID(t *testing.T) {
	t.Parallel()
	repo := &mockRepository{
		findBySessionIDFn: func(_ context.Context, _ string) (*model.Runner, error) {
			t.Fatal("FindBySessionID should not be called for empty session ID")
			return nil, store.ErrNotFound
		},
		acquireIdleFn: func(_ context.Context, sessionID string) (*model.Runner, error) {
			return &model.Runner{
				RunnerID:         "r1",
				CurrentSessionID: sessionID,
				PrivateURL:       "http://10.0.0.1:8080",
			}, nil
		},
	}
	svc := NewBrokerService(repo, WithSessionFn(func() (string, error) {
		return "new-session", nil
	}))

	result, err := svc.ResolveSession(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Created {
		t.Error("expected Created=true")
	}
}
