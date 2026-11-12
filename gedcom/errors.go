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

// ErrNilDocument is returned by Document.AddRecord called on a nil *Document.
var ErrNilDocument = errors.New("nil document")

// ErrNilRecord is returned by Document.AddRecord for a nil *Record.
var ErrNilRecord = errors.New("nil record")

// ErrDuplicateXRef is wrapped by the Document.AddRecord error for a record
// whose XRef the document already indexes. The error's message names the XRef.
var ErrDuplicateXRef = errors.New("duplicate xref")

// ErrBelowMinimumAge is wrapped by the ValidateParentChildDates and
// ValidateMarriageDates error for a parent or spouse younger than the minimum
// age. The error's message gives the age.
var ErrBelowMinimumAge = errors.New("below minimum age")

// ErrEventBeforeBirth is wrapped by the ValidateParentChildDates and
// ValidateMarriageDates error for a child's birth or marriage provably before
// the parent's or spouse's birth. The error's message gives both dates.
var ErrEventBeforeBirth = errors.New("event before birth")
