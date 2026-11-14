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
//
// The walk visits every value of a document, so its per-value cost is what a
// large file pays (#846). It allocates nothing until it finds a nil: the path
// is a stack of steps rendered only by report, and each struct type's fields
// are planned once per walk.

package validator

import (
	"context"
	"reflect"
	"strconv"
	"strings"

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
// RecordXRef is the record holding the nil. A record with no XRef, or with an
// XRef another record shares, is also named by its index, so the path still
// locates it ("Records[4].Names[0]").
//
// A nil pointer that is not a slice element -- an unset optional field such as
// Event.ParsedDate, a nil Header, a Record with no Entity at all -- is not
// reported: it is how a document says "absent", not a lost element.
func validateNilElements(ctx context.Context, doc *gedcom.Document) []Issue {
	w := &nilWalker{}
	xrefCount := make(map[string]int, len(doc.Records))
	for _, record := range doc.Records {
		if record != nil && record.XRef != "" {
			xrefCount[record.XRef]++
		}
	}
	if doc.Header != nil {
		w.steps = append(w.steps[:0], nilStep{name: "Header"})
		w.walk(reflect.ValueOf(doc.Header).Elem())
	}
	for i, record := range doc.Records {
		// Stops early, with the issues found so far, once ctx is done.
		if ctx.Err() != nil {
			break
		}
		w.xref = ""
		w.steps = append(w.steps[:0], nilStep{name: "Records"}, nilStep{index: i, isIndex: true})
		if record == nil {
			w.report()
			continue
		}
		w.xref = record.XRef
		if record.XRef != "" && xrefCount[record.XRef] == 1 {
			w.steps = w.steps[:0]
		}
		w.walk(reflect.ValueOf(record).Elem())
	}
	return w.issues
}

// nilStep is one step of the path to the value being walked: a field name, or
// an index into the slice or array the step before it names.
type nilStep struct {
	name    string
	index   int
	isIndex bool
}

// nilField is a struct field the walk descends into.
type nilField struct {
	index int
	name  string
}

// nilWalker collects CodeNilElement issues for one document.
type nilWalker struct {
	issues []Issue

	// xref is the XRef of the record being walked, or "" outside one.
	xref string

	// steps is the path from the walk's root to its current position. It is
	// rendered only when a nil is reported.
	steps []nilStep

	// fields caches, per struct type, the fields that can hold a nil element.
	fields map[reflect.Type][]nilField

	// onPath holds the pointers between the walk's root and its current
	// position. A hand-built value can be cyclic (a Date whose EndDate is
	// itself); without this the walk would never end.
	onPath map[uintptr]bool
}

// path renders steps as Details["field"] spells it: "Events[2].Tags[0]".
func (w *nilWalker) path() string {
	var b strings.Builder
	for _, s := range w.steps {
		switch {
		case s.isIndex:
			b.WriteByte('[')
			b.WriteString(strconv.Itoa(s.index))
			b.WriteByte(']')
		case b.Len() > 0:
			b.WriteByte('.')
			b.WriteString(s.name)
		default:
			b.WriteString(s.name)
		}
	}
	return b.String()
}

// report records the value at the walk's current position as a nil element.
func (w *nilWalker) report() {
	path := w.path()
	w.issues = append(w.issues, NewIssue(
		SeverityError,
		CodeNilElement,
		path+" is nil",
		w.xref,
	).WithDetail("field", path))
}

// walk reports every nil slice element and typed-nil interface reachable from
// v, the value at the walk's current position.
func (w *nilWalker) walk(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		w.walkPointer(v)
	case reflect.Interface:
		if v.IsNil() {
			return
		}
		if e := v.Elem(); e.Kind() == reflect.Pointer && e.IsNil() {
			w.report()
			return
		}
		w.walk(v.Elem())
	case reflect.Struct:
		w.walkStruct(v)
	case reflect.Slice, reflect.Array:
		w.walkElements(v)
	}
}

// walkPointer walks the value a non-nil pointer points to, unless the walk is
// already inside it. A nil pointer that is not a slice element is an unset
// field, not a nil element, and is not reported.
func (w *nilWalker) walkPointer(v reflect.Value) {
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
	w.walk(v.Elem())
	delete(w.onPath, p)
}

// walkStruct walks every exported field that can hold a nil element.
func (w *nilWalker) walkStruct(v reflect.Value) {
	last := len(w.steps)
	w.steps = append(w.steps, nilStep{})
	for _, f := range w.fieldsOf(v.Type()) {
		w.steps[last].name = f.name
		w.walk(v.Field(f.index))
	}
	w.steps = w.steps[:last]
}

// fieldsOf returns the exported fields of struct type t that can hold a nil
// element.
func (w *nilWalker) fieldsOf(t reflect.Type) []nilField {
	fields, ok := w.fields[t]
	if ok {
		return fields
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.IsExported() && mayHoldElements(f.Type.Kind()) {
			fields = append(fields, nilField{index: i, name: f.Name})
		}
	}
	if w.fields == nil {
		w.fields = make(map[reflect.Type][]nilField)
	}
	w.fields[t] = fields
	return fields
}

// walkElements reports each nil element of a slice or array and walks the rest.
func (w *nilWalker) walkElements(v reflect.Value) {
	elemKind := v.Type().Elem().Kind()
	if !mayHoldElements(elemKind) {
		return
	}
	nilable := elemKind == reflect.Pointer || elemKind == reflect.Interface
	last := len(w.steps)
	w.steps = append(w.steps, nilStep{isIndex: true})
	for i := 0; i < v.Len(); i++ {
		w.steps[last].index = i
		e := v.Index(i)
		if nilable && e.IsNil() {
			w.report()
			continue
		}
		w.walk(e)
	}
	w.steps = w.steps[:last]
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
