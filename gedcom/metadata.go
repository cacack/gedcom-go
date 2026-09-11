package gedcom

// ChangeDate represents when a record was created or last modified.
// Used by CHAN (change date) and CREA (creation date) tags.
type ChangeDate struct {
	// Date is the date of the change (in GEDCOM date format)
	Date string

	// Time is the time of the change (in HH:MM:SS format)
	Time string

	// NoteXRefs are XRef pointers to shared NOTE/SNOTE records (e.g. "@N1@").
	NoteXRefs []string

	// InlineNotes are note text values written directly on this change date
	// (NOTE <text> form, including CONT/CONC continuations).
	InlineNotes []string
}
