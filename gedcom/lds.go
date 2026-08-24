package gedcom

// LDSOrdinanceType represents the type of LDS (Latter-Day Saints) ordinance.
type LDSOrdinanceType string

const (
	// LDSBaptism (BAPL) - LDS baptism for the dead
	LDSBaptism LDSOrdinanceType = "BAPL"

	// LDSConfirmation (CONL) - LDS confirmation
	LDSConfirmation LDSOrdinanceType = "CONL"

	// LDSEndowment (ENDL) - LDS endowment
	LDSEndowment LDSOrdinanceType = "ENDL"

	// LDSSealingChild (SLGC) - LDS sealing of child to parents
	LDSSealingChild LDSOrdinanceType = "SLGC"

	// LDSSealingSpouse (SLGS) - LDS sealing of husband and wife
	LDSSealingSpouse LDSOrdinanceType = "SLGS"
)

// LDSOrdinance represents an LDS (Latter-Day Saints) ordinance.
// Used by FamilySearch and other LDS genealogy software.
type LDSOrdinance struct {
	// Type is the ordinance type (BAPL, CONL, ENDL, SLGC, SLGS)
	Type LDSOrdinanceType

	// Date is when the ordinance was performed (DATE subordinate)
	Date string

	// ParsedDate is the parsed representation of Date.
	// This is nil if the date string could not be parsed.
	ParsedDate *Date

	// Temple is the temple code where the ordinance was performed (TEMP subordinate)
	Temple string

	// Place is the place where the ordinance was performed (PLAC subordinate)
	Place string

	// Status is the ordinance status (STAT subordinate, e.g., "COMPLETED")
	Status string

	// FamilyXRef is the family cross-reference for SLGC (child sealing to parents)
	// Only used with SLGC ordinance type (FAMC subordinate)
	FamilyXRef string

	// NoteXRefs are XRef pointers to shared NOTE/SNOTE records (e.g. "@N1@").
	NoteXRefs []string

	// InlineNotes are note text values written directly on this ordinance
	// (NOTE <text> form, including CONT/CONC continuations).
	InlineNotes []string

	// Notes is deprecated: use NoteXRefs and InlineNotes instead. It is kept
	// for backward compatibility and populated during decode with the inline
	// note text and shared-note XRefs interleaved in their original GEDCOM
	// order (not the NoteXRefs-then-InlineNotes order of the split fields).
	//
	// Deprecated: use NoteXRefs and InlineNotes.
	Notes []string
}
