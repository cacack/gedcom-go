package gedcom

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ParseCoordinate parses a GEDCOM coordinate string ("N42.3601", "W71.0589")
// to signed decimal degrees. N/E are positive; S/W are negative. Direction
// letters are case-insensitive and surrounding whitespace is ignored.
//
// The value following the direction letter must be an unsigned decimal number:
// at least one digit and at most one decimal point. The sign is carried by the
// direction, so an explicitly signed value such as "N-42.3601" is rejected
// rather than silently inverted. The wider syntax strconv.ParseFloat accepts —
// NaN ("Nnan"), infinities ("Ninf"), hexadecimal float literals ("N0x1p3"),
// exponent notation ("N1e2") and underscore separators ("N1_0.5") — is rejected
// too: none of those is a GEDCOM coordinate.
//
// It performs format conversion only and does not range-check the result, nor
// does it verify that the direction matches a particular axis, because a single
// component carries no information about whether it is a latitude or a
// longitude. Callers are responsible for using N/S for latitudes and E/W for
// longitudes; Coordinates.AsDecimal enforces this. Returns an error for empty
// or malformed input.
func ParseCoordinate(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty coordinate")
	}

	direction := s[0]
	num := s[1:]
	if strings.HasPrefix(num, "+") || strings.HasPrefix(num, "-") {
		return 0, fmt.Errorf("coordinate value must be unsigned, got %q", s)
	}
	if !isPlainDecimal(num) {
		return 0, fmt.Errorf("coordinate value must be a decimal number, got %q", s)
	}

	value, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid coordinate %q: %w", s, err)
	}

	switch direction {
	case 'N', 'n', 'E', 'e':
		return value, nil
	case 'S', 's', 'W', 'w':
		return -value, nil
	default:
		return 0, fmt.Errorf("invalid direction %q in coordinate %q", string(direction), s)
	}
}

// ParseLatitude parses a GEDCOM latitude component ("N42.3601", "S33.8688") to
// signed decimal degrees. It is ParseCoordinate plus the two checks a lone
// component cannot make for itself: that the direction is N or S, and that the
// result falls within [-90, 90].
//
// Prefer it over ParseCoordinate whenever the axis is known — ParseCoordinate
// accepts any direction letter by design, so it cannot tell a latitude from a
// longitude and will happily return a value for "E42.3601".
func ParseLatitude(s string) (float64, error) {
	return latitudeAxis.parse(s)
}

// ParseLongitude parses a GEDCOM longitude component ("E151.2093", "W71.0589")
// to signed decimal degrees. It is ParseCoordinate plus a check that the
// direction is E or W and that the result falls within [-180, 180]. See
// ParseLatitude for why this is preferable when the axis is known.
func ParseLongitude(s string) (float64, error) {
	return longitudeAxis.parse(s)
}

// isPlainDecimal reports whether s is an unsigned decimal number: at least one
// digit, at most one decimal point, and nothing else. It excludes the wider
// syntax strconv.ParseFloat accepts — NaN, infinities, hex float literals,
// exponents, and underscore separators — none of which is a GEDCOM coordinate.
func isPlainDecimal(s string) bool {
	seenDigit, seenDot := false, false
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c >= '0' && c <= '9':
			seenDigit = true
		case c == '.':
			if seenDot {
				return false
			}
			seenDot = true
		default:
			return false
		}
	}
	return seenDigit
}

// IsEmpty reports whether the coordinate pair carries no data. A nil receiver
// and a pair whose latitude and longitude are both empty (ignoring whitespace)
// are both empty. Use this to distinguish absent coordinates from valid
// coordinates at the origin (0°, 0°), which AsDecimal also reports as
// (0, 0, nil).
func (c *Coordinates) IsEmpty() bool {
	if c == nil {
		return true
	}
	return strings.TrimSpace(c.Latitude) == "" && strings.TrimSpace(c.Longitude) == ""
}

// AsDecimal returns latitude and longitude as signed decimal degrees.
// N/E are positive; S/W are negative.
//
// It returns (0, 0, nil) when the pair is empty — that is, when the receiver is
// nil or both components are blank. Note that valid coordinates at the origin
// (latitude "N0", longitude "E0") also yield (0, 0, nil); callers that must
// distinguish the absent case should consult IsEmpty first.
//
// If exactly one component is present the pair is incomplete and an error is
// returned. An error is also returned when a component is malformed, uses the
// wrong axis direction (latitude must use N/S, longitude E/W), is not a finite
// number, or is out of range (latitude within [-90, 90], longitude within
// [-180, 180]).
func (c *Coordinates) AsDecimal() (lat, long float64, err error) {
	if c == nil {
		return 0, 0, nil
	}

	latStr := strings.TrimSpace(c.Latitude)
	longStr := strings.TrimSpace(c.Longitude)

	if latStr == "" && longStr == "" {
		return 0, 0, nil
	}
	if latStr == "" || longStr == "" {
		return 0, 0, fmt.Errorf("incomplete coordinates: latitude %q, longitude %q", c.Latitude, c.Longitude)
	}

	lat, err = latitudeAxis.parse(c.Latitude)
	if err != nil {
		return 0, 0, err
	}
	long, err = longitudeAxis.parse(c.Longitude)
	if err != nil {
		return 0, 0, err
	}

	return lat, long, nil
}

// axis describes one of the two coordinate axes: the direction letters it
// accepts and the range its values must fall in. It is the single home for the
// N/S-versus-E/W rule; both AsDecimal and the decoder's MAP validation consult
// it rather than restating the letters.
type axis struct {
	name     string
	positive byte // N for latitude, E for longitude
	negative byte // S for latitude, W for longitude
	lo, hi   float64
}

var (
	latitudeAxis  = axis{name: "latitude", positive: 'N', negative: 'S', lo: -90, hi: 90}
	longitudeAxis = axis{name: "longitude", positive: 'E', negative: 'W', lo: -180, hi: 180}
)

// accepts reports whether c is one of the axis's direction letters, in either
// case. The axis letters are uppercase, so a lowercase c is folded up first.
func (a axis) accepts(c byte) bool {
	if c >= 'a' && c <= 'z' {
		c -= 'a' - 'A'
	}
	return c == a.positive || c == a.negative
}

// directions renders the axis's letters for an error message, e.g. "N/S".
func (a axis) directions() string {
	return fmt.Sprintf("%c/%c", a.positive, a.negative)
}

// parse converts one component of a coordinate pair to signed decimal degrees,
// enforcing this axis's direction and range. raw is reported verbatim in
// errors; surrounding whitespace is ignored, and an empty component is
// rejected.
func (a axis) parse(raw string) (float64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, fmt.Errorf("empty %s", a.name)
	}
	if !a.accepts(trimmed[0]) {
		return 0, fmt.Errorf("%s must use %s direction, got %q", a.name, a.directions(), raw)
	}

	value, err := ParseCoordinate(trimmed)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", a.name, err)
	}
	// Unreachable while ParseCoordinate enforces plain-decimal input. Kept
	// explicit so a future relaxation there cannot silently reintroduce a NaN
	// success return: NaN fails both range comparisons below.
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("%s %q is not a finite number", a.name, raw)
	}
	if value < a.lo || value > a.hi {
		return 0, fmt.Errorf("%s %g out of range [%g, %g]", a.name, value, a.lo, a.hi)
	}

	return value, nil
}
