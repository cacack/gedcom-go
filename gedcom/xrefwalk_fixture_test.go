package gedcom

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// Fixture completeness: the reachability fixture must populate every field
// that can carry an XRef, on every type reachable from a record.
//
// TestVisitReachesEveryPointerField only proves the walker reaches what the
// fixture happens to set. That makes the fixture a hand-maintained list, and a
// hand-maintained list drifts: SharedNote.ChangeDate was missing from both the
// walker and the fixture at the same time, so the reachability test passed
// while merge.RemapXRefs silently dropped that structure's note pointers.
//
// This test closes that loop. It reflects over the fixture and fails on any
// carrier-typed field left at its zero value, so a newly added pointer field
// is a failure here first -- before anyone has to remember it exists.
//
// carrierTypes are the field types that can hold or contain an XRef. A field
// of one of these types left unset in the fixture means the walker's handling
// of it is untested.
func carrierTypeNames() map[string]bool {
	return map[string]bool{
		"*gedcom.ChangeDate":           true,
		"*gedcom.SourceRepositoryLink": true,
		"[]*gedcom.Association":        true,
		"[]*gedcom.Attribute":          true,
		"[]*gedcom.Event":              true,
		"[]*gedcom.LDSOrdinance":       true,
		"[]*gedcom.MediaLink":          true,
		"[]*gedcom.SourceCitation":     true,
		"[]gedcom.FamilyLink":          true,
		"[]*gedcom.Tag":                true,
	}
}

// carrierFieldNames are string / []string fields that hold a pointer rather
// than free text. Matched by name because their type says nothing.
func carrierFieldNames() map[string]bool {
	return map[string]bool{
		"NoteXRefs": true, "Notes": true, "Children": true,
		"SpouseInFamilies": true, "Husband": true, "Wife": true,
		"IndividualXRef": true, "FamilyXRef": true, "SourceXRef": true,
		"MediaXRef": true,
	}
}

// exemptFromFixture records carrier-typed fields the fixture deliberately
// leaves unset, with the reason. Keep this small and justified -- an entry
// here is a hole in the reachability guarantee. Empty is the goal state.
var exemptFromFixture = map[string]string{}

func TestReachabilityFixtureIsComplete(t *testing.T) {
	carrierTypes := carrierTypeNames()
	carrierNames := carrierFieldNames()

	var unset []string
	var inspect func(v reflect.Value, path string, depth int)
	inspect = func(v reflect.Value, path string, depth int) {
		if depth > 8 || !v.IsValid() {
			return
		}
		switch v.Kind() {
		case reflect.Pointer:
			if !v.IsNil() {
				inspect(v.Elem(), path, depth+1)
			}
			return
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				inspect(v.Index(i), path, depth+1)
			}
			return
		case reflect.Struct:
		default:
			return
		}

		rt := v.Type()
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			if f.PkgPath != "" { // unexported
				continue
			}
			fieldPath := rt.Name() + "." + f.Name
			// Suffix-match XRef/XRefs as well as the literal names: the guard
			// did not catch MediaObject.SharedNoteXRefs (it was populated by
			// hand), and the next FooXRefs field would have slipped through
			// the same way.
			isCarrier := carrierTypes[f.Type.String()] ||
				((carrierNames[f.Name] ||
					strings.HasSuffix(f.Name, "XRef") ||
					strings.HasSuffix(f.Name, "XRefs")) &&
					(f.Type.Kind() == reflect.Slice || f.Type.Kind() == reflect.String))

			if isCarrier {
				empty := v.Field(i).IsZero()
				if s, ok := v.Field(i).Interface().([]string); ok && len(s) == 0 {
					empty = true
				}
				if empty {
					if _, ok := exemptFromFixture[fieldPath]; !ok {
						unset = append(unset, fieldPath)
					}
				}
			}
			inspect(v.Field(i), fieldPath, depth+1)
		}
	}

	for _, r := range fullyPopulatedRecords() {
		inspect(reflect.ValueOf(r.Entity), "", 0)
	}

	seen := map[string]bool{}
	var uniq []string
	for _, u := range unset {
		if !seen[u] {
			seen[u] = true
			uniq = append(uniq, u)
		}
	}
	sort.Strings(uniq)

	if len(uniq) > 0 {
		t.Errorf("reachability fixture leaves %d XRef-carrying field(s) unset:\n  %v\n\n"+
			"Populate them in fullyPopulatedRecords and add the planted values to\n"+
			"TestVisitReachesEveryPointerField's want list, or add an entry to\n"+
			"exemptFromFixture explaining why the field cannot hold a pointer.\n"+
			"An unset carrier field means the walker's handling of it is untested --\n"+
			"which is exactly how SharedNote.ChangeDate was missed.",
			len(uniq), uniq)
	}

	// Self-cleaning, same rationale as cloneKnownBad: an exemption that no
	// longer corresponds to a real field is a stale claim about the code.
	for path := range exemptFromFixture {
		if !fieldExists(path) {
			t.Errorf("exemptFromFixture names %q, which no longer exists -- remove the entry", path)
		}
	}
}

// fieldExists reports whether "Type.Field" names a real exported field on a
// type reachable from a record entity.
func fieldExists(path string) bool {
	var typeName, fieldName string
	if _, err := fmt.Sscanf(path, "%s", &typeName); err != nil {
		return false
	}
	for i := 0; i < len(path); i++ {
		if path[i] == '.' {
			typeName, fieldName = path[:i], path[i+1:]
			break
		}
	}
	for _, probe := range []any{
		Individual{}, Family{}, Source{}, MediaObject{}, Repository{},
		Submitter{}, SharedNote{}, Note{}, Event{}, Attribute{},
		SourceCitation{}, Association{}, LDSOrdinance{}, ChangeDate{},
		MediaLink{}, FamilyLink{}, SourceRepositoryLink{},
	} {
		rt := reflect.TypeOf(probe)
		if rt.Name() != typeName {
			continue
		}
		_, ok := rt.FieldByName(fieldName)
		return ok
	}
	return false
}
