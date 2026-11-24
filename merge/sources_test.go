package merge_test

import (
	"bytes"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/merge"
	"github.com/cacack/gedcom-go/v3/validator"
)

const sourceHead = `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
1 SUBM @U1@
0 @U1@ SUBM
1 NAME Example Submitter
`

// sourceTree holds one source twice, as @S1@ and @S2@, cited at every depth
// and through vendor tags.
const sourceTree = sourceHead + `0 @I1@ INDI
1 NAME John /Example/
2 SOUR @S2@
3 PAGE p. 1
1 SEX M
1 BIRT
2 DATE 1 JAN 1900
2 SOUR @S2@
3 PAGE p. 2
3 DATA
4 DATE 1900
4 TEXT Born here.
3 QUAY 2
3 _CONF high
1 SOUR @S1@
1 SOUR @S2@
1 _SRC @S2@
2 _WHY @S2@
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @I1@
1 MARR
2 SOUR @S2@
3 PAGE p. 3
0 @N1@ NOTE A note.
1 SOUR @S2@
0 @S1@ SOUR
1 TITL Parish Register
1 AUTH Example Parish
1 NOTE Shared note.
1 REPO @R1@
1 _VEND one
1 CHAN
2 DATE 1 JAN 2020
0 @S2@ SOUR
1 TITL Parish register, vol. 1
2 CONT second line
1 AUTH Example Parish
1 PUBL Example Press
2 CONT 1901
1 ABBR Register
1 NOTE Shared note.
1 NOTE Only in drop.
1 REPO @R1@
1 REPO @R2@
2 CALN 12-B
1 _VEND one
1 _VEND two
1 CHAN
2 DATE 2 FEB 2021
0 @R1@ REPO
1 NAME County Archive
0 @R2@ REPO
1 NAME City Library
0 TRLR
`

// grammarErrors returns doc's error issues with the grammar check on, so a
// line repeated beyond its cardinality is among them.
func grammarErrors(doc *gedcom.Document) []validator.Issue {
	var out []validator.Issue
	for _, is := range validator.NewWithOptions(&validator.ValidateOptions{CheckGrammar: true}).ValidateAll(doc) {
		if is.Severity == validator.SeverityError {
			out = append(out, is)
		}
	}
	return out
}

func checkNoNewErrors(t *testing.T, before, after *gedcom.Document) {
	t.Helper()
	for name, issues := range map[string]func(*gedcom.Document) []validator.Issue{"ValidateAll": errorIssues, "grammar": grammarErrors} {
		if got, base := issues(after), issues(before); len(got) > len(base) {
			t.Errorf("%s: merged document has %d error issues, the original %d:", name, len(got), len(base))
			for _, is := range got {
				t.Logf("  %s %s %s", is.Code, is.RecordXRef, is.Message)
			}
		}
	}
}

// countLines returns how many lines below level 0 have tag and one of values.
func countLines(doc *gedcom.Document, tag string, values ...string) int {
	n := 0
	for _, r := range doc.Records {
		for _, t := range r.Tags {
			if t.Tag == tag && slices.Contains(values, t.Value) {
				n++
			}
		}
	}
	return n
}

// checkOnlyPointersChanged checks every record of before but skip has the
// same lines in after once drop is read as keep.
func checkOnlyPointersChanged(t *testing.T, before, after *gedcom.Document, keep, drop string, skip ...string) {
	t.Helper()
	for _, r := range before.Records {
		if r.XRef == drop || slices.Contains(skip, r.XRef) {
			continue
		}
		want := strings.Split(strings.ReplaceAll(strings.Join(tagLines(r.Tags), "\n"), drop, keep), "\n")
		if got := recordLines(t, after, r.XRef); !reflect.DeepEqual(got, want) {
			t.Errorf("%s lines =\n%q\nwant\n%q", r.XRef, got, want)
		}
	}
}

func substructureLines(subs []merge.Substructure) [][]string {
	var out [][]string
	for _, s := range subs {
		out = append(out, tagLines(s.Tags))
	}
	return out
}

func firstTags(subs []merge.Substructure) []string {
	var out []string
	for _, s := range subs {
		out = append(out, s.Tags[0].Tag)
	}
	return out
}

func TestSources(t *testing.T) {
	doc := decodeZip(t, sourceTree)
	before := doc.Clone()

	out, report := mustMerge(t, merge.Sources, doc, "@S1@", "@S2@")
	if !reflect.DeepEqual(doc, before) {
		t.Fatal("Sources modified its input")
	}

	// Six citations before, six after; every other record differs only in
	// the pointer, so each citation's own lines are as they were.
	if got, want := countLines(out, "SOUR", "@S1@"), countLines(doc, "SOUR", "@S1@", "@S2@"); got != want || want != 6 {
		t.Errorf("%d citations after the merge, %d before, want 6", got, want)
	}
	checkOnlyPointersChanged(t, doc, out, "@S1@", "@S2@", "@S1@")
	if len(report.Collapsed) != 0 {
		t.Errorf("Collapsed = %v, want none", report.Collapsed)
	}
	var repointed []string
	for _, ref := range report.Repointed {
		repointed = append(repointed, ref.XRef+" "+strings.Join(ref.Path, "."))
	}
	wantRepointed := []string{
		"@I1@ INDI.NAME.SOUR", "@I1@ INDI.BIRT.SOUR", "@I1@ INDI.SOUR", "@I1@ INDI._SRC", "@I1@ INDI._SRC._WHY",
		"@F1@ FAM.MARR.SOUR", "@N1@ NOTE.SOUR",
	}
	if !reflect.DeepEqual(repointed, wantRepointed) {
		t.Errorf("Repointed = %q, want %q", repointed, wantRepointed)
	}

	want := []string{
		"1 TITL Parish Register", "1 AUTH Example Parish", "1 NOTE Shared note.", "1 NOTE Only in drop.",
		"1 REPO @R1@", "1 REPO @R2@", "2 CALN 12-B", "1 _VEND one", "1 _VEND two",
		"1 CHAN", "2 DATE 1 JAN 2020", "1 PUBL Example Press", "2 CONT 1901", "1 ABBR Register",
	}
	if got := recordLines(t, out, "@S1@"); !reflect.DeepEqual(got, want) {
		t.Errorf("@S1@ lines =\n%q\nwant\n%q", got, want)
	}
	if got, want := firstTags(report.Moved), []string{"PUBL", "ABBR", "NOTE", "REPO", "_VEND"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Moved = %v, want %v", got, want)
	}
	if got, want := firstTags(report.Deduplicated), []string{"AUTH", "NOTE", "REPO", "_VEND"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Deduplicated = %v, want %v", got, want)
	}

	// A differing TITL and CHAN stay keep's, and drop's are reported whole.
	var keeps, drops []merge.Substructure
	for _, c := range report.Conflicts {
		keeps, drops = append(keeps, c.Keep), append(drops, c.Drop)
	}
	wantKeeps := [][]string{{"1 TITL Parish Register"}, {"1 CHAN", "2 DATE 1 JAN 2020"}}
	wantDrops := [][]string{{"1 TITL Parish register, vol. 1", "2 CONT second line"}, {"1 CHAN", "2 DATE 2 FEB 2021"}}
	if got := substructureLines(keeps); !reflect.DeepEqual(got, wantKeeps) {
		t.Errorf("conflicts' Keep = %q, want %q", got, wantKeeps)
	}
	if got := substructureLines(drops); !reflect.DeepEqual(got, wantDrops) {
		t.Errorf("conflicts' Drop = %q, want %q", got, wantDrops)
	}

	// The typed view follows the raw lines.
	src := out.GetSource("@S1@")
	if src == nil || src.Title != "Parish Register" || src.Abbreviation != "Register" || len(src.RepositoryLinks) != 2 {
		t.Errorf("typed @S1@ = %+v", src)
	}

	checkNoNewErrors(t, doc, out)

	var buf bytes.Buffer
	if err := encoder.Encode(&buf, out); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	back, err := decoder.Decode(&buf)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got := recordLines(t, back, "@S1@"); !reflect.DeepEqual(got, want) {
		t.Errorf("read back @S1@ lines =\n%q\nwant\n%q", got, want)
	}
}

// TestSources_CarriesWhatKeepLacks merges a bare source with a full one: no
// line conflicts, so every line of drop is under keep.
func TestSources_CarriesWhatKeepLacks(t *testing.T) {
	doc := decodeZip(t, sourceHead+`0 @S1@ SOUR
1 NOTE Bare.
0 @S2@ SOUR
1 TITL A Title
1 AUTH An Author
1 PUBL A Publisher
1 ABBR Abbr
1 TEXT Some text
2 CONT and more
1 DATA
2 EVEN BIRT
3 DATE FROM 1800 TO 1900
2 AGNC An Agency
1 RIN 7
1 CHAN
2 DATE 2 FEB 2021
0 TRLR
`)
	out, report := mustMerge(t, merge.Sources, doc, "@S1@", "@S2@")
	want := append([]string{"1 NOTE Bare."}, recordLines(t, doc, "@S2@")...)
	if got := recordLines(t, out, "@S1@"); !reflect.DeepEqual(got, want) {
		t.Errorf("@S1@ lines =\n%q\nwant\n%q", got, want)
	}
	if len(report.Conflicts) != 0 || len(report.Deduplicated) != 0 || len(report.Moved) != 8 {
		t.Errorf("report = %+v", report)
	}
	checkNoNewErrors(t, doc, out)
}

// TestSources_OncePerVersion checks which lines may occur only once is the
// document's version's own rule.
func TestSources_OncePerVersion(t *testing.T) {
	const records = `0 @S1@ SOUR
1 TITL One
1 REPO @R1@
1 RIN 1
1 CREA
2 DATE 1 JAN 2020
0 @S2@ SOUR
1 TITL Two
1 REPO @R2@
1 RIN 2
1 CREA
2 DATE 2 FEB 2021
0 @R1@ REPO
1 NAME County Archive
0 @R2@ REPO
1 NAME City Library
0 TRLR
`
	head := func(version string) string {
		return "0 HEAD\n1 GEDC\n2 VERS " + version + "\n"
	}
	undeclared := func(d *gedcom.Document) { d.Header.DeclaredVersion = "" }
	unknown := func(d *gedcom.Document) { d.Header.DeclaredVersion, d.Header.Version = "", "" }
	tests := []struct {
		name      string
		version   string
		edit      func(*gedcom.Document)
		conflicts []string
		moved     []string
	}{
		{"5.5", "5.5", nil, []string{"TITL", "REPO", "RIN"}, []string{"CREA"}},
		{"5.5.1", "5.5.1", nil, []string{"TITL", "RIN"}, []string{"REPO", "CREA"}},
		{"7.0", "7.0", nil, []string{"TITL", "CREA"}, []string{"REPO", "RIN"}},
		{"7.1 is read by 7.0's rules", "7.1", nil, []string{"TITL", "CREA"}, []string{"REPO", "RIN"}},
		{"detected version when none is declared", "5.5", undeclared, []string{"TITL", "REPO", "RIN"}, []string{"CREA"}},
		{"no version", "5.5.1", unknown, []string{"TITL"}, []string{"REPO", "RIN", "CREA"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := decodeZip(t, head(tt.version)+records)
			if tt.edit != nil {
				tt.edit(doc)
			}
			_, report := mustMerge(t, merge.Sources, doc, "@S1@", "@S2@")
			var conflicts []string
			for _, c := range report.Conflicts {
				conflicts = append(conflicts, c.Drop.Tags[0].Tag)
			}
			if !reflect.DeepEqual(conflicts, tt.conflicts) {
				t.Errorf("Conflicts = %v, want %v", conflicts, tt.conflicts)
			}
			if got := firstTags(report.Moved); !reflect.DeepEqual(got, tt.moved) {
				t.Errorf("Moved = %v, want %v", got, tt.moved)
			}
		})
	}

	t.Run("no header", func(t *testing.T) {
		doc := decodeZip(t, head("5.5.1")+records)
		doc.Header = nil
		_, report := mustMerge(t, merge.Sources, doc, "@S1@", "@S2@")
		if len(report.Conflicts) != 1 || report.Conflicts[0].Drop.Tags[0].Value != "Two" {
			t.Errorf("Conflicts = %+v, want the TITL alone", report.Conflicts)
		}
	})
}

// TestSources_SecondOnceOnlyLineOfDrop checks a drop that itself repeats a
// line that may occur once gives keep only the first.
func TestSources_SecondOnceOnlyLineOfDrop(t *testing.T) {
	doc := decodeZip(t, sourceHead+`0 @S1@ SOUR
1 NOTE Bare.
0 @S2@ SOUR
1 TITL First
1 TITL Second
0 TRLR
`)
	out, report := mustMerge(t, merge.Sources, doc, "@S1@", "@S2@")
	if want := []string{"1 NOTE Bare.", "1 TITL First"}; !reflect.DeepEqual(recordLines(t, out, "@S1@"), want) {
		t.Errorf("@S1@ = %q, want %q", recordLines(t, out, "@S1@"), want)
	}
	if len(report.Conflicts) != 1 || report.Conflicts[0].Keep.Tags[0].Value != "First" || report.Conflicts[0].Drop.Tags[0].Value != "Second" {
		t.Errorf("Conflicts = %+v", report.Conflicts)
	}
}

const repositoryTree = sourceHead + `0 @I1@ INDI
1 NAME John /Example/
1 _ARCH @R2@
2 _DESK @R2@
0 @S1@ SOUR
1 TITL One
1 REPO @R1@
2 CALN 1-A
1 REPO @R2@
2 CALN 1-A
0 @S2@ SOUR
1 TITL Two
1 REPO @R2@
2 CALN 2-B
3 MEDI Book
0 @S3@ SOUR
1 TITL Three
1 REPO @R1@
1 REPO @R2@
2 CALN 3-C
0 @R1@ REPO
1 NAME County Archive
1 PHON 555-0100
1 NOTE Shared.
0 @R2@ REPO
1 NAME County Archives
1 ADDR 1 Example Street
2 CITY Springfield
1 PHON 555-0100
1 PHON 555-0101
1 EMAIL archive@example.com
1 NOTE Shared.
1 _HOURS weekdays
0 TRLR
`

func TestRepositories(t *testing.T) {
	doc := decodeZip(t, repositoryTree)
	before := doc.Clone()

	out, report := mustMerge(t, merge.Repositories, doc, "@R1@", "@R2@")
	if !reflect.DeepEqual(doc, before) {
		t.Fatal("Repositories modified its input")
	}

	want := []string{
		"1 NAME County Archive", "1 PHON 555-0100", "1 PHON 555-0101", "1 NOTE Shared.",
		"1 ADDR 1 Example Street", "2 CITY Springfield", "1 EMAIL archive@example.com", "1 _HOURS weekdays",
	}
	if got := recordLines(t, out, "@R1@"); !reflect.DeepEqual(got, want) {
		t.Errorf("@R1@ lines =\n%q\nwant\n%q", got, want)
	}
	if got, want := firstTags(report.Moved), []string{"ADDR", "PHON", "EMAIL", "_HOURS"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Moved = %v, want %v", got, want)
	}
	if got, want := firstTags(report.Deduplicated), []string{"PHON", "NOTE"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Deduplicated = %v, want %v", got, want)
	}
	if len(report.Conflicts) != 1 {
		t.Fatalf("Conflicts = %+v, want one", report.Conflicts)
	}
	if c := report.Conflicts[0]; !reflect.DeepEqual(tagLines(c.Keep.Tags), []string{"1 NAME County Archive"}) ||
		!reflect.DeepEqual(tagLines(c.Drop.Tags), []string{"1 NAME County Archives"}) {
		t.Errorf("conflict = %q / %q", tagLines(c.Keep.Tags), tagLines(c.Drop.Tags))
	}

	// A REPO that would repeat a sibling is removed; every other pointer,
	// vendor tags included, is repointed with its lines as they were.
	if want := []string{"1 TITL One", "1 REPO @R1@", "2 CALN 1-A"}; !reflect.DeepEqual(recordLines(t, out, "@S1@"), want) {
		t.Errorf("@S1@ = %q, want %q", recordLines(t, out, "@S1@"), want)
	}
	if want := []gedcom.Reference{{XRef: "@S1@", Path: []string{"SOUR", "REPO"}}}; !reflect.DeepEqual(report.Collapsed, want) {
		t.Errorf("Collapsed = %v, want %v", report.Collapsed, want)
	}
	checkOnlyPointersChanged(t, doc, out, "@R1@", "@R2@", "@R1@", "@S1@")
	if got := countLines(out, "REPO", "@R1@"); got != 4 {
		t.Errorf("%d REPO pointers to @R1@, want 4", got)
	}
	if links := out.GetSource("@S1@").RepositoryLinks; len(links) != 1 {
		t.Errorf("typed @S1@ has %d repository links, want 1", len(links))
	}
	if repo := out.GetRepository("@R1@"); repo == nil || repo.Name != "County Archive" || len(repo.Phone) != 2 || repo.Address == nil {
		t.Errorf("typed @R1@ = %+v", repo)
	}

	checkNoNewErrors(t, doc, out)
}

func TestSourcesRepositories_Errors(t *testing.T) {
	type mergeFunc = func(*gedcom.Document, string, string, merge.RecordMergeOptions) (*gedcom.Document, *merge.RecordMergeReport, error)
	tests := []struct {
		name       string
		fn         mergeFunc
		keep, drop string
		unknown    bool
	}{
		{"source into itself", merge.Sources, "@S1@", "@S1@", false},
		{"repository into itself", merge.Repositories, "@R1@", "@R1@", false},
		{"missing source drop", merge.Sources, "@S1@", "@S9@", true},
		{"missing source keep", merge.Sources, "@S9@", "@S1@", true},
		{"missing repository drop", merge.Repositories, "@R1@", "@R9@", true},
		{"missing repository keep", merge.Repositories, "@R9@", "@R1@", true},
		{"repository as source", merge.Sources, "@S1@", "@R1@", false},
		{"individual as source", merge.Sources, "@I1@", "@S1@", false},
		{"source as repository", merge.Repositories, "@R1@", "@S1@", false},
		{"sources as individuals", merge.Individuals, "@S1@", "@S2@", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := decodeZip(t, sourceTree)
			before := doc.Clone()
			out, report, err := tt.fn(doc, tt.keep, tt.drop, merge.RecordMergeOptions{})
			if !errors.Is(err, merge.ErrInvalidMerge) {
				t.Fatalf("err = %v, want ErrInvalidMerge", err)
			}
			if got := errors.Is(err, gedcom.ErrUnknownXRef); got != tt.unknown {
				t.Errorf("errors.Is(err, ErrUnknownXRef) = %t, want %t", got, tt.unknown)
			}
			if out != nil || report != nil {
				t.Error("want nil document and report")
			}
			if !reflect.DeepEqual(doc, before) {
				t.Error("document modified")
			}
		})
	}

	if _, _, err := merge.Sources(nil, "@S1@", "@S2@", merge.RecordMergeOptions{}); !errors.Is(err, merge.ErrNilDocument) {
		t.Errorf("nil doc: err = %v, want ErrNilDocument", err)
	}
}
