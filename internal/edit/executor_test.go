package edit

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/koh-sh/apcdeploy/internal/prompt"
	promptTesting "github.com/koh-sh/apcdeploy/internal/prompt/testing"
	"github.com/koh-sh/apcdeploy/internal/reporter"
	reporterTesting "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

func TestExecutorValidatesWaitFlags(t *testing.T) {
	t.Parallel()

	rep := &reporterTesting.MockReporter{}
	prom := &promptTesting.MockPrompter{}
	executor := NewExecutor(rep, prom)

	err := executor.Execute(context.Background(), &Options{
		WaitDeploy: true,
		WaitBake:   true,
		Timeout:    300,
	})
	if err == nil {
		t.Fatal("expected error for mutually exclusive wait flags")
	}
	if !strings.Contains(err.Error(), "cannot be used together") {
		t.Errorf("expected wait-flag conflict error, got: %v", err)
	}
}

func TestExecutorValidatesNegativeTimeout(t *testing.T) {
	t.Parallel()

	executor := NewExecutor(&reporterTesting.MockReporter{}, &promptTesting.MockPrompter{})

	err := executor.Execute(context.Background(), &Options{Timeout: -1})
	if err == nil {
		t.Fatal("expected error for negative timeout")
	}
	if !strings.Contains(err.Error(), "timeout must be a non-negative value") {
		t.Errorf("expected timeout validation error, got: %v", err)
	}
}

func TestExecutorFactoryErrorWrapped(t *testing.T) {
	t.Parallel()

	factory := func(ctx context.Context, opts *Options, p prompt.Prompter, r reporter.Reporter) (*workflow, error) {
		return nil, errors.New("boom")
	}
	executor := NewExecutorWithFactory(&reporterTesting.MockReporter{}, &promptTesting.MockPrompter{}, factory)

	err := executor.Execute(context.Background(), &Options{Timeout: 300})
	if err == nil {
		t.Fatal("expected factory error")
	}
	if !strings.Contains(err.Error(), "failed to create edit workflow") {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

func TestExecutorTTYErrorPassesThrough(t *testing.T) {
	t.Parallel()

	factory := func(ctx context.Context, opts *Options, p prompt.Prompter, r reporter.Reporter) (*workflow, error) {
		return nil, fmt.Errorf("%w: please provide --region, --app, --profile, and --env flags", prompt.ErrNoTTY)
	}
	executor := NewExecutorWithFactory(&reporterTesting.MockReporter{}, &promptTesting.MockPrompter{}, factory)

	err := executor.Execute(context.Background(), &Options{Timeout: 300})
	if err == nil {
		t.Fatal("expected TTY error")
	}
	if !errors.Is(err, prompt.ErrNoTTY) {
		t.Errorf("expected wrapped ErrNoTTY, got: %v", err)
	}
	if strings.Contains(err.Error(), "failed to create edit workflow") {
		t.Errorf("TTY error should not be wrapped, got: %v", err)
	}
}
