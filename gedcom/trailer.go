package gedcom

// Trailer marks the end of a GEDCOM file.
// The trailer is always a single line "0 TRLR" in valid GEDCOM files.
type Trailer struct {
	// LineNumber is the line number where the trailer appears
	LineNumber int
}
