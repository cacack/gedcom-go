package scalecheck

import (
	"fmt"
	"strings"
	"testing"
)

// recorder is a TB that keeps what Errorf was given.
type recorder struct{ errors []string }

func (r *recorder) Helper() {}

func (r *recorder) Errorf(format string, args ...any) {
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

// sink keeps the loops below from being optimized away.
var sink int

// work returns an operation that does steps(n) units of work.
func work(steps func(n int) int) func(n int) func() {
	return func(n int) func() {
		return func() {
			for k := range steps(n) {
				sink += k
			}
		}
	}
}

func TestLinear(t *testing.T) {
	const small, large = 2000, 32000

	t.Run("linear passes", func(t *testing.T) {
		r := &recorder{}
		Linear(r, "linear", small, large, work(func(n int) int { return 200 * n }))
		if len(r.errors) != 0 {
			t.Errorf("Linear reported %q for a linear operation", r.errors)
		}
	})

	t.Run("quadratic fails", func(t *testing.T) {
		r := &recorder{}
		Linear(r, "quadratic", small, large, work(func(n int) int { return n * n / 250 }))
		if len(r.errors) != 1 || !strings.HasPrefix(r.errors[0], "quadratic: 16 runs at size 2000 took ") {
			t.Errorf("Linear reported %q for a quadratic operation, want one error naming it", r.errors)
		}
	})
}
