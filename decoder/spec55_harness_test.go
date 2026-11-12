package decoder

// spec55_harness_test.go checks the GEDCOM 5.5/5.5.1 coverage harness itself.
//
// TestSpec55Coverage compares its output against a checked-in document, so it
// catches change but not degradation: if probe documents stopped decoding
// meaningfully, every pair would classify as raw and the fix would look like
// regenerating the document. These tests anchor the harness to facts that can
// be read directly out of the decoder, and pin the two claims the published
// report makes in prose, so degradation fails loudly instead.

import (
	"slices"
	"sort"
	"strings"
	"testing"
)

// TestSpec55SentinelIsUnknown verifies the assumption the raw (accepted) /
// raw (undiagnosed) split rests on: that the sentinel tag is not a structure
// either specification defines anywhere.
func TestSpec55SentinelIsUnknown(t *testing.T) {
	for _, edition := range spec55Editions {
		for _, pair := range loadSpec55(t, edition).pairs {
			if pair.tag == specSentinelTag {
				t.Fatalf("GEDCOM %s defines %s under %s; the sentinel tag must be a tag "+
					"no decoder could recognize", edition.version, specSentinelTag,
					pair.superstructure)
			}
		}
	}
}

// TestSpec55ProbesAreWellFormed checks that each probe document differs from
// its control in exactly one line, and that the line the classifier reads
// diagnostics for is that line.
func TestSpec55ProbesAreWellFormed(t *testing.T) {
	for _, edition := range spec55Editions {
		spec := loadSpec55(t, edition)
		specProbesAreWellFormed(t, spec.specGraph, spec.specForm, func(pair specPair) string {
			return edition.version + " " + pair.superstructure + "." + pair.tag
		})
	}
}

// TestSpec55AnchorStatuses pins four pairs, one per status the decoder can
// produce here, to what it visibly does with them. Each is verifiable by
// reading the cited code, so a harness that stopped measuring would fail here
// rather than quietly reporting everything as unsupported.
func TestSpec55AnchorStatuses(t *testing.T) {
	anchors := []struct {
		version        string
		superstructure string
		tag            string
		want           specStatus
		because        string
	}{
		{
			version:        "5.5.1",
			superstructure: "INDIVIDUAL_RECORD.INDI",
			tag:            "NAME",
			want:           specTyped,
			because:        "parseIndividual builds Individual.Names from it",
		},
		{
			version:        "5.5.1",
			superstructure: "INDIVIDUAL_RECORD.INDI",
			tag:            "ALIA",
			want:           specRawAccepted,
			because:        "parseIndividual lists ALIA as recognized with no typed field (issue #375)",
		},
		{
			version:        "5.5",
			superstructure: "MULTIMEDIA_RECORD.OBJE",
			tag:            "BLOB",
			want:           specRawFlagged,
			because:        "BLOB was dropped in 5.5.1 and the multimedia parser has no case for it",
		},
		{
			version:        "5.5.1",
			superstructure: "HEADER.HEAD",
			tag:            "NOTE",
			want:           specRawAccepted,
			because:        "buildHeader knows HEAD.NOTE and has no typed field for it (issue #676)",
		},
		{
			version:        "5.5.1",
			superstructure: "LDS_SPOUSE_SEALING.SLGS.STAT",
			tag:            "DATE",
			want:           specTyped,
			because:        "parseLDSOrdinance reads STAT's DATE into StatusDate (issue #740)",
		},
	}

	measured := spec55Statuses(t)
	for _, anchor := range anchors {
		status, ok := spec55StatusOf(t, measured, anchor.version,
			anchor.superstructure, anchor.tag)
		if !ok {
			continue
		}
		if status != anchor.want {
			t.Errorf("%s: %s.%s got %s, want %s (%s)", anchor.version,
				anchor.superstructure, anchor.tag, status, anchor.want, anchor.because)
		}
	}
}

// spec55StatusOf looks one pair up by superstructure and tag, and fails if that
// does not identify exactly one structure.
//
// (superstructure, tag) is not a key: the grammar's alternations reuse a tag
// under one superstructure, so NOTE_STRUCTURE.NOTE and NOTE_STRUCTURE.NOTE#2
// are both "NOTE" under the same parent. An anchor naming one of those is
// ambiguous, and the ambiguity has to be an error rather than whichever came
// last in file order.
func spec55StatusOf(t *testing.T, measured map[string]map[specPair]specStatus,
	version, superstructure, tag string) (specStatus, bool) {
	t.Helper()

	var found []specPair
	for pair := range measured[version] {
		if pair.superstructure == superstructure && pair.tag == tag {
			found = append(found, pair)
		}
	}
	switch len(found) {
	case 1:
		return measured[version][found[0]], true
	case 0:
		t.Errorf("%s: %s.%s is no longer in the specification data",
			version, superstructure, tag)
	default:
		structures := make([]string, 0, len(found))
		for _, pair := range found {
			structures = append(structures, pair.structure)
		}
		sort.Strings(structures)
		t.Errorf("%s: %s.%s names %d structures (%s); anchor one of them by structure",
			version, superstructure, tag, len(found), strings.Join(structures, ", "))
	}
	return "", false
}

// TestSpec55VersionStatementIsVisible pins the note the published report makes
// in prose: the decoder falls back to 5.5 when a document states no version,
// which would hide a 5.5 document's version statement from the typed model,
// but Header.DeclaredVersion records only a declared version, so the header
// chain that carries it reads typed in 5.5 as in 5.5.1.
//
// If the decoder stops recording the declared version, this fails -- and the
// note in the report has to be rewritten rather than left standing as a
// description of behaviour that has moved.
func TestSpec55VersionStatementIsVisible(t *testing.T) {
	// The header record, the container, and the statement itself. The first is
	// at level 0, so it has no superstructure.
	chain := []struct{ superstructure, tag string }{
		{"", "HEAD"},
		{spec55Head, "GEDC"},
		{spec55Gedc, "VERS"},
	}
	measured := spec55Statuses(t)

	for _, link := range chain {
		got55, ok := spec55StatusOf(t, measured, "5.5", link.superstructure, link.tag)
		if ok && got55 != specTyped {
			t.Errorf("5.5 %s.%s: got %s, want %s -- the report explains this column by "+
				"Header.DeclaredVersion, so a different status means the explanation is stale",
				link.superstructure, link.tag, got55, specTyped)
		}
		got551, ok := spec55StatusOf(t, measured, "5.5.1", link.superstructure, link.tag)
		if ok && got551 != specTyped {
			t.Errorf("5.5.1 %s.%s: got %s, want %s -- the report contrasts it with 5.5's "+
				"column, so a different status means the contrast is stale",
				link.superstructure, link.tag, got551, specTyped)
		}
	}
}

// spec55Statuses measures every pair of both versions once, keyed by version
// and by the whole pair. The measurement is the expensive part of these tests,
// so the anchors share one pass.
func spec55Statuses(t *testing.T) map[string]map[specPair]specStatus {
	t.Helper()

	measured := map[string]map[specPair]specStatus{}
	for _, edition := range spec55Editions {
		spec := loadSpec55(t, edition)
		prober := newSpecProber(spec.specGraph, spec.specForm)
		measured[edition.version] = map[specPair]specStatus{}

		for _, pair := range spec.pairs {
			_, status, ok := prober.measure(t, pair)
			if !ok {
				continue // reported by TestSpec55Coverage
			}
			measured[edition.version][pair] = status
		}
	}
	return measured
}

// TestSpec55StructuresAreShared checks that the transcription kept one identity
// for a structure spliced in many places, which is what makes the inventory no
// coarser than the grammar it came from. EVENT_DETAIL is the clearest case: a
// dozen event tags splice it, and DATE under all of them is one structure.
func TestSpec55StructuresAreShared(t *testing.T) {
	for _, edition := range spec55Editions {
		spec := loadSpec55(t, edition)

		superstructures := map[string]int{}
		for _, pair := range spec.pairs {
			if pair.structure == "EVENT_DETAIL.DATE" {
				superstructures[pair.superstructure]++
			}
		}
		if len(superstructures) < 2 {
			t.Errorf("%s: EVENT_DETAIL.DATE has %d superstructures, want several -- a "+
				"structure spliced in many places must stay one structure",
				edition.version, len(superstructures))
		}
		for superstructure, count := range superstructures {
			if count != 1 {
				t.Errorf("%s: EVENT_DETAIL.DATE appears %d times under %s; a pair must "+
					"be listed once", edition.version, count, superstructure)
			}
		}
	}
}

// TestSpec55VersionsDiffer checks that the two transcriptions are not the same
// file read twice: 5.5.1's additions have to be present in 5.5.1 and absent
// from 5.5, or a copy-paste in the transcription would go unnoticed and the
// version columns would be decorative.
func TestSpec55VersionsDiffer(t *testing.T) {
	// Tags 5.5.1 introduced, and the one it removed. Each is stated in 5.5.1's
	// own "Modifications in Version 5.5.1" chapter.
	added := []string{"EMAIL", "FAX", "WWW", "MAP", "LATI", "LONG", "FONE", "ROMN", "FACT"}
	const removed = "BLOB"

	tags := map[string]map[string]bool{}
	for _, edition := range spec55Editions {
		tags[edition.version] = map[string]bool{}
		for _, pair := range loadSpec55(t, edition).pairs {
			tags[edition.version][pair.tag] = true
		}
	}

	for _, tag := range added {
		if !tags["5.5.1"][tag] {
			t.Errorf("5.5.1 does not define %s, which it introduced", tag)
		}
		if tags["5.5"][tag] {
			t.Errorf("5.5 defines %s, which was introduced in 5.5.1", tag)
		}
	}
	if !tags["5.5"][removed] {
		t.Errorf("5.5 does not define %s, which it has and 5.5.1 dropped", removed)
	}
	if tags["5.5.1"][removed] {
		t.Errorf("5.5.1 defines %s, which it dropped", removed)
	}
}

// TestSpec55DefectsAreRecorded checks that the transcription's record of what
// is wrong with the specification text survived into the harness, since the
// published report is where that record is meant to be readable. It is a
// property of the printed documents, so it cannot become empty by any change
// on this side.
func TestSpec55DefectsAreRecorded(t *testing.T) {
	for _, edition := range spec55Editions {
		spec := loadSpec55(t, edition)
		if len(spec.defects) == 0 {
			t.Errorf("%s: no specification defects recorded; both documents have "+
				"several, so an empty list means the transcription lost them",
				edition.version)
			continue
		}
		for _, d := range spec.defects {
			if d.block == "" || d.line == "" || d.problem == "" || d.reading == "" {
				t.Errorf("%s: incomplete defect %+v; each needs the production, the "+
					"line as printed, what is wrong, and what was read instead",
					edition.version, d)
			}
		}
	}
}

// TestSpec55EnumerationsAreTranscribed checks the value sets the primitive
// definitions print, in the shapes that once defeated the transcription: lists
// that wrap, literals carrying "-" or "/", alternatives that are placeholders
// rather than literals, a table split across a page, values defined only in
// prose, and a list the prose beneath it contradicts. A definition that only
// looks like a list -- a syntax, as DATE and AGE_AT_EVENT are -- must read as
// none. Values are compared whole, in order, so an extra, duplicated or
// misattributed value fails as surely as a missing one.
func TestSpec55EnumerationsAreTranscribed(t *testing.T) {
	split := func(s string) []string { return strings.Split(s, "|") }
	tests := []struct {
		version      string
		primitive    string
		values       []string
		placeholders string
	}{
		{"5.5.1", "SEX_VALUE", split("M|F|U"), ""},
		{"5.5", "SEX_VALUE", split("M|F"), ""},
		{"5.5.1", "SOURCE_MEDIA_TYPE", split("audio|book|card|electronic|fiche|film|" +
			"magazine|manuscript|map|newspaper|photo|tombstone|video"), ""},
		{"5.5.1", "LDS_BAPTISM_DATE_STATUS", split("CHILD|COMPLETED|EXCLUDED|PRE-1970|" +
			"STILLBORN|SUBMITTED|UNCLEARED"), ""},
		{"5.5.1", "LDS_CHILD_SEALING_DATE_STATUS", split("BIC|COMPLETED|EXCLUDED|DNS|" +
			"PRE-1970|STILLBORN|SUBMITTED|UNCLEARED"), ""},
		// INFANT is defined beneath the list but omitted from it; it is
		// appended, and recorded in defects.tsv.
		{"5.5.1", "LDS_ENDOWMENT_DATE_STATUS", split("CHILD|COMPLETED|EXCLUDED|PRE-1970|" +
			"STILLBORN|SUBMITTED|UNCLEARED|INFANT"), ""},
		{"5.5.1", "LDS_SPOUSE_SEALING_DATE_STATUS", split("CANCELED|COMPLETED|DNS|" +
			"EXCLUDED|DNS/CAN|PRE-1970|SUBMITTED|UNCLEARED"), ""},
		{"5.5.1", "CHARACTER_SET", split("ANSEL|UTF-8|UNICODE|ASCII"), ""},
		{"5.5.1", "GEDCOM_FORM", split("LINEAGE-LINKED"), ""},
		{"5.5.1", "MONTH", split("JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC"), ""},
		{"5.5.1", "MONTH_FREN", split("VEND|BRUM|FRIM|NIVO|PLUV|VENT|GERM|FLOR|PRAI|" +
			"MESS|THER|FRUC|COMP"), ""}, //nolint:misspell // THER is the GEDCOM code for Thermidor
		{"5.5.1", "MONTH_HEBR", split("TSH|CSH|KSL|TVT|SHV|ADR|ADS|NSN|IYR|SVN|TMZ|AAV|ELL"), ""},
		{"5.5.1", "DATE_CALENDAR_ESCAPE", split("@#DHEBREW@|@#DROMAN@|@#DFRENCH R@|" +
			"@#DGREGORIAN@|@#DJULIAN@|@#DUNKNOWN@"), ""},
		{"5.5.1", "NAME_TYPE", split("aka|birth|immigrant|maiden|married"), "<user defined>"},
		{"5.5.1", "PHONETIC_TYPE", split("hangul|kana"), "<user defined>"},
		{"5.5.1", "ROMANIZED_TYPE", split("pinyin|romaji|wadegiles"), "<user defined>"},
		{"5.5.1", "ROLE_IN_EVENT", split("CHIL|HUSB|WIFE|MOTH|FATH|SPOU"), "(<ROLE_DESCRIPTOR>)"},
		{"5.5", "ROLE_IN_EVENT", split("CHIL|HUSB|WIFE|MOTH|FATH|SPOU"), "(<ROLE_DESCRIPTOR>)"},
		{"5.5.1", "LANGUAGE_OF_TEXT", nil, "<LANGUAGE_ID>"},
		{"5.5.1", "LANGUAGE_PREFERENCE", nil, "<LANGUAGE_ID>"},
		{"5.5.1", "DATE", nil, ""},
		{"5.5.1", "AGE_AT_EVENT", nil, ""},
	}

	rows := map[string]map[string][]string{}
	for _, edition := range spec55Editions {
		rows[edition.version] = map[string][]string{}
		for _, row := range specTSV(t, edition.dir, "primitives.tsv",
			"primitive", "size", "values", "placeholders") {
			rows[edition.version][row[0]] = row
		}
	}
	valuesOf := func(row []string) []string {
		if row[2] == "" {
			return nil
		}
		return strings.Split(row[2], "|")
	}

	for _, tt := range tests {
		row, ok := rows[tt.version][tt.primitive]
		if !ok {
			t.Errorf("%s: %s is not transcribed", tt.version, tt.primitive)
			continue
		}
		if got := valuesOf(row); !slices.Equal(got, tt.values) {
			t.Errorf("%s: %s values = %q, want %q", tt.version, tt.primitive, got, tt.values)
		}
		if row[3] != tt.placeholders {
			t.Errorf("%s: %s placeholders = %q, want %q", tt.version, tt.primitive, row[3], tt.placeholders)
		}
	}

	// LANGUAGE_ID's table is split across a page: both halves must be read,
	// joined in order, with nothing lost at the break.
	if row, ok := rows["5.5.1"]["LANGUAGE_ID"]; !ok {
		t.Error("5.5.1: LANGUAGE_ID is not transcribed")
	} else {
		values := valuesOf(row)
		if len(values) != 86 || values[0] != "Afrikaans" || values[len(values)-1] != "Yiddish" {
			t.Errorf("5.5.1: LANGUAGE_ID values = %q, want 86 from Afrikaans to Yiddish", values)
		}
		if i := slices.Index(values, "Wendic"); i < 0 || i+1 >= len(values) || values[i+1] != "Amharic" {
			t.Errorf("5.5.1: LANGUAGE_ID does not join the page break (Wendic, Amharic): %q", values)
		}
	}

	// Every value of every primitive is distinct, and none is also read as a
	// placeholder.
	for version, byName := range rows {
		for name, row := range byName {
			seen := map[string]bool{}
			for _, v := range valuesOf(row) {
				if seen[v] {
					t.Errorf("%s: %s repeats value %q", version, name, v)
				}
				seen[v] = true
				if row[3] != "" && slices.Contains(strings.Split(row[3], "|"), v) {
					t.Errorf("%s: %s reads %q as both value and placeholder", version, name, v)
				}
			}
		}
	}
}
