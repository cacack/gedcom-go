package scalecheck

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// recorder is a TB that keeps what Errorf was given.
type recorder struct{ errors []string }

func (r *recorder) Helper() {}

func (r *recorder) Errorf(format string, args ...any) {
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

// fakeClock stands in for the package clock, so that the test neither waits
// nor depends on the machine. It advances only when an operation built by
// work runs, and reads in whole steps of tick, as a coarse system clock does.
type fakeClock struct {
	at   time.Time
	tick time.Duration
}

func (c *fakeClock) now() time.Time { return c.at.Truncate(c.tick) }

// work returns an operation that takes cost(n) microseconds on the clock.
func (c *fakeClock) work(cost func(n int) int) func(n int) func() {
	return func(n int) func() {
		return func() { c.at = c.at.Add(time.Duration(cost(n)) * time.Microsecond) }
	}
}

func TestLinear(t *testing.T) {
	const small, large = 2000, 32000

	// windowsTick is the coarsest step the Windows clock advances in.
	const windowsTick = 15625 * time.Microsecond

	// The linear cost is not strictly proportional: like the operations
	// Linear checks, it costs about 1.7 times as much per element at large.
	linear := func(n int) int { return n/10 + n*n/400000 }
	quadratic := func(n int) int { return n * n / 4000 }

	t.Cleanup(func() { now = time.Now })
	for _, tick := range []time.Duration{time.Nanosecond, windowsTick} {
		clock := &fakeClock{at: time.Unix(0, 0), tick: tick}
		now = clock.now

		t.Run(fmt.Sprintf("linear passes, %v clock", tick), func(t *testing.T) {
			r := &recorder{}
			Linear(r, "linear", small, large, clock.work(linear))
			if len(r.errors) != 0 {
				t.Errorf("Linear reported %q for a linear operation", r.errors)
			}
		})

		t.Run(fmt.Sprintf("quadratic fails, %v clock", tick), func(t *testing.T) {
			r := &recorder{}
			Linear(r, "quadratic", small, large, clock.work(quadratic))
			if len(r.errors) != 1 || !strings.HasPrefix(r.errors[0], "quadratic: 64 runs at size 2000 took ") {
				t.Errorf("Linear reported %q for a quadratic operation, want one error naming it", r.errors)
			}
		})
	}
}
