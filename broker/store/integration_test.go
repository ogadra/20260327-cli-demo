//go:build integration

// Package store は DynamoDB Local を使った統合テストを提供する。
package store

import (
	"context"
	"errors"
	"testing"
)

// TestIntegration_RegisterAndFindByID はレコード登録と ID 検索の統合テスト。
func TestIntegration_RegisterAndFindByID(t *testing.T) {
	t.Parallel()
	client, tableName := setupIntegrationTable(t)
	repo := NewDynamoRepository(client, tableName)
	repo.bucketFn = func() string { return "bucket-0" }

	ctx := context.Background()

	err := repo.Register(ctx, "r1")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	runner, err := repo.FindByID(ctx, "r1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if runner.RunnerID != "r1" {
		t.Errorf("runnerID = %q, want %q", runner.RunnerID, "r1")
	}
	if runner.IdleBucket != "bucket-0" {
		t.Errorf("idleBucket = %q, want %q", runner.IdleBucket, "bucket-0")
	}
}

// TestIntegration_RegisterIdempotent は登録の冪等性を検証する統合テスト。
func TestIntegration_RegisterIdempotent(t *testing.T) {
	t.Parallel()
	client, tableName := setupIntegrationTable(t)
	repo := NewDynamoRepository(client, tableName)
	repo.bucketFn = func() string { return "bucket-0" }

	ctx := context.Background()

	if err := repo.Register(ctx, "r1"); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	if err := repo.Register(ctx, "r1"); err != nil {
		t.Fatalf("second Register should be idempotent: %v", err)
	}
}

// TestIntegration_AcquireIdle は idle runner 確保の統合テスト。
func TestIntegration_AcquireIdle(t *testing.T) {
	t.Parallel()
	client, tableName := setupIntegrationTable(t)
	repo := NewDynamoRepository(client, tableName)
	repo.bucketFn = func() string { return "bucket-1" }

	ctx := context.Background()

	if err := repo.Register(ctx, "r1"); err != nil {
		t.Fatalf("Register: %v", err)
	}

	runner, err := repo.AcquireIdle(ctx, "sess-1")
	if err != nil {
		t.Fatalf("AcquireIdle: %v", err)
	}
	if runner.RunnerID != "r1" {
		t.Errorf("runnerID = %q, want %q", runner.RunnerID, "r1")
	}
	if runner.CurrentSessionID != "sess-1" {
		t.Errorf("currentSessionId = %q, want %q", runner.CurrentSessionID, "sess-1")
	}
}

// TestIntegration_AcquireIdle_Empty は runner がいない場合に ErrNoIdleRunner を返す統合テスト。
func TestIntegration_AcquireIdle_Empty(t *testing.T) {
	t.Parallel()
	client, tableName := setupIntegrationTable(t)
	repo := NewDynamoRepository(client, tableName)

	_, err := repo.AcquireIdle(context.Background(), "sess-1")
	if !errors.Is(err, ErrNoIdleRunner) {
		t.Fatalf("expected ErrNoIdleRunner, got: %v", err)
	}
}

// TestIntegration_AcquireIdle_FindBySessionID はセッション確保後にセッション検索できることを検証する統合テスト。
func TestIntegration_AcquireIdle_FindBySessionID(t *testing.T) {
	t.Parallel()
	client, tableName := setupIntegrationTable(t)
	repo := NewDynamoRepository(client, tableName)
	repo.bucketFn = func() string { return "bucket-0" }

	ctx := context.Background()

	if err := repo.Register(ctx, "r1"); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if _, err := repo.AcquireIdle(ctx, "sess-1"); err != nil {
		t.Fatalf("AcquireIdle: %v", err)
	}

	runner, err := repo.FindBySessionID(ctx, "sess-1")
	if err != nil {
		t.Fatalf("FindBySessionID: %v", err)
	}
	if runner.RunnerID != "r1" {
		t.Errorf("runnerID = %q, want %q", runner.RunnerID, "r1")
	}
}

// TestIntegration_AcquireIdle_AlreadyBusy は全 runner が busy の場合に ErrNoIdleRunner を返す統合テスト。
func TestIntegration_AcquireIdle_AlreadyBusy(t *testing.T) {
	t.Parallel()
	client, tableName := setupIntegrationTable(t)
	repo := NewDynamoRepository(client, tableName)
	repo.bucketFn = func() string { return "bucket-0" }

	ctx := context.Background()

	if err := repo.Register(ctx, "r1"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if _, err := repo.AcquireIdle(ctx, "sess-1"); err != nil {
		t.Fatalf("first AcquireIdle: %v", err)
	}

	_, err := repo.AcquireIdle(ctx, "sess-2")
	if !errors.Is(err, ErrNoIdleRunner) {
		t.Fatalf("expected ErrNoIdleRunner, got: %v", err)
	}
}

// TestIntegration_Delete は runner 削除の統合テスト。
func TestIntegration_Delete(t *testing.T) {
	t.Parallel()
	client, tableName := setupIntegrationTable(t)
	repo := NewDynamoRepository(client, tableName)
	repo.bucketFn = func() string { return "bucket-0" }

	ctx := context.Background()

	if err := repo.Register(ctx, "r1"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := repo.Delete(ctx, "r1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := repo.FindByID(ctx, "r1")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got: %v", err)
	}
}

// TestIntegration_Delete_Idempotent は存在しない runner の削除が冪等に成功する統合テスト。
func TestIntegration_Delete_Idempotent(t *testing.T) {
	t.Parallel()
	client, tableName := setupIntegrationTable(t)
	repo := NewDynamoRepository(client, tableName)

	err := repo.Delete(context.Background(), "r-nonexistent")
	if err != nil {
		t.Fatalf("expected nil for idempotent delete, got: %v", err)
	}
}

// TestIntegration_FindBySessionID_NotFound は存在しないセッションの検索で ErrNotFound を返す統合テスト。
func TestIntegration_FindBySessionID_NotFound(t *testing.T) {
	t.Parallel()
	client, tableName := setupIntegrationTable(t)
	repo := NewDynamoRepository(client, tableName)

	_, err := repo.FindBySessionID(context.Background(), "sess-missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

// TestIntegration_FindByID_NotFound は存在しない runner の検索で ErrNotFound を返す統合テスト。
func TestIntegration_FindByID_NotFound(t *testing.T) {
	t.Parallel()
	client, tableName := setupIntegrationTable(t)
	repo := NewDynamoRepository(client, tableName)

	_, err := repo.FindByID(context.Background(), "r-missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}
