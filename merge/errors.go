package merge

import (
	"errors"
	"fmt"
)

// ErrInvalidRemap is returned by RemapXRefs when the caller-supplied
// transform produces output that cannot be applied — either a malformed
// XRef or a collision with another transform output in the same call.
// Use errors.Is to detect this case; use errors.As with *RemapError to
// recover the specific failing input/output pair and reason.
var ErrInvalidRemap = errors.New("invalid xref remap")

// RemapError records a specific transform failure. It is returned by
// RemapXRefs wrapped with ErrInvalidRemap.
type RemapError struct {
	// Old is the input XRef passed to the transform.
	Old string
	// New is the XRef returned by the transform.
	New string
	// Reason describes why the mapping was rejected (malformed shape,
	// collision with another output, etc.).
	Reason string
}

func (e *RemapError) Error() string {
	return fmt.Sprintf("merge: remap %q -> %q: %s", e.Old, e.New, e.Reason)
}

func (e *RemapError) Is(target error) bool {
	return target == ErrInvalidRemap
}
