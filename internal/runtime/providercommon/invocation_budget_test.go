package providercommon

import (
	"context"
	"errors"
	"sync"
	"testing"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestProviderInvocationBudgetCapsAllTransitions(t *testing.T) {
	budget := NewProviderInvocationBudget(3)
	ctx := WithProviderInvocationBudget(context.Background(), budget)
	task := acpruntime.Task{TaskID: "task-budget", RunID: "run-budget", StepID: "step-budget"}
	spec := CommandSpec{Provider: acpruntime.ProviderClaudeCode, Command: "/bin/sh", Args: []string{"-c", "exit 0"}}

	for _, transition := range []string{"normal", "focused_repair", "transport_retry"} {
		result, err := runCommandSpecWithTransition(ctx, task, spec, ActivityPolicy{}, transition)
		if err != nil {
			t.Fatalf("transition %s failed: %v", transition, err)
		}
		lifecycle, ok := result.Diagnostics["provider_lifecycle"].(map[string]any)
		if !ok || lifecycle["provider_invocation_budget_max"] != 3 {
			t.Fatalf("transition %s missing budget diagnostics: %#v", transition, result.Diagnostics)
		}
	}

	result, err := runCommandSpecWithTransition(ctx, task, spec, ActivityPolicy{}, "focused_repair")
	if !errors.Is(err, ErrProviderInvocationBudgetExceeded) {
		t.Fatalf("expected budget exhaustion, got result=%#v err=%v", result, err)
	}
	lifecycle, ok := result.Diagnostics["provider_lifecycle"].(map[string]any)
	if !ok {
		t.Fatalf("missing exhausted budget diagnostics: %#v", result.Diagnostics)
	}
	if lifecycle["provider_invocation_budget_exhausted"] != true || lifecycle["terminal_exhaustion_reason"] != "provider_invocation_budget_exhausted" {
		t.Fatalf("invalid exhausted budget diagnostics: %#v", lifecycle)
	}
	if snapshot := budget.Snapshot(); snapshot.Used != 3 || snapshot.Remaining != 0 || !snapshot.Exhausted {
		t.Fatalf("invalid budget snapshot: %#v", snapshot)
	}
}

func TestProviderInvocationBudgetHasProviderParity(t *testing.T) {
	providers := []acpruntime.Provider{
		acpruntime.ProviderClaudeCode,
		acpruntime.ProviderQwenCode,
		acpruntime.ProviderCodexCode,
	}
	for _, provider := range providers {
		t.Run(string(provider), func(t *testing.T) {
			budget := NewProviderInvocationBudget(DefaultProviderInvocationBudget)
			ctx := WithProviderInvocationBudget(context.Background(), budget)
			task := acpruntime.Task{TaskID: "task-" + string(provider), RunID: "run-budget", StepID: "step-budget"}
			spec := CommandSpec{Provider: provider, Command: "/bin/sh", Args: []string{"-c", "exit 0"}}
			for i := 0; i < DefaultProviderInvocationBudget; i++ {
				if _, err := runCommandSpecWithTransition(ctx, task, spec, ActivityPolicy{}, "focused_repair"); err != nil {
					t.Fatalf("provider %s invocation %d failed: %v", provider, i+1, err)
				}
			}
			if _, err := runCommandSpecWithTransition(ctx, task, spec, ActivityPolicy{}, "focused_repair"); !errors.Is(err, ErrProviderInvocationBudgetExceeded) {
				t.Fatalf("provider %s exceeded budget with err=%v", provider, err)
			}
			if got := budget.Snapshot().Used; got != DefaultProviderInvocationBudget {
				t.Fatalf("provider %s used %d starts, want %d", provider, got, DefaultProviderInvocationBudget)
			}
		})
	}
}

func TestProviderInvocationBudgetIsAtomicUnderConcurrentReservations(t *testing.T) {
	budget := NewProviderInvocationBudget(DefaultProviderInvocationBudget)
	var wg sync.WaitGroup
	var mu sync.Mutex
	reserved := 0
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := budget.Reserve("focused_repair"); err == nil {
				mu.Lock()
				reserved++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if reserved != DefaultProviderInvocationBudget || budget.Snapshot().Used != DefaultProviderInvocationBudget {
		t.Fatalf("concurrent reservations exceeded or underused hard budget: reserved=%d snapshot=%#v", reserved, budget.Snapshot())
	}
}
