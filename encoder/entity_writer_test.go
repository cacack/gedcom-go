package encoder

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/decoder"
	"github.com/cacack/gedcom-go/gedcom"
)

func TestEntityToTags_NilEntity(t *testing.T) {
	record := &gedcom.Record{
		Type:   gedcom.RecordTypeIndividual,
		Entity: nil,
	}

	tags := entityToTags(record, nil)
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

	tags := entityToTags(record, nil)
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

	tags := entityToTags(record, nil)
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
			tags := individualToTags(tt.indi, nil)
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
			tags := familyToTags(tt.fam, nil)
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
		{
			name: "source with inline repository",
			src: &gedcom.Source{
				Title:      "Source with inline repo",
				Repository: &gedcom.InlineRepository{Name: "State Archives"},
			},
			contains: []string{"TITL", "REPO", "NAME"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := sourceToTags(tt.src, nil)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("sourceToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

// TestSourceInlineRepositoryEncoding tests inline repository encoding in detail
func TestSourceInlineRepositoryEncoding(t *testing.T) {
	tests := []struct {
		name            string
		src             *gedcom.Source
		expectRepoTag   bool
		expectRepoValue string
		expectNameTag   bool
		expectNameValue string
	}{
		{
			name: "repository XRef takes precedence",
			src: &gedcom.Source{
				Title:         "Test Source",
				RepositoryRef: "@R1@",
				Repository:    &gedcom.InlineRepository{Name: "Should be ignored"},
			},
			expectRepoTag:   true,
			expectRepoValue: "@R1@",
			expectNameTag:   false,
		},
		{
			name: "inline repository when no XRef",
			src: &gedcom.Source{
				Title:      "Test Source",
				Repository: &gedcom.InlineRepository{Name: "State Archives"},
			},
			expectRepoTag:   true,
			expectRepoValue: "",
			expectNameTag:   true,
			expectNameValue: "State Archives",
		},
		{
			name: "no repository when both empty",
			src: &gedcom.Source{
				Title: "Test Source",
			},
			expectRepoTag: false,
		},
		{
			name: "no repository when inline repo has empty name",
			src: &gedcom.Source{
				Title:      "Test Source",
				Repository: &gedcom.InlineRepository{Name: ""},
			},
			expectRepoTag: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := sourceToTags(tt.src, nil)

			// Find REPO tag
			var repoTag, nameTag *gedcom.Tag
			for i, tag := range tags {
				if tag.Tag == "REPO" && tag.Level == 1 {
					repoTag = tag
					// Look for NAME at next position
					if i+1 < len(tags) && tags[i+1].Tag == "NAME" && tags[i+1].Level == 2 {
						nameTag = tags[i+1]
					}
					break
				}
			}

			if tt.expectRepoTag {
				if repoTag == nil {
					t.Error("expected REPO tag but not found")
					return
				}
				if repoTag.Value != tt.expectRepoValue {
					t.Errorf("REPO value = %q, want %q", repoTag.Value, tt.expectRepoValue)
				}
			} else if repoTag != nil {
				t.Errorf("expected no REPO tag but found one with value %q", repoTag.Value)
			}

			if tt.expectNameTag {
				if nameTag == nil {
					t.Error("expected NAME tag but not found")
					return
				}
				if nameTag.Value != tt.expectNameValue {
					t.Errorf("NAME value = %q, want %q", nameTag.Value, tt.expectNameValue)
				}
			} else if nameTag != nil {
				t.Errorf("expected no NAME tag but found one with value %q", nameTag.Value)
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
			tags := submitterToTags(tt.subm, nil)
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
			tags := repositoryToTags(tt.repo, nil)
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
			tags := mediaObjectToTags(tt.media, nil)
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
			tags := eventToTags(tt.event, tt.level, nil)
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
			tags := attributeToTags(tt.attr, tt.level, nil)
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
		{
			name: "citation with Ancestry APID",
			cite: &gedcom.SourceCitation{
				SourceXRef: "@S1@",
				AncestryAPID: &gedcom.AncestryAPID{
					Raw:      "1,7602::2771226",
					Database: "7602",
					Record:   "2771226",
				},
			},
			level:    2,
			contains: []string{"SOUR", "_APID"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := sourceCitationToTags(tt.cite, tt.level, nil)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("sourceCitationToTags() missing expected tag %q", expected)
				}
			}
		})
	}
}

// TestAncestryAPIDEncoding tests that Ancestry APID is correctly encoded
func TestAncestryAPIDEncoding(t *testing.T) {
	tests := []struct {
		name          string
		cite          *gedcom.SourceCitation
		expectAPID    bool
		expectedValue string
	}{
		{
			name: "citation with APID",
			cite: &gedcom.SourceCitation{
				SourceXRef: "@S1@",
				AncestryAPID: &gedcom.AncestryAPID{
					Raw:      "1,7602::2771226",
					Database: "7602",
					Record:   "2771226",
				},
			},
			expectAPID:    true,
			expectedValue: "1,7602::2771226",
		},
		{
			name: "citation without APID",
			cite: &gedcom.SourceCitation{
				SourceXRef: "@S1@",
			},
			expectAPID: false,
		},
		{
			name: "citation with nil APID",
			cite: &gedcom.SourceCitation{
				SourceXRef:   "@S1@",
				AncestryAPID: nil,
			},
			expectAPID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := sourceCitationToTags(tt.cite, 2, nil)

			// Find _APID tag
			var apidTag *gedcom.Tag
			for _, tag := range tags {
				if tag.Tag == "_APID" {
					apidTag = tag
					break
				}
			}

			if tt.expectAPID {
				if apidTag == nil {
					t.Error("expected _APID tag but not found")
					return
				}
				if apidTag.Value != tt.expectedValue {
					t.Errorf("_APID value = %q, want %q", apidTag.Value, tt.expectedValue)
				}
				if apidTag.Level != 3 { // level 2 (SOUR) + 1
					t.Errorf("_APID level = %d, want 3", apidTag.Level)
				}
			} else if apidTag != nil {
				t.Errorf("expected no _APID tag but found one with value %q", apidTag.Value)
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
			tags := associationToTags(tt.assoc, tt.level, nil)
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
			tags := sourceCitationDataToTags(tt.data, tt.level, nil)
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
			tags := entityToTags(tt.record, nil)
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

// TestTextToTags tests the textToTags helper function for CONT continuation handling
func TestTextToTags(t *testing.T) {
	tests := []struct {
		name          string
		value         string
		level         int
		tagName       string
		expectedTags  int
		expectedFirst string // Value of first tag
		expectedConts int    // Number of CONT tags expected
	}{
		{
			name:          "empty value",
			value:         "",
			level:         1,
			tagName:       "NOTE",
			expectedTags:  1,
			expectedFirst: "",
			expectedConts: 0,
		},
		{
			name:          "single line without newline",
			value:         "This is a simple note",
			level:         1,
			tagName:       "NOTE",
			expectedTags:  1,
			expectedFirst: "This is a simple note",
			expectedConts: 0,
		},
		{
			name:          "two lines",
			value:         "Line 1\nLine 2",
			level:         1,
			tagName:       "NOTE",
			expectedTags:  2,
			expectedFirst: "Line 1",
			expectedConts: 1,
		},
		{
			name:          "three lines",
			value:         "Line 1\nLine 2\nLine 3",
			level:         1,
			tagName:       "NOTE",
			expectedTags:  3,
			expectedFirst: "Line 1",
			expectedConts: 2,
		},
		{
			name:          "trailing newline",
			value:         "Line 1\nLine 2\n",
			level:         1,
			tagName:       "NOTE",
			expectedTags:  3,
			expectedFirst: "Line 1",
			expectedConts: 2, // includes empty line from trailing newline
		},
		{
			name:          "leading newline",
			value:         "\nLine 2",
			level:         1,
			tagName:       "NOTE",
			expectedTags:  2,
			expectedFirst: "",
			expectedConts: 1,
		},
		{
			name:          "multiple empty lines",
			value:         "Line 1\n\n\nLine 4",
			level:         1,
			tagName:       "NOTE",
			expectedTags:  4,
			expectedFirst: "Line 1",
			expectedConts: 3,
		},
		{
			name:          "TEXT tag at level 2",
			value:         "Source text\nwith continuation",
			level:         2,
			tagName:       "TEXT",
			expectedTags:  2,
			expectedFirst: "Source text",
			expectedConts: 1,
		},
		{
			name:          "level 3 with multiple lines",
			value:         "A\nB\nC",
			level:         3,
			tagName:       "NOTE",
			expectedTags:  3,
			expectedFirst: "A",
			expectedConts: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := textToTags(tt.value, tt.level, tt.tagName, nil)

			// Check total number of tags
			if len(tags) != tt.expectedTags {
				t.Errorf("textToTags() returned %d tags, want %d", len(tags), tt.expectedTags)
			}

			// Check first tag
			if len(tags) > 0 {
				if tags[0].Tag != tt.tagName {
					t.Errorf("first tag = %q, want %q", tags[0].Tag, tt.tagName)
				}
				if tags[0].Level != tt.level {
					t.Errorf("first tag level = %d, want %d", tags[0].Level, tt.level)
				}
				if tags[0].Value != tt.expectedFirst {
					t.Errorf("first tag value = %q, want %q", tags[0].Value, tt.expectedFirst)
				}
			}

			// Count CONT tags and verify their levels
			contCount := 0
			for i := 1; i < len(tags); i++ {
				if tags[i].Tag == "CONT" {
					contCount++
					// CONT tags should be at level+1
					if tags[i].Level != tt.level+1 {
						t.Errorf("CONT tag at index %d has level %d, want %d", i, tags[i].Level, tt.level+1)
					}
				}
			}
			if contCount != tt.expectedConts {
				t.Errorf("found %d CONT tags, want %d", contCount, tt.expectedConts)
			}
		})
	}
}

// TestTextToTagsValues verifies the exact values of generated tags
func TestTextToTagsValues(t *testing.T) {
	tests := []struct {
		name           string
		value          string
		expectedValues []string
	}{
		{
			name:           "simple multiline",
			value:          "First\nSecond\nThird",
			expectedValues: []string{"First", "Second", "Third"},
		},
		{
			name:           "with empty middle line",
			value:          "Start\n\nEnd",
			expectedValues: []string{"Start", "", "End"},
		},
		{
			name:           "only newlines",
			value:          "\n\n",
			expectedValues: []string{"", "", ""},
		},
		{
			name:           "unicode content",
			value:          "日本語\n中文\nHello",
			expectedValues: []string{"日本語", "中文", "Hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := textToTags(tt.value, 1, "NOTE", nil)

			if len(tags) != len(tt.expectedValues) {
				t.Fatalf("got %d tags, want %d", len(tags), len(tt.expectedValues))
			}

			for i, expectedValue := range tt.expectedValues {
				if tags[i].Value != expectedValue {
					t.Errorf("tag[%d].Value = %q, want %q", i, tags[i].Value, expectedValue)
				}
			}
		})
	}
}

// TestMultilineNoteEncoding tests that multiline notes are encoded correctly
func TestMultilineNoteEncoding(t *testing.T) {
	// Test with Individual
	t.Run("individual with multiline note", func(t *testing.T) {
		indi := &gedcom.Individual{
			Notes: []string{"Line 1\nLine 2\nLine 3"},
		}

		tags := individualToTags(indi, nil)
		tagMap := tagNamesToMap(tags)

		if !tagMap["NOTE"] {
			t.Error("missing NOTE tag")
		}
		if !tagMap["CONT"] {
			t.Error("missing CONT tag for multiline note")
		}

		// Verify structure: NOTE at level 1, CONT at level 2
		noteFound := false
		contCount := 0
		for _, tag := range tags {
			if tag.Tag == "NOTE" && tag.Level == 1 {
				noteFound = true
			}
			if tag.Tag == "CONT" && tag.Level == 2 {
				contCount++
			}
		}
		if !noteFound {
			t.Error("NOTE tag not at level 1")
		}
		if contCount != 2 {
			t.Errorf("expected 2 CONT tags at level 2, got %d", contCount)
		}
	})

	// Test with Family
	t.Run("family with multiline note", func(t *testing.T) {
		fam := &gedcom.Family{
			Notes: []string{"Family note\nwith continuation"},
		}

		tags := familyToTags(fam, nil)
		tagMap := tagNamesToMap(tags)

		if !tagMap["NOTE"] {
			t.Error("missing NOTE tag")
		}
		if !tagMap["CONT"] {
			t.Error("missing CONT tag for multiline note")
		}
	})

	// Test with Source (TEXT field)
	t.Run("source with multiline text", func(t *testing.T) {
		src := &gedcom.Source{
			Text: "Source text line 1\nSource text line 2",
		}

		tags := sourceToTags(src, nil)
		tagMap := tagNamesToMap(tags)

		if !tagMap["TEXT"] {
			t.Error("missing TEXT tag")
		}
		if !tagMap["CONT"] {
			t.Error("missing CONT tag for multiline text")
		}
	})

	// Test with Event notes
	t.Run("event with multiline note", func(t *testing.T) {
		event := &gedcom.Event{
			Type:  gedcom.EventBirth,
			Notes: []string{"Event note\nwith details"},
		}

		tags := eventToTags(event, 1, nil)
		tagMap := tagNamesToMap(tags)

		if !tagMap["NOTE"] {
			t.Error("missing NOTE tag")
		}
		if !tagMap["CONT"] {
			t.Error("missing CONT tag for multiline event note")
		}

		// Verify NOTE at level 2 (event at level 1, note subordinate)
		// and CONT at level 3
		for _, tag := range tags {
			if tag.Tag == "NOTE" && tag.Level != 2 {
				t.Errorf("event NOTE should be at level 2, got %d", tag.Level)
			}
			if tag.Tag == "CONT" && tag.Level != 3 {
				t.Errorf("event CONT should be at level 3, got %d", tag.Level)
			}
		}
	})

	// Test SourceCitationData TEXT
	t.Run("source citation data with multiline text", func(t *testing.T) {
		data := &gedcom.SourceCitationData{
			Text: "Citation text\nwith more info",
		}

		tags := sourceCitationDataToTags(data, 3, nil)
		tagMap := tagNamesToMap(tags)

		if !tagMap["TEXT"] {
			t.Error("missing TEXT tag")
		}
		if !tagMap["CONT"] {
			t.Error("missing CONT tag for multiline citation text")
		}

		// TEXT at level 4, CONT at level 5
		for _, tag := range tags {
			if tag.Tag == "TEXT" && tag.Level != 4 {
				t.Errorf("citation TEXT should be at level 4, got %d", tag.Level)
			}
			if tag.Tag == "CONT" && tag.Level != 5 {
				t.Errorf("citation CONT should be at level 5, got %d", tag.Level)
			}
		}
	})

	// Test Association notes
	t.Run("association with multiline note", func(t *testing.T) {
		assoc := &gedcom.Association{
			IndividualXRef: "@I2@",
			Notes:          []string{"Association note\nwith continuation"},
		}

		tags := associationToTags(assoc, 1, nil)
		tagMap := tagNamesToMap(tags)

		if !tagMap["NOTE"] {
			t.Error("missing NOTE tag")
		}
		if !tagMap["CONT"] {
			t.Error("missing CONT tag for multiline association note")
		}
	})

	// Test Submitter notes
	t.Run("submitter with multiline note", func(t *testing.T) {
		subm := &gedcom.Submitter{
			Name:  "Test Submitter",
			Notes: []string{"Submitter note\nwith continuation"},
		}

		tags := submitterToTags(subm, nil)
		tagMap := tagNamesToMap(tags)

		if !tagMap["NOTE"] {
			t.Error("missing NOTE tag")
		}
		if !tagMap["CONT"] {
			t.Error("missing CONT tag for multiline submitter note")
		}
	})

	// Test Repository notes
	t.Run("repository with multiline note", func(t *testing.T) {
		repo := &gedcom.Repository{
			Name:  "Test Repository",
			Notes: []string{"Repository note\nwith continuation"},
		}

		tags := repositoryToTags(repo, nil)
		tagMap := tagNamesToMap(tags)

		if !tagMap["NOTE"] {
			t.Error("missing NOTE tag")
		}
		if !tagMap["CONT"] {
			t.Error("missing CONT tag for multiline repository note")
		}
	})

	// Test MediaObject notes
	t.Run("media object with multiline note", func(t *testing.T) {
		media := &gedcom.MediaObject{
			Files: []*gedcom.MediaFile{{FileRef: "test.jpg"}},
			Notes: []string{"Media note\nwith continuation"},
		}

		tags := mediaObjectToTags(media, nil)
		tagMap := tagNamesToMap(tags)

		if !tagMap["NOTE"] {
			t.Error("missing NOTE tag")
		}
		if !tagMap["CONT"] {
			t.Error("missing CONT tag for multiline media note")
		}
	})
}

// TestSingleLineNoteNoConts verifies single line notes don't generate CONT tags
func TestSingleLineNoteNoConts(t *testing.T) {
	indi := &gedcom.Individual{
		Notes: []string{"Single line note without newlines"},
	}

	tags := individualToTags(indi, nil)

	for _, tag := range tags {
		if tag.Tag == "CONT" {
			t.Error("single line note should not generate CONT tags")
		}
	}
}

// TestSplitLineForLength tests the line splitting helper function
func TestSplitLineForLength(t *testing.T) {
	tests := []struct {
		name          string
		line          string
		opts          *EncodeOptions
		expectedCount int
		expectedFirst string
		expectedLast  string
	}{
		{
			name:          "short line - no split",
			line:          "Short line",
			opts:          DefaultOptions(),
			expectedCount: 1,
			expectedFirst: "Short line",
			expectedLast:  "Short line",
		},
		{
			name:          "exactly at max length - no split",
			line:          string(make([]byte, 248)),
			opts:          DefaultOptions(),
			expectedCount: 1,
			expectedFirst: string(make([]byte, 248)),
			expectedLast:  string(make([]byte, 248)),
		},
		{
			name:          "nil opts uses default",
			line:          "Short line",
			opts:          nil,
			expectedCount: 1,
			expectedFirst: "Short line",
			expectedLast:  "Short line",
		},
		{
			name:          "disabled line wrap",
			line:          "This is a very long line that exceeds the maximum length but should not be split because DisableLineWrap is true. " + string(make([]byte, 200)),
			opts:          &EncodeOptions{DisableLineWrap: true},
			expectedCount: 1,
		},
		{
			name:          "custom max length",
			line:          "This is a test line with more than 20 chars",
			opts:          &EncodeOptions{MaxLineLength: 20},
			expectedCount: 3, // 20, 20, rest
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments := splitLineForLength(tt.line, tt.opts)

			if len(segments) != tt.expectedCount {
				t.Errorf("splitLineForLength() returned %d segments, want %d", len(segments), tt.expectedCount)
			}

			if tt.expectedFirst != "" && len(segments) > 0 && segments[0] != tt.expectedFirst {
				t.Errorf("first segment = %q, want %q", segments[0], tt.expectedFirst)
			}

			if tt.expectedLast != "" && len(segments) > 0 && segments[len(segments)-1] != tt.expectedLast {
				t.Errorf("last segment = %q, want %q", segments[len(segments)-1], tt.expectedLast)
			}
		})
	}
}

// TestFindWordBoundary tests the word boundary finding helper
func TestFindWordBoundary(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		maxLen   int
		expected int
	}{
		{
			name:     "line shorter than max",
			line:     "Short",
			maxLen:   10,
			expected: 5,
		},
		{
			name:     "split at space",
			line:     "Hello world this is a test",
			maxLen:   15,
			expected: 12, // After "Hello world "
		},
		{
			name:     "no space found - split at max",
			line:     "Supercalifragilisticexpialidocious",
			maxLen:   10,
			expected: 10,
		},
		{
			name:     "space exactly at max",
			line:     "12345678 x",
			maxLen:   9,
			expected: 9, // After "12345678 "
		},
		{
			name:     "multiple spaces - uses last one within limit",
			line:     "one two three four",
			maxLen:   14,
			expected: 14, // After "one two three " (index 14 is right after the space)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findWordBoundary(tt.line, tt.maxLen)
			if result != tt.expected {
				t.Errorf("findWordBoundary() = %d, want %d", result, tt.expected)
			}
		})
	}
}

// TestTextToTagsWithCONC tests CONC tag generation for long lines
func TestTextToTagsWithCONC(t *testing.T) {
	// Create a line exactly 300 chars (well over default 248)
	longLine := "This is a very long note that exceeds the recommended line length. "
	for len(longLine) < 300 {
		longLine += "More text here. "
	}
	longLine = longLine[:300] // Trim to exactly 300

	t.Run("long line splits into CONC tags", func(t *testing.T) {
		opts := DefaultOptions()
		tags := textToTags(longLine, 1, "NOTE", opts)

		// Should have primary tag + at least one CONC
		if len(tags) < 2 {
			t.Errorf("expected at least 2 tags for 300 char line, got %d", len(tags))
		}

		// First tag should be NOTE at level 1
		if tags[0].Tag != "NOTE" || tags[0].Level != 1 {
			t.Errorf("first tag should be NOTE at level 1, got %s at %d", tags[0].Tag, tags[0].Level)
		}

		// Remaining tags should be CONC at level 2
		for i := 1; i < len(tags); i++ {
			if tags[i].Tag != "CONC" {
				t.Errorf("tag %d should be CONC, got %s", i, tags[i].Tag)
			}
			if tags[i].Level != 2 {
				t.Errorf("tag %d level should be 2, got %d", i, tags[i].Level)
			}
		}

		// Reconstruct and verify content matches
		var reconstructed string
		for _, tag := range tags {
			reconstructed += tag.Value
		}
		if reconstructed != longLine {
			t.Errorf("reconstructed content does not match original")
		}
	})

	t.Run("disabled line wrap produces single tag", func(t *testing.T) {
		opts := &EncodeOptions{DisableLineWrap: true}
		tags := textToTags(longLine, 1, "NOTE", opts)

		if len(tags) != 1 {
			t.Errorf("with DisableLineWrap, expected 1 tag, got %d", len(tags))
		}
		if tags[0].Value != longLine {
			t.Errorf("value should be unchanged when DisableLineWrap is true")
		}
	})

	t.Run("custom max length", func(t *testing.T) {
		opts := &EncodeOptions{MaxLineLength: 50}
		tags := textToTags("This is a test line that exceeds fifty characters by quite a bit", 1, "NOTE", opts)

		// Should split into multiple segments
		if len(tags) < 2 {
			t.Errorf("expected at least 2 tags with MaxLineLength=50, got %d", len(tags))
		}

		// First segment should be <= 50 chars
		if len(tags[0].Value) > 50 {
			t.Errorf("first segment length %d exceeds MaxLineLength 50", len(tags[0].Value))
		}
	})

	t.Run("multiline with long line produces CONT and CONC", func(t *testing.T) {
		// First line is short, second line is long
		shortLine := "Short first line"
		longSecondLine := longLine

		opts := DefaultOptions()
		tags := textToTags(shortLine+"\n"+longSecondLine, 1, "NOTE", opts)

		// Should have: NOTE (short), CONT (first part of long), CONC (rest of long)
		hasNote := false
		hasCont := false
		hasConc := false

		for _, tag := range tags {
			switch tag.Tag {
			case "NOTE":
				hasNote = true
				if tag.Value != shortLine {
					t.Errorf("NOTE value should be %q, got %q", shortLine, tag.Value)
				}
			case "CONT":
				hasCont = true
			case "CONC":
				hasConc = true
			}
		}

		if !hasNote {
			t.Error("missing NOTE tag")
		}
		if !hasCont {
			t.Error("missing CONT tag for newline")
		}
		if !hasConc {
			t.Error("missing CONC tag for long line split")
		}
	})
}

// TestCONCPreservesContent verifies that CONC splitting preserves all content
func TestCONCPreservesContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		maxLen  int
	}{
		{
			name:    "simple long text",
			content: "This is a very long text that needs to be split. It contains multiple words and should be split at word boundaries when possible. The reconstructed content should match exactly.",
			maxLen:  50,
		},
		{
			name:    "text without spaces",
			content: "NoSpacesInThisTextSoItMustBeSplitAtExactMaxLengthPositionWithoutWordBoundarySupport",
			maxLen:  20,
		},
		{
			name:    "unicode content",
			content: "日本語テスト文字列です。This text contains Japanese characters and should be handled correctly.",
			maxLen:  30,
		},
		{
			name:    "multiline with long lines",
			content: "First line is short\nSecond line is much longer and will need to be split into multiple CONC segments for proper encoding.\nThird line",
			maxLen:  40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &EncodeOptions{MaxLineLength: tt.maxLen}
			tags := textToTags(tt.content, 1, "NOTE", opts)

			// Reconstruct content from tags
			var reconstructed string
			for i, tag := range tags {
				if i > 0 && tag.Tag == "CONT" {
					reconstructed += "\n"
				}
				reconstructed += tag.Value
			}

			if reconstructed != tt.content {
				t.Errorf("content mismatch:\ngot:  %q\nwant: %q", reconstructed, tt.content)
			}
		})
	}
}

// TestEncodeOptionsDefaults verifies default options are set correctly
func TestEncodeOptionsDefaults(t *testing.T) {
	opts := DefaultOptions()

	if opts.MaxLineLength != DefaultMaxLineLength {
		t.Errorf("MaxLineLength = %d, want %d", opts.MaxLineLength, DefaultMaxLineLength)
	}

	if opts.DisableLineWrap != false {
		t.Error("DisableLineWrap should default to false")
	}

	if opts.LineEnding != "\n" {
		t.Errorf("LineEnding = %q, want %q", opts.LineEnding, "\n")
	}
}

// TestEffectiveMaxLineLength tests the helper method
func TestEffectiveMaxLineLength(t *testing.T) {
	tests := []struct {
		name     string
		opts     *EncodeOptions
		expected int
	}{
		{
			name:     "nil opts returns default",
			opts:     nil,
			expected: DefaultMaxLineLength,
		},
		{
			name:     "zero MaxLineLength returns default",
			opts:     &EncodeOptions{MaxLineLength: 0},
			expected: DefaultMaxLineLength,
		},
		{
			name:     "negative MaxLineLength returns default",
			opts:     &EncodeOptions{MaxLineLength: -10},
			expected: DefaultMaxLineLength,
		},
		{
			name:     "positive MaxLineLength returns value",
			opts:     &EncodeOptions{MaxLineLength: 100},
			expected: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.opts.effectiveMaxLineLength()
			if result != tt.expected {
				t.Errorf("effectiveMaxLineLength() = %d, want %d", result, tt.expected)
			}
		})
	}
}

// TestFamilySearchIDEncoding tests encoding of the _FSFTID tag (FamilySearch Family Tree ID).
// This is a vendor extension from FamilySearch.org.
// Ref: Issue #80
func TestFamilySearchIDEncoding(t *testing.T) {
	tests := []struct {
		name          string
		indi          *gedcom.Individual
		expectFSFTID  bool
		expectedValue string
	}{
		{
			name: "individual with FamilySearchID",
			indi: &gedcom.Individual{
				XRef:           "@I1@",
				FamilySearchID: "KWCJ-QN7",
			},
			expectFSFTID:  true,
			expectedValue: "KWCJ-QN7",
		},
		{
			name: "individual without FamilySearchID",
			indi: &gedcom.Individual{
				XRef: "@I2@",
			},
			expectFSFTID: false,
		},
		{
			name: "individual with empty FamilySearchID",
			indi: &gedcom.Individual{
				XRef:           "@I3@",
				FamilySearchID: "",
			},
			expectFSFTID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := individualToTags(tt.indi, nil)

			// Find _FSFTID tag
			var fsftidTag *gedcom.Tag
			for _, tag := range tags {
				if tag.Tag == "_FSFTID" {
					fsftidTag = tag
					break
				}
			}

			if tt.expectFSFTID {
				if fsftidTag == nil {
					t.Error("expected _FSFTID tag but not found")
					return
				}
				if fsftidTag.Value != tt.expectedValue {
					t.Errorf("_FSFTID value = %q, want %q", fsftidTag.Value, tt.expectedValue)
				}
				if fsftidTag.Level != 1 {
					t.Errorf("_FSFTID level = %d, want 1", fsftidTag.Level)
				}
			} else if fsftidTag != nil {
				t.Errorf("expected no _FSFTID tag but found one with value %q", fsftidTag.Value)
			}
		})
	}
}

// === GEDCOM 7.0 ASSO/PHRASE Encoder Tests ===
// These tests validate encoding of GEDCOM 7.0 association features including
// PHRASE subordinates for human-readable descriptions and SOUR citations.
// Ref: Issues #40, #39

// TestAssociationToTagsWithPhrase tests encoding ASSO with PHRASE subordinate.
func TestAssociationToTagsWithPhrase(t *testing.T) {
	tests := []struct {
		name     string
		assoc    *gedcom.Association
		level    int
		contains []string
	}{
		{
			name: "association with phrase only",
			assoc: &gedcom.Association{
				IndividualXRef: "@I2@",
				Phrase:         "Godparent at baptism",
			},
			level:    1,
			contains: []string{"ASSO", "PHRASE"},
		},
		{
			name: "association with phrase and role",
			assoc: &gedcom.Association{
				IndividualXRef: "@I2@",
				Phrase:         "Godparent at baptism",
				Role:           "GODP",
			},
			level:    1,
			contains: []string{"ASSO", "PHRASE", "ROLE"},
		},
		{
			name: "association with source citations",
			assoc: &gedcom.Association{
				IndividualXRef: "@I2@",
				Role:           "WITN",
				SourceCitations: []*gedcom.SourceCitation{
					{SourceXRef: "@S1@", Page: "Page 123"},
				},
			},
			level:    1,
			contains: []string{"ASSO", "ROLE", "SOUR", "PAGE"},
		},
		{
			name: "association with phrase, source and notes",
			assoc: &gedcom.Association{
				IndividualXRef: "@I3@",
				Phrase:         "Association text",
				Role:           "OTHER",
				Notes:          []string{"Note text"},
				SourceCitations: []*gedcom.SourceCitation{
					{SourceXRef: "@S1@", Page: "1"},
					{SourceXRef: "@S2@", Page: "2"},
				},
			},
			level:    1,
			contains: []string{"ASSO", "PHRASE", "ROLE", "NOTE", "SOUR", "PAGE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := associationToTags(tt.assoc, tt.level, nil)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("associationToTags() missing expected tag %q", expected)
				}
			}

			// Verify ASSO tag has correct level and value
			if len(tags) > 0 {
				if tags[0].Tag != "ASSO" {
					t.Errorf("First tag = %s, want ASSO", tags[0].Tag)
				}
				if tags[0].Level != tt.level {
					t.Errorf("ASSO level = %d, want %d", tags[0].Level, tt.level)
				}
				if tags[0].Value != tt.assoc.IndividualXRef {
					t.Errorf("ASSO value = %s, want %s", tags[0].Value, tt.assoc.IndividualXRef)
				}
			}

			// Verify PHRASE tag value if present
			if tt.assoc.Phrase != "" {
				var phraseTag *gedcom.Tag
				for _, tag := range tags {
					if tag.Tag == "PHRASE" {
						phraseTag = tag
						break
					}
				}
				if phraseTag == nil {
					t.Error("expected PHRASE tag but not found")
				} else if phraseTag.Value != tt.assoc.Phrase {
					t.Errorf("PHRASE value = %q, want %q", phraseTag.Value, tt.assoc.Phrase)
				}
			}
		})
	}
}

// TestAssociationToTagsSourceCitationCount tests correct encoding of multiple source citations.
func TestAssociationToTagsSourceCitationCount(t *testing.T) {
	assoc := &gedcom.Association{
		IndividualXRef: "@I2@",
		Role:           "GODP",
		SourceCitations: []*gedcom.SourceCitation{
			{SourceXRef: "@S1@", Page: "1"},
			{SourceXRef: "@S2@", Page: "2"},
			{SourceXRef: "@S3@", Page: "3"},
		},
	}

	tags := associationToTags(assoc, 1, nil)

	// Count SOUR tags
	sourCount := 0
	for _, tag := range tags {
		if tag.Tag == "SOUR" {
			sourCount++
		}
	}

	if sourCount != 3 {
		t.Errorf("Expected 3 SOUR tags, got %d", sourCount)
	}
}

// === GEDCOM 7.0 NAME TRAN (Transliteration) Encoder Tests ===
// These tests validate encoding of GEDCOM 7.0 name transliteration features.
// Ref: Issue #39

// TestNameToTagsWithTransliterations tests encoding NAME with TRAN subordinates.
func TestNameToTagsWithTransliterations(t *testing.T) {
	tests := []struct {
		name     string
		pname    *gedcom.PersonalName
		level    int
		contains []string
	}{
		{
			name: "name with single transliteration",
			pname: &gedcom.PersonalName{
				Full:    "John /Doe/",
				Given:   "John",
				Surname: "Doe",
				Transliterations: []*gedcom.Transliteration{
					{
						Value:    "John /Doe/",
						Language: "en-GB",
					},
				},
			},
			level:    1,
			contains: []string{"NAME", "GIVN", "SURN", "TRAN", "LANG"},
		},
		{
			name: "name with multiple transliterations",
			pname: &gedcom.PersonalName{
				Full:    "John /Doe/",
				Given:   "John",
				Surname: "Doe",
				Transliterations: []*gedcom.Transliteration{
					{
						Value:    "John /Doe/",
						Language: "en-GB",
					},
					{
						Value:    "John /Doe/",
						Language: "en-CA",
					},
				},
			},
			level:    1,
			contains: []string{"NAME", "GIVN", "SURN", "TRAN", "LANG"},
		},
		{
			name: "name with transliteration and all components",
			pname: &gedcom.PersonalName{
				Full:          "Lt. Cmndr. Joseph /Allen/ jr.",
				Given:         "Joseph",
				Surname:       "Allen",
				Prefix:        "Lt. Cmndr.",
				Suffix:        "jr.",
				SurnamePrefix: "de",
				Transliterations: []*gedcom.Transliteration{
					{
						Value:         "npfx John /spfx Doe/ nsfx",
						Language:      "en-GB",
						Prefix:        "npfx",
						Given:         "John",
						Surname:       "Doe",
						Suffix:        "nsfx",
						SurnamePrefix: "spfx",
						Nickname:      "Johnny",
					},
				},
			},
			level:    1,
			contains: []string{"NAME", "GIVN", "SURN", "NPFX", "NSFX", "SPFX", "TRAN", "LANG"},
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
		})
	}
}

// TestTransliterationToTags tests encoding of individual transliteration records.
func TestTransliterationToTags(t *testing.T) {
	tests := []struct {
		name     string
		tran     *gedcom.Transliteration
		level    int
		contains []string
	}{
		{
			name: "minimal transliteration",
			tran: &gedcom.Transliteration{
				Value: "John /Doe/",
			},
			level:    2,
			contains: []string{"TRAN"},
		},
		{
			name: "transliteration with language",
			tran: &gedcom.Transliteration{
				Value:    "John /Doe/",
				Language: "en-GB",
			},
			level:    2,
			contains: []string{"TRAN", "LANG"},
		},
		{
			name: "transliteration with all components",
			tran: &gedcom.Transliteration{
				Value:         "npfx John /spfx Doe/ nsfx",
				Language:      "en-GB",
				Prefix:        "npfx",
				Given:         "John",
				Surname:       "Doe",
				Suffix:        "nsfx",
				SurnamePrefix: "spfx",
				Nickname:      "Johnny",
			},
			level:    2,
			contains: []string{"TRAN", "LANG", "NPFX", "GIVN", "SURN", "NSFX", "SPFX", "NICK"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := transliterationToTags(tt.tran, tt.level)
			tagMap := tagNamesToMap(tags)

			for _, expected := range tt.contains {
				if !tagMap[expected] {
					t.Errorf("transliterationToTags() missing expected tag %q", expected)
				}
			}

			// Verify TRAN tag has correct level and value
			if len(tags) > 0 {
				if tags[0].Tag != "TRAN" {
					t.Errorf("First tag = %s, want TRAN", tags[0].Tag)
				}
				if tags[0].Level != tt.level {
					t.Errorf("TRAN level = %d, want %d", tags[0].Level, tt.level)
				}
				if tags[0].Value != tt.tran.Value {
					t.Errorf("TRAN value = %q, want %q", tags[0].Value, tt.tran.Value)
				}
			}
		})
	}
}

// TestTransliterationToTagsSubordinateLevels tests that subordinate tags have correct levels.
func TestTransliterationToTagsSubordinateLevels(t *testing.T) {
	tran := &gedcom.Transliteration{
		Value:    "John /Doe/",
		Language: "en-GB",
		Given:    "John",
		Surname:  "Doe",
	}

	tags := transliterationToTags(tran, 2)

	// TRAN should be at level 2
	if tags[0].Level != 2 {
		t.Errorf("TRAN level = %d, want 2", tags[0].Level)
	}

	// All subordinates should be at level 3
	for _, tag := range tags[1:] {
		if tag.Level != 3 {
			t.Errorf("%s level = %d, want 3", tag.Tag, tag.Level)
		}
	}
}

// TestNameToTagsTransliterationCount tests correct encoding of multiple transliterations.
func TestNameToTagsTransliterationCount(t *testing.T) {
	pname := &gedcom.PersonalName{
		Full:    "John /Doe/",
		Given:   "John",
		Surname: "Doe",
		Transliterations: []*gedcom.Transliteration{
			{Value: "John /Doe/", Language: "en-GB"},
			{Value: "John /Doe/", Language: "en-CA"},
			{Value: "John /Doe/", Language: "en-AU"},
		},
	}

	tags := nameToTags(pname, 1)

	// Count TRAN tags
	tranCount := 0
	for _, tag := range tags {
		if tag.Tag == "TRAN" {
			tranCount++
		}
	}

	if tranCount != 3 {
		t.Errorf("Expected 3 TRAN tags, got %d", tranCount)
	}
}

// TestRoundTripAssociationWithPhrase tests decode -> encode consistency for ASSO with PHRASE.
func TestRoundTripAssociationWithPhrase(t *testing.T) {
	original := `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME John /Doe/
1 ASSO @I2@
2 PHRASE Godparent at baptism
2 ROLE GODP
2 SOUR @S1@
3 PAGE Page 123
0 @I2@ INDI
1 NAME Jane /Smith/
0 @S1@ SOUR
1 TITL Baptism Registry
0 TRLR
`
	doc, err := decoder.Decode(strings.NewReader(original))
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Re-encode the document
	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Re-decode the encoded document
	doc2, err := decoder.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Re-decode failed: %v", err)
	}

	// Verify the association is preserved
	indi := doc2.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual @I1@ not found after round-trip")
	}

	if len(indi.Associations) != 1 {
		t.Fatalf("len(Associations) = %d, want 1", len(indi.Associations))
	}

	assoc := indi.Associations[0]
	if assoc.IndividualXRef != "@I2@" {
		t.Errorf("IndividualXRef = %s, want @I2@", assoc.IndividualXRef)
	}
	if assoc.Phrase != "Godparent at baptism" {
		t.Errorf("Phrase = %s, want 'Godparent at baptism'", assoc.Phrase)
	}
	if assoc.Role != "GODP" {
		t.Errorf("Role = %s, want GODP", assoc.Role)
	}
	if len(assoc.SourceCitations) != 1 {
		t.Fatalf("len(SourceCitations) = %d, want 1", len(assoc.SourceCitations))
	}
}

// TestRoundTripNameWithTransliteration tests decode -> encode consistency for NAME with TRAN.
func TestRoundTripNameWithTransliteration(t *testing.T) {
	original := `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME John /Doe/
2 GIVN John
2 SURN Doe
2 TRAN John /Doe/
3 LANG en-GB
3 GIVN John
3 SURN Doe
2 TRAN John /Doe/
3 LANG en-CA
0 TRLR
`
	doc, err := decoder.Decode(strings.NewReader(original))
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Re-encode the document
	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Re-decode the encoded document
	doc2, err := decoder.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Re-decode failed: %v", err)
	}

	// Verify the transliterations are preserved
	indi := doc2.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual @I1@ not found after round-trip")
	}

	if len(indi.Names) != 1 {
		t.Fatalf("len(Names) = %d, want 1", len(indi.Names))
	}

	name := indi.Names[0]
	if len(name.Transliterations) != 2 {
		t.Fatalf("len(Transliterations) = %d, want 2", len(name.Transliterations))
	}

	// Verify first transliteration
	tran1 := name.Transliterations[0]
	if tran1.Language != "en-GB" {
		t.Errorf("Transliteration[0].Language = %s, want 'en-GB'", tran1.Language)
	}
	if tran1.Given != "John" {
		t.Errorf("Transliteration[0].Given = %s, want 'John'", tran1.Given)
	}
	if tran1.Surname != "Doe" {
		t.Errorf("Transliteration[0].Surname = %s, want 'Doe'", tran1.Surname)
	}

	// Verify second transliteration
	tran2 := name.Transliterations[1]
	if tran2.Language != "en-CA" {
		t.Errorf("Transliteration[1].Language = %s, want 'en-CA'", tran2.Language)
	}
}
