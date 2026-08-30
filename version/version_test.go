package version

import (
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
	"github.com/cacack/gedcom-go/v2/parser"
)

// T030: Write tests for version detection (header-based and tag-based fallback)
func TestDetectVersion(t *testing.T) {
	tests := []struct {
		name  string
		lines []*parser.Line
		want  gedcom.Version
	}{
		{
			name: "detect 5.5 from header",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "GEDC"},
				{Level: 2, Tag: "VERS", Value: "5.5"},
			},
			want: gedcom.Version55,
		},
		{
			name: "detect 5.5.1 from header",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "GEDC"},
				{Level: 2, Tag: "VERS", Value: "5.5.1"},
			},
			want: gedcom.Version551,
		},
		{
			name: "detect 7.0 from header",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "GEDC"},
				{Level: 2, Tag: "VERS", Value: "7.0"},
			},
			want: gedcom.Version70,
		},
		{
			name: "no version in header",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 0, Tag: "TRLR"},
			},
			want: gedcom.Version55,
		},
		{
			name:  "empty input",
			lines: []*parser.Line{},
			want:  gedcom.Version55,
		},
		{
			name: "detect 7.0.0 from header (alternative format)",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "GEDC"},
				{Level: 2, Tag: "VERS", Value: "7.0.0"},
			},
			want: gedcom.Version70,
		},
		{
			name: "detect with extra whitespace in version",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "GEDC"},
				{Level: 2, Tag: "VERS", Value: "  5.5.1  "},
			},
			want: gedcom.Version551,
		},
		{
			name: "unknown version falls back to 5.5",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "GEDC"},
				{Level: 2, Tag: "VERS", Value: "6.0"},
			},
			want: gedcom.Version55,
		},
		{
			name: "GEDC without VERS falls back",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "GEDC"},
				{Level: 1, Tag: "CHAR", Value: "UTF-8"},
			},
			want: gedcom.Version55,
		},
		{
			name: "detect 7.0 from tags (EXID)",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 0, Tag: "INDI"},
				{Level: 1, Tag: "EXID", Value: "123"},
			},
			want: gedcom.Version70,
		},
		{
			name: "detect 7.0 from tags (PHRASE)",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "PHRASE", Value: "test"},
			},
			want: gedcom.Version70,
		},
		{
			name: "detect 7.0 from tags (SNOTE)",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "SNOTE", Value: "@N1@"},
			},
			want: gedcom.Version70,
		},
		{
			name: "detect 5.5.1 from tags (MAP)",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "MAP"},
				{Level: 2, Tag: "LATI", Value: "N123"},
			},
			want: gedcom.Version551,
		},
		{
			name: "detect 5.5.1 from tags (EMAIL)",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "EMAIL", Value: "test@example.com"},
			},
			want: gedcom.Version551,
		},
		{
			name: "detect 5.5.1 from tags (WWW)",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "WWW", Value: "http://example.com"},
			},
			want: gedcom.Version551,
		},
		{
			name: "detect 5.5.1 from tags (FACT)",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "FACT", Value: "something"},
			},
			want: gedcom.Version551,
		},
		// ADR-005 cascade: the header VERS is trusted first and wins even
		// when version-specific tags for a *different* version are present.
		// The tag-based heuristic is a fallback, never an override.
		{
			name: "header 5.5 wins over 7.0 indicator tag (EXID)",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "GEDC"},
				{Level: 2, Tag: "VERS", Value: "5.5"},
				{Level: 0, Tag: "INDI"},
				{Level: 1, Tag: "EXID", Value: "123"},
			},
			want: gedcom.Version55,
		},
		{
			name: "header 7.0 wins over 5.5.1 indicator tags (MAP/LATI)",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "GEDC"},
				{Level: 2, Tag: "VERS", Value: "7.0"},
				{Level: 1, Tag: "MAP"},
				{Level: 2, Tag: "LATI", Value: "N123"},
			},
			want: gedcom.Version70,
		},
		{
			name: "header 5.5.1 wins over 7.0 indicator tag (SCHMA)",
			lines: []*parser.Line{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "GEDC"},
				{Level: 2, Tag: "VERS", Value: "5.5.1"},
				{Level: 1, Tag: "SCHMA"},
			},
			want: gedcom.Version551,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectVersion(tt.lines)

			if got != tt.want {
				t.Errorf("DetectVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
