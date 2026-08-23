package encoder

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// ErrNilDocument is returned when there is no document to encode. A nil
// Document is a missing argument rather than an empty one: it carries no header
// to write and no records to skip, so nothing -- not even the "0 HEAD" and
// "0 TRLR" of an empty document -- can be written for it.
var ErrNilDocument = errors.New("document is nil")

// Encode writes a GEDCOM document to a writer.
//
// Returns ErrNilDocument if doc is nil. See the package documentation for how
// nil values inside a document are handled.
func Encode(w io.Writer, doc *gedcom.Document) error {
	return EncodeWithOptions(w, doc, DefaultOptions())
}

// EncodeWithOptions writes a GEDCOM document with custom options.
//
// Returns ErrNilDocument if doc is nil. See the package documentation for how
// nil values inside a document are handled.
func EncodeWithOptions(w io.Writer, doc *gedcom.Document, opts *EncodeOptions) error {
	if doc == nil {
		return ErrNilDocument
	}

	if opts == nil {
		opts = DefaultOptions()
	}

	// Write header
	if err := writeHeader(w, doc.Header, opts); err != nil {
		return err
	}

	// Write records
	for _, record := range doc.Records {
		if err := writeRecord(w, record, opts); err != nil {
			return err
		}
	}

	// Write trailer
	if err := writeTrailer(w, opts); err != nil {
		return err
	}

	return nil
}

// writtenEncoding is the charset every Encode call actually produces. The
// encoder converts nothing on the way out, so this is what the CHAR line has to
// declare regardless of what the source file said -- a header claiming ANSEL
// over UTF-8 bytes is a file this library cannot read back (issue #425).
//
// GEDCOM 5.5 has no legal token for UTF-8 (ANSEL, ASCII or UNICODE only), and
// UNICODE conventionally means UTF-16 to other tools, so declaring UTF-8
// everywhere is the reading that matches the bytes. compatibility.md already
// records "input ANSEL -> output UTF-8" as an acceptable normalization.
const writtenEncoding = gedcom.EncodingUTF8

func writeHeader(w io.Writer, header *gedcom.Header, opts *EncodeOptions) error {
	// A document assembled in memory may carry no header at all. Encoding it as
	// an empty one keeps every header field on a single code path, so options
	// like TargetVersion still apply.
	if header == nil {
		header = &gedcom.Header{}
	}

	if _, err := fmt.Fprintf(w, "0 HEAD%s", opts.LineEnding); err != nil {
		return err
	}

	// A decoded document carries every header sub-tag in Tags, so writing those
	// preserves the header the source actually had -- SCHMA, SUBM, DEST, DATE,
	// COPR, the SOUR subtree and header NOTEs all reach the output, where
	// rebuilding from four scalar fields discarded them (issue #429).
	if len(header.Tags) > 0 {
		return writeHeaderTags(w, header.Tags, opts)
	}

	return writeHeaderFields(w, header, opts)
}

// writeHeaderTags writes a header from its raw tags, overriding the two values
// the encoder owns rather than the document: the charset it actually wrote, and
// the version it was asked to target.
func writeHeaderTags(w io.Writer, tags []*gedcom.Tag, opts *EncodeOptions) error {
	// inGEDC tracks the parent structure because VERS is not unique: it appears
	// under GEDC as the specification version and under SOUR as the source
	// system's own version (comprehensive.ged has "1 SOUR FamilyTreeMaker /
	// 2 VERS 16.0"). Overriding by tag name alone would rewrite the source
	// system's version to a GEDCOM version.
	inGEDC := false
	wroteVersion := false

	filtered := filterTags(tags, opts.PreserveUnknownTags)

	for i, tag := range filtered {
		if tag == nil {
			continue
		}

		if tag.Level <= 1 {
			inGEDC = tag.Level == 1 && tag.Tag == "GEDC"
		}

		if err := writeTag(w, overrideHeaderTag(tag, inGEDC, opts), opts); err != nil {
			return err
		}

		wrote, err := writeRetargetedVersion(w, filtered, i, inGEDC, opts)
		if err != nil {
			return err
		}
		if wrote || isGEDCVersion(tag, inGEDC) {
			wroteVersion = true
		}
	}

	// No GEDC at all in the source -- royal92.ged is such a file. Preserving
	// that absence is right for a plain re-encode, but a caller who asked for a
	// specific version must get a document that declares it.
	if opts.TargetVersion != "" && !wroteVersion {
		return writeGEDCVersion(w, opts)
	}

	return nil
}

// isGEDCVersion reports whether tag is the specification version, as opposed to
// a source system's own VERS.
func isGEDCVersion(tag *gedcom.Tag, inGEDC bool) bool {
	return inGEDC && tag.Level == 2 && tag.Tag == "VERS"
}

// overrideHeaderTag returns the tag to write: the original, or a copy carrying
// the value the encoder owns. Never the original mutated -- encoding a document
// must not edit it.
func overrideHeaderTag(tag *gedcom.Tag, inGEDC bool, opts *EncodeOptions) *gedcom.Tag {
	override, ok := headerTagOverride(tag, inGEDC, opts)
	if !ok {
		return tag
	}
	replaced := *tag
	replaced.Value = override
	return &replaced
}

// writeRetargetedVersion supplies the VERS a retarget needs when the source's
// GEDC has none of its own to override. Writing it here rather than appending
// at the end keeps it inside the GEDC structure it belongs to.
func writeRetargetedVersion(w io.Writer, tags []*gedcom.Tag, i int, inGEDC bool, opts *EncodeOptions) (bool, error) {
	if !inGEDC || tags[i].Level != 1 || opts.TargetVersion == "" || hasVersionChild(tags, i) {
		return false, nil
	}
	err := writeTag(w, &gedcom.Tag{Level: 2, Tag: "VERS", Value: string(opts.TargetVersion)}, opts)
	return err == nil, err
}

// writeGEDCVersion writes a complete GEDC structure for the target version.
func writeGEDCVersion(w io.Writer, opts *EncodeOptions) error {
	if err := writeTag(w, &gedcom.Tag{Level: 1, Tag: "GEDC"}, opts); err != nil {
		return err
	}
	return writeTag(w, &gedcom.Tag{Level: 2, Tag: "VERS", Value: string(opts.TargetVersion)}, opts)
}

// hasVersionChild reports whether the GEDC tag at index i is followed by its
// own VERS subtag.
func hasVersionChild(tags []*gedcom.Tag, i int) bool {
	for _, tag := range tags[i+1:] {
		if tag == nil {
			continue
		}
		// Anything back at GEDC's level or above ends the structure.
		if tag.Level <= 1 {
			return false
		}
		if tag.Level == 2 && tag.Tag == "VERS" {
			return true
		}
	}
	return false
}

// headerTagOverride reports the value the encoder must substitute for a header
// tag, if any. Everything it does not claim is written through verbatim --
// including a GEDC.VERS the library does not model, such as 555SAMPLE.GED's
// "5.5.5", which reconstruction used to silently normalize to 5.5.1.
func headerTagOverride(tag *gedcom.Tag, inGEDC bool, opts *EncodeOptions) (string, bool) {
	if tag.Level == 1 && tag.Tag == "CHAR" {
		return string(writtenEncoding), true
	}
	if inGEDC && tag.Level == 2 && tag.Tag == "VERS" && opts.TargetVersion != "" {
		return string(opts.TargetVersion), true
	}
	return "", false
}

// writeHeaderFields reconstructs a header from the typed fields. This is the
// hand-built path: a document assembled in memory has no raw tags to preserve.
func writeHeaderFields(w io.Writer, header *gedcom.Header, opts *EncodeOptions) error {
	// Use TargetVersion if set, otherwise use header.Version
	version := header.Version
	if opts.TargetVersion != "" {
		version = opts.TargetVersion
	}

	if version != "" {
		if _, err := fmt.Fprintf(w, "1 GEDC%s", opts.LineEnding); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "2 VERS %s%s", version, opts.LineEnding); err != nil {
			return err
		}
	}

	// Declared only when the header declares one at all: GEDCOM 7.0 removed
	// CHAR, and synthesizing one there would add a line the source never had.
	if header.Encoding != "" {
		if _, err := fmt.Fprintf(w, "1 CHAR %s%s", writtenEncoding, opts.LineEnding); err != nil {
			return err
		}
	}

	if header.SourceSystem != "" {
		if _, err := fmt.Fprintf(w, "1 SOUR %s%s", header.SourceSystem, opts.LineEnding); err != nil {
			return err
		}
	}

	if header.Language != "" {
		if _, err := fmt.Fprintf(w, "1 LANG %s%s", header.Language, opts.LineEnding); err != nil {
			return err
		}
	}

	return nil
}

func writeRecord(w io.Writer, record *gedcom.Record, opts *EncodeOptions) error {
	if record == nil {
		return nil
	}

	// Determine the tags to write and the level-0 line value together:
	// - If record.Tags has content, use those (preserves lossless behavior) and the
	//   stored record.Value.
	// - If record.Tags is empty/nil but Entity is set, convert the entity to tags.
	//   Some records (NOTE, SNOTE) carry text on the level-0 line; when record.Value
	//   is empty, derive that value (and, for SNOTE, its CONT/CONC continuation) from
	//   the entity so a hand-built note's text is not lost.
	tags := record.Tags
	value := record.Value
	if len(tags) == 0 && record.Entity != nil {
		tags = entityToTags(record, opts)
		if value == "" {
			var contTags []*gedcom.Tag
			value, contTags = entityRecordText(record, opts)
			tags = append(contTags, tags...)
		}
	}

	// Write record line
	if record.XRef != "" {
		if value != "" {
			if _, err := fmt.Fprintf(w, "0 %s %s %s%s", record.XRef, record.Type, value, opts.LineEnding); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(w, "0 %s %s%s", record.XRef, record.Type, opts.LineEnding); err != nil {
				return err
			}
		}
	} else {
		if _, err := fmt.Fprintf(w, "0 %s%s", record.Type, opts.LineEnding); err != nil {
			return err
		}
	}

	// Filter out custom tags if PreserveUnknownTags is false
	tags = filterTags(tags, opts.PreserveUnknownTags)

	// Write tags
	for _, tag := range tags {
		if err := writeTag(w, tag, opts); err != nil {
			return err
		}
	}

	return nil
}

func writeTag(w io.Writer, tag *gedcom.Tag, opts *EncodeOptions) error {
	if tag == nil {
		return nil
	}

	// A subordinate tag normally carries no XRef, so prefix is just the level
	// and output is unchanged for valid GEDCOM. A subordinate XRef exists only
	// when the decoder recovered a malformed identifier from the source line
	// (e.g. "1 @I 1@ NOTE"); write it back verbatim so the document round-trips
	// losslessly. The re-encoded line stays malformed, exactly as the input was.
	prefix := strconv.Itoa(tag.Level)
	if tag.XRef != "" {
		prefix += " " + tag.XRef
	}

	// A value carrying line breaks has to be split back across CONT lines. The
	// converter embeds newlines when it consolidates CONT for 7.0, and a
	// hand-built Tag may hold them too; writing such a value inline would emit
	// a continuation with no level and no tag, which this library cannot read
	// back. CONT is the encoding for a line break in every GEDCOM version,
	// including 7.0 -- 7.0 removed CONC, not CONT.
	//
	// Long lines are deliberately NOT split with CONC here: an over-length line
	// is still parseable, CONC is invalid in 7.0, and writeTag has no target
	// version to decide with. Length-based splitting stays on the typed-entity
	// path (textToTags), which does know the options.
	//
	// A line break in a value at level MaxNestingDepth-1 (99) cannot be encoded
	// at all: its CONT would sit at level 100, past the two-digit ceiling the
	// grammar allows. The CONT is still written, because an over-level line is
	// at least reported as INVALID_LEVEL on re-read, where the bare
	// continuation this replaces produced a misleading SYNTAX_ERROR against the
	// following line. Neither is valid GEDCOM -- such a document is
	// un-encodable -- and textToTags has always had the same limit.
	// Almost every value is a single line, so that case must not pay for the
	// split: writeTag runs for every subordinate line of every record, and
	// allocating a one-element slice here cost ~8000 allocations and 21% on
	// BenchmarkEncodeLarge, well past the 10% gate in make perf-regression.
	// IndexByte rather than ContainsAny: the two-byte set makes ContainsAny
	// build an asciiSet and scan in Go, while IndexByte is assembly. The write
	// is inlined rather than delegated so the common path is exactly the two
	// Fprintf calls it was before.
	if strings.IndexByte(tag.Value, '\n') < 0 && strings.IndexByte(tag.Value, '\r') < 0 {
		if tag.Value != "" {
			_, err := fmt.Fprintf(w, "%s %s %s%s", prefix, tag.Tag, tag.Value, opts.LineEnding)
			return err
		}
		_, err := fmt.Fprintf(w, "%s %s%s", prefix, tag.Tag, opts.LineEnding)
		return err
	}

	lines := splitValueLines(tag.Value)

	if err := writeTagLine(w, prefix, tag.Tag, lines[0], opts); err != nil {
		return err
	}
	contPrefix := strconv.Itoa(tag.Level + 1)
	for _, cont := range lines[1:] {
		if err := writeTagLine(w, contPrefix, "CONT", cont, opts); err != nil {
			return err
		}
	}
	return nil
}

// splitValueLines splits a tag value on line breaks, returning at least one
// element. All three terminators the parser recognises are normalised first --
// CRLF, bare LF and bare CR -- so that re-encoding is the exact inverse of
// reading: whichever form the source used, the parser saw a line break there,
// and a CONT is how that line break is written back.
// Callers on the hot path should skip it entirely for a value with no line
// break rather than rely on it returning a single element, so that the common
// case allocates nothing.
func splitValueLines(value string) []string {
	if !strings.ContainsAny(value, "\n\r") {
		return []string{value}
	}
	normalized := strings.ReplaceAll(value, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return strings.Split(normalized, "\n")
}

// writeTagLine writes one physical GEDCOM line. prefix carries the level and,
// for a recovered malformed identifier, the XRef.
func writeTagLine(w io.Writer, prefix, tag, value string, opts *EncodeOptions) error {
	var err error
	if value != "" {
		_, err = fmt.Fprintf(w, "%s %s %s%s", prefix, tag, value, opts.LineEnding)
	} else {
		_, err = fmt.Fprintf(w, "%s %s%s", prefix, tag, opts.LineEnding)
	}
	return err
}

func writeTrailer(w io.Writer, opts *EncodeOptions) error {
	_, err := fmt.Fprintf(w, "0 TRLR%s", opts.LineEnding)
	return err
}

// isCustomTag returns true if the tag name is a custom/extension tag.
// Custom tags are underscore-prefixed by convention (e.g., _CUSTOM, _UID).
func isCustomTag(tagName string) bool {
	return strings.HasPrefix(tagName, "_")
}

// filterTags returns tags with custom tags filtered out if PreserveUnknownTags is false.
// When a custom tag is filtered, its child tags (higher level) are also removed.
func filterTags(tags []*gedcom.Tag, preserveUnknown bool) []*gedcom.Tag {
	if preserveUnknown {
		return tags
	}

	result := make([]*gedcom.Tag, 0, len(tags))
	skipUntilLevel := -1 // -1 means not skipping

	for _, tag := range tags {
		// A nil tag writes nothing, so drop it here rather than let it decide
		// whether a custom tag's children are still being skipped.
		if tag == nil {
			continue
		}

		// If we're skipping and encounter a tag at same or lower level, stop skipping
		if skipUntilLevel >= 0 && tag.Level <= skipUntilLevel {
			skipUntilLevel = -1
		}

		// If still skipping, continue
		if skipUntilLevel >= 0 {
			continue
		}

		// Check if this tag should be skipped
		if isCustomTag(tag.Tag) {
			skipUntilLevel = tag.Level
			continue
		}

		result = append(result, tag)
	}

	return result
}
