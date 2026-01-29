package converter

import "testing"

func TestRecordTypeLabel(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		want string
	}{
		{
			name: "INDI to Individual",
			tag:  "INDI",
			want: "Individual",
		},
		{
			name: "FAM to Family",
			tag:  "FAM",
			want: "Family",
		},
		{
			name: "SOUR to Source",
			tag:  "SOUR",
			want: "Source",
		},
		{
			name: "REPO to Repository",
			tag:  "REPO",
			want: "Repository",
		},
		{
			name: "NOTE to Note",
			tag:  "NOTE",
			want: "Note",
		},
		{
			name: "OBJE to MediaObject",
			tag:  "OBJE",
			want: "MediaObject",
		},
		{
			name: "SUBM to Submitter",
			tag:  "SUBM",
			want: "Submitter",
		},
		{
			name: "HEAD to Header",
			tag:  "HEAD",
			want: "Header",
		},
		{
			name: "TRLR to Trailer",
			tag:  "TRLR",
			want: "Trailer",
		},
		{
			name: "unknown tag passes through",
			tag:  "ZZZZ",
			want: "ZZZZ",
		},
		{
			name: "empty tag passes through",
			tag:  "",
			want: "",
		},
		{
			name: "custom extension tag passes through",
			tag:  "_CUSTOM",
			want: "_CUSTOM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RecordTypeLabel(tt.tag)
			if got != tt.want {
				t.Errorf("RecordTypeLabel(%q) = %q, want %q", tt.tag, got, tt.want)
			}
		})
	}
}

func TestBuildRecordPath(t *testing.T) {
	tests := []struct {
		name       string
		recordType string
		xref       string
		want       string
	}{
		{
			name:       "Individual with XRef",
			recordType: "INDI",
			xref:       "@I1@",
			want:       "Individual @I1@",
		},
		{
			name:       "Family with XRef",
			recordType: "FAM",
			xref:       "@F1@",
			want:       "Family @F1@",
		},
		{
			name:       "Header without XRef",
			recordType: "HEAD",
			xref:       "",
			want:       "Header",
		},
		{
			name:       "Trailer without XRef",
			recordType: "TRLR",
			xref:       "",
			want:       "Trailer",
		},
		{
			name:       "Source with XRef",
			recordType: "SOUR",
			xref:       "@S123@",
			want:       "Source @S123@",
		},
		{
			name:       "unknown record type with XRef",
			recordType: "UNKN",
			xref:       "@U1@",
			want:       "UNKN @U1@",
		},
		{
			name:       "unknown record type without XRef",
			recordType: "UNKN",
			xref:       "",
			want:       "UNKN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildRecordPath(tt.recordType, tt.xref)
			if got != tt.want {
				t.Errorf("BuildRecordPath(%q, %q) = %q, want %q", tt.recordType, tt.xref, got, tt.want)
			}
		})
	}
}

func TestAppendToPath(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		segment string
		want    string
	}{
		{
			name:    "append to record path",
			base:    "Individual @I1@",
			segment: "BIRT",
			want:    "Individual @I1@ > BIRT",
		},
		{
			name:    "append to nested path",
			base:    "Individual @I1@ > BIRT",
			segment: "DATE",
			want:    "Individual @I1@ > BIRT > DATE",
		},
		{
			name:    "append to empty base",
			base:    "",
			segment: "HEAD",
			want:    "HEAD",
		},
		{
			name:    "append empty segment",
			base:    "Header",
			segment: "",
			want:    "Header > ",
		},
		{
			name:    "header path",
			base:    "Header",
			segment: "CHAR",
			want:    "Header > CHAR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AppendToPath(tt.base, tt.segment)
			if got != tt.want {
				t.Errorf("AppendToPath(%q, %q) = %q, want %q", tt.base, tt.segment, got, tt.want)
			}
		})
	}
}

func TestBuildPath(t *testing.T) {
	tests := []struct {
		name     string
		segments []string
		want     string
	}{
		{
			name:     "single segment",
			segments: []string{"Individual @I1@"},
			want:     "Individual @I1@",
		},
		{
			name:     "two segments",
			segments: []string{"Individual @I1@", "BIRT"},
			want:     "Individual @I1@ > BIRT",
		},
		{
			name:     "three segments",
			segments: []string{"Individual @I1@", "BIRT", "DATE"},
			want:     "Individual @I1@ > BIRT > DATE",
		},
		{
			name:     "skips empty segments",
			segments: []string{"Individual @I1@", "", "BIRT", "", "DATE"},
			want:     "Individual @I1@ > BIRT > DATE",
		},
		{
			name:     "all empty segments",
			segments: []string{"", "", ""},
			want:     "",
		},
		{
			name:     "no segments",
			segments: []string{},
			want:     "",
		},
		{
			name:     "nil segments",
			segments: nil,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildPath(tt.segments...)
			if got != tt.want {
				t.Errorf("BuildPath(%v) = %q, want %q", tt.segments, got, tt.want)
			}
		})
	}
}

func TestBuildNestedPath(t *testing.T) {
	tests := []struct {
		name       string
		recordType string
		xref       string
		tags       []string
		want       string
	}{
		{
			name:       "Individual with nested tags",
			recordType: "INDI",
			xref:       "@I1@",
			tags:       []string{"BIRT", "DATE"},
			want:       "Individual @I1@ > BIRT > DATE",
		},
		{
			name:       "Individual with single tag",
			recordType: "INDI",
			xref:       "@I1@",
			tags:       []string{"NAME"},
			want:       "Individual @I1@ > NAME",
		},
		{
			name:       "Individual with no tags",
			recordType: "INDI",
			xref:       "@I1@",
			tags:       []string{},
			want:       "Individual @I1@",
		},
		{
			name:       "Individual with nil tags",
			recordType: "INDI",
			xref:       "@I1@",
			tags:       nil,
			want:       "Individual @I1@",
		},
		{
			name:       "Header without XRef",
			recordType: "HEAD",
			xref:       "",
			tags:       []string{"CHAR"},
			want:       "Header > CHAR",
		},
		{
			name:       "Family with deeply nested path",
			recordType: "FAM",
			xref:       "@F1@",
			tags:       []string{"MARR", "PLAC", "FORM"},
			want:       "Family @F1@ > MARR > PLAC > FORM",
		},
		{
			name:       "unknown record type",
			recordType: "_CUSTOM",
			xref:       "@X1@",
			tags:       []string{"DATA"},
			want:       "_CUSTOM @X1@ > DATA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildNestedPath(tt.recordType, tt.xref, tt.tags...)
			if got != tt.want {
				t.Errorf("BuildNestedPath(%q, %q, %v) = %q, want %q",
					tt.recordType, tt.xref, tt.tags, got, tt.want)
			}
		})
	}
}

func TestPathSeparator(t *testing.T) {
	// Verify the separator constant is what we expect
	if PathSeparator != " > " {
		t.Errorf("PathSeparator = %q, want %q", PathSeparator, " > ")
	}
}

// Example-style test to verify path construction matches expected ConversionNote.Path format
func TestPathFormatMatchesConversionNoteExamples(t *testing.T) {
	// These examples come from the ConversionNote.Path documentation
	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "record level from docs",
			got:  BuildRecordPath("INDI", "@I1@"),
			want: "Individual @I1@",
		},
		{
			name: "nested element from docs",
			got:  BuildNestedPath("INDI", "@I1@", "BIRT", "DATE"),
			want: "Individual @I1@ > BIRT > DATE",
		},
		{
			name: "header element from docs (generalized)",
			got:  BuildNestedPath("HEAD", "", "CHAR"),
			want: "Header > CHAR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("Path format mismatch: got %q, want %q", tt.got, tt.want)
			}
		})
	}
}
