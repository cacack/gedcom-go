package decoder

import (
	"strconv"
	"strings"

	"github.com/cacack/gedcom-go/gedcom"
)

// populateEntities converts raw tags in each record into proper entities.
func populateEntities(doc *gedcom.Document) {
	for _, record := range doc.Records {
		switch record.Type {
		case gedcom.RecordTypeIndividual:
			record.Entity = parseIndividual(record)
		case gedcom.RecordTypeFamily:
			record.Entity = parseFamily(record)
		case gedcom.RecordTypeSource:
			record.Entity = parseSource(record)
		case gedcom.RecordTypeSubmitter:
			record.Entity = parseSubmitter(record)
		case gedcom.RecordTypeRepository:
			record.Entity = parseRepository(record)
		case gedcom.RecordTypeNote:
			record.Entity = parseNote(record)
		case gedcom.RecordTypeMedia:
			record.Entity = parseMediaObject(record)
		}
	}
}

// parseIndividual converts record tags to an Individual entity.
//
//nolint:gocyclo // GEDCOM parsing inherently requires handling many tag types
func parseIndividual(record *gedcom.Record) *gedcom.Individual {
	indi := &gedcom.Individual{
		XRef: record.XRef,
		Tags: record.Tags,
	}

	for i := 0; i < len(record.Tags); i++ {
		tag := record.Tags[i]
		if tag.Level != 1 {
			continue
		}

		switch tag.Tag {
		case "NAME":
			name := parsePersonalName(record.Tags, i)
			indi.Names = append(indi.Names, name)

		case "SEX":
			indi.Sex = tag.Value

		case "BIRT", "DEAT", "BAPM", "BURI", "CENS", "CHR", "ADOP", "RESI", "IMMI", "EMIG",
			"BARM", "BASM", "BLES", "CHRA", "CONF", "FCOM",
			"GRAD", "RETI", "NATU", "ORDN", "PROB", "WILL", "CREM":
			event := parseEvent(record.Tags, i, tag.Tag)
			indi.Events = append(indi.Events, event)

		case "BAPL", "CONL", "ENDL", "SLGC":
			ord := parseLDSOrdinance(record.Tags, i, ldsOrdinanceType(tag.Tag))
			indi.LDSOrdinances = append(indi.LDSOrdinances, ord)

		case "OCCU", "CAST", "DSCR", "EDUC", "IDNO", "NATI", "SSN", "TITL", "RELI", "NCHI", "NMR", "PROP":
			attr := parseAttribute(record.Tags, i, tag.Tag)
			indi.Attributes = append(indi.Attributes, attr)

		case "FAMC":
			famLink := parseFamilyLink(record.Tags, i)
			indi.ChildInFamilies = append(indi.ChildInFamilies, famLink)

		case "FAMS":
			indi.SpouseInFamilies = append(indi.SpouseInFamilies, tag.Value)

		case "ASSO":
			assoc := parseAssociation(record.Tags, i)
			indi.Associations = append(indi.Associations, assoc)

		case "SOUR":
			cite := parseSourceCitation(record.Tags, i, tag.Level)
			indi.SourceCitations = append(indi.SourceCitations, cite)

		case "NOTE":
			indi.Notes = append(indi.Notes, tag.Value)

		case "OBJE":
			link := parseMediaLink(record.Tags, i, tag.Level)
			indi.Media = append(indi.Media, link)

		case "CHAN":
			indi.ChangeDate = parseChangeDate(record.Tags, i)

		case "CREA":
			indi.CreationDate = parseChangeDate(record.Tags, i)

		case "REFN":
			indi.RefNumber = tag.Value

		case "UID":
			indi.UID = tag.Value

		case "_FSFTID":
			indi.FamilySearchID = tag.Value
		}
	}

	return indi
}

// parsePersonalName extracts name components from tags starting at nameIdx.
func parsePersonalName(tags []*gedcom.Tag, nameIdx int) *gedcom.PersonalName {
	name := &gedcom.PersonalName{
		Full: tags[nameIdx].Value,
	}

	// Parse the full name to extract given and surname
	// GEDCOM format: "Given /Surname/"
	full := tags[nameIdx].Value
	if slashIdx := strings.Index(full, "/"); slashIdx >= 0 {
		name.Given = strings.TrimSpace(full[:slashIdx])
		surname := full[slashIdx+1:]
		if endSlash := strings.Index(surname, "/"); endSlash >= 0 {
			name.Surname = surname[:endSlash]
		} else {
			name.Surname = strings.TrimSpace(surname)
		}
	} else {
		name.Given = strings.TrimSpace(full)
	}

	// Look for subordinate tags (level 2)
	for i := nameIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= 1 {
			break
		}
		if tag.Level == 2 {
			switch tag.Tag {
			case "GIVN":
				name.Given = tag.Value
			case "SURN":
				name.Surname = tag.Value
			case "NPFX":
				name.Prefix = tag.Value
			case "NSFX":
				name.Suffix = tag.Value
			case "NICK":
				name.Nickname = tag.Value
			case "SPFX":
				name.SurnamePrefix = tag.Value
			case "TYPE":
				name.Type = tag.Value
			}
		}
	}

	return name
}

// parseFamilyLink extracts a family link from tags starting at famcIdx.
func parseFamilyLink(tags []*gedcom.Tag, famcIdx int) gedcom.FamilyLink {
	famLink := gedcom.FamilyLink{
		FamilyXRef: tags[famcIdx].Value,
	}

	// Look for subordinate tags (level 2)
	for i := famcIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= 1 {
			break
		}
		if tag.Level == 2 && tag.Tag == "PEDI" {
			famLink.Pedigree = tag.Value
		}
	}

	return famLink
}

// parseAssociation extracts an association from tags starting at assoIdx.
func parseAssociation(tags []*gedcom.Tag, assoIdx int) *gedcom.Association {
	assoc := &gedcom.Association{
		IndividualXRef: tags[assoIdx].Value,
	}

	// Look for subordinate tags (level 2)
	for i := assoIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= 1 {
			break
		}
		if tag.Level == 2 {
			switch tag.Tag {
			case "RELA", "ROLE": // RELA in 5.5.1, ROLE in 7.0
				assoc.Role = tag.Value
			case "NOTE":
				assoc.Notes = append(assoc.Notes, tag.Value)
			}
		}
	}

	return assoc
}

// parseSourceCitation extracts a source citation from tags starting at sourIdx.
func parseSourceCitation(tags []*gedcom.Tag, sourIdx, baseLevel int) *gedcom.SourceCitation {
	cite := &gedcom.SourceCitation{
		SourceXRef: tags[sourIdx].Value,
	}

	// Look for subordinate tags at baseLevel+1
	for i := sourIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "PAGE":
				cite.Page = tag.Value
			case "QUAY":
				// Parse quality as integer (0-3)
				if q, err := strconv.Atoi(tag.Value); err == nil {
					cite.Quality = q
				}
			case "DATA":
				// Parse DATA subordinates at baseLevel+2
				cite.Data = parseSourceCitationData(tags, i, baseLevel+1)
			case "_APID":
				// Parse Ancestry Permanent Identifier (vendor extension)
				cite.AncestryAPID = gedcom.ParseAPID(tag.Value)
			}
		}
	}

	return cite
}

// parseSourceCitationData extracts source citation data from tags starting at dataIdx.
func parseSourceCitationData(tags []*gedcom.Tag, dataIdx, baseLevel int) *gedcom.SourceCitationData {
	data := &gedcom.SourceCitationData{}

	// Look for subordinate tags at baseLevel+1
	for i := dataIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "DATE":
				data.Date = tag.Value
			case "TEXT":
				data.Text = tag.Value
			}
		}
	}

	return data
}

// parseEvent extracts an event from tags starting at eventIdx.
//
//nolint:gocyclo // GEDCOM parsing inherently requires handling many tag types
func parseEvent(tags []*gedcom.Tag, eventIdx int, eventTag string) *gedcom.Event {
	event := &gedcom.Event{
		Type: gedcom.EventType(eventTag),
	}

	baseLevel := tags[eventIdx].Level

	// Look for subordinate tags at baseLevel+1
	for i := eventIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "DATE":
				event.Date = tag.Value
				if parsed, err := gedcom.ParseDate(tag.Value); err == nil {
					event.ParsedDate = parsed
				}
			case "PLAC":
				event.Place = tag.Value
				event.PlaceDetail = parsePlaceDetail(tags, i, tag.Level)
			case "TYPE":
				event.EventTypeDetail = tag.Value
			case "CAUS":
				event.Cause = tag.Value
			case "AGE":
				event.Age = tag.Value
			case "AGNC":
				event.Agency = tag.Value
			case "ADDR":
				event.Address = parseEventAddress(tags, i, tag.Level)
			case "PHON":
				event.Phone = append(event.Phone, tag.Value)
			case "EMAIL":
				event.Email = append(event.Email, tag.Value)
			case "FAX":
				event.Fax = append(event.Fax, tag.Value)
			case "WWW":
				event.Website = append(event.Website, tag.Value)
			case "RESN":
				event.Restriction = tag.Value
			case "UID":
				event.UID = tag.Value
			case "SDATE":
				event.SortDate = tag.Value
			case "NOTE":
				event.Notes = append(event.Notes, tag.Value)
			case "SOUR":
				cite := parseSourceCitation(tags, i, tag.Level)
				event.SourceCitations = append(event.SourceCitations, cite)
			case "OBJE":
				link := parseMediaLink(tags, i, tag.Level)
				event.Media = append(event.Media, link)
			}
		}
	}

	return event
}

// parseEventAddress extracts an address structure from tags starting at addrIdx.
func parseEventAddress(tags []*gedcom.Tag, addrIdx, baseLevel int) *gedcom.Address {
	addr := &gedcom.Address{
		Line1: tags[addrIdx].Value,
	}

	// Look for subordinate tags at baseLevel+1
	for i := addrIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "ADR1":
				addr.Line1 = tag.Value
			case "ADR2":
				addr.Line2 = tag.Value
			case "ADR3":
				addr.Line3 = tag.Value
			case "CITY":
				addr.City = tag.Value
			case "STAE":
				addr.State = tag.Value
			case "POST":
				addr.PostalCode = tag.Value
			case "CTRY":
				addr.Country = tag.Value
			case "CONT":
				// Continue address on next line
				if addr.Line1 != "" {
					addr.Line1 += "\n" + tag.Value
				} else {
					addr.Line1 = tag.Value
				}
			case "CONC":
				// Concatenate to address
				addr.Line1 += tag.Value
			}
		}
	}

	return addr
}

// parsePlaceDetail extracts a place structure with optional coordinates from tags starting at placIdx.
func parsePlaceDetail(tags []*gedcom.Tag, placIdx, baseLevel int) *gedcom.PlaceDetail {
	place := &gedcom.PlaceDetail{
		Name: tags[placIdx].Value,
	}

	// Look for subordinate tags at baseLevel+1
	for i := placIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "FORM":
				place.Form = tag.Value
			case "MAP":
				place.Coordinates = parseCoordinates(tags, i, tag.Level)
			}
		}
	}

	return place
}

// parseCoordinates extracts geographic coordinates from tags starting at mapIdx.
func parseCoordinates(tags []*gedcom.Tag, mapIdx, baseLevel int) *gedcom.Coordinates {
	coords := &gedcom.Coordinates{}

	// Look for subordinate tags at baseLevel+1
	for i := mapIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "LATI":
				coords.Latitude = tag.Value
			case "LONG":
				coords.Longitude = tag.Value
			}
		}
	}

	return coords
}

// parseAttribute extracts an attribute from tags starting at attrIdx.
func parseAttribute(tags []*gedcom.Tag, attrIdx int, attrTag string) *gedcom.Attribute {
	attr := &gedcom.Attribute{
		Type:  attrTag,
		Value: tags[attrIdx].Value,
	}

	// Look for subordinate tags (level 2)
	for i := attrIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= 1 {
			break
		}
		if tag.Level == 2 {
			switch tag.Tag {
			case "DATE":
				attr.Date = tag.Value
				if parsed, err := gedcom.ParseDate(tag.Value); err == nil {
					attr.ParsedDate = parsed
				}
			case "PLAC":
				attr.Place = tag.Value
			case "SOUR":
				cite := parseSourceCitation(tags, i, tag.Level)
				attr.SourceCitations = append(attr.SourceCitations, cite)
			}
		}
	}

	return attr
}

// ldsOrdinanceType maps a GEDCOM tag to its LDSOrdinanceType.
func ldsOrdinanceType(tag string) gedcom.LDSOrdinanceType {
	switch tag {
	case "BAPL":
		return gedcom.LDSBaptism
	case "CONL":
		return gedcom.LDSConfirmation
	case "ENDL":
		return gedcom.LDSEndowment
	case "SLGC":
		return gedcom.LDSSealingChild
	case "SLGS":
		return gedcom.LDSSealingSpouse
	default:
		return gedcom.LDSOrdinanceType(tag)
	}
}

// parseLDSOrdinance extracts an LDS ordinance from tags starting at ordIdx.
func parseLDSOrdinance(tags []*gedcom.Tag, ordIdx int, ordType gedcom.LDSOrdinanceType) *gedcom.LDSOrdinance {
	ord := &gedcom.LDSOrdinance{
		Type: ordType,
	}

	baseLevel := tags[ordIdx].Level

	// Look for subordinate tags at baseLevel+1
	for i := ordIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "DATE":
				ord.Date = tag.Value
				if parsed, err := gedcom.ParseDate(tag.Value); err == nil {
					ord.ParsedDate = parsed
				}
			case "TEMP":
				ord.Temple = tag.Value
			case "PLAC":
				ord.Place = tag.Value
			case "STAT":
				ord.Status = tag.Value
			case "FAMC":
				ord.FamilyXRef = tag.Value
			}
		}
	}

	return ord
}

// parseFamily converts record tags to a Family entity.
//
//nolint:gocyclo // GEDCOM parsing inherently requires handling many tag types
func parseFamily(record *gedcom.Record) *gedcom.Family {
	fam := &gedcom.Family{
		XRef: record.XRef,
		Tags: record.Tags,
	}

	for i := 0; i < len(record.Tags); i++ {
		tag := record.Tags[i]
		if tag.Level != 1 {
			continue
		}

		switch tag.Tag {
		case "HUSB":
			fam.Husband = tag.Value

		case "WIFE":
			fam.Wife = tag.Value

		case "CHIL":
			fam.Children = append(fam.Children, tag.Value)

		case "NCHI":
			fam.NumberOfChildren = tag.Value

		case "MARR", "DIV", "ENGA", "ANUL", "MARB", "MARC", "MARL", "MARS", "DIVF":
			event := parseEvent(record.Tags, i, tag.Tag)
			fam.Events = append(fam.Events, event)

		case "SLGS":
			ord := parseLDSOrdinance(record.Tags, i, ldsOrdinanceType(tag.Tag))
			fam.LDSOrdinances = append(fam.LDSOrdinances, ord)

		case "SOUR":
			cite := parseSourceCitation(record.Tags, i, tag.Level)
			fam.SourceCitations = append(fam.SourceCitations, cite)

		case "NOTE":
			fam.Notes = append(fam.Notes, tag.Value)

		case "OBJE":
			link := parseMediaLink(record.Tags, i, tag.Level)
			fam.Media = append(fam.Media, link)

		case "CHAN":
			fam.ChangeDate = parseChangeDate(record.Tags, i)

		case "CREA":
			fam.CreationDate = parseChangeDate(record.Tags, i)

		case "REFN":
			fam.RefNumber = tag.Value

		case "UID":
			fam.UID = tag.Value
		}
	}

	return fam
}

// parseSource converts record tags to a Source entity.
func parseSource(record *gedcom.Record) *gedcom.Source {
	src := &gedcom.Source{
		XRef: record.XRef,
		Tags: record.Tags,
	}

	for i := 0; i < len(record.Tags); i++ {
		tag := record.Tags[i]
		if tag.Level != 1 {
			continue
		}

		switch tag.Tag {
		case "TITL":
			src.Title = tag.Value
		case "AUTH":
			src.Author = tag.Value
		case "PUBL":
			src.Publication = tag.Value
		case "TEXT":
			src.Text = tag.Value
		case "REPO":
			if tag.Value != "" {
				src.RepositoryRef = tag.Value
			} else {
				// Look for inline repository with NAME subordinate
				src.Repository = parseInlineRepository(record.Tags, i)
			}
		case "NOTE":
			src.Notes = append(src.Notes, tag.Value)
		case "OBJE":
			link := parseMediaLink(record.Tags, i, tag.Level)
			src.Media = append(src.Media, link)
		case "CHAN":
			src.ChangeDate = parseChangeDate(record.Tags, i)
		case "CREA":
			src.CreationDate = parseChangeDate(record.Tags, i)
		case "REFN":
			src.RefNumber = tag.Value
		case "UID":
			src.UID = tag.Value
		}
	}

	return src
}

// parseInlineRepository extracts an inline repository from tags starting at repoIdx.
// An inline repository has no XRef value and contains subordinate tags like NAME.
func parseInlineRepository(tags []*gedcom.Tag, repoIdx int) *gedcom.InlineRepository {
	repo := &gedcom.InlineRepository{}

	baseLevel := tags[repoIdx].Level

	// Look for subordinate tags at baseLevel+1
	for i := repoIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 && tag.Tag == "NAME" {
			repo.Name = tag.Value
			break
		}
	}

	return repo
}

// parseChangeDate extracts a change date structure from tags starting at chanIdx.
// Used for both CHAN (change date) and CREA (creation date) tags.
func parseChangeDate(tags []*gedcom.Tag, chanIdx int) *gedcom.ChangeDate {
	cd := &gedcom.ChangeDate{}

	baseLevel := tags[chanIdx].Level

	// Look for subordinate tags at baseLevel+1
	for i := chanIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 && tag.Tag == "DATE" {
			cd.Date = tag.Value
			// Look for TIME subordinate at baseLevel+2
			for j := i + 1; j < len(tags); j++ {
				timeTag := tags[j]
				if timeTag.Level <= baseLevel+1 {
					break
				}
				if timeTag.Level == baseLevel+2 && timeTag.Tag == "TIME" {
					cd.Time = timeTag.Value
					break
				}
			}
		}
	}

	return cd
}

// parseSubmitter converts record tags to a Submitter entity.
func parseSubmitter(record *gedcom.Record) *gedcom.Submitter {
	subm := &gedcom.Submitter{
		XRef: record.XRef,
		Tags: record.Tags,
	}

	for i := 0; i < len(record.Tags); i++ {
		tag := record.Tags[i]
		if tag.Level != 1 {
			continue
		}

		switch tag.Tag {
		case "NAME":
			subm.Name = tag.Value

		case "ADDR":
			subm.Address = parseEventAddress(record.Tags, i, tag.Level)

		case "PHON":
			subm.Phone = append(subm.Phone, tag.Value)

		case "EMAIL":
			subm.Email = append(subm.Email, tag.Value)

		case "LANG":
			subm.Language = append(subm.Language, tag.Value)

		case "NOTE":
			subm.Notes = append(subm.Notes, tag.Value)
		}
	}

	return subm
}

// parseRepository converts record tags to a Repository entity.
func parseRepository(record *gedcom.Record) *gedcom.Repository {
	repo := &gedcom.Repository{
		XRef: record.XRef,
		Tags: record.Tags,
	}

	for i := 0; i < len(record.Tags); i++ {
		tag := record.Tags[i]
		if tag.Level != 1 {
			continue
		}

		switch tag.Tag {
		case "NAME":
			repo.Name = tag.Value

		case "ADDR":
			repo.Address = parseEventAddress(record.Tags, i, tag.Level)

		case "PHON":
			if repo.Address == nil {
				repo.Address = &gedcom.Address{}
			}
			repo.Address.Phone = tag.Value

		case "EMAIL":
			if repo.Address == nil {
				repo.Address = &gedcom.Address{}
			}
			repo.Address.Email = tag.Value

		case "FAX":
			// FAX is not in Address struct, skip for now

		case "WWW":
			if repo.Address == nil {
				repo.Address = &gedcom.Address{}
			}
			repo.Address.Website = tag.Value

		case "NOTE":
			repo.Notes = append(repo.Notes, tag.Value)
		}
	}

	return repo
}

// parseNote converts record tags to a Note entity.
func parseNote(record *gedcom.Record) *gedcom.Note {
	note := &gedcom.Note{
		XRef: record.XRef,
		Tags: record.Tags,
		Text: record.Value, // The note text is in the value of the level 0 NOTE tag
	}

	// Process continuation lines
	for i := 0; i < len(record.Tags); i++ {
		tag := record.Tags[i]
		if tag.Level != 1 {
			continue
		}

		switch tag.Tag {
		case "CONT":
			// Continue with newline
			note.Continuation = append(note.Continuation, tag.Value)

		case "CONC":
			// Concatenate without newline to the last piece of text
			if len(note.Continuation) > 0 {
				// Append to last continuation
				note.Continuation[len(note.Continuation)-1] += tag.Value
			} else {
				// Append to main text
				note.Text += tag.Value
			}
		}
	}

	return note
}

// parseMediaObject converts record tags to a MediaObject entity.
//
//nolint:gocyclo // GEDCOM parsing inherently requires handling many tag types
func parseMediaObject(record *gedcom.Record) *gedcom.MediaObject {
	media := &gedcom.MediaObject{
		XRef: record.XRef,
		Tags: record.Tags,
	}

	for i := 0; i < len(record.Tags); i++ {
		tag := record.Tags[i]
		if tag.Level != 1 {
			continue
		}

		switch tag.Tag {
		case "FILE":
			file := parseMediaFile(record.Tags, i, tag.Level)
			media.Files = append(media.Files, file)
		case "NOTE":
			media.Notes = append(media.Notes, tag.Value)
		case "SOUR":
			cite := parseSourceCitation(record.Tags, i, tag.Level)
			media.SourceCitations = append(media.SourceCitations, cite)
		case "CHAN":
			media.ChangeDate = parseChangeDate(record.Tags, i)
		case "CREA":
			media.CreationDate = parseChangeDate(record.Tags, i)
		case "REFN":
			media.RefNumbers = append(media.RefNumbers, tag.Value)
		case "UID":
			media.UIDs = append(media.UIDs, tag.Value)
		case "RESN":
			media.Restriction = tag.Value
		}
	}

	return media
}

// parseMediaFile extracts a MediaFile from FILE tag and its subordinates.
func parseMediaFile(tags []*gedcom.Tag, fileIdx, baseLevel int) *gedcom.MediaFile {
	file := &gedcom.MediaFile{
		FileRef: tags[fileIdx].Value,
	}

	for i := fileIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "FORM":
				file.Form = tag.Value
				// Look for MEDI at baseLevel+2
				for j := i + 1; j < len(tags); j++ {
					mediTag := tags[j]
					if mediTag.Level <= baseLevel+1 {
						break
					}
					if mediTag.Level == baseLevel+2 && mediTag.Tag == "MEDI" {
						file.MediaType = mediTag.Value
						break
					}
				}
			case "TITL":
				file.Title = tag.Value
			case "TRAN":
				tran := parseMediaTranslation(tags, i, tag.Level)
				file.Translations = append(file.Translations, tran)
			}
		}
	}

	return file
}

// parseMediaTranslation extracts a MediaTranslation from TRAN tag and its subordinates.
func parseMediaTranslation(tags []*gedcom.Tag, tranIdx, baseLevel int) *gedcom.MediaTranslation {
	tran := &gedcom.MediaTranslation{
		FileRef: tags[tranIdx].Value,
	}

	for i := tranIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 && tag.Tag == "FORM" {
			tran.Form = tag.Value
			break
		}
	}

	return tran
}

// parseMediaLink extracts a MediaLink from OBJE reference tag and its subordinates.
func parseMediaLink(tags []*gedcom.Tag, objeIdx, baseLevel int) *gedcom.MediaLink {
	link := &gedcom.MediaLink{
		MediaXRef: tags[objeIdx].Value,
	}

	for i := objeIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "CROP":
				link.Crop = parseCropRegion(tags, i, tag.Level)
			case "TITL":
				link.Title = tag.Value
			}
		}
	}

	return link
}

// parseCropRegion extracts a CropRegion from CROP tag and its subordinates.
func parseCropRegion(tags []*gedcom.Tag, cropIdx, baseLevel int) *gedcom.CropRegion {
	crop := &gedcom.CropRegion{}

	for i := cropIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "TOP":
				if v, err := strconv.Atoi(tag.Value); err == nil {
					crop.Top = v
				}
			case "LEFT":
				if v, err := strconv.Atoi(tag.Value); err == nil {
					crop.Left = v
				}
			case "HEIGHT":
				if v, err := strconv.Atoi(tag.Value); err == nil {
					crop.Height = v
				}
			case "WIDTH":
				if v, err := strconv.Atoi(tag.Value); err == nil {
					crop.Width = v
				}
			}
		}
	}

	return crop
}
