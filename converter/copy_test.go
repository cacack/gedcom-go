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

	t.Run("creates independent copied", func(t *testing.T) {
		original := createFullTestDocument()
		copied := deepCopyDocument(original)

		if copied == nil {
			t.Fatal("deepCopyDocument() returned nil")
		}

		// Verify different pointers
		if copied == original {
			t.Error("Copy should have different pointer than original")
		}
		if copied.Header == original.Header {
			t.Error("Copy header should have different pointer")
		}
		if len(copied.Records) > 0 && len(original.Records) > 0 && copied.Records[0] == original.Records[0] {
			t.Error("Copy records should have different pointers")
		}

		// Verify values are equal
		if copied.Header.Version != original.Header.Version {
			t.Errorf("Copy version = %v, want %v", copied.Header.Version, original.Header.Version)
		}
		if copied.Vendor != original.Vendor {
			t.Errorf("Copy vendor = %v, want %v", copied.Vendor, original.Vendor)
		}
	})

	t.Run("modifications to copied don't affect original", func(t *testing.T) {
		original := createFullTestDocument()
		originalVersion := original.Header.Version
		originalXRef := original.Records[0].XRef

		copied := deepCopyDocument(original)

		// Modify the copied
		copied.Header.Version = gedcom.Version70
		copied.Records[0].XRef = "@MODIFIED@"

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

		copied := deepCopyDocument(original)

		// XRefMap should point to the copied records, not originals
		if copied.XRefMap["@I1@"] == original.XRefMap["@I1@"] {
			t.Error("XRefMap entries should point to copied records")
		}
		if copied.XRefMap["@I1@"] != copied.Records[0] {
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

		copied := deepCopyHeader(original)

		if copied == original {
			t.Error("Copy should have different pointer")
		}
		if copied.Version != original.Version {
			t.Errorf("Version = %v, want %v", copied.Version, original.Version)
		}
		if copied.Encoding != original.Encoding {
			t.Errorf("Encoding = %v, want %v", copied.Encoding, original.Encoding)
		}
		if copied.SourceSystem != original.SourceSystem {
			t.Errorf("SourceSystem = %v, want %v", copied.SourceSystem, original.SourceSystem)
		}
		if len(copied.Tags) != len(original.Tags) {
			t.Errorf("Tags length = %d, want %d", len(copied.Tags), len(original.Tags))
		}
		if len(copied.Tags) > 0 && copied.Tags[0] == original.Tags[0] {
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
		copied := deepCopyTrailer(original)

		if copied == original {
			t.Error("Copy should have different pointer")
		}
		if copied.LineNumber != original.LineNumber {
			t.Errorf("LineNumber = %d, want %d", copied.LineNumber, original.LineNumber)
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

		copied := deepCopyRecord(original)

		if copied == original {
			t.Error("Copy should have different pointer")
		}
		if copied.XRef != original.XRef {
			t.Errorf("XRef = %v, want %v", copied.XRef, original.XRef)
		}
		if copied.Type != original.Type {
			t.Errorf("Type = %v, want %v", copied.Type, original.Type)
		}
		if copied.LineNumber != original.LineNumber {
			t.Errorf("LineNumber = %d, want %d", copied.LineNumber, original.LineNumber)
		}
		if len(copied.Tags) > 0 && copied.Tags[0] == original.Tags[0] {
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

		copied := deepCopyTags(original)

		if len(copied) != len(original) {
			t.Errorf("Copy length = %d, want %d", len(copied), len(original))
		}
		for i := range copied {
			if copied[i] == original[i] {
				t.Errorf("Tag[%d] should have different pointer", i)
			}
			if copied[i].Tag != original[i].Tag {
				t.Errorf("Tag[%d].Tag = %v, want %v", i, copied[i].Tag, original[i].Tag)
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

		copied := deepCopyTag(original)

		if copied == original {
			t.Error("Copy should have different pointer")
		}
		if copied.Level != original.Level {
			t.Errorf("Level = %d, want %d", copied.Level, original.Level)
		}
		if copied.Tag != original.Tag {
			t.Errorf("Tag = %v, want %v", copied.Tag, original.Tag)
		}
		if copied.Value != original.Value {
			t.Errorf("Value = %v, want %v", copied.Value, original.Value)
		}
		if copied.XRef != original.XRef {
			t.Errorf("XRef = %v, want %v", copied.XRef, original.XRef)
		}
		if copied.LineNumber != original.LineNumber {
			t.Errorf("LineNumber = %d, want %d", copied.LineNumber, original.LineNumber)
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
		copied, ok := result.(*gedcom.Individual)

		if !ok {
			t.Fatal("Result should be *Individual")
		}
		if copied == original {
			t.Error("Copy should have different pointer")
		}
		if copied.XRef != original.XRef {
			t.Errorf("XRef = %v, want %v", copied.XRef, original.XRef)
		}
		if len(copied.Names) != len(original.Names) {
			t.Errorf("Names length = %d, want %d", len(copied.Names), len(original.Names))
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
		copied, ok := result.(*gedcom.Family)

		if !ok {
			t.Fatal("Result should be *Family")
		}
		if copied == original {
			t.Error("Copy should have different pointer")
		}
		if copied.Husband != original.Husband {
			t.Errorf("Husband = %v, want %v", copied.Husband, original.Husband)
		}
		if len(copied.Children) != len(original.Children) {
			t.Errorf("Children length = %d, want %d", len(copied.Children), len(original.Children))
		}
	})

	t.Run("copies Source", func(t *testing.T) {
		original := &gedcom.Source{
			XRef:   "@S1@",
			Title:  "Test Source",
			Author: "Test Author",
		}

		result := deepCopyEntity(original)
		copied, ok := result.(*gedcom.Source)

		if !ok {
			t.Fatal("Result should be *Source")
		}
		if copied.Title != original.Title {
			t.Errorf("Title = %v, want %v", copied.Title, original.Title)
		}
	})

	t.Run("copies Repository", func(t *testing.T) {
		original := &gedcom.Repository{
			XRef: "@R1@",
			Name: "Test Repository",
		}

		result := deepCopyEntity(original)
		copied, ok := result.(*gedcom.Repository)

		if !ok {
			t.Fatal("Result should be *Repository")
		}
		if copied.Name != original.Name {
			t.Errorf("Name = %v, want %v", copied.Name, original.Name)
		}
	})

	t.Run("copies Note", func(t *testing.T) {
		original := &gedcom.Note{
			XRef: "@N1@",
			Text: "Test note text",
		}

		result := deepCopyEntity(original)
		copied, ok := result.(*gedcom.Note)

		if !ok {
			t.Fatal("Result should be *Note")
		}
		if copied.Text != original.Text {
			t.Errorf("Text = %v, want %v", copied.Text, original.Text)
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
		copied, ok := result.(*gedcom.MediaObject)

		if !ok {
			t.Fatal("Result should be *MediaObject")
		}
		if len(copied.Files) != len(original.Files) {
			t.Errorf("Files length = %d, want %d", len(copied.Files), len(original.Files))
		}
	})

	t.Run("copies Submitter", func(t *testing.T) {
		original := &gedcom.Submitter{
			XRef: "@SUBM1@",
			Name: "Test Submitter",
		}

		result := deepCopyEntity(original)
		copied, ok := result.(*gedcom.Submitter)

		if !ok {
			t.Fatal("Result should be *Submitter")
		}
		if copied.Name != original.Name {
			t.Errorf("Name = %v, want %v", copied.Name, original.Name)
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

		copied := deepCopyIndividual(original)

		if copied == original {
			t.Error("Copy should have different pointer")
		}
		if copied.XRef != original.XRef {
			t.Errorf("XRef = %v, want %v", copied.XRef, original.XRef)
		}
		if len(copied.Names) != len(original.Names) {
			t.Errorf("Names length = %d, want %d", len(copied.Names), len(original.Names))
		}
		if len(copied.ChildInFamilies) != len(original.ChildInFamilies) {
			t.Errorf("ChildInFamilies length = %d, want %d", len(copied.ChildInFamilies), len(original.ChildInFamilies))
		}
		if len(copied.Events) != len(original.Events) {
			t.Errorf("Events length = %d, want %d", len(copied.Events), len(original.Events))
		}
		if len(copied.Attributes) != len(original.Attributes) {
			t.Errorf("Attributes length = %d, want %d", len(copied.Attributes), len(original.Attributes))
		}
		if len(copied.Associations) != len(original.Associations) {
			t.Errorf("Associations length = %d, want %d", len(copied.Associations), len(original.Associations))
		}
		if copied.ChangeDate == original.ChangeDate {
			t.Error("ChangeDate should have different pointer")
		}
		if copied.CreationDate == original.CreationDate {
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

		copied := deepCopyFamily(original)

		if copied == original {
			t.Error("Copy should have different pointer")
		}
		if copied.XRef != original.XRef {
			t.Errorf("XRef = %v, want %v", copied.XRef, original.XRef)
		}
		if copied.Husband != original.Husband {
			t.Errorf("Husband = %v, want %v", copied.Husband, original.Husband)
		}
		if len(copied.Children) != len(original.Children) {
			t.Errorf("Children length = %d, want %d", len(copied.Children), len(original.Children))
		}
		if len(copied.Events) != len(original.Events) {
			t.Errorf("Events length = %d, want %d", len(copied.Events), len(original.Events))
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

		copied := deepCopySource(original)

		if copied == original {
			t.Error("Copy should have different pointer")
		}
		if copied.Title != original.Title {
			t.Errorf("Title = %v, want %v", copied.Title, original.Title)
		}
		if copied.Repository == original.Repository {
			t.Error("Repository should have different pointer")
		}
		if copied.Repository.Name != original.Repository.Name {
			t.Errorf("Repository.Name = %v, want %v", copied.Repository.Name, original.Repository.Name)
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

		copied := deepCopyEvent(original)

		if copied == original {
			t.Error("Copy should have different pointer")
		}
		if copied.Type != original.Type {
			t.Errorf("Type = %v, want %v", copied.Type, original.Type)
		}
		if copied.ParsedDate == original.ParsedDate {
			t.Error("ParsedDate should have different pointer")
		}
		if copied.PlaceDetail == original.PlaceDetail {
			t.Error("PlaceDetail should have different pointer")
		}
		if copied.PlaceDetail.Coordinates == original.PlaceDetail.Coordinates {
			t.Error("Coordinates should have different pointer")
		}
		if copied.Address == original.Address {
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

		copied := deepCopyDate(original)

		if copied == original {
			t.Error("Copy should have different pointer")
		}
		if copied.EndDate == original.EndDate {
			t.Error("EndDate should have different pointer")
		}
		if copied.Year != original.Year {
			t.Errorf("Year = %d, want %d", copied.Year, original.Year)
		}
		if copied.EndDate.Year != original.EndDate.Year {
			t.Errorf("EndDate.Year = %d, want %d", copied.EndDate.Year, original.EndDate.Year)
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

		copied := deepCopyMediaObject(original)

		if copied == original {
			t.Error("Copy should have different pointer")
		}
		if len(copied.Files) != len(original.Files) {
			t.Errorf("Files length = %d, want %d", len(copied.Files), len(original.Files))
		}
		if copied.Files[0] == original.Files[0] {
			t.Error("Files should have different pointers")
		}
		if len(copied.Files[0].Translations) != len(original.Files[0].Translations) {
			t.Errorf("Translations length = %d, want %d", len(copied.Files[0].Translations), len(original.Files[0].Translations))
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
		copied := copyStringSlice(original)

		if len(copied) != len(original) {
			t.Errorf("Copy length = %d, want %d", len(copied), len(original))
		}

		// Modify copied
		copied[0] = "modified"

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

func TestDeepCopyTransliteration(t *testing.T) {
	t.Run("nil transliteration", func(t *testing.T) {
		result := deepCopyTransliteration(nil)
		if result != nil {
			t.Error("Expected nil for nil input")
		}
	})

	t.Run("full transliteration", func(t *testing.T) {
		original := &gedcom.Transliteration{
			Value:         "John Doe",
			Language:      "en",
			Given:         "John",
			Surname:       "Doe",
			Prefix:        "Dr",
			Suffix:        "Jr",
			Nickname:      "Johnny",
			SurnamePrefix: "van",
		}

		copied := deepCopyTransliteration(original)

		if copied.Value != original.Value {
			t.Errorf("Value = %v, want %v", copied.Value, original.Value)
		}
		if copied.Language != original.Language {
			t.Errorf("Language = %v, want %v", copied.Language, original.Language)
		}
		if copied.Given != original.Given {
			t.Errorf("Given = %v, want %v", copied.Given, original.Given)
		}
		if copied.Surname != original.Surname {
			t.Errorf("Surname = %v, want %v", copied.Surname, original.Surname)
		}
		if copied.Prefix != original.Prefix {
			t.Errorf("Prefix = %v, want %v", copied.Prefix, original.Prefix)
		}
		if copied.Suffix != original.Suffix {
			t.Errorf("Suffix = %v, want %v", copied.Suffix, original.Suffix)
		}
		if copied.Nickname != original.Nickname {
			t.Errorf("Nickname = %v, want %v", copied.Nickname, original.Nickname)
		}
		if copied.SurnamePrefix != original.SurnamePrefix {
			t.Errorf("SurnamePrefix = %v, want %v", copied.SurnamePrefix, original.SurnamePrefix)
		}

		// Verify it's a copy, not the same pointer
		if copied == original {
			t.Error("Should be a copy, not the same pointer")
		}
	})
}

func TestDeepCopyAncestryAPID(t *testing.T) {
	t.Run("nil apid", func(t *testing.T) {
		result := deepCopyAncestryAPID(nil)
		if result != nil {
			t.Error("Expected nil for nil input")
		}
	})

	t.Run("full apid", func(t *testing.T) {
		original := &gedcom.AncestryAPID{
			Raw:      "1234:5678:90",
			Database: "1234",
			Record:   "5678",
		}

		copied := deepCopyAncestryAPID(original)

		if copied.Raw != original.Raw {
			t.Errorf("Raw = %v, want %v", copied.Raw, original.Raw)
		}
		if copied.Database != original.Database {
			t.Errorf("Database = %v, want %v", copied.Database, original.Database)
		}
		if copied.Record != original.Record {
			t.Errorf("Record = %v, want %v", copied.Record, original.Record)
		}

		// Verify it's a copy
		if copied == original {
			t.Error("Should be a copy, not the same pointer")
		}
	})
}

func TestDeepCopyAssociationWithCitations(t *testing.T) {
	t.Run("association with source citations", func(t *testing.T) {
		original := &gedcom.Association{
			IndividualXRef: "@I2@",
			Role:           "Witness",
			Phrase:         "witness to the event",
			Notes:          []string{"Note 1", "Note 2"},
			SourceCitations: []*gedcom.SourceCitation{
				{SourceXRef: "@S1@", Page: "Page 10"},
				{SourceXRef: "@S2@", Page: "Page 20"},
			},
		}

		copied := deepCopyAssociation(original)

		if copied.IndividualXRef != original.IndividualXRef {
			t.Errorf("IndividualXRef = %v, want %v", copied.IndividualXRef, original.IndividualXRef)
		}
		if copied.Role != original.Role {
			t.Errorf("Role = %v, want %v", copied.Role, original.Role)
		}
		if copied.Phrase != original.Phrase {
			t.Errorf("Phrase = %v, want %v", copied.Phrase, original.Phrase)
		}
		if len(copied.Notes) != len(original.Notes) {
			t.Errorf("Notes len = %d, want %d", len(copied.Notes), len(original.Notes))
		}
		if len(copied.SourceCitations) != len(original.SourceCitations) {
			t.Errorf("SourceCitations len = %d, want %d", len(copied.SourceCitations), len(original.SourceCitations))
		}

		// Verify source citations are copied
		if copied.SourceCitations[0].SourceXRef != original.SourceCitations[0].SourceXRef {
			t.Errorf("SourceCitation[0].SourceXRef = %v, want %v", copied.SourceCitations[0].SourceXRef, original.SourceCitations[0].SourceXRef)
		}

		// Verify it's a deep copy
		original.SourceCitations[0].Page = "Modified"
		if copied.SourceCitations[0].Page == "Modified" {
			t.Error("Should be a deep copy, modification affected copy")
		}
	})
}

func TestDeepCopyAttributeWithCitations(t *testing.T) {
	t.Run("attribute with source citations", func(t *testing.T) {
		original := &gedcom.Attribute{
			Type:  "OCCU",
			Value: "Farmer",
			Date:  "1900",
			Place: "Iowa",
			ParsedDate: &gedcom.Date{
				Original: "1900",
				Year:     1900,
			},
			SourceCitations: []*gedcom.SourceCitation{
				{SourceXRef: "@S1@", Page: "Page 5"},
			},
		}

		copied := deepCopyAttribute(original)

		if copied.Type != original.Type {
			t.Errorf("Type = %v, want %v", copied.Type, original.Type)
		}
		if copied.Value != original.Value {
			t.Errorf("Value = %v, want %v", copied.Value, original.Value)
		}
		if copied.Date != original.Date {
			t.Errorf("Date = %v, want %v", copied.Date, original.Date)
		}
		if copied.Place != original.Place {
			t.Errorf("Place = %v, want %v", copied.Place, original.Place)
		}
		if copied.ParsedDate == nil {
			t.Error("ParsedDate should not be nil")
		}
		if len(copied.SourceCitations) != len(original.SourceCitations) {
			t.Errorf("SourceCitations len = %d, want %d", len(copied.SourceCitations), len(original.SourceCitations))
		}

		// Verify deep copy
		original.SourceCitations[0].Page = "Modified"
		if copied.SourceCitations[0].Page == "Modified" {
			t.Error("Should be a deep copy")
		}
	})
}

func TestDeepCopyPersonalNameFull(t *testing.T) {
	t.Run("personal name with all fields", func(t *testing.T) {
		original := &gedcom.PersonalName{
			Full:          "Dr. John /van Doe/ Jr.",
			Given:         "John",
			Surname:       "Doe",
			Prefix:        "Dr.",
			Suffix:        "Jr.",
			Nickname:      "Johnny",
			SurnamePrefix: "van",
			Type:          "birth",
			Transliterations: []*gedcom.Transliteration{
				{Value: "John Doe", Language: "en"},
				{Value: "Jon Do", Language: "en-phonetic"},
			},
		}

		copied := deepCopyPersonalName(original)

		if copied.Full != original.Full {
			t.Errorf("Full = %v, want %v", copied.Full, original.Full)
		}
		if copied.Given != original.Given {
			t.Errorf("Given = %v, want %v", copied.Given, original.Given)
		}
		if copied.Surname != original.Surname {
			t.Errorf("Surname = %v, want %v", copied.Surname, original.Surname)
		}
		if copied.Prefix != original.Prefix {
			t.Errorf("Prefix = %v, want %v", copied.Prefix, original.Prefix)
		}
		if copied.Suffix != original.Suffix {
			t.Errorf("Suffix = %v, want %v", copied.Suffix, original.Suffix)
		}
		if copied.Nickname != original.Nickname {
			t.Errorf("Nickname = %v, want %v", copied.Nickname, original.Nickname)
		}
		if copied.SurnamePrefix != original.SurnamePrefix {
			t.Errorf("SurnamePrefix = %v, want %v", copied.SurnamePrefix, original.SurnamePrefix)
		}
		if copied.Type != original.Type {
			t.Errorf("Type = %v, want %v", copied.Type, original.Type)
		}
		if len(copied.Transliterations) != len(original.Transliterations) {
			t.Errorf("Transliterations len = %d, want %d", len(copied.Transliterations), len(original.Transliterations))
		}

		// Verify deep copy
		original.Transliterations[0].Value = "Modified"
		if copied.Transliterations[0].Value == "Modified" {
			t.Error("Should be a deep copy")
		}
	})
}

func TestDeepCopyMediaLinkWithCrop(t *testing.T) {
	t.Run("media link with crop region", func(t *testing.T) {
		original := &gedcom.MediaLink{
			MediaXRef: "@M1@",
			Title:     "Photo",
			Crop: &gedcom.CropRegion{
				Height: 100,
				Left:   10,
				Top:    20,
				Width:  200,
			},
		}

		copied := deepCopyMediaLink(original)

		if copied.MediaXRef != original.MediaXRef {
			t.Errorf("MediaXRef = %v, want %v", copied.MediaXRef, original.MediaXRef)
		}
		if copied.Title != original.Title {
			t.Errorf("Title = %v, want %v", copied.Title, original.Title)
		}
		if copied.Crop == nil {
			t.Fatal("Crop should not be nil")
		}
		if copied.Crop.Height != original.Crop.Height {
			t.Errorf("Crop.Height = %v, want %v", copied.Crop.Height, original.Crop.Height)
		}
		if copied.Crop.Left != original.Crop.Left {
			t.Errorf("Crop.Left = %v, want %v", copied.Crop.Left, original.Crop.Left)
		}
		if copied.Crop.Top != original.Crop.Top {
			t.Errorf("Crop.Top = %v, want %v", copied.Crop.Top, original.Crop.Top)
		}
		if copied.Crop.Width != original.Crop.Width {
			t.Errorf("Crop.Width = %v, want %v", copied.Crop.Width, original.Crop.Width)
		}

		// Verify deep copy
		if copied.Crop == original.Crop {
			t.Error("Crop should be a deep copy")
		}
	})
}

func TestDeepCopySourceCitationFull(t *testing.T) {
	t.Run("source citation with all fields", func(t *testing.T) {
		original := &gedcom.SourceCitation{
			SourceXRef: "@S1@",
			Page:       "Page 123",
			Quality:    2,
			AncestryAPID: &gedcom.AncestryAPID{
				Raw:      "1:2:3",
				Database: "1",
				Record:   "2",
			},
		}

		copied := deepCopySourceCitation(original)

		if copied.SourceXRef != original.SourceXRef {
			t.Errorf("SourceXRef = %v, want %v", copied.SourceXRef, original.SourceXRef)
		}
		if copied.Page != original.Page {
			t.Errorf("Page = %v, want %v", copied.Page, original.Page)
		}
		if copied.Quality != original.Quality {
			t.Errorf("Quality = %v, want %v", copied.Quality, original.Quality)
		}
		if copied.AncestryAPID == nil {
			t.Fatal("AncestryAPID should not be nil")
		}
		if copied.AncestryAPID.Raw != original.AncestryAPID.Raw {
			t.Errorf("AncestryAPID.Raw = %v, want %v", copied.AncestryAPID.Raw, original.AncestryAPID.Raw)
		}

		// Verify deep copy
		if copied.AncestryAPID == original.AncestryAPID {
			t.Error("AncestryAPID should be a deep copy")
		}
	})
}

func TestDeepCopyLDSOrdinanceFull(t *testing.T) {
	t.Run("LDS ordinance with all fields", func(t *testing.T) {
		original := &gedcom.LDSOrdinance{
			Type:       "BAPL",
			Date:       "1 JAN 2000",
			Temple:     "SLAKE",
			Place:      "Salt Lake City",
			Status:     "COMPLETED",
			FamilyXRef: "@F1@",
		}

		copied := deepCopyLDSOrdinance(original)

		if string(copied.Type) != string(original.Type) {
			t.Errorf("Type = %v, want %v", copied.Type, original.Type)
		}
		if copied.Date != original.Date {
			t.Errorf("Date = %v, want %v", copied.Date, original.Date)
		}
		if copied.Temple != original.Temple {
			t.Errorf("Temple = %v, want %v", copied.Temple, original.Temple)
		}
		if copied.Place != original.Place {
			t.Errorf("Place = %v, want %v", copied.Place, original.Place)
		}
		if copied.Status != original.Status {
			t.Errorf("Status = %v, want %v", copied.Status, original.Status)
		}
		if copied.FamilyXRef != original.FamilyXRef {
			t.Errorf("FamilyXRef = %v, want %v", copied.FamilyXRef, original.FamilyXRef)
		}
	})
}

func TestDeepCopyMediaFileFull(t *testing.T) {
	t.Run("media file with translations", func(t *testing.T) {
		original := &gedcom.MediaFile{
			FileRef:   "/path/to/file.jpg",
			Form:      "image/jpeg",
			MediaType: "photo",
			Title:     "Family Photo",
			Translations: []*gedcom.MediaTranslation{
				{FileRef: "/path/to/file.png", Form: "image/png"},
			},
		}

		copied := deepCopyMediaFile(original)

		if copied.FileRef != original.FileRef {
			t.Errorf("FileRef = %v, want %v", copied.FileRef, original.FileRef)
		}
		if copied.Form != original.Form {
			t.Errorf("Form = %v, want %v", copied.Form, original.Form)
		}
		if copied.MediaType != original.MediaType {
			t.Errorf("MediaType = %v, want %v", copied.MediaType, original.MediaType)
		}
		if copied.Title != original.Title {
			t.Errorf("Title = %v, want %v", copied.Title, original.Title)
		}
		if len(copied.Translations) != len(original.Translations) {
			t.Errorf("Translations len = %d, want %d", len(copied.Translations), len(original.Translations))
		}
	})
}
