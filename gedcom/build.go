package gedcom

import "fmt"

// NewDocument returns an empty document for building in memory: a header whose
// Version is version, an empty Records slice, an empty XRefMap and a Trailer.
//
// When version is one this package recognizes (Version.IsValid), the header's
// DeclaredVersion is set to it too, so the document applies that version's
// identifier grammar (Document.IsPointerXRef) as a decoded file declaring it
// would. Otherwise DeclaredVersion is left empty, as the decoder leaves it for
// a version it does not recognize.
//
// Add records with AddRecord, which keeps XRefMap in step with Records.
func NewDocument(version Version) *Document {
	header := &Header{Version: version}
	if version.IsValid() {
		header.DeclaredVersion = version
	}
	return &Document{
		Header:  header,
		Records: []*Record{},
		XRefMap: make(map[string]*Record),
		Trailer: &Trailer{},
	}
}

// AddRecord appends r to d.Records and indexes it in d.XRefMap under r.XRef,
// so GetRecord and the typed lookups (GetIndividual, GetFamily, ...) find it.
//
// It returns an error wrapping ErrDuplicateXRef, and leaves d unchanged, when
// d.XRefMap already holds r.XRef. A record with an empty XRef is appended but
// not indexed: there is nothing to look it up by. It returns ErrNilRecord for
// a nil r and ErrNilDocument for a nil d. The XRef's shape is not checked; the
// validator reports a malformed one.
//
// AddRecord trusts XRefMap. After editing Records or XRefMap directly, call
// ReindexXRefs before adding more records so duplicates are still caught. A
// nil XRefMap, as in a Document literal, is rebuilt from Records first.
func (d *Document) AddRecord(r *Record) error {
	if d == nil {
		return ErrNilDocument
	}
	if r == nil {
		return ErrNilRecord
	}
	if d.XRefMap == nil {
		d.ReindexXRefs()
	}
	if r.XRef != "" {
		if _, exists := d.XRefMap[r.XRef]; exists {
			return fmt.Errorf("add record %q: %w", r.XRef, ErrDuplicateXRef)
		}
		d.XRefMap[r.XRef] = r
	}
	d.Records = append(d.Records, r)
	return nil
}

// ReindexXRefs rebuilds d.XRefMap from d.Records, replacing whatever it held.
// Call it after editing Records directly (appending, removing or renaming a
// record) so lookups and AddRecord's duplicate check see the change.
//
// When two records share an XRef the map holds the first, the rule the
// decoder, Clone, Subset and merge.Combine follow. Nil records and records
// with an empty XRef are not indexed. A nil d is a no-op.
func (d *Document) ReindexXRefs() {
	if d == nil {
		return
	}
	d.XRefMap = make(map[string]*Record, len(d.Records))
	for _, r := range d.Records {
		if r == nil || r.XRef == "" {
			continue
		}
		if _, exists := d.XRefMap[r.XRef]; !exists {
			d.XRefMap[r.XRef] = r
		}
	}
}
