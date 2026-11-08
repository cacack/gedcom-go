package encoder

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// A GEDCOM 7 patch version is GEDCOM 7.0 for every encoder version gate, as it
// is for the decoder (issue #696, #587). GEDC.VERS still carries the value the
// caller gave.

// longNote551 has a NOTE longer than DefaultMaxLineLength, which 5.x output
// splits with CONC and 7.0 output never does.
var longNote551 = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
	"0 @I1@ INDI\n1 NOTE " + strings.Repeat("x", 300) + "\n" +
	"0 TRLR\n"

const sharedNote70 = "0 HEAD\n1 GEDC\n2 VERS 7.0\n" +
	"0 @N1@ SNOTE shared\n" +
	"0 @I1@ INDI\n1 SNOTE @N1@\n" +
	"0 TRLR\n"

var patchVersionInputs = map[string]string{
	"FONE and ROMN": nameVariations551,
	"TRAN":          nameVariations70,
	"date phrase":   datePhraseInput70,
	"long note":     longNote551,
	"shared note":   sharedNote70,
	// The synthesized GEDC goes where the 7.0 header grammar puts it: first.
	"header without GEDC": "0 HEAD\n1 SOUR X\n1 CHAR UTF-8\n0 TRLR\n",
}

// TestTargetVersionPatch70MatchesVersion70 pins that TargetVersion 7.0.14
// writes what 7.0 does -- no CONC, TRAN but no FONE/ROMN, DATE PHRASE, SNOTE
// pointers, 7.0 header order -- apart from the GEDC.VERS value, which stays as
// given.
func TestTargetVersionPatch70MatchesVersion70(t *testing.T) {
	for inName, input := range patchVersionInputs {
		for encName, encode := range nameVariationEncoders {
			t.Run(inName+"/"+encName, func(t *testing.T) {
				want := encodeWithFunc(t, encode, decodeEntityOnly(t, input), &EncodeOptions{TargetVersion: gedcom.Version70})
				opts := &EncodeOptions{TargetVersion: "7.0.14"}
				got := encodeWithFunc(t, encode, decodeEntityOnly(t, input), opts)

				if !strings.Contains(got, "\n2 VERS 7.0.14\n") {
					t.Errorf("output does not declare VERS 7.0.14:\n%s", got)
				}
				if normalized := strings.Replace(got, "\n2 VERS 7.0.14\n", "\n2 VERS 7.0\n", 1); normalized != want {
					t.Errorf("7.0.14 output =\n%s\nwant the 7.0 output apart from VERS:\n%s", got, want)
				}
				if *opts != (EncodeOptions{TargetVersion: "7.0.14"}) {
					t.Errorf("caller's options were modified: %+v", *opts)
				}
			})
		}
	}
}

// TestTargetVersionUnrecognizedNot70 pins that a value the normalization rule
// does not recognize, such as 7.1, is not taken for 7.0: CONC splitting still
// applies and no DATE PHRASE is written. VERS is written as given.
func TestTargetVersionUnrecognizedNot70(t *testing.T) {
	for encName, encode := range nameVariationEncoders {
		t.Run(encName, func(t *testing.T) {
			opts := &EncodeOptions{TargetVersion: "7.1"}
			out := encodeWithFunc(t, encode, decodeEntityOnly(t, longNote551), opts)
			if !strings.Contains(out, "\n2 VERS 7.1\n") {
				t.Errorf("output does not declare VERS 7.1:\n%s", out)
			}
			if !hasTagLine(out, "CONC") {
				t.Errorf("7.1 output was not CONC-split:\n%s", out)
			}

			out = encodeWithFunc(t, encode, decodeEntityOnly(t, datePhraseInput70), opts)
			if hasTagLine(out, "PHRASE") {
				t.Errorf("7.1 output has a PHRASE line:\n%s", out)
			}
		})
	}
}
