package validator

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/charset"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/parser"
	"github.com/cacack/gedcom-go/v3/version"
)

// Issue #712: parser.Records -> decoder.DecodeRecord -> ValidateRecord is a
// working streaming pipeline, reporting what batch validation reports.

const streamingPipelineGEDCOM = `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 1 JAN 1900
1 DEAT
2 DATE 1 JAN 1850
1 FAMS @F1@
1 FAMC @F404@
1 SOUR @S404@
0 @I2@ INDI
1 NAME Jane /Doe/
1 FAMS @F405@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I404@
1 CHIL @I405@
0 @S1@ SOUR
1 TITL A source
0 TRLR
`

// issueKey is what makes two reported issues the same issue.
func issueKey(i *Issue) string {
	return fmt.Sprintf("%s %s -> %s", i.Code, i.RecordXRef, i.RelatedXRef)
}

func streamValidate(t *testing.T, text string) []string {
	t.Helper()
	var sv *StreamingValidator
	var ver gedcom.Version
	var keys []string
	for raw, err := range parser.Records(charset.NewReader(strings.NewReader(text))) {
		if err != nil {
			t.Fatal(err)
		}
		if raw.Type == "HEAD" {
			ver = version.DeclaredVersion(raw.Lines)
		}
		if sv == nil {
			sv = NewStreamingValidator(StreamingOptions{Version: ver})
		}
		rec, err := decoder.DecodeRecord(raw, ver, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, issue := range sv.ValidateRecord(rec) {
			keys = append(keys, issueKey(&issue))
		}
	}
	for _, issue := range sv.Finalize() {
		keys = append(keys, issueKey(&issue))
	}
	sort.Strings(keys)
	return keys
}

func TestStreamingPipeline_DecodeRecordMatchesBatch(t *testing.T) {
	got := streamValidate(t, streamingPipelineGEDCOM)

	doc, err := decoder.Decode(strings.NewReader(streamingPipelineGEDCOM))
	if err != nil {
		t.Fatal(err)
	}
	// Batch validation covers more than streaming; compare the issues
	// streaming is documented to report: orphaned references and the
	// per-individual date logic.
	streamed := map[string]bool{
		CodeOrphanedFAMC: true, CodeOrphanedFAMS: true, CodeOrphanedHUSB: true,
		CodeOrphanedWIFE: true, CodeOrphanedCHIL: true, CodeOrphanedSOUR: true,
		CodeDeathBeforeBirth: true,
	}
	var want []string
	for _, issue := range New().ValidateAll(doc) {
		if streamed[issue.Code] {
			want = append(want, issueKey(&issue))
		}
	}
	sort.Strings(want)

	for _, k := range []string{
		CodeDeathBeforeBirth + " @I1@ -> ",
		CodeOrphanedFAMC + " @I1@ -> @F404@",
		CodeOrphanedSOUR + " @I1@ -> @S404@",
		CodeOrphanedFAMS + " @I2@ -> @F405@",
		CodeOrphanedWIFE + " @F1@ -> @I404@",
		CodeOrphanedCHIL + " @F1@ -> @I405@",
	} {
		if !slices.Contains(want, k) {
			t.Fatalf("batch ValidateAll does not report %q; the fixture no longer exercises it (batch: %v)", k, want)
		}
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("streaming pipeline reports\n  %s\nbatch ValidateAll reports\n  %s",
			strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
}

// benchmarkShapeGEDCOM builds a file shaped like the 1,000,000-individual
// file measured in docs/guides/performance.md (Streaming Validator): all
// individuals first, then families. Individual i is a spouse in family
// (i+1)/2; family j has HUSB 2j-1 and WIFE 2j; and the first half of the
// families each have two CHIL, 4j-1 and 4j, whose FAMC names family j. All of
// those are forward references to families. Every hundredth individual cites
// a source written after all families, so source references are forward
// references until the end.
//
// That file declares everything it references, so it has no reference
// issues. The returned set lists those injected here, one or more per
// orphan code, each as issueKey formats it.
func benchmarkShapeGEDCOM(individuals int) (text string, injected []string) {
	families := individuals / 2
	sources := individuals / 100
	orphan := func(code, record, related string) {
		injected = append(injected, code+" "+record+" -> "+related)
	}
	extra := map[string][]string{} // record XRef -> extra level-1 lines
	addExtra := func(code, record, tag, related string) {
		extra[record] = append(extra[record], "1 "+tag+" "+related)
		orphan(code, record, related)
	}
	addExtra(CodeOrphanedFAMC, "@I17@", "FAMC", "@F900001@")
	addExtra(CodeOrphanedFAMS, fmt.Sprintf("@I%d@", individuals/2), "FAMS", "@F900002@")
	addExtra(CodeOrphanedSOUR, fmt.Sprintf("@I%d@", individuals-1), "SOUR", "@S900003@")
	addExtra(CodeOrphanedSOUR, "@I1@", "SOUR", "@S900004@")
	addExtra(CodeOrphanedCHIL, fmt.Sprintf("@F%d@", families-1), "CHIL", "@I900005@")
	addExtra(CodeOrphanedFAMC, fmt.Sprintf("@I%d@", individuals), "FAMC", "@F900006@")
	// Replaced, not added: these families lose a spouse that is declared.
	husbandOf := map[int]string{42: "@I900007@"}
	wifeOf := map[int]string{families / 2: "@I900008@", families: "@I900009@"}
	for j, x := range husbandOf {
		orphan(CodeOrphanedHUSB, fmt.Sprintf("@F%d@", j), x)
	}
	for j, x := range wifeOf {
		orphan(CodeOrphanedWIFE, fmt.Sprintf("@F%d@", j), x)
	}

	var b strings.Builder
	b.WriteString("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n")
	for i := 1; i <= individuals; i++ {
		x := fmt.Sprintf("@I%d@", i)
		fmt.Fprintf(&b, "0 %s INDI\n1 NAME Given%d /Surname%d/\n", x, i, i%997)
		fmt.Fprintf(&b, "1 FAMS @F%d@\n", (i+1)/2)
		if i%4 == 3 || i%4 == 0 {
			fmt.Fprintf(&b, "1 FAMC @F%d@\n", (i+1)/4)
		}
		if i%100 == 0 {
			fmt.Fprintf(&b, "1 SOUR @S%d@\n", i/100)
		}
		for _, l := range extra[x] {
			b.WriteString(l + "\n")
		}
	}
	for j := 1; j <= families; j++ {
		x := fmt.Sprintf("@F%d@", j)
		husb, wife := fmt.Sprintf("@I%d@", 2*j-1), fmt.Sprintf("@I%d@", 2*j)
		if h, ok := husbandOf[j]; ok {
			husb = h
		}
		if w, ok := wifeOf[j]; ok {
			wife = w
		}
		fmt.Fprintf(&b, "0 %s FAM\n1 HUSB %s\n1 WIFE %s\n", x, husb, wife)
		if j <= families/2 {
			fmt.Fprintf(&b, "1 CHIL @I%d@\n1 CHIL @I%d@\n", 4*j-1, 4*j)
		}
		for _, l := range extra[x] {
			b.WriteString(l + "\n")
		}
	}
	for s := 1; s <= sources; s++ {
		fmt.Fprintf(&b, "0 @S%d@ SOUR\n1 TITL Source %d\n", s, s)
	}
	b.WriteString("0 TRLR\n")
	sort.Strings(injected)
	return b.String(), injected
}

// TestStreamingPipeline_MatchesBatchOnBenchmarkSubset checks issue #712's
// parity criterion on a 10,000-individual file of the benchmark file's shape:
// the streaming pipeline reports exactly the reference issues batch
// ValidateAll reports, which are exactly the injected ones.
func TestStreamingPipeline_MatchesBatchOnBenchmarkSubset(t *testing.T) {
	text, injected := benchmarkShapeGEDCOM(10_000)

	got := streamValidate(t, text)

	doc, err := decoder.Decode(strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, issue := range New().ValidateAll(doc) {
		if strings.HasPrefix(issue.Code, "ORPHANED_") {
			want = append(want, issueKey(&issue))
		}
	}
	sort.Strings(want)

	if strings.Join(want, "\n") != strings.Join(injected, "\n") {
		t.Fatalf("batch ValidateAll reports\n  %s\nbut the file injects\n  %s",
			strings.Join(want, "\n  "), strings.Join(injected, "\n  "))
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("streaming pipeline reports\n  %s\nbatch ValidateAll reports\n  %s",
			strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
}
