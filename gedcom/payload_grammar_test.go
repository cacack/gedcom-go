package gedcom

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/internal/payloadgrammar"
)

// payload_grammar_test.go holds the payload parsers of this package to the
// GEDCOM 7.0 ABNF (ADR 0013): every corpus value and grammar-generated
// string of a datatype goes to both the oracle and the parser, and a
// disagreement fails unless a row of the spec interpretation register
// allows it.

// parseDateAccepts is ParseDate's answer as a DateValue check. An empty
// DateValue is valid, and ParseDate reports it as ErrEmptyDate so a caller
// can tell an absent date from a parsed one: that is acceptance.
func parseDateAccepts(s string) bool {
	_, err := ParseDate(s)
	return err == nil || s == "" && errors.Is(err, ErrEmptyDate)
}

func TestParseDateMatchesGrammar(t *testing.T) {
	o := payloadgrammar.Load(t)
	payloadgrammar.Compare(t, payloadgrammar.DateValue, parseDateAccepts, dateDepartures(o))
}

// FuzzParseDateMatchesGrammar compares ParseDate's acceptance with the
// DateValue grammar on arbitrary input.
func FuzzParseDateMatchesGrammar(f *testing.F) {
	for _, s := range []string{
		"", "1 JAN 2000", "ABT 1850", "BET 1850 AND 1860", "FROM 1880 TO 1920", "TO 1920",
		"JULIAN 44 BCE", "FRENCH_R 1 VEND 1", "HEBREW 1 TSH 5785", "_MAYAN 5 _KIN 2000",
		"BET JULIAN 1 JAN 1700 AND 1 JAN 1800", "31 FEB 2000", "abt  1850 ", "21 FEB 1750/51",
		"44 B.C.", "@#DJULIAN@ 1 JAN 1750", "INT 1850 (about 1850)", "(unknown)", "HEBREW 5000 BCE",
	} {
		f.Add(s)
	}
	o := payloadgrammar.Load(f)
	departures := dateDepartures(o)
	f.Fuzz(func(t *testing.T, s string) {
		payloadgrammar.Check(t, payloadgrammar.DateValue, s, parseDateAccepts, departures)
	})
}

var (
	dualYear    = regexp.MustCompile(`\b(\d+)/\d+\b`)
	bcEpoch     = regexp.MustCompile(`(?i) (B\.C\.E\.|B\.C\.|BC)$| (B\.C\.E\.|B\.C\.|BC)( (AND|TO) )`)
	datePhrase  = regexp.MustCompile(`^\(.*\)$`)
	interpreted = regexp.MustCompile(`^INT (.*?)(?: ?\(.*)?$`)
	dayOfMonth  = regexp.MustCompile(`\b\d+( [A-Z]+\b)`)
	zeroNumber  = regexp.MustCompile(`(^| )0+( |$)`)
	calendarEsc = strings.NewReplacer("@#DGREGORIAN@", "GREGORIAN ", "@#DJULIAN@", "JULIAN ", "@#DHEBREW@", "HEBREW ", "@#DFRENCH R@", "FRENCH_R ")
)

// dateDepartures names the rows that allow ParseDate to differ from the
// DateValue grammar. When ParseDate accepts what the grammar rejects, it
// undoes one leniency at a time until the grammar accepts, naming the row of
// each it undid; when ParseDate rejects what the grammar accepts, it names
// the unsupported form.
func dateDepartures(o *payloadgrammar.Oracle) payloadgrammar.Departures {
	undo := []struct {
		id   string
		undo func(string) string
	}{
		{"SI-016", func(s string) string { return strings.Join(strings.Fields(s), " ") }},
		{"SI-003", strings.ToUpper},
		{"SI-002", func(s string) string { return strings.Join(strings.Fields(calendarEsc.Replace(s)), " ") }},
		{"SI-015", func(s string) string {
			s = datePhrase.ReplaceAllString(s, "")
			s = interpreted.ReplaceAllString(s, "$1")
			s = dualYear.ReplaceAllString(s, "$1")
			return bcEpoch.ReplaceAllString(s, " BCE$3")
		}},
		{"SI-017", func(s string) string { return dayOfMonth.ReplaceAllString(s, "1$1") }},
	}
	return func(s string, libAccepts bool) []string {
		if !libAccepts {
			// A day or year of 0, which an extension calendar or month may
			// have, is Date's absent value.
			if zeroNumber.MatchString(s) && parseDateAccepts(zeroNumber.ReplaceAllString(s, "${1}1$2")) {
				return []string{"SI-018"}
			}
			for _, tok := range strings.Fields(s) {
				if _, err := strconv.Atoi(tok); o.Match("Integer", tok) && err != nil {
					return []string{"SI-021"}
				}
			}
			return nil
		}
		var ids []string
		for _, step := range undo {
			if next := step.undo(s); next != s {
				s = next
				ids = append(ids, step.id)
				if o.Accepts(payloadgrammar.DateValue, s) {
					return ids
				}
			}
		}
		return nil
	}
}

func TestParseLatitudeMatchesGrammar(t *testing.T) {
	payloadgrammar.Compare(t, payloadgrammar.Latitude, func(s string) bool {
		_, err := ParseLatitude(s)
		return err == nil
	}, coordinateDepartures(payloadgrammar.Load(t), payloadgrammar.Latitude))
}

func TestParseLongitudeMatchesGrammar(t *testing.T) {
	payloadgrammar.Compare(t, payloadgrammar.Longitude, func(s string) bool {
		_, err := ParseLongitude(s)
		return err == nil
	}, coordinateDepartures(payloadgrammar.Load(t), payloadgrammar.Longitude))
}

// plainCoordinate is a coordinate ParseCoordinate's leniency reads: a
// direction, then digits with at most one point, either side of it empty.
var plainCoordinate = regexp.MustCompile(`^([NSEWnsew])0*(\d*)(?:\.(\d*))?$`)

// coordinateDepartures allows ParseCoordinate's leniency: surrounding
// whitespace, and a number with leading zeros or no digits before or after
// its point (SI-011).
func coordinateDepartures(o *payloadgrammar.Oracle, t payloadgrammar.Type) payloadgrammar.Departures {
	return func(s string, libAccepts bool) []string {
		m := plainCoordinate.FindStringSubmatch(strings.TrimSpace(s))
		if !libAccepts || m == nil {
			return nil
		}
		whole, frac := m[2], m[3]
		if whole == "" {
			whole = "0"
		}
		canonical := m[1] + whole
		if frac != "" {
			canonical += "." + frac
		}
		if o.Accepts(t, canonical) {
			return []string{"SI-011"}
		}
		return nil
	}
}

// TestSplitPersonalNameMatchesGrammar holds SplitPersonalName to the
// PersonalName grammar. SplitPersonalName rejects nothing, so every value
// the grammar rejects is read as written, which SI-020 records; a value the
// grammar accepts must split where the grammar does, its slashes delimiting
// the surname (SI-009 trims the given name and suffix).
func TestSplitPersonalNameMatchesGrammar(t *testing.T) {
	o := payloadgrammar.Load(t)
	payloadgrammar.Compare(t, payloadgrammar.PersonalName, func(string) bool { return true },
		func(_ string, libAccepts bool) []string {
			if libAccepts {
				return []string{"SI-020"}
			}
			return nil
		})
	for _, s := range payloadgrammar.Inputs(t, payloadgrammar.PersonalName) {
		if !o.Accepts(payloadgrammar.PersonalName, s) {
			continue
		}
		wantGiven, wantSurname, wantSuffix := strings.TrimSpace(s), "", ""
		if parts := strings.Split(s, "/"); len(parts) == 3 {
			wantGiven, wantSurname, wantSuffix = strings.TrimSpace(parts[0]), parts[1], strings.TrimSpace(parts[2])
		}
		given, surname, suffix := SplitPersonalName(s)
		if given != wantGiven || surname != wantSurname || suffix != wantSuffix {
			t.Errorf("SplitPersonalName(%q) = %q, %q, %q; the grammar splits it %q, %q, %q",
				s, given, surname, suffix, wantGiven, wantSurname, wantSuffix)
		}
	}
}

// TestIsPointerXRefMatchesGrammar holds IsPointerXRef to the Xref grammar.
// It accepts any character but whitespace and "@" where 7.0 has only
// tagchar (SI-010).
func TestIsPointerXRefMatchesGrammar(t *testing.T) {
	payloadgrammar.Compare(t, payloadgrammar.Xref, IsPointerXRef, func(s string, libAccepts bool) []string {
		if libAccepts {
			return []string{"SI-010"}
		}
		return nil
	})
}
