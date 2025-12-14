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
		}
	}
}

// parseIndividual converts record tags to an Individual entity.
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

		case "OCCU", "CAST", "DSCR", "EDUC", "IDNO", "NATI", "SSN", "TITL", "RELI":
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
			indi.MediaRefs = append(indi.MediaRefs, tag.Value)
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
func parseEvent(tags []*gedcom.Tag, eventIdx int, eventTag string) *gedcom.Event {
	event := &gedcom.Event{
		Type: gedcom.EventType(eventTag),
	}

	// Look for subordinate tags (level 2)
	for i := eventIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= 1 {
			break
		}
		if tag.Level == 2 {
			switch tag.Tag {
			case "DATE":
				event.Date = tag.Value
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
			case "NOTE":
				event.Notes = append(event.Notes, tag.Value)
			case "SOUR":
				cite := parseSourceCitation(tags, i, tag.Level)
				event.SourceCitations = append(event.SourceCitations, cite)
			}
		}
	}

	return event
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
			fam.MediaRefs = append(fam.MediaRefs, tag.Value)
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

	for _, tag := range record.Tags {
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
			src.RepositoryRef = tag.Value
		case "NOTE":
			src.Notes = append(src.Notes, tag.Value)
		case "OBJE":
			src.MediaRefs = append(src.MediaRefs, tag.Value)
		}
	}

	return src
}
