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

	t.Run("GetSubmitter", func(t *testing.T) {
		subm := &Submitter{XRef: "@U1@"}
		record := &Record{
			Type:   RecordTypeSubmitter,
			Entity: subm,
		}

		got, ok := record.GetSubmitter()
		if !ok {
			t.Error("Should get submitter")
		}
		if got != subm {
			t.Error("Should return same submitter")
		}

		// Wrong type
		indRecord := &Record{Type: RecordTypeIndividual}
		_, ok = indRecord.GetSubmitter()
		if ok {
			t.Error("Should not get submitter from individual record")
		}
	})

	t.Run("GetRepository", func(t *testing.T) {
		repo := &Repository{XRef: "@R1@"}
		record := &Record{
			Type:   RecordTypeRepository,
			Entity: repo,
		}

		got, ok := record.GetRepository()
		if !ok {
			t.Error("Should get repository")
		}
		if got != repo {
			t.Error("Should return same repository")
		}

		// Wrong type
		indRecord := &Record{Type: RecordTypeIndividual}
		_, ok = indRecord.GetRepository()
		if ok {
			t.Error("Should not get repository from individual record")
		}
	})

	t.Run("GetNote", func(t *testing.T) {
		note := &Note{XRef: "@N1@", Text: "Test note"}
		record := &Record{
			Type:   RecordTypeNote,
			Entity: note,
		}

		got, ok := record.GetNote()
		if !ok {
			t.Error("Should get note")
		}
		if got != note {
			t.Error("Should return same note")
		}

		// Wrong type
		indRecord := &Record{Type: RecordTypeIndividual}
		_, ok = indRecord.GetNote()
		if ok {
			t.Error("Should not get note from individual record")
		}
	})

	t.Run("GetMediaObject", func(t *testing.T) {
		media := &MediaObject{XRef: "@M1@"}
		record := &Record{
			Type:   RecordTypeMedia,
			Entity: media,
		}

		got, ok := record.GetMediaObject()
		if !ok {
			t.Error("Should get media object")
		}
		if got != media {
			t.Error("Should return same media object")
		}

		// Wrong type
		indRecord := &Record{Type: RecordTypeIndividual}
		_, ok = indRecord.GetMediaObject()
		if ok {
			t.Error("Should not get media object from individual record")
		}
	})
}

func TestDocument(t *testing.T) {
	ind1 := &Individual{XRef: "@I1@"}
	ind2 := &Individual{XRef: "@I2@"}
	fam1 := &Family{XRef: "@F1@"}
	src1 := &Source{XRef: "@S1@"}
	subm1 := &Submitter{XRef: "@U1@"}
	repo1 := &Repository{XRef: "@R1@"}
	note1 := &Note{XRef: "@N1@", Text: "Test note"}
	media1 := &MediaObject{XRef: "@M1@"}

	doc := &Document{
		Records: []*Record{
			{XRef: "@I1@", Type: RecordTypeIndividual, Entity: ind1},
			{XRef: "@I2@", Type: RecordTypeIndividual, Entity: ind2},
			{XRef: "@F1@", Type: RecordTypeFamily, Entity: fam1},
			{XRef: "@S1@", Type: RecordTypeSource, Entity: src1},
			{XRef: "@U1@", Type: RecordTypeSubmitter, Entity: subm1},
			{XRef: "@R1@", Type: RecordTypeRepository, Entity: repo1},
			{XRef: "@N1@", Type: RecordTypeNote, Entity: note1},
			{XRef: "@M1@", Type: RecordTypeMedia, Entity: media1},
		},
		XRefMap: map[string]*Record{
			"@I1@": {XRef: "@I1@", Type: RecordTypeIndividual, Entity: ind1},
			"@I2@": {XRef: "@I2@", Type: RecordTypeIndividual, Entity: ind2},
			"@F1@": {XRef: "@F1@", Type: RecordTypeFamily, Entity: fam1},
			"@S1@": {XRef: "@S1@", Type: RecordTypeSource, Entity: src1},
			"@U1@": {XRef: "@U1@", Type: RecordTypeSubmitter, Entity: subm1},
			"@R1@": {XRef: "@R1@", Type: RecordTypeRepository, Entity: repo1},
			"@N1@": {XRef: "@N1@", Type: RecordTypeNote, Entity: note1},
			"@M1@": {XRef: "@M1@", Type: RecordTypeMedia, Entity: media1},
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

	t.Run("GetSubmitter", func(t *testing.T) {
		subm := doc.GetSubmitter("@U1@")
		if subm == nil {
			t.Fatal("Should find submitter")
		}
		if subm.XRef != "@U1@" {
			t.Errorf("Got XRef %s, want @U1@", subm.XRef)
		}

		// Try to get submitter from individual XRef
		wrongType := doc.GetSubmitter("@I1@")
		if wrongType != nil {
			t.Error("Should not get submitter from individual XRef")
		}

		// Non-existent XRef
		notFound := doc.GetSubmitter("@U999@")
		if notFound != nil {
			t.Error("Should return nil for non-existent XRef")
		}
	})

	t.Run("Submitters", func(t *testing.T) {
		submitters := doc.Submitters()
		if len(submitters) != 1 {
			t.Errorf("Got %d submitters, want 1", len(submitters))
		}
	})

	t.Run("GetRepository", func(t *testing.T) {
		repo := doc.GetRepository("@R1@")
		if repo == nil {
			t.Fatal("Should find repository")
		}
		if repo.XRef != "@R1@" {
			t.Errorf("Got XRef %s, want @R1@", repo.XRef)
		}

		// Try to get repository from individual XRef
		wrongType := doc.GetRepository("@I1@")
		if wrongType != nil {
			t.Error("Should not get repository from individual XRef")
		}

		// Non-existent XRef
		notFound := doc.GetRepository("@R999@")
		if notFound != nil {
			t.Error("Should return nil for non-existent XRef")
		}
	})

	t.Run("Repositories", func(t *testing.T) {
		repositories := doc.Repositories()
		if len(repositories) != 1 {
			t.Errorf("Got %d repositories, want 1", len(repositories))
		}
	})

	t.Run("GetNote", func(t *testing.T) {
		note := doc.GetNote("@N1@")
		if note == nil {
			t.Fatal("Should find note")
		}
		if note.XRef != "@N1@" {
			t.Errorf("Got XRef %s, want @N1@", note.XRef)
		}

		// Try to get note from individual XRef
		wrongType := doc.GetNote("@I1@")
		if wrongType != nil {
			t.Error("Should not get note from individual XRef")
		}

		// Non-existent XRef
		notFound := doc.GetNote("@N999@")
		if notFound != nil {
			t.Error("Should return nil for non-existent XRef")
		}
	})

	t.Run("Notes", func(t *testing.T) {
		notes := doc.Notes()
		if len(notes) != 1 {
			t.Errorf("Got %d notes, want 1", len(notes))
		}
	})

	t.Run("GetMediaObject", func(t *testing.T) {
		media := doc.GetMediaObject("@M1@")
		if media == nil {
			t.Fatal("Should find media object")
		}
		if media.XRef != "@M1@" {
			t.Errorf("Got XRef %s, want @M1@", media.XRef)
		}

		// Try to get media object from individual XRef
		wrongType := doc.GetMediaObject("@I1@")
		if wrongType != nil {
			t.Error("Should not get media object from individual XRef")
		}

		// Non-existent XRef
		notFound := doc.GetMediaObject("@M999@")
		if notFound != nil {
			t.Error("Should return nil for non-existent XRef")
		}
	})

	t.Run("MediaObjects", func(t *testing.T) {
		objects := doc.MediaObjects()
		if len(objects) != 1 {
			t.Errorf("Got %d media objects, want 1", len(objects))
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
