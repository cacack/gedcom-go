package gedcom

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseCoordinate parses a GEDCOM coordinate string ("N42.3601", "W71.0589")
// to signed decimal degrees. N/E are positive; S/W are negative. Direction
// letters are case-insensitive and surrounding whitespace is ignored.
//
// The value following the direction letter must be unsigned: the sign is
// carried by the direction, so an explicitly signed value such as "N-42.3601"
// is rejected rather than silently inverted.
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
// wrong axis direction (latitude must use N/S, longitude E/W), or is out of
// range (latitude within [-90, 90], longitude within [-180, 180]).
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

	switch latStr[0] {
	case 'N', 'n', 'S', 's':
	default:
		return 0, 0, fmt.Errorf("latitude must use N/S direction, got %q", c.Latitude)
	}
	switch longStr[0] {
	case 'E', 'e', 'W', 'w':
	default:
		return 0, 0, fmt.Errorf("longitude must use E/W direction, got %q", c.Longitude)
	}

	lat, err = ParseCoordinate(latStr)
	if err != nil {
		return 0, 0, fmt.Errorf("latitude: %w", err)
	}
	if lat < -90 || lat > 90 {
		return 0, 0, fmt.Errorf("latitude %g out of range [-90, 90]", lat)
	}

	long, err = ParseCoordinate(longStr)
	if err != nil {
		return 0, 0, fmt.Errorf("longitude: %w", err)
	}
	if long < -180 || long > 180 {
		return 0, 0, fmt.Errorf("longitude %g out of range [-180, 180]", long)
	}

	return lat, long, nil
}
