package main

import (
	"fmt"
	"testing"
)

// TestMain_success は main 関数が正常に完了することを検証する。
func TestMain_success(t *testing.T) {
	t.Parallel()

	origFatalf := fatalf
	origStart := startLambda
	defer func() {
		fatalf = origFatalf
		startLambda = origStart
	}()

	fatalf = func(format string, args ...any) {
		t.Fatalf("unexpected fatalf: "+format, args...)
	}
	startLambda = func(handler any) {}

	main()
}

// TestMain_error は run がエラーを返した場合に fatalf が呼ばれることを検証する。
func TestMain_error(t *testing.T) {
	t.Parallel()

	origFatalf := fatalf
	origRun := runFn
	defer func() {
		fatalf = origFatalf
		runFn = origRun
	}()

	var called bool
	fatalf = func(format string, args ...any) {
		called = true
	}
	runFn = func() error {
		return fmt.Errorf("test error")
	}

	main()

	if !called {
		t.Fatal("fatalf was not called")
	}
}

// TestRun は run が startLambda を呼び出しエラーなく完了することを検証する。
func TestRun(t *testing.T) {
	t.Parallel()

	origStart := startLambda
	defer func() { startLambda = origStart }()

	startLambda = func(handler any) {}

	if err := run(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestHandler はプレースホルダーハンドラーがエラーを返さないことを検証する。
func TestHandler(t *testing.T) {
	t.Parallel()

	if err := handler(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
