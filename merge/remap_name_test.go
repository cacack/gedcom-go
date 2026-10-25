package merge_test

import (
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/merge"
)

// A name's citation and note pointers are remapped with everything else
// (issue #643); left alone, they would resolve to the other document's records
// after a merge.
func TestRemapXRefs_PersonalNamePointers(t *testing.T) {
	doc := &gedcom.Document{Header: &gedcom.Header{Version: gedcom.Version551}, XRefMap: map[string]*gedcom.Record{}}
	add := func(xref string, typ gedcom.RecordType, entity interface{}) {
		rec := &gedcom.Record{XRef: xref, Type: typ, Entity: entity}
		doc.Records = append(doc.Records, rec)
		doc.XRefMap[xref] = rec
	}
	add("@I1@", gedcom.RecordTypeIndividual, &gedcom.Individual{
		XRef: "@I1@",
		Names: []*gedcom.PersonalName{{
			Full:            "John /Doe/",
			SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@S1@"}},
			NoteXRefs:       []string{"@N1@"},
		}},
	})
	add("@S1@", gedcom.RecordTypeSource, &gedcom.Source{XRef: "@S1@"})
	add("@N1@", gedcom.RecordTypeNote, &gedcom.Note{XRef: "@N1@"})

	out, _, err := merge.RemapXRefs(doc, func(old string) string { return "@A" + old[1:] })
	if err != nil {
		t.Fatalf("RemapXRefs: %v", err)
	}
	name := out.GetIndividual("@AI1@").Names[0]
	if got := name.SourceCitations[0].SourceXRef; got != "@AS1@" {
		t.Errorf("name citation SourceXRef = %q, want @AS1@", got)
	}
	if got := name.NoteXRefs[0]; got != "@AN1@" {
		t.Errorf("name NoteXRefs[0] = %q, want @AN1@", got)
	}
}
