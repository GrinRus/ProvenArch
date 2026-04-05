package testutil

import (
	"fmt"
	"testing"
	"time"
)

// WaitFor polls check until it returns done=true, or fails on timeout/check errors.
func WaitFor(t testing.TB, timeout time.Duration, description string, check func() (done bool, err error)) {
	t.Helper()

	if timeout <= 0 {
		t.Fatalf("wait timeout must be positive: %s", timeout)
	}
	interval := 20 * time.Millisecond
	timer := time.NewTimer(timeout)
	ticker := time.NewTicker(interval)
	defer timer.Stop()
	defer ticker.Stop()

	for {
		done, err := check()
		if err != nil {
			t.Fatalf("%s: %v", description, err)
		}
		if done {
			return
		}

		select {
		case <-timer.C:
			t.Fatalf("%s before timeout %s", description, timeout)
		case <-ticker.C:
		}
	}
}

func WaitDescription(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}
