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
