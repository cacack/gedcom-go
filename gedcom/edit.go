package gedcom

import "fmt"

// Reference is one pointer to a record: where it is, not what it names.
type Reference struct {
	// XRef is the record holding the pointer, or "" for the header.
	XRef string

	// Path is the tag path to the pointer's line, from the level-0 tag
	// down: ["INDI", "FAMC"], ["INDI", "BIRT", "SOUR"], ["HEAD", "SUBM"].
	// A record with no raw Tags (one built in code) has no lines, so a
	// pointer in its Entity has a Path of the record's type alone.
	Path []string
}

// ReplaceRecord puts r in place of the record d indexes under r.XRef: at
// the same position in d.Records, and in d.XRefMap, so GetRecord and the
// typed lookups (GetIndividual, GetFamily, ...) return r. Nothing else
// changes; r may be of another type than the record it replaces.
//
// It returns an error wrapping ErrUnknownXRef, and leaves d unchanged, when
// d indexes no record under r.XRef, including when r.XRef is empty: use
// AddRecord to add a record. It returns ErrNilRecord for a nil r and
// ErrNilDocument for a nil d. When records share an XRef, the one indexed
// (the first, see ReindexXRefs) is replaced.
//
// With a layout-preserving encode (encoder.EncodeOptions.PreserveLayout),
// every other record is still written as read. r's lines are written as
// new lines unless they keep the LineNumber of a source line they still read
// as: a record from decoder.DecodeRecords has none, and an edited Clone of
// the old record keeps its unchanged lines.
func (d *Document) ReplaceRecord(r *Record) error {
	if d == nil {
		return ErrNilDocument
	}
	if r == nil {
		return ErrNilRecord
	}
	i := d.indexOf(r.XRef)
	if i < 0 {
		return fmt.Errorf("replace record %q: %w", r.XRef, ErrUnknownXRef)
	}
	d.Records[i] = r
	d.XRefMap[r.XRef] = r
	return nil
}

// ReplaceHeader puts h in place of d's header, and sets the two views of the
// header d holds beside it: Vendor, detected from h as DetectVendorFromHeader
// detects it, and Schema, from h's SCHMA structure when h.Version is GEDCOM
// 7.0 (nil otherwise). Records are not changed.
//
// h is taken as it is: nothing in it is set or checked. Its Version,
// DeclaredVersion and DetectedEncoding must already say what d holds, since
// decoder.DecodeRecords and decoder.SyncEntities read them and Schema is
// read only when Version is 7.0; decoder.DecodeHeader sets all three from d.
//
// It returns ErrNilHeader for a nil h and ErrNilDocument for a nil d, and
// leaves d unchanged.
//
// With a layout-preserving encode (encoder.EncodeOptions.PreserveLayout),
// every record and the trailer are still written as read, and so is the
// "0 HEAD" line. h's lines are written as new lines unless they keep the
// LineNumber of a source line they still read as: a header from
// decoder.DecodeHeader has none.
func (d *Document) ReplaceHeader(h *Header) error {
	if d == nil {
		return ErrNilDocument
	}
	if h == nil {
		return ErrNilHeader
	}
	d.Header = h
	d.Vendor = DetectVendorFromHeader(h)
	d.Schema = headerSchema(h)
	return nil
}

// RemoveRecord removes the record d indexes under xref from d.Records and
// d.XRefMap, and returns every pointer to xref left in the rest of d: in its
// records' raw Tags (or, for a record without Tags, its Entity) and in the
// header, in document order. Those pointers are reported, not changed; they
// dangle unless another record shares xref, which is then indexed in its
// place as ReindexXRefs would index it. A pointer is matched by the rule of
// IsRecordPointer, so a GEDCOM 7.0 "@VOID@" null pointer is never reported.
// A record's Tags, when it has any, are what the encoder writes and so what is
// searched: an Entity edited since decoding is seen only once its edit is
// carried into Tags (encoder.SyncTags).
//
// It returns an error wrapping ErrUnknownXRef, and leaves d unchanged, when d
// indexes no record under xref, and ErrNilDocument for a nil d.
func (d *Document) RemoveRecord(xref string) ([]Reference, error) {
	if d == nil {
		return nil, ErrNilDocument
	}
	i := d.indexOf(xref)
	if i < 0 {
		return nil, fmt.Errorf("remove record %q: %w", xref, ErrUnknownXRef)
	}
	// Decided before the record goes: a record named @VOID@ is what makes
	// @VOID@ an identifier in a document that declares no version.
	isPointer := d.IsRecordPointer(xref)

	d.Records = append(d.Records[:i:i], d.Records[i+1:]...)
	delete(d.XRefMap, xref)
	for _, r := range d.Records {
		if r != nil && r.XRef == xref {
			d.XRefMap[xref] = r
			break
		}
	}
	if !isPointer {
		return nil, nil
	}
	return d.referencesTo(xref), nil
}

// indexOf returns the position in d.Records of the record d.XRefMap indexes
// under xref, or -1 when there is none. A nil XRefMap is rebuilt from Records
// first, as AddRecord rebuilds it.
func (d *Document) indexOf(xref string) int {
	if xref == "" {
		return -1
	}
	if d.XRefMap == nil {
		d.ReindexXRefs()
	}
	target, ok := d.XRefMap[xref]
	if !ok {
		return -1
	}
	for i, r := range d.Records {
		if r == target {
			return i
		}
	}
	return -1
}

// referencesTo returns every pointer to xref in d's header and records, as
// RemoveRecord reports them.
func (d *Document) referencesTo(xref string) []Reference {
	var refs []Reference
	if h := d.Header; h != nil {
		if len(h.Tags) > 0 {
			refs = appendTagReferences(refs, "", "HEAD", h.Tags, xref)
		} else if h.Submitter == xref {
			refs = append(refs, Reference{Path: []string{"HEAD", "SUBM"}})
		}
	}
	for _, r := range d.Records {
		if r == nil {
			continue
		}
		if len(r.Tags) > 0 {
			refs = appendTagReferences(refs, r.XRef, string(r.Type), r.Tags, xref)
			continue
		}
		walkEntity(r.Entity, func(p *string) {
			if p != nil && *p == xref {
				refs = append(refs, Reference{XRef: r.XRef, Path: []string{string(r.Type)}})
			}
		})
	}
	return refs
}

// appendTagReferences appends to refs each tag in tags whose value is xref,
// with the tag path to it from the level-0 tag root. A tag's parent is the
// nearest earlier tag one level shallower, as for Record.FindAll; a tag whose
// level skips past its parent's child level is placed under the deepest
// tag open.
func appendTagReferences(refs []Reference, owner, root string, tags []*Tag, xref string) []Reference {
	path := []string{root}
	for _, t := range tags {
		if t == nil || t.Level < 1 {
			continue
		}
		if depth := t.Level; depth < len(path) {
			path = path[:depth]
		}
		path = append(path, t.Tag)
		if t.Value == xref {
			refs = append(refs, Reference{XRef: owner, Path: append([]string(nil), path...)})
		}
	}
	return refs
}
