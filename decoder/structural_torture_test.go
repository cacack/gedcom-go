package decoder

import (
	"os"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// ============================================================================
// Structural torture tests (issue #301).
//
// Fixtures under testdata/edge-cases/ and testdata/malformed/ recreated from
// the GEDCOM spec's structure rules or vendored verbatim from
// gedcom7code/test-files (public domain, Unlicense). Each test asserts the
// decoder's actual behavior, not just absence of a crash.
// ============================================================================

// decodeFixture decodes a fixture in lenient mode and fails the test on error.
func decodeFixture(t *testing.T, path string) *DecodeResult {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	result, err := DecodeWithDiagnostics(f, nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	if result == nil || result.Document == nil {
		t.Fatal("DecodeWithDiagnostics() returned nil result or document")
	}
	return result
}

// decodeFixtureStrict decodes a fixture in strict mode and returns the error.
func decodeFixtureStrict(t *testing.T, path string) error {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	opts := DefaultOptions()
	opts.StrictMode = true
	_, err = DecodeWithOptions(f, opts)
	return err
}

// TestStructuralTortureFamilyVariants covers the Tamura-Jones-style family
// structure cases: children-only families, empty FAM records, self-marriage,
// and same-sex marriage encodings in a 5.5.1 context.
func TestStructuralTortureFamilyVariants(t *testing.T) {
	const path = "../testdata/edge-cases/family-structure-variants.ged"

	if err := decodeFixtureStrict(t, path); err != nil {
		t.Fatalf("strict decode should succeed, got %v", err)
	}

	result := decodeFixture(t, path)
	doc := result.Document

	if len(result.Diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d: %v", len(result.Diagnostics), result.Diagnostics)
	}

	t.Run("children-only family", func(t *testing.T) {
		fam := doc.GetFamily("@F1@")
		if fam == nil {
			t.Fatal("GetFamily(@F1@) returned nil")
		}
		if fam.Husband != "" || fam.Wife != "" {
			t.Errorf("expected no spouses, got husband=%q wife=%q", fam.Husband, fam.Wife)
		}
		want := []string{"@I1@", "@I2@"}
		if len(fam.Children) != len(want) {
			t.Fatalf("Children = %v, want %v", fam.Children, want)
		}
		for i, c := range want {
			if fam.Children[i] != c {
				t.Errorf("Children[%d] = %q, want %q", i, fam.Children[i], c)
			}
		}
	})

	t.Run("empty FAM record", func(t *testing.T) {
		rec := doc.GetRecord("@F2@")
		if rec == nil {
			t.Fatal("GetRecord(@F2@) returned nil")
		}
		if len(rec.Tags) != 0 {
			t.Errorf("expected zero substructures, got %d tags", len(rec.Tags))
		}
		fam := doc.GetFamily("@F2@")
		if fam == nil {
			t.Fatal("GetFamily(@F2@) returned nil; empty FAM should still yield a Family entity")
		}
		if fam.Husband != "" || fam.Wife != "" || len(fam.Children) != 0 {
			t.Errorf("empty FAM should have no members, got husband=%q wife=%q children=%v",
				fam.Husband, fam.Wife, fam.Children)
		}
	})

	t.Run("self-marriage", func(t *testing.T) {
		fam := doc.GetFamily("@F3@")
		if fam == nil {
			t.Fatal("GetFamily(@F3@) returned nil")
		}
		// The decoder accepts self-marriage; flagging it is validator territory.
		if fam.Husband != "@I3@" || fam.Wife != "@I3@" {
			t.Errorf("expected HUSB == WIFE == @I3@, got husband=%q wife=%q", fam.Husband, fam.Wife)
		}
	})

	t.Run("same-sex marriage via two HUSB lines", func(t *testing.T) {
		fam := doc.GetFamily("@F4@")
		if fam == nil {
			t.Fatal("GetFamily(@F4@) returned nil")
		}
		// Typed field: last HUSB wins. Lossless: both HUSB tags preserved raw.
		if fam.Husband != "@I5@" {
			t.Errorf("Husband = %q, want %q (last HUSB line wins)", fam.Husband, "@I5@")
		}
		rec := doc.GetRecord("@F4@")
		husbCount := 0
		for _, tag := range rec.Tags {
			if tag.Level == 1 && tag.Tag == "HUSB" {
				husbCount++
			}
		}
		if husbCount != 2 {
			t.Errorf("raw tags contain %d HUSB lines, want 2 (lossless storage)", husbCount)
		}
	})

	t.Run("same-sex marriage via HUSB and WIFE", func(t *testing.T) {
		fam := doc.GetFamily("@F5@")
		if fam == nil {
			t.Fatal("GetFamily(@F5@) returned nil")
		}
		if fam.Husband != "@I4@" || fam.Wife != "@I5@" {
			t.Errorf("got husband=%q wife=%q, want @I4@/@I5@", fam.Husband, fam.Wife)
		}
		for _, xref := range []string{fam.Husband, fam.Wife} {
			ind := doc.GetIndividual(xref)
			if ind == nil {
				t.Fatalf("GetIndividual(%s) returned nil", xref)
			}
			if ind.Sex != "M" {
				t.Errorf("Sex of %s = %q, want M", xref, ind.Sex)
			}
		}
	})
}

// TestStructuralTortureSexValues covers every SEX payload variant: the 5.5.1
// values M/F/U, the 7.0-only X in a 5.5.1 file, empty payload, lowercase
// value, and a missing SEX line. Values are stored verbatim (lossless), never
// normalized or rejected.
func TestStructuralTortureSexValues(t *testing.T) {
	const path = "../testdata/edge-cases/sex-value-variants.ged"

	if err := decodeFixtureStrict(t, path); err != nil {
		t.Fatalf("strict decode should succeed, got %v", err)
	}

	doc := decodeFixture(t, path).Document

	tests := []struct {
		xref string
		want string
	}{
		{"@I1@", "M"},
		{"@I2@", "F"},
		{"@I3@", "U"},
		{"@I4@", "X"}, // 7.0 value in a 5.5.1 file: invalid-but-tolerated
		{"@I5@", ""},  // empty payload
		{"@I6@", "m"}, // lowercase: preserved verbatim, not upcased
		{"@I7@", ""},  // no SEX line at all
	}
	for _, tt := range tests {
		ind := doc.GetIndividual(tt.xref)
		if ind == nil {
			t.Errorf("GetIndividual(%s) returned nil", tt.xref)
			continue
		}
		if ind.Sex != tt.want {
			t.Errorf("Sex of %s = %q, want %q", tt.xref, ind.Sex, tt.want)
		}
	}

	// Distinguish empty-payload SEX (@I5@) from absent SEX (@I7@) via raw tags.
	hasSexTag := func(xref string) bool {
		for _, tag := range doc.GetRecord(xref).Tags {
			if tag.Tag == "SEX" {
				return true
			}
		}
		return false
	}
	if !hasSexTag("@I5@") {
		t.Error("@I5@ should retain its empty-payload SEX tag (lossless)")
	}
	if hasSexTag("@I7@") {
		t.Error("@I7@ should have no SEX tag")
	}
}

// TestStructuralTortureMixedCases covers three small structural quirks folded
// into one fixture: a NOTE record with both an XRef ID and a pointer payload
// (gedcom4j issue 96 shape), a physical line longer than the 5.5.1
// 255-character limit, and an unknown tag without a leading underscore.
func TestStructuralTortureMixedCases(t *testing.T) {
	const path = "../testdata/edge-cases/structural-torture.ged"

	// Unknown tags and long lines are warnings, not strict-mode errors.
	if err := decodeFixtureStrict(t, path); err != nil {
		t.Fatalf("strict decode should succeed, got %v", err)
	}

	result := decodeFixture(t, path)
	doc := result.Document

	t.Run("NOTE record with xref and pointer payload", func(t *testing.T) {
		note := doc.GetNote("@N2@")
		if note == nil {
			t.Fatal("GetNote(@N2@) returned nil")
		}
		// The pointer payload is preserved as literal text; the decoder does
		// not resolve it to @N1@'s content (lossless, resolution is caller's).
		if note.Text != "@N1@" {
			t.Errorf("Note.Text = %q, want %q", note.Text, "@N1@")
		}
		target := doc.GetNote("@N1@")
		if target == nil || target.Text != "This is the target note with ordinary text." {
			t.Errorf("target note @N1@ not decoded correctly: %+v", target)
		}
	})

	t.Run("line longer than 255 chars", func(t *testing.T) {
		rec := doc.GetRecord("@I1@")
		if rec == nil {
			t.Fatal("GetRecord(@I1@) returned nil")
		}
		var longNote string
		for _, tag := range rec.Tags {
			if tag.Tag == "NOTE" && len(tag.Value) > len(longNote) {
				longNote = tag.Value
			}
		}
		// "1 NOTE " + value exceeds the 5.5.1 255-char physical line limit;
		// the parser has no fixed line buffer and must preserve it intact.
		if len("1 NOTE ")+len(longNote) <= 255 {
			t.Errorf("longest NOTE line is %d chars; fixture should exceed 255", len("1 NOTE ")+len(longNote))
		}
		if !strings.HasPrefix(longNote, "This line deliberately exceeds the 255-character") {
			t.Errorf("long NOTE value truncated or altered: %.60q...", longNote)
		}
	})

	t.Run("unknown non-underscore tag", func(t *testing.T) {
		var found bool
		for _, d := range result.Diagnostics {
			if d.Code == CodeUnknownTag && strings.Contains(d.Message, "BUST") {
				found = true
			}
		}
		if !found {
			t.Errorf("expected UNKNOWN_TAG diagnostic for BUST, got %v", result.Diagnostics)
		}

		// Lossless: the unknown tag and its subordinate DATE are preserved raw.
		rec := doc.GetRecord("@I1@")
		for i, tag := range rec.Tags {
			if tag.Tag == "BUST" {
				if tag.Value != "Marble bust in city hall" {
					t.Errorf("BUST value = %q", tag.Value)
				}
				if i+1 >= len(rec.Tags) || rec.Tags[i+1].Tag != "DATE" || rec.Tags[i+1].Level != 2 {
					t.Error("BUST should keep its subordinate level-2 DATE tag")
				}
				return
			}
		}
		t.Error("BUST tag not preserved in raw tags")
	})
}

// TestStructuralTortureDeepNesting covers two-digit levels: a chain nested one
// level at a time down to level 99, the deepest level expressible in GEDCOM's
// two-digit level field.
func TestStructuralTortureDeepNesting(t *testing.T) {
	const path = "../testdata/edge-cases/deep-nesting-levels.ged"

	if err := decodeFixtureStrict(t, path); err != nil {
		t.Fatalf("strict decode should succeed, got %v", err)
	}

	result := decodeFixture(t, path)
	if len(result.Diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d: %v", len(result.Diagnostics), result.Diagnostics)
	}

	rec := result.Document.GetRecord("@I1@")
	if rec == nil {
		t.Fatal("GetRecord(@I1@) returned nil")
	}
	chainCount := 0
	var deepest *gedcom.Tag
	for _, tag := range rec.Tags {
		if strings.HasPrefix(tag.Tag, "_D") {
			chainCount++
			deepest = tag
		}
	}
	if chainCount != 99 {
		t.Errorf("chain has %d links, want 99", chainCount)
	}
	if deepest == nil || deepest.Level != 99 || deepest.Tag != "_D99" || deepest.Value != "depth 99" {
		t.Errorf("deepest tag = %+v, want level 99 _D99 %q", deepest, "depth 99")
	}
}

// TestStructuralTortureIndentedLines covers leading-whitespace-indented lines
// (Geni.com-style exports). The parser tolerates leading spaces and tabs in
// both modes because it splits on whitespace fields.
func TestStructuralTortureIndentedLines(t *testing.T) {
	const path = "../testdata/edge-cases/indented-lines.ged"

	if err := decodeFixtureStrict(t, path); err != nil {
		t.Fatalf("strict decode should succeed, got %v", err)
	}

	result := decodeFixture(t, path)
	if len(result.Diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d: %v", len(result.Diagnostics), result.Diagnostics)
	}

	ind := result.Document.GetIndividual("@I1@")
	if ind == nil {
		t.Fatal("GetIndividual(@I1@) returned nil")
	}
	if len(ind.Names) != 1 || ind.Names[0].Full != "Space /Indented/" {
		t.Errorf("Names = %+v, want one name %q", ind.Names, "Space /Indented/")
	}
	if ind.Sex != "M" {
		t.Errorf("Sex = %q, want M (tab-indented SEX line)", ind.Sex)
	}
	if len(ind.Events) != 1 || ind.Events[0].Date != "1 JAN 1900" {
		t.Errorf("Events = %+v, want one BIRT with date %q", ind.Events, "1 JAN 1900")
	}
}

// TestStructuralTortureBlankLines covers blank and whitespace-only lines
// mid-file: strict mode errors on the first one, lenient mode skips each with
// an EMPTY_LINE diagnostic and keeps the surrounding records.
func TestStructuralTortureBlankLines(t *testing.T) {
	const path = "../testdata/malformed/blank-lines.ged"

	err := decodeFixtureStrict(t, path)
	if err == nil {
		t.Fatal("strict decode should fail on blank line")
	}
	if !strings.Contains(err.Error(), "empty line") {
		t.Errorf("strict error = %v, want mention of empty line", err)
	}

	result := decodeFixture(t, path)

	var emptyLines []int
	for _, d := range result.Diagnostics {
		if d.Code == CodeEmptyLine {
			emptyLines = append(emptyLines, d.Line)
		}
	}
	wantLines := []int{15, 17, 20} // blank, spaces-only, tab-only
	if len(emptyLines) != len(wantLines) {
		t.Fatalf("EMPTY_LINE diagnostics at %v, want %v", emptyLines, wantLines)
	}
	for i, want := range wantLines {
		if emptyLines[i] != want {
			t.Errorf("EMPTY_LINE[%d] at line %d, want %d", i, emptyLines[i], want)
		}
	}

	// Both individuals survive around the skipped lines.
	doc := result.Document
	if ind := doc.GetIndividual("@I1@"); ind == nil || ind.Sex != "M" {
		t.Errorf("@I1@ = %+v, want SEX M preserved after blank line", ind)
	}
	if ind := doc.GetIndividual("@I2@"); ind == nil {
		t.Error("@I2@ missing; records after whitespace-only lines should survive")
	}
}

// TestStructuralTortureLevelOver99 covers the level-number ceiling. GEDCOM's
// level field is at most two digits, so 99 is the deepest valid level.
//
// Current behavior (tracked in issue #379): the parser's MaxNestingDepth
// boundary is 100, so a spec-invalid level-100 line is accepted by the parser
// and only reduced to a BAD_LEVEL_JUMP clamp in lenient mode, while level 101
// is rejected outright ("maximum nesting depth exceeded") — and that rejection
// is classified as BAD_LEVEL_JUMP rather than INVALID_LEVEL.
func TestStructuralTortureLevelOver99(t *testing.T) {
	const path = "../testdata/malformed/level-over-99.ged"

	err := decodeFixtureStrict(t, path)
	if err == nil {
		t.Fatal("strict decode should fail on level 101")
	}
	if !strings.Contains(err.Error(), "maximum nesting depth exceeded") {
		t.Errorf("strict error = %v, want nesting depth message", err)
	}

	result := decodeFixture(t, path)

	var warnings, errors int
	for _, d := range result.Diagnostics {
		if d.Code != CodeBadLevelJump {
			t.Errorf("unexpected diagnostic %s: %s", d.Code, d.Message)
			continue
		}
		switch d.Severity {
		case SeverityWarning:
			warnings++ // level 100 clamped to prevLevel+1
		case SeverityError:
			errors++ // level 101 rejected by the parser
		}
	}
	if warnings != 1 || errors != 1 {
		t.Errorf("got %d warnings and %d errors, want 1 clamp warning (level 100) and 1 rejection error (level 101)",
			warnings, errors)
	}

	// The level-100 line is preserved (clamped to level 2); level 101 is gone.
	rec := result.Document.GetRecord("@I1@")
	if rec == nil {
		t.Fatal("GetRecord(@I1@) returned nil")
	}
	var clamped bool
	for _, tag := range rec.Tags {
		if tag.Value == "level one hundred" && tag.Level == 2 {
			clamped = true
		}
		if tag.Value == "level one hundred one" {
			t.Error("level-101 line should have been skipped by the parser")
		}
	}
	if !clamped {
		t.Error("level-100 line should be clamped to level 2 and preserved")
	}
}

// TestStructuralTortureAtSign55 covers GEDCOM 5.5 at-sign torture: @@ doubling
// in text, @#...@ escapes inside NOTE payloads, and @@ at the start of
// CONC/CONT lines. Fixture vendored verbatim from gedcom7code/test-files
// (public domain). The decoder preserves all payloads verbatim: it neither
// collapses @@ nor interprets @#...@ escapes outside DATE values.
func TestStructuralTortureAtSign55(t *testing.T) {
	const path = "../testdata/edge-cases/atsign-55.ged"

	if err := decodeFixtureStrict(t, path); err != nil {
		t.Fatalf("strict decode should succeed, got %v", err)
	}

	result := decodeFixture(t, path)
	if len(result.Diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %v", result.Diagnostics)
	}
	doc := result.Document

	tests := []struct {
		xref string
		want string
	}{
		{"@N01@", "@ one leading"},
		{"@N03@", "@@ two leading"},
		{"@N05@", "doubled @@ internal"},
		{"@N09@", "@#DJULIAN@date escape zero spaces"},
		{"@N13@", "@#DJULIAN AND SUCH@ date escape internal spaces"},
		{"@N18@", "@@all in @one@@thing @#DWITH DATES@ , @#OBSOLETE@ etc"},
	}
	for _, tt := range tests {
		note := doc.GetNote(tt.xref)
		if note == nil {
			t.Errorf("GetNote(%s) returned nil", tt.xref)
			continue
		}
		if note.Text != tt.want {
			t.Errorf("Note %s = %q, want %q (verbatim, no unescaping)", tt.xref, note.Text, tt.want)
		}
	}

	// @@ at the start of CONC and CONT lines survives folding.
	n19 := doc.GetNote("@N19@")
	if n19 == nil {
		t.Fatal("GetNote(@N19@) returned nil")
	}
	want := "@@ at at front and @@ at after CONC and \n@@ at after CONT and @ inside CONT too."
	if n19.FullText() != want {
		t.Errorf("N19 FullText = %q, want %q", n19.FullText(), want)
	}
}

// TestStructuralTortureXRefCase covers XRef case-mismatch (pointer @test@ vs
// record @TEST@) and an XRef containing a space. Fixture vendored verbatim
// from gedcom7code/test-files (public domain).
//
// Current behavior documented here:
//   - XRefMap lookup is exact (case-sensitive); @test@ does not find @TEST@.
//     Whether pointers should match case-insensitively is an open decision.
//   - Known bug (issue #377): an XRef containing a space (@NoTe ref@) is
//     silently mangled — the parser takes "@NoTe" as the tag, producing a
//     bogus record type with an empty XRef instead of erroring or accepting.
func TestStructuralTortureXRefCase(t *testing.T) {
	const path = "../testdata/edge-cases/xref-case.ged"

	if err := decodeFixtureStrict(t, path); err != nil {
		t.Fatalf("strict decode should succeed, got %v", err)
	}

	doc := decodeFixture(t, path).Document

	t.Run("case-sensitive lookup", func(t *testing.T) {
		if doc.GetRecord("@TEST@") == nil {
			t.Error("record @TEST@ should exist")
		}
		if doc.GetRecord("@test@") != nil {
			t.Error("lookup is currently exact; @test@ should not match @TEST@")
		}
		if doc.GetNote("@NoTe@") == nil {
			t.Error("mixed-case note record @NoTe@ should exist")
		}
	})

	t.Run("xref containing a space", func(t *testing.T) {
		if doc.GetRecord("@NoTe ref@") != nil {
			t.Error("spaced xref is currently NOT stored under its full identifier")
		}
		var mangled *gedcom.Record
		for _, rec := range doc.Records {
			if rec.Type == gedcom.RecordType("@NoTe") {
				mangled = rec
			}
		}
		if mangled == nil {
			t.Fatal("expected the spaced-xref line to surface as a bogus record type @NoTe")
		}
		if mangled.XRef != "" || mangled.Value != "ref@ NOTE mixed case and space" {
			t.Errorf("mangled record = xref %q value %q; current behavior changed, revisit spaced-xref handling",
				mangled.XRef, mangled.Value)
		}
	})
}

// TestStructuralTortureAgeKeywords551 covers 5.5.1 AGE keyword payloads
// (CHILD/INFANT/STILLBORN, case variants, </> bounds). Fixture vendored
// verbatim from gedcom7code/test-files (public domain).
//
// Known bug (issue #378): the generic EVEN event tag is not decoded into
// Individual.Events — each of the 40 EVEN structures is flagged UNKNOWN_TAG
// and only preserved in raw tags (parseIndividual handles BIRT/DEAT/... but
// not EVEN, although parseFamily does handle EVEN).
func TestStructuralTortureAgeKeywords551(t *testing.T) {
	const path = "../testdata/edge-cases/age-keywords-551.ged"

	if err := decodeFixtureStrict(t, path); err != nil {
		t.Fatalf("strict decode should succeed, got %v", err)
	}

	result := decodeFixture(t, path)
	doc := result.Document

	unknownEven := 0
	for _, d := range result.Diagnostics {
		if d.Code == CodeUnknownTag && strings.Contains(d.Message, "EVEN") {
			unknownEven++
		}
	}
	if unknownEven != 40 {
		t.Errorf("got %d UNKNOWN_TAG diagnostics for EVEN, want 40 (current behavior; EVEN decoding is a follow-up)", unknownEven)
	}

	ind := doc.GetIndividual("@SIMPLE@")
	if ind == nil {
		t.Fatal("GetIndividual(@SIMPLE@) returned nil")
	}
	if len(ind.Events) != 0 {
		t.Errorf("Individual.Events has %d entries; EVEN is currently not decoded — update this test when it is", len(ind.Events))
	}

	// Lossless: every AGE payload survives verbatim in raw tags.
	var ages []string
	for _, tag := range doc.GetRecord("@SIMPLE@").Tags {
		if tag.Tag == "AGE" && tag.Level == 2 {
			ages = append(ages, tag.Value)
		}
	}
	if len(ages) != 40 {
		t.Fatalf("raw tags contain %d AGE values, want 40", len(ages))
	}
	spotChecks := map[string]bool{
		"child": false, "CHILD": false, "STILLBORN": false,
		"<0y": false, ">99y": false, "99y 11m 30d": false,
	}
	for _, age := range ages {
		if _, ok := spotChecks[age]; ok {
			spotChecks[age] = true
		}
	}
	for value, seen := range spotChecks {
		if !seen {
			t.Errorf("AGE keyword %q not preserved in raw tags", value)
		}
	}
}

// TestStructuralTortureDualYearDates covers the dual-year date matrix
// (1699/00 in plain, ABT, FROM/TO, and BET/AND forms). Fixture vendored
// verbatim from gedcom7code/test-files (public domain). Date payloads are
// stored verbatim on the event (lossless per ADR 0001).
func TestStructuralTortureDualYearDates(t *testing.T) {
	const path = "../testdata/edge-cases/date-dual-years.ged"

	if err := decodeFixtureStrict(t, path); err != nil {
		t.Fatalf("strict decode should succeed, got %v", err)
	}

	doc := decodeFixture(t, path).Document

	ind := doc.GetIndividual("@1@")
	if ind == nil {
		t.Fatal("GetIndividual(@1@) returned nil")
	}

	want := []string{
		"1699/00",
		"JAN 1699/00",
		"8 JAN 1699/00",
		"ABT 1699/00",
		"ABT JAN 1699/00",
		"ABT 8 JAN 1699/00",
		"FROM 1699/00",
		"FROM JAN 1699/00",
		"FROM 8 JAN 1699/00",
		"FROM JAN 1699/00 TO FEB 1699/00",
		"BET JAN 1699/00 AND FEB 1699/00",
	}
	if len(ind.Events) != len(want) {
		t.Fatalf("got %d events, want %d", len(ind.Events), len(want))
	}
	for i, w := range want {
		if ind.Events[i].Type != gedcom.EventBirth {
			t.Errorf("event %d type = %q, want BIRT", i, ind.Events[i].Type)
		}
		if ind.Events[i].Date != w {
			t.Errorf("event %d date = %q, want %q (verbatim dual-year)", i, ind.Events[i].Date, w)
		}
	}
}
