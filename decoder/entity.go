package decoder

import (
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

		case "BIRT", "DEAT", "BAPM", "BURI", "CENS", "CHR", "OCCU", "RESI", "IMMI", "EMIG":
			event := parseEvent(record.Tags, i, tag.Tag)
			indi.Events = append(indi.Events, event)

		case "FAMC":
			indi.ChildInFamilies = append(indi.ChildInFamilies, tag.Value)

		case "FAMS":
			indi.SpouseInFamilies = append(indi.SpouseInFamilies, tag.Value)

		case "SOUR":
			indi.Sources = append(indi.Sources, tag.Value)

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
			case "TYPE":
				name.Type = tag.Value
			}
		}
	}

	return name
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
			case "NOTE":
				event.Notes = append(event.Notes, tag.Value)
			case "SOUR":
				event.Sources = append(event.Sources, tag.Value)
			}
		}
	}

	return event
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

		case "MARR", "DIV", "ENGA", "ANUL":
			event := parseEvent(record.Tags, i, tag.Tag)
			fam.Events = append(fam.Events, event)

		case "SOUR":
			fam.Sources = append(fam.Sources, tag.Value)

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
