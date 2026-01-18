package converter

import (
	"testing"
	"time"

	"github.com/cacack/gedcom-go/gedcom"
)

func TestDeepCopyDocument(t *testing.T) {
	t.Run("nil document returns nil", func(t *testing.T) {
		result := deepCopyDocument(nil)
		if result != nil {
			t.Error("deepCopyDocument(nil) should return nil")
		}
	})

	t.Run("creates independent copy", func(t *testing.T) {
		original := createFullTestDocument()
		copy := deepCopyDocument(original)

		if copy == nil {
			t.Fatal("deepCopyDocument() returned nil")
		}

		// Verify different pointers
		if copy == original {
			t.Error("Copy should have different pointer than original")
		}
		if copy.Header == original.Header {
			t.Error("Copy header should have different pointer")
		}
		if len(copy.Records) > 0 && len(original.Records) > 0 && copy.Records[0] == original.Records[0] {
			t.Error("Copy records should have different pointers")
		}

		// Verify values are equal
		if copy.Header.Version != original.Header.Version {
			t.Errorf("Copy version = %v, want %v", copy.Header.Version, original.Header.Version)
		}
		if copy.Vendor != original.Vendor {
			t.Errorf("Copy vendor = %v, want %v", copy.Vendor, original.Vendor)
		}
	})

	t.Run("modifications to copy don't affect original", func(t *testing.T) {
		original := createFullTestDocument()
		originalVersion := original.Header.Version
		originalXRef := original.Records[0].XRef

		copy := deepCopyDocument(original)

		// Modify the copy
		copy.Header.Version = gedcom.Version70
		copy.Records[0].XRef = "@MODIFIED@"

		// Original should be unchanged
		if original.Header.Version != originalVersion {
			t.Errorf("Original version was mutated: got %v, want %v", original.Header.Version, originalVersion)
		}
		if original.Records[0].XRef != originalXRef {
			t.Errorf("Original XRef was mutated: got %v, want %v", original.Records[0].XRef, originalXRef)
		}
	})

	t.Run("XRefMap keys are updated for new records", func(t *testing.T) {
		original := &gedcom.Document{
			Header: &gedcom.Header{Version: gedcom.Version55},
			Records: []*gedcom.Record{
				{XRef: "@I1@", Type: gedcom.RecordTypeIndividual},
				{XRef: "@F1@", Type: gedcom.RecordTypeFamily},
			},
			XRefMap: map[string]*gedcom.Record{},
		}
		original.XRefMap["@I1@"] = original.Records[0]
		original.XRefMap["@F1@"] = original.Records[1]

		copy := deepCopyDocument(original)

		// XRefMap should point to the copied records, not originals
		if copy.XRefMap["@I1@"] == original.XRefMap["@I1@"] {
			t.Error("XRefMap entries should point to copied records")
		}
		if copy.XRefMap["@I1@"] != copy.Records[0] {
			t.Error("XRefMap should point to the correct copied record")
		}
	})
}

func TestDeepCopyHeader(t *testing.T) {
	t.Run("nil header returns nil", func(t *testing.T) {
		result := deepCopyHeader(nil)
		if result != nil {
			t.Error("deepCopyHeader(nil) should return nil")
		}
	})

	t.Run("copies all header fields", func(t *testing.T) {
		original := &gedcom.Header{
			Version:        gedcom.Version551,
			Encoding:       gedcom.EncodingUTF8,
			SourceSystem:   "TestSystem",
			Date:           time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Language:       "English",
			Copyright:      "(c) 2024",
			Submitter:      "@SUBM1@",
			AncestryTreeID: "tree123",
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "TEST", Value: "value"},
			},
		}

		copy := deepCopyHeader(original)

		if copy == original {
			t.Error("Copy should have different pointer")
		}
		if copy.Version != original.Version {
			t.Errorf("Version = %v, want %v", copy.Version, original.Version)
		}
		if copy.Encoding != original.Encoding {
			t.Errorf("Encoding = %v, want %v", copy.Encoding, original.Encoding)
		}
		if copy.SourceSystem != original.SourceSystem {
			t.Errorf("SourceSystem = %v, want %v", copy.SourceSystem, original.SourceSystem)
		}
		if len(copy.Tags) != len(original.Tags) {
			t.Errorf("Tags length = %d, want %d", len(copy.Tags), len(original.Tags))
		}
		if len(copy.Tags) > 0 && copy.Tags[0] == original.Tags[0] {
			t.Error("Tags should be deep copied")
		}
	})
}

func TestDeepCopyTrailer(t *testing.T) {
	t.Run("nil trailer returns nil", func(t *testing.T) {
		result := deepCopyTrailer(nil)
		if result != nil {
			t.Error("deepCopyTrailer(nil) should return nil")
		}
	})

	t.Run("copies trailer fields", func(t *testing.T) {
		original := &gedcom.Trailer{LineNumber: 100}
		copy := deepCopyTrailer(original)

		if copy == original {
			t.Error("Copy should have different pointer")
		}
		if copy.LineNumber != original.LineNumber {
			t.Errorf("LineNumber = %d, want %d", copy.LineNumber, original.LineNumber)
		}
	})
}

func TestDeepCopyRecord(t *testing.T) {
	t.Run("nil record returns nil", func(t *testing.T) {
		result := deepCopyRecord(nil)
		if result != nil {
			t.Error("deepCopyRecord(nil) should return nil")
		}
	})

	t.Run("copies record fields", func(t *testing.T) {
		original := &gedcom.Record{
			XRef:       "@I1@",
			Type:       gedcom.RecordTypeIndividual,
			Value:      "test value",
			LineNumber: 10,
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: "John /Doe/"},
			},
		}

		copy := deepCopyRecord(original)

		if copy == original {
			t.Error("Copy should have different pointer")
		}
		if copy.XRef != original.XRef {
			t.Errorf("XRef = %v, want %v", copy.XRef, original.XRef)
		}
		if copy.Type != original.Type {
			t.Errorf("Type = %v, want %v", copy.Type, original.Type)
		}
		if copy.LineNumber != original.LineNumber {
			t.Errorf("LineNumber = %d, want %d", copy.LineNumber, original.LineNumber)
		}
		if len(copy.Tags) > 0 && copy.Tags[0] == original.Tags[0] {
			t.Error("Tags should be deep copied")
		}
	})
}

func TestDeepCopyTags(t *testing.T) {
	t.Run("nil tags returns nil", func(t *testing.T) {
		result := deepCopyTags(nil)
		if result != nil {
			t.Error("deepCopyTags(nil) should return nil")
		}
	})

	t.Run("copies tags deeply", func(t *testing.T) {
		original := []*gedcom.Tag{
			{Level: 0, Tag: "INDI"},
			{Level: 1, Tag: "NAME", Value: "John"},
		}

		copy := deepCopyTags(original)

		if len(copy) != len(original) {
			t.Errorf("Copy length = %d, want %d", len(copy), len(original))
		}
		for i := range copy {
			if copy[i] == original[i] {
				t.Errorf("Tag[%d] should have different pointer", i)
			}
			if copy[i].Tag != original[i].Tag {
				t.Errorf("Tag[%d].Tag = %v, want %v", i, copy[i].Tag, original[i].Tag)
			}
		}
	})
}

func TestDeepCopyTag(t *testing.T) {
	t.Run("nil tag returns nil", func(t *testing.T) {
		result := deepCopyTag(nil)
		if result != nil {
			t.Error("deepCopyTag(nil) should return nil")
		}
	})

	t.Run("copies all tag fields", func(t *testing.T) {
		original := &gedcom.Tag{
			Level:      2,
			Tag:        "NOTE",
			Value:      "Some note",
			XRef:       "@N1@",
			LineNumber: 42,
		}

		copy := deepCopyTag(original)

		if copy == original {
			t.Error("Copy should have different pointer")
		}
		if copy.Level != original.Level {
			t.Errorf("Level = %d, want %d", copy.Level, original.Level)
		}
		if copy.Tag != original.Tag {
			t.Errorf("Tag = %v, want %v", copy.Tag, original.Tag)
		}
		if copy.Value != original.Value {
			t.Errorf("Value = %v, want %v", copy.Value, original.Value)
		}
		if copy.XRef != original.XRef {
			t.Errorf("XRef = %v, want %v", copy.XRef, original.XRef)
		}
		if copy.LineNumber != original.LineNumber {
			t.Errorf("LineNumber = %d, want %d", copy.LineNumber, original.LineNumber)
		}
	})
}

func TestDeepCopyEntity(t *testing.T) {
	t.Run("nil entity returns nil", func(t *testing.T) {
		result := deepCopyEntity(nil)
		if result != nil {
			t.Error("deepCopyEntity(nil) should return nil")
		}
	})

	t.Run("copies Individual", func(t *testing.T) {
		original := &gedcom.Individual{
			XRef: "@I1@",
			Sex:  "M",
			Names: []*gedcom.PersonalName{
				{Full: "John /Doe/", Given: "John", Surname: "Doe"},
			},
		}

		result := deepCopyEntity(original)
		copy, ok := result.(*gedcom.Individual)

		if !ok {
			t.Fatal("Result should be *Individual")
		}
		if copy == original {
			t.Error("Copy should have different pointer")
		}
		if copy.XRef != original.XRef {
			t.Errorf("XRef = %v, want %v", copy.XRef, original.XRef)
		}
		if len(copy.Names) != len(original.Names) {
			t.Errorf("Names length = %d, want %d", len(copy.Names), len(original.Names))
		}
	})

	t.Run("copies Family", func(t *testing.T) {
		original := &gedcom.Family{
			XRef:     "@F1@",
			Husband:  "@I1@",
			Wife:     "@I2@",
			Children: []string{"@I3@", "@I4@"},
		}

		result := deepCopyEntity(original)
		copy, ok := result.(*gedcom.Family)

		if !ok {
			t.Fatal("Result should be *Family")
		}
		if copy == original {
			t.Error("Copy should have different pointer")
		}
		if copy.Husband != original.Husband {
			t.Errorf("Husband = %v, want %v", copy.Husband, original.Husband)
		}
		if len(copy.Children) != len(original.Children) {
			t.Errorf("Children length = %d, want %d", len(copy.Children), len(original.Children))
		}
	})

	t.Run("copies Source", func(t *testing.T) {
		original := &gedcom.Source{
			XRef:   "@S1@",
			Title:  "Test Source",
			Author: "Test Author",
		}

		result := deepCopyEntity(original)
		copy, ok := result.(*gedcom.Source)

		if !ok {
			t.Fatal("Result should be *Source")
		}
		if copy.Title != original.Title {
			t.Errorf("Title = %v, want %v", copy.Title, original.Title)
		}
	})

	t.Run("copies Repository", func(t *testing.T) {
		original := &gedcom.Repository{
			XRef: "@R1@",
			Name: "Test Repository",
		}

		result := deepCopyEntity(original)
		copy, ok := result.(*gedcom.Repository)

		if !ok {
			t.Fatal("Result should be *Repository")
		}
		if copy.Name != original.Name {
			t.Errorf("Name = %v, want %v", copy.Name, original.Name)
		}
	})

	t.Run("copies Note", func(t *testing.T) {
		original := &gedcom.Note{
			XRef: "@N1@",
			Text: "Test note text",
		}

		result := deepCopyEntity(original)
		copy, ok := result.(*gedcom.Note)

		if !ok {
			t.Fatal("Result should be *Note")
		}
		if copy.Text != original.Text {
			t.Errorf("Text = %v, want %v", copy.Text, original.Text)
		}
	})

	t.Run("copies MediaObject", func(t *testing.T) {
		original := &gedcom.MediaObject{
			XRef: "@M1@",
			Files: []*gedcom.MediaFile{
				{FileRef: "/path/to/file.jpg", Form: "JPG"},
			},
		}

		result := deepCopyEntity(original)
		copy, ok := result.(*gedcom.MediaObject)

		if !ok {
			t.Fatal("Result should be *MediaObject")
		}
		if len(copy.Files) != len(original.Files) {
			t.Errorf("Files length = %d, want %d", len(copy.Files), len(original.Files))
		}
	})

	t.Run("copies Submitter", func(t *testing.T) {
		original := &gedcom.Submitter{
			XRef: "@SUBM1@",
			Name: "Test Submitter",
		}

		result := deepCopyEntity(original)
		copy, ok := result.(*gedcom.Submitter)

		if !ok {
			t.Fatal("Result should be *Submitter")
		}
		if copy.Name != original.Name {
			t.Errorf("Name = %v, want %v", copy.Name, original.Name)
		}
	})

	t.Run("unknown entity returns as-is", func(t *testing.T) {
		original := "unknown type"
		result := deepCopyEntity(original)
		if result != original {
			t.Error("Unknown entity should be returned as-is")
		}
	})
}

func TestDeepCopyIndividual(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		result := deepCopyIndividual(nil)
		if result != nil {
			t.Error("deepCopyIndividual(nil) should return nil")
		}
	})

	t.Run("copies all fields", func(t *testing.T) {
		original := &gedcom.Individual{
			XRef:             "@I1@",
			Sex:              "M",
			SpouseInFamilies: []string{"@F1@"},
			Notes:            []string{"@N1@"},
			RefNumber:        "123",
			UID:              "uid-123",
			FamilySearchID:   "FSID",
			Names: []*gedcom.PersonalName{
				{Full: "John /Doe/"},
			},
			ChildInFamilies: []gedcom.FamilyLink{
				{FamilyXRef: "@F2@", Pedigree: "birth"},
			},
			Events: []*gedcom.Event{
				{Type: "BIRT", Date: "1 JAN 1900"},
			},
			Attributes: []*gedcom.Attribute{
				{Type: "OCCU", Value: "Farmer"},
			},
			Associations: []*gedcom.Association{
				{IndividualXRef: "@I2@", Role: "GODP"},
			},
			SourceCitations: []*gedcom.SourceCitation{
				{SourceXRef: "@S1@"},
			},
			Media: []*gedcom.MediaLink{
				{MediaXRef: "@M1@"},
			},
			LDSOrdinances: []*gedcom.LDSOrdinance{
				{Type: "BAPL", Temple: "SALT LAKE"},
			},
			ChangeDate:   &gedcom.ChangeDate{Date: "1 JAN 2024"},
			CreationDate: &gedcom.ChangeDate{Date: "1 JAN 2020"},
			Tags: []*gedcom.Tag{
				{Tag: "CUSTOM"},
			},
		}

		copy := deepCopyIndividual(original)

		if copy == original {
			t.Error("Copy should have different pointer")
		}
		if copy.XRef != original.XRef {
			t.Errorf("XRef = %v, want %v", copy.XRef, original.XRef)
		}
		if len(copy.Names) != len(original.Names) {
			t.Errorf("Names length = %d, want %d", len(copy.Names), len(original.Names))
		}
		if len(copy.ChildInFamilies) != len(original.ChildInFamilies) {
			t.Errorf("ChildInFamilies length = %d, want %d", len(copy.ChildInFamilies), len(original.ChildInFamilies))
		}
		if len(copy.Events) != len(original.Events) {
			t.Errorf("Events length = %d, want %d", len(copy.Events), len(original.Events))
		}
		if len(copy.Attributes) != len(original.Attributes) {
			t.Errorf("Attributes length = %d, want %d", len(copy.Attributes), len(original.Attributes))
		}
		if len(copy.Associations) != len(original.Associations) {
			t.Errorf("Associations length = %d, want %d", len(copy.Associations), len(original.Associations))
		}
		if copy.ChangeDate == original.ChangeDate {
			t.Error("ChangeDate should have different pointer")
		}
		if copy.CreationDate == original.CreationDate {
			t.Error("CreationDate should have different pointer")
		}
	})
}

func TestDeepCopyFamily(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		result := deepCopyFamily(nil)
		if result != nil {
			t.Error("deepCopyFamily(nil) should return nil")
		}
	})

	t.Run("copies all fields", func(t *testing.T) {
		original := &gedcom.Family{
			XRef:             "@F1@",
			Husband:          "@I1@",
			Wife:             "@I2@",
			Children:         []string{"@I3@"},
			NumberOfChildren: "1",
			Notes:            []string{"@N1@"},
			RefNumber:        "456",
			UID:              "uid-456",
			Events: []*gedcom.Event{
				{Type: "MARR", Date: "1 JAN 1920"},
			},
			SourceCitations: []*gedcom.SourceCitation{
				{SourceXRef: "@S1@"},
			},
			Media: []*gedcom.MediaLink{
				{MediaXRef: "@M1@"},
			},
			LDSOrdinances: []*gedcom.LDSOrdinance{
				{Type: "SLGS"},
			},
			ChangeDate:   &gedcom.ChangeDate{Date: "1 JAN 2024"},
			CreationDate: &gedcom.ChangeDate{Date: "1 JAN 2020"},
			Tags: []*gedcom.Tag{
				{Tag: "CUSTOM"},
			},
		}

		copy := deepCopyFamily(original)

		if copy == original {
			t.Error("Copy should have different pointer")
		}
		if copy.XRef != original.XRef {
			t.Errorf("XRef = %v, want %v", copy.XRef, original.XRef)
		}
		if copy.Husband != original.Husband {
			t.Errorf("Husband = %v, want %v", copy.Husband, original.Husband)
		}
		if len(copy.Children) != len(original.Children) {
			t.Errorf("Children length = %d, want %d", len(copy.Children), len(original.Children))
		}
		if len(copy.Events) != len(original.Events) {
			t.Errorf("Events length = %d, want %d", len(copy.Events), len(original.Events))
		}
	})
}

func TestDeepCopySource(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		result := deepCopySource(nil)
		if result != nil {
			t.Error("deepCopySource(nil) should return nil")
		}
	})

	t.Run("copies all fields", func(t *testing.T) {
		original := &gedcom.Source{
			XRef:          "@S1@",
			Title:         "Test Source",
			Author:        "Test Author",
			Publication:   "Publisher",
			Text:          "Source text",
			RepositoryRef: "@R1@",
			Notes:         []string{"@N1@"},
			RefNumber:     "789",
			UID:           "uid-789",
			Repository: &gedcom.InlineRepository{
				Name: "Inline Repo",
			},
			Media: []*gedcom.MediaLink{
				{MediaXRef: "@M1@"},
			},
			ChangeDate:   &gedcom.ChangeDate{Date: "1 JAN 2024"},
			CreationDate: &gedcom.ChangeDate{Date: "1 JAN 2020"},
			Tags: []*gedcom.Tag{
				{Tag: "CUSTOM"},
			},
		}

		copy := deepCopySource(original)

		if copy == original {
			t.Error("Copy should have different pointer")
		}
		if copy.Title != original.Title {
			t.Errorf("Title = %v, want %v", copy.Title, original.Title)
		}
		if copy.Repository == original.Repository {
			t.Error("Repository should have different pointer")
		}
		if copy.Repository.Name != original.Repository.Name {
			t.Errorf("Repository.Name = %v, want %v", copy.Repository.Name, original.Repository.Name)
		}
	})
}

func TestDeepCopyEvent(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		result := deepCopyEvent(nil)
		if result != nil {
			t.Error("deepCopyEvent(nil) should return nil")
		}
	})

	t.Run("copies all fields", func(t *testing.T) {
		original := &gedcom.Event{
			Type:            "BIRT",
			Date:            "1 JAN 1900",
			Place:           "New York",
			Description:     "Birth event",
			EventTypeDetail: "Birth",
			Cause:           "Natural",
			Age:             "0y",
			Agency:          "Hospital",
			Restriction:     "none",
			UID:             "event-uid",
			SortDate:        "19000101",
			Notes:           []string{"@N1@"},
			Phone:           []string{"123-456"},
			Email:           []string{"test@example.com"},
			Fax:             []string{"123-789"},
			Website:         []string{"http://example.com"},
			ParsedDate: &gedcom.Date{
				Year:  1900,
				Month: 1,
				Day:   1,
			},
			PlaceDetail: &gedcom.PlaceDetail{
				Name: "New York",
				Coordinates: &gedcom.Coordinates{
					Latitude:  "N40.7128",
					Longitude: "W74.0060",
				},
			},
			Address: &gedcom.Address{
				Line1: "123 Main St",
				City:  "New York",
			},
			SourceCitations: []*gedcom.SourceCitation{
				{SourceXRef: "@S1@"},
			},
			Media: []*gedcom.MediaLink{
				{MediaXRef: "@M1@"},
			},
			Tags: []*gedcom.Tag{
				{Tag: "CUSTOM"},
			},
		}

		copy := deepCopyEvent(original)

		if copy == original {
			t.Error("Copy should have different pointer")
		}
		if copy.Type != original.Type {
			t.Errorf("Type = %v, want %v", copy.Type, original.Type)
		}
		if copy.ParsedDate == original.ParsedDate {
			t.Error("ParsedDate should have different pointer")
		}
		if copy.PlaceDetail == original.PlaceDetail {
			t.Error("PlaceDetail should have different pointer")
		}
		if copy.PlaceDetail.Coordinates == original.PlaceDetail.Coordinates {
			t.Error("Coordinates should have different pointer")
		}
		if copy.Address == original.Address {
			t.Error("Address should have different pointer")
		}
	})
}

func TestDeepCopyDate(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		result := deepCopyDate(nil)
		if result != nil {
			t.Error("deepCopyDate(nil) should return nil")
		}
	})

	t.Run("copies date with range", func(t *testing.T) {
		original := &gedcom.Date{
			Original: "BET 1 JAN 1900 AND 31 DEC 1900",
			Day:      1,
			Month:    1,
			Year:     1900,
			Modifier: gedcom.ModifierBetween,
			Calendar: gedcom.CalendarGregorian,
			IsBC:     false,
			DualYear: 0,
			EndDate: &gedcom.Date{
				Day:   31,
				Month: 12,
				Year:  1900,
			},
		}

		copy := deepCopyDate(original)

		if copy == original {
			t.Error("Copy should have different pointer")
		}
		if copy.EndDate == original.EndDate {
			t.Error("EndDate should have different pointer")
		}
		if copy.Year != original.Year {
			t.Errorf("Year = %d, want %d", copy.Year, original.Year)
		}
		if copy.EndDate.Year != original.EndDate.Year {
			t.Errorf("EndDate.Year = %d, want %d", copy.EndDate.Year, original.EndDate.Year)
		}
	})
}

func TestDeepCopyMediaObject(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		result := deepCopyMediaObject(nil)
		if result != nil {
			t.Error("deepCopyMediaObject(nil) should return nil")
		}
	})

	t.Run("copies all fields", func(t *testing.T) {
		original := &gedcom.MediaObject{
			XRef:        "@M1@",
			Notes:       []string{"@N1@"},
			RefNumbers:  []string{"REF1"},
			Restriction: "none",
			UIDs:        []string{"uid-1"},
			Files: []*gedcom.MediaFile{
				{
					FileRef:   "/path/to/file.jpg",
					Form:      "JPG",
					MediaType: "photo",
					Title:     "Photo",
					Translations: []*gedcom.MediaTranslation{
						{FileRef: "/path/to/file2.jpg", Form: "JPG"},
					},
				},
			},
			SourceCitations: []*gedcom.SourceCitation{
				{SourceXRef: "@S1@"},
			},
			ChangeDate:   &gedcom.ChangeDate{Date: "1 JAN 2024"},
			CreationDate: &gedcom.ChangeDate{Date: "1 JAN 2020"},
			Tags: []*gedcom.Tag{
				{Tag: "CUSTOM"},
			},
		}

		copy := deepCopyMediaObject(original)

		if copy == original {
			t.Error("Copy should have different pointer")
		}
		if len(copy.Files) != len(original.Files) {
			t.Errorf("Files length = %d, want %d", len(copy.Files), len(original.Files))
		}
		if copy.Files[0] == original.Files[0] {
			t.Error("Files should have different pointers")
		}
		if len(copy.Files[0].Translations) != len(original.Files[0].Translations) {
			t.Errorf("Translations length = %d, want %d", len(copy.Files[0].Translations), len(original.Files[0].Translations))
		}
	})
}

func TestCopyStringSlice(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		result := copyStringSlice(nil)
		if result != nil {
			t.Error("copyStringSlice(nil) should return nil")
		}
	})

	t.Run("copies slice independently", func(t *testing.T) {
		original := []string{"a", "b", "c"}
		copy := copyStringSlice(original)

		if len(copy) != len(original) {
			t.Errorf("Copy length = %d, want %d", len(copy), len(original))
		}

		// Modify copy
		copy[0] = "modified"

		// Original should be unchanged
		if original[0] != "a" {
			t.Error("Original was mutated")
		}
	})
}

// Helper function to create a full test document with various entities
func createFullTestDocument() *gedcom.Document {
	ind := &gedcom.Individual{
		XRef: "@I1@",
		Sex:  "M",
		Names: []*gedcom.PersonalName{
			{Full: "John /Doe/", Given: "John", Surname: "Doe"},
		},
		Events: []*gedcom.Event{
			{Type: "BIRT", Date: "1 JAN 1900"},
		},
	}

	fam := &gedcom.Family{
		XRef:     "@F1@",
		Husband:  "@I1@",
		Wife:     "@I2@",
		Children: []string{"@I3@"},
	}

	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:      gedcom.Version551,
			Encoding:     gedcom.EncodingUTF8,
			SourceSystem: "TestSystem",
		},
		Trailer: &gedcom.Trailer{LineNumber: 100},
		Vendor:  "TestVendor",
		Records: []*gedcom.Record{
			{
				XRef:   "@I1@",
				Type:   gedcom.RecordTypeIndividual,
				Entity: ind,
				Tags: []*gedcom.Tag{
					{Level: 0, Tag: "INDI", XRef: "@I1@"},
					{Level: 1, Tag: "NAME", Value: "John /Doe/"},
				},
			},
			{
				XRef:   "@F1@",
				Type:   gedcom.RecordTypeFamily,
				Entity: fam,
				Tags: []*gedcom.Tag{
					{Level: 0, Tag: "FAM", XRef: "@F1@"},
				},
			},
		},
		XRefMap: make(map[string]*gedcom.Record),
	}

	doc.XRefMap["@I1@"] = doc.Records[0]
	doc.XRefMap["@F1@"] = doc.Records[1]

	return doc
}
