package validator

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// nilIssues returns the CodeNilElement issues ValidateAll reports for doc.
func nilIssues(doc *gedcom.Document) []Issue {
	var out []Issue
	for _, issue := range New().ValidateAll(doc) {
		if issue.Code == CodeNilElement {
			out = append(out, issue)
		}
	}
	return out
}

// nilFields returns "RecordXRef field" for each issue, the shape a test can
// compare in one line.
func nilFields(issues []Issue) []string {
	out := make([]string, 0, len(issues))
	for _, issue := range issues {
		if issue.Severity != SeverityError {
			panic("NIL_ELEMENT issue with severity " + issue.Severity.String())
		}
		if issue.Message != issue.Details["field"]+" is nil" {
			panic("NIL_ELEMENT message " + issue.Message + " does not name field " + issue.Details["field"])
		}
		out = append(out, strings.TrimSpace(issue.RecordXRef+" "+issue.Details["field"]))
	}
	return out
}

func TestValidateAll_NilElementShapes(t *testing.T) {
	citation := &gedcom.SourceCitation{SourceXRef: "@S1@"}
	tests := []struct {
		name string
		doc  *gedcom.Document
		want []string
	}{
		{
			name: "nil record",
			doc: &gedcom.Document{Records: []*gedcom.Record{
				{XRef: "@I1@", Type: gedcom.RecordTypeIndividual}, nil,
			}},
			want: []string{"Records[1]"},
		},
		{
			name: "nil tag in a record",
			doc: &gedcom.Document{Records: []*gedcom.Record{
				{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Tags: []*gedcom.Tag{{Level: 1, Tag: "NAME"}, nil}},
			}},
			want: []string{"@I1@ Tags[1]"},
		},
		{
			name: "nil tag in the header",
			doc:  &gedcom.Document{Header: &gedcom.Header{Tags: []*gedcom.Tag{nil}}},
			want: []string{"Header.Tags[0]"},
		},
		{
			name: "typed-nil entity",
			doc: &gedcom.Document{Records: []*gedcom.Record{
				{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: (*gedcom.Family)(nil)},
			}},
			want: []string{"@F1@ Entity"},
		},
		{
			name: "nested nil citation",
			doc: &gedcom.Document{Records: []*gedcom.Record{{
				XRef: "@I1@", Type: gedcom.RecordTypeIndividual,
				Entity: &gedcom.Individual{XRef: "@I1@", Events: []*gedcom.Event{
					{Type: gedcom.EventBirth},
					{Type: gedcom.EventDeath, SourceCitations: []*gedcom.SourceCitation{citation, nil}},
				}},
			}}},
			want: []string{"@I1@ Entity.Events[1].SourceCitations[1]"},
		},
		{
			name: "record with no XRef is named by index",
			doc: &gedcom.Document{Records: []*gedcom.Record{
				{XRef: "@I1@", Type: gedcom.RecordTypeIndividual},
				{Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{Names: []*gedcom.PersonalName{nil}}},
			}},
			want: []string{"Records[1].Entity.Names[0]"},
		},
		{
			name: "every nil is reported",
			doc: &gedcom.Document{Records: []*gedcom.Record{
				nil,
				{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Tags: []*gedcom.Tag{nil, nil}},
				nil,
			}},
			want: []string{"Records[0]", "@I1@ Tags[0]", "@I1@ Tags[1]", "Records[2]"},
		},
		{
			// Absent, not lost: none of these is a nil element.
			name: "nil header, untyped-nil entity and unset optional fields",
			doc: &gedcom.Document{Records: []*gedcom.Record{
				{XRef: "@I1@", Type: gedcom.RecordTypeIndividual},
				{XRef: "@I2@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{
					Events: []*gedcom.Event{{Type: gedcom.EventBirth}},
				}},
			}},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nilFields(nilIssues(tt.doc))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NIL_ELEMENT fields = %q, want %q", got, tt.want)
			}
		})
	}
}

// A hand-built value can point back at itself. The walk must still end.
func TestValidateAll_NilElementCycle(t *testing.T) {
	date := &gedcom.Date{Year: 1900}
	date.EndDate = date
	doc := &gedcom.Document{Records: []*gedcom.Record{{
		XRef: "@I1@", Type: gedcom.RecordTypeIndividual,
		Entity: &gedcom.Individual{XRef: "@I1@", Events: []*gedcom.Event{{ParsedDate: date}, nil}},
	}}}

	want := []string{"@I1@ Entity.Events[1]"}
	if got := nilFields(nilIssues(doc)); !reflect.DeepEqual(got, want) {
		t.Errorf("NIL_ELEMENT fields = %q, want %q", got, want)
	}
}

func TestValidateAll_NilElementSkipRule(t *testing.T) {
	doc := &gedcom.Document{Records: []*gedcom.Record{nil}}
	if got := NewWithConfig(&ValidatorConfig{SkipRules: []string{CodeNilElement}}).ValidateAll(doc); len(got) != 0 {
		t.Errorf("ValidateAll() with NIL_ELEMENT skipped = %v, want none", got)
	}
}

// A decoded document never holds a nil, so no decoded file reports one.
func TestValidateAll_NilElementDecodedFiles(t *testing.T) {
	for _, path := range []string{
		"../testdata/gedcom-5.5/royal92.ged",
		"../testdata/gedcom-5.5.1/comprehensive.ged",
		"../testdata/gedcom-7.0/maximal70.ged",
	} {
		t.Run(path, func(t *testing.T) {
			doc := mustDecodeFile(t, path)
			if got := nilIssues(doc); len(got) != 0 {
				t.Errorf("decoded %s reported %v", path, got)
			}
		})
	}
}

// entityTypes are the types a decoded Record.Entity holds, plus the header,
// which is walked the same way.
var entityTypes = []interface{}{
	gedcom.Individual{}, gedcom.Family{}, gedcom.Source{}, gedcom.Repository{},
	gedcom.Note{}, gedcom.MediaObject{}, gedcom.Submitter{}, gedcom.SharedNote{},
	gedcom.Header{},
}

// TestValidateAll_NilElementEveryField puts a nil in every slice-of-pointer
// field reachable from every entity type -- top-level and nested, found by
// reflection so a field added later is covered without editing this test --
// and checks the rule reports exactly that nil, at exactly that path.
func TestValidateAll_NilElementEveryField(t *testing.T) {
	for _, entity := range entityTypes {
		typ := reflect.TypeOf(entity)
		paths := sliceOfPointerPaths(typ, nil, map[reflect.Type]bool{})
		if len(paths) == 0 {
			t.Errorf("%s has no slice-of-pointer field; drop it from entityTypes or fix the enumeration", typ.Name())
		}

		for _, path := range paths {
			value, field := buildWithNil(typ, path)
			t.Run(typ.Name()+"."+field, func(t *testing.T) {
				var doc *gedcom.Document
				var want string
				if header, ok := value.(*gedcom.Header); ok {
					doc = &gedcom.Document{Header: header}
					want = "Header." + field
				} else {
					doc = &gedcom.Document{Records: []*gedcom.Record{{XRef: "@X1@", Entity: value}}}
					want = "@X1@ Entity." + field
				}

				got := nilFields(nilIssues(doc))
				if len(got) != 1 || got[0] != want {
					t.Errorf("NIL_ELEMENT fields = %q, want [%q]", got, want)
				}
			})
		}
	}
}

// fieldStep is one field on the way from an entity to a slice-of-pointer field.
type fieldStep struct {
	index int
	name  string
}

// sliceOfPointerPaths returns the path to every slice-of-pointer field
// reachable from struct type t, through pointer, struct and slice fields. A
// type already on the path is not entered again, so a recursive type such as
// Date (EndDate *Date) ends the search.
func sliceOfPointerPaths(t reflect.Type, prefix []fieldStep, onPath map[reflect.Type]bool) [][]fieldStep {
	onPath[t] = true
	defer delete(onPath, t)

	var out [][]fieldStep
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		path := append(append([]fieldStep{}, prefix...), fieldStep{i, f.Name})

		ft := f.Type
		if ft.Kind() == reflect.Slice {
			if ft.Elem().Kind() == reflect.Pointer {
				out = append(out, path)
			}
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct && !onPath[ft] {
			out = append(out, sliceOfPointerPaths(ft, path, onPath)...)
		}
	}
	return out
}

// buildWithNil returns a new *t whose only content is the chain of elements
// along path, ending in a one-element slice holding nil, and the reported path
// of that nil relative to the entity.
func buildWithNil(t reflect.Type, path []fieldStep) (value interface{}, field string) {
	root := reflect.New(t)
	cur := root.Elem()
	var names []string
	for i, step := range path {
		f := cur.Field(step.index)
		name := step.name
		if f.Kind() == reflect.Slice {
			f.Set(reflect.MakeSlice(f.Type(), 1, 1))
			name += "[0]"
			f = f.Index(0)
		}
		names = append(names, name)
		if i == len(path)-1 {
			break // f is the nil element
		}
		if f.Kind() == reflect.Pointer {
			f.Set(reflect.New(f.Type().Elem()))
			f = f.Elem()
		}
		cur = f
	}
	return root.Interface(), strings.Join(names, ".")
}

func mustDecodeFile(t *testing.T, path string) *gedcom.Document {
	t.Helper()
	f, err := os.Open(path) // #nosec G304 -- path is a constant in the repo's own testdata
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	doc, err := decoder.Decode(f)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return doc
}
