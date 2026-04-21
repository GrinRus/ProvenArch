package orchestrator

import (
	"context"
	"fmt"
	"sync"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

type stepRunnerResolver struct {
	mu       sync.Mutex
	factory  stepRunnerFactory
	profiles acpruntime.StepProviderValues
	cache    map[acpruntime.Provider]acpruntime.Runner
	ready    map[acpruntime.Provider]error
}

func newStepRunnerResolver(factory stepRunnerFactory, profiles acpruntime.StepProviderValues) *stepRunnerResolver {
	if factory == nil {
		factory = stepRunnerFactoryFunc(func(acpruntime.Provider) (acpruntime.Runner, error) {
			return nil, fmt.Errorf("runner factory is not configured")
		})
	}
	return &stepRunnerResolver{
		factory:  factory,
		profiles: profiles,
		cache:    map[acpruntime.Provider]acpruntime.Runner{},
		ready:    map[acpruntime.Provider]error{},
	}
}

func (r *stepRunnerResolver) ProviderForStep(stepID string) acpruntime.Provider {
	if r == nil {
		return acpruntime.ProviderClaudeCode
	}
	return r.profiles.ProviderForStep(stepID)
}

func (r *stepRunnerResolver) RunnerForStep(stepID string) (acpruntime.Provider, acpruntime.Runner, error) {
	provider := r.ProviderForStep(stepID)
	runner, err := r.runnerForProvider(provider)
	if err != nil {
		return "", nil, err
	}
	return provider, runner, nil
}

func (r *stepRunnerResolver) ReadyRunnerForStep(ctx context.Context, stepID string) (acpruntime.Provider, acpruntime.Runner, error) {
	provider, runner, err := r.RunnerForStep(stepID)
	if err != nil {
		return "", nil, err
	}
	if err := r.ensureProviderReady(ctx, provider, runner); err != nil {
		return "", nil, err
	}
	return provider, runner, nil
}

func (r *stepRunnerResolver) Preflight(ctx context.Context) error {
	for _, provider := range r.profiles.All() {
		runner, err := r.runnerForProvider(provider)
		if err != nil {
			return err
		}
		if err := r.ensureProviderReady(ctx, provider, runner); err != nil {
			return err
		}
	}
	return nil
}

func (r *stepRunnerResolver) runnerForProvider(provider acpruntime.Provider) (acpruntime.Runner, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if runner, ok := r.cache[provider]; ok {
		return runner, nil
	}
	runner, err := r.factory.Build(provider)
	if err != nil {
		return nil, err
	}
	r.cache[provider] = runner
	return runner, nil
}

func (r *stepRunnerResolver) ensureProviderReady(ctx context.Context, provider acpruntime.Provider, runner acpruntime.Runner) error {
	r.mu.Lock()
	if err, ok := r.ready[provider]; ok {
		r.mu.Unlock()
		return err
	}
	checker, ok := runner.(acpruntime.PreflightRunner)
	if !ok {
		r.ready[provider] = nil
		r.mu.Unlock()
		return nil
	}
	err := checker.Preflight(ctx)
	r.ready[provider] = err
	r.mu.Unlock()
	return err
}
