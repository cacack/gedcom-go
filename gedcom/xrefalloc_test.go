package gedcom

import (
	"fmt"
	"testing"
)

func TestFreeXRef(t *testing.T) {
	withHeader := func(v Version, h *Header, records ...*Record) *Document {
		if h == nil {
			h = &Header{}
		}
		h.DeclaredVersion = v
		return &Document{Header: h, Records: records}
	}
	objePointer := func(value string) *Record {
		return &Record{XRef: "@I1@", Type: RecordTypeIndividual, Tags: []*Tag{{Level: 1, Tag: "OBJE", Value: value}}}
	}

	tests := []struct {
		name   string
		doc    *Document
		prefix string
		want   string
	}{
		{"nil document", nil, "O", "@O1@"},
		{"empty document", &Document{}, "O", "@O1@"},
		{"new document", NewDocument(Version70), "I", "@I1@"},
		{
			"record in Records only",
			&Document{Records: []*Record{nil, {XRef: "@O1@"}, {XRef: ""}}},
			"O", "@O2@",
		},
		{
			"record in XRefMap only",
			&Document{XRefMap: map[string]*Record{"@O1@": {XRef: "@O1@"}}},
			"O", "@O2@",
		},
		{
			"gap is filled",
			&Document{Records: []*Record{{XRef: "@O1@"}, {XRef: "@O3@"}}},
			"O", "@O2@",
		},
		{
			"other prefix is not counted",
			&Document{Records: []*Record{{XRef: "@I1@"}, {XRef: "@I2@"}}},
			"O", "@O1@",
		},
		{
			"dangling raw-tag pointer is skipped",
			&Document{Records: []*Record{objePointer("@O1@")}},
			"O", "@O2@",
		},
		{
			"dangling entity pointer is skipped",
			&Document{Records: []*Record{{
				XRef:   "@I1@",
				Type:   RecordTypeIndividual,
				Entity: &Individual{XRef: "@I1@", Media: []*MediaLink{{MediaXRef: "@O1@"}}},
			}}},
			"O", "@O2@",
		},
		{
			"raw-tag pointer with trailing space is skipped",
			&Document{Records: []*Record{objePointer("@O1@ ")}},
			"O", "@O2@",
		},
		{
			"header tag pointer with trailing space is skipped",
			withHeader(Version70, &Header{Tags: []*Tag{{Level: 1, Tag: "SUBM", Value: "@U1@ "}}}),
			"U", "@U2@",
		},
		{
			"inline text is not a pointer",
			&Document{Records: []*Record{objePointer("see @O1@ and @O2@")}},
			"O", "@O1@",
		},
		{
			"spaced pointer counts when 5.5.1 is declared",
			withHeader(Version551, nil, objePointer("@O 1@")),
			"O ", "@O 2@",
		},
		{
			"spaced pointer counts when 5.5 is declared",
			withHeader(Version55, nil, objePointer("@O 1@")),
			"O ", "@O 2@",
		},
		{
			"spaced value is text when 7.0 is declared",
			withHeader(Version70, nil, objePointer("@O 1@")),
			"O ", "@O 1@",
		},
		{
			"spaced value is text when no version is declared",
			&Document{Header: &Header{Version: Version551}, Records: []*Record{objePointer("@O 1@")}},
			"O ", "@O 1@",
		},
		{
			"header submitter is skipped",
			withHeader(Version70, &Header{Submitter: "@U1@"}),
			"U", "@U2@",
		},
		{
			"header tag pointer is skipped",
			withHeader(Version70, &Header{Tags: []*Tag{{Level: 1, Tag: "SUBM", Value: "@U1@"}, nil}}),
			"U", "@U2@",
		},
		{
			"spaced header pointer counts when 5.5.1 is declared",
			withHeader(Version551, &Header{Submitter: "@U 1@"}),
			"U ", "@U 2@",
		},
		{"prefix is not validated", &Document{}, "bad prefix!", "@bad prefix!1@"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.doc.FreeXRef(tt.prefix); got != tt.want {
				t.Errorf("FreeXRef(%q) = %q, want %q", tt.prefix, got, tt.want)
			}
		})
	}
}

func TestFreeXRefDoesNotReserve(t *testing.T) {
	doc := NewDocument(Version70)
	if a, b := doc.FreeXRef("O"), doc.FreeXRef("O"); a != "@O1@" || b != "@O1@" {
		t.Errorf("FreeXRef twice = %q, %q; want @O1@ both times", a, b)
	}
}

func TestFreeXRefAddRecordLoopNeverRepeats(t *testing.T) {
	doc := NewDocument(Version70)
	// A dangling pointer in the middle of the range must be stepped over.
	if err := doc.AddRecord(&Record{
		XRef: "@I1@",
		Type: RecordTypeIndividual,
		Tags: []*Tag{{Level: 1, Tag: "OBJE", Value: "@O3@"}},
	}); err != nil {
		t.Fatal(err)
	}
	var got []string
	for range 4 {
		xref := doc.FreeXRef("O")
		if err := doc.AddRecord(&Record{XRef: xref, Type: RecordTypeMedia}); err != nil {
			t.Fatalf("AddRecord(%q): %v", xref, err)
		}
		got = append(got, xref)
	}
	want := []string{"@O1@", "@O2@", "@O4@", "@O5@"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("allocated %v, want %v", got, want)
	}
}

func TestXRefAllocatorNext(t *testing.T) {
	doc := &Document{Records: []*Record{
		{XRef: "@O2@"},
		{XRef: "@I1@", Tags: []*Tag{{Level: 1, Tag: "OBJE", Value: "@O4@"}}},
	}}
	a := doc.NewXRefAllocator()

	t.Run("successive calls differ and skip used", func(t *testing.T) {
		for _, want := range []string{"@O1@", "@O3@", "@O5@", "@O6@"} {
			if got := a.Next("O"); got != want {
				t.Errorf("Next(O) = %q, want %q", got, want)
			}
		}
	})
	t.Run("prefixes count independently", func(t *testing.T) {
		if got := a.Next("S"); got != "@S1@" {
			t.Errorf("Next(S) = %q, want @S1@", got)
		}
		if got := a.Next("I"); got != "@I2@" {
			t.Errorf("Next(I) = %q, want @I2@", got)
		}
		if got := a.Next("O"); got != "@O7@" {
			t.Errorf("Next(O) = %q, want @O7@", got)
		}
	})
	t.Run("a prefix that spells another's XRef cannot collide", func(t *testing.T) {
		// Prefix "O" hands out @O11@, which prefix "O1" would spell first.
		b := (&Document{}).NewXRefAllocator()
		for range 11 {
			b.Next("O")
		}
		if got := b.Next("O1"); got != "@O12@" {
			t.Errorf("Next(O1) = %q, want @O12@", got)
		}
	})
	t.Run("does not see later additions", func(t *testing.T) {
		d := NewDocument(Version70)
		snapshot := d.NewXRefAllocator()
		if err := d.AddRecord(&Record{XRef: "@N1@"}); err != nil {
			t.Fatal(err)
		}
		if got := snapshot.Next("N"); got != "@N1@" {
			t.Errorf("Next(N) = %q, want the snapshot's @N1@", got)
		}
	})
	t.Run("zero value", func(t *testing.T) {
		var z XRefAllocator
		if a, b := z.Next("I"), z.Next("I"); a != "@I1@" || b != "@I2@" {
			t.Errorf("zero value Next = %q, %q", a, b)
		}
	})
}

// ExampleDocument_NewXRefAllocator adds two media records to a document that
// still holds a pointer to a record it no longer has.
func ExampleDocument_NewXRefAllocator() {
	doc := NewDocument(Version70)
	_ = doc.AddRecord(&Record{
		XRef: "@I1@",
		Type: RecordTypeIndividual,
		Tags: []*Tag{{Level: 1, Tag: "OBJE", Value: "@O1@"}}, // dangling
	})

	xrefs := doc.NewXRefAllocator()
	for range 2 {
		xref := xrefs.Next("O")
		_ = doc.AddRecord(&Record{XRef: xref, Type: RecordTypeMedia})
		fmt.Println(xref)
	}

	// Output:
	// @O2@
	// @O3@
}
