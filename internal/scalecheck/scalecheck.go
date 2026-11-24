// Package scalecheck holds the one timing check behind the tests that guard
// an operation against going quadratic (issue #948), so that every such test
// measures the same way.
package scalecheck

import (
	"math"
	"time"
)

// TB is the part of testing.TB that Linear uses.
type TB interface {
	Helper()
	Errorf(format string, args ...any)
}

const (
	// limit is the largest ratio Linear accepts. A quadratic operation gives
	// about large/small, 16 or more. A linear one would give 1 if size did
	// not matter at all; the operations checked here measure 1.5 to 2.5,
	// because the larger input is harder on the cache and the allocator.
	// The limit sits about halfway between the two on a log scale.
	limit = 8.0

	// minSpan is how long the timed spans are sized to last, at least. The
	// clock on Windows can advance in steps of up to 15.6 ms, so a span of
	// a few milliseconds reads as zero or as one whole step there.
	minSpan = 50 * time.Millisecond

	// minRounds is how many rounds are always timed; maxRounds is where
	// Linear stops waiting for a quiet sample and fails.
	minRounds = 3
	maxRounds = 8
)

// Linear fails t unless an operation costs about the same at one input of
// size large as over large/small inputs of size small, which is what linear
// scaling means. setup builds the input of a given size, untimed, and returns
// the operation to time. large must be a multiple of small, 16 times or more.
//
// Both timed spans do the same amount of work when the operation is linear,
// so the ratio between them stays small (see limit); a quadratic operation
// makes it about large/small. Each span repeats its work as many times as it
// takes the small one to last minSpan, so neither is short enough for timer
// or scheduler noise to dominate. Rounds alternate the two spans and the
// fastest of each is kept, since noise only ever adds time. After minRounds
// rounds Linear returns as soon as the ratio is within the limit; a ratio
// over it gets more rounds, up to maxRounds, before it counts as a failure.
// One slow stretch on a shared machine therefore cannot fail the check, and
// a quadratic operation fails every round.
func Linear(t TB, name string, small, large int, setup func(n int) func()) {
	t.Helper()
	reps := large / small
	runSmall, runLarge := setup(small), setup(large)

	passes := 1
	smallSpan := func() {
		for range passes * reps {
			runSmall()
		}
	}
	largeSpan := func() {
		for range passes {
			runLarge()
		}
	}
	for elapsed(smallSpan) < minSpan {
		passes *= 2
	}

	tSmall, tLarge := time.Duration(math.MaxInt64), time.Duration(math.MaxInt64)
	ratio := 0.0
	for round := 1; round <= maxRounds; round++ {
		tSmall = min(tSmall, elapsed(smallSpan))
		tLarge = min(tLarge, elapsed(largeSpan))
		ratio = float64(tLarge) / float64(tSmall)
		if round >= minRounds && ratio <= limit {
			return
		}
	}
	t.Errorf("%s: %d runs at size %d took %v, %d at size %d took %v (%.1fx; linear is under 3x, quadratic about %dx, limit %.0fx)",
		name, passes*reps, small, tSmall, passes, large, tLarge, ratio, reps, limit)
}

// now is the clock Linear reads; the package's own test replaces it.
var now = time.Now

// elapsed returns how long f takes.
func elapsed(f func()) time.Duration {
	start := now()
	f()
	return now().Sub(start)
}
