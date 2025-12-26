package encoder

import (
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
)

func TestEntityToTags_NilEntity(t *testing.T) {
	record := &gedcom.Record{
		Type:   gedcom.RecordTypeIndividual,
		Entity: nil,
	}

	tags := entityToTags(record)
	if tags != nil {
		t.Errorf("entityToTags() with nil entity should return nil, got %v", tags)
	}
}

func TestEntityToTags_UnsupportedType(t *testing.T) {
	// Record with entity but unrecognized type
	record := &gedcom.Record{
		Type:   "UNKNOWN",
		Entity: &gedcom.Individual{},
	}

	tags := entityToTags(record)
	if tags != nil {
		t.Errorf("entityToTags() with unsupported type should return nil, got %v", tags)
	}
}

func TestEntityToTags_TypeMismatch(t *testing.T) {
	// Record type is INDI but entity is Family
	record := &gedcom.Record{
		Type:   gedcom.RecordTypeIndividual,
		Entity: &gedcom.Family{}, // Wrong type
	}

	tags := entityToTags(record)
	if tags != nil {
		t.Errorf("entityToTags() with type mismatch should return nil, got %v", tags)
	}
}

func TestIndividualToTags(t *testing.T) {
	tests := []struct {
		name     string
		indi     *gedcom.Individual
		contains []string
	}{
		{
			name: "basic individual with name and sex",
			indi: &gedcom.Individual{
				XRef: "@I1@",
				Names: []*gedcom.PersonalName{
					{Full: "John /Doe/", Given: "John", Surname: "Doe"},
				},
				Sex: "M",
			},
			contains: []string{"NAME", "GIVN", "SURN", "SEX"},
		},
		{
			name: "individual with complete name parts",
			indi: &gedcom.Individual{
				Names: []*gedcom.PersonalName{
					{
						Full:          "Dr. Johann Ludwig /von Beethoven/ III",
						Given:         "Johann Ludwig",
						Surname:       "Beethoven",
						Prefix:        "Dr.",
						Suffix:        "III",
						Nickname:      "Ludwig",
						SurnamePrefix: "von",
						Type:          "birth",
					},
				},
			},
			contains: []string{"NAME", "GIVN", "SURN", "NPFX", "NSFX", "NICK", "SPFX", "TYPE"},
		},
		{
			name: "individual with events",
			indi: &gedcom.Individual{
				Events: []*gedcom.Event{
					{
						Type:  gedcom.EventBirth,
						Date:  "1 JAN 1900",
						Place: "London, England",
					},
					{
						Type:  gedcom.EventDeath,
						Date:  "31 DEC 1980",
						Cause: "Natural causes",
					},
				},
			},
			contains: []string{"BIRT", "DATE", "PLAC", "DEAT", "CAUS"},
		},
		{
			name: "individual with attributes",
			indi: &gedcom.Individual{
				Attributes: []*gedcom.Attribute{
					{Type: "OCCU", Value: "Software Engineer", Date: "2000"},
					{Type: "EDUC", Value: "PhD Computer Science", Place: "MIT"},
				},
			},
			contains: []string{"OCCU", "EDUC", "DATE", "PLAC"},
		},
		{
			name: "individual with family links",
			indi: &gedcom.Individual{
				ChildInFamilies:  []gedcom.FamilyLink{{FamilyXRef: "@F1@", Pedigree: "birth"}},
				SpouseInFamilies: []string{"@F2@"},
			},
			contains: []string{"FAMC", "PEDI", "FAMS"},
		},
		{
			name: "individual with associations",
			indi: &gedcom.Individual{
				Associations: []*gedcom.Association{
					{IndividualXRef: "@I2@", Role: "GODP", Notes: []string{"Godparent note"}},
				},
			},
			contains: []string{"ASSO", "ROLE", "NOTE"},
		},
		{
			name: "individual with LDS ordinances",
			indi: &gedcom.Individual{
				LDSOrdinances: []*gedcom.LDSOrdinance{
					{Type: "BAPL", Date: "1 JAN 1900", Temple: "SLAKE", Status: "COMPLETED"},
					{Type: "SLGC", Date: "15 JAN 1900", FamilyXRef: "@F1@"},
				},
			},
			contains: []string{"BAPL", "DATE", "TEMP", "STAT", "SLGC", "FAMC"},
		},
		{
			name: "individual with source citations",
			indi: &gedcom.Individual{
				SourceCitations: []*gedcom.SourceCitation{
					{SourceXRef: "@S1@", Page: "p. 42", Quality: 3},
				},
			},
			contains: []string{"SOUR", "PAGE", "QUAY"},
		},
		{
			name: "individual with notes and media",
			indi: &gedcom.Individual{
				Notes: []string{"A note about this person"},
				Media: []*gedcom.MediaLink{
					{MediaXRef: "@O1@", Title: "Photo"},
				},
			},
			contains: []string{"NOTE", "OBJE", "TITL"},
		},
		{
			name: "individual with metadata",
			indi: &gedcom.Individual{
				ChangeDate:   &gedcom.ChangeDate{Date: "1 JAN 2024", Time: "12:00:00"},
				CreationDate: &gedcom.ChangeDate{Date: "1 JAN 2020"},
				RefNumber:    "REF123",
				UID:          "UID-12345",
			},
			contains: []string{"CHAN", "CREA", "REFN", "UID", "TIME"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := individualToTags(tt.indi)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("individualToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

func TestFamilyToTags(t *testing.T) {
	tests := []struct {
		name     string
		fam      *gedcom.Family
		contains []string
	}{
		{
			name: "basic family",
			fam: &gedcom.Family{
				Husband:  "@I1@",
				Wife:     "@I2@",
				Children: []string{"@I3@", "@I4@"},
			},
			contains: []string{"HUSB", "WIFE", "CHIL"},
		},
		{
			name: "family with number of children",
			fam: &gedcom.Family{
				NumberOfChildren: "3",
			},
			contains: []string{"NCHI"},
		},
		{
			name: "family with events",
			fam: &gedcom.Family{
				Events: []*gedcom.Event{
					{Type: gedcom.EventMarriage, Date: "15 JUN 1920", Place: "Boston, MA"},
					{Type: gedcom.EventDivorce, Date: "1 JAN 1940"},
				},
			},
			contains: []string{"MARR", "DATE", "PLAC", "DIV"},
		},
		{
			name: "family with LDS ordinances",
			fam: &gedcom.Family{
				LDSOrdinances: []*gedcom.LDSOrdinance{
					{Type: "SLGS", Date: "1 JAN 1920", Temple: "SLAKE", Status: "COMPLETED"},
				},
			},
			contains: []string{"SLGS", "DATE", "TEMP", "STAT"},
		},
		{
			name: "family with source citations",
			fam: &gedcom.Family{
				SourceCitations: []*gedcom.SourceCitation{
					{SourceXRef: "@S1@", Page: "Marriage cert"},
				},
			},
			contains: []string{"SOUR", "PAGE"},
		},
		{
			name: "family with notes and media",
			fam: &gedcom.Family{
				Notes: []string{"Family note"},
				Media: []*gedcom.MediaLink{{MediaXRef: "@O1@"}},
			},
			contains: []string{"NOTE", "OBJE"},
		},
		{
			name: "family with metadata",
			fam: &gedcom.Family{
				ChangeDate:   &gedcom.ChangeDate{Date: "1 JAN 2024"},
				CreationDate: &gedcom.ChangeDate{Date: "1 JAN 2020"},
				RefNumber:    "REF456",
				UID:          "UID-FAMILY",
			},
			contains: []string{"CHAN", "CREA", "REFN", "UID"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := familyToTags(tt.fam)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("familyToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

func TestSourceToTags(t *testing.T) {
	tests := []struct {
		name     string
		src      *gedcom.Source
		contains []string
	}{
		{
			name: "basic source",
			src: &gedcom.Source{
				Title:  "Birth Records",
				Author: "John Archivist",
			},
			contains: []string{"TITL", "AUTH"},
		},
		{
			name: "full source",
			src: &gedcom.Source{
				Title:         "County Records",
				Author:        "Jane Historian",
				Publication:   "Published 2000",
				Text:          "Source text content",
				RepositoryRef: "@R1@",
			},
			contains: []string{"TITL", "AUTH", "PUBL", "TEXT", "REPO"},
		},
		{
			name: "source with media and notes",
			src: &gedcom.Source{
				Title: "Source with attachments",
				Media: []*gedcom.MediaLink{{MediaXRef: "@O1@"}},
				Notes: []string{"Source note"},
			},
			contains: []string{"TITL", "OBJE", "NOTE"},
		},
		{
			name: "source with metadata",
			src: &gedcom.Source{
				Title:        "Source with metadata",
				ChangeDate:   &gedcom.ChangeDate{Date: "1 JAN 2024"},
				CreationDate: &gedcom.ChangeDate{Date: "1 JAN 2020"},
				RefNumber:    "REF789",
				UID:          "UID-SOURCE",
			},
			contains: []string{"TITL", "CHAN", "CREA", "REFN", "UID"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := sourceToTags(tt.src)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("sourceToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

func TestSubmitterToTags(t *testing.T) {
	tests := []struct {
		name     string
		subm     *gedcom.Submitter
		contains []string
	}{
		{
			name: "basic submitter",
			subm: &gedcom.Submitter{
				Name: "John Genealogist",
			},
			contains: []string{"NAME"},
		},
		{
			name: "submitter with address",
			subm: &gedcom.Submitter{
				Name: "Jane Researcher",
				Address: &gedcom.Address{
					Line1:      "123 Main St",
					City:       "Boston",
					State:      "MA",
					PostalCode: "02101",
					Country:    "USA",
				},
			},
			contains: []string{"NAME", "ADDR", "ADR1", "CITY", "STAE", "POST", "CTRY"},
		},
		{
			name: "submitter with contact info",
			subm: &gedcom.Submitter{
				Name:     "Bob Archivist",
				Phone:    []string{"555-1234", "555-5678"},
				Email:    []string{"bob@example.com"},
				Language: []string{"English", "German"},
			},
			contains: []string{"NAME", "PHON", "EMAIL", "LANG"},
		},
		{
			name: "submitter with notes",
			subm: &gedcom.Submitter{
				Name:  "Alice Compiler",
				Notes: []string{"Submitter note"},
			},
			contains: []string{"NAME", "NOTE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := submitterToTags(tt.subm)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("submitterToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

func TestRepositoryToTags(t *testing.T) {
	tests := []struct {
		name     string
		repo     *gedcom.Repository
		contains []string
	}{
		{
			name: "basic repository",
			repo: &gedcom.Repository{
				Name: "City Archives",
			},
			contains: []string{"NAME"},
		},
		{
			name: "repository with address",
			repo: &gedcom.Repository{
				Name: "State Library",
				Address: &gedcom.Address{
					Line1:   "100 Library Way",
					City:    "Albany",
					State:   "NY",
					Country: "USA",
				},
			},
			contains: []string{"NAME", "ADDR", "ADR1", "CITY", "STAE", "CTRY"},
		},
		{
			name: "repository with notes",
			repo: &gedcom.Repository{
				Name:  "Family History Center",
				Notes: []string{"Open Mon-Fri 9-5"},
			},
			contains: []string{"NAME", "NOTE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := repositoryToTags(tt.repo)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("repositoryToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

func TestNoteToTags(t *testing.T) {
	tests := []struct {
		name     string
		note     *gedcom.Note
		contains []string
	}{
		{
			name:     "note without continuation",
			note:     &gedcom.Note{Text: "Simple note"},
			contains: []string{}, // No CONT expected
		},
		{
			name: "note with continuation",
			note: &gedcom.Note{
				Text:         "First line",
				Continuation: []string{"Second line", "Third line"},
			},
			contains: []string{"CONT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := noteToTags(tt.note)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("noteToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

func TestMediaObjectToTags(t *testing.T) {
	tests := []struct {
		name     string
		media    *gedcom.MediaObject
		contains []string
	}{
		{
			name: "media with single file",
			media: &gedcom.MediaObject{
				Files: []*gedcom.MediaFile{
					{FileRef: "photo.jpg", Form: "image/jpeg"},
				},
			},
			contains: []string{"FILE", "FORM"},
		},
		{
			name: "media with complete file info",
			media: &gedcom.MediaObject{
				Files: []*gedcom.MediaFile{
					{
						FileRef:   "photo.jpg",
						Form:      "image/jpeg",
						MediaType: "PHOTO",
						Title:     "Family Photo",
					},
				},
			},
			contains: []string{"FILE", "FORM", "MEDI", "TITL"},
		},
		{
			name: "media with translations",
			media: &gedcom.MediaObject{
				Files: []*gedcom.MediaFile{
					{
						FileRef: "audio.mp3",
						Form:    "audio/mpeg",
						Translations: []*gedcom.MediaTranslation{
							{FileRef: "transcript.txt", Form: "text/plain"},
						},
					},
				},
			},
			contains: []string{"FILE", "FORM", "TRAN"},
		},
		{
			name: "media with source citations",
			media: &gedcom.MediaObject{
				Files: []*gedcom.MediaFile{
					{FileRef: "document.pdf", Form: "application/pdf"},
				},
				SourceCitations: []*gedcom.SourceCitation{
					{SourceXRef: "@S1@"},
				},
			},
			contains: []string{"FILE", "SOUR"},
		},
		{
			name: "media with notes",
			media: &gedcom.MediaObject{
				Files: []*gedcom.MediaFile{
					{FileRef: "image.png"},
				},
				Notes: []string{"Media note"},
			},
			contains: []string{"FILE", "NOTE"},
		},
		{
			name: "media with metadata",
			media: &gedcom.MediaObject{
				Files: []*gedcom.MediaFile{
					{FileRef: "file.jpg"},
				},
				ChangeDate:   &gedcom.ChangeDate{Date: "1 JAN 2024"},
				CreationDate: &gedcom.ChangeDate{Date: "1 JAN 2020"},
				RefNumbers:   []string{"REF001", "REF002"},
				UIDs:         []string{"UID-001"},
				Restriction:  "locked",
			},
			contains: []string{"FILE", "CHAN", "CREA", "REFN", "UID", "RESN"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := mediaObjectToTags(tt.media)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("mediaObjectToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

func TestNameToTags(t *testing.T) {
	tests := []struct {
		name     string
		pname    *gedcom.PersonalName
		level    int
		contains []string
	}{
		{
			name:     "minimal name",
			pname:    &gedcom.PersonalName{Full: "John /Doe/"},
			level:    1,
			contains: []string{"NAME"},
		},
		{
			name: "full name with all parts",
			pname: &gedcom.PersonalName{
				Full:          "Dr. Johann /von Beethoven/ Jr.",
				Given:         "Johann",
				Surname:       "Beethoven",
				Prefix:        "Dr.",
				Suffix:        "Jr.",
				Nickname:      "Jo",
				SurnamePrefix: "von",
				Type:          "birth",
			},
			level:    1,
			contains: []string{"NAME", "GIVN", "SURN", "NPFX", "NSFX", "NICK", "SPFX", "TYPE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := nameToTags(tt.pname, tt.level)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("nameToTags() missing expected tag %q", expected)
				}
			}

			// Verify level is correct for NAME tag
			if len(tags) > 0 && tags[0].Level != tt.level {
				t.Errorf("nameToTags() NAME tag level = %d, want %d", tags[0].Level, tt.level)
			}
		})
	}
}

func TestEventToTags(t *testing.T) {
	tests := []struct {
		name     string
		event    *gedcom.Event
		level    int
		contains []string
	}{
		{
			name:     "minimal event",
			event:    &gedcom.Event{Type: gedcom.EventBirth},
			level:    1,
			contains: []string{"BIRT"},
		},
		{
			name: "event with date and place",
			event: &gedcom.Event{
				Type:  gedcom.EventDeath,
				Date:  "31 DEC 1999",
				Place: "New York, NY",
			},
			level:    1,
			contains: []string{"DEAT", "DATE", "PLAC"},
		},
		{
			name: "event with place coordinates",
			event: &gedcom.Event{
				Type:  gedcom.EventBirth,
				Place: "Boston, MA",
				PlaceDetail: &gedcom.PlaceDetail{
					Form: "City, State",
					Coordinates: &gedcom.Coordinates{
						Latitude:  "N42.3601",
						Longitude: "W71.0589",
					},
				},
			},
			level:    1,
			contains: []string{"BIRT", "PLAC", "FORM", "MAP", "LATI", "LONG"},
		},
		{
			name: "event with subordinates",
			event: &gedcom.Event{
				Type:            gedcom.EventMarriage,
				Date:            "15 JUN 1920",
				EventTypeDetail: "Church wedding",
				Cause:           "Love",
				Age:             "25y",
				Agency:          "St. Mary's Church",
			},
			level:    1,
			contains: []string{"MARR", "DATE", "TYPE", "CAUS", "AGE", "AGNC"},
		},
		{
			name: "event with address",
			event: &gedcom.Event{
				Type: gedcom.EventBirth,
				Address: &gedcom.Address{
					Line1: "123 Hospital Road",
					City:  "Boston",
				},
			},
			level:    1,
			contains: []string{"BIRT", "ADDR", "ADR1", "CITY"},
		},
		{
			name: "event with contact info",
			event: &gedcom.Event{
				Type:    gedcom.EventMarriage,
				Phone:   []string{"555-1234"},
				Email:   []string{"info@church.org"},
				Fax:     []string{"555-4321"},
				Website: []string{"http://church.org"},
			},
			level:    1,
			contains: []string{"MARR", "PHON", "EMAIL", "FAX", "WWW"},
		},
		{
			name: "event with restriction and UID",
			event: &gedcom.Event{
				Type:        gedcom.EventDeath,
				Restriction: "confidential",
				UID:         "EVENT-UID-123",
				SortDate:    "1999-12-31",
			},
			level:    1,
			contains: []string{"DEAT", "RESN", "UID", "SDATE"},
		},
		{
			name: "event with notes and citations",
			event: &gedcom.Event{
				Type:  gedcom.EventBirth,
				Notes: []string{"Birth note"},
				SourceCitations: []*gedcom.SourceCitation{
					{SourceXRef: "@S1@", Page: "p. 100"},
				},
			},
			level:    1,
			contains: []string{"BIRT", "NOTE", "SOUR", "PAGE"},
		},
		{
			name: "event with media",
			event: &gedcom.Event{
				Type: gedcom.EventMarriage,
				Media: []*gedcom.MediaLink{
					{MediaXRef: "@O1@", Title: "Wedding Photo"},
				},
			},
			level:    1,
			contains: []string{"MARR", "OBJE", "TITL"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := eventToTags(tt.event, tt.level)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("eventToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

func TestAttributeToTags(t *testing.T) {
	tests := []struct {
		name     string
		attr     *gedcom.Attribute
		level    int
		contains []string
	}{
		{
			name:     "basic attribute",
			attr:     &gedcom.Attribute{Type: "OCCU", Value: "Engineer"},
			level:    1,
			contains: []string{"OCCU"},
		},
		{
			name: "attribute with date and place",
			attr: &gedcom.Attribute{
				Type:  "EDUC",
				Value: "Bachelor's Degree",
				Date:  "1985",
				Place: "MIT",
			},
			level:    1,
			contains: []string{"EDUC", "DATE", "PLAC"},
		},
		{
			name: "attribute with source citation",
			attr: &gedcom.Attribute{
				Type:  "OCCU",
				Value: "Doctor",
				SourceCitations: []*gedcom.SourceCitation{
					{SourceXRef: "@S1@"},
				},
			},
			level:    1,
			contains: []string{"OCCU", "SOUR"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := attributeToTags(tt.attr, tt.level)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("attributeToTags() missing expected tag %q", expected)
				}
			}

			// Verify level and value for main tag
			if len(tags) > 0 {
				if tags[0].Level != tt.level {
					t.Errorf("attributeToTags() main tag level = %d, want %d", tags[0].Level, tt.level)
				}
				if tags[0].Value != tt.attr.Value {
					t.Errorf("attributeToTags() main tag value = %q, want %q", tags[0].Value, tt.attr.Value)
				}
			}
		})
	}
}

func TestSourceCitationToTags(t *testing.T) {
	tests := []struct {
		name     string
		cite     *gedcom.SourceCitation
		level    int
		contains []string
	}{
		{
			name:     "minimal citation",
			cite:     &gedcom.SourceCitation{SourceXRef: "@S1@"},
			level:    2,
			contains: []string{"SOUR"},
		},
		{
			name: "citation with page and quality",
			cite: &gedcom.SourceCitation{
				SourceXRef: "@S1@",
				Page:       "p. 42",
				Quality:    3,
			},
			level:    2,
			contains: []string{"SOUR", "PAGE", "QUAY"},
		},
		{
			name: "citation with data",
			cite: &gedcom.SourceCitation{
				SourceXRef: "@S1@",
				Data: &gedcom.SourceCitationData{
					Date: "1 JAN 1900",
					Text: "Original text",
				},
			},
			level:    2,
			contains: []string{"SOUR", "DATA", "DATE", "TEXT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := sourceCitationToTags(tt.cite, tt.level)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("sourceCitationToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

func TestAddressToTags(t *testing.T) {
	tests := []struct {
		name     string
		addr     *gedcom.Address
		level    int
		contains []string
	}{
		{
			name:     "address with line1 only",
			addr:     &gedcom.Address{Line1: "123 Main St"},
			level:    1,
			contains: []string{"ADDR", "ADR1"},
		},
		{
			name: "full address",
			addr: &gedcom.Address{
				Line1:      "123 Main St",
				Line2:      "Apt 4",
				Line3:      "Building B",
				City:       "Boston",
				State:      "MA",
				PostalCode: "02101",
				Country:    "USA",
			},
			level:    1,
			contains: []string{"ADDR", "ADR1", "ADR2", "ADR3", "CITY", "STAE", "POST", "CTRY"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := addressToTags(tt.addr, tt.level)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("addressToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

func TestLDSOrdinanceToTags(t *testing.T) {
	tests := []struct {
		name     string
		ord      *gedcom.LDSOrdinance
		level    int
		contains []string
	}{
		{
			name:     "minimal ordinance",
			ord:      &gedcom.LDSOrdinance{Type: "BAPL"},
			level:    1,
			contains: []string{"BAPL"},
		},
		{
			name: "full ordinance",
			ord: &gedcom.LDSOrdinance{
				Type:   "ENDL",
				Date:   "1 JAN 1900",
				Temple: "LOGAN",
				Place:  "Logan, Utah",
				Status: "COMPLETED",
			},
			level:    1,
			contains: []string{"ENDL", "DATE", "TEMP", "PLAC", "STAT"},
		},
		{
			name: "SLGC with family reference",
			ord: &gedcom.LDSOrdinance{
				Type:       "SLGC",
				Date:       "1 FEB 1900",
				Temple:     "SLAKE",
				FamilyXRef: "@F1@",
			},
			level:    1,
			contains: []string{"SLGC", "DATE", "TEMP", "FAMC"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := ldsOrdinanceToTags(tt.ord, tt.level)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("ldsOrdinanceToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

func TestFamilyLinkToTags(t *testing.T) {
	tests := []struct {
		name     string
		link     *gedcom.FamilyLink
		level    int
		contains []string
	}{
		{
			name:     "link without pedigree",
			link:     &gedcom.FamilyLink{FamilyXRef: "@F1@"},
			level:    1,
			contains: []string{"FAMC"},
		},
		{
			name: "link with pedigree",
			link: &gedcom.FamilyLink{
				FamilyXRef: "@F1@",
				Pedigree:   "birth",
			},
			level:    1,
			contains: []string{"FAMC", "PEDI"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := familyLinkToTags(tt.link, tt.level)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("familyLinkToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

func TestAssociationToTags(t *testing.T) {
	tests := []struct {
		name     string
		assoc    *gedcom.Association
		level    int
		contains []string
	}{
		{
			name:     "minimal association",
			assoc:    &gedcom.Association{IndividualXRef: "@I2@"},
			level:    1,
			contains: []string{"ASSO"},
		},
		{
			name: "association with role",
			assoc: &gedcom.Association{
				IndividualXRef: "@I2@",
				Role:           "GODP",
			},
			level:    1,
			contains: []string{"ASSO", "ROLE"},
		},
		{
			name: "association with notes",
			assoc: &gedcom.Association{
				IndividualXRef: "@I2@",
				Role:           "WITN",
				Notes:          []string{"Witness at wedding"},
			},
			level:    1,
			contains: []string{"ASSO", "ROLE", "NOTE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := associationToTags(tt.assoc, tt.level)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("associationToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

func TestChangeDateToTags(t *testing.T) {
	tests := []struct {
		name     string
		cd       *gedcom.ChangeDate
		level    int
		tagName  string
		contains []string
	}{
		{
			name:     "date only",
			cd:       &gedcom.ChangeDate{Date: "1 JAN 2024"},
			level:    1,
			tagName:  "CHAN",
			contains: []string{"CHAN", "DATE"},
		},
		{
			name:     "date with time",
			cd:       &gedcom.ChangeDate{Date: "1 JAN 2024", Time: "12:30:00"},
			level:    1,
			tagName:  "CHAN",
			contains: []string{"CHAN", "DATE", "TIME"},
		},
		{
			name:     "creation date",
			cd:       &gedcom.ChangeDate{Date: "1 JAN 2020"},
			level:    1,
			tagName:  "CREA",
			contains: []string{"CREA", "DATE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := changeDateToTags(tt.cd, tt.level, tt.tagName)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("changeDateToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

func TestMediaLinkToTags(t *testing.T) {
	tests := []struct {
		name     string
		link     *gedcom.MediaLink
		level    int
		contains []string
	}{
		{
			name:     "minimal media link",
			link:     &gedcom.MediaLink{MediaXRef: "@O1@"},
			level:    1,
			contains: []string{"OBJE"},
		},
		{
			name: "media link with title",
			link: &gedcom.MediaLink{
				MediaXRef: "@O1@",
				Title:     "Family Photo",
			},
			level:    1,
			contains: []string{"OBJE", "TITL"},
		},
		{
			name: "media link with crop",
			link: &gedcom.MediaLink{
				MediaXRef: "@O1@",
				Crop: &gedcom.CropRegion{
					Top:    10,
					Left:   20,
					Height: 100,
					Width:  200,
				},
			},
			level:    1,
			contains: []string{"OBJE", "CROP", "TOP", "LEFT", "HEIGHT", "WIDTH"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := mediaLinkToTags(tt.link, tt.level)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("mediaLinkToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

func TestCropRegionToTags(t *testing.T) {
	tests := []struct {
		name     string
		crop     *gedcom.CropRegion
		level    int
		contains []string
	}{
		{
			name:     "zero values not written",
			crop:     &gedcom.CropRegion{},
			level:    2,
			contains: []string{"CROP"},
		},
		{
			name: "all values set",
			crop: &gedcom.CropRegion{
				Top:    10,
				Left:   20,
				Height: 100,
				Width:  200,
			},
			level:    2,
			contains: []string{"CROP", "TOP", "LEFT", "HEIGHT", "WIDTH"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := cropRegionToTags(tt.crop, tt.level)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("cropRegionToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

func TestMediaFileToTags(t *testing.T) {
	tests := []struct {
		name     string
		file     *gedcom.MediaFile
		level    int
		contains []string
	}{
		{
			name:     "minimal file",
			file:     &gedcom.MediaFile{FileRef: "photo.jpg"},
			level:    1,
			contains: []string{"FILE"},
		},
		{
			name: "file with form",
			file: &gedcom.MediaFile{
				FileRef: "photo.jpg",
				Form:    "image/jpeg",
			},
			level:    1,
			contains: []string{"FILE", "FORM"},
		},
		{
			name: "file with media type",
			file: &gedcom.MediaFile{
				FileRef:   "photo.jpg",
				Form:      "image/jpeg",
				MediaType: "PHOTO",
			},
			level:    1,
			contains: []string{"FILE", "FORM", "MEDI"},
		},
		{
			name: "file with title",
			file: &gedcom.MediaFile{
				FileRef: "photo.jpg",
				Title:   "Wedding Photo",
			},
			level:    1,
			contains: []string{"FILE", "TITL"},
		},
		{
			name: "file with translations",
			file: &gedcom.MediaFile{
				FileRef: "audio.mp3",
				Form:    "audio/mpeg",
				Translations: []*gedcom.MediaTranslation{
					{FileRef: "transcript.txt", Form: "text/plain"},
				},
			},
			level:    1,
			contains: []string{"FILE", "FORM", "TRAN"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := mediaFileToTags(tt.file, tt.level)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("mediaFileToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

func TestMediaTranslationToTags(t *testing.T) {
	tests := []struct {
		name     string
		tran     *gedcom.MediaTranslation
		level    int
		contains []string
	}{
		{
			name:     "minimal translation",
			tran:     &gedcom.MediaTranslation{FileRef: "alt.txt"},
			level:    2,
			contains: []string{"TRAN"},
		},
		{
			name: "translation with form",
			tran: &gedcom.MediaTranslation{
				FileRef: "alt.txt",
				Form:    "text/plain",
			},
			level:    2,
			contains: []string{"TRAN", "FORM"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := mediaTranslationToTags(tt.tran, tt.level)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("mediaTranslationToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

func TestPlaceToTags(t *testing.T) {
	tests := []struct {
		name     string
		place    string
		detail   *gedcom.PlaceDetail
		level    int
		contains []string
	}{
		{
			name:     "place name only",
			place:    "Boston, MA",
			detail:   nil,
			level:    2,
			contains: []string{"PLAC"},
		},
		{
			name:     "place with form",
			place:    "Boston, MA",
			detail:   &gedcom.PlaceDetail{Form: "City, State"},
			level:    2,
			contains: []string{"PLAC", "FORM"},
		},
		{
			name:  "place with coordinates",
			place: "Boston, MA",
			detail: &gedcom.PlaceDetail{
				Coordinates: &gedcom.Coordinates{
					Latitude:  "N42.3601",
					Longitude: "W71.0589",
				},
			},
			level:    2,
			contains: []string{"PLAC", "MAP", "LATI", "LONG"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := placeToTags(tt.place, tt.detail, tt.level)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("placeToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

func TestCoordinatesToTags(t *testing.T) {
	tests := []struct {
		name     string
		coords   *gedcom.Coordinates
		level    int
		contains []string
	}{
		{
			name:     "empty coordinates",
			coords:   &gedcom.Coordinates{},
			level:    3,
			contains: []string{"MAP"},
		},
		{
			name: "full coordinates",
			coords: &gedcom.Coordinates{
				Latitude:  "N42.3601",
				Longitude: "W71.0589",
			},
			level:    3,
			contains: []string{"MAP", "LATI", "LONG"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := coordinatesToTags(tt.coords, tt.level)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("coordinatesToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

func TestSourceCitationDataToTags(t *testing.T) {
	tests := []struct {
		name     string
		data     *gedcom.SourceCitationData
		level    int
		contains []string
	}{
		{
			name:     "empty data",
			data:     &gedcom.SourceCitationData{},
			level:    3,
			contains: []string{"DATA"},
		},
		{
			name: "data with date only",
			data: &gedcom.SourceCitationData{
				Date: "1 JAN 1900",
			},
			level:    3,
			contains: []string{"DATA", "DATE"},
		},
		{
			name: "full data",
			data: &gedcom.SourceCitationData{
				Date: "1 JAN 1900",
				Text: "Original text from source",
			},
			level:    3,
			contains: []string{"DATA", "DATE", "TEXT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := sourceCitationDataToTags(tt.data, tt.level)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("sourceCitationDataToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

// TestEntityToTagsDispatch tests that entityToTags correctly dispatches to entity-specific converters
func TestEntityToTagsDispatch(t *testing.T) {
	tests := []struct {
		name       string
		record     *gedcom.Record
		expectTags bool
	}{
		{
			name: "individual dispatch",
			record: &gedcom.Record{
				Type:   gedcom.RecordTypeIndividual,
				Entity: &gedcom.Individual{Sex: "M"},
			},
			expectTags: true,
		},
		{
			name: "family dispatch",
			record: &gedcom.Record{
				Type:   gedcom.RecordTypeFamily,
				Entity: &gedcom.Family{Husband: "@I1@"},
			},
			expectTags: true,
		},
		{
			name: "source dispatch",
			record: &gedcom.Record{
				Type:   gedcom.RecordTypeSource,
				Entity: &gedcom.Source{Title: "Test"},
			},
			expectTags: true,
		},
		{
			name: "submitter dispatch",
			record: &gedcom.Record{
				Type:   gedcom.RecordTypeSubmitter,
				Entity: &gedcom.Submitter{Name: "Test"},
			},
			expectTags: true,
		},
		{
			name: "repository dispatch",
			record: &gedcom.Record{
				Type:   gedcom.RecordTypeRepository,
				Entity: &gedcom.Repository{Name: "Test"},
			},
			expectTags: true,
		},
		{
			name: "note dispatch",
			record: &gedcom.Record{
				Type:   gedcom.RecordTypeNote,
				Entity: &gedcom.Note{Continuation: []string{"line"}},
			},
			expectTags: true,
		},
		{
			name: "media dispatch",
			record: &gedcom.Record{
				Type: gedcom.RecordTypeMedia,
				Entity: &gedcom.MediaObject{
					Files: []*gedcom.MediaFile{{FileRef: "test.jpg"}},
				},
			},
			expectTags: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := entityToTags(tt.record)
			if tt.expectTags && tags == nil {
				t.Errorf("entityToTags() returned nil, expected tags")
			}
			if !tt.expectTags && tags != nil {
				t.Errorf("entityToTags() returned tags, expected nil")
			}
		})
	}
}

// tagNamesToMap converts a slice of tags to a map of tag names for easier assertion
func tagNamesToMap(tags []*gedcom.Tag) map[string]bool {
	result := make(map[string]bool)
	for _, tag := range tags {
		result[tag.Tag] = true
	}
	return result
}
