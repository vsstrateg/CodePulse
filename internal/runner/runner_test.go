package runner

import (
	"codepulse/internal/domain"
	"context"
	"testing"
	"time"
)

func TestPing(t *testing.T) {
	r, err := NewDockerRunner()
	if err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			t.Fatalf("failed to close runner: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := r.Ping(ctx); err != nil {
		t.Fatalf("failed to ping docker daemon: %v", err)
	}
}

func TestRun(t *testing.T) {
	r, err := NewDockerRunner()
	if err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	defer func() {
		if err := r.Close(); err != nil {
			t.Fatalf("failed to close runner: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []domain.Test{
		{Input: "2 3",
			ExpectedOutput: "5"},
	}

	t.Run("StatusCompilationError", func(t *testing.T) {
		code := `package main
import "fmt"

func main() {
    var a, b int
    fmt.Scan(&a, &b)
    fmt.Pintln(a + b)
}`
		res, err := r.Run(ctx, code, tests, 1000, 128)
		if err != nil {
			t.Fatalf("system error: %v", err)
		}

		if res.Verdict != domain.StatusCompilationError {
			t.Errorf("expected COMPILATION_ERROR, got %v", res.Verdict)
		}
		t.Logf("Compiler error output:\n%s", res.ErrorMessage)
	})

	t.Run("StatusAccepted", func(t *testing.T) {
		code := `package main
import "fmt"

func main() {
    var a, b int
    fmt.Scan(&a, &b)
    fmt.Println(a + b)
}`

		res, err := r.Run(ctx, code, tests, 1000, 128)
		if err != nil {
			t.Fatalf("system error: %v", err)
		}

		if res.Verdict != domain.StatusAccepted {
			t.Errorf("expected ACCEPTED, got %v", res.Verdict)
		}

		t.Logf("Compiler error output:\n%s", res.ErrorMessage)
	})

	t.Run("StatusWrongAnswer", func(t *testing.T) {
		code := `package main
import "fmt"

func main() {
    var a, b int
    fmt.Scan(&a, &b)
    fmt.Println(a * b)
}`
		res, err := r.Run(ctx, code, tests, 1000, 128)
		if err != nil {
			t.Fatalf("system error: %v", err)
		}

		if res.Verdict != domain.StatusWrongAnswer {
			t.Errorf("expected WRONG_ANSWER, got %v", res.Verdict)
		}
		t.Logf("Compiler error output:\n%s", res.ErrorMessage)
	})

	t.Run("StatusTimeLimitExceeded", func(t *testing.T) {
		code := `package main

func main() {
	for {}
}`
		res, err := r.Run(ctx, code, tests, 500, 128)
		if err != nil {
			t.Fatalf("system error: %v", err)
		}

		if res.Verdict != domain.StatusTimeLimitExceeded {
			t.Errorf("expected TIME_LIMIT_EXCEEDED, got %v", res.Verdict)
		}
		t.Logf("Compiler error output:\n%s", res.ErrorMessage)
	})

	t.Run("StatusRuntimeError", func(t *testing.T) {
		code := `package main

func main() {
	panic("panic")
}`
		res, err := r.Run(ctx, code, tests, 1000, 128)
		if err != nil {
			t.Fatalf("system error: %v", err)
		}

		if res.Verdict != domain.StatusRuntimeError {
			t.Errorf("expected RUNTIME_ERROR, got %v", res.Verdict)
		}
		t.Logf("Compiler error output:\n%s", res.ErrorMessage)
	})

	t.Run("StatusMemoryLimitExceeded", func(t *testing.T) {
		code := `package main
func main() {
    var data [][]byte
    for {
        data = append(data, make([]byte, 10*1024*1024)) // allocate 10 MB
    }
}`
		res, err := r.Run(ctx, code, tests, 1000, 128)
		if err != nil {
			t.Fatalf("system error: %v", err)
		}

		if res.Verdict != domain.StatusMemoryLimitExceeded {
			t.Errorf("expected MEMORY_LIMIT_EXCEEDED, got %v", res.Verdict)
		}
		t.Logf("Compiler error output:\n%s", res.ErrorMessage)
	})
}
