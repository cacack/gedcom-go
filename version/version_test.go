package version

import (
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
	"github.com/cacack/gedcom-go/parser"
)

// T030: Write tests for version detection (header-based and tag-based fallback)
func TestDetectVersion(t *testing.T) {
	tests := []struct {
		name     string
		lines    []*parser.Line
		want     gedcom.Version
		wantErr  bool
		fallback bool
	}{
		{
			name: "detect 5.5 from header",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "GEDC"},
				{Level: 2, Tag: "VERS", Value: "5.5"},
			},
			want:     gedcom.Version55,
			wantErr:  false,
			fallback: false,
		},
		{
			name: "detect 5.5.1 from header",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "GEDC"},
				{Level: 2, Tag: "VERS", Value: "5.5.1"},
			},
			want:     gedcom.Version551,
			wantErr:  false,
			fallback: false,
		},
		{
			name: "detect 7.0 from header",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "GEDC"},
				{Level: 2, Tag: "VERS", Value: "7.0"},
			},
			want:     gedcom.Version70,
			wantErr:  false,
			fallback: false,
		},
		{
			name: "no version in header",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 0, Tag: "TRLR"},
			},
			want:     gedcom.Version55,
			wantErr:  false,
			fallback: true,
		},
		{
			name:     "empty input",
			lines:    []*parser.Line{},
			want:     gedcom.Version55,
			wantErr:  false,
			fallback: true,
		},
		{
			name: "detect 7.0.0 from header (alternative format)",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "GEDC"},
				{Level: 2, Tag: "VERS", Value: "7.0.0"},
			},
			want:     gedcom.Version70,
			wantErr:  false,
			fallback: false,
		},
		{
			name: "detect with extra whitespace in version",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "GEDC"},
				{Level: 2, Tag: "VERS", Value: "  5.5.1  "},
			},
			want:     gedcom.Version551,
			wantErr:  false,
			fallback: false,
		},
		{
			name: "unknown version falls back to 5.5",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "GEDC"},
				{Level: 2, Tag: "VERS", Value: "6.0"},
			},
			want:     gedcom.Version55,
			wantErr:  false,
			fallback: true,
		},
		{
			name: "GEDC without VERS falls back",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "GEDC"},
				{Level: 1, Tag: "CHAR", Value: "UTF-8"},
			},
			want:     gedcom.Version55,
			wantErr:  false,
			fallback: true,
		},
		{
			name: "detect 7.0 from tags (EXID)",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 0, Tag: "INDI"},
				{Level: 1, Tag: "EXID", Value: "123"},
			},
			want:     gedcom.Version70,
			wantErr:  false,
			fallback: true,
		},
		{
			name: "detect 7.0 from tags (PHRASE)",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "PHRASE", Value: "test"},
			},
			want:     gedcom.Version70,
			wantErr:  false,
			fallback: true,
		},
		{
			name: "detect 7.0 from tags (SNOTE)",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "SNOTE", Value: "@N1@"},
			},
			want:     gedcom.Version70,
			wantErr:  false,
			fallback: true,
		},
		{
			name: "detect 5.5.1 from tags (MAP)",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "MAP"},
				{Level: 2, Tag: "LATI", Value: "N123"},
			},
			want:     gedcom.Version551,
			wantErr:  false,
			fallback: true,
		},
		{
			name: "detect 5.5.1 from tags (EMAIL)",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "EMAIL", Value: "test@example.com"},
			},
			want:     gedcom.Version551,
			wantErr:  false,
			fallback: true,
		},
		{
			name: "detect 5.5.1 from tags (WWW)",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "WWW", Value: "http://example.com"},
			},
			want:     gedcom.Version551,
			wantErr:  false,
			fallback: true,
		},
		{
			name: "detect 5.5.1 from tags (FACT)",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "FACT", Value: "something"},
			},
			want:     gedcom.Version551,
			wantErr:  false,
			fallback: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DetectVersion(tt.lines)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DetectVersion() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("DetectVersion() unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("DetectVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidVersion(t *testing.T) {
	tests := []struct {
		name    string
		version gedcom.Version
		want    bool
	}{
		{"5.5 is valid", gedcom.Version55, true},
		{"5.5.1 is valid", gedcom.Version551, true},
		{"7.0 is valid", gedcom.Version70, true},
		{"empty is invalid", gedcom.Version(""), false},
		{"unknown is invalid", gedcom.Version("1.0"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidVersion(tt.version); got != tt.want {
				t.Errorf("IsValidVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
