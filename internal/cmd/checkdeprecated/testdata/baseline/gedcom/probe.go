// Package gedcom is a fake baseline package for the checkdeprecated tests.
package gedcom

// ProbeType is a probe.
type ProbeType struct {
	// Keep stays.
	Keep string

	// Gone is removed.
	//
	// Deprecated: use Keep.
	Gone string

	Legacy string // Deprecated: use Keep.

	// Undeprecated is removed without a marker.
	Undeprecated string

	// Retyped changes type.
	Retyped string
}

// ProbeMethod is a probe method.
//
// Deprecated: use ProbeFunc.
func (p *ProbeType) ProbeMethod() {}

// ProbeVal is a value method removed without a marker.
func (p ProbeType) ProbeVal() {}

// ProbeFunc is a probe func.
//
// Deprecated: use nothing.
func ProbeFunc() {}

// PlainFunc is removed without a marker.
func PlainFunc() {}

// ProbeConst changes value.
const ProbeConst = 1

var (
	// GroupedVar is deprecated inside its group.
	//
	// Deprecated: use ProbeConst.
	GroupedVar = 1

	// OtherVar is removed without a marker.
	OtherVar = 2
)

// Deprecated: the Old constants are unused.
const (
	// OldA is old.
	OldA = 1
)

// ProbeIface is a probe interface.
type ProbeIface interface {
	// M is a method.
	M()

	// Gone is removed.
	//
	// Deprecated: use M.
	Gone()
}

// ProbeCmp loses comparability.
type ProbeCmp struct{ A int }

// Embedder embeds a type.
type Embedder struct {
	// ProbeCmp is embedded.
	//
	// Deprecated: embed nothing.
	ProbeCmp
}

// Mentions mentions the word Deprecated: here but not at a line start.
type Mentions int
