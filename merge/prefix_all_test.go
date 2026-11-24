package merge_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/merge"
)

// prefixAllDoc1 and prefixAllDoc2 share @I1@ and @F1@; doc2 alone holds
// @I2@, @U1@ (its header submitter) and @N1@.
const (
	prefixAllDoc1 = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n" +
		"0 @I1@ INDI\n1 NAME One /Person/\n1 FAMS @F1@\n" +
		"0 @F1@ FAM\n1 HUSB @I1@\n" +
		"0 TRLR\n"
	prefixAllDoc2 = "0 HEAD\n1 SUBM @U1@\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n" +
		"0 @I1@ INDI\n1 NAME Two /Person/\n1 FAMS @F1@\n1 NOTE @N1@\n" +
		"0 @I2@ INDI\n1 NAME Child /Person/\n1 FAMC @F1@\n" +
		"0 @F1@ FAM\n1 HUSB @I1@\n1 CHIL @I2@\n" +
		"0 @U1@ SUBM\n1 NAME Sub /Mitter/\n" +
		"0 @N1@ NOTE shared\n" +
		"0 TRLR\n"
)

func decodePrefixAll(t *testing.T, src string) *gedcom.Document {
	t.Helper()
	doc, err := decoder.Decode(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	return doc
}

func prefixAllOpts(prefix string) merge.CombineOptions {
	return merge.CombineOptions{CollisionStrategy: merge.PrefixAllDoc2, Prefix: prefix}
}

// rawValue returns the value of the first tag named tag in the record xref.
func rawValue(t *testing.T, doc *gedcom.Document, xref, tag string) string {
	t.Helper()
	rec := doc.GetRecord(xref)
	if rec == nil {
		t.Fatalf("record %s missing", xref)
	}
	for _, tg := range rec.Tags {
		if tg.Tag == tag {
			return tg.Value
		}
	}
	t.Fatalf("record %s has no %s line", xref, tag)
	return ""
}

// TestCombine_PrefixAllDoc2_PrefixesEveryXRef pins #924: colliding and
// non-colliding doc2 XRefs are both prefixed, and the report lists every
// rename with no identity entry.
func TestCombine_PrefixAllDoc2_PrefixesEveryXRef(t *testing.T) {
	doc1, doc2 := decodePrefixAll(t, prefixAllDoc1), decodePrefixAll(t, prefixAllDoc2)

	out, report, err := merge.Combine(doc1, doc2, prefixAllOpts("B_"))
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}

	want := map[string]string{
		"@I1@": "@B_I1@", // collides with doc1
		"@F1@": "@B_F1@", // collides with doc1
		"@I2@": "@B_I2@", // doc2 only
		"@U1@": "@B_U1@", // doc2 only
		"@N1@": "@B_N1@", // doc2 only
	}
	if !reflect.DeepEqual(report.RemappedXRefs, want) {
		t.Errorf("RemappedXRefs = %q, want %q", report.RemappedXRefs, want)
	}
	for old, renamed := range report.RemappedXRefs {
		if old == renamed {
			t.Errorf("RemappedXRefs holds identity entry %q", old)
		}
	}
	for _, renamed := range want {
		if out.GetRecord(renamed) == nil {
			t.Errorf("doc2 record %s missing from result", renamed)
		}
	}
	for _, bare := range []string{"@I2@", "@U1@", "@N1@"} {
		if out.GetRecord(bare) != nil {
			t.Errorf("doc2-only record %s kept its bare XRef", bare)
		}
	}
	if got, wantLen := len(out.Records), len(doc1.Records)+len(doc2.Records); got != wantLen {
		t.Errorf("record count = %d, want %d", got, wantLen)
	}
}

// TestCombine_PrefixAllDoc2_RewritesPointers pins that every pointer to a
// renamed doc2 record follows it: typed fields, raw tags and the header
// submitter doc2 contributes.
func TestCombine_PrefixAllDoc2_RewritesPointers(t *testing.T) {
	doc1, doc2 := decodePrefixAll(t, prefixAllDoc1), decodePrefixAll(t, prefixAllDoc2)

	out, _, err := merge.Combine(doc1, doc2, prefixAllOpts("B_"))
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}

	fam := out.GetFamily("@B_F1@")
	if fam == nil {
		t.Fatal("doc2 family @B_F1@ missing")
	}
	if fam.Husband != "@B_I1@" {
		t.Errorf("@B_F1@.Husband = %q, want @B_I1@", fam.Husband)
	}
	if len(fam.Children) != 1 || fam.Children[0] != "@B_I2@" {
		t.Errorf("@B_F1@.Children = %q, want [@B_I2@]", fam.Children)
	}
	raw := []struct{ xref, tag, want string }{
		{"@B_F1@", "HUSB", "@B_I1@"},
		{"@B_F1@", "CHIL", "@B_I2@"},
		{"@B_I1@", "FAMS", "@B_F1@"},
		{"@B_I1@", "NOTE", "@B_N1@"},
		{"@B_I2@", "FAMC", "@B_F1@"},
	}
	for _, r := range raw {
		if got := rawValue(t, out, r.xref, r.tag); got != r.want {
			t.Errorf("%s %s = %q, want %q", r.xref, r.tag, got, r.want)
		}
	}
	// doc1 names no submitter, so the combined header takes doc2's, renamed.
	if out.Header.Submitter != "@B_U1@" {
		t.Errorf("Header.Submitter = %q, want @B_U1@", out.Header.Submitter)
	}
	for _, tg := range out.Header.Tags {
		if tg.Tag == "SUBM" && tg.Value != "@B_U1@" {
			t.Errorf("header SUBM line = %q, want @B_U1@", tg.Value)
		}
	}
}

// TestCombine_PrefixAllDoc2_Doc1Untouched pins that doc1's records keep
// their XRefs and pointers, and that neither input is mutated.
func TestCombine_PrefixAllDoc2_Doc1Untouched(t *testing.T) {
	doc1, doc2 := decodePrefixAll(t, prefixAllDoc1), decodePrefixAll(t, prefixAllDoc2)
	snap1, snap2 := doc1.Clone(), doc2.Clone()

	out, _, err := merge.Combine(doc1, doc2, prefixAllOpts("B_"))
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}

	if !reflect.DeepEqual(doc1, snap1) {
		t.Error("doc1 was mutated")
	}
	if !reflect.DeepEqual(doc2, snap2) {
		t.Error("doc2 was mutated")
	}
	for i, rec := range doc1.Records {
		if !reflect.DeepEqual(out.Records[i], rec) {
			t.Errorf("doc1 record %s changed in the result", rec.XRef)
		}
	}
	if fam := out.GetFamily("@F1@"); fam == nil || fam.Husband != "@I1@" {
		t.Errorf("doc1 family @F1@ = %+v, want Husband @I1@", fam)
	}
	if got := rawValue(t, out, "@I1@", "NAME"); got != "One /Person/" {
		t.Errorf("@I1@ NAME = %q, want doc1's", got)
	}
}

// TestCombine_PrefixAllDoc2_NoCollisions pins that the strategy renames
// doc2's records even when no XRef collides, where PrefixDoc2 renames none.
func TestCombine_PrefixAllDoc2_NoCollisions(t *testing.T) {
	tests := []struct {
		name     string
		strategy merge.CollisionStrategy
		want     map[string]string
		wantInd  string
	}{
		{"PrefixAllDoc2", merge.PrefixAllDoc2,
			map[string]string{"@b_I1@": "@Z_b_I1@", "@b_F1@": "@Z_b_F1@", "@b_U1@": "@Z_b_U1@"}, "@Z_b_I1@"},
		{"PrefixDoc2", merge.PrefixDoc2, map[string]string{}, "@b_I1@"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, report, err := merge.Combine(buildDoc("a_"), buildDoc("b_"),
				merge.CombineOptions{CollisionStrategy: tt.strategy, Prefix: "Z_"})
			if err != nil {
				t.Fatalf("Combine: %v", err)
			}
			if !reflect.DeepEqual(report.RemappedXRefs, tt.want) {
				t.Errorf("RemappedXRefs = %q, want %q", report.RemappedXRefs, tt.want)
			}
			if out.GetIndividual(tt.wantInd) == nil {
				t.Errorf("doc2 individual %s missing", tt.wantInd)
			}
			if out.GetIndividual("@a_I1@") == nil {
				t.Error("doc1 individual @a_I1@ missing")
			}
		})
	}
}

// TestCombine_PrefixDoc2_NonCollidingStayBare pins that PrefixDoc2 is
// unchanged by #924: beside colliding XRefs it still keeps doc2's
// non-colliding ones bare.
func TestCombine_PrefixDoc2_NonCollidingStayBare(t *testing.T) {
	doc1, doc2 := decodePrefixAll(t, prefixAllDoc1), decodePrefixAll(t, prefixAllDoc2)

	out, report, err := merge.Combine(doc1, doc2,
		merge.CombineOptions{CollisionStrategy: merge.PrefixDoc2, Prefix: "B_"})
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}

	want := map[string]string{"@I1@": "@B_I1@", "@F1@": "@B_F1@"}
	if !reflect.DeepEqual(report.RemappedXRefs, want) {
		t.Errorf("RemappedXRefs = %q, want %q", report.RemappedXRefs, want)
	}
	for _, bare := range []string{"@I2@", "@U1@", "@N1@"} {
		if out.GetRecord(bare) == nil {
			t.Errorf("non-colliding %s should keep its bare XRef", bare)
		}
	}
	if got := rawValue(t, out, "@B_F1@", "CHIL"); got != "@I2@" {
		t.Errorf("@B_F1@ CHIL = %q, want @I2@", got)
	}
}

// TestCombine_PrefixAllDoc2_NonPointerXRef pins that a doc2 record whose
// XRef is not pointer-shaped is not renamed (RemapXRefs skips it), while
// the records beside it are; and that when doc1 holds the same XRef the
// combine fails as a collision, since the record cannot be renamed apart.
func TestCombine_PrefixAllDoc2_NonPointerXRef(t *testing.T) {
	t.Run("kept", func(t *testing.T) {
		doc1 := decodePrefixAll(t, prefixAllDoc1)
		out, report, err := merge.Combine(doc1, decodeRecovered(t), prefixAllOpts("B_"))
		if err != nil {
			t.Fatalf("Combine: %v", err)
		}
		if out.GetIndividual("@I1") == nil {
			t.Error("recovered record @I1 missing or renamed")
		}
		want := map[string]string{"@I9@": "@B_I9@"}
		if !reflect.DeepEqual(report.RemappedXRefs, want) {
			t.Errorf("RemappedXRefs = %q, want %q", report.RemappedXRefs, want)
		}
	})
	t.Run("shared with doc1", func(t *testing.T) {
		_, _, err := merge.Combine(decodeRecovered(t), decodeRecovered(t), prefixAllOpts("B_"))
		if !errors.Is(err, merge.ErrXRefCollision) {
			t.Fatalf("err = %v, want ErrXRefCollision", err)
		}
		var cerr *merge.CombineError
		if !errors.As(err, &cerr) {
			t.Fatalf("err = %T, want *CombineError", err)
		}
		if cerr.Kind != "prefix-collision" || !reflect.DeepEqual(cerr.Colliding, []string{"@I1"}) {
			t.Errorf("Kind = %q, Colliding = %q, want prefix-collision [@I1]", cerr.Kind, cerr.Colliding)
		}
	})
}

// TestCombine_PrefixAllDoc2_PrefixCollision pins that a prefixed XRef doc1
// already holds fails the combine, including one that collided with nothing
// before the prefix was applied.
func TestCombine_PrefixAllDoc2_PrefixCollision(t *testing.T) {
	doc1 := decodePrefixAll(t,
		"0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n0 @B_I2@ INDI\n1 NAME Taken /Name/\n0 TRLR\n")
	doc2 := decodePrefixAll(t, prefixAllDoc2)

	out, report, err := merge.Combine(doc1, doc2, prefixAllOpts("B_"))
	if out != nil || report != nil {
		t.Error("error path should return nil doc and nil report")
	}
	if !errors.Is(err, merge.ErrXRefCollision) {
		t.Fatalf("err = %v, want ErrXRefCollision", err)
	}
	var cerr *merge.CombineError
	if !errors.As(err, &cerr) {
		t.Fatalf("err = %T, want *CombineError", err)
	}
	if cerr.Kind != "prefix-collision" || !reflect.DeepEqual(cerr.Colliding, []string{"@B_I2@"}) {
		t.Errorf("Kind = %q, Colliding = %q, want prefix-collision [@B_I2@]", cerr.Kind, cerr.Colliding)
	}
}

// TestCombine_PrefixAllDoc2_RequiresPrefix pins that an empty Prefix fails
// as it does for PrefixDoc2, whether or not anything collides.
func TestCombine_PrefixAllDoc2_RequiresPrefix(t *testing.T) {
	docs := map[string]func() *gedcom.Document{
		"colliding":     buildCollidingDoc,
		"non-colliding": func() *gedcom.Document { return buildDoc("b_") },
	}
	for name, doc2 := range docs {
		t.Run(name, func(t *testing.T) {
			_, _, err := merge.Combine(buildCollidingDoc(), doc2(), prefixAllOpts(""))
			if !errors.Is(err, merge.ErrInvalidOptions) {
				t.Fatalf("err = %v, want ErrInvalidOptions", err)
			}
			var cerr *merge.CombineError
			if !errors.As(err, &cerr) || cerr.Kind != "missing-prefix" {
				t.Fatalf("err = %#v, want *CombineError of kind missing-prefix", err)
			}
			if !strings.Contains(err.Error(), "PrefixAllDoc2") {
				t.Errorf("message should name the strategy: %v", err)
			}
		})
	}
}

// TestCombine_PrefixAllDoc2_VoidRecord pins the 5.5.1 @VOID@ record case. A
// record named @VOID@ is renamed with its pointers, as PrefixDoc2 renames
// one that collides; PrefixDoc2 keeps one that does not.
func TestCombine_PrefixAllDoc2_VoidRecord(t *testing.T) {
	const head = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n"
	const withVoid = head + "0 @I1@ INDI\n1 NAME Two /Person/\n1 NOTE @VOID@\n0 @VOID@ NOTE hi\n0 TRLR\n"
	const plain = head + "0 @I7@ INDI\n1 NAME One /Person/\n0 TRLR\n"
	tests := []struct {
		name     string
		strategy merge.CollisionStrategy
		doc1     string
		wantVoid string // XRef of doc2's @VOID@ record in the result
	}{
		{"PrefixAllDoc2 non-colliding", merge.PrefixAllDoc2, plain, "@B_VOID@"},
		{"PrefixAllDoc2 colliding", merge.PrefixAllDoc2, withVoid, "@B_VOID@"},
		{"PrefixDoc2 colliding", merge.PrefixDoc2, withVoid, "@B_VOID@"},
		{"PrefixDoc2 non-colliding", merge.PrefixDoc2, plain, "@VOID@"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc1, doc2 := decodePrefixAll(t, tt.doc1), decodePrefixAll(t, withVoid)
			out, report, err := merge.Combine(doc1, doc2,
				merge.CombineOptions{CollisionStrategy: tt.strategy, Prefix: "B_"})
			if err != nil {
				t.Fatalf("Combine: %v", err)
			}
			rec := out.GetRecord(tt.wantVoid)
			if rec == nil || rec.Type != gedcom.RecordTypeNote {
				t.Fatalf("doc2's @VOID@ note not found at %s", tt.wantVoid)
			}
			renamed, reported := report.RemappedXRefs["@VOID@"]
			if wantReported := tt.wantVoid != "@VOID@"; reported != wantReported || (reported && renamed != tt.wantVoid) {
				t.Errorf("RemappedXRefs[@VOID@] = %q (present %v), want %q", renamed, reported, tt.wantVoid)
			}
			// doc2's individual is the last INDI in the result; its NOTE
			// pointer follows the record.
			var ind *gedcom.Record
			for _, r := range out.Records {
				if r.Type == gedcom.RecordTypeIndividual {
					ind = r
				}
			}
			if got := rawValue(t, out, ind.XRef, "NOTE"); got != tt.wantVoid {
				t.Errorf("%s NOTE = %q, want %q", ind.XRef, got, tt.wantVoid)
			}
		})
	}
}

// TestCombine_PrefixAllDoc2_DanglingPointer pins that an XRef doc2 points at
// but does not define is prefixed with doc2's records. Left bare, doc2's
// broken "FAMC @F9@" would become a link to doc1's family @F9@.
func TestCombine_PrefixAllDoc2_DanglingPointer(t *testing.T) {
	const head = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n"
	doc1 := decodePrefixAll(t, head+
		"0 @I1@ INDI\n1 NAME One /Person/\n1 FAMS @F9@\n"+
		"0 @F9@ FAM\n1 HUSB @I1@\n0 TRLR\n")
	doc2 := decodePrefixAll(t, "0 HEAD\n1 SUBM @U7@\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n"+
		"0 @I1@ INDI\n1 NAME Two /Person/\n1 FAMC @F9@\n1 NOTE @N 3@\n0 TRLR\n")

	out, report, err := merge.Combine(doc1, doc2, prefixAllOpts("B_"))
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	want := map[string]string{
		"@I1@":  "@B_I1@",
		"@F9@":  "@B_F9@",
		"@N 3@": "@B_N 3@",
		"@U7@":  "@B_U7@",
	}
	if !reflect.DeepEqual(report.RemappedXRefs, want) {
		t.Errorf("RemappedXRefs = %v, want %v", report.RemappedXRefs, want)
	}
	if got := rawValue(t, out, "@B_I1@", "FAMC"); got != "@B_F9@" {
		t.Errorf("doc2 FAMC = %q, want @B_F9@", got)
	}
	if got := rawValue(t, out, "@B_I1@", "NOTE"); got != "@B_N 3@" {
		t.Errorf("doc2 spaced NOTE pointer = %q, want @B_N 3@", got)
	}
	if ind := out.GetIndividual("@B_I1@"); ind == nil || len(ind.ChildInFamilies) != 1 ||
		ind.ChildInFamilies[0].FamilyXRef != "@B_F9@" {
		t.Errorf("doc2 typed FAMC not rewritten: %+v", ind)
	}
	if out.GetRecord("@B_F9@") != nil {
		t.Error("the dangling pointer's target must stay undefined")
	}
	// doc1's own link to its @F9@ is untouched.
	if got := rawValue(t, out, "@I1@", "FAMS"); got != "@F9@" {
		t.Errorf("doc1 FAMS = %q, want @F9@", got)
	}
	if got := rawValue(t, doc2, "@I1@", "FAMC"); got != "@F9@" {
		t.Errorf("input doc2 was mutated: FAMC = %q", got)
	}

	// PrefixDoc2 renames only colliding records, so it leaves the pointer.
	out, report, err = merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: merge.PrefixDoc2, Prefix: "B_"})
	if err != nil {
		t.Fatalf("Combine(PrefixDoc2): %v", err)
	}
	if !reflect.DeepEqual(report.RemappedXRefs, map[string]string{"@I1@": "@B_I1@"}) {
		t.Errorf("PrefixDoc2 RemappedXRefs = %v, want only @I1@", report.RemappedXRefs)
	}
	if got := rawValue(t, out, "@B_I1@", "FAMC"); got != "@F9@" {
		t.Errorf("PrefixDoc2 FAMC = %q, want @F9@", got)
	}
}

// TestCombine_PrefixAllDoc2_DanglingPointerCollision pins that a prefixed
// dangling pointer may not land on a record doc1 holds.
func TestCombine_PrefixAllDoc2_DanglingPointerCollision(t *testing.T) {
	const head = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n"
	doc1 := decodePrefixAll(t, head+"0 @B_F9@ FAM\n0 TRLR\n")
	doc2 := decodePrefixAll(t, head+"0 @I1@ INDI\n1 FAMC @F9@\n0 TRLR\n")

	_, _, err := merge.Combine(doc1, doc2, prefixAllOpts("B_"))
	var cerr *merge.CombineError
	if !errors.As(err, &cerr) {
		t.Fatalf("err = %v, want *CombineError", err)
	}
	if cerr.Kind != "prefix-collision" || !reflect.DeepEqual(cerr.Colliding, []string{"@B_F9@"}) {
		t.Errorf("Kind = %q, Colliding = %q, want prefix-collision [@B_F9@]", cerr.Kind, cerr.Colliding)
	}
}
