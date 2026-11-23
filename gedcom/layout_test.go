package gedcom

import "testing"

func TestLayoutLine(t *testing.T) {
	layout := &Layout{Lines: []LayoutLine{
		{Level: 0, Tag: "HEAD", Text: "0 HEAD", Terminator: "\n"},
		{}, // a skipped line
	}}
	var none *Layout

	tests := []struct {
		name   string
		layout *Layout
		n      int
		found  bool
	}{
		{"nil layout", none, 1, false},
		{"line zero", layout, 0, false},
		{"parsed line", layout, 1, true},
		{"skipped line", layout, 2, false},
		{"past the end", layout, 3, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.layout.Line(tt.n); (got != nil) != tt.found {
				t.Errorf("Line(%d) = %v; want found %v", tt.n, got, tt.found)
			}
		})
	}
}

func TestLayoutLineReads(t *testing.T) {
	line := LayoutLine{Level: 1, XRef: "", Tag: "NAME", Value: "Ann /Example/"}
	if !line.Reads(1, "", "NAME", "Ann /Example/") {
		t.Error("line does not read as itself")
	}
	for _, c := range []struct {
		level            int
		xref, tag, value string
	}{
		{2, "", "NAME", "Ann /Example/"},
		{1, "@X@", "NAME", "Ann /Example/"},
		{1, "", "GIVN", "Ann /Example/"},
		{1, "", "NAME", "Anne /Example/"},
	} {
		if line.Reads(c.level, c.xref, c.tag, c.value) {
			t.Errorf("line reads as %+v", c)
		}
	}
}

func TestCloneSharesLayout(t *testing.T) {
	doc := &Document{Layout: &Layout{BOM: true}}
	if got := doc.Clone().Layout; got != doc.Layout {
		t.Errorf("Clone().Layout = %p; want the original %p", got, doc.Layout)
	}
}
