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

	// Notes is deprecated: use NoteXRefs and InlineNotes instead. It is kept
	// for backward compatibility and populated during decode with the inline
	// note text and shared-note XRefs interleaved in their original GEDCOM
	// order (not the NoteXRefs-then-InlineNotes order of the split fields).
	//
	// Deprecated: use NoteXRefs and InlineNotes.
	Notes []string
}
