package gedcom

import (
	"sort"
	"testing"
)

// A source or note reached only through a name's citation or note must land in
// the closure (issue #643).
func TestSubset_FollowsPersonalNamePointers(t *testing.T) {
	doc := &Document{Header: &Header{Version: Version551}, XRefMap: map[string]*Record{}}
	add := func(xref string, typ RecordType, entity interface{}) {
		rec := &Record{XRef: xref, Type: typ, Entity: entity}
		doc.Records = append(doc.Records, rec)
		doc.XRefMap[xref] = rec
	}
	add("@I1@", RecordTypeIndividual, &Individual{
		XRef: "@I1@",
		Names: []*PersonalName{{
			Full:            "John /Doe/",
			SourceCitations: []*SourceCitation{{SourceXRef: "@S1@"}},
			NoteXRefs:       []string{"@N1@"},
		}},
	})
	add("@S1@", RecordTypeSource, &Source{XRef: "@S1@", Title: "Source 1"})
	add("@N1@", RecordTypeNote, &Note{XRef: "@N1@", Text: "Note 1"})

	sub, err := doc.Subset([]string{"@I1@"})
	if err != nil {
		t.Fatalf("Subset errored: %v", err)
	}
	var got []string
	for _, rec := range sub.Records {
		got = append(got, rec.XRef)
	}
	sort.Strings(got)
	if want := []string{"@I1@", "@N1@", "@S1@"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("closure = %v, want %v", got, want)
	}
}
