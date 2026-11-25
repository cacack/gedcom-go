// Package payloadgrammar is the test oracle for the library's GEDCOM 7.0
// payload parsers: the specification's own ABNF, vendored as
// testdata/spec/gedcom-7.0/grammar.abnf, plus the constraints the
// specification states in prose beside it (ADR 0013).
//
// It is for tests only. A test feeds [Inputs] to [Oracle.Accepts] and to the
// library's parser or check for the same datatype, and fails on a
// disagreement the spec interpretation register (internal/specinterp) does
// not list. No library package may import it.
package payloadgrammar

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/cacack/gedcom-go/v3/internal/abnf"
	"github.com/cacack/gedcom-go/v3/internal/corpora"
	"github.com/cacack/gedcom-go/v3/internal/specinterp"
)

// Path is the vendored grammar's location, relative to the repository root.
const Path = "testdata/spec/gedcom-7.0/grammar.abnf"

// Type is a payload datatype the oracle checks, named for its ABNF rule.
type Type string

// The datatypes the oracle checks.
const (
	DateValue    Type = "DateValue"
	DateExact    Type = "DateExact"
	Time         Type = "Time"
	Age          Type = "Age"
	PersonalName Type = "PersonalName"
	Latitude     Type = "Latitude"
	Longitude    Type = "Longitude"
	Xref         Type = "Xref"
)

// Types lists every Type.
var Types = []Type{DateValue, DateExact, Time, Age, PersonalName, Latitude, Longitude, Xref}

// Required lists every rule the oracle names: the datatypes, the rules its
// prose checks match tokens against, the rules the generator steers, and the
// rules the tests match whole lines and pointers against. [Load] fails when
// the grammar defines any of them under no name.
var Required = []string{
	"DateValue", "DateExact", "Time", "Age", "PersonalName", "Latitude", "Longitude", "Xref",
	"date", "calendar", "month", "epoch", "day", "year", "Integer",
	"stdTag", "extTag", "banned", "pointer", "voidPtr", "Line",
}

// Oracle answers whether a string is a valid payload of a datatype.
type Oracle struct {
	g *abnf.Grammar
}

var (
	loadOnce sync.Once
	loaded   *Oracle
	errLoad  error
)

// Load returns the oracle built from the vendored grammar, failing tb if it
// does not read or lacks a [Required] rule.
func Load(tb testing.TB) *Oracle {
	tb.Helper()
	loadOnce.Do(func() {
		root, err := corpora.RepoRoot()
		if err != nil {
			errLoad = err
			return
		}
		src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(Path))) // #nosec G304 -- the vendored grammar under the repository root
		if err != nil {
			errLoad = err
			return
		}
		loaded, errLoad = New(string(src))
	})
	if errLoad != nil {
		tb.Fatalf("%s: %v", Path, errLoad)
	}
	return loaded
}

// New builds an oracle from grammar source, failing if it lacks a
// [Required] rule.
func New(src string) (*Oracle, error) {
	g, err := abnf.Parse(src)
	if err != nil {
		return nil, err
	}
	if missing := Missing(g); len(missing) > 0 {
		return nil, fmt.Errorf("no rule %s, which the oracle depends on", strings.Join(missing, ", "))
	}
	return &Oracle{g: g}, nil
}

// Missing returns the [Required] rules g does not define.
func Missing(g *abnf.Grammar) []string {
	var missing []string
	for _, rule := range Required {
		if !g.Has(rule) {
			missing = append(missing, rule)
		}
	}
	return missing
}

// Grammar returns the parsed grammar.
func (o *Oracle) Grammar() *abnf.Grammar {
	return o.g
}

// Match reports whether s matches rule, which must be [Required].
func (o *Oracle) Match(rule, s string) bool {
	if !slices.Contains(Required, rule) {
		panic(fmt.Sprintf("payloadgrammar: rule %s is not in Required", rule))
	}
	ok, err := o.g.Match(rule, s)
	if err != nil {
		panic(fmt.Sprintf("payloadgrammar: %v", err)) // New checked every Required rule
	}
	return ok
}

// Accepts reports whether s is a valid payload of t: it matches t's rule and
// meets the constraints the specification states beside the rule.
//
//   - Dates (7.0 §2.4 and Appendix A): a standard calendar decides its
//     standard months and epochs. GREGORIAN and JULIAN have JAN to DEC and
//     the epoch BCE, FRENCH_R has VEND to COMP and HEBREW TSH to ELL, neither
//     with an epoch; an absent calendar is GREGORIAN, and a DateExact is
//     Gregorian. A day of a standard month is between 1 and the most days the
//     month has in any year (31 for JAN, 29 for FEB, 6 for COMP), and a year
//     of a standard calendar is at least 1: Appendix A says Gregorian and
//     Julian have no year 0, and names 1 Vendémiaire 1 and 1 Tishrei 1 as the
//     first days of the other two.
//
// Whether an extension tag is valid depends on the document: a SCHMA can
// document an extension calendar, month or epoch as a standard one
// (gedcom.io's extensions.ged writes FRENCH_R 2 _JOUR 8, _JOUR being
// month-COMP). The oracle has no document, so it accepts what is valid in
// some document: an extension calendar takes whatever the grammar allows, and
// a standard calendar also takes extension months and epochs.
//
// The other constraints:
//
//   - Latitude and Longitude (§2.10 and §2.11): the degrees are at most 90
//     and 180.
//   - Xref (§1.3): not @VOID@, the comment on the rule.
func (o *Oracle) Accepts(t Type, s string) bool {
	if !o.Match(string(t), s) {
		return false
	}
	switch t {
	case DateValue:
		return o.dateValueProse(s)
	case DateExact:
		return o.dateProse(strings.Split(s, " "), "GREGORIAN", false)
	case Latitude:
		return degreesAtMost(s, 90)
	case Longitude:
		return degreesAtMost(s, 180)
	case Xref:
		return s != "@VOID@"
	}
	return true
}

func degreesAtMost(s string, limit float64) bool {
	v, err := strconv.ParseFloat(s[1:], 64)
	return err == nil && v <= limit
}

// dateRestrict are the words dateRestrict names. No calendar, month or
// epoch is one of them, so in a DateValue they stand only where DateValue
// puts its keywords, and they divide it into its dates.
var dateRestrict = []string{"FROM", "TO", "BET", "AND", "BEF", "AFT", "ABT", "CAL", "EST"}

// dateKeywords are the keyword sequences DateValue allows, each keyword
// followed by a date; the empty sequence is a bare date.
var dateKeywords = []string{"", "TO", "FROM", "FROM TO", "BET AND", "AFT", "BEF", "ABT", "CAL", "EST"}

// dateValueProse checks a DateValue that matches the grammar: its keywords
// stand where DateValue puts them, and each of its dates meets the prose.
func (o *Oracle) dateValueProse(s string) bool {
	if s == "" {
		return true
	}
	var keywords []string
	dates := [][]string{nil}
	for _, tok := range strings.Split(s, " ") {
		if slices.Contains(dateRestrict, tok) {
			keywords = append(keywords, tok)
			dates = append(dates, nil)
			continue
		}
		dates[len(dates)-1] = append(dates[len(dates)-1], tok)
	}
	if len(keywords) > 0 {
		if len(dates[0]) > 0 {
			return false // a date before the first keyword
		}
		dates = dates[1:]
	}
	if !slices.Contains(dateKeywords, strings.Join(keywords, " ")) {
		return false
	}
	for _, date := range dates {
		if len(date) == 0 || !o.dateProse(date, "", true) {
			return false
		}
	}
	return true
}

// calendarMonths gives each standard calendar's months and the most days
// each has in any year.
var calendarMonths = map[string]map[string]int{
	"GREGORIAN": gregorianMonths,
	"JULIAN":    gregorianMonths,
	"FRENCH_R": {
		"VEND": 30, "BRUM": 30, "FRIM": 30, "NIVO": 30, "PLUV": 30, "VENT": 30,
		"GERM": 30, "FLOR": 30, "PRAI": 30, "MESS": 30, "THER": 30, "FRUC": 30, "COMP": 6, //nolint:misspell // THER is Thermidor
	},
	"HEBREW": {
		"TSH": 30, "CSH": 30, "KSL": 30, "TVT": 29, "SHV": 30, "ADR": 30, "ADS": 29,
		"NSN": 30, "IYR": 29, "SVN": 30, "TMZ": 29, "AAV": 30, "ELL": 29,
	},
}

var gregorianMonths = map[string]int{
	"JAN": 31, "FEB": 29, "MAR": 31, "APR": 30, "MAY": 31, "JUN": 30,
	"JUL": 31, "AUG": 31, "SEP": 30, "OCT": 31, "NOV": 30, "DEC": 31,
}

// calendarEpochs gives each standard calendar's epochs.
var calendarEpochs = map[string][]string{"GREGORIAN": {"BCE"}, "JULIAN": {"BCE"}}

// dateProse reports whether some reading of the tokens of one date, as
// [calendar] [[day] month] year [epoch], meets the prose constraints. With
// mayName unset the calendar is fixed and no token names it.
func (o *Oracle) dateProse(toks []string, calendar string, mayName bool) bool {
	if mayName && len(toks) > 1 && o.Match("calendar", toks[0]) && o.dateProse(toks[1:], toks[0], false) {
		return true
	}
	if calendar == "" {
		calendar = "GREGORIAN"
	}
	months, standard := calendarMonths[calendar]
	if !standard {
		return true // an extension calendar's own rules, which a SCHMA may document
	}
	if n := len(toks); n > 1 && o.Match("epoch", toks[n-1]) {
		epoch := toks[n-1]
		epochOK := o.Match("extTag", epoch) || slices.Contains(calendarEpochs[calendar], epoch)
		if epochOK && o.dateParts(toks[:n-1], months) {
			return true
		}
	}
	return o.dateParts(toks, months)
}

// dateParts checks [[day] month] year in a standard calendar with months.
func (o *Oracle) dateParts(toks []string, months map[string]int) bool {
	n := len(toks)
	if n == 0 || n > 3 || !o.Match("year", toks[n-1]) || atoi(toks[n-1]) < 1 {
		return false
	}
	if n == 1 {
		return true
	}
	month := toks[n-2]
	if !o.Match("month", month) {
		return false
	}
	extension := o.Match("extTag", month)
	if !extension && months[month] == 0 {
		return false
	}
	if n == 2 {
		return true
	}
	if !o.Match("day", toks[0]) {
		return false
	}
	day := atoi(toks[0])
	return extension || day >= 1 && day <= months[month]
}

// atoi reads an Integer, saturating rather than overflowing.
func atoi(s string) int {
	n := 0
	for _, c := range s {
		if n > 1e8 {
			return n
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// Departures names the register rows that allow the library to disagree with
// the oracle about s, where libAccepts is the library's answer; none means
// the disagreement is a defect.
type Departures func(s string, libAccepts bool) []string

// Compare runs every input of t through the oracle and through lib, the
// library's parser or check, and fails tb on each disagreement departures
// does not allow. It logs how many disagreements each row allowed.
func Compare(tb testing.TB, t Type, lib func(string) bool, departures Departures) {
	tb.Helper()
	o := Load(tb)
	allowed := map[string]int{}
	failures := 0
	for _, s := range Inputs(tb, t) {
		if ids, ok := o.check(tb, t, s, lib, departures); !ok {
			failures++
			if failures <= maxReported {
				tb.Errorf("%s %q: library accepts = %v, grammar accepts = %v, and no register row allows it",
					t, s, lib(s), o.Accepts(t, s))
			}
		} else {
			for _, id := range ids {
				allowed[id]++
			}
		}
	}
	if failures > maxReported {
		tb.Errorf("%s: %d more disagreements", t, failures-maxReported)
	}
	for _, id := range slices.Sorted(maps.Keys(allowed)) {
		tb.Logf("%s: %d disagreements allowed by %s", t, allowed[id], id)
	}
}

// Check is [Compare] for one input, as a fuzz target needs.
func Check(tb testing.TB, t Type, s string, lib func(string) bool, departures Departures) {
	tb.Helper()
	o := Load(tb)
	if _, ok := o.check(tb, t, s, lib, departures); !ok {
		tb.Errorf("%s %q: library accepts = %v, grammar accepts = %v, and no register row allows it",
			t, s, lib(s), o.Accepts(t, s))
	}
}

// maxReported caps the disagreements Compare reports one by one.
const maxReported = 25

// check reports whether the library agrees with the oracle about s or a
// register row allows the disagreement, and which rows did.
func (o *Oracle) check(tb testing.TB, t Type, s string, lib func(string) bool, departures Departures) ([]string, bool) {
	tb.Helper()
	libAccepts := lib(s)
	if libAccepts == o.Accepts(t, s) {
		return nil, true
	}
	ids := departures(s, libAccepts)
	for _, id := range ids {
		requireRow(tb, id)
	}
	return ids, len(ids) > 0
}

var (
	rowsMu sync.Mutex
	rows   = map[string]bool{} // the register rows already found
)

// requireRow is specinterp.Require, reading the register once per row.
func requireRow(tb testing.TB, id string) {
	tb.Helper()
	rowsMu.Lock()
	defer rowsMu.Unlock()
	if !rows[id] {
		specinterp.Require(tb, id)
		rows[id] = true
	}
}
