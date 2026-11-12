package gedcom

import "errors"

// ErrEmptyDate is wrapped by every ParseDate error for an input that is empty
// or only whitespace. Use errors.Is to tell an absent date from a malformed one.
var ErrEmptyDate = errors.New("empty date")

// ErrInvalidDate is wrapped by every ParseDate error for a non-empty input that
// does not parse. The error's message says what is wrong.
var ErrInvalidDate = errors.New("invalid date")

// ErrEmptyCoordinate is wrapped by every ParseCoordinate, ParseLatitude and
// ParseLongitude error for an input that is empty or only whitespace.
var ErrEmptyCoordinate = errors.New("empty coordinate")

// ErrInvalidCoordinate is wrapped by every ParseCoordinate, ParseLatitude,
// ParseLongitude and Coordinates.AsDecimal error for a coordinate that is
// malformed, uses the wrong direction, is out of range, or (for AsDecimal) is
// missing one component of the pair.
var ErrInvalidCoordinate = errors.New("invalid coordinate")
