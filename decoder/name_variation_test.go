package decoder

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestParsePersonalNameFoneRomn pins 5.5.1's FONE and ROMN name variations
// (issue #453) decoding into PersonalName.Phonetic and Romanized, with the
// required TYPE naming the scheme. NOTE and SOUR beneath them stay raw but are
// recognized; any other tag is reported.
func TestParsePersonalNameFoneRomn(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME 山田 /太郎/
2 FONE Yamada /Taro/
3 TYPE kana
3 NPFX Dr
3 GIVN Taro
3 NICK Taro-chan
3 SPFX no
3 SURN Yamada
3 NSFX Jr
3 NOTE A phonetic note
3 SOUR @S1@
3 BOGUS x
2 ROMN Yamada /Taro/
3 TYPE romaji
3 SURN Yamada
2 ROMN Jamada /Taro/
3 TYPE user defined
0 @S1@ SOUR
0 TRLR
`
	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	var unknown []string
	for _, d := range result.Diagnostics {
		if d.Code == CodeUnknownTag {
			unknown = append(unknown, d.Message)
		}
	}
	if len(unknown) != 1 || unknown[0] != "unknown tag: BOGUS" {
		t.Errorf("UNKNOWN_TAG diagnostics = %v, want only BOGUS", unknown)
	}

	indi := result.Document.GetIndividual("@I1@")
	if indi == nil || len(indi.Names) != 1 {
		t.Fatal("expected @I1@ with one name")
	}
	name := indi.Names[0]

	if len(name.Transliterations) != 0 {
		t.Errorf("len(Transliterations) = %d, want 0", len(name.Transliterations))
	}
	wantFone := []*gedcom.NameVariation{{
		Value: "Yamada /Taro/", Type: "kana", Prefix: "Dr", Given: "Taro",
		Nickname: "Taro-chan", SurnamePrefix: "no", Surname: "Yamada", Suffix: "Jr",
	}}
	wantRomn := []*gedcom.NameVariation{
		{Value: "Yamada /Taro/", Type: "romaji", Surname: "Yamada"},
		{Value: "Jamada /Taro/", Type: "user defined"},
	}
	if !reflect.DeepEqual(name.Phonetic, wantFone) {
		t.Errorf("Phonetic = %+v, want %+v", name.Phonetic, wantFone)
	}
	if !reflect.DeepEqual(name.Romanized, wantRomn) {
		t.Errorf("Romanized = %+v, want %+v", name.Romanized, wantRomn)
	}
}
