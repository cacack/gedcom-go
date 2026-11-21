package decoder

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestSourceCitationExtensionTags checks that a citation's untyped extension
// subtags reach SourceCitation.Tags with their subordinates, at levels
// relative to the citation, wherever the citation sits; that _APID, which has
// a typed field, stays out; and that Record.Tags is left as it was.
// Ref: Issue #864
func TestSourceCitationExtensionTags(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 SOUR @S1@
2 _RESN confidential
2 PAGE 12
1 BIRT
2 SOUR @S1@
3 _RESN confidential
4 _WHY family request
3 _APID 1,7602::2771226
3 QUAY 2
0 @S1@ SOUR
1 TITL Census
0 TRLR
`
	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	for _, d := range result.Diagnostics {
		if d.Code == CodeUnknownTag {
			t.Errorf("unexpected diagnostic on line %d: %s", d.Line, d.Message)
		}
	}
	indi := result.Document.GetIndividual("@I1@")
	if indi == nil || len(indi.SourceCitations) != 1 || len(indi.Events) != 1 || len(indi.Events[0].SourceCitations) != 1 {
		t.Fatalf("unexpected structure: %+v", indi)
	}

	strip := func(tags []*gedcom.Tag) []gedcom.Tag {
		out := make([]gedcom.Tag, len(tags))
		for i, tag := range tags {
			out[i] = gedcom.Tag{Level: tag.Level, Tag: tag.Tag, Value: tag.Value}
		}
		return out
	}

	record := indi.SourceCitations[0]
	if got, want := strip(record.Tags), []gedcom.Tag{{Level: 1, Tag: "_RESN", Value: "confidential"}}; !reflect.DeepEqual(got, want) {
		t.Errorf("record citation Tags = %+v, want %+v", got, want)
	}
	if record.Page != "12" {
		t.Errorf("record citation Page = %q, want 12 (typed lines after an extension still parse)", record.Page)
	}

	event := indi.Events[0].SourceCitations[0]
	want := []gedcom.Tag{
		{Level: 1, Tag: "_RESN", Value: "confidential"},
		{Level: 2, Tag: "_WHY", Value: "family request"},
	}
	if got := strip(event.Tags); !reflect.DeepEqual(got, want) {
		t.Errorf("event citation Tags = %+v, want %+v", got, want)
	}
	if event.AncestryAPID == nil || event.Quality == nil || *event.Quality != 2 {
		t.Errorf("event citation typed fields lost: APID %+v, Quality %v", event.AncestryAPID, event.Quality)
	}
	// The copies keep their source line and leave the raw record's levels
	// alone: the event citation's _RESN is the one beneath "2 SOUR".
	line := event.Tags[0].LineNumber
	if lines := strings.Split(input, "\n"); line < 2 || lines[line-1] != "3 _RESN confidential" || lines[line-2] != "2 SOUR @S1@" {
		t.Errorf("_RESN LineNumber = %d, does not point at the event citation's _RESN", line)
	}
	found := false
	for _, rec := range result.Document.Records {
		for _, tag := range rec.Tags {
			if tag.Tag == "_RESN" && tag.LineNumber == line {
				found = true
				if tag.Level != 3 {
					t.Errorf("Record.Tags _RESN level = %d, want 3", tag.Level)
				}
			}
		}
	}
	if !found {
		t.Errorf("Record.Tags lacks the line-%d _RESN", line)
	}
}
