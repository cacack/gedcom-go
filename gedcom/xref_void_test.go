package gedcom

import (
	"errors"
	"slices"
	"testing"
)

// voidDocument returns a document whose header claims version v, with an
// individual @I1@ whose NOTE pointer (typed and raw) names a note record
// @VOID@.
func voidDocument(v Version) *Document {
	ind := &Record{
		XRef: "@I1@", Type: RecordTypeIndividual,
		Entity: &Individual{XRef: "@I1@", NoteXRefs: []string{"@VOID@"}},
		Tags:   []*Tag{{Level: 1, Tag: "NOTE", Value: "@VOID@"}},
	}
	note := &Record{XRef: "@VOID@", Type: RecordTypeNote, Entity: &Note{XRef: "@VOID@", Text: "hello"}}
	return &Document{
		Header:  &Header{Version: v, DeclaredVersion: v},
		Records: []*Record{ind, note},
		XRefMap: map[string]*Record{"@I1@": ind, "@VOID@": note},
	}
}

// TestIsRecordPointer pins issue #614: @VOID@ is a record pointer in every
// document except a 7.0 one, the rule Apply uses.
func TestIsRecordPointer(t *testing.T) {
	for _, tt := range []struct {
		name string
		doc  *Document
		want bool
	}{
		{"5.5", &Document{Header: &Header{Version: Version55}}, true},
		{"5.5.1", &Document{Header: &Header{Version: Version551}}, true},
		{"no version", &Document{Header: &Header{}}, true},
		{"nil header", &Document{}, true},
		{"nil document", nil, true},
		{"7.0", &Document{Header: &Header{Version: Version70}}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.doc.IsRecordPointer("@VOID@"); got != tt.want {
				t.Errorf("IsRecordPointer(@VOID@) = %v, want %v", got, tt.want)
			}
			if !tt.doc.IsRecordPointer("@I1@") {
				t.Error("IsRecordPointer(@I1@) = false, want true")
			}
			if tt.doc.IsRecordPointer("text") {
				t.Error("IsRecordPointer(text) = true, want false")
			}
		})
	}
}

// TestVisitForVersionVoid pins issue #614: VisitForVersion reports @VOID@
// under 5.5/5.5.1, where it is an ordinary identifier, and not under 7.0 or
// from Visit, which uses the 7.0 grammar.
func TestVisitForVersionVoid(t *testing.T) {
	r := voidDocument(Version551).GetRecord("@I1@")
	collect := func(walk func(func(string))) []string {
		var got []string
		walk(func(ref string) { got = append(got, ref) })
		return got
	}
	for _, v := range []Version{Version55, Version551} {
		got := collect(func(f func(string)) { VisitForVersion(r, v, f) })
		if want := []string{"@VOID@", "@VOID@"}; !slices.Equal(got, want) {
			t.Errorf("VisitForVersion(%s) = %q, want %q", v, got, want)
		}
	}
	if got := collect(func(f func(string)) { VisitForVersion(r, Version70, f) }); len(got) != 0 {
		t.Errorf("VisitForVersion(7.0) = %q, want none", got)
	}
	if got := collect(func(f func(string)) { Visit(r, f) }); len(got) != 0 {
		t.Errorf("Visit = %q, want none", got)
	}
}

// TestSubsetVoid pins issue #614: a 5.5.1 subset follows a pointer to a
// record named @VOID@ and accepts it as a seed, while in 7.0 @VOID@ stays the
// null pointer.
func TestSubsetVoid(t *testing.T) {
	sub, err := voidDocument(Version551).Subset([]string{"@I1@"})
	if err != nil {
		t.Fatalf("5.5.1 Subset errored: %v", err)
	}
	if sub.GetRecord("@VOID@") == nil {
		t.Error("5.5.1 Subset dropped the referenced @VOID@ record")
	}
	if sub, err := voidDocument(Version551).Subset([]string{"@VOID@"}); err != nil || sub.GetRecord("@VOID@") == nil {
		t.Errorf("5.5.1 Subset(@VOID@ seed) = %v, %v; want the @VOID@ record", sub, err)
	}

	sub, err = voidDocument(Version70).Subset([]string{"@I1@"})
	if err != nil {
		t.Fatalf("7.0 Subset errored: %v", err)
	}
	if sub.GetRecord("@VOID@") != nil {
		t.Error("7.0 Subset followed the @VOID@ null pointer")
	}
	if _, err := voidDocument(Version70).Subset([]string{"@VOID@"}); !errors.Is(err, ErrUnknownXRef) {
		t.Errorf("7.0 Subset(@VOID@ seed) error = %v, want ErrUnknownXRef", err)
	}
}
