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
	Type LDSOrdinanceType `json:",omitempty"`

	// Date is when the ordinance was performed (DATE subordinate)
	Date string `json:",omitempty"`

	// ParsedDate is the parsed representation of Date.
	// This is nil if the date string could not be parsed.
	ParsedDate *Date `json:",omitempty"`

	// Temple is the temple code where the ordinance was performed (TEMP subordinate)
	Temple string `json:",omitempty"`

	// Place is the place where the ordinance was performed (PLAC subordinate)
	Place string `json:",omitempty"`

	// Status is the ordinance status (STAT subordinate, e.g., "COMPLETED").
	// 5.5.1 and 7.0 require a StatusDate beneath it; see StatusDate.
	Status string `json:",omitempty"`

	// StatusDate is the date the status was set (STAT.DATE, GEDCOM 5.5.1 and
	// 7.0), kept as written. Both versions require it beneath STAT: set it
	// for valid 5.5.1 or 7.0 output, since the encoder writes a Status that
	// has no StatusDate alone rather than drop it. 5.5 defines no STAT.DATE,
	// so 5.5 output writes Status alone.
	StatusDate string `json:",omitempty"`

	// StatusTime is the time beneath StatusDate (STAT.DATE.TIME, GEDCOM 7.0),
	// written only beneath a StatusDate and only for 7.0 output.
	StatusTime string `json:",omitempty"`

	// FamilyXRef is the family cross-reference for SLGC (child sealing to parents)
	// Only used with SLGC ordinance type (FAMC subordinate)
	FamilyXRef string `json:",omitempty"`

	// NoteXRefs are XRef pointers to shared NOTE/SNOTE records (e.g. "@N1@").
	NoteXRefs []string `json:",omitempty"`

	// InlineNotes are note text values written directly on this ordinance
	// (NOTE <text> form, including CONT/CONC continuations).
	InlineNotes []string `json:",omitempty"`
}
