package gedcom

import (
	"testing"
	"time"
)

func TestDocumentClone(t *testing.T) {
	t.Run("nil document returns nil", func(t *testing.T) {
		var d *Document
		if d.Clone() != nil {
			t.Error("(*Document)(nil).Clone() should return nil")
		}
	})

	t.Run("creates independent copy", func(t *testing.T) {
		original := createFullTestDocument()
		copied := original.Clone()

		if copied == nil {
			t.Fatal("Clone() returned nil")
		}
		if copied == original {
			t.Error("Copy should have different pointer than original")
		}
		if copied.Header == original.Header {
			t.Error("Copy header should have different pointer")
		}
		if len(copied.Records) > 0 && len(original.Records) > 0 && copied.Records[0] == original.Records[0] {
			t.Error("Copy records should have different pointers")
		}

		if copied.Header.Version != original.Header.Version {
			t.Errorf("Copy version = %v, want %v", copied.Header.Version, original.Header.Version)
		}
		if copied.Vendor != original.Vendor {
			t.Errorf("Copy vendor = %v, want %v", copied.Vendor, original.Vendor)
		}
	})

	t.Run("modifications to copy do not affect original", func(t *testing.T) {
		original := createFullTestDocument()
		originalVersion := original.Header.Version
		originalXRef := original.Records[0].XRef

		copied := original.Clone()
		copied.Header.Version = Version70
		copied.Records[0].XRef = "@MODIFIED@"

		if original.Header.Version != originalVersion {
			t.Errorf("Original version mutated: got %v, want %v", original.Header.Version, originalVersion)
		}
		if original.Records[0].XRef != originalXRef {
			t.Errorf("Original XRef mutated: got %v, want %v", original.Records[0].XRef, originalXRef)
		}
	})

	t.Run("XRefMap keys are updated for new records", func(t *testing.T) {
		original := &Document{
			Header: &Header{Version: Version55},
			Records: []*Record{
				{XRef: "@I1@", Type: RecordTypeIndividual},
				{XRef: "@F1@", Type: RecordTypeFamily},
			},
			XRefMap: map[string]*Record{},
		}
		original.XRefMap["@I1@"] = original.Records[0]
		original.XRefMap["@F1@"] = original.Records[1]

		copied := original.Clone()

		if copied.XRefMap["@I1@"] == original.XRefMap["@I1@"] {
			t.Error("XRefMap entries should point to copied records")
		}
		if copied.XRefMap["@I1@"] != copied.Records[0] {
			t.Error("XRefMap should point to the correct copied record")
		}
	})

	t.Run("Schema is deep copied", func(t *testing.T) {
		original := &Document{
			Header: &Header{Version: Version70},
			Schema: &SchemaDefinition{
				TagMappings: map[string]string{
					"_SKYPEID": "http://example.com/skype",
				},
			},
			XRefMap: map[string]*Record{},
		}

		copied := original.Clone()

		if copied.Schema == original.Schema {
			t.Error("Schema should be a deep copy, not shared pointer")
		}
		if copied.Schema.TagMappings["_SKYPEID"] != original.Schema.TagMappings["_SKYPEID"] {
			t.Error("Schema TagMappings should be copied with same values")
		}
		copied.Schema.TagMappings["_SKYPEID"] = "modified"
		if original.Schema.TagMappings["_SKYPEID"] == "modified" {
			t.Error("Original Schema mutated by changing copy")
		}
	})
}

func TestHeaderClone(t *testing.T) {
	t.Run("nil header returns nil", func(t *testing.T) {
		var h *Header
		if h.Clone() != nil {
			t.Error("(*Header)(nil).Clone() should return nil")
		}
	})

	t.Run("copies all header fields", func(t *testing.T) {
		original := &Header{
			Version:        Version551,
			Encoding:       EncodingUTF8,
			SourceSystem:   "TestSystem",
			Date:           time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Language:       "English",
			Copyright:      "(c) 2024",
			Submitter:      "@SUBM1@",
			AncestryTreeID: "tree123",
			Tags: []*Tag{
				{Level: 1, Tag: "TEST", Value: "value"},
			},
		}

		copied := original.Clone()

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

func TestTrailerClone(t *testing.T) {
	t.Run("nil trailer returns nil", func(t *testing.T) {
		var tr *Trailer
		if tr.Clone() != nil {
			t.Error("(*Trailer)(nil).Clone() should return nil")
		}
	})

	t.Run("copies trailer fields", func(t *testing.T) {
		original := &Trailer{LineNumber: 100}
		copied := original.Clone()

		if copied == original {
			t.Error("Copy should have different pointer")
		}
		if copied.LineNumber != original.LineNumber {
			t.Errorf("LineNumber = %d, want %d", copied.LineNumber, original.LineNumber)
		}
	})
}

func TestRecordClone(t *testing.T) {
	t.Run("nil record returns nil", func(t *testing.T) {
		var r *Record
		if r.Clone() != nil {
			t.Error("(*Record)(nil).Clone() should return nil")
		}
	})

	t.Run("copies record fields", func(t *testing.T) {
		original := &Record{
			XRef:       "@I1@",
			Type:       RecordTypeIndividual,
			Value:      "test value",
			LineNumber: 10,
			Tags: []*Tag{
				{Level: 1, Tag: "NAME", Value: "John /Doe/"},
			},
		}

		copied := original.Clone()

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

func TestCloneTags(t *testing.T) {
	t.Run("nil tags returns nil", func(t *testing.T) {
		if CloneTags(nil) != nil {
			t.Error("CloneTags(nil) should return nil")
		}
	})

	t.Run("copies tags deeply", func(t *testing.T) {
		original := []*Tag{
			{Level: 0, Tag: "INDI"},
			{Level: 1, Tag: "NAME", Value: "John"},
		}

		copied := CloneTags(original)
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

func TestTagClone(t *testing.T) {
	t.Run("nil tag returns nil", func(t *testing.T) {
		var tag *Tag
		if tag.Clone() != nil {
			t.Error("(*Tag)(nil).Clone() should return nil")
		}
	})

	t.Run("copies all tag fields", func(t *testing.T) {
		original := &Tag{
			Level:      2,
			Tag:        "NOTE",
			Value:      "Some note",
			XRef:       "@N1@",
			LineNumber: 42,
		}

		copied := original.Clone()
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

func TestCloneEntity(t *testing.T) {
	t.Run("nil entity returns nil", func(t *testing.T) {
		if cloneEntity(nil) != nil {
			t.Error("cloneEntity(nil) should return nil")
		}
	})

	t.Run("copies Individual", func(t *testing.T) {
		original := &Individual{
			XRef: "@I1@",
			Sex:  "M",
			Names: []*PersonalName{
				{Full: "John /Doe/", Given: "John", Surname: "Doe"},
			},
		}

		result := cloneEntity(original)
		copied, ok := result.(*Individual)
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
		original := &Family{
			XRef:     "@F1@",
			Husband:  "@I1@",
			Wife:     "@I2@",
			Children: []string{"@I3@", "@I4@"},
		}

		result := cloneEntity(original)
		copied, ok := result.(*Family)
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
		original := &Source{XRef: "@S1@", Title: "Test Source", Author: "Test Author"}
		result := cloneEntity(original)
		copied, ok := result.(*Source)
		if !ok {
			t.Fatal("Result should be *Source")
		}
		if copied.Title != original.Title {
			t.Errorf("Title = %v, want %v", copied.Title, original.Title)
		}
	})

	t.Run("copies Repository", func(t *testing.T) {
		original := &Repository{XRef: "@R1@", Name: "Test Repository"}
		result := cloneEntity(original)
		copied, ok := result.(*Repository)
		if !ok {
			t.Fatal("Result should be *Repository")
		}
		if copied.Name != original.Name {
			t.Errorf("Name = %v, want %v", copied.Name, original.Name)
		}
	})

	t.Run("copies Note", func(t *testing.T) {
		original := &Note{XRef: "@N1@", Text: "Test note text"}
		result := cloneEntity(original)
		copied, ok := result.(*Note)
		if !ok {
			t.Fatal("Result should be *Note")
		}
		if copied.Text != original.Text {
			t.Errorf("Text = %v, want %v", copied.Text, original.Text)
		}
	})

	t.Run("copies MediaObject", func(t *testing.T) {
		original := &MediaObject{
			XRef: "@M1@",
			Files: []*MediaFile{
				{FileRef: "/path/to/file.jpg", Form: "JPG"},
			},
		}
		result := cloneEntity(original)
		copied, ok := result.(*MediaObject)
		if !ok {
			t.Fatal("Result should be *MediaObject")
		}
		if len(copied.Files) != len(original.Files) {
			t.Errorf("Files length = %d, want %d", len(copied.Files), len(original.Files))
		}
	})

	t.Run("copies Submitter", func(t *testing.T) {
		original := &Submitter{XRef: "@SUBM1@", Name: "Test Submitter"}
		result := cloneEntity(original)
		copied, ok := result.(*Submitter)
		if !ok {
			t.Fatal("Result should be *Submitter")
		}
		if copied.Name != original.Name {
			t.Errorf("Name = %v, want %v", copied.Name, original.Name)
		}
	})

	t.Run("copies SharedNote", func(t *testing.T) {
		original := &SharedNote{
			XRef:     "@SN1@",
			Text:     "Shared text",
			MIME:     "text/plain",
			Language: "en",
			Translations: []*SharedNoteTranslation{
				{Value: "Texte partagé", Language: "fr"},
			},
		}
		result := cloneEntity(original)
		copied, ok := result.(*SharedNote)
		if !ok {
			t.Fatal("Result should be *SharedNote")
		}
		if copied == original {
			t.Error("Copy should have different pointer")
		}
		if copied.Text != original.Text {
			t.Errorf("Text = %v, want %v", copied.Text, original.Text)
		}
		if len(copied.Translations) != 1 {
			t.Fatalf("Translations length = %d, want 1", len(copied.Translations))
		}
		if copied.Translations[0] == original.Translations[0] {
			t.Error("Translation entries should be deep copied")
		}
	})

	t.Run("unknown entity returns as-is", func(t *testing.T) {
		original := "unknown type"
		result := cloneEntity(original)
		if result != original {
			t.Error("Unknown entity should be returned as-is")
		}
	})
}

func TestIndividualClone(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		var i *Individual
		if i.Clone() != nil {
			t.Error("(*Individual)(nil).Clone() should return nil")
		}
	})

	t.Run("copies all fields", func(t *testing.T) {
		original := &Individual{
			XRef:             "@I1@",
			Sex:              "M",
			SpouseInFamilies: []string{"@F1@"},
			Notes:            []string{"@N1@"},
			RefNumber:        "123",
			UID:              "uid-123",
			FamilySearchID:   "FSID",
			Names:            []*PersonalName{{Full: "John /Doe/"}},
			ChildInFamilies:  []FamilyLink{{FamilyXRef: "@F2@", Pedigree: "birth"}},
			Events:           []*Event{{Type: "BIRT", Date: "1 JAN 1900"}},
			Attributes:       []*Attribute{{Type: "OCCU", Value: "Farmer"}},
			Associations:     []*Association{{IndividualXRef: "@I2@", Role: "GODP"}},
			SourceCitations:  []*SourceCitation{{SourceXRef: "@S1@"}},
			Media:            []*MediaLink{{MediaXRef: "@M1@"}},
			LDSOrdinances:    []*LDSOrdinance{{Type: "BAPL", Temple: "SALT LAKE"}},
			ExternalIDs:      []*ExternalID{{Value: "abc", Type: "http://example.com/id"}},
			ChangeDate:       &ChangeDate{Date: "1 JAN 2024"},
			CreationDate:     &ChangeDate{Date: "1 JAN 2020"},
			Tags:             []*Tag{{Tag: "CUSTOM"}},
		}

		copied := original.Clone()
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
		if len(copied.ExternalIDs) != 1 || copied.ExternalIDs[0] == original.ExternalIDs[0] {
			t.Error("ExternalIDs should be deep copied")
		}
		if copied.ChangeDate == original.ChangeDate {
			t.Error("ChangeDate should have different pointer")
		}
		if copied.CreationDate == original.CreationDate {
			t.Error("CreationDate should have different pointer")
		}
	})
}

func TestFamilyClone(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		var f *Family
		if f.Clone() != nil {
			t.Error("(*Family)(nil).Clone() should return nil")
		}
	})

	t.Run("copies all fields", func(t *testing.T) {
		original := &Family{
			XRef:             "@F1@",
			Husband:          "@I1@",
			Wife:             "@I2@",
			Children:         []string{"@I3@"},
			NumberOfChildren: "1",
			Notes:            []string{"@N1@"},
			RefNumber:        "456",
			UID:              "uid-456",
			Events:           []*Event{{Type: "MARR", Date: "1 JAN 1920"}},
			SourceCitations:  []*SourceCitation{{SourceXRef: "@S1@"}},
			Media:            []*MediaLink{{MediaXRef: "@M1@"}},
			LDSOrdinances:    []*LDSOrdinance{{Type: "SLGS"}},
			ChangeDate:       &ChangeDate{Date: "1 JAN 2024"},
			CreationDate:     &ChangeDate{Date: "1 JAN 2020"},
			Tags:             []*Tag{{Tag: "CUSTOM"}},
		}

		copied := original.Clone()
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

func TestSourceClone(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		var s *Source
		if s.Clone() != nil {
			t.Error("(*Source)(nil).Clone() should return nil")
		}
	})

	t.Run("copies all fields", func(t *testing.T) {
		original := &Source{
			XRef:          "@S1@",
			Title:         "Test Source",
			Author:        "Test Author",
			Publication:   "Publisher",
			Text:          "Source text",
			RepositoryRef: "@R1@",
			Notes:         []string{"@N1@"},
			RefNumber:     "789",
			UID:           "uid-789",
			Repository:    &InlineRepository{Name: "Inline Repo"},
			RepositoryLink: &SourceRepositoryLink{
				XRef:            "@R1@",
				CallNumbers:     []string{"MS-1234"},
				MediaType:       "Manuscript",
				CallNumberMedia: map[string]string{"MS-1234": "Manuscript"},
				Notes:           []string{"Held in archives"},
			},
			Media:        []*MediaLink{{MediaXRef: "@M1@"}},
			ChangeDate:   &ChangeDate{Date: "1 JAN 2024"},
			CreationDate: &ChangeDate{Date: "1 JAN 2020"},
			Tags:         []*Tag{{Tag: "CUSTOM"}},
		}

		copied := original.Clone()
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
		if copied.RepositoryLink == original.RepositoryLink {
			t.Error("RepositoryLink should have different pointer")
		}
		if copied.RepositoryLink.XRef != original.RepositoryLink.XRef {
			t.Errorf("RepositoryLink.XRef = %v, want %v", copied.RepositoryLink.XRef, original.RepositoryLink.XRef)
		}
		// Mutating the copy's maps/slices must not affect the original.
		copied.RepositoryLink.CallNumberMedia["MS-1234"] = "changed"
		if original.RepositoryLink.CallNumberMedia["MS-1234"] != "Manuscript" {
			t.Error("RepositoryLink.CallNumberMedia map shares backing storage with original")
		}
		copied.RepositoryLink.CallNumbers[0] = "changed"
		if original.RepositoryLink.CallNumbers[0] != "MS-1234" {
			t.Error("RepositoryLink.CallNumbers slice shares backing storage with original")
		}
	})

	t.Run("nil RepositoryLink clones to nil", func(t *testing.T) {
		original := &Source{XRef: "@S2@"}
		if original.Clone().RepositoryLink != nil {
			t.Error("nil RepositoryLink should clone to nil")
		}
	})
}

func TestCloneEvent(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		if cloneEvent(nil) != nil {
			t.Error("cloneEvent(nil) should return nil")
		}
	})

	t.Run("copies all fields", func(t *testing.T) {
		original := &Event{
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
			ParsedDate:      &Date{Year: 1900, Month: 1, Day: 1},
			PlaceDetail: &PlaceDetail{
				Name:        "New York",
				Coordinates: &Coordinates{Latitude: "N40.7128", Longitude: "W74.0060"},
			},
			Address:         &Address{Line1: "123 Main St", City: "New York"},
			SourceCitations: []*SourceCitation{{SourceXRef: "@S1@"}},
			Media:           []*MediaLink{{MediaXRef: "@M1@"}},
			Tags:            []*Tag{{Tag: "CUSTOM"}},
		}

		copied := cloneEvent(original)
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

func TestCloneDate(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		if cloneDate(nil) != nil {
			t.Error("cloneDate(nil) should return nil")
		}
	})

	t.Run("copies date with range", func(t *testing.T) {
		original := &Date{
			Original: "BET 1 JAN 1900 AND 31 DEC 1900",
			Day:      1,
			Month:    1,
			Year:     1900,
			Modifier: ModifierBetween,
			Calendar: CalendarGregorian,
			IsBC:     false,
			DualYear: 0,
			EndDate:  &Date{Day: 31, Month: 12, Year: 1900},
		}

		copied := cloneDate(original)
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

func TestMediaObjectClone(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		var m *MediaObject
		if m.Clone() != nil {
			t.Error("(*MediaObject)(nil).Clone() should return nil")
		}
	})

	t.Run("copies all fields", func(t *testing.T) {
		original := &MediaObject{
			XRef:        "@M1@",
			Notes:       []string{"@N1@"},
			RefNumbers:  []string{"REF1"},
			Restriction: "none",
			UIDs:        []string{"uid-1"},
			Files: []*MediaFile{
				{
					FileRef:      "/path/to/file.jpg",
					Form:         "JPG",
					MediaType:    "photo",
					Title:        "Photo",
					Translations: []*MediaTranslation{{FileRef: "/path/to/file2.jpg", Form: "JPG"}},
				},
			},
			SourceCitations: []*SourceCitation{{SourceXRef: "@S1@"}},
			ChangeDate:      &ChangeDate{Date: "1 JAN 2024"},
			CreationDate:    &ChangeDate{Date: "1 JAN 2020"},
			Tags:            []*Tag{{Tag: "CUSTOM"}},
		}

		copied := original.Clone()
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

func TestSubmitterClone(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		var s *Submitter
		if s.Clone() != nil {
			t.Error("(*Submitter)(nil).Clone() should return nil")
		}
	})

	t.Run("copies all fields", func(t *testing.T) {
		original := &Submitter{
			XRef:     "@SUBM1@",
			Name:     "Test",
			Address:  &Address{Line1: "1 Main St"},
			Phone:    []string{"555-1212"},
			Email:    []string{"a@b.c"},
			Language: []string{"en"},
			Notes:    []string{"@N1@"},
			Tags:     []*Tag{{Tag: "CUSTOM"}},
		}

		copied := original.Clone()
		if copied == original {
			t.Error("Copy should have different pointer")
		}
		if copied.Address == original.Address {
			t.Error("Address should be deep copied")
		}
		copied.Phone[0] = "modified"
		if original.Phone[0] == "modified" {
			t.Error("Phone slice was not deep copied")
		}
	})
}

func TestSharedNoteClone(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		var s *SharedNote
		if s.Clone() != nil {
			t.Error("(*SharedNote)(nil).Clone() should return nil")
		}
	})

	t.Run("copies all fields", func(t *testing.T) {
		original := &SharedNote{
			XRef:            "@SN1@",
			Text:            "shared",
			MIME:            "text/plain",
			Language:        "en",
			Translations:    []*SharedNoteTranslation{{Value: "v", MIME: "text/plain", Language: "fr"}},
			SourceCitations: []*SourceCitation{{SourceXRef: "@S1@"}},
			ExternalIDs:     []*ExternalID{{Value: "x", Type: "y"}},
			ChangeDate:      &ChangeDate{Date: "1 JAN 2024"},
			Tags:            []*Tag{{Tag: "CUSTOM"}},
		}

		copied := original.Clone()
		if copied == original {
			t.Error("Copy should have different pointer")
		}
		if copied.Translations[0] == original.Translations[0] {
			t.Error("Translations should be deep copied")
		}
		if copied.SourceCitations[0] == original.SourceCitations[0] {
			t.Error("SourceCitations should be deep copied")
		}
		if copied.ExternalIDs[0] == original.ExternalIDs[0] {
			t.Error("ExternalIDs should be deep copied")
		}
	})
}

func TestCloneStringSlice(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		if cloneStringSlice(nil) != nil {
			t.Error("cloneStringSlice(nil) should return nil")
		}
	})

	t.Run("copies slice independently", func(t *testing.T) {
		original := []string{"a", "b", "c"}
		copied := cloneStringSlice(original)

		if len(copied) != len(original) {
			t.Errorf("Copy length = %d, want %d", len(copied), len(original))
		}
		copied[0] = "modified"
		if original[0] != "a" {
			t.Error("Original was mutated")
		}
	})
}

func TestCloneTransliteration(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if cloneTransliteration(nil) != nil {
			t.Error("Expected nil for nil input")
		}
	})

	t.Run("full", func(t *testing.T) {
		original := &Transliteration{
			Value:         "John Doe",
			Language:      "en",
			Given:         "John",
			Surname:       "Doe",
			Prefix:        "Dr",
			Suffix:        "Jr",
			Nickname:      "Johnny",
			SurnamePrefix: "van",
		}

		copied := cloneTransliteration(original)
		if copied == original {
			t.Error("Should be a copy, not the same pointer")
		}
		if copied.Value != original.Value || copied.Language != original.Language || copied.Given != original.Given || copied.Surname != original.Surname {
			t.Error("Field mismatch after clone")
		}
		if copied.Prefix != original.Prefix || copied.Suffix != original.Suffix || copied.Nickname != original.Nickname || copied.SurnamePrefix != original.SurnamePrefix {
			t.Error("Field mismatch after clone")
		}
	})
}

func TestCloneAncestryAPID(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if cloneAncestryAPID(nil) != nil {
			t.Error("Expected nil for nil input")
		}
	})

	t.Run("full", func(t *testing.T) {
		original := &AncestryAPID{Raw: "1234:5678:90", Database: "1234", Record: "5678"}
		copied := cloneAncestryAPID(original)
		if copied == original {
			t.Error("Should be a copy")
		}
		if copied.Raw != original.Raw || copied.Database != original.Database || copied.Record != original.Record {
			t.Error("Field mismatch after clone")
		}
	})
}

func TestCloneAssociationWithCitations(t *testing.T) {
	original := &Association{
		IndividualXRef: "@I2@",
		Role:           "Witness",
		Phrase:         "witness to the event",
		Notes:          []string{"Note 1", "Note 2"},
		SourceCitations: []*SourceCitation{
			{SourceXRef: "@S1@", Page: "Page 10"},
			{SourceXRef: "@S2@", Page: "Page 20"},
		},
	}

	copied := cloneAssociation(original)
	if copied.IndividualXRef != original.IndividualXRef {
		t.Errorf("IndividualXRef mismatch")
	}
	if copied.Role != original.Role || copied.Phrase != original.Phrase {
		t.Errorf("Role/Phrase mismatch")
	}
	if len(copied.Notes) != len(original.Notes) {
		t.Errorf("Notes len = %d, want %d", len(copied.Notes), len(original.Notes))
	}
	if len(copied.SourceCitations) != len(original.SourceCitations) {
		t.Errorf("SourceCitations len = %d, want %d", len(copied.SourceCitations), len(original.SourceCitations))
	}
	original.SourceCitations[0].Page = "Modified"
	if copied.SourceCitations[0].Page == "Modified" {
		t.Error("Should be a deep copy")
	}
}

func TestCloneAttributeWithCitations(t *testing.T) {
	original := &Attribute{
		Type:            "OCCU",
		Value:           "Farmer",
		Date:            "1900",
		Place:           "Iowa",
		ParsedDate:      &Date{Original: "1900", Year: 1900},
		SourceCitations: []*SourceCitation{{SourceXRef: "@S1@", Page: "Page 5"}},
	}

	copied := cloneAttribute(original)
	if copied.Type != original.Type || copied.Value != original.Value || copied.Date != original.Date || copied.Place != original.Place {
		t.Error("Field mismatch")
	}
	if copied.ParsedDate == nil {
		t.Error("ParsedDate should not be nil")
	}
	original.SourceCitations[0].Page = "Modified"
	if copied.SourceCitations[0].Page == "Modified" {
		t.Error("Should be a deep copy")
	}
}

func TestClonePersonalNameFull(t *testing.T) {
	original := &PersonalName{
		Full:          "Dr. John /van Doe/ Jr.",
		Given:         "John",
		Surname:       "Doe",
		Prefix:        "Dr.",
		Suffix:        "Jr.",
		Nickname:      "Johnny",
		SurnamePrefix: "van",
		Type:          "birth",
		Transliterations: []*Transliteration{
			{Value: "John Doe", Language: "en"},
			{Value: "Jon Do", Language: "en-phonetic"},
		},
	}

	copied := clonePersonalName(original)
	if copied.Full != original.Full || copied.Given != original.Given || copied.Surname != original.Surname {
		t.Error("Field mismatch")
	}
	if copied.Prefix != original.Prefix || copied.Suffix != original.Suffix || copied.Nickname != original.Nickname {
		t.Error("Field mismatch")
	}
	if copied.SurnamePrefix != original.SurnamePrefix || copied.Type != original.Type {
		t.Error("Field mismatch")
	}
	if len(copied.Transliterations) != len(original.Transliterations) {
		t.Errorf("Transliterations len = %d, want %d", len(copied.Transliterations), len(original.Transliterations))
	}
	original.Transliterations[0].Value = "Modified"
	if copied.Transliterations[0].Value == "Modified" {
		t.Error("Should be a deep copy")
	}
}

func TestCloneMediaLinkWithCrop(t *testing.T) {
	original := &MediaLink{
		MediaXRef: "@M1@",
		Title:     "Photo",
		Crop:      &CropRegion{Height: 100, Left: 10, Top: 20, Width: 200},
	}

	copied := cloneMediaLink(original)
	if copied.MediaXRef != original.MediaXRef || copied.Title != original.Title {
		t.Error("Field mismatch")
	}
	if copied.Crop == nil {
		t.Fatal("Crop should not be nil")
	}
	if copied.Crop.Height != original.Crop.Height || copied.Crop.Left != original.Crop.Left {
		t.Error("Crop field mismatch")
	}
	if copied.Crop.Top != original.Crop.Top || copied.Crop.Width != original.Crop.Width {
		t.Error("Crop field mismatch")
	}
	if copied.Crop == original.Crop {
		t.Error("Crop should be a deep copy")
	}
}

func TestCloneSourceCitationFull(t *testing.T) {
	original := &SourceCitation{
		SourceXRef:   "@S1@",
		Page:         "Page 123",
		Quality:      2,
		AncestryAPID: &AncestryAPID{Raw: "1:2:3", Database: "1", Record: "2"},
	}

	copied := cloneSourceCitation(original)
	if copied.SourceXRef != original.SourceXRef || copied.Page != original.Page {
		t.Error("Field mismatch")
	}
	if copied.Quality != original.Quality {
		t.Errorf("Quality = %v, want %v", copied.Quality, original.Quality)
	}
	if copied.AncestryAPID == nil {
		t.Fatal("AncestryAPID should not be nil")
	}
	if copied.AncestryAPID == original.AncestryAPID {
		t.Error("AncestryAPID should be a deep copy")
	}
}

func TestCloneLDSOrdinanceFull(t *testing.T) {
	original := &LDSOrdinance{
		Type:       "BAPL",
		Date:       "1 JAN 2000",
		Temple:     "SLAKE",
		Place:      "Salt Lake City",
		Status:     "COMPLETED",
		FamilyXRef: "@F1@",
	}

	copied := cloneLDSOrdinance(original)
	if string(copied.Type) != string(original.Type) {
		t.Errorf("Type mismatch")
	}
	if copied.Date != original.Date || copied.Temple != original.Temple {
		t.Error("Field mismatch")
	}
	if copied.Place != original.Place || copied.Status != original.Status || copied.FamilyXRef != original.FamilyXRef {
		t.Error("Field mismatch")
	}
}

func TestCloneMediaFileFull(t *testing.T) {
	original := &MediaFile{
		FileRef:      "/path/to/file.jpg",
		Form:         "image/jpeg",
		MediaType:    "photo",
		Title:        "Family Photo",
		Translations: []*MediaTranslation{{FileRef: "/path/to/file.png", Form: "image/png"}},
	}

	copied := cloneMediaFile(original)
	if copied.FileRef != original.FileRef || copied.Form != original.Form {
		t.Error("Field mismatch")
	}
	if copied.MediaType != original.MediaType || copied.Title != original.Title {
		t.Error("Field mismatch")
	}
	if len(copied.Translations) != len(original.Translations) {
		t.Errorf("Translations len = %d, want %d", len(copied.Translations), len(original.Translations))
	}
}

func TestRepositoryClone(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		var r *Repository
		if r.Clone() != nil {
			t.Error("(*Repository)(nil).Clone() should return nil")
		}
	})

	t.Run("deep copies address and notes", func(t *testing.T) {
		original := &Repository{
			XRef:    "@R1@",
			Name:    "Repo",
			Address: &Address{Line1: "1 Main St"},
			Notes:   []string{"@N1@"},
			Tags:    []*Tag{{Tag: "CUSTOM"}},
		}
		copied := original.Clone()
		if copied.Address == original.Address {
			t.Error("Address should be deep copied")
		}
		copied.Notes[0] = "modified"
		if original.Notes[0] == "modified" {
			t.Error("Notes was not deep copied")
		}
	})
}

func TestNoteClone(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		var n *Note
		if n.Clone() != nil {
			t.Error("(*Note)(nil).Clone() should return nil")
		}
	})

	t.Run("deep copies continuation", func(t *testing.T) {
		original := &Note{
			XRef:         "@N1@",
			Text:         "Hello",
			Continuation: []string{"line2", "line3"},
		}
		copied := original.Clone()
		copied.Continuation[0] = "modified"
		if original.Continuation[0] == "modified" {
			t.Error("Continuation was not deep copied")
		}
	})
}

func createFullTestDocument() *Document {
	ind := &Individual{
		XRef:   "@I1@",
		Sex:    "M",
		Names:  []*PersonalName{{Full: "John /Doe/", Given: "John", Surname: "Doe"}},
		Events: []*Event{{Type: "BIRT", Date: "1 JAN 1900"}},
	}

	fam := &Family{
		XRef:     "@F1@",
		Husband:  "@I1@",
		Wife:     "@I2@",
		Children: []string{"@I3@"},
	}

	doc := &Document{
		Header:  &Header{Version: Version551, Encoding: EncodingUTF8, SourceSystem: "TestSystem"},
		Trailer: &Trailer{LineNumber: 100},
		Vendor:  "TestVendor",
		Records: []*Record{
			{
				XRef:   "@I1@",
				Type:   RecordTypeIndividual,
				Entity: ind,
				Tags: []*Tag{
					{Level: 0, Tag: "INDI", XRef: "@I1@"},
					{Level: 1, Tag: "NAME", Value: "John /Doe/"},
				},
			},
			{
				XRef:   "@F1@",
				Type:   RecordTypeFamily,
				Entity: fam,
				Tags:   []*Tag{{Level: 0, Tag: "FAM", XRef: "@F1@"}},
			},
		},
		XRefMap: make(map[string]*Record),
	}

	doc.XRefMap["@I1@"] = doc.Records[0]
	doc.XRefMap["@F1@"] = doc.Records[1]

	return doc
}
