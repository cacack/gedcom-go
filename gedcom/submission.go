package gedcom

// Submission represents a GEDCOM 5.5/5.5.1 submission record (SUBN): a
// request, addressed to the Family History Department, to process the
// transmitted data (for example through TempleReady). GEDCOM 7.0 removed it.
//
// Every value is stored as written: ANCE, DESC and ORDI are not parsed into
// numbers or booleans, so a malformed value is kept rather than lost.
type Submission struct {
	// XRef is the cross-reference identifier for this submission
	XRef string

	// SubmitterXRef is the SUBM pointer to the submitter record (e.g. "@U1@").
	SubmitterXRef string

	// FamilyFileName is the FAMF value: the name of the family file.
	FamilyFileName string

	// TempleCode is the TEMP value: the temple where ordinances are to be
	// performed.
	TempleCode string

	// AncestorGenerations is the ANCE value: the number of generations of
	// ancestors to include.
	AncestorGenerations string

	// DescendantGenerations is the DESC value: the number of generations of
	// descendants to include.
	DescendantGenerations string

	// OrdinanceProcessFlag is the ORDI value ("yes" or "no"): whether
	// ordinance data should be processed.
	OrdinanceProcessFlag string

	// AutomatedRecordID is the RIN value: a record identifier assigned by the
	// originating system.
	AutomatedRecordID string

	// NoteXRefs are XRef pointers to shared NOTE records (e.g. "@N1@").
	NoteXRefs []string

	// InlineNotes are note text values written directly on this record
	// (1 NOTE <text> form, including CONT/CONC continuations).
	InlineNotes []string

	// ChangeDate is the CHAN structure (GEDCOM 5.5.1).
	ChangeDate *ChangeDate

	// Tags contains all raw tags for this submission (for unknown/custom tags),
	// a lossless read-side record per ADR 0003. It does not drive encoding --
	// see Record.Tags for the rule that governs what a decoded record writes.
	Tags []*Tag
}

// AllNotes returns this submission's inline notes followed by the text of any
// shared notes referenced by NoteXRefs, resolved against doc. Shared notes that
// do not resolve are skipped. Returns nil when there are no notes.
func (s *Submission) AllNotes(doc *Document) []string {
	return allNotes(doc, s.InlineNotes, s.NoteXRefs)
}
