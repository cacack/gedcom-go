package gedcom_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// differingFields names the top-level exported fields where two entities of the
// same type disagree. Entities carry hundreds of raw tags, so dumping them
// whole buries the one field that actually differs.
func differingFields(want, got interface{}) []string {
	a, b := reflect.ValueOf(want), reflect.ValueOf(got)
	if a.Kind() != reflect.Ptr || b.Kind() != reflect.Ptr || a.IsNil() || b.IsNil() {
		return []string{fmt.Sprintf("entity shape (%T vs %T)", want, got)}
	}
	a, b = a.Elem(), b.Elem()
	if a.Type() != b.Type() {
		return []string{fmt.Sprintf("entity type (%s vs %s)", a.Type(), b.Type())}
	}

	var out []string
	for i := 0; i < a.NumField(); i++ {
		f := a.Type().Field(i)
		if f.PkgPath != "" { // unexported
			continue
		}
		if reflect.DeepEqual(a.Field(i).Interface(), b.Field(i).Interface()) {
			continue
		}
		if a.Field(i).Kind() == reflect.Slice && a.Field(i).Len() != b.Field(i).Len() {
			out = append(out, fmt.Sprintf("%s (len %d vs %d)", f.Name, a.Field(i).Len(), b.Field(i).Len()))
			continue
		}
		out = append(out, f.Name)
	}
	return out
}

// TestSubset_TypedEntitiesMatchSource asserts that the typed entity on every
// record a subset returns is field-for-field identical to the source's.
//
// Subset copies through the hand-written Clone methods, so a field added to an
// entity but not to its Clone disappears from the subset while the encoded
// bytes stay identical -- the encoder writes from Record.Tags, not from the
// entity. That made #428 invisible to every round-trip test in the package.
//
// TestCloneCompleteness guards the same property against synthetic values;
// this one guards it on real decoded data, so a field the decoder populates
// but the harness cannot fill is still covered.
func TestSubset_TypedEntitiesMatchSource(t *testing.T) {
	doc := mustDecode(t, "../testdata/gedcom-7.0/maximal70.ged")

	seed := firstIndividualXRef(t, doc)
	sub, err := doc.Subset([]string{seed})
	if err != nil {
		t.Fatalf("Subset returned error: %v", err)
	}
	if len(sub.Records) == 0 {
		t.Fatal("subset has no records")
	}

	// The fields #428 dropped are all note and external-ID fields. Assert the
	// fixture actually carries some, so the comparison below cannot pass by
	// comparing empty against empty.
	ind, ok := doc.XRefMap[seed].Entity.(*gedcom.Individual)
	if !ok {
		t.Fatalf("seed %q is not an *Individual", seed)
	}
	if len(ind.NoteXRefs) == 0 && len(ind.InlineNotes) == 0 {
		t.Fatalf("seed %q carries no notes; fixture no longer exercises #428", seed)
	}
	var externalIDs int
	for _, rec := range doc.Records {
		if fam, isFamily := rec.Entity.(*gedcom.Family); isFamily {
			externalIDs += len(fam.ExternalIDs)
		}
	}
	if externalIDs == 0 {
		t.Fatal("fixture families carry no external IDs; fixture no longer exercises #428")
	}

	for _, rec := range sub.Records {
		src, found := doc.XRefMap[rec.XRef]
		if !found {
			t.Errorf("subset record %q has no counterpart in the source", rec.XRef)
			continue
		}
		if !reflect.DeepEqual(src.Entity, rec.Entity) {
			t.Errorf("subset entity for %q differs from source in: %s",
				rec.XRef, strings.Join(differingFields(src.Entity, rec.Entity), ", "))
		}
		if src.Entity != nil && src.Entity == rec.Entity {
			t.Errorf("subset entity for %q aliases the source entity", rec.XRef)
		}
	}
}
