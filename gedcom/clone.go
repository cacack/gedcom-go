package gedcom

// Clone returns a deep copy of the document. The returned document
// shares no pointers with the original; mutating one will not affect
// the other. Returns nil if d is nil.
func (d *Document) Clone() *Document {
	if d == nil {
		return nil
	}

	copied := &Document{
		Header:  d.Header.Clone(),
		Trailer: d.Trailer.Clone(),
		XRefMap: make(map[string]*Record),
		Vendor:  d.Vendor,
		Schema:  cloneSchemaDefinition(d.Schema),
	}

	copied.Records = make([]*Record, len(d.Records))
	for i, record := range d.Records {
		copiedRecord := record.Clone()
		copied.Records[i] = copiedRecord
		if copiedRecord != nil && copiedRecord.XRef != "" {
			copied.XRefMap[copiedRecord.XRef] = copiedRecord
		}
	}

	return copied
}

// Clone returns a deep copy of the header. Returns nil if h is nil.
//
// Schema is not copied here: SchemaDefinition lives on Document, not
// Header. Callers cloning a whole document via (*Document).Clone get
// Schema deep-copied at the document level.
func (h *Header) Clone() *Header {
	if h == nil {
		return nil
	}

	return &Header{
		Version:        h.Version,
		Encoding:       h.Encoding,
		SourceSystem:   h.SourceSystem,
		Date:           h.Date,
		Language:       h.Language,
		Copyright:      h.Copyright,
		Submitter:      h.Submitter,
		AncestryTreeID: h.AncestryTreeID,
		Tags:           CloneTags(h.Tags),
	}
}

// Clone returns a deep copy of the trailer. Returns nil if t is nil.
func (t *Trailer) Clone() *Trailer {
	if t == nil {
		return nil
	}
	return &Trailer{LineNumber: t.LineNumber}
}

// Clone returns a deep copy of the record. The Entity field is
// deep-copied based on its concrete type. Returns nil if r is nil.
func (r *Record) Clone() *Record {
	if r == nil {
		return nil
	}

	return &Record{
		XRef:       r.XRef,
		Type:       r.Type,
		Value:      r.Value,
		LineNumber: r.LineNumber,
		Entity:     cloneEntity(r.Entity),
		Tags:       CloneTags(r.Tags),
	}
}

// Clone returns a deep copy of the tag. Returns nil if t is nil.
func (t *Tag) Clone() *Tag {
	if t == nil {
		return nil
	}
	return &Tag{
		Level:      t.Level,
		Tag:        t.Tag,
		Value:      t.Value,
		XRef:       t.XRef,
		LineNumber: t.LineNumber,
	}
}

// Clone returns a deep copy of the individual. Returns nil if i is nil.
//
//nolint:gocyclo // Individual has many optional fields requiring nil checks for safe deep copy
func (i *Individual) Clone() *Individual {
	if i == nil {
		return nil
	}

	copied := &Individual{
		XRef:             i.XRef,
		Sex:              i.Sex,
		SpouseInFamilies: cloneStringSlice(i.SpouseInFamilies),
		Notes:            cloneStringSlice(i.Notes),
		RefNumber:        i.RefNumber,
		UID:              i.UID,
		FamilySearchID:   i.FamilySearchID,
	}

	if i.Names != nil {
		copied.Names = make([]*PersonalName, len(i.Names))
		for k, name := range i.Names {
			copied.Names[k] = clonePersonalName(name)
		}
	}

	if i.ChildInFamilies != nil {
		copied.ChildInFamilies = make([]FamilyLink, len(i.ChildInFamilies))
		for k, link := range i.ChildInFamilies {
			copied.ChildInFamilies[k] = FamilyLink{
				FamilyXRef: link.FamilyXRef,
				Pedigree:   link.Pedigree,
			}
		}
	}

	if i.Events != nil {
		copied.Events = make([]*Event, len(i.Events))
		for k, event := range i.Events {
			copied.Events[k] = cloneEvent(event)
		}
	}

	if i.Attributes != nil {
		copied.Attributes = make([]*Attribute, len(i.Attributes))
		for k, attr := range i.Attributes {
			copied.Attributes[k] = cloneAttribute(attr)
		}
	}

	if i.Associations != nil {
		copied.Associations = make([]*Association, len(i.Associations))
		for k, assoc := range i.Associations {
			copied.Associations[k] = cloneAssociation(assoc)
		}
	}

	if i.SourceCitations != nil {
		copied.SourceCitations = make([]*SourceCitation, len(i.SourceCitations))
		for k, sc := range i.SourceCitations {
			copied.SourceCitations[k] = cloneSourceCitation(sc)
		}
	}

	if i.Media != nil {
		copied.Media = make([]*MediaLink, len(i.Media))
		for k, media := range i.Media {
			copied.Media[k] = cloneMediaLink(media)
		}
	}

	if i.LDSOrdinances != nil {
		copied.LDSOrdinances = make([]*LDSOrdinance, len(i.LDSOrdinances))
		for k, ord := range i.LDSOrdinances {
			copied.LDSOrdinances[k] = cloneLDSOrdinance(ord)
		}
	}

	if i.ExternalIDs != nil {
		copied.ExternalIDs = make([]*ExternalID, len(i.ExternalIDs))
		for k, ext := range i.ExternalIDs {
			copied.ExternalIDs[k] = cloneExternalID(ext)
		}
	}

	copied.ChangeDate = cloneChangeDate(i.ChangeDate)
	copied.CreationDate = cloneChangeDate(i.CreationDate)
	copied.Tags = CloneTags(i.Tags)

	return copied
}

// Clone returns a deep copy of the family. Returns nil if f is nil.
func (f *Family) Clone() *Family {
	if f == nil {
		return nil
	}

	copied := &Family{
		XRef:             f.XRef,
		Husband:          f.Husband,
		Wife:             f.Wife,
		Children:         cloneStringSlice(f.Children),
		NumberOfChildren: f.NumberOfChildren,
		Notes:            cloneStringSlice(f.Notes),
		RefNumber:        f.RefNumber,
		UID:              f.UID,
	}

	if f.Events != nil {
		copied.Events = make([]*Event, len(f.Events))
		for k, event := range f.Events {
			copied.Events[k] = cloneEvent(event)
		}
	}

	if f.SourceCitations != nil {
		copied.SourceCitations = make([]*SourceCitation, len(f.SourceCitations))
		for k, sc := range f.SourceCitations {
			copied.SourceCitations[k] = cloneSourceCitation(sc)
		}
	}

	if f.Media != nil {
		copied.Media = make([]*MediaLink, len(f.Media))
		for k, media := range f.Media {
			copied.Media[k] = cloneMediaLink(media)
		}
	}

	if f.LDSOrdinances != nil {
		copied.LDSOrdinances = make([]*LDSOrdinance, len(f.LDSOrdinances))
		for k, ord := range f.LDSOrdinances {
			copied.LDSOrdinances[k] = cloneLDSOrdinance(ord)
		}
	}

	copied.ChangeDate = cloneChangeDate(f.ChangeDate)
	copied.CreationDate = cloneChangeDate(f.CreationDate)
	copied.Tags = CloneTags(f.Tags)

	return copied
}

// Clone returns a deep copy of the source. Returns nil if s is nil.
func (s *Source) Clone() *Source {
	if s == nil {
		return nil
	}

	copied := &Source{
		XRef:          s.XRef,
		Title:         s.Title,
		Author:        s.Author,
		Publication:   s.Publication,
		Text:          s.Text,
		RepositoryRef: s.RepositoryRef,
		Notes:         cloneStringSlice(s.Notes),
		RefNumber:     s.RefNumber,
		UID:           s.UID,
	}

	if s.Repository != nil {
		copied.Repository = &InlineRepository{Name: s.Repository.Name}
	}

	if s.Media != nil {
		copied.Media = make([]*MediaLink, len(s.Media))
		for k, media := range s.Media {
			copied.Media[k] = cloneMediaLink(media)
		}
	}

	copied.ChangeDate = cloneChangeDate(s.ChangeDate)
	copied.CreationDate = cloneChangeDate(s.CreationDate)
	copied.Tags = CloneTags(s.Tags)

	return copied
}

// Clone returns a deep copy of the repository. Returns nil if r is nil.
func (r *Repository) Clone() *Repository {
	if r == nil {
		return nil
	}

	return &Repository{
		XRef:    r.XRef,
		Name:    r.Name,
		Address: cloneAddress(r.Address),
		Notes:   cloneStringSlice(r.Notes),
		Tags:    CloneTags(r.Tags),
	}
}

// Clone returns a deep copy of the note. Returns nil if n is nil.
func (n *Note) Clone() *Note {
	if n == nil {
		return nil
	}

	return &Note{
		XRef:         n.XRef,
		Text:         n.Text,
		Continuation: cloneStringSlice(n.Continuation),
		Tags:         CloneTags(n.Tags),
	}
}

// Clone returns a deep copy of the media object. Returns nil if m is nil.
func (m *MediaObject) Clone() *MediaObject {
	if m == nil {
		return nil
	}

	copied := &MediaObject{
		XRef:        m.XRef,
		Notes:       cloneStringSlice(m.Notes),
		RefNumbers:  cloneStringSlice(m.RefNumbers),
		Restriction: m.Restriction,
		UIDs:        cloneStringSlice(m.UIDs),
	}

	if m.Files != nil {
		copied.Files = make([]*MediaFile, len(m.Files))
		for k, f := range m.Files {
			copied.Files[k] = cloneMediaFile(f)
		}
	}

	if m.SourceCitations != nil {
		copied.SourceCitations = make([]*SourceCitation, len(m.SourceCitations))
		for k, sc := range m.SourceCitations {
			copied.SourceCitations[k] = cloneSourceCitation(sc)
		}
	}

	copied.ChangeDate = cloneChangeDate(m.ChangeDate)
	copied.CreationDate = cloneChangeDate(m.CreationDate)
	copied.Tags = CloneTags(m.Tags)

	return copied
}

// Clone returns a deep copy of the submitter. Returns nil if s is nil.
func (s *Submitter) Clone() *Submitter {
	if s == nil {
		return nil
	}

	return &Submitter{
		XRef:     s.XRef,
		Name:     s.Name,
		Address:  cloneAddress(s.Address),
		Phone:    cloneStringSlice(s.Phone),
		Email:    cloneStringSlice(s.Email),
		Language: cloneStringSlice(s.Language),
		Notes:    cloneStringSlice(s.Notes),
		Tags:     CloneTags(s.Tags),
	}
}

// Clone returns a deep copy of the shared note. Returns nil if s is nil.
func (s *SharedNote) Clone() *SharedNote {
	if s == nil {
		return nil
	}

	copied := &SharedNote{
		XRef:     s.XRef,
		Text:     s.Text,
		MIME:     s.MIME,
		Language: s.Language,
	}

	if s.Translations != nil {
		copied.Translations = make([]*SharedNoteTranslation, len(s.Translations))
		for k, t := range s.Translations {
			copied.Translations[k] = &SharedNoteTranslation{
				Value:    t.Value,
				MIME:     t.MIME,
				Language: t.Language,
			}
		}
	}

	if s.SourceCitations != nil {
		copied.SourceCitations = make([]*SourceCitation, len(s.SourceCitations))
		for k, sc := range s.SourceCitations {
			copied.SourceCitations[k] = cloneSourceCitation(sc)
		}
	}

	if s.ExternalIDs != nil {
		copied.ExternalIDs = make([]*ExternalID, len(s.ExternalIDs))
		for k, ext := range s.ExternalIDs {
			copied.ExternalIDs[k] = cloneExternalID(ext)
		}
	}

	copied.ChangeDate = cloneChangeDate(s.ChangeDate)
	copied.Tags = CloneTags(s.Tags)

	return copied
}

// cloneEntity returns a deep copy of a record's Entity field. Unknown
// entity types are returned as-is (shallow copy).
func cloneEntity(entity interface{}) interface{} {
	if entity == nil {
		return nil
	}

	switch e := entity.(type) {
	case *Individual:
		return e.Clone()
	case *Family:
		return e.Clone()
	case *Source:
		return e.Clone()
	case *Repository:
		return e.Clone()
	case *Note:
		return e.Clone()
	case *MediaObject:
		return e.Clone()
	case *Submitter:
		return e.Clone()
	case *SharedNote:
		return e.Clone()
	default:
		return entity
	}
}

// CloneTags returns a deep copy of a slice of tags. Returns nil if tags is nil.
// Each element is duplicated via Tag.Clone, so the result shares no
// pointers with the input.
func CloneTags(tags []*Tag) []*Tag {
	if tags == nil {
		return nil
	}
	copied := make([]*Tag, len(tags))
	for i, tag := range tags {
		copied[i] = tag.Clone()
	}
	return copied
}

func cloneSchemaDefinition(s *SchemaDefinition) *SchemaDefinition {
	if s == nil {
		return nil
	}
	copied := &SchemaDefinition{}
	if s.TagMappings != nil {
		copied.TagMappings = make(map[string]string, len(s.TagMappings))
		for k, v := range s.TagMappings {
			copied.TagMappings[k] = v
		}
	}
	return copied
}

func clonePersonalName(n *PersonalName) *PersonalName {
	if n == nil {
		return nil
	}

	copied := &PersonalName{
		Full:          n.Full,
		Given:         n.Given,
		Surname:       n.Surname,
		Prefix:        n.Prefix,
		Suffix:        n.Suffix,
		Nickname:      n.Nickname,
		SurnamePrefix: n.SurnamePrefix,
		Type:          n.Type,
	}

	if n.Transliterations != nil {
		copied.Transliterations = make([]*Transliteration, len(n.Transliterations))
		for i, t := range n.Transliterations {
			copied.Transliterations[i] = cloneTransliteration(t)
		}
	}

	return copied
}

func cloneTransliteration(t *Transliteration) *Transliteration {
	if t == nil {
		return nil
	}
	return &Transliteration{
		Value:         t.Value,
		Language:      t.Language,
		Given:         t.Given,
		Surname:       t.Surname,
		Prefix:        t.Prefix,
		Suffix:        t.Suffix,
		Nickname:      t.Nickname,
		SurnamePrefix: t.SurnamePrefix,
	}
}

func cloneAssociation(a *Association) *Association {
	if a == nil {
		return nil
	}

	copied := &Association{
		IndividualXRef: a.IndividualXRef,
		Role:           a.Role,
		Phrase:         a.Phrase,
		Notes:          cloneStringSlice(a.Notes),
	}

	if a.SourceCitations != nil {
		copied.SourceCitations = make([]*SourceCitation, len(a.SourceCitations))
		for i, sc := range a.SourceCitations {
			copied.SourceCitations[i] = cloneSourceCitation(sc)
		}
	}

	return copied
}

func cloneEvent(e *Event) *Event {
	if e == nil {
		return nil
	}

	copied := &Event{
		Type:            e.Type,
		Date:            e.Date,
		Place:           e.Place,
		Description:     e.Description,
		EventTypeDetail: e.EventTypeDetail,
		Cause:           e.Cause,
		Age:             e.Age,
		Agency:          e.Agency,
		Restriction:     e.Restriction,
		UID:             e.UID,
		SortDate:        e.SortDate,
		Notes:           cloneStringSlice(e.Notes),
		Phone:           cloneStringSlice(e.Phone),
		Email:           cloneStringSlice(e.Email),
		Fax:             cloneStringSlice(e.Fax),
		Website:         cloneStringSlice(e.Website),
	}

	copied.ParsedDate = cloneDate(e.ParsedDate)
	copied.PlaceDetail = clonePlaceDetail(e.PlaceDetail)
	copied.Address = cloneAddress(e.Address)

	if e.SourceCitations != nil {
		copied.SourceCitations = make([]*SourceCitation, len(e.SourceCitations))
		for i, sc := range e.SourceCitations {
			copied.SourceCitations[i] = cloneSourceCitation(sc)
		}
	}

	if e.Media != nil {
		copied.Media = make([]*MediaLink, len(e.Media))
		for i, media := range e.Media {
			copied.Media[i] = cloneMediaLink(media)
		}
	}

	copied.Tags = CloneTags(e.Tags)
	return copied
}

func cloneAttribute(a *Attribute) *Attribute {
	if a == nil {
		return nil
	}

	copied := &Attribute{
		Type:       a.Type,
		Value:      a.Value,
		Date:       a.Date,
		Place:      a.Place,
		ParsedDate: cloneDate(a.ParsedDate),
	}

	if a.SourceCitations != nil {
		copied.SourceCitations = make([]*SourceCitation, len(a.SourceCitations))
		for i, sc := range a.SourceCitations {
			copied.SourceCitations[i] = cloneSourceCitation(sc)
		}
	}

	return copied
}

func cloneDate(d *Date) *Date {
	if d == nil {
		return nil
	}

	copied := &Date{
		Original: d.Original,
		Day:      d.Day,
		Month:    d.Month,
		Year:     d.Year,
		Modifier: d.Modifier,
		Calendar: d.Calendar,
		IsBC:     d.IsBC,
		DualYear: d.DualYear,
		Phrase:   d.Phrase,
		IsPhrase: d.IsPhrase,
	}
	copied.EndDate = cloneDate(d.EndDate)
	return copied
}

func clonePlaceDetail(p *PlaceDetail) *PlaceDetail {
	if p == nil {
		return nil
	}

	copied := &PlaceDetail{
		Name: p.Name,
		Form: p.Form,
	}

	if p.Coordinates != nil {
		copied.Coordinates = &Coordinates{
			Latitude:  p.Coordinates.Latitude,
			Longitude: p.Coordinates.Longitude,
		}
	}

	return copied
}

func cloneAddress(a *Address) *Address {
	if a == nil {
		return nil
	}
	return &Address{
		Line1:      a.Line1,
		Line2:      a.Line2,
		Line3:      a.Line3,
		City:       a.City,
		State:      a.State,
		PostalCode: a.PostalCode,
		Country:    a.Country,
		Phone:      a.Phone,
		Email:      a.Email,
		Website:    a.Website,
	}
}

func cloneSourceCitation(sc *SourceCitation) *SourceCitation {
	if sc == nil {
		return nil
	}

	copied := &SourceCitation{
		SourceXRef: sc.SourceXRef,
		Page:       sc.Page,
		Quality:    sc.Quality,
	}

	if sc.Data != nil {
		copied.Data = &SourceCitationData{
			Date: sc.Data.Date,
			Text: sc.Data.Text,
		}
	}

	if sc.AncestryAPID != nil {
		copied.AncestryAPID = cloneAncestryAPID(sc.AncestryAPID)
	}

	return copied
}

func cloneAncestryAPID(apid *AncestryAPID) *AncestryAPID {
	if apid == nil {
		return nil
	}
	return &AncestryAPID{
		Raw:      apid.Raw,
		Database: apid.Database,
		Record:   apid.Record,
	}
}

func cloneMediaLink(ml *MediaLink) *MediaLink {
	if ml == nil {
		return nil
	}

	copied := &MediaLink{
		MediaXRef: ml.MediaXRef,
		Title:     ml.Title,
	}

	if ml.Crop != nil {
		copied.Crop = &CropRegion{
			Height: ml.Crop.Height,
			Left:   ml.Crop.Left,
			Top:    ml.Crop.Top,
			Width:  ml.Crop.Width,
		}
	}

	return copied
}

func cloneMediaFile(mf *MediaFile) *MediaFile {
	if mf == nil {
		return nil
	}

	copied := &MediaFile{
		FileRef:   mf.FileRef,
		Form:      mf.Form,
		MediaType: mf.MediaType,
		Title:     mf.Title,
	}

	if mf.Translations != nil {
		copied.Translations = make([]*MediaTranslation, len(mf.Translations))
		for i, t := range mf.Translations {
			copied.Translations[i] = &MediaTranslation{
				FileRef: t.FileRef,
				Form:    t.Form,
			}
		}
	}

	return copied
}

func cloneLDSOrdinance(ord *LDSOrdinance) *LDSOrdinance {
	if ord == nil {
		return nil
	}
	return &LDSOrdinance{
		Type:       ord.Type,
		Date:       ord.Date,
		ParsedDate: cloneDate(ord.ParsedDate),
		Temple:     ord.Temple,
		Place:      ord.Place,
		Status:     ord.Status,
		FamilyXRef: ord.FamilyXRef,
	}
}

func cloneChangeDate(cd *ChangeDate) *ChangeDate {
	if cd == nil {
		return nil
	}
	return &ChangeDate{
		Date: cd.Date,
		Time: cd.Time,
	}
}

func cloneExternalID(ext *ExternalID) *ExternalID {
	if ext == nil {
		return nil
	}
	return &ExternalID{
		Value: ext.Value,
		Type:  ext.Type,
	}
}

func cloneStringSlice(s []string) []string {
	if s == nil {
		return nil
	}
	copied := make([]string, len(s))
	copy(copied, s)
	return copied
}
