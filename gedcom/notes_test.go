package gedcom

import (
	"reflect"
	"testing"
)

// noteDoc builds a Document with one NOTE record (@N1@, multi-line) and one
// SNOTE record (@S1@) for resolution tests.
func noteDoc() *Document {
	noteRec := &Record{
		XRef: "@N1@",
		Type: RecordTypeNote,
		Entity: &Note{
			XRef:         "@N1@",
			Text:         "Shared note line 1",
			Continuation: []string{"line 2"},
		},
	}
	snoteRec := &Record{
		XRef: "@S1@",
		Type: RecordTypeSharedNote,
		Entity: &SharedNote{
			XRef: "@S1@",
			Text: "A shared SNOTE",
		},
	}
	return &Document{
		Records: []*Record{noteRec, snoteRec},
		XRefMap: map[string]*Record{
			"@N1@": noteRec,
			"@S1@": snoteRec,
		},
	}
}

func TestResolveSharedNoteText(t *testing.T) {
	doc := noteDoc()

	tests := []struct {
		name     string
		doc      *Document
		xref     string
		wantText string
		wantOK   bool
	}{
		{
			name:     "resolves NOTE record with continuation",
			doc:      doc,
			xref:     "@N1@",
			wantText: "Shared note line 1\nline 2",
			wantOK:   true,
		},
		{
			name:     "resolves SNOTE record",
			doc:      doc,
			xref:     "@S1@",
			wantText: "A shared SNOTE",
			wantOK:   true,
		},
		{
			name:     "unknown xref does not resolve",
			doc:      doc,
			xref:     "@N99@",
			wantText: "",
			wantOK:   false,
		},
		{
			name:     "nil document does not resolve",
			doc:      nil,
			xref:     "@N1@",
			wantText: "",
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, ok := resolveSharedNoteText(tt.doc, tt.xref)
			if ok != tt.wantOK {
				t.Fatalf("resolveSharedNoteText() ok = %v, want %v", ok, tt.wantOK)
			}
			if text != tt.wantText {
				t.Errorf("resolveSharedNoteText() text = %q, want %q", text, tt.wantText)
			}
		})
	}
}

func TestAllNotes(t *testing.T) {
	doc := noteDoc()

	tests := []struct {
		name   string
		inline []string
		xrefs  []string
		want   []string
	}{
		{
			name: "no notes returns nil",
			want: nil,
		},
		{
			name:   "inline only, in order",
			inline: []string{"first", "second"},
			want:   []string{"first", "second"},
		},
		{
			name:  "xrefs only, resolved in order",
			xrefs: []string{"@N1@", "@S1@"},
			want:  []string{"Shared note line 1\nline 2", "A shared SNOTE"},
		},
		{
			name:   "inline first then resolved xrefs",
			inline: []string{"inline note"},
			xrefs:  []string{"@S1@"},
			want:   []string{"inline note", "A shared SNOTE"},
		},
		{
			name:   "unresolved xref is skipped",
			inline: []string{"keep"},
			xrefs:  []string{"@MISSING@", "@N1@"},
			want:   []string{"keep", "Shared note line 1\nline 2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := allNotes(doc, tt.inline, tt.xrefs)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("allNotes() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// TestRecordAllNotes exercises the AllNotes method on each note-bearing record
// type, confirming they all delegate to allNotes with the same semantics.
func TestRecordAllNotes(t *testing.T) {
	doc := noteDoc()
	want := []string{"inline", "A shared SNOTE"}

	tests := []struct {
		name string
		got  []string
	}{
		{
			name: "Individual",
			got: (&Individual{
				InlineNotes: []string{"inline"},
				NoteXRefs:   []string{"@S1@"},
			}).AllNotes(doc),
		},
		{
			name: "Family",
			got: (&Family{
				InlineNotes: []string{"inline"},
				NoteXRefs:   []string{"@S1@"},
			}).AllNotes(doc),
		},
		{
			name: "Source",
			got: (&Source{
				InlineNotes: []string{"inline"},
				NoteXRefs:   []string{"@S1@"},
			}).AllNotes(doc),
		},
		{
			name: "Repository",
			got: (&Repository{
				InlineNotes: []string{"inline"},
				NoteXRefs:   []string{"@S1@"},
			}).AllNotes(doc),
		},
		{
			name: "Submitter",
			got: (&Submitter{
				InlineNotes: []string{"inline"},
				NoteXRefs:   []string{"@S1@"},
			}).AllNotes(doc),
		},
		{
			name: "MediaObject",
			got: (&MediaObject{
				InlineNotes: []string{"inline"},
				NoteXRefs:   []string{"@S1@"},
			}).AllNotes(doc),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.got, want) {
				t.Errorf("%s.AllNotes() = %#v, want %#v", tt.name, tt.got, want)
			}
		})
	}
}
