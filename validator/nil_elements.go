// nil_elements.go reports the nils a hand-built document can hold, which the
// rest of the library skips silently (docs/decisions/0007-error-transparency.md).
//
// A decoded document never holds a nil. One built in code can: a lookup that
// returned nil, an append of a function result that was nil. The encoder skips
// such an element and writes a clean file with one fewer name, event or citation
// than the caller thinks they wrote, so a nil almost always means a bug in the
// code that built the document. This rule is where that bug becomes visible.
//
// The walk is reflective rather than a hand-written list of fields like
// gedcom/xrefwalk.go. Entities carry over a hundred slice-of-pointer fields,
// nested several levels deep; a second hand-kept list of them would drift as
// entities gain fields, and a field it missed would hide exactly the nil this
// rule exists to report. Reflection reaches every exported field of every
// type, today's and tomorrow's.

package validator

import (
	"fmt"
	"reflect"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// validateNilElements reports each nil a document holds where an element is
// expected, as CodeNilElement with Details["field"] naming the path to it:
//
//   - a nil *Record in Document.Records ("Records[7]");
//   - a Record.Entity holding a typed nil pointer ("Entity");
//   - a nil element in any slice of a record, its entity or the header, at any
//     depth ("Tags[3]", "Events[2].SourceCitations[1]", "Header.Tags[0]").
//
// RecordXRef is the record holding the nil. A record with no XRef is named by
// its index instead, so the path still locates it ("Records[4].Names[0]").
//
// A nil pointer that is not a slice element -- an unset optional field such as
// Event.ParsedDate, a nil Header, a Record with no Entity at all -- is not
// reported: it is how a document says "absent", not a lost element.
func validateNilElements(doc *gedcom.Document) []Issue {
	w := &nilWalker{}
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
		if record.XRef != "" {
			w.xref = record.XRef
			path = ""
		}
		w.walk(reflect.ValueOf(record).Elem(), path)
	}
	return w.issues
}

// nilWalker collects CodeNilElement issues for one document.
type nilWalker struct {
	issues []Issue

	// xref is the XRef of the record being walked, or "" outside one.
	xref string

	// onPath holds the pointers between the walk's root and its current
	// position. A hand-built value can be cyclic (a Date whose EndDate is
	// itself); without this the walk would never end.
	onPath map[uintptr]bool
}

func (w *nilWalker) report(path string) {
	w.issues = append(w.issues, NewIssue(
		SeverityError,
		CodeNilElement,
		path+" is nil",
		w.xref,
	).WithDetail("field", path))
}

// walk reports every nil slice element and typed-nil interface reachable from
// v, whose own path is path.
func (w *nilWalker) walk(v reflect.Value, path string) {
	switch v.Kind() {
	case reflect.Pointer:
		w.walkPointer(v, path)
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
		w.walkStruct(v, path)
	case reflect.Slice, reflect.Array:
		w.walkElements(v, path)
	}
}

// walkPointer walks the value a non-nil pointer points to, unless the walk is
// already inside it. A nil pointer that is not a slice element is an unset
// field, not a nil element, and is not reported.
func (w *nilWalker) walkPointer(v reflect.Value, path string) {
	if v.IsNil() {
		return
	}
	p := v.Pointer()
	if w.onPath[p] {
		return
	}
	if w.onPath == nil {
		w.onPath = make(map[uintptr]bool)
	}
	w.onPath[p] = true
	w.walk(v.Elem(), path)
	delete(w.onPath, p)
}

// walkStruct walks every exported field that can hold a nil element.
func (w *nilWalker) walkStruct(v reflect.Value, path string) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() || !mayHoldElements(f.Type.Kind()) {
			continue
		}
		field := f.Name
		if path != "" {
			field = path + "." + f.Name
		}
		w.walk(v.Field(i), field)
	}
}

// walkElements reports each nil element of a slice or array and walks the rest.
func (w *nilWalker) walkElements(v reflect.Value, path string) {
	elemKind := v.Type().Elem().Kind()
	if !mayHoldElements(elemKind) {
		return
	}
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

// mayHoldElements reports whether a value of kind k can be, or contain, a
// nil element. Strings, numbers and the like cannot, and are not walked.
func mayHoldElements(k reflect.Kind) bool {
	switch k {
	case reflect.Pointer, reflect.Interface, reflect.Struct, reflect.Slice, reflect.Array:
		return true
	default:
		return false
	}
}
