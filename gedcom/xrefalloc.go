package gedcom

import (
	"strconv"
	"strings"
)

// XRefAllocator hands out XRefs that are free in the document it was made
// from: ones no record defines and no pointer names. Make one with
// [Document.NewXRefAllocator] to add several records; for a single record
// [Document.FreeXRef] is shorter.
//
// An allocator is a snapshot. It does not see records or pointers added to the
// document after it was made, only the XRefs it handed out itself, so make a
// new one after changing the document some other way. The zero value is an
// allocator for an empty document.
type XRefAllocator struct {
	used map[string]bool // XRefs defined, pointed to, or already handed out
	next map[string]int  // per prefix, the first number not yet tried
}

// NewXRefAllocator returns an allocator for the XRefs free in d. It walks d
// once, collecting:
//
//   - every key of XRefMap and every record's XRef, so a record is counted
//     whether or not it is indexed;
//   - every pointer in every record, found as [VisitForVersion] finds them
//     under the version d's header declares (Header.DeclaredVersion), so a
//     spaced pointer ("@N 1@") counts in a document declaring GEDCOM 5.5 or
//     5.5.1;
//   - the header's pointers, Header.Submitter and pointer-valued Header.Tags,
//     under the same grammar.
//
// It also reserves the whitespace-trimmed value of each raw tag (Record.Tags
// and Header.Tags) when that is pointer-shaped under the declared version
// ("@O1@ " with a trailing space), so the set is never smaller than what a
// lenient reader would treat as a pointer.
//
// Counting pointers keeps a new record from taking the XRef of a dangling
// pointer and silently becoming its target. A nil d, like an empty one, has
// no XRef in use.
func (d *Document) NewXRefAllocator() *XRefAllocator {
	a := &XRefAllocator{used: map[string]bool{}, next: map[string]int{}}
	if d == nil {
		return a
	}
	version := d.declaredVersion()
	reserve := func(xref string) { a.used[xref] = true }
	reserveTrimmed := func(tags []*Tag) {
		for _, t := range tags {
			if t == nil {
				continue
			}
			if v := strings.TrimSpace(t.Value); IsPointerXRefForVersion(v, version) {
				reserve(v)
			}
		}
	}

	for xref := range d.XRefMap {
		reserve(xref)
	}
	for _, r := range d.Records {
		if r == nil {
			continue
		}
		if r.XRef != "" {
			reserve(r.XRef)
		}
		VisitForVersion(r, version, reserve)
		reserveTrimmed(r.Tags)
	}
	if h := d.Header; h != nil {
		// The header is not a record; wrapping its pointer-bearing values in
		// one classifies them by the same rule as every record's.
		tags := make([]*Tag, 0, len(h.Tags)+1)
		tags = append(tags, &Tag{Value: h.Submitter})
		tags = append(tags, h.Tags...)
		VisitForVersion(&Record{Tags: tags}, version, reserve)
		reserveTrimmed(h.Tags)
	}
	return a
}

// Next returns the first free "@<prefix><n>@", n counting from 1, and
// reserves it, so no later call returns it again. Each prefix counts on its
// own: Next("I") and Next("O") can both return a number 1.
//
// The prefix is not checked; the validator reports a malformed XRef. Use one
// the document's version allows (for GEDCOM 7.0, uppercase letters, digits
// and "_").
func (a *XRefAllocator) Next(prefix string) string {
	if a.used == nil {
		a.used = map[string]bool{}
	}
	if a.next == nil {
		a.next = map[string]int{}
	}
	n := a.next[prefix]
	if n == 0 {
		n = 1
	}
	for {
		xref := "@" + prefix + strconv.Itoa(n) + "@"
		n++
		if !a.used[xref] {
			a.used[xref] = true
			a.next[prefix] = n
			return xref
		}
	}
}

// FreeXRef returns the first "@<prefix><n>@", n counting from 1, that no
// record in d defines and no pointer in d names; see
// [Document.NewXRefAllocator] for what is counted and [XRefAllocator.Next]
// for the prefix.
//
// It reserves nothing and walks d on every call, so two calls return the same
// XRef unless a record taking the first was added in between (AddRecord). To
// add several records, make one allocator with NewXRefAllocator and call Next
// for each.
func (d *Document) FreeXRef(prefix string) string {
	return d.NewXRefAllocator().Next(prefix)
}
