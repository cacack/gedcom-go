package converter

import "github.com/cacack/gedcom-go/gedcom"

// deepCopyDocument creates a deep copy of a GEDCOM document.
// The returned document shares no pointers with the original.
func deepCopyDocument(doc *gedcom.Document) *gedcom.Document {
	if doc == nil {
		return nil
	}

	copied := &gedcom.Document{
		Header:  deepCopyHeader(doc.Header),
		Trailer: deepCopyTrailer(doc.Trailer),
		XRefMap: make(map[string]*gedcom.Record),
		Vendor:  doc.Vendor,
	}

	// Copy records
	copied.Records = make([]*gedcom.Record, len(doc.Records))
	for i, record := range doc.Records {
		copiedRecord := deepCopyRecord(record)
		copied.Records[i] = copiedRecord
		if copiedRecord.XRef != "" {
			copied.XRefMap[copiedRecord.XRef] = copiedRecord
		}
	}

	return copied
}

func deepCopyHeader(h *gedcom.Header) *gedcom.Header {
	if h == nil {
		return nil
	}

	copied := &gedcom.Header{
		Version:        h.Version,
		Encoding:       h.Encoding,
		SourceSystem:   h.SourceSystem,
		Date:           h.Date,
		Language:       h.Language,
		Copyright:      h.Copyright,
		Submitter:      h.Submitter,
		AncestryTreeID: h.AncestryTreeID,
	}

	// Copy tags
	copied.Tags = deepCopyTags(h.Tags)

	return copied
}

func deepCopyTrailer(t *gedcom.Trailer) *gedcom.Trailer {
	if t == nil {
		return nil
	}
	return &gedcom.Trailer{
		LineNumber: t.LineNumber,
	}
}

func deepCopyRecord(r *gedcom.Record) *gedcom.Record {
	if r == nil {
		return nil
	}

	copied := &gedcom.Record{
		XRef:       r.XRef,
		Type:       r.Type,
		Value:      r.Value,
		LineNumber: r.LineNumber,
		Entity:     deepCopyEntity(r.Entity),
	}

	// Copy tags
	copied.Tags = deepCopyTags(r.Tags)

	return copied
}

func deepCopyTags(tags []*gedcom.Tag) []*gedcom.Tag {
	if tags == nil {
		return nil
	}
	copied := make([]*gedcom.Tag, len(tags))
	for i, tag := range tags {
		copied[i] = deepCopyTag(tag)
	}
	return copied
}

func deepCopyTag(t *gedcom.Tag) *gedcom.Tag {
	if t == nil {
		return nil
	}

	return &gedcom.Tag{
		Level:      t.Level,
		Tag:        t.Tag,
		Value:      t.Value,
		XRef:       t.XRef,
		LineNumber: t.LineNumber,
	}
}

// deepCopyEntity creates a deep copy of an entity interface.
// The entity can be Individual, Family, Source, etc.
func deepCopyEntity(entity interface{}) interface{} {
	if entity == nil {
		return nil
	}

	switch e := entity.(type) {
	case *gedcom.Individual:
		return deepCopyIndividual(e)
	case *gedcom.Family:
		return deepCopyFamily(e)
	case *gedcom.Source:
		return deepCopySource(e)
	case *gedcom.Repository:
		return deepCopyRepository(e)
	case *gedcom.Note:
		return deepCopyNote(e)
	case *gedcom.MediaObject:
		return deepCopyMediaObject(e)
	case *gedcom.Submitter:
		return deepCopySubmitter(e)
	default:
		// Unknown entity type - return as-is (shallow copy)
		return entity
	}
}

func deepCopyIndividual(ind *gedcom.Individual) *gedcom.Individual {
	if ind == nil {
		return nil
	}

	copied := &gedcom.Individual{
		XRef:             ind.XRef,
		Sex:              ind.Sex,
		SpouseInFamilies: copyStringSlice(ind.SpouseInFamilies),
		Notes:            copyStringSlice(ind.Notes),
		RefNumber:        ind.RefNumber,
		UID:              ind.UID,
		FamilySearchID:   ind.FamilySearchID,
	}

	// Copy Names slice
	if ind.Names != nil {
		copied.Names = make([]*gedcom.PersonalName, len(ind.Names))
		for i, name := range ind.Names {
			copied.Names[i] = deepCopyPersonalName(name)
		}
	}

	// Copy ChildInFamilies
	if ind.ChildInFamilies != nil {
		copied.ChildInFamilies = make([]gedcom.FamilyLink, len(ind.ChildInFamilies))
		for i, link := range ind.ChildInFamilies {
			copied.ChildInFamilies[i] = gedcom.FamilyLink{
				FamilyXRef: link.FamilyXRef,
				Pedigree:   link.Pedigree,
			}
		}
	}

	// Copy Events
	if ind.Events != nil {
		copied.Events = make([]*gedcom.Event, len(ind.Events))
		for i, event := range ind.Events {
			copied.Events[i] = deepCopyEvent(event)
		}
	}

	// Copy Attributes
	if ind.Attributes != nil {
		copied.Attributes = make([]*gedcom.Attribute, len(ind.Attributes))
		for i, attr := range ind.Attributes {
			copied.Attributes[i] = deepCopyAttribute(attr)
		}
	}

	// Copy Associations
	if ind.Associations != nil {
		copied.Associations = make([]*gedcom.Association, len(ind.Associations))
		for i, assoc := range ind.Associations {
			copied.Associations[i] = deepCopyAssociation(assoc)
		}
	}

	// Copy SourceCitations
	if ind.SourceCitations != nil {
		copied.SourceCitations = make([]*gedcom.SourceCitation, len(ind.SourceCitations))
		for i, sc := range ind.SourceCitations {
			copied.SourceCitations[i] = deepCopySourceCitation(sc)
		}
	}

	// Copy Media
	if ind.Media != nil {
		copied.Media = make([]*gedcom.MediaLink, len(ind.Media))
		for i, media := range ind.Media {
			copied.Media[i] = deepCopyMediaLink(media)
		}
	}

	// Copy LDSOrdinances
	if ind.LDSOrdinances != nil {
		copied.LDSOrdinances = make([]*gedcom.LDSOrdinance, len(ind.LDSOrdinances))
		for i, ord := range ind.LDSOrdinances {
			copied.LDSOrdinances[i] = deepCopyLDSOrdinance(ord)
		}
	}

	// Copy ChangeDate
	copied.ChangeDate = deepCopyChangeDate(ind.ChangeDate)

	// Copy CreationDate
	copied.CreationDate = deepCopyChangeDate(ind.CreationDate)

	// Copy Tags
	copied.Tags = deepCopyTags(ind.Tags)

	return copied
}

func deepCopyFamily(fam *gedcom.Family) *gedcom.Family {
	if fam == nil {
		return nil
	}

	copied := &gedcom.Family{
		XRef:             fam.XRef,
		Husband:          fam.Husband,
		Wife:             fam.Wife,
		Children:         copyStringSlice(fam.Children),
		NumberOfChildren: fam.NumberOfChildren,
		Notes:            copyStringSlice(fam.Notes),
		RefNumber:        fam.RefNumber,
		UID:              fam.UID,
	}

	// Copy Events
	if fam.Events != nil {
		copied.Events = make([]*gedcom.Event, len(fam.Events))
		for i, event := range fam.Events {
			copied.Events[i] = deepCopyEvent(event)
		}
	}

	// Copy SourceCitations
	if fam.SourceCitations != nil {
		copied.SourceCitations = make([]*gedcom.SourceCitation, len(fam.SourceCitations))
		for i, sc := range fam.SourceCitations {
			copied.SourceCitations[i] = deepCopySourceCitation(sc)
		}
	}

	// Copy Media
	if fam.Media != nil {
		copied.Media = make([]*gedcom.MediaLink, len(fam.Media))
		for i, media := range fam.Media {
			copied.Media[i] = deepCopyMediaLink(media)
		}
	}

	// Copy LDSOrdinances
	if fam.LDSOrdinances != nil {
		copied.LDSOrdinances = make([]*gedcom.LDSOrdinance, len(fam.LDSOrdinances))
		for i, ord := range fam.LDSOrdinances {
			copied.LDSOrdinances[i] = deepCopyLDSOrdinance(ord)
		}
	}

	// Copy ChangeDate
	copied.ChangeDate = deepCopyChangeDate(fam.ChangeDate)

	// Copy CreationDate
	copied.CreationDate = deepCopyChangeDate(fam.CreationDate)

	// Copy Tags
	copied.Tags = deepCopyTags(fam.Tags)

	return copied
}

func deepCopySource(src *gedcom.Source) *gedcom.Source {
	if src == nil {
		return nil
	}

	copied := &gedcom.Source{
		XRef:          src.XRef,
		Title:         src.Title,
		Author:        src.Author,
		Publication:   src.Publication,
		Text:          src.Text,
		RepositoryRef: src.RepositoryRef,
		Notes:         copyStringSlice(src.Notes),
		RefNumber:     src.RefNumber,
		UID:           src.UID,
	}

	// Copy Repository (inline)
	if src.Repository != nil {
		copied.Repository = &gedcom.InlineRepository{
			Name: src.Repository.Name,
		}
	}

	// Copy Media
	if src.Media != nil {
		copied.Media = make([]*gedcom.MediaLink, len(src.Media))
		for i, media := range src.Media {
			copied.Media[i] = deepCopyMediaLink(media)
		}
	}

	// Copy ChangeDate
	copied.ChangeDate = deepCopyChangeDate(src.ChangeDate)

	// Copy CreationDate
	copied.CreationDate = deepCopyChangeDate(src.CreationDate)

	// Copy Tags
	copied.Tags = deepCopyTags(src.Tags)

	return copied
}

func deepCopyRepository(repo *gedcom.Repository) *gedcom.Repository {
	if repo == nil {
		return nil
	}

	copied := &gedcom.Repository{
		XRef:    repo.XRef,
		Name:    repo.Name,
		Address: deepCopyAddress(repo.Address),
		Notes:   copyStringSlice(repo.Notes),
	}

	// Copy Tags
	copied.Tags = deepCopyTags(repo.Tags)

	return copied
}

func deepCopyNote(note *gedcom.Note) *gedcom.Note {
	if note == nil {
		return nil
	}

	copied := &gedcom.Note{
		XRef:         note.XRef,
		Text:         note.Text,
		Continuation: copyStringSlice(note.Continuation),
	}

	// Copy Tags
	copied.Tags = deepCopyTags(note.Tags)

	return copied
}

func deepCopyMediaObject(media *gedcom.MediaObject) *gedcom.MediaObject {
	if media == nil {
		return nil
	}

	copied := &gedcom.MediaObject{
		XRef:        media.XRef,
		Notes:       copyStringSlice(media.Notes),
		RefNumbers:  copyStringSlice(media.RefNumbers),
		Restriction: media.Restriction,
		UIDs:        copyStringSlice(media.UIDs),
	}

	// Copy Files
	if media.Files != nil {
		copied.Files = make([]*gedcom.MediaFile, len(media.Files))
		for i, f := range media.Files {
			copied.Files[i] = deepCopyMediaFile(f)
		}
	}

	// Copy SourceCitations
	if media.SourceCitations != nil {
		copied.SourceCitations = make([]*gedcom.SourceCitation, len(media.SourceCitations))
		for i, sc := range media.SourceCitations {
			copied.SourceCitations[i] = deepCopySourceCitation(sc)
		}
	}

	// Copy ChangeDate
	copied.ChangeDate = deepCopyChangeDate(media.ChangeDate)

	// Copy CreationDate
	copied.CreationDate = deepCopyChangeDate(media.CreationDate)

	// Copy Tags
	copied.Tags = deepCopyTags(media.Tags)

	return copied
}

func deepCopySubmitter(subm *gedcom.Submitter) *gedcom.Submitter {
	if subm == nil {
		return nil
	}

	copied := &gedcom.Submitter{
		XRef:     subm.XRef,
		Name:     subm.Name,
		Address:  deepCopyAddress(subm.Address),
		Phone:    copyStringSlice(subm.Phone),
		Email:    copyStringSlice(subm.Email),
		Language: copyStringSlice(subm.Language),
		Notes:    copyStringSlice(subm.Notes),
	}

	// Copy Tags
	copied.Tags = deepCopyTags(subm.Tags)

	return copied
}

// Helper functions for deep copying nested types

func deepCopyPersonalName(n *gedcom.PersonalName) *gedcom.PersonalName {
	if n == nil {
		return nil
	}

	copied := &gedcom.PersonalName{
		Full:          n.Full,
		Given:         n.Given,
		Surname:       n.Surname,
		Prefix:        n.Prefix,
		Suffix:        n.Suffix,
		Nickname:      n.Nickname,
		SurnamePrefix: n.SurnamePrefix,
		Type:          n.Type,
	}

	// Copy Transliterations
	if n.Transliterations != nil {
		copied.Transliterations = make([]*gedcom.Transliteration, len(n.Transliterations))
		for i, t := range n.Transliterations {
			copied.Transliterations[i] = deepCopyTransliteration(t)
		}
	}

	return copied
}

func deepCopyTransliteration(t *gedcom.Transliteration) *gedcom.Transliteration {
	if t == nil {
		return nil
	}

	return &gedcom.Transliteration{
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

func deepCopyAssociation(a *gedcom.Association) *gedcom.Association {
	if a == nil {
		return nil
	}

	copied := &gedcom.Association{
		IndividualXRef: a.IndividualXRef,
		Role:           a.Role,
		Phrase:         a.Phrase,
		Notes:          copyStringSlice(a.Notes),
	}

	// Copy SourceCitations
	if a.SourceCitations != nil {
		copied.SourceCitations = make([]*gedcom.SourceCitation, len(a.SourceCitations))
		for i, sc := range a.SourceCitations {
			copied.SourceCitations[i] = deepCopySourceCitation(sc)
		}
	}

	return copied
}

func deepCopyEvent(e *gedcom.Event) *gedcom.Event {
	if e == nil {
		return nil
	}

	copied := &gedcom.Event{
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
		Notes:           copyStringSlice(e.Notes),
		Phone:           copyStringSlice(e.Phone),
		Email:           copyStringSlice(e.Email),
		Fax:             copyStringSlice(e.Fax),
		Website:         copyStringSlice(e.Website),
	}

	// Copy ParsedDate
	copied.ParsedDate = deepCopyDate(e.ParsedDate)

	// Copy PlaceDetail
	copied.PlaceDetail = deepCopyPlaceDetail(e.PlaceDetail)

	// Copy Address
	copied.Address = deepCopyAddress(e.Address)

	// Copy SourceCitations
	if e.SourceCitations != nil {
		copied.SourceCitations = make([]*gedcom.SourceCitation, len(e.SourceCitations))
		for i, sc := range e.SourceCitations {
			copied.SourceCitations[i] = deepCopySourceCitation(sc)
		}
	}

	// Copy Media
	if e.Media != nil {
		copied.Media = make([]*gedcom.MediaLink, len(e.Media))
		for i, media := range e.Media {
			copied.Media[i] = deepCopyMediaLink(media)
		}
	}

	// Copy Tags
	copied.Tags = deepCopyTags(e.Tags)

	return copied
}

func deepCopyAttribute(a *gedcom.Attribute) *gedcom.Attribute {
	if a == nil {
		return nil
	}

	copied := &gedcom.Attribute{
		Type:       a.Type,
		Value:      a.Value,
		Date:       a.Date,
		Place:      a.Place,
		ParsedDate: deepCopyDate(a.ParsedDate),
	}

	// Copy SourceCitations
	if a.SourceCitations != nil {
		copied.SourceCitations = make([]*gedcom.SourceCitation, len(a.SourceCitations))
		for i, sc := range a.SourceCitations {
			copied.SourceCitations[i] = deepCopySourceCitation(sc)
		}
	}

	return copied
}

func deepCopyDate(d *gedcom.Date) *gedcom.Date {
	if d == nil {
		return nil
	}

	copied := &gedcom.Date{
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

	// Copy EndDate (for ranges)
	copied.EndDate = deepCopyDate(d.EndDate)

	return copied
}

func deepCopyPlaceDetail(p *gedcom.PlaceDetail) *gedcom.PlaceDetail {
	if p == nil {
		return nil
	}

	copied := &gedcom.PlaceDetail{
		Name: p.Name,
		Form: p.Form,
	}

	// Copy Coordinates
	if p.Coordinates != nil {
		copied.Coordinates = &gedcom.Coordinates{
			Latitude:  p.Coordinates.Latitude,
			Longitude: p.Coordinates.Longitude,
		}
	}

	return copied
}

func deepCopyAddress(a *gedcom.Address) *gedcom.Address {
	if a == nil {
		return nil
	}

	return &gedcom.Address{
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

func deepCopySourceCitation(sc *gedcom.SourceCitation) *gedcom.SourceCitation {
	if sc == nil {
		return nil
	}

	copied := &gedcom.SourceCitation{
		SourceXRef: sc.SourceXRef,
		Page:       sc.Page,
		Quality:    sc.Quality,
	}

	// Copy Data
	if sc.Data != nil {
		copied.Data = &gedcom.SourceCitationData{
			Date: sc.Data.Date,
			Text: sc.Data.Text,
		}
	}

	// Copy AncestryAPID
	if sc.AncestryAPID != nil {
		copied.AncestryAPID = deepCopyAncestryAPID(sc.AncestryAPID)
	}

	return copied
}

func deepCopyAncestryAPID(apid *gedcom.AncestryAPID) *gedcom.AncestryAPID {
	if apid == nil {
		return nil
	}

	return &gedcom.AncestryAPID{
		Raw:      apid.Raw,
		Database: apid.Database,
		Record:   apid.Record,
	}
}

func deepCopyMediaLink(ml *gedcom.MediaLink) *gedcom.MediaLink {
	if ml == nil {
		return nil
	}

	copied := &gedcom.MediaLink{
		MediaXRef: ml.MediaXRef,
		Title:     ml.Title,
	}

	// Copy Crop
	if ml.Crop != nil {
		copied.Crop = &gedcom.CropRegion{
			Height: ml.Crop.Height,
			Left:   ml.Crop.Left,
			Top:    ml.Crop.Top,
			Width:  ml.Crop.Width,
		}
	}

	return copied
}

func deepCopyMediaFile(mf *gedcom.MediaFile) *gedcom.MediaFile {
	if mf == nil {
		return nil
	}

	copied := &gedcom.MediaFile{
		FileRef:   mf.FileRef,
		Form:      mf.Form,
		MediaType: mf.MediaType,
		Title:     mf.Title,
	}

	// Copy Translations
	if mf.Translations != nil {
		copied.Translations = make([]*gedcom.MediaTranslation, len(mf.Translations))
		for i, t := range mf.Translations {
			copied.Translations[i] = &gedcom.MediaTranslation{
				FileRef: t.FileRef,
				Form:    t.Form,
			}
		}
	}

	return copied
}

func deepCopyLDSOrdinance(ord *gedcom.LDSOrdinance) *gedcom.LDSOrdinance {
	if ord == nil {
		return nil
	}

	return &gedcom.LDSOrdinance{
		Type:       ord.Type,
		Date:       ord.Date,
		ParsedDate: deepCopyDate(ord.ParsedDate),
		Temple:     ord.Temple,
		Place:      ord.Place,
		Status:     ord.Status,
		FamilyXRef: ord.FamilyXRef,
	}
}

func deepCopyChangeDate(cd *gedcom.ChangeDate) *gedcom.ChangeDate {
	if cd == nil {
		return nil
	}

	return &gedcom.ChangeDate{
		Date: cd.Date,
		Time: cd.Time,
	}
}

func copyStringSlice(s []string) []string {
	if s == nil {
		return nil
	}
	copied := make([]string, len(s))
	copy(copied, s)
	return copied
}
