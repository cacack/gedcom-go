package validator

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// newNoteTestDocument builds a document holding one media object record with the
// given note pointer slices, plus an individual so the document is not media-only.
func newNoteTestDocument(media ...*gedcom.MediaObject) *gedcom.Document {
	doc := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version70},
		XRefMap: make(map[string]*gedcom.Record),
	}
	for _, m := range media {
		record := &gedcom.Record{
			XRef:   m.XRef,
			Type:   gedcom.RecordTypeMedia,
			Entity: m,
		}
		doc.Records = append(doc.Records, record)
		doc.XRefMap[m.XRef] = record
	}
	return doc
}

func TestNewNoteValidator(t *testing.T) {
	if NewNoteValidator() == nil {
		t.Error("NewNoteValidator() returned nil")
	}
}

func TestNoteValidator_ValidateNotePointers(t *testing.T) {
	tests := []struct {
		name            string
		media           *gedcom.MediaObject
		wantOverlapping []string // xrefs expected to be reported, in order
	}{
		{
			name: "single overlapping pointer",
			media: &gedcom.MediaObject{
				XRef:            "@O1@",
				NoteXRefs:       []string{"@N1@", "@N2@"},
				SharedNoteXRefs: []string{"@N2@"},
			},
			wantOverlapping: []string{"@N2@"},
		},
		{
			name: "multiple overlapping pointers",
			media: &gedcom.MediaObject{
				XRef:            "@O1@",
				NoteXRefs:       []string{"@N1@", "@N2@", "@N3@"},
				SharedNoteXRefs: []string{"@N1@", "@N3@"},
			},
			wantOverlapping: []string{"@N1@", "@N3@"},
		},
		{
			name: "v2 decoder shape repeats every shared note pointer",
			// The pre-partition decoder appended each SNOTE pointer to both
			// slices; such a value reports one issue per shared pointer. A
			// current decode reaches the same state from an OBJE carrying both
			// NOTE @N@ and SNOTE @N@, so this is not only a migration artifact.
			media: &gedcom.MediaObject{
				XRef:            "@O1@",
				NoteXRefs:       []string{"@N1@", "@S1@", "@S2@"},
				SharedNoteXRefs: []string{"@S1@", "@S2@"},
			},
			wantOverlapping: []string{"@S1@", "@S2@"},
		},
		{
			name: "disjoint slices are the documented shape",
			media: &gedcom.MediaObject{
				XRef:            "@O1@",
				NoteXRefs:       []string{"@N1@", "@N2@"},
				SharedNoteXRefs: []string{"@S1@", "@S2@"},
			},
			wantOverlapping: nil,
		},
		{
			name: "pointer repeated within NoteXRefs only is not an overlap",
			media: &gedcom.MediaObject{
				XRef:            "@O1@",
				NoteXRefs:       []string{"@N1@", "@N1@"},
				SharedNoteXRefs: []string{"@S1@"},
			},
			wantOverlapping: nil,
		},
		{
			name: "overlapping pointer repeated in SharedNoteXRefs reports once",
			media: &gedcom.MediaObject{
				XRef:            "@O1@",
				NoteXRefs:       []string{"@N1@"},
				SharedNoteXRefs: []string{"@N1@", "@N1@"},
			},
			wantOverlapping: []string{"@N1@"},
		},
		{
			name: "both slices empty",
			media: &gedcom.MediaObject{
				XRef: "@O1@",
			},
			wantOverlapping: nil,
		},
		{
			name: "only NoteXRefs populated",
			media: &gedcom.MediaObject{
				XRef:      "@O1@",
				NoteXRefs: []string{"@N1@"},
			},
			wantOverlapping: nil,
		},
		{
			name: "only SharedNoteXRefs populated",
			media: &gedcom.MediaObject{
				XRef:            "@O1@",
				SharedNoteXRefs: []string{"@S1@"},
			},
			wantOverlapping: nil,
		},
	}

	v := NewNoteValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := v.ValidateNotePointers(newNoteTestDocument(tt.media))

			if len(issues) != len(tt.wantOverlapping) {
				t.Fatalf("got %d issues, want %d: %v", len(issues), len(tt.wantOverlapping), issues)
			}

			for i, wantXRef := range tt.wantOverlapping {
				issue := issues[i]
				if issue.Code != CodeOverlappingNotePointers {
					t.Errorf("issue[%d].Code = %q, want %q", i, issue.Code, CodeOverlappingNotePointers)
				}
				if issue.Severity != SeverityWarning {
					t.Errorf("issue[%d].Severity = %v, want %v", i, issue.Severity, SeverityWarning)
				}
				if issue.RecordXRef != tt.media.XRef {
					t.Errorf("issue[%d].RecordXRef = %q, want %q", i, issue.RecordXRef, tt.media.XRef)
				}
				if got := issue.Details["xref"]; got != wantXRef {
					t.Errorf("issue[%d].Details[\"xref\"] = %q, want %q", i, got, wantXRef)
				}
				// The message must name the offending pointer; a caller reading
				// only the rendered issue has nothing else to act on.
				if !strings.Contains(issue.Message, wantXRef) {
					t.Errorf("issue[%d].Message = %q, want it to mention %q", i, issue.Message, wantXRef)
				}
				// Typed note fields carry no source line; 0 must not be
				// rendered as a location (see Issue.LineNumber).
				if issue.LineNumber != 0 {
					t.Errorf("issue[%d].LineNumber = %d, want 0", i, issue.LineNumber)
				}
			}
		})
	}
}

func TestNoteValidator_ValidateNotePointers_NilDocument(t *testing.T) {
	if issues := NewNoteValidator().ValidateNotePointers(nil); issues != nil {
		t.Errorf("ValidateNotePointers(nil) = %v, want nil", issues)
	}
}

func TestNoteValidator_ValidateNotePointers_MultipleRecords(t *testing.T) {
	doc := newNoteTestDocument(
		&gedcom.MediaObject{
			XRef:            "@O1@",
			NoteXRefs:       []string{"@N1@"},
			SharedNoteXRefs: []string{"@N1@"},
		},
		&gedcom.MediaObject{
			XRef:            "@O2@",
			NoteXRefs:       []string{"@N2@"},
			SharedNoteXRefs: []string{"@S2@"},
		},
		&gedcom.MediaObject{
			XRef:            "@O3@",
			NoteXRefs:       []string{"@N3@"},
			SharedNoteXRefs: []string{"@N3@"},
		},
	)

	issues := NewNoteValidator().ValidateNotePointers(doc)
	if len(issues) != 2 {
		t.Fatalf("got %d issues, want 2: %v", len(issues), issues)
	}
	if issues[0].RecordXRef != "@O1@" || issues[1].RecordXRef != "@O3@" {
		t.Errorf("reported records = %q, %q; want @O1@, @O3@", issues[0].RecordXRef, issues[1].RecordXRef)
	}
}

// A hand-built document can hold a nil *Record or a non-media entity where a
// media object is expected. Neither may panic (ADR 0007).
func TestNoteValidator_ValidateNotePointers_NilSafety(t *testing.T) {
	doc := newNoteTestDocument(&gedcom.MediaObject{
		XRef:            "@O1@",
		NoteXRefs:       []string{"@N1@"},
		SharedNoteXRefs: []string{"@N1@"},
	})
	doc.Records = append(doc.Records,
		nil,
		&gedcom.Record{XRef: "@O2@", Type: gedcom.RecordTypeMedia, Entity: nil},
		&gedcom.Record{XRef: "@O3@", Type: gedcom.RecordTypeMedia, Entity: (*gedcom.MediaObject)(nil)},
		&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "@I1@"}},
	)

	issues := NewNoteValidator().ValidateNotePointers(doc)
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1 (the nils must be skipped, not reported): %v", len(issues), issues)
	}
}

func TestValidator_ValidateNotePointers_NilDocument(t *testing.T) {
	if issues := New().ValidateNotePointers(nil); issues != nil {
		t.Errorf("ValidateNotePointers(nil) = %v, want nil", issues)
	}
}

func TestValidator_ValidateNotePointers_Strictness(t *testing.T) {
	overlapping := func() *gedcom.Document {
		return newNoteTestDocument(&gedcom.MediaObject{
			XRef:            "@O1@",
			NoteXRefs:       []string{"@N1@"},
			SharedNoteXRefs: []string{"@N1@"},
		})
	}

	tests := []struct {
		name       string
		opts       *ValidateOptions
		wantIssues int
	}{
		{
			name:       "normal strictness keeps the warning",
			opts:       &ValidateOptions{Strictness: StrictnessNormal},
			wantIssues: 1,
		},
		{
			name:       "strict strictness keeps the warning",
			opts:       &ValidateOptions{Strictness: StrictnessStrict},
			wantIssues: 1,
		},
		{
			name:       "relaxed strictness drops the warning",
			opts:       &ValidateOptions{Strictness: StrictnessRelaxed},
			wantIssues: 0,
		},
		{
			name: "skip rules suppress the code",
			opts: &ValidateOptions{
				Strictness: StrictnessNormal,
				SkipRules:  []string{CodeOverlappingNotePointers},
			},
			wantIssues: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := NewWithOptions(tt.opts).ValidateNotePointers(overlapping())
			if len(issues) != tt.wantIssues {
				t.Errorf("got %d issues, want %d: %v", len(issues), tt.wantIssues, issues)
			}
		})
	}
}

// ValidateAll must surface the code, so a caller running the full suite sees the
// violated partition without opting into a specific check.
func TestValidator_ValidateAll_ReportsOverlappingNotePointers(t *testing.T) {
	doc := newNoteTestDocument(&gedcom.MediaObject{
		XRef:            "@O1@",
		NoteXRefs:       []string{"@N1@"},
		SharedNoteXRefs: []string{"@N1@"},
	})

	found := 0
	for _, issue := range New().ValidateAll(doc) {
		if issue.Code == CodeOverlappingNotePointers {
			found++
		}
	}
	if found != 1 {
		t.Errorf("ValidateAll reported %d %s issues, want 1", found, CodeOverlappingNotePointers)
	}
}
