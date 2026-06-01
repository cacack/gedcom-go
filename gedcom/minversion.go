package gedcom

// RequiresGEDCOM7 reports whether this document uses any feature that GEDCOM
// 5.5.1 cannot represent. The result is conservative: returning true means a
// 5.5.1 export would lose information; returning false means 5.5.1 can carry
// the content.
//
// It detects the 7.0-only features the library models as typed fields:
//
//   - SCHMA schema extension declarations (Document.Schema)
//   - SNOTE shared-note records
//   - EXID external identifiers on any record
//   - CREA creation timestamps on any record
//   - NO negative event assertions
//   - SDATE event sort dates
//   - TRAN name transliterations
//   - PHRASE on associations
//   - CROP media crop regions
//   - TRAN media-file translations
//   - SNOTE references from media objects
//
// Note that INT (interpreted) dates and non-Gregorian calendar dates are NOT
// treated as 7.0-only, because both are valid in GEDCOM 5.5.1. Features stored
// only as raw tags (rather than typed fields) are not inspected.
func (d *Document) RequiresGEDCOM7() bool {
	if d == nil {
		return false
	}

	// SCHMA (schema extension declaration) is a 7.0 header substructure.
	if d.Schema != nil && len(d.Schema.TagMappings) > 0 {
		return true
	}

	for _, rec := range d.Records {
		if recordRequiresGEDCOM7(rec) {
			return true
		}
	}
	return false
}

func recordRequiresGEDCOM7(rec *Record) bool {
	if rec == nil {
		return false
	}
	// SNOTE records exist only in 7.0.
	if rec.Type == RecordTypeSharedNote {
		return true
	}
	switch e := rec.Entity.(type) {
	case *Individual:
		return individualRequiresGEDCOM7(e)
	case *Family:
		return familyRequiresGEDCOM7(e)
	case *Source:
		return len(e.ExternalIDs) > 0 || e.CreationDate != nil || mediaLinksRequireGEDCOM7(e.Media)
	case *MediaObject:
		return mediaObjectRequiresGEDCOM7(e)
	// Repository, Submitter, and Note have no CreationDate field (only
	// Individual, Family, Source, and MediaObject do), so EXID is their only
	// 7.0-only typed signal.
	case *Repository:
		return len(e.ExternalIDs) > 0
	case *Submitter:
		return len(e.ExternalIDs) > 0
	case *Note:
		return len(e.ExternalIDs) > 0
	}
	return false
}

// MinimumVersion returns the lowest GEDCOM version that can losslessly
// represent the document: Version70 when the document uses any 7.0-only
// feature (see RequiresGEDCOM7), and Version551 otherwise. It pairs with
// encoder.EncodeOptions.TargetVersion for "emit at the lowest compatible
// version" export flows.
//
// It never returns Version55: for writing purposes 5.5.1 is a superset of 5.5,
// so 5.5.1 is the natural floor for lossless export and no consumer benefits
// from emitting 5.5 over 5.5.1.
func (d *Document) MinimumVersion() Version {
	if d.RequiresGEDCOM7() {
		return Version70
	}
	return Version551
}

func individualRequiresGEDCOM7(i *Individual) bool {
	if i == nil {
		return false
	}
	if len(i.ExternalIDs) > 0 || i.CreationDate != nil || mediaLinksRequireGEDCOM7(i.Media) {
		return true
	}
	for _, n := range i.Names {
		if n != nil && len(n.Transliterations) > 0 {
			return true
		}
	}
	for _, a := range i.Associations {
		if a != nil && a.Phrase != "" {
			return true
		}
	}
	for _, ev := range i.Events {
		if eventRequiresGEDCOM7(ev) {
			return true
		}
	}
	// Individual.Attributes is intentionally not inspected: the Attribute struct
	// carries no 7.0-only typed fields. Revisit if that ever changes.
	return false
}

func familyRequiresGEDCOM7(f *Family) bool {
	if f == nil {
		return false
	}
	if len(f.ExternalIDs) > 0 || f.CreationDate != nil || mediaLinksRequireGEDCOM7(f.Media) {
		return true
	}
	for _, ev := range f.Events {
		if eventRequiresGEDCOM7(ev) {
			return true
		}
	}
	return false
}

func eventRequiresGEDCOM7(ev *Event) bool {
	if ev == nil {
		return false
	}
	// NO (negative assertion) and SDATE (sort date) are 7.0-only.
	if ev.IsNegative || ev.SortDate != "" {
		return true
	}
	return mediaLinksRequireGEDCOM7(ev.Media)
}

func mediaObjectRequiresGEDCOM7(m *MediaObject) bool {
	if m == nil {
		return false
	}
	if len(m.ExternalIDs) > 0 || m.CreationDate != nil || len(m.SharedNoteXRefs) > 0 {
		return true
	}
	for _, f := range m.Files {
		if f != nil && len(f.Translations) > 0 {
			return true
		}
	}
	return false
}

// mediaLinksRequireGEDCOM7 reports whether any media link carries a CROP region,
// which is a GEDCOM 7.0-only feature.
func mediaLinksRequireGEDCOM7(links []*MediaLink) bool {
	for _, l := range links {
		if l != nil && l.Crop != nil {
			return true
		}
	}
	return false
}
