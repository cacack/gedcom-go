package gedcom

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

// Clone completeness: every exported field must survive a clone.
//
// The clone functions in clone.go copy field by field, by hand. Nothing fails
// when a field is added to an entity and not added to its clone, so each new
// field is a latent data-loss bug -- #428 (Individual/Family/MediaObject drop
// note and external-ID fields), #400 (cloneDate drops IsInterpreted and
// InterpretedFrom) and the TypeDetail omission fixed in passing by #386 are the
// same defect three times over.
//
// This fills every exported field with a distinct non-zero value, clones, and
// reports any field that did not come across. It is the counterpart to
// TestByteRoundTrip: that one guards the serialized form, this one guards the
// in-memory copy.
//
// cloneKnownBad is self-cleaning in both directions, for the same reason as the
// byte harness: an expected-failure list that only ever grows is how #410
// happened.

// cloneKnownBad maps "Type.field.path" to the issue responsible. Every entry is
// a live bug. Delete an entry when its issue is fixed; the test will say so.
// Empty: #428 (note and external-ID fields, 21 paths) and #436 (cloneEvent
// dropping IsNegative, 4 paths) are fixed. Add an entry only alongside a filed
// issue, and delete it with the fix.
var cloneKnownBad = map[string]string{}

// unfillable records field paths that reflection cannot populate meaningfully,
// so their absence from the clone proves nothing. Kept explicit rather than
// silently skipped.
var unfillable = map[string]string{
	"Document.XRefMap": "aliases Records by design; identity is asserted by TestDocumentClone",
}

// cloneSharedByDesign maps "Type.field.path" to the reason a clone may
// legitimately share that slice, map or pointer with its original. Every entry
// is a live exception, self-cleaning like cloneKnownBad: delete it when the
// sharing stops, and the test will say so. Empty: every reference field in
// clone.go is copied.
var cloneSharedByDesign = map[string]string{}

var fixedTime = time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)

// fillCounter gives every scalar a distinct value, so a clone that copies the
// wrong field into a slot is caught as well as one that copies nothing.
type filler struct{ n int }

func (f *filler) next() int { f.n++; return f.n }

// fill populates every exported field of v with a distinct non-zero value.
func (f *filler) fill(v reflect.Value, path string, depth int) {
	if depth > 6 || !v.CanSet() {
		return
	}

	// time.Time has unexported fields; assign wholesale.
	if v.Type() == reflect.TypeOf(time.Time{}) {
		v.Set(reflect.ValueOf(fixedTime))
		return
	}

	switch v.Kind() {
	case reflect.String:
		v.SetString(fmt.Sprintf("v%d", f.next()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(int64(f.next()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(uint64(f.next()))
	case reflect.Float32, reflect.Float64:
		v.SetFloat(float64(f.next()) + 0.5)
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Pointer:
		v.Set(reflect.New(v.Type().Elem()))
		f.fill(v.Elem(), path, depth+1)
	case reflect.Slice:
		s := reflect.MakeSlice(v.Type(), 1, 1)
		f.fill(s.Index(0), path+"[0]", depth+1)
		v.Set(s)
	case reflect.Map:
		m := reflect.MakeMap(v.Type())
		k := reflect.New(v.Type().Key()).Elem()
		f.fill(k, path+".key", depth+1)
		val := reflect.New(v.Type().Elem()).Elem()
		f.fill(val, path+".value", depth+1)
		m.SetMapIndex(k, val)
		v.Set(m)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			sf := v.Type().Field(i)
			if sf.PkgPath != "" { // unexported
				continue
			}
			f.fill(v.Field(i), path+"."+sf.Name, depth+1)
		}
	case reflect.Interface:
		// Record.Entity is the only interface field. Give it a concrete value
		// so cloneEntity is exercised rather than skipped.
		if path == "Record.Entity" {
			ind := &Individual{}
			f.fill(reflect.ValueOf(ind).Elem(), "Individual", depth+1)
			v.Set(reflect.ValueOf(ind))
		}
	}
}

// diffFields walks two values in parallel and appends the path of every
// exported field whose value did not survive.
func diffFields(a, b reflect.Value, path string, out *[]string, depth int) {
	if depth > 8 {
		return
	}
	if _, skip := unfillable[path]; skip {
		return
	}

	switch a.Kind() {
	case reflect.Pointer, reflect.Interface:
		if a.IsNil() {
			return
		}
		if b.IsNil() {
			*out = append(*out, path)
			return
		}
		diffFields(a.Elem(), b.Elem(), path, out, depth+1)
	case reflect.Slice, reflect.Map:
		if a.Len() != b.Len() {
			*out = append(*out, path)
			return
		}
		if a.Kind() == reflect.Slice {
			for i := 0; i < a.Len(); i++ {
				diffFields(a.Index(i), b.Index(i), path, out, depth+1)
			}
			return
		}
		for _, k := range a.MapKeys() {
			bv := b.MapIndex(k)
			if !bv.IsValid() {
				*out = append(*out, path)
				return
			}
			diffFields(a.MapIndex(k), bv, path, out, depth+1)
		}
	case reflect.Struct:
		if a.Type() == reflect.TypeOf(time.Time{}) {
			if !a.Interface().(time.Time).Equal(b.Interface().(time.Time)) {
				*out = append(*out, path)
			}
			return
		}
		for i := 0; i < a.NumField(); i++ {
			sf := a.Type().Field(i)
			if sf.PkgPath != "" {
				continue
			}
			diffFields(a.Field(i), b.Field(i), path+"."+sf.Name, out, depth+1)
		}
	default:
		if !reflect.DeepEqual(a.Interface(), b.Interface()) {
			*out = append(*out, path)
		}
	}
}

// aliasedFields walks an original and its clone in parallel and appends the path
// of every non-empty slice, map or non-nil pointer the clone shares with the
// original. diffFields compares values, so an aliased slice is equal to itself
// and passes there; this asserts identity (#474). Nil and empty slices are
// skipped: they hold no backing array to share.
func aliasedFields(a, b reflect.Value, path string, out *[]string, depth int) {
	if depth > 8 {
		return
	}
	if _, skip := unfillable[path]; skip {
		return
	}

	switch a.Kind() {
	case reflect.Pointer:
		if a.IsNil() || b.IsNil() {
			return
		}
		if a.Pointer() == b.Pointer() {
			*out = append(*out, path)
			return
		}
		aliasedFields(a.Elem(), b.Elem(), path, out, depth+1)
	case reflect.Interface:
		if a.IsNil() || b.IsNil() {
			return
		}
		aliasedFields(a.Elem(), b.Elem(), path, out, depth+1)
	case reflect.Slice:
		if a.Len() == 0 || b.Len() == 0 {
			return
		}
		if a.Pointer() == b.Pointer() {
			*out = append(*out, path)
			return
		}
		for i := 0; i < a.Len() && i < b.Len(); i++ {
			aliasedFields(a.Index(i), b.Index(i), path, out, depth+1)
		}
	case reflect.Map:
		if a.Len() == 0 || b.Len() == 0 {
			return
		}
		if a.Pointer() == b.Pointer() {
			*out = append(*out, path)
			return
		}
		for _, k := range a.MapKeys() {
			if bv := b.MapIndex(k); bv.IsValid() {
				aliasedFields(a.MapIndex(k), bv, path, out, depth+1)
			}
		}
	case reflect.Struct:
		for i := 0; i < a.NumField(); i++ {
			if a.Type().Field(i).PkgPath != "" {
				continue
			}
			aliasedFields(a.Field(i), b.Field(i), path+"."+a.Type().Field(i).Name, out, depth+1)
		}
	}
}

// cloneFuncs lists every clone function of the shape func(*T) *T, which is all
// of them except the four noted below.
//
// Not covered here, deliberately:
//   - cloneEntity(interface{}) interface{} -- exercised via Record.Entity
//   - CloneTags([]*Tag) []*Tag             -- exercised via Record.Tags
//   - cloneStringSlice([]string) []string  -- no fields to lose
//   - cloneSourceRepositoryLink            -- exercised via Source.Repositories
var cloneFuncs = []struct {
	name string
	fn   interface{}
}{
	{"Document", (*Document).Clone},
	{"Header", (*Header).Clone},
	{"Trailer", (*Trailer).Clone},
	{"Record", (*Record).Clone},
	{"Tag", (*Tag).Clone},
	{"Individual", (*Individual).Clone},
	{"Family", (*Family).Clone},
	{"Source", (*Source).Clone},
	{"Repository", (*Repository).Clone},
	{"Note", (*Note).Clone},
	{"MediaObject", (*MediaObject).Clone},
	{"Submitter", (*Submitter).Clone},
	{"SharedNote", (*SharedNote).Clone},
	{"SchemaDefinition", cloneSchemaDefinition},
	{"PersonalName", clonePersonalName},
	{"Transliteration", cloneTransliteration},
	{"Association", cloneAssociation},
	{"Event", cloneEvent},
	{"Attribute", cloneAttribute},
	{"Date", cloneDate},
	{"PlaceDetail", clonePlaceDetail},
	{"Address", cloneAddress},
	{"SourceCitation", cloneSourceCitation},
	{"AncestryAPID", cloneAncestryAPID},
	{"MediaLink", cloneMediaLink},
	{"MediaFile", cloneMediaFile},
	{"LDSOrdinance", cloneLDSOrdinance},
	{"ChangeDate", cloneChangeDate},
	{"ExternalID", cloneExternalID},
	{"SourceRepositoryLink", cloneSourceRepositoryLink},
}

func TestCloneCompleteness(t *testing.T) {
	seen := map[string]bool{}
	seenShared := map[string]bool{}

	for _, tc := range cloneFuncs {
		t.Run(tc.name, func(t *testing.T) {
			fnVal := reflect.ValueOf(tc.fn)
			if fnVal.Kind() != reflect.Func || fnVal.Type().NumIn() != 1 || fnVal.Type().NumOut() != 1 {
				t.Fatalf("%s: unexpected clone signature %s", tc.name, fnVal.Type())
			}

			ptrType := fnVal.Type().In(0)
			orig := reflect.New(ptrType.Elem())
			f := &filler{}
			f.fill(orig.Elem(), tc.name, 0)

			cloned := fnVal.Call([]reflect.Value{orig})[0]
			if cloned.IsNil() {
				t.Fatalf("%s: clone returned nil for a non-nil input", tc.name)
			}

			var lost []string
			diffFields(orig, cloned, tc.name, &lost, 0)
			sort.Strings(lost)

			for _, path := range lost {
				seen[path] = true
				if reason, known := cloneKnownBad[path]; !known {
					t.Errorf("field does not survive Clone: %s", path)
				} else {
					_ = reason
				}
			}

			var shared []string
			aliasedFields(orig, cloned, tc.name, &shared, 0)
			sort.Strings(shared)

			for _, path := range shared {
				seenShared[path] = true
				if _, known := cloneSharedByDesign[path]; !known {
					t.Errorf("Clone shares memory with the original (aliased, not copied): %s", path)
				}
			}
		})
	}

	// Self-cleaning: an entry that no longer reproduces must be removed, or the
	// list rots into a record of bugs that were fixed years ago.
	var fixed []string
	for path, reason := range cloneKnownBad {
		if !seen[path] {
			fixed = append(fixed, fmt.Sprintf("%s (was: %s)", path, reason))
		}
	}
	if len(fixed) > 0 {
		sort.Strings(fixed)
		t.Errorf("these fields now survive Clone; remove them from cloneKnownBad:\n  %s",
			strings.Join(fixed, "\n  "))
	}

	var unshared []string
	for path, reason := range cloneSharedByDesign {
		if !seenShared[path] {
			unshared = append(unshared, fmt.Sprintf("%s (was: %s)", path, reason))
		}
	}
	if len(unshared) > 0 {
		sort.Strings(unshared)
		t.Errorf("these fields are no longer shared by Clone; remove them from cloneSharedByDesign:\n  %s",
			strings.Join(unshared, "\n  "))
	}
}

// TestAliasedFields pins the identity check itself: a shared backing array, map
// or pointer is reported at its path, including one nested inside a slice
// element or held in an interface, while nil and empty slices and maps -- which
// may legitimately share the zero-length address -- are not.
func TestAliasedFields(t *testing.T) {
	shared := []string{"@N1@"}
	sharedMap := map[string]string{"_X": "http://example.com/x"}
	sharedDate := &ChangeDate{}
	sharedInd := &Individual{}
	tests := []struct {
		name string
		a, b any
		want []string
	}{
		{"nil slices", &Individual{}, &Individual{}, nil},
		{"empty slices", &Individual{NoteXRefs: []string{}}, &Individual{NoteXRefs: make([]string, 0)}, nil},
		{"distinct slices", &Individual{NoteXRefs: []string{"@N1@"}}, &Individual{NoteXRefs: []string{"@N1@"}}, nil},
		{"shared slice", &Individual{NoteXRefs: shared}, &Individual{NoteXRefs: shared}, []string{"Individual.NoteXRefs"}},
		{"empty maps", &SchemaDefinition{TagMappings: map[string]string{}}, &SchemaDefinition{TagMappings: map[string]string{}}, nil},
		{"distinct maps", &SchemaDefinition{TagMappings: map[string]string{"_X": "x"}}, &SchemaDefinition{TagMappings: map[string]string{"_X": "x"}}, nil},
		{"shared map", &SchemaDefinition{TagMappings: sharedMap}, &SchemaDefinition{TagMappings: sharedMap}, []string{"SchemaDefinition.TagMappings"}},
		{"nil pointers", &Individual{}, &Individual{ChangeDate: &ChangeDate{}}, nil},
		{"distinct pointers", &Individual{ChangeDate: &ChangeDate{}}, &Individual{ChangeDate: &ChangeDate{}}, nil},
		{"shared pointer", &Individual{ChangeDate: sharedDate}, &Individual{ChangeDate: sharedDate}, []string{"Individual.ChangeDate"}},
		{"shared slice nested in element", &Individual{Events: []*Event{{Phone: shared}}}, &Individual{Events: []*Event{{Phone: shared}}}, []string{"Individual.Events.Phone"}},
		{"shared pointer in interface", &Record{Entity: sharedInd}, &Record{Entity: sharedInd}, []string{"Record.Entity"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			root := reflect.TypeOf(tt.a).Elem().Name()
			aliasedFields(reflect.ValueOf(tt.a), reflect.ValueOf(tt.b), root, &got, 0)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("aliasedFields = %v, want %v", got, tt.want)
			}
		})
	}
}
