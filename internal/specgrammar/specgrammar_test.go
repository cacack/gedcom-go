package specgrammar

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The checked-in tables must be what the vendored inventories generate:
// after updating an inventory, run `go generate ./validator`.
func TestGeneratedTablesUpToDate(t *testing.T) {
	got, err := Generate("../../testdata/spec")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	want, err := os.ReadFile("../../validator/grammar_tables.go")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Error("validator/grammar_tables.go is stale; run `go generate ./validator`")
	}
}

func TestGenerateContent(t *testing.T) {
	src, err := Generate("../../testdata/spec")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// gofmt aligns map values; compare with runs of white space collapsed.
	flat := strings.Join(strings.Fields(string(src)), " ")
	for _, want := range []string{
		// 7.0 SEX is an enumeration of the tags its set's URIs stand for.
		`"SEX": {payload: payloadEnum, enum: []string{"F", "M", "U", "X"}}`,
		// 7.0 cardinalities come from cardinalities.tsv.
		`"GEDC": {{"GEDC", 1, 1}}`,
		// 5.5.1 ROLE accepts a parenthesized descriptor.
		`enum: []string{"CHIL", "FATH", "HUSB", "MOTH", "SPOU", "WIFE"}, open: enumOpenParenthesized`,
		// 5.5.1 NAME.TYPE is open ("<user defined>").
		`"PERSONAL_NAME_STRUCTURE.NAME.TYPE": {payload: payloadEnum, enum: []string{"aka", "birth", "immigrant", "maiden", "married"}, open: enumOpen}`,
		// 5.5.x NOTE is a pointer or text, both permitted.
		`"NOTE": {{"NOTE_STRUCTURE.NOTE", 0, -1}, {"NOTE_STRUCTURE.NOTE#2", 0, -1}}`,
		`"CHANGE_DATE.CHAN.DATE": {payload: payloadDateExact`,
	} {
		if !strings.Contains(flat, want) {
			t.Errorf("generated source lacks %s", want)
		}
	}
}

func TestEnumTag(t *testing.T) {
	for uri, want := range map[string]string{
		"https://gedcom.io/terms/v7/enum-ADOP-HUSB": "HUSB",
		"https://gedcom.io/terms/v7/enum-DNS_CAN":   "DNS_CAN",
		"https://gedcom.io/terms/v7/INDI-RELI":      "RELI",
		"https://gedcom.io/terms/v7/BIRT":           "BIRT",
		"https://gedcom.io/terms/v7/enum-0":         "0",
	} {
		if got := enumTag(uri); got != want {
			t.Errorf("enumTag(%s) = %q, want %q", uri, got, want)
		}
	}
}

func TestPayload55(t *testing.T) {
	prims := map[string]primitive{
		"A":     {values: []string{"x", "y"}},
		"B":     {placeholders: []string{"<A>"}, values: []string{"z", "x"}},
		"OPEN":  {values: []string{"p"}, placeholders: []string{"<user defined>"}},
		"VIA":   {placeholders: []string{"<OPEN>"}},
		"DATE":  {placeholders: []string{"<DATE_GREG>"}},
		"ODD":   {placeholders: []string{"something else"}},
		"LOOP":  {placeholders: []string{"<LOOP>"}},
		"EMPTY": {},
	}
	for payload, want := range map[string]string{
		"[Y|<NULL>]":                kindYNull,
		"@<XREF:INDI>@":             kindPointer,
		"[ @<XREF:REPO>@ | <NULL>]": kindPointerOrNull,
		"<AGE_AT_EVENT>":            kindAge,
		"<A>":                       kindEnum,
		"<B>":                       kindEnum,
		"<VIA>":                     kindEnum,
		"<DATE>":                    kindText,
		"<ODD>":                     kindText,
		"<LOOP>":                    kindText,
		"<EMPTY>":                   kindText,
		"<MISSING>":                 kindText,
		"[<TEXT> | <NULL>]":         kindText,
	} {
		if got, _, _ := payload55(payload, prims); got != want {
			t.Errorf("payload55(%q) = %s, want %s", payload, got, want)
		}
	}
	if _, enum, open := payload55("<B>", prims); !slices.Equal(enum, []string{"x", "y", "z"}) || open != openClosed {
		t.Errorf("<B> = %v %s, want [x y z] closed", enum, open)
	}
	if _, _, open := payload55("<VIA>", prims); open != openAny {
		t.Errorf("<VIA> open = %s, want %s", open, openAny)
	}
}

func TestParseCardinality(t *testing.T) {
	for in, want := range map[string][2]int{"{0:1}": {0, 1}, "1:M": {1, -1}, "0:3": {0, 3}} {
		lo, hi, err := parseCardinality(in)
		if err != nil || lo != want[0] || hi != want[1] {
			t.Errorf("parseCardinality(%q) = %d, %d, %v", in, lo, hi, err)
		}
	}
	for _, in := range []string{"", "1", "x:1", "1:x"} {
		if _, _, err := parseCardinality(in); err == nil {
			t.Errorf("parseCardinality(%q): no error", in)
		}
	}
}

// writeSpec writes a spec directory holding the real inventories, with the
// named files of one version replaced.
func writeSpec(t *testing.T, version string, replace map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for _, v := range []string{"gedcom-5.5", "gedcom-5.5.1", "gedcom-7.0"} {
		if err := os.MkdirAll(filepath.Join(dir, v), 0o755); err != nil {
			t.Fatal(err)
		}
		matches, _ := filepath.Glob(filepath.Join("..", "..", "testdata", "spec", v, "*.tsv"))
		for _, m := range matches {
			data, err := os.ReadFile(m) // #nosec G304 -- test fixture
			if err != nil {
				t.Fatal(err)
			}
			if v == version {
				if r, ok := replace[filepath.Base(m)]; ok {
					data = []byte(r)
				}
			}
			if err := os.WriteFile(filepath.Join(dir, v, filepath.Base(m)), data, 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	return dir
}

func TestGenerateErrors(t *testing.T) {
	const sub = "superstructure\ttag\tstructure\n\tINDI\tI\nI\tSEX\tS\n"
	for name, tc := range map[string]struct {
		version string
		files   map[string]string
		want    string
	}{
		"missing dir":          {"", nil, "substructures.tsv"},
		"columns":              {"gedcom-5.5", map[string]string{"payloads.tsv": "structure\tpayload\nonly-one\n"}, "columns"},
		"no cardinality":       {"gedcom-5.5.1", map[string]string{"substructures.tsv": sub, "cardinalities.tsv": "a\tb\tc\n"}, "no cardinality"},
		"bad cardinality":      {"gedcom-5.5.1", map[string]string{"substructures.tsv": sub, "cardinalities.tsv": "a\tb\tc\nI\tS\tz\n"}, "cardinality"},
		"5.5 primitives":       {"gedcom-5.5", map[string]string{"primitives.tsv": "a\nb\n"}, "columns"},
		"7.0 sets":             {"gedcom-7.0", map[string]string{"enumerationsets.tsv": "a\nb\n"}, "columns"},
		"7.0 enumerations":     {"gedcom-7.0", map[string]string{"enumerations.tsv": "a\nb\n"}, "columns"},
		"7.0 payloads":         {"gedcom-7.0", map[string]string{"payloads.tsv": "a\nb\n"}, "columns"},
		"7.0 substructures":    {"gedcom-7.0", map[string]string{"substructures.tsv": "a\nb\n"}, "columns"},
		"7.0 empty set":        {"gedcom-7.0", map[string]string{"enumerationsets.tsv": "set\tvalue\n"}, "is empty"},
		"7.0 enum with no set": {"gedcom-7.0", map[string]string{"enumerations.tsv": "structure\tset\n"}, "no set"},
		"5.5.1 cardinalities":  {"gedcom-5.5.1", map[string]string{"cardinalities.tsv": "a\nb\n"}, "columns"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "absent")
			if tc.version != "" {
				dir = writeSpec(t, tc.version, tc.files)
			}
			_, err := Generate(dir)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Generate: %v, want an error containing %q", err, tc.want)
			}
		})
	}
}
