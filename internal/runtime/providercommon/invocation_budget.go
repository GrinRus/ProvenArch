package providercommon

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

const DefaultProviderInvocationBudget = 3

var ErrProviderInvocationBudgetExceeded = errors.New("provider invocation budget exhausted")

type invocationBudgetContextKey struct{}
type invocationTransitionContextKey struct{}

// ProviderInvocationBudget bounds provider process starts for one runtime
// execution unit. The budget is intentionally shared by the normal call and
// every recovery transition that receives the same context.
type ProviderInvocationBudget struct {
	mu      sync.Mutex
	max     int
	used    int
	last    string
	exhaust bool
}

type ProviderInvocationReservation struct {
	Max        int
	Used       int
	Remaining  int
	Transition string
	Exhausted  bool
}

func NewProviderInvocationBudget(max int) *ProviderInvocationBudget {
	if max <= 0 {
		max = DefaultProviderInvocationBudget
	}
	return &ProviderInvocationBudget{max: max}
}

func WithProviderInvocationBudget(ctx context.Context, budget *ProviderInvocationBudget) context.Context {
	if budget == nil {
		return ctx
	}
	return context.WithValue(ctx, invocationBudgetContextKey{}, budget)
}

func ProviderInvocationBudgetFromContext(ctx context.Context) *ProviderInvocationBudget {
	if ctx == nil {
		return nil
	}
	budget, _ := ctx.Value(invocationBudgetContextKey{}).(*ProviderInvocationBudget)
	return budget
}

func ProviderInvocationTransitionFromContext(ctx context.Context) string {
	if ctx == nil {
		return "normal"
	}
	return normalizeInvocationTransition(fmt.Sprint(ctx.Value(invocationTransitionContextKey{})))
}

func (b *ProviderInvocationBudget) Reserve(transition string) (ProviderInvocationReservation, error) {
	transition = normalizeInvocationTransition(transition)
	if b == nil {
		return ProviderInvocationReservation{Transition: transition}, nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.max <= 0 {
		b.max = DefaultProviderInvocationBudget
	}
	if b.used >= b.max {
		b.exhaust = true
		b.last = transition
		return ProviderInvocationReservation{
			Max:        b.max,
			Used:       b.used,
			Remaining:  0,
			Transition: transition,
			Exhausted:  true,
		}, &ProviderInvocationBudgetError{Max: b.max, Used: b.used, Transition: transition}
	}
	b.used++
	b.last = transition
	return ProviderInvocationReservation{
		Max:        b.max,
		Used:       b.used,
		Remaining:  b.max - b.used,
		Transition: transition,
	}, nil
}

func (b *ProviderInvocationBudget) Snapshot() ProviderInvocationReservation {
	if b == nil {
		return ProviderInvocationReservation{}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	remaining := b.max - b.used
	if remaining < 0 {
		remaining = 0
	}
	return ProviderInvocationReservation{
		Max:        b.max,
		Used:       b.used,
		Remaining:  remaining,
		Transition: b.last,
		Exhausted:  b.exhaust,
	}
}

// ProviderInvocationP95 returns the nearest-rank p95 for a deterministic set
// of runtime-unit invocation counts. Empty and non-positive samples are
// ignored; callers use this for provider-free conformance evidence, not for
// changing the hard process-start limit.
func ProviderInvocationP95(samples []int) int {
	values := make([]int, 0, len(samples))
	for _, sample := range samples {
		if sample > 0 {
			values = append(values, sample)
		}
	}
	if len(values) == 0 {
		return 0
	}
	sort.Ints(values)
	index := (95*len(values) + 99) / 100
	if index < 1 {
		index = 1
	}
	if index > len(values) {
		index = len(values)
	}
	return values[index-1]
}

type ProviderInvocationBudgetError struct {
	Max        int
	Used       int
	Transition string
}

func (e *ProviderInvocationBudgetError) Error() string {
	if e == nil {
		return ErrProviderInvocationBudgetExceeded.Error()
	}
	return fmt.Sprintf("%s: max=%d used=%d transition=%s", ErrProviderInvocationBudgetExceeded, e.Max, e.Used, normalizeInvocationTransition(e.Transition))
}

func (e *ProviderInvocationBudgetError) Unwrap() error { return ErrProviderInvocationBudgetExceeded }

func normalizeInvocationTransition(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "focused_repair", "transport_retry", "normal":
		return value
	default:
		return "normal"
	}
}
