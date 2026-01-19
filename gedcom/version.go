package gedcom

// Version represents a GEDCOM specification version.
type Version string

const (
	// Version55 represents GEDCOM 5.5 specification
	Version55 Version = "5.5"

	// Version551 represents GEDCOM 5.5.1 specification
	Version551 Version = "5.5.1"

	// Version70 represents GEDCOM 7.0 specification
	Version70 Version = "7.0"
)

// String returns the string representation of the version.
func (v Version) String() string {
	return string(v)
}

// IsValid returns true if the version is a known GEDCOM version.
func (v Version) IsValid() bool {
	switch v {
	case Version55, Version551, Version70:
		return true
	default:
		return false
	}
}

// Before returns true if v is an older version than other.
// Returns false if either version is unknown.
func (v Version) Before(other Version) bool {
	order := map[Version]int{Version55: 1, Version551: 2, Version70: 3}
	vOrder, vOK := order[v]
	otherOrder, otherOK := order[other]
	if !vOK || !otherOK {
		return false
	}
	return vOrder < otherOrder
}
