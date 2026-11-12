package decoder_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cacack/gedcom-go/v3/decoder"
)

// ExampleDecodeWithOptions_timeout is the timeout pattern from the README and
// FEATURES.md: a deadline on the context stops the decode, and errors.Is
// recognizes the deadline in the error returned. The deadline here has
// already passed, so the example's output does not depend on timing.
func ExampleDecodeWithOptions_timeout() {
	ctx, cancel := context.WithTimeout(context.Background(), -time.Second)
	defer cancel()

	opts := &decoder.DecodeOptions{Context: ctx}
	_, err := decoder.DecodeWithOptions(strings.NewReader("0 HEAD\n0 TRLR\n"), opts)
	if errors.Is(err, context.DeadlineExceeded) {
		fmt.Println("Decode timed out")
	}

	// Output:
	// Decode timed out
}
