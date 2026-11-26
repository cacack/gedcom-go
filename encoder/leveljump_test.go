package encoder

import (
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// levelJumpGED holds a record whose levels jump past one, as a strict decode
// keeps them (SI-006): _FOO jumps from 1 to 4, and the level-3 SOURs after
// it are no deeper than _FOO, so their parent is BIRT, not _FOO.
const levelJumpGED = `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 BIRT
4 _FOO x
3 SOUR @S1@
3 SOUR @S2@
1 DEAT Y
0 @S1@ SOUR
0 @S2@ SOUR
0 TRLR
`

func decodeLevelJump(t *testing.T) (*gedcom.Document, *gedcom.Record) {
	t.Helper()
	doc, err := decoder.DecodeWithOptions(strings.NewReader(levelJumpGED), &decoder.DecodeOptions{StrictMode: true})
	if err != nil {
		t.Fatal(err)
	}
	rec := doc.GetRecord("@I1@")
	if rec == nil || len(rec.Tags) != 5 || rec.Tags[1].Level != 4 {
		t.Fatalf("strict decode did not keep the level jump: %+v", rec)
	}
	return doc, rec
}

// TestLevelJumpParentedAlikeByEveryReader runs one level-jumped record
// through the five readers of Record.Tags nesting (SI-023, issue #968): path
// lookup, the references RemoveRecord reports, MoveSubstructure, Diff and the
// encoder's structure tree. Each must put _FOO and both SOURs under BIRT.
func TestLevelJumpParentedAlikeByEveryReader(t *testing.T) {
	doc, rec := decodeLevelJump(t)
	birt, foo, s1, s2 := rec.Tags[0], rec.Tags[1], rec.Tags[2], rec.Tags[3]

	// 1. Record.FindAll.
	if got := rec.FindAll("BIRT", "SOUR"); !slices.Equal(got, []*gedcom.Tag{s1, s2}) {
		t.Errorf("FindAll(BIRT, SOUR) = %v, want both SOURs", got)
	}
	if got := rec.FindAll("BIRT", "_FOO"); !slices.Equal(got, []*gedcom.Tag{foo}) {
		t.Errorf("FindAll(BIRT, _FOO) = %v, want _FOO", got)
	}

	// 2. appendTagReferences, through RemoveRecord.
	refs, err := doc.RemoveRecord("@S1@")
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 || !slices.Equal(refs[0].Path, []string{"INDI", "BIRT", "SOUR"}) {
		t.Errorf("RemoveRecord(@S1@) references = %+v, want one at INDI.BIRT.SOUR", refs)
	}

	// 3. parentIndex, through MoveSubstructure: moving s1 after its last
	// sibling puts it at the end of BIRT's lines, after s2.
	moved := &gedcom.Record{Tags: slices.Clone(rec.Tags)}
	if !moved.MoveSubstructure(s1, nil) {
		t.Fatal("MoveSubstructure(SOUR @S1@, nil) = false")
	}
	if want := []*gedcom.Tag{birt, foo, s2, s1, rec.Tags[4]}; !slices.Equal(moved.Tags, want) {
		t.Errorf("MoveSubstructure put SOUR @S1@ at %d, want after SOUR @S2@", slices.Index(moved.Tags, s1))
	}

	// 4. buildDiffTree, through Diff: a changed SOUR is a change under BIRT.
	oldDoc, _ := decodeLevelJump(t)
	newDoc, newRec := decodeLevelJump(t)
	newRec.Tags[2].Value = "@S2@"
	diff := gedcom.Diff(oldDoc, newDoc, nil)
	if len(diff.Modified) != 1 || len(diff.Modified[0].Changes) != 1 ||
		!slices.Equal(diff.Modified[0].Changes[0].Path, []string{"BIRT"}) {
		t.Errorf("Diff changes = %+v, want one hunk with Path [BIRT]", diff.Modified)
	}

	// 5. The encoder's tagTree.
	roots := tagTree(rec.Tags)
	if len(roots) != 2 || roots[0].tag != birt {
		t.Fatalf("tagTree roots = %d, want BIRT and DEAT", len(roots))
	}
	var kids []*gedcom.Tag
	for _, k := range roots[0].kids {
		kids = append(kids, k.tag)
	}
	if !slices.Equal(kids, []*gedcom.Tag{foo, s1, s2}) {
		t.Errorf("tagTree BIRT children = %v, want _FOO and both SOURs", kids)
	}
}
