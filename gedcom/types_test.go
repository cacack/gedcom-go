package gedcom

import (
	"testing"
)

func TestVersion(t *testing.T) {
	tests := []struct {
		name    string
		version Version
		want    bool
	}{
		{"5.5 is valid", Version55, true},
		{"5.5.1 is valid", Version551, true},
		{"7.0 is valid", Version70, true},
		{"invalid version", Version("999"), false},
		{"empty version", Version(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.version.IsValid(); got != tt.want {
				t.Errorf("Version.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersionString(t *testing.T) {
	tests := []struct {
		version Version
		want    string
	}{
		{Version55, "5.5"},
		{Version551, "5.5.1"},
		{Version70, "7.0"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.version.String(); got != tt.want {
				t.Errorf("Version.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEncoding(t *testing.T) {
	tests := []struct {
		name     string
		encoding Encoding
		want     bool
	}{
		{"UTF-8 is valid", EncodingUTF8, true},
		{"ANSEL is valid", EncodingANSEL, true},
		{"ASCII is valid", EncodingASCII, true},
		{"LATIN1 is valid", EncodingLATIN1, true},
		{"UNICODE is valid", EncodingUNICODE, true},
		{"invalid encoding", Encoding("EBCDIC"), false},
		{"empty encoding", Encoding(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.encoding.IsValid(); got != tt.want {
				t.Errorf("Encoding.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEncodingString(t *testing.T) {
	tests := []struct {
		encoding Encoding
		want     string
	}{
		{EncodingUTF8, "UTF-8"},
		{EncodingANSEL, "ANSEL"},
		{EncodingASCII, "ASCII"},
		{EncodingLATIN1, "LATIN1"},
		{EncodingUNICODE, "UNICODE"},
		{Encoding("CUSTOM"), "CUSTOM"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.encoding.String(); got != tt.want {
				t.Errorf("Encoding.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTag(t *testing.T) {
	t.Run("HasValue", func(t *testing.T) {
		tag := &Tag{Level: 1, Tag: "NAME", Value: "John /Doe/"}
		if !tag.HasValue() {
			t.Error("Tag should have value")
		}

		emptyTag := &Tag{Level: 1, Tag: "NAME"}
		if emptyTag.HasValue() {
			t.Error("Tag should not have value")
		}
	})

	t.Run("HasXRef", func(t *testing.T) {
		tag := &Tag{Level: 0, Tag: "INDI", XRef: "@I1@"}
		if !tag.HasXRef() {
			t.Error("Tag should have XRef")
		}

		noXRefTag := &Tag{Level: 1, Tag: "NAME"}
		if noXRefTag.HasXRef() {
			t.Error("Tag should not have XRef")
		}
	})
}

func TestRecord(t *testing.T) {
	t.Run("IsIndividual", func(t *testing.T) {
		record := &Record{Type: RecordTypeIndividual}
		if !record.IsIndividual() {
			t.Error("Record should be individual")
		}

		famRecord := &Record{Type: RecordTypeFamily}
		if famRecord.IsIndividual() {
			t.Error("Record should not be individual")
		}
	})

	t.Run("IsFamily", func(t *testing.T) {
		record := &Record{Type: RecordTypeFamily}
		if !record.IsFamily() {
			t.Error("Record should be family")
		}
	})

	t.Run("IsSource", func(t *testing.T) {
		record := &Record{Type: RecordTypeSource}
		if !record.IsSource() {
			t.Error("Record should be source")
		}
	})

	t.Run("GetIndividual", func(t *testing.T) {
		ind := &Individual{XRef: "@I1@"}
		record := &Record{
			Type:   RecordTypeIndividual,
			Entity: ind,
		}

		got, ok := record.GetIndividual()
		if !ok {
			t.Error("Should get individual")
		}
		if got != ind {
			t.Error("Should return same individual")
		}

		// Wrong type
		famRecord := &Record{Type: RecordTypeFamily}
		_, ok = famRecord.GetIndividual()
		if ok {
			t.Error("Should not get individual from family record")
		}
	})

	t.Run("GetFamily", func(t *testing.T) {
		fam := &Family{XRef: "@F1@"}
		record := &Record{
			Type:   RecordTypeFamily,
			Entity: fam,
		}

		got, ok := record.GetFamily()
		if !ok {
			t.Error("Should get family")
		}
		if got != fam {
			t.Error("Should return same family")
		}

		// Wrong type
		indRecord := &Record{Type: RecordTypeIndividual}
		_, ok = indRecord.GetFamily()
		if ok {
			t.Error("Should not get family from individual record")
		}
	})

	t.Run("GetSource", func(t *testing.T) {
		src := &Source{XRef: "@S1@"}
		record := &Record{
			Type:   RecordTypeSource,
			Entity: src,
		}

		got, ok := record.GetSource()
		if !ok {
			t.Error("Should get source")
		}
		if got != src {
			t.Error("Should return same source")
		}

		// Wrong type
		indRecord := &Record{Type: RecordTypeIndividual}
		_, ok = indRecord.GetSource()
		if ok {
			t.Error("Should not get source from individual record")
		}
	})
}

func TestDocument(t *testing.T) {
	ind1 := &Individual{XRef: "@I1@"}
	ind2 := &Individual{XRef: "@I2@"}
	fam1 := &Family{XRef: "@F1@"}
	src1 := &Source{XRef: "@S1@"}

	doc := &Document{
		Records: []*Record{
			{XRef: "@I1@", Type: RecordTypeIndividual, Entity: ind1},
			{XRef: "@I2@", Type: RecordTypeIndividual, Entity: ind2},
			{XRef: "@F1@", Type: RecordTypeFamily, Entity: fam1},
			{XRef: "@S1@", Type: RecordTypeSource, Entity: src1},
		},
		XRefMap: map[string]*Record{
			"@I1@": {XRef: "@I1@", Type: RecordTypeIndividual, Entity: ind1},
			"@I2@": {XRef: "@I2@", Type: RecordTypeIndividual, Entity: ind2},
			"@F1@": {XRef: "@F1@", Type: RecordTypeFamily, Entity: fam1},
			"@S1@": {XRef: "@S1@", Type: RecordTypeSource, Entity: src1},
		},
	}

	t.Run("GetRecord", func(t *testing.T) {
		record := doc.GetRecord("@I1@")
		if record == nil {
			t.Fatal("Should find record")
		}
		if record.XRef != "@I1@" {
			t.Errorf("Got XRef %s, want @I1@", record.XRef)
		}

		notFound := doc.GetRecord("@I999@")
		if notFound != nil {
			t.Error("Should not find non-existent record")
		}
	})

	t.Run("GetRecord with nil XRefMap", func(t *testing.T) {
		emptyDoc := &Document{}
		record := emptyDoc.GetRecord("@I1@")
		if record != nil {
			t.Error("Should return nil when XRefMap is nil")
		}
	})

	t.Run("GetIndividual", func(t *testing.T) {
		ind := doc.GetIndividual("@I1@")
		if ind == nil {
			t.Fatal("Should find individual")
		}
		if ind.XRef != "@I1@" {
			t.Errorf("Got XRef %s, want @I1@", ind.XRef)
		}

		// Try to get individual from family XRef
		wrongType := doc.GetIndividual("@F1@")
		if wrongType != nil {
			t.Error("Should not get individual from family XRef")
		}

		// Non-existent XRef
		notFound := doc.GetIndividual("@I999@")
		if notFound != nil {
			t.Error("Should return nil for non-existent XRef")
		}
	})

	t.Run("GetFamily", func(t *testing.T) {
		fam := doc.GetFamily("@F1@")
		if fam == nil {
			t.Fatal("Should find family")
		}
		if fam.XRef != "@F1@" {
			t.Errorf("Got XRef %s, want @F1@", fam.XRef)
		}

		// Try to get family from individual XRef
		wrongType := doc.GetFamily("@I1@")
		if wrongType != nil {
			t.Error("Should not get family from individual XRef")
		}

		// Non-existent XRef
		notFound := doc.GetFamily("@F999@")
		if notFound != nil {
			t.Error("Should return nil for non-existent XRef")
		}
	})

	t.Run("GetSource", func(t *testing.T) {
		src := doc.GetSource("@S1@")
		if src == nil {
			t.Fatal("Should find source")
		}
		if src.XRef != "@S1@" {
			t.Errorf("Got XRef %s, want @S1@", src.XRef)
		}

		// Try to get source from individual XRef
		wrongType := doc.GetSource("@I1@")
		if wrongType != nil {
			t.Error("Should not get source from individual XRef")
		}

		// Non-existent XRef
		notFound := doc.GetSource("@S999@")
		if notFound != nil {
			t.Error("Should return nil for non-existent XRef")
		}
	})

	t.Run("Individuals", func(t *testing.T) {
		individuals := doc.Individuals()
		if len(individuals) != 2 {
			t.Errorf("Got %d individuals, want 2", len(individuals))
		}
	})

	t.Run("Families", func(t *testing.T) {
		families := doc.Families()
		if len(families) != 1 {
			t.Errorf("Got %d families, want 1", len(families))
		}
	})

	t.Run("Sources", func(t *testing.T) {
		sources := doc.Sources()
		if len(sources) != 1 {
			t.Errorf("Got %d sources, want 1", len(sources))
		}
	})
}

func TestNoteFullText(t *testing.T) {
	t.Run("Single line", func(t *testing.T) {
		note := &Note{Text: "This is a note"}
		if got := note.FullText(); got != "This is a note" {
			t.Errorf("Got %q, want %q", got, "This is a note")
		}
	})

	t.Run("Multi-line", func(t *testing.T) {
		note := &Note{
			Text:         "Line 1",
			Continuation: []string{"Line 2", "Line 3"},
		}
		want := "Line 1\nLine 2\nLine 3"
		if got := note.FullText(); got != want {
			t.Errorf("Got %q, want %q", got, want)
		}
	})
}
