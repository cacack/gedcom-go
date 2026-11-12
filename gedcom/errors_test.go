package gedcom

import (
	"errors"
	"testing"
)

// TestParseDateErrorSentinels checks that every ParseDate failure wraps
// exactly one of ErrEmptyDate and ErrInvalidDate, and that marking the error
// leaves its message as it was.
func TestParseDateErrorSentinels(t *testing.T) {
	tests := []struct {
		input   string
		want    error
		message string
	}{
		{"", ErrEmptyDate, "empty date string"},
		{"   ", ErrEmptyDate, "empty date"},
		{"BET 1850 1860", ErrInvalidDate, "invalid date range: missing AND keyword in 'BET 1850 1860'"},
		{"BET FOO 1850 AND 1860", ErrInvalidDate, "invalid start date in range: invalid month code 'FOO' for Gregorian calendar"},
		{"FROM 1850 TO 0", ErrInvalidDate, "invalid end date in period: invalid year: 0 (there is no year 0; 1 BC is followed by AD 1)"},
		{"INT (just a phrase)", ErrInvalidDate, "empty date in interpreted date: 'INT (just a phrase)'"},
		{"ABT", ErrInvalidDate, "empty date"},
		{"BC", ErrInvalidDate, "empty date after BC suffix"},
		{"1 2 3 4", ErrInvalidDate, "invalid date format: too many components in '1 2 3 4'"},
		{"0 JAN 1850", ErrInvalidDate, "invalid day: 0 (a day of the month is 1-31)"},
		{"-1850", ErrInvalidDate, "invalid year: -1850 (a date number has no sign; write a B.C. year with a BC suffix)"},
		{"1750/51/52", ErrInvalidDate, "invalid dual year format: 1750/51/52"},
		{"xyz", ErrInvalidDate, "invalid year: xyz"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := ParseDate(tt.input)
			if err == nil {
				t.Fatalf("ParseDate(%q) error = nil, want %v", tt.input, tt.want)
			}
			if !errors.Is(err, tt.want) {
				t.Errorf("ParseDate(%q) error %q does not wrap %v", tt.input, err, tt.want)
			}
			other := ErrInvalidDate
			if tt.want == ErrInvalidDate {
				other = ErrEmptyDate
			}
			if errors.Is(err, other) {
				t.Errorf("ParseDate(%q) error %q also wraps %v", tt.input, err, other)
			}
			if err.Error() != tt.message {
				t.Errorf("ParseDate(%q) error = %q, want %q", tt.input, err, tt.message)
			}
		})
	}
}

// TestCoordinateErrorSentinels checks the coordinate parsers mark every
// failure with ErrEmptyCoordinate or ErrInvalidCoordinate.
func TestCoordinateErrorSentinels(t *testing.T) {
	tests := []struct {
		name    string
		parse   func() error
		want    error
		message string
	}{
		{"coordinate empty", func() error { _, err := ParseCoordinate(" "); return err }, ErrEmptyCoordinate, "empty coordinate"},
		{"coordinate signed", func() error { _, err := ParseCoordinate("N-1"); return err }, ErrInvalidCoordinate, `coordinate value must be unsigned, got "N-1"`},
		{"coordinate not decimal", func() error { _, err := ParseCoordinate("Nx"); return err }, ErrInvalidCoordinate, `coordinate value must be a decimal number, got "Nx"`},
		{"coordinate direction", func() error { _, err := ParseCoordinate("X1"); return err }, ErrInvalidCoordinate, `invalid direction "X" in coordinate "X1"`},
		{"latitude empty", func() error { _, err := ParseLatitude(""); return err }, ErrEmptyCoordinate, "empty latitude"},
		{"latitude axis", func() error { _, err := ParseLatitude("E1"); return err }, ErrInvalidCoordinate, `latitude must use N/S direction, got "E1"`},
		{"latitude malformed", func() error { _, err := ParseLatitude("N1.2.3"); return err }, ErrInvalidCoordinate, `latitude: coordinate value must be a decimal number, got "N1.2.3"`},
		{"latitude range", func() error { _, err := ParseLatitude("N91"); return err }, ErrInvalidCoordinate, "latitude 91 out of range [-90, 90]"},
		{"longitude range", func() error { _, err := ParseLongitude("W181"); return err }, ErrInvalidCoordinate, "longitude -181 out of range [-180, 180]"},
		{"pair incomplete", func() error { _, _, err := (&Coordinates{Latitude: "N1"}).AsDecimal(); return err }, ErrInvalidCoordinate, `incomplete coordinates: latitude "N1", longitude ""`},
		{"pair longitude axis", func() error {
			_, _, err := (&Coordinates{Latitude: "N1", Longitude: "N2"}).AsDecimal()
			return err
		}, ErrInvalidCoordinate, `longitude must use E/W direction, got "N2"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.parse()
			if err == nil {
				t.Fatalf("error = nil, want %v", tt.want)
			}
			if !errors.Is(err, tt.want) {
				t.Errorf("error %q does not wrap %v", err, tt.want)
			}
			other := ErrInvalidCoordinate
			if tt.want == ErrInvalidCoordinate {
				other = ErrEmptyCoordinate
			}
			if errors.Is(err, other) {
				t.Errorf("error %q also wraps %v", err, other)
			}
			if err.Error() != tt.message {
				t.Errorf("error = %q, want %q", err, tt.message)
			}
		})
	}
}
