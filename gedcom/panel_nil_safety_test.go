package gedcom

import "testing"

// Clone builds translation elements inline instead of delegating to a
// nil-guarded clone helper the way every other slice field does, so a nil
// element there dereferenced rather than being skipped. Convert clones, so this
// was reachable from the converter too.
func TestClone_NilTranslationElement(t *testing.T) {
	doc := &Document{
		Header: &Header{Version: Version70},
		Records: []*Record{
			{
				XRef: "@X1@", Type: RecordTypeSharedNote,
				Entity: &SharedNote{XRef: "@X1@", Translations: []*SharedNoteTranslation{
					nil, {Value: "bonjour", Language: "fr"},
				}},
			},
			{
				XRef: "@O1@", Type: RecordTypeMedia,
				Entity: &MediaObject{XRef: "@O1@", Files: []*MediaFile{
					{FileRef: "a.jpg", Form: "image/jpeg", Translations: []*MediaTranslation{
						nil, {FileRef: "b.jpg", Form: "image/png"},
					}},
				}},
			},
		},
	}

	clone := doc.Clone()

	snote, ok := clone.GetRecord("@X1@").GetSharedNote()
	if !ok {
		t.Fatal("cloned shared note missing")
	}
	if len(snote.Translations) != 2 {
		t.Fatalf("shared note translations = %d, want 2 (the nil must be preserved)", len(snote.Translations))
	}
	if snote.Translations[0] != nil {
		t.Errorf("shared note translation[0] = %#v, want nil", snote.Translations[0])
	}
	if snote.Translations[1] == nil || snote.Translations[1].Value != "bonjour" {
		t.Errorf("shared note translation[1] = %#v, want the real translation", snote.Translations[1])
	}

	media, ok := clone.GetRecord("@O1@").GetMediaObject()
	if !ok {
		t.Fatal("cloned media object missing")
	}
	trans := media.Files[0].Translations
	if len(trans) != 2 {
		t.Fatalf("media translations = %d, want 2 (the nil must be preserved)", len(trans))
	}
	if trans[0] != nil {
		t.Errorf("media translation[0] = %#v, want nil", trans[0])
	}
	if trans[1] == nil || trans[1].FileRef != "b.jpg" {
		t.Errorf("media translation[1] = %#v, want the real translation", trans[1])
	}
}
