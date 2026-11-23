package gedcom

// Layout is the physical form of the file a Document was decoded from: its
// byte order mark, each line's indentation and terminator, and the blank or
// skipped lines between them. The decoder records it when asked to
// (decoder.DecodeOptions.PreserveLayout), and the encoder writes it back when
// asked to (encoder.EncodeOptions.PreserveLayout), so a line the caller did
// not change is written exactly as it was read.
//
// A Layout describes the source, not the document: it is never updated as the
// document is edited. The encoder writes a line's source bytes only when the
// line it is about to write still reads as that source line (same level,
// identifier, tag and value), so an edited, moved or deleted structure can
// never bring stale text back. It writes each source line at most once, so a
// copied line, which keeps its LineNumber, is written from source only at its
// first occurrence. Treat a Layout as read-only; a cloned Document shares it.
//
// Text is held as decoded, so for a source that was not UTF-8 (ANSEL, UTF-16,
// a code page) it is the UTF-8 the encoder writes, not the original bytes.
type Layout struct {
	// BOM reports whether the source opened with a UTF-8 byte order mark. A
	// UTF-16 byte order mark is not recorded: the encoder writes UTF-8.
	BOM bool

	// Lines holds the source's physical lines by line number: Lines[n-1] is
	// line n, as numbered in Tag.LineNumber and Record.LineNumber. A line the
	// parser skipped (blank, or malformed beyond recovery) has a zero entry,
	// and its bytes are carried in the Before of the next line that parsed.
	Lines []LayoutLine

	// Head and Trailer are the line numbers of the document's "0 HEAD" and
	// "0 TRLR" lines, or 0 when the source had none.
	Head, Trailer int

	// Tail is the input after the last line that parsed, such as trailing
	// blank lines.
	Tail string
}

// LayoutLine is one source line that parsed: what it parsed to, and the bytes
// it was read from.
type LayoutLine struct {
	// Level, XRef, Tag and Value are the line as the parser read it, before
	// any lenient recovery (such as level-jump clamping) changed it.
	Level int
	XRef  string
	Tag   string
	Value string

	// Before is the skipped lines immediately preceding this one, their
	// terminators included: blank lines between records, say.
	Before string

	// Text is the line exactly as read, from its first byte (indentation
	// included) up to, not including, its terminator.
	Text string

	// Terminator is the line's terminator: "\r\n", "\n", "\r", or "" for a
	// final line with none.
	Terminator string
}

// Line returns the source line numbered n, or nil when l is nil or has no
// line n that parsed.
func (l *Layout) Line(n int) *LayoutLine {
	if l == nil || n < 1 || n > len(l.Lines) {
		return nil
	}
	line := &l.Lines[n-1]
	if line.Text == "" {
		return nil
	}
	return line
}

// Reads reports whether the line reads as a line with this level, identifier,
// tag and value, so its Text can be written in that line's place.
func (ll *LayoutLine) Reads(level int, xref, tag, value string) bool {
	return ll.Level == level && ll.XRef == xref && ll.Tag == tag && ll.Value == value
}
