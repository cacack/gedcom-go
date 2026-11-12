// Package errmark attaches a matchable sentinel to an error without changing
// its message, so a package can make an existing error errors.Is-matchable
// while callers that print or compare the message see what they saw before.
//
// Use Mark when the existing message does not begin with the sentinel's text.
// When it does (the message is "<sentinel text>: <detail>"), wrap with
// fmt.Errorf("%w: ...") instead, as converter does; a test pinning the message
// then guards against the sentinel text being reworded.
package errmark

// Mark returns err marked with sentinel: errors.Is matches both sentinel and
// anything err wraps, Error returns err's message unchanged, and Unwrap
// returns err. A nil err stays nil.
func Mark(sentinel, err error) error {
	if err == nil {
		return nil
	}
	return &marked{sentinel: sentinel, err: err}
}

type marked struct {
	sentinel error
	err      error
}

func (e *marked) Error() string { return e.err.Error() }

func (e *marked) Is(target error) bool { return target == e.sentinel }

func (e *marked) Unwrap() error { return e.err }
