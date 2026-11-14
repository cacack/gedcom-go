package validator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// The nil-element walk was rewritten for speed (#846): it keeps its path as a
// stack, plans each struct type's fields once, and does not enter types that
// cannot hold a nil element. None of that may change what it reports. The
// walk below is the one it replaced, kept as the reference: plain reflection
// over every exported field, a path string built at every step.

// referenceNilElements is validateNilElements as it was before #846.
func referenceNilElements(doc *gedcom.Document) []Issue {
	w := &referenceNilWalker{onPath: make(map[uintptr]bool)}
	xrefCount := make(map[string]int, len(doc.Records))
	for _, record := range doc.Records {
		if record != nil && record.XRef != "" {
			xrefCount[record.XRef]++
		}
	}
	if doc.Header != nil {
		w.walk(reflect.ValueOf(doc.Header).Elem(), "Header")
	}
	for i, record := range doc.Records {
		w.xref = ""
		path := fmt.Sprintf("Records[%d]", i)
		if record == nil {
			w.report(path)
			continue
		}
		w.xref = record.XRef
		if record.XRef != "" && xrefCount[record.XRef] == 1 {
			path = ""
		}
		w.walk(reflect.ValueOf(record).Elem(), path)
	}
	return w.issues
}

type referenceNilWalker struct {
	issues []Issue
	xref   string
	onPath map[uintptr]bool
}

func (w *referenceNilWalker) report(path string) {
	w.issues = append(w.issues, NewIssue(SeverityError, CodeNilElement, path+" is nil", w.xref).WithDetail("field", path))
}

func (w *referenceNilWalker) walk(v reflect.Value, path string) {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() || w.onPath[v.Pointer()] {
			return
		}
		w.onPath[v.Pointer()] = true
		w.walk(v.Elem(), path)
		delete(w.onPath, v.Pointer())
	case reflect.Interface:
		if v.IsNil() {
			return
		}
		if e := v.Elem(); e.Kind() == reflect.Pointer && e.IsNil() {
			w.report(path)
			return
		}
		w.walk(v.Elem(), path)
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			field := f.Name
			if path != "" {
				field = path + "." + f.Name
			}
			w.walk(v.Field(i), field)
		}
	case reflect.Slice, reflect.Array:
		elemKind := v.Type().Elem().Kind()
		nilable := elemKind == reflect.Pointer || elemKind == reflect.Interface
		for i := 0; i < v.Len(); i++ {
			e := v.Index(i)
			elemPath := fmt.Sprintf("%s[%d]", path, i)
			if nilable && e.IsNil() {
				w.report(elemPath)
				continue
			}
			w.walk(e, elemPath)
		}
	}
}

// nilInjector plants nils in a decoded document: every nth nilable slice
// element it meets becomes nil, and every nth non-nil interface holding a
// pointer becomes a typed nil. A decoded document has neither, so without
// this both walks would agree on reporting nothing.
type nilInjector struct {
	every   int
	met     int
	planted int
	seen    map[uintptr]bool
}

func (in *nilInjector) due() bool {
	in.met++
	if in.met%in.every != 0 {
		return false
	}
	in.planted++
	return true
}

func (in *nilInjector) inject(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() || in.seen[v.Pointer()] {
			return
		}
		in.seen[v.Pointer()] = true
		in.inject(v.Elem())
	case reflect.Interface:
		if v.IsNil() {
			return
		}
		if e := v.Elem(); e.Kind() == reflect.Pointer && v.CanSet() && in.due() {
			v.Set(reflect.Zero(e.Type()))
			return
		}
		in.inject(v.Elem())
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				in.inject(v.Field(i))
			}
		}
	case reflect.Slice, reflect.Array:
		elemKind := v.Type().Elem().Kind()
		nilable := elemKind == reflect.Pointer || elemKind == reflect.Interface
		for i := 0; i < v.Len(); i++ {
			e := v.Index(i)
			if nilable && e.CanSet() && !e.IsNil() && in.due() {
				e.Set(reflect.Zero(e.Type()))
				continue
			}
			in.inject(e)
		}
	}
}

// requireSameNilIssues fails unless the walk and the reference report the same
// issues for doc, field for field and in the same order.
func requireSameNilIssues(t *testing.T, doc *gedcom.Document) []Issue {
	t.Helper()
	got := validateNilElements(context.Background(), doc)
	want := referenceNilElements(doc)
	if len(got) != len(want) {
		t.Fatalf("validateNilElements() reported %d issues, the reference walk %d", len(got), len(want))
	}
	for i := range want {
		if !reflect.DeepEqual(got[i], want[i]) {
			t.Fatalf("issue %d = %+v, the reference walk has %+v", i, got[i], want[i])
		}
	}
	return got
}

// Every fixture the decoder accepts, as decoded and then with nils planted at
// three densities, gets the same issues from the walk and from the reference.
//
// The two fixtures over 2 MiB are left to the tests that validate the whole
// corpus: the reference walk is the slow one #846 replaced, it runs four times
// a file here, and those files' types are no different from the others'.
func TestValidateNilElements_MatchesReferenceWalk(t *testing.T) {
	t.Parallel()
	const maxSize = 2 << 20

	var files []string
	err := filepath.Walk("../testdata", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".ged") && info.Size() <= maxSize {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking testdata: %v", err)
	}

	// The group waits for every subtest, so the totals are final after it.
	type total struct{ docs, planted, reported int }
	totals := make(chan total, len(files))
	t.Run("fixtures", func(t *testing.T) {
		for _, path := range files {
			t.Run(strings.TrimPrefix(filepath.ToSlash(path), "../testdata/"), func(t *testing.T) {
				t.Parallel()
				f, err := os.Open(path) // #nosec G304 -- path comes from walking the repo's own testdata
				if err != nil {
					t.Fatalf("open %s: %v", path, err)
				}
				doc, err := decoder.Decode(f)
				_ = f.Close()
				if err != nil {
					return // malformed fixtures exercise the decoder, not this
				}

				sum := total{docs: 1}
				if issues := requireSameNilIssues(t, doc); len(issues) != 0 {
					t.Errorf("a decoded document has %d nil elements, want none", len(issues))
				}
				// Sparse first: each pass adds nils to those already there.
				for _, every := range []int{97, 7, 2} {
					in := &nilInjector{every: every, seen: make(map[uintptr]bool)}
					in.inject(reflect.ValueOf(doc))
					sum.planted += in.planted
					sum.reported += len(requireSameNilIssues(t, doc))
				}
				totals <- sum
			})
		}
	})
	close(totals)

	var sum total
	for s := range totals {
		sum.docs += s.docs
		sum.planted += s.planted
		sum.reported += s.reported
	}
	if sum.docs == 0 || sum.planted == 0 || sum.reported == 0 {
		t.Fatalf("compared %d documents, %d planted nils, %d reported issues; the comparison would be vacuous", sum.docs, sum.planted, sum.reported)
	}
	t.Logf("compared %d documents: %d nils planted, %d issues reported identically", sum.docs, sum.planted, sum.reported)
}

// A type that contains itself must not send the type search round forever,
// and a nilable slice reached only through the cycle must still be found.
func TestCanHoldElements(t *testing.T) {
	type leaf struct {
		Name  string
		Count [2]int
		Words []string
	}
	type loop struct {
		Next *loop
		Leaf leaf
	}
	type loopWithSlice struct {
		Next  *loopWithSlice
		Leafs []*leaf
	}
	type hidden struct {
		Name  string
		leafs []*leaf //nolint:unused // unexported: the walk never reads it
	}
	tests := []struct {
		name string
		typ  reflect.Type
		want bool
	}{
		{"flat struct", reflect.TypeOf(leaf{}), false},
		{"pointer to flat struct", reflect.TypeOf(&leaf{}), false},
		{"slice of flat structs", reflect.TypeOf([]leaf{}), false},
		{"slice of pointers", reflect.TypeOf([]*leaf{}), true},
		{"slice of structs holding a pointer slice", reflect.TypeOf([]loopWithSlice{}), true},
		{"array of structs holding a pointer slice", reflect.TypeOf([1]loopWithSlice{}), true},
		{"array of pointers", reflect.TypeOf([1]*leaf{}), true},
		{"slice of interfaces", reflect.TypeOf([]any{}), true},
		{"interface field", reflect.TypeOf(gedcom.Record{}), true},
		{"self-referential with nothing nilable", reflect.TypeOf(loop{}), false},
		{"self-referential with a pointer slice", reflect.TypeOf(loopWithSlice{}), true},
		{"unexported pointer slice", reflect.TypeOf(hidden{}), false},
		{"tag", reflect.TypeOf(gedcom.Tag{}), false},
		{"string", reflect.TypeOf(""), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canHoldElements(tt.typ, make(map[reflect.Type]bool)); got != tt.want {
				t.Errorf("canHoldElements(%v) = %v, want %v", tt.typ, got, tt.want)
			}
		})
	}
}

// The fixtures hold only the shapes gedcom types have. These are shapes they
// lack, each of which one guard in the walk decides: an unexported field is
// not read, a slice of struct values is entered for the slices inside them,
// and a pointer reached twice without a cycle is walked both times.
func TestNilWalker_ShapesOutsideGedcomTypes(t *testing.T) {
	type inner struct {
		Ptrs []*int
	}
	type hiddenSlice struct {
		Kept   []*int
		hidden []*int
	}
	type structSlice struct {
		Items []inner
		Fixed [2]inner
	}
	type shared struct {
		A, B *inner
	}
	n := 1
	both := &inner{Ptrs: []*int{nil, &n}}

	tests := []struct {
		name  string
		value any
		want  []string
	}{
		{
			"unexported field is not read",
			hiddenSlice{Kept: []*int{&n, nil}, hidden: []*int{nil}},
			[]string{"Kept[1]"},
		},
		{
			"slice and array of struct values are entered",
			structSlice{
				Items: []inner{{}, {Ptrs: []*int{&n, nil}}},
				Fixed: [2]inner{{Ptrs: []*int{nil}}, {}},
			},
			[]string{"Items[1].Ptrs[1]", "Fixed[0].Ptrs[0]"},
		},
		{
			"pointer shared by two fields is walked twice",
			shared{A: both, B: both},
			[]string{"A.Ptrs[0]", "B.Ptrs[0]"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &nilWalker{}
			w.walk(reflect.ValueOf(tt.value))
			var got []string
			for _, issue := range w.issues {
				got = append(got, issue.Details["field"])
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("walk reported %q, want %q", got, tt.want)
			}

			ref := &referenceNilWalker{onPath: make(map[uintptr]bool)}
			ref.walk(reflect.ValueOf(tt.value), "")
			if !reflect.DeepEqual(w.issues, ref.issues) {
				t.Errorf("walk = %v, reference walk = %v", w.issues, ref.issues)
			}
		})
	}
}
