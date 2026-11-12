package encoder

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/headergrammar"
)

// ErrNilDocument is returned when there is no document to encode. A nil
// Document is a missing argument rather than an empty one: it carries no header
// to write and no records to skip, so nothing -- not even the "0 HEAD" and
// "0 TRLR" of an empty document -- can be written for it.
var ErrNilDocument = errors.New("document is nil")

// ErrLineBreak is returned when a tag name or cross-reference identifier --
// [gedcom.Tag].Tag, [gedcom.Tag].XRef, [gedcom.Record].Type,
// [gedcom.Record].XRef or [gedcom.Header].Submitter, including a tag name
// derived from a typed entity field such as [gedcom.Event].Type -- contains a
// line break (CR or LF). GEDCOM has no way to spell one there, and writing it
// through would start a new physical line that a reader could take for a
// record the caller never wrote. A line break in a value is not an error: it
// is written as CONT.
var ErrLineBreak = errors.New("line break in tag or cross-reference identifier")

// ErrMalformedLine is returned when a line's level, tag name or record
// identifier is one the decoder cannot read back as written (issue #726):
//
//   - a [gedcom.Tag] below a record or header with a Level outside 1-99 --
//     level 0 would open a new record mid-record, and the parser rejects a
//     negative level or one past 99 -- or at 99 with a line break in its
//     Value, whose CONT would sit at level 100;
//   - an empty tag name, or one containing whitespace, in [gedcom.Tag].Tag or
//     [gedcom.Record].Type -- the parser splits a line on whitespace, so the
//     tag would be read back shifted into the value;
//   - a [gedcom.Record].XRef that does not open with "@" -- "0 I1 INDI" is
//     read back as a record of type I1 -- or a [gedcom.Tag].XRef not closed
//     by "@" at both ends;
//   - an identifier holding whitespace that the parser would split
//     differently: one with a further "@" inside ("@A@ SUBM @B@" reads as
//     "@A@" of type SUBM), with no ASCII space among its whitespace, or
//     opening "@ " ("@ I1@" reads as "@" of type "I1@"). A 5.5 spaced
//     identifier such as "@I 1@" reads back whole and is accepted.
//
// A record identifier with no closing "@" is accepted only in the shape the
// decoder itself recovers from such input ("@I1", no whitespace and no
// further "@"), and a spaced identifier standing alone as a record's Type
// ("0 @I 1@") only with no value, so a leniently decoded document still
// re-encodes byte for byte (issue #385).
//
// The error names the record (or HEAD) and the line it was found on.
var ErrMalformedLine = errors.New("level, tag name or record identifier the decoder cannot read back")

// ErrInvalidUTF8 is returned when a value, tag name or identifier is not
// valid UTF-8 (issue #726). The encoder converts nothing and declares UTF-8
// (see [gedcom.EncodingUTF8]), so such bytes would make the output a file the
// decoder rejects. Convert strings taken from a legacy source to UTF-8 before
// encoding, or replace the bad bytes with [strings.ToValidUTF8]. The error
// names the record (or HEAD) and the line it was found on. It is distinct
// from charset.ErrInvalidUTF8Sequence, which reports invalid input bytes while
// decoding; neither matches the other.
var ErrInvalidUTF8 = errors.New("value is not valid UTF-8")

// maxLevel is the deepest level the parser accepts (parser.MaxNestingDepth-1).
const maxLevel = 99

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
	opts = opts.forOutput(doc.Header)

	// Write header
	if err := writeHeader(w, doc.Header, opts); err != nil {
		return err
	}

	// Write records
	sharedNotes := sharedNotePointers(doc, opts)
	for _, record := range doc.Records {
		if err := opts.cancelled(); err != nil {
			return err
		}
		if err := writeRecord(w, record, opts, sharedNotes); err != nil {
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

	if _, err := fmt.Fprintf(w, "0 HEAD%s", opts.effectiveLineEnding()); err != nil {
		return err
	}

	// A decoded document carries every header sub-tag in Tags, so writing those
	// preserves the header the source actually had -- SCHMA, SUBM, DEST, DATE,
	// COPR, the SOUR subtree and header NOTEs all reach the output, where
	// rebuilding from four scalar fields discarded them (issue #429).
	if len(header.Tags) > 0 {
		return inRecord("HEAD", writeHeaderTags(w, header.Tags, opts))
	}

	return inRecord("HEAD", writeHeaderFields(w, header, opts))
}

// inRecord prefixes an input error -- one of [ErrLineBreak],
// [ErrMalformedLine] or [ErrInvalidUTF8] -- with the record it was found in,
// so a caller encoding thousands of records can find the one at fault. A write
// error is returned as is: it says nothing about the record.
func inRecord(record string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrLineBreak) || errors.Is(err, ErrMalformedLine) || errors.Is(err, ErrInvalidUTF8) {
		return fmt.Errorf("record %q: %w", record, err)
	}
	return err
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
	//
	// converter.updateVersionTag walks header tags for the same structure and
	// must agree with this on what "inside GEDC" means. The two are deliberately
	// not shared: that one edits the document so a conversion is visible to
	// every reader of it, while this one substitutes at write time and must
	// leave the caller's document untouched. Change one, check the other.
	inGEDC := false

	filtered := filterTags(tags, opts.DropUnknownTags)

	// No GEDC at all in the source -- royal92.ged is such a file. Preserving
	// that absence is right for a plain re-encode, but a caller who asked for a
	// specific version must get a document that declares it. The synthesized
	// GEDC goes where the target grammar puts it: before the first level-1
	// structure ranked after it, not at the end of the header (#465).
	// Only consulted when a target is set, where opts.version is that target as
	// the version gates read it (7.0.N and 7.1 rank as 7.0).
	needGEDC := opts.TargetVersion != "" && !hasGEDC(filtered)
	gedcRank := headergrammar.Rank(opts.version, "GEDC")

	for i, tag := range filtered {
		if tag == nil {
			continue
		}

		if tag.Level <= 1 {
			inGEDC = tag.Level == 1 && tag.Tag == "GEDC"
			if needGEDC && headergrammar.Rank(opts.version, tag.Tag) > gedcRank {
				if err := writeGEDCVersion(w, opts); err != nil {
					return err
				}
				needGEDC = false
			}
		}

		if err := writeTag(w, overrideHeaderTag(tag, inGEDC, opts), opts); err != nil {
			return err
		}

		if err := writeRetargetedVersion(w, filtered, i, inGEDC, opts); err != nil {
			return err
		}

		if err := writeRetargetedForm(w, filtered, i, inGEDC, opts); err != nil {
			return err
		}
	}

	// Every structure present ranks before GEDC, so it goes last.
	if needGEDC {
		return writeGEDCVersion(w, opts)
	}

	return nil
}

// hasGEDC reports whether tags carry a level-1 GEDC structure, by the same
// test the write loop uses for "inside GEDC".
func hasGEDC(tags []*gedcom.Tag) bool {
	for _, tag := range tags {
		if tag != nil && tag.Level == 1 && tag.Tag == "GEDC" {
			return true
		}
	}
	return false
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
func writeRetargetedVersion(w io.Writer, tags []*gedcom.Tag, i int, inGEDC bool, opts *EncodeOptions) error {
	if !inGEDC || tags[i].Level != 1 || opts.TargetVersion == "" || hasGEDCChild(tags, i, "VERS") {
		return nil
	}
	return writeTag(w, &gedcom.Tag{Level: 2, Tag: "VERS", Value: string(opts.TargetVersion)}, opts)
}

// writeRetargetedForm supplies the GEDC.FORM 5.5 and 5.5.1 require when the
// source's GEDC has none, such as a 7.0 header retargeted without the
// converter (#674). It goes after the GEDC structure's last line, so after VERS
// as the 5.x grammar orders them.
func writeRetargetedForm(w io.Writer, tags []*gedcom.Tag, i int, inGEDC bool, opts *EncodeOptions) error {
	if !inGEDC || !headergrammar.RequiresGEDCForm(opts.TargetVersion) || !endsStructure(tags, i) {
		return nil
	}
	gedc := i
	for tags[gedc] == nil || tags[gedc].Level > 1 {
		gedc--
	}
	if hasGEDCChild(tags, gedc, "FORM") {
		return nil
	}
	return writeGEDCForm(w, opts)
}

// endsStructure reports whether the tag at index i is the last line of its
// level-1 structure: nothing but nils follows it before the next tag at level 1
// or above, or the end of the header.
func endsStructure(tags []*gedcom.Tag, i int) bool {
	for _, tag := range tags[i+1:] {
		if tag != nil {
			return tag.Level <= 1
		}
	}
	return true
}

// writeGEDCVersion writes a complete GEDC structure for the target version:
// VERS, and for 5.5 and 5.5.1 the FORM those versions require (#465).
func writeGEDCVersion(w io.Writer, opts *EncodeOptions) error {
	if err := writeTag(w, &gedcom.Tag{Level: 1, Tag: "GEDC"}, opts); err != nil {
		return err
	}
	if err := writeTag(w, &gedcom.Tag{Level: 2, Tag: "VERS", Value: string(opts.TargetVersion)}, opts); err != nil {
		return err
	}
	if !headergrammar.RequiresGEDCForm(opts.TargetVersion) {
		return nil
	}
	return writeGEDCForm(w, opts)
}

// writeGEDCForm writes the GEDC.FORM 5.5 and 5.5.1 require.
func writeGEDCForm(w io.Writer, opts *EncodeOptions) error {
	return writeTag(w, &gedcom.Tag{Level: 2, Tag: "FORM", Value: headergrammar.GEDCForm}, opts)
}

// hasGEDCChild reports whether the GEDC tag at index i is followed by its own
// level-2 subtag named name.
func hasGEDCChild(tags []*gedcom.Tag, i int, name string) bool {
	for _, tag := range tags[i+1:] {
		if tag == nil {
			continue
		}
		// Anything back at GEDC's level or above ends the structure.
		if tag.Level <= 1 {
			return false
		}
		if tag.Level == 2 && tag.Tag == name {
			return true
		}
	}
	return false
}

// headerTagOverride reports the value the encoder must substitute for a header
// tag, if any. Everything it does not claim is written through verbatim --
// including a GEDC.VERS the library does not model, such as 555SAMPLE.GED's
// "5.5.5", which reconstruction used to silently normalize to 5.5.1.
// CHAR is claimed at level 1 only, the one level the decoder reads
// Header.Encoding from (#464).
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
	tags, err := headerFieldTags(header, opts)
	if err != nil {
		return err
	}
	for _, tag := range tags {
		if err := writeTag(w, tag, opts); err != nil {
			return err
		}
	}
	return nil
}

// headerFieldTags returns the level-1 header tags the typed fields are written
// as. writeHeaderFields writes them and SyncTags compares them, so both agree
// on what a header field encodes to.
//
// VERS, SOUR, DATE and LANG are values, so each is written through writeValue
// rather than a bare Fprintf: a line break in a hand-built value (say, one
// taken from user input) becomes a CONT line instead of starting a new GEDCOM
// line -- a raw "\n0 @X@ INDI" would otherwise inject a record into the file
// (issue #466).
func headerFieldTags(header *gedcom.Header, opts *EncodeOptions) ([]*gedcom.Tag, error) {
	version := outputVersion(header, opts)

	var gedc []*gedcom.Tag
	if version != "" {
		gedc = append(gedc,
			&gedcom.Tag{Level: 1, Tag: "GEDC"},
			&gedcom.Tag{Level: 2, Tag: "VERS", Value: string(version)})
		if headergrammar.RequiresGEDCForm(version) {
			gedc = append(gedc, &gedcom.Tag{Level: 2, Tag: "FORM", Value: headergrammar.GEDCForm})
		}
	}

	// optional is the line for a field, or nothing when the field is empty.
	optional := func(tag, value string) []*gedcom.Tag {
		if value == "" {
			return nil
		}
		return []*gedcom.Tag{{Level: 1, Tag: tag, Value: value}}
	}

	// CHAR is declared for every output but 7.0, whether or not the header
	// sets Encoding: a 5.x reader that finds no CHAR may assume ANSEL, the 5.x
	// default, and misread every non-ASCII byte the encoder wrote as UTF-8
	// (issue #726). GEDCOM 7.0 removed CHAR -- 7.0 is UTF-8 by definition --
	// so there it is written only when the header asks for one.
	var char []*gedcom.Tag
	if header.Encoding != "" || !opts.is70() {
		char = optional("CHAR", string(writtenEncoding))
	}

	// The SUBM pointer is written verbatim: whatever the caller put in the
	// field is the pointer the document's SUBM record is keyed by, and
	// re-delimiting it would break that link rather than repair it. A pointer
	// is an identifier, so like an XRef a line break in it has no CONT form
	// and is rejected.
	if hasLineBreak(header.Submitter) {
		return nil, fmt.Errorf("level 1 SUBM pointer %q: %w", header.Submitter, ErrLineBreak)
	}

	// SOUR, DATE and LANG are text, escaped for the output version like every
	// typed value (#708); the DATE is otherwise written as given, like every
	// other GEDCOM date. The SUBM pointer is not text.
	sour := optional("SOUR", opts.text(header.SourceSystem))
	date := optional("DATE", opts.text(header.Date))
	subm := optional("SUBM", header.Submitter)
	lang := optional("LANG", opts.text(header.Language))

	// Order is version-dependent because the two grammars disagree about where
	// GEDC sits: 5.5/5.5.1 open the header with SOUR and declare GEDC late
	// (SOUR, DEST, DATE, SUBM, SUBN, FILE, COPR, GEDC, CHAR, LANG, ...), while
	// 7.0 requires GEDC first so a reader learns the version before anything
	// else (GEDC, SCHMA, SOUR, DEST, DATE, SUBM, COPR, LANG, ...). Both put
	// DATE between SOUR and SUBM. Unknown or absent versions take the 5.5
	// order, which is what 5.5-era readers expect and what a 7.0 reader
	// tolerates anyway.
	//
	// CHAR has no place in the 7.0 grammar at all, so its position there is
	// arbitrary; it is kept in the sequence so a hand-built header that sets
	// Encoding still gets the line it asked for.
	//
	// The order reads the gated version (7.0.N and 7.1 are 7.0); VERS above keeps the
	// value as given (issue #696).
	groups := [][]*gedcom.Tag{sour, date, subm, gedc, char, lang}
	if opts.is70() {
		groups = [][]*gedcom.Tag{gedc, sour, date, subm, char, lang}
	}

	var tags []*gedcom.Tag
	for _, group := range groups {
		tags = append(tags, group...)
	}
	return tags, nil
}

// writeRecord writes record, from its preserved Tags when it has them and from
// its Entity otherwise. sharedNotes is the set from sharedNotePointers: note
// pointers in an entity-built record that resolve to one of these XRefs are
// written as SNOTE rather than NOTE.
func writeRecord(w io.Writer, record *gedcom.Record, opts *EncodeOptions, sharedNotes *sharedNoteSet) error {
	if record == nil {
		return nil
	}

	// A record whose type is itself a custom tag ("0 _ROOT", RootsMagic's
	// "0 _EVDEF", Legacy's "0 _EVENT_DEFN") is an unknown structure in its
	// entirety, so a caller who asked not to preserve unknown tags gets none of
	// it -- the level-0 line, its value and its subtree alike. filterTags
	// applies exactly this rule to a custom tag one level down; excluding the
	// record from it emitted a "0 _ROOT" stub whose children had been stripped,
	// which is neither the record nor its absence, and which no GEDCOM reader
	// can do anything with. Writing the level-0 value (issue #404) widened that
	// stub rather than creating it.
	//
	// This is the lossy mode the option exists to request; the zero value is
	// DropUnknownTags=false, where nothing here applies.
	if opts.DropUnknownTags && isCustomTag(string(record.Type)) {
		return nil
	}

	// Determine the tags to write and the level-0 line value together:
	// - If record.Tags has content, use those (preserves lossless behavior) and the
	//   stored record.Value.
	// - If record.Tags is empty/nil but Entity is set, the entity is the whole
	//   record: its typed fields become tags, and for a NOTE or SNOTE its Text
	//   becomes the level-0 value plus CONT/CONC. The text replaces record.Value
	//   whenever the entity has any: a decoded note keeps its first line in
	//   Value, and gating on an empty Value wrote that line alone, dropping
	//   every continuation of a note whose Tags were cleared (#707). Last come
	//   the entity's own extension Tags, so a hand-built record can carry a
	//   vendor tag beside its typed structures (#711); see entityOwnTags for
	//   the rule and why standard tags are never taken from there.
	tags := record.Tags
	value := record.Value
	if len(tags) == 0 && record.Entity != nil {
		value, tags = entityRecordLines(record, opts, sharedNotes)
	}

	// The level-0 line goes through the same writer as every other line. Two
	// raw Fprintf branches used to sit here and each was wrong in its own way:
	// the XRef-less one formatted only the type, so a value on a level-0 line
	// was dropped outright (issue #404), and the XRef one wrote the value
	// inline, so a line break in it emitted a continuation with no level and no
	// tag -- text beginning with a digit forged a whole record on re-read
	// (issue #421). Mirroring the XRef branch to fix the first would have handed
	// the second to XRef-less records too.
	//
	// writeValue already is the value-safe line writer that writeTag delegates
	// to: it emits the level, the XRef and the tag, and folds a line break in
	// the value into CONT at level+1, which is 1 here. Sharing it leaves one
	// place that decides what a value on a line means, rather than two that
	// drifted apart.
	if err := checkRecordLine(record); err != nil {
		return inRecord(recordName(record), err)
	}
	if err := writeValue(w, 0, record.XRef, string(record.Type), value, opts); err != nil {
		return inRecord(recordName(record), err)
	}

	// Filter out custom tags if DropUnknownTags is set
	tags = filterTags(tags, opts.DropUnknownTags)

	// Write tags
	for _, tag := range tags {
		if err := writeTag(w, tag, opts); err != nil {
			return inRecord(recordName(record), err)
		}
	}

	return nil
}

// recordName is how an error names record: its XRef and type. Built only on
// the error path, so a successful encode pays nothing for it.
func recordName(record *gedcom.Record) string {
	return strings.TrimSpace(record.XRef + " " + string(record.Type))
}

// checkRecordLine rejects a level-0 line the decoder would not read back as
// this record (issue #726). A HEAD or TRLR with no identifier is the
// document's own header or trailer to the decoder, so a record written as one
// would end the file early or stand in for the header; with an identifier it
// is an ordinary record (#416). A record identifier not opening with "@"
// becomes the record type on re-read ("0 I1 INDI" is a record of type I1). An
// identifier closed by "@" must be one the parser reads back whole (see
// closedXRefReadsBack). An identifier with no closing "@" is accepted only in
// the exact shape the parser recovers as an identifier -- no whitespace and no
// further "@" -- because that is what a lenient decode of "0 @I1 INDI" holds,
// and it must re-encode byte for byte (issue #385). A line break is left to
// writeValue, which reports it as ErrLineBreak.
func checkRecordLine(record *gedcom.Record) error {
	xref := record.XRef
	if xref == "" {
		if record.Type == "HEAD" || record.Type == "TRLR" {
			return fmt.Errorf("level 0 tag %q without an xref is not a record: %w", record.Type, ErrMalformedLine)
		}
		return nil
	}
	if hasLineBreak(xref) {
		return nil
	}
	if closedXRefReadsBack(xref) {
		return nil
	}
	if xref[0] == '@' && !strings.ContainsRune(xref[1:], '@') && !containsSpace(xref) {
		return nil
	}
	return fmt.Errorf("level 0 xref %q: %w", xref, ErrMalformedLine)
}

// entityRecordLines returns the level-0 value and the tags a record is written
// with when it is written from its Entity: the typed fields, a NOTE or SNOTE's
// Text as the value plus CONT/CONC, then the entity's own extension Tags.
// writeRecord and SyncTags both derive a record this way, so what SyncTags
// compares is exactly what the encoder would write.
func entityRecordLines(record *gedcom.Record, opts *EncodeOptions, sharedNotes *sharedNoteSet) (string, []*gedcom.Tag) {
	value := record.Value
	typed := entityToTags(record, opts)
	if text, contTags := entityRecordText(record, opts); text != "" || len(contTags) > 0 {
		value = text
		typed = append(contTags, typed...)
	}
	typed = append(typed, entityOwnTags(record.Entity)...)
	return value, retagSharedNotePointers(typed, sharedNotes)
}

// writeTag writes a line subordinate to a record or the header, so its level
// must be one the parser reads as subordinate: 0 would open a new record, and a
// negative level or one past 99 is rejected on re-read (issue #726). Level 0 is
// also Tag's zero value, so a hand-built &gedcom.Tag{Tag: "NAME"} lands here.
func writeTag(w io.Writer, tag *gedcom.Tag, opts *EncodeOptions) error {
	if tag == nil {
		return nil
	}
	if tag.Level < 1 || tag.Level > maxLevel {
		return fmt.Errorf("level %d tag %q: level must be 1-%d: %w", tag.Level, tag.Tag, maxLevel, ErrMalformedLine)
	}
	return writeValue(w, tag.Level, tag.XRef, tag.Tag, tag.Value, opts)
}

// writeValue writes a value at a given level: the line that opens it, plus a
// CONT line for each line break inside it. It is the many-lines half of the
// pair -- [writeLine] is the one that writes exactly one physical line, and
// this delegates to it once per line produced.
//
// It takes the four parts rather than a *gedcom.Tag so writeRecord can share it
// for the level-0 line without allocating a Tag to describe a record it already
// has.
func writeValue(w io.Writer, level int, xref, tag, value string, opts *EncodeOptions) error {
	// Checked here rather than in writeLine so a multi-line value pays for it
	// once, not once per CONT.
	if err := checkLine(level, xref, tag, value); err != nil {
		return err
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
	// grammar allows, which the parser rejects. It is an ErrMalformedLine, like
	// any other level the decoder cannot read back (issue #726).
	// Almost every value is a single line, so that case must not pay for the
	// split: writeValue runs for every line of every record, and allocating a
	// one-element slice here cost ~8000 allocations and 21% on
	// BenchmarkEncodeLarge, well past the 10% gate in make perf-regression.
	// IndexByte rather than ContainsAny: the two-byte set makes ContainsAny
	// build an asciiSet and scan in Go, while IndexByte is assembly.
	if strings.IndexByte(value, '\n') < 0 && strings.IndexByte(value, '\r') < 0 {
		return writeLine(w, level, xref, tag, value, opts)
	}

	if level >= maxLevel {
		return fmt.Errorf("level %d tag %q: line break in value needs a CONT at level %d, past %d: %w",
			level, tag, level+1, maxLevel, ErrMalformedLine)
	}

	lines := splitValueLines(value)

	if err := writeLine(w, level, xref, tag, lines[0], opts); err != nil {
		return err
	}
	// A CONT never carries the XRef: the identifier belongs to the line that
	// opened the structure, not to its continuations.
	for _, cont := range lines[1:] {
		if err := writeLine(w, level+1, "", "CONT", cont, opts); err != nil {
			return err
		}
	}
	return nil
}

// hasLineBreak reports whether s, a tag name or identifier, contains a CR or
// LF. It runs twice per writeValue call, on strings that are almost always a
// handful of bytes or empty, so a plain loop over both bytes is enough.
func hasLineBreak(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' || s[i] == '\r' {
			return true
		}
	}
	return false
}

// checkLine rejects a line the decoder would not read back as written:
//
//   - a line break in the tag name or identifier, which has no CONT form
//     (ErrLineBreak, issue #466);
//   - a tag name that is empty or holds whitespace, which the parser's split
//     on whitespace reads back shifted -- "1 NAME X" + " A /B/" gives tag NAME
//     and value "X A /B/" (ErrMalformedLine, issue #726);
//   - bytes that are not UTF-8 anywhere on the line: the output declares
//     UTF-8 and the decoder rejects anything else, so one string carried over
//     unconverted from a legacy charset would make the whole file unreadable
//     (ErrInvalidUTF8, issue #726).
//
// It runs for every line written, so the tag name every real line has -- a
// few ASCII letters -- is cleared by plainTagName in one pass, and the
// identifier, almost always empty, is checked only when present.
func checkLine(level int, xref, tag, value string) error {
	if !plainTagName(tag) {
		switch {
		case hasLineBreak(tag):
			return fmt.Errorf("level %d tag %q: %w", level, tag, ErrLineBreak)
		case !utf8.ValidString(tag):
			return fmt.Errorf("level %d tag %q: %w", level, strings.ToValidUTF8(tag, "\uFFFD"), ErrInvalidUTF8)
		case tag == "" || containsSpace(tag) && !tagIsLoneXRef(level, xref, tag, value):
			return fmt.Errorf("level %d tag %q: empty or contains whitespace: %w", level, tag, ErrMalformedLine)
		}
	}
	if xref != "" {
		if hasLineBreak(xref) {
			return fmt.Errorf("level %d xref %q: %w", level, xref, ErrLineBreak)
		}
		if !utf8.ValidString(xref) {
			return fmt.Errorf("level %d tag %q xref: %w", level, tag, ErrInvalidUTF8)
		}
		// Level 0 was checked by checkRecordLine, which also admits the
		// unterminated "@I1" a lenient decode recovers; below it the parser
		// recovers no such shape, so the identifier must be closed.
		if level > 0 && !closedXRefReadsBack(xref) {
			return fmt.Errorf("level %d tag %q xref %q: %w", level, tag, xref, ErrMalformedLine)
		}
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("level %d tag %q value: %w", level, tag, ErrInvalidUTF8)
	}
	return nil
}

// closedXRefReadsBack reports whether xref, delimited by "@" at both ends,
// reads back as one identifier when written before a tag. With no whitespace
// it is a single field, which the parser takes whole. With whitespace it must
// be the 5.5 spaced shape the parser recovers (splitSpacedXRef in package
// parser): no "@" inside, which would close it early ("@A@ SUBM @B@" reads as
// "@A@" of type SUBM); an ASCII space, which is what the recovery looks for;
// and a first field that is not "@" alone, which the parser takes for a whole
// identifier ("@ I1@" reads as "@" of type "I1@"). A spaced identifier is only
// checked here on the rare path, so its strings.Fields allocation is not paid
// per line.
func closedXRefReadsBack(xref string) bool {
	if len(xref) < 2 || xref[0] != '@' || xref[len(xref)-1] != '@' {
		return false
	}
	if !containsSpace(xref) {
		return true
	}
	body := xref[1 : len(xref)-1]
	return !strings.ContainsRune(body, '@') && strings.ContainsRune(body, ' ') &&
		strings.Fields(xref)[0] != "@"
}

// tagIsLoneXRef reports whether tag is a spaced identifier standing alone on a
// level-0 line, the shape a lenient decode of "0 @I 1@" holds: with no tag
// after the identifier, the parser keeps it as the record type, and it must
// re-encode byte for byte as the unterminated "@I1" does (issue #385). Any
// value after it would be read back as the record type instead.
func tagIsLoneXRef(level int, xref, tag, value string) bool {
	return level == 0 && xref == "" && value == "" && closedXRefReadsBack(tag)
}

// plainTagName reports whether tag is non-empty, ASCII and free of
// whitespace: a tag name checkLine need look at no further.
func plainTagName(tag string) bool {
	if tag == "" {
		return false
	}
	for i := 0; i < len(tag); i++ {
		switch b := tag[i]; {
		case b >= utf8.RuneSelf, b == ' ', b == '\t', b == '\n', b == '\v', b == '\f', b == '\r':
			return false
		}
	}
	return true
}

// containsSpace reports whether s holds a character the parser's field split
// (strings.Fields) treats as a separator. Tag names are a few ASCII bytes, so
// the loop decodes a rune only for a non-ASCII byte.
func containsSpace(s string) bool {
	for i := 0; i < len(s); {
		if s[i] < utf8.RuneSelf {
			if s[i] == ' ' || s[i] == '\t' || s[i] == '\v' || s[i] == '\f' || s[i] == '\n' || s[i] == '\r' {
				return true
			}
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if unicode.IsSpace(r) {
			return true
		}
		i += size
	}
	return false
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

// writeLine writes exactly one physical GEDCOM line: the level, then the
// XRef if there is one, the tag, and the value if there is one.
//
// A line normally carries no XRef, so the output is unchanged for every
// subordinate line of valid GEDCOM. A subordinate XRef exists only when the
// decoder recovered a malformed identifier from the source line (e.g.
// "1 @I 1@ NOTE"); write it back verbatim so the document round-trips
// losslessly. The re-encoded line stays malformed, exactly as the input was. At
// level 0 the XRef is the ordinary record identifier and this is how "0 @I1@
// INDI" gets written.
//
// The four shapes are spelled out rather than assembled from a level-plus-XRef
// prefix string, because formatting that prefix with %s boxes a string into an
// interface and allocates on every line. Formatting the level with %d instead
// costs nothing: a GEDCOM level is 0-99, so it comes from the runtime's static
// small-integer table. Measured against the pre-#404 encoder that is 8000 fewer
// allocations on BenchmarkEncodeLarge -- one per line, -24% -- so sharing this
// path from writeRecord lands well inside the parity issue #421 asked for.
func writeLine(w io.Writer, level int, xref, tag, value string, opts *EncodeOptions) error {
	var err error
	switch {
	case xref == "" && value == "":
		_, err = fmt.Fprintf(w, "%d %s%s", level, tag, opts.effectiveLineEnding())
	case xref == "":
		_, err = fmt.Fprintf(w, "%d %s %s%s", level, tag, value, opts.effectiveLineEnding())
	case value == "":
		_, err = fmt.Fprintf(w, "%d %s %s%s", level, xref, tag, opts.effectiveLineEnding())
	default:
		_, err = fmt.Fprintf(w, "%d %s %s %s%s", level, xref, tag, value, opts.effectiveLineEnding())
	}
	return err
}

func writeTrailer(w io.Writer, opts *EncodeOptions) error {
	_, err := fmt.Fprintf(w, "0 TRLR%s", opts.effectiveLineEnding())
	return err
}

// isCustomTag returns true if the tag name is a custom/extension tag.
// Custom tags are underscore-prefixed by convention (e.g., _CUSTOM, _UID).
func isCustomTag(tagName string) bool {
	return strings.HasPrefix(tagName, "_")
}

// filterTags returns tags with custom tags filtered out if dropUnknown is set.
// When a custom tag is filtered, its child tags (higher level) are also removed.
func filterTags(tags []*gedcom.Tag, dropUnknown bool) []*gedcom.Tag {
	if !dropUnknown {
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
