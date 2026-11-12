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
//   - PHRASE on associations, and the typed PHRASE beside a value
//     (TypePhrase, PedigreePhrase, RolePhrase, AgePhrase, HusbandAgePhrase
//     and WifeAgePhrase on a family's events and attributes, SortDatePhrase, AdoptedByPhrase, HusbandPhrase,
//     WifePhrase, MediaPhrase)
//   - TIME beneath an LDS status date (LDSOrdinance.StatusTime)
//   - DATE PHRASE on event, attribute and LDS ordinance dates
//     (ParsedDate.PhraseSubstructure; a 5.5.1 "(text)" date phrase is not one)
//   - MIME and LANG beneath a source record's TEXT (Source.TextMIME,
//     Source.TextLanguage)
//   - DATE PHRASE beneath a source record's DATA.EVEN
//     (SourceDataEvent.DatePhrase)
//   - CROP media crop regions
//   - TRAN media-file translations
//   - SNOTE references from media objects
//   - on source citations (SourceCitation) held by individuals, families,
//     names, associations, events, attributes and media objects: MIME and
//     LANG beneath DATA.TEXT (SourceText.MIME, SourceText.Language), PHRASE
//     beneath EVEN and its ROLE (SourceCitationEvent.Phrase, RolePhrase),
//     TIME and PHRASE beneath DATA.DATE (SourceCitationData.Time,
//     DatePhrase), and CROP on the citation's media links
//   - EXID beneath a PLAC (PlaceDetail.ExternalIDs)
//   - MIME, LANG and TRAN beneath an individual's inline NOTE
//     (Individual.InlineNoteDetails)
//   - repeated name pieces (PersonalName.AdditionalGiven and its siblings),
//     since 5.5.1 allows one of each
//
// Note that INT (interpreted) dates and non-Gregorian calendar dates are NOT
// treated as 7.0-only, because both are valid in GEDCOM 5.5.1. Nor are UIDs:
// the encoder writes them as the vendor tag _UID for 5.5/5.5.1 output, the
// spelling every 5.5.x desktop producer uses and the decoder reads (#715).
// Features stored only as raw tags (rather than typed fields) are not
// inspected.
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

//nolint:gocyclo // One case per record type with a 7.0-only typed signal
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
		return len(e.ExternalIDs) > 0 || e.CreationDate != nil ||
			mediaLinksRequireGEDCOM7(e.Media) || sourceTextRequiresGEDCOM7(e) ||
			sourceDataRequiresGEDCOM7(e.Data)
	case *MediaObject:
		return mediaObjectRequiresGEDCOM7(e)
	// UIDs are not checked, as for Individual: the encoder writes them as
	// _UID for 5.5/5.5.1 output (#715).
	case *Repository:
		return len(e.ExternalIDs) > 0 || e.CreationDate != nil
	// REFN is 7.0-only on SUBM; 5.5.1 has only RFN there.
	case *Submitter:
		return len(e.ExternalIDs) > 0 || e.CreationDate != nil || len(e.RefNumbers) > 0
	// Note has no CreationDate field (CREA is 7.0-only and the NOTE record is
	// 5.x-only), so EXID is its only 7.0-only typed signal.
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
	// UIDs are not checked here or one level down: the encoder writes them as
	// _UID for 5.5/5.5.1 output, so a record carrying one is representable
	// there (#715).
	if len(i.ExternalIDs) > 0 || i.CreationDate != nil ||
		mediaLinksRequireGEDCOM7(i.Media) || citationsRequireGEDCOM7(i.SourceCitations) {
		return true
	}
	if namesRequireGEDCOM7(i.Names) || pedigreePhraseRequiresGEDCOM7(i.ChildInFamilies) ||
		individualAssociationsRequireGEDCOM7(i.Associations) || ldsOrdinancesRequireGEDCOM7(i.LDSOrdinances) ||
		inlineNoteDetailsRequireGEDCOM7(i.InlineNoteDetails) {
		return true
	}
	for _, ev := range i.Events {
		if eventRequiresGEDCOM7(ev) {
			return true
		}
	}
	for _, at := range i.Attributes {
		if attributeRequiresGEDCOM7(at) {
			return true
		}
	}
	return false
}

// namesRequireGEDCOM7 reports whether any name carries a TRAN, a TYPE
// PHRASE, a repeated name piece or a 7.0-only citation substructure.
func namesRequireGEDCOM7(names []*PersonalName) bool {
	for _, n := range names {
		if n == nil {
			continue
		}
		if len(n.Transliterations) > 0 || n.TypePhrase != "" || namePiecesRepeat(n) ||
			citationsRequireGEDCOM7(n.SourceCitations) {
			return true
		}
	}
	return false
}

// namePiecesRepeat reports whether any name piece has a second value, which
// 7.0 allows ({0:M}) and 5.5.1 does not ({0:1}).
func namePiecesRepeat(n *PersonalName) bool {
	return len(n.AdditionalGiven) > 0 || len(n.AdditionalSurnames) > 0 ||
		len(n.AdditionalPrefixes) > 0 || len(n.AdditionalSuffixes) > 0 ||
		len(n.AdditionalNicknames) > 0 || len(n.AdditionalSurnamePrefixes) > 0
}

// placeRequiresGEDCOM7 reports whether a place carries an EXID, 7.0-only.
func placeRequiresGEDCOM7(p *PlaceDetail) bool {
	return p != nil && len(p.ExternalIDs) > 0
}

// inlineNoteDetailsRequireGEDCOM7 reports whether any inline note carries a
// MIME, LANG or TRAN, all 7.0-only.
func inlineNoteDetailsRequireGEDCOM7(details []*InlineNoteDetail) bool {
	for _, d := range details {
		if d != nil && (d.MIME != "" || d.Language != "" || len(d.Translations) > 0) {
			return true
		}
	}
	return false
}

// pedigreePhraseRequiresGEDCOM7 reports whether any child link carries a PEDI
// PHRASE, which 5.5.1 has no equivalent for.
func pedigreePhraseRequiresGEDCOM7(links []FamilyLink) bool {
	for _, link := range links {
		if link.PedigreePhrase != "" {
			return true
		}
	}
	return false
}

// ldsOrdinancesRequireGEDCOM7 reports whether any LDS ordinance date carries
// a PHRASE substructure, or its status date a TIME, which 5.5.1 has no
// equivalent for.
func ldsOrdinancesRequireGEDCOM7(ords []*LDSOrdinance) bool {
	for _, o := range ords {
		if o != nil && (o.ParsedDate.PhraseSubstructure(o.Date) != "" || o.StatusTime != "") {
			return true
		}
	}
	return false
}

func familyRequiresGEDCOM7(f *Family) bool {
	if f == nil {
		return false
	}
	if len(f.ExternalIDs) > 0 || f.CreationDate != nil ||
		f.HusbandPhrase != "" || f.WifePhrase != "" || mediaLinksRequireGEDCOM7(f.Media) ||
		ldsOrdinancesRequireGEDCOM7(f.LDSOrdinances) || citationsRequireGEDCOM7(f.SourceCitations) {
		return true
	}
	for _, ev := range f.Events {
		if familyEventRequiresGEDCOM7(ev) {
			return true
		}
	}
	for _, at := range f.Attributes {
		if familyAttributeRequiresGEDCOM7(at) {
			return true
		}
	}
	return false
}

// familyEventRequiresGEDCOM7 and familyAttributeRequiresGEDCOM7 add the
// spouse-age PHRASEs (HUSB.AGE.PHRASE, WIFE.AGE.PHRASE) to the shared checks.
// They apply to a family's events and attributes only: the encoder writes
// spouse ages nowhere else, so one set on an individual's event produces no
// 7.0-only output.
func familyEventRequiresGEDCOM7(ev *Event) bool {
	return eventRequiresGEDCOM7(ev) || (ev != nil && (ev.HusbandAgePhrase != "" || ev.WifeAgePhrase != ""))
}

func familyAttributeRequiresGEDCOM7(at *Attribute) bool {
	return attributeRequiresGEDCOM7(at) || (at != nil && (at.HusbandAgePhrase != "" || at.WifeAgePhrase != ""))
}

func eventRequiresGEDCOM7(ev *Event) bool {
	if ev == nil {
		return false
	}
	// NO (negative assertion) and SDATE (sort date) are 7.0-only. UIDs are
	// not: see individualRequiresGEDCOM7.
	if ev.IsNegative || ev.SortDate != "" ||
		ev.AgePhrase != "" || ev.SortDatePhrase != "" || ev.AdoptedByPhrase != "" || ev.ParsedDate.PhraseSubstructure(ev.Date) != "" ||
		placeRequiresGEDCOM7(ev.PlaceDetail) {
		return true
	}
	// An ASSO on an event is 7.0-only by position: 5.5 and 5.5.1 define
	// ASSOCIATION_STRUCTURE under INDI only, never inside EVENT_DETAIL.
	// Presence is the test here, unlike Individual.Associations where 5.5.1
	// has the structure and only PHRASE is new.
	if len(ev.Associations) > 0 {
		return true
	}
	return mediaLinksRequireGEDCOM7(ev.Media) || citationsRequireGEDCOM7(ev.SourceCitations)
}

// attributeRequiresGEDCOM7 reports whether an attribute carries a
// structure only GEDCOM 7.0 defines. SDATE on an attribute, a typed PHRASE,
// and an ASSO with a PHRASE, are 7.0-only; the encoder emits them
// unconditionally, so a document holding one cannot honestly be labelled
// 5.5.1. Spouse-age phrases are checked by familyAttributeRequiresGEDCOM7, and UIDs
// not at all: see individualRequiresGEDCOM7.
func attributeRequiresGEDCOM7(at *Attribute) bool {
	if at == nil {
		return false
	}
	if at.SortDate != "" || at.SortDatePhrase != "" || at.AgePhrase != "" ||
		at.ParsedDate.PhraseSubstructure(at.Date) != "" ||
		placeRequiresGEDCOM7(at.PlaceDetail) {
		return true
	}
	// As on an event, an ASSO under an attribute has no 5.5.1 position.
	if len(at.Associations) > 0 {
		return true
	}
	return mediaLinksRequireGEDCOM7(at.Media) || citationsRequireGEDCOM7(at.SourceCitations)
}

// individualAssociationsRequireGEDCOM7 reports whether any association
// carries a PHRASE, on the ASSO or its ROLE, which 5.5.1 has no equivalent
// for, or a citation needing 7.0.
//
// This is the right test only for Individual.Associations, where 5.5.1
// already has INDI.ASSO, so only its 7.0 substructures (PHRASE, and 7.0-only
// citation parts) raise the version. On an event or attribute the structure
// itself is 7.0-only, so those callers test presence.
func individualAssociationsRequireGEDCOM7(assocs []*Association) bool {
	for _, a := range assocs {
		if a != nil && (a.Phrase != "" || a.RolePhrase != "" ||
			citationsRequireGEDCOM7(a.SourceCitations)) {
			return true
		}
	}
	return false
}

func mediaObjectRequiresGEDCOM7(m *MediaObject) bool {
	if m == nil {
		return false
	}
	// RESN on an OBJE record is 7.0-only: the 5.5.1 MULTIMEDIA_RECORD has no
	// RESN substructure. RESN inside EVENT_DETAIL is a different position and
	// is 5.5.1-valid, which is why Event/Attribute do not check it.
	if len(m.ExternalIDs) > 0 || m.CreationDate != nil || len(m.SharedNoteXRefs) > 0 ||
		m.Restriction != "" || citationsRequireGEDCOM7(m.SourceCitations) {
		return true
	}
	for _, f := range m.Files {
		if f != nil && (len(f.Translations) > 0 || f.MediaPhrase != "") {
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

// sourceTextRequiresGEDCOM7 reports whether a source record's TEXT carries a
// MIME or LANG qualifier, both GEDCOM 7.0-only substructures.
func sourceTextRequiresGEDCOM7(s *Source) bool {
	return s.TextMIME != "" || s.TextLanguage != ""
}

// sourceDataRequiresGEDCOM7 reports whether any DATA.EVEN.DATE carries a
// PHRASE or any DATA.EVEN.PLAC an EXID, both GEDCOM 7.0-only substructures.
func sourceDataRequiresGEDCOM7(d *SourceData) bool {
	if d == nil {
		return false
	}
	for _, ev := range d.Events {
		if ev != nil && (ev.DatePhrase != "" || placeRequiresGEDCOM7(ev.PlaceDetail)) {
			return true
		}
	}
	return false
}

// citationsRequireGEDCOM7 reports whether any source citation carries a
// GEDCOM 7.0-only substructure: MIME or LANG beneath a DATA.TEXT, TIME or
// PHRASE beneath DATA.DATE, PHRASE beneath EVEN or its ROLE, or CROP on a
// media link.
func citationsRequireGEDCOM7(cits []*SourceCitation) bool {
	for _, c := range cits {
		if c == nil {
			continue
		}
		if c.Event != nil && (c.Event.Phrase != "" || c.Event.RolePhrase != "") {
			return true
		}
		if mediaLinksRequireGEDCOM7(c.Media) {
			return true
		}
		if c.Data == nil {
			continue
		}
		if c.Data.Time != "" || c.Data.DatePhrase != "" {
			return true
		}
		for _, t := range c.Data.Text {
			if t != nil && (t.MIME != "" || t.Language != "") {
				return true
			}
		}
	}
	return false
}
