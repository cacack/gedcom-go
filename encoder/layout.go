package encoder

import (
	"io"
	"strings"
	"unicode"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// utf8BOM is the UTF-8 byte order mark, written first when the source had one.
const utf8BOM = "\xEF\xBB\xBF"

// sourceLayout is the layout an encode writes back, with the source lines
// already written, by line number, so each is written at most once (see
// writeSourced).
type sourceLayout struct {
	*gedcom.Layout
	used []bool
}

// attachLayout sets opts, a copy forOutput returned, to write layout back (see
// EncodeOptions.PreserveLayout). It does nothing unless the caller asked for
// that and the document has a layout to write.
//
// A line with no source line -- one the caller added, or one the encoder
// synthesizes -- is written with the source's own terminator, so a CRLF file
// stays CRLF throughout rather than gaining LF lines where it was edited.
func (opts *EncodeOptions) attachLayout(layout *gedcom.Layout) {
	if !opts.PreserveLayout || layout == nil {
		return
	}
	opts.layout = &sourceLayout{Layout: layout, used: make([]bool, len(layout.Lines)+1)}
	if ending := layoutLineEnding(layout); ending != "" {
		opts.LineEnding = ending
	}
}

// layoutLineEnding returns the terminator of the source's first terminated
// line, or "" when no line has one.
func layoutLineEnding(layout *gedcom.Layout) string {
	for i := range layout.Lines {
		if t := layout.Lines[i].Terminator; t != "" {
			return t
		}
	}
	return ""
}

// writeSourced writes a line at level, with xref, tag and value, that was
// read from source line n (0 when it was not read from one).
//
// Without a layout, or with no source line n, it is writeValue. Otherwise, if
// the line still reads as source line n, that line is written exactly as it
// was read, preceded by the skipped lines that preceded it. If it does not --
// the caller changed it -- it is written as writeValue writes it, keeping the
// source line's preceding lines, indentation and terminator, so an edit
// changes the line and nothing around it.
//
// Each source line is written at most once. A second line carrying number n
// (a cloned or copied tag or record, which keeps its LineNumber) is a new
// line, so a copy does not repeat its original's blank lines or borrow its
// indentation and terminator.
//
// checkLine runs either way, so a line the encoder would reject is rejected
// whether or not it has source text.
func writeSourced(w io.Writer, n, level int, xref, tag, value string, opts *EncodeOptions) error {
	src := opts.claimSourceLine(n)
	if src == nil {
		return writeValue(w, level, xref, tag, value, opts)
	}
	if err := checkLine(level, xref, tag, value); err != nil {
		return err
	}

	// Only the input's last line can lack a terminator. It is kept off the
	// trailer, which the encoder writes last; any other line needs one, or
	// the next line written would run onto it.
	ending := src.Terminator
	if ending == "" && n != opts.layout.Trailer {
		ending = opts.effectiveLineEnding()
	}

	if _, err := io.WriteString(w, src.Before); err != nil {
		return err
	}
	if src.Reads(level, xref, tag, value) {
		if _, err := io.WriteString(w, src.Text); err != nil {
			return err
		}
		_, err := io.WriteString(w, ending)
		return err
	}

	indent := src.Text[:len(src.Text)-len(strings.TrimLeftFunc(src.Text, unicode.IsSpace))]
	if _, err := io.WriteString(w, indent); err != nil {
		return err
	}
	lineOpts := *opts
	lineOpts.LineEnding = ending
	return writeValue(w, level, xref, tag, value, &lineOpts)
}

// claimSourceLine returns source line n of the layout opts writes back and
// marks it written, or returns nil when there is no layout, no such line, or
// the line was already written in this encode.
func (opts *EncodeOptions) claimSourceLine(n int) *gedcom.LayoutLine {
	if opts == nil || opts.layout == nil {
		return nil
	}
	src := opts.layout.Line(n)
	if src == nil || opts.layout.used[n] {
		return nil
	}
	opts.layout.used[n] = true
	return src
}
