package validator

import (
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// life describes the birth and death of a test individual; an empty field is
// not recorded.
type life struct {
	birth, birthPlace string
	death, deathPlace string
}

// lifeEvent builds an event of the given type from a GEDCOM date and a place,
// or returns nil when neither is given.
func lifeEvent(t *testing.T, eventType gedcom.EventType, date, place string) *gedcom.Event {
	t.Helper()
	if date == "" && place == "" {
		return nil
	}
	event := &gedcom.Event{Type: eventType}
	if date != "" {
		parsed, err := gedcom.ParseDate(date)
		if err != nil {
			t.Fatalf("ParseDate(%q): %v", date, err)
		}
		event.Date, event.ParsedDate = date, parsed
	}
	event.SetPlaceName(place)
	return event
}

// lifePerson returns a John /Example/, sex M, with the given life.
func lifePerson(t *testing.T, xref string, l life) *gedcom.Individual {
	t.Helper()
	ind := &gedcom.Individual{
		XRef:  xref,
		Names: []*gedcom.PersonalName{{Full: "John /Example/"}},
		Sex:   "M",
	}
	for _, event := range []*gedcom.Event{
		lifeEvent(t, gedcom.EventBirth, l.birth, l.birthPlace),
		lifeEvent(t, gedcom.EventDeath, l.death, l.deathPlace),
	} {
		if event != nil {
			ind.Events = append(ind.Events, event)
		}
	}
	return ind
}

// baseReasons are the reasons of a pair agreeing on name, birth year and sex,
// which scores 0.9.
var baseReasons = []string{"exact surname match", "exact given name match", "same birth year", "same sex"}

func assertConfidence(t *testing.T, pair DuplicatePair, want float64) {
	t.Helper()
	if math.Abs(pair.Confidence-want) > 1e-9 {
		t.Errorf("confidence = %v, want %v (reasons %v)", pair.Confidence, want, pair.MatchReasons)
	}
}

// TestFindDuplicates_LifeSignals pins each of the #906 signals on its own:
// its weight, its reason, and the cases that must not earn it.
func TestFindDuplicates_LifeSignals(t *testing.T) {
	tests := []struct {
		name        string
		a, b        life
		want        float64
		wantReasons []string // appended to baseReasons
	}{
		{
			name: "name, birth year and sex alone",
			a:    life{birth: "1850"}, b: life{birth: "1850"},
			want: 0.9,
		},
		{
			name: "full birth date",
			a:    life{birth: "3 MAR 1850"}, b: life{birth: "3 MAR 1850"},
			want: 0.94, wantReasons: []string{"same birth date"},
		},
		{
			name: "birth date missing a day on one side",
			a:    life{birth: "3 MAR 1850"}, b: life{birth: "MAR 1850"},
			want: 0.9,
		},
		{
			name: "birth date missing a day on both sides",
			a:    life{birth: "MAR 1850"}, b: life{birth: "MAR 1850"},
			want: 0.9,
		},
		{
			name: "birth date with a modifier",
			a:    life{birth: "3 MAR 1850"}, b: life{birth: "ABT 3 MAR 1850"},
			want: 0.9,
		},
		{
			name: "different birth day",
			a:    life{birth: "3 MAR 1850"}, b: life{birth: "25 DEC 1850"},
			want: 0.9,
		},
		{
			name: "same day and month in different calendars",
			a:    life{birth: "3 MAR 1850"}, b: life{birth: "@#DJULIAN@ 3 MAR 1850"},
			want: 0.9,
		},
		{
			name: "death year",
			a:    life{birth: "1850", death: "9 OCT 1920"}, b: life{birth: "1850", death: "1920"},
			want: 0.93, wantReasons: []string{"same death year"},
		},
		{
			name: "approximate death date earns no credit",
			a:    life{birth: "1850", death: "ABT 1920"}, b: life{birth: "1850", death: "1920"},
			want: 0.9,
		},
		{
			name: "death date on one side only",
			a:    life{birth: "1850", death: "1920"}, b: life{birth: "1850"},
			want: 0.9,
		},
		{
			name: "birth place",
			a:    life{birth: "1850", birthPlace: "Springfield, Example County"},
			b:    life{birth: "1850", birthPlace: "Springfield, Example County"},
			want: 0.92, wantReasons: []string{"same birth place"},
		},
		{
			name: "birth place compared whole",
			a:    life{birth: "1850", birthPlace: "Springfield, Example County"},
			b:    life{birth: "1850", birthPlace: "Springfield"},
			want: 0.9,
		},
		{
			name: "birth place on one side only",
			a:    life{birth: "1850", birthPlace: "Springfield"}, b: life{birth: "1850"},
			want: 0.9,
		},
		{
			name: "death place",
			a:    life{birth: "1850", deathPlace: "Springfield"}, b: life{birth: "1850", deathPlace: "Springfield"},
			want: 0.91, wantReasons: []string{"same death place"},
		},
		{
			name: "a birth place does not match a death place",
			a:    life{birth: "1850", birthPlace: "Springfield"}, b: life{birth: "1850", deathPlace: "Springfield"},
			want: 0.9,
		},
		{
			name: "every signal",
			a:    life{birth: "3 MAR 1850", birthPlace: "Springfield", death: "9 OCT 1920", deathPlace: "Othertown"},
			b:    life{birth: "3 MAR 1850", birthPlace: "Springfield", death: "9 OCT 1920", deathPlace: "Othertown"},
			want: 1.0,
			wantReasons: []string{
				"same birth date", "same death year", "same birth place", "same death place",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := twoIndividualDoc(lifePerson(t, "@I1@", tt.a), lifePerson(t, "@I2@", tt.b))
			got := NewDuplicateDetector(nil).FindDuplicates(doc)
			if len(got) != 1 {
				t.Fatalf("got %d pairs, want 1", len(got))
			}
			assertConfidence(t, got[0], tt.want)
			if got[0].Confidence > 1.0 {
				t.Errorf("confidence = %v, exceeds the documented maximum 1.0", got[0].Confidence)
			}
			want := append(slices.Clone(baseReasons), tt.wantReasons...)
			if !slices.Equal(got[0].MatchReasons, want) {
				t.Errorf("reasons = %v, want %v", got[0].MatchReasons, want)
			}
		})
	}
}

// TestFindDuplicates_DeathVeto: exact death years further apart than
// MaxBirthYearDiff veto the pair as exact birth years do; anything short of
// two exact dates does not.
func TestFindDuplicates_DeathVeto(t *testing.T) {
	tests := []struct {
		name           string
		death1, death2 string
		config         DuplicateConfig
		wantPair       bool
		wantDeathYear  bool
	}{
		{name: "same year", death1: "1920", death2: "1920", wantPair: true, wantDeathYear: true},
		{name: "at the tolerance", death1: "1920", death2: "1922", wantPair: true},
		{name: "one year past the tolerance", death1: "1920", death2: "1923"},
		{name: "decades apart", death1: "9 OCT 1920", death2: "1 JAN 1851"},
		{
			name:   "veto holds with no confidence threshold",
			death1: "1920", death2: "1923",
			config: DuplicateConfig{MinConfidence: ZeroThreshold},
		},
		{
			name:   "ZeroThreshold vetoes any difference",
			death1: "1920", death2: "1921",
			config: DuplicateConfig{MaxBirthYearDiff: ZeroThreshold},
		},
		{
			name:   "ZeroThreshold keeps the same year",
			death1: "1920", death2: "1920",
			config:   DuplicateConfig{MaxBirthYearDiff: ZeroThreshold},
			wantPair: true, wantDeathYear: true,
		},
		{
			name:   "wider tolerance",
			death1: "1920", death2: "1925",
			config:   DuplicateConfig{MaxBirthYearDiff: 5},
			wantPair: true,
		},
		{name: "approximate date does not veto", death1: "ABT 1851", death2: "1920", wantPair: true},
		{name: "bounded date does not veto", death1: "BEF 1851", death2: "1920", wantPair: true},
		{name: "range does not veto", death1: "BET 1851 AND 1860", death2: "1920", wantPair: true},
		{name: "missing date does not veto", death1: "", death2: "1920", wantPair: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := twoIndividualDoc(
				lifePerson(t, "@I1@", life{birth: "1850", death: tt.death1}),
				lifePerson(t, "@I2@", life{birth: "1850", death: tt.death2}),
			)
			config := tt.config
			got := NewDuplicateDetector(&config).FindDuplicates(doc)
			if !tt.wantPair {
				if len(got) != 0 {
					t.Errorf("want no pair, got confidence %.2f reasons %v", got[0].Confidence, got[0].MatchReasons)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("got %d pairs, want 1", len(got))
			}
			want := 0.9
			if tt.wantDeathYear {
				want = 0.93
			}
			assertConfidence(t, got[0], want)
			if has := slices.Contains(got[0].MatchReasons, "same death year"); has != tt.wantDeathYear {
				t.Errorf("reasons = %v; same death year present = %v, want %v", got[0].MatchReasons, has, tt.wantDeathYear)
			}
		})
	}
}

// TestFindDuplicates_PlaceNormalization: places differing only in case,
// accents or surrounding space match, unless DisableNameNormalization is set.
func TestFindDuplicates_PlaceNormalization(t *testing.T) {
	doc := func() *gedcom.Document {
		return twoIndividualDoc(
			lifePerson(t, "@I1@", life{birth: "1850", birthPlace: "Montréal, Québec", deathPlace: "SPRINGFIELD"}),
			lifePerson(t, "@I2@", life{birth: "1850", birthPlace: " montreal, quebec ", deathPlace: "Springfield"}),
		)
	}

	got := NewDuplicateDetector(nil).FindDuplicates(doc())
	if len(got) != 1 {
		t.Fatalf("normalized: got %d pairs, want 1", len(got))
	}
	assertConfidence(t, got[0], 0.93)
	for _, reason := range []string{"same birth place", "same death place"} {
		if !slices.Contains(got[0].MatchReasons, reason) {
			t.Errorf("normalized: reasons = %v, want %q", got[0].MatchReasons, reason)
		}
	}

	got = NewDuplicateDetector(&DuplicateConfig{DisableNameNormalization: true}).FindDuplicates(doc())
	if len(got) != 1 {
		t.Fatalf("not normalized: got %d pairs, want 1", len(got))
	}
	assertConfidence(t, got[0], 0.9)
	if !slices.Equal(got[0].MatchReasons, baseReasons) {
		t.Errorf("not normalized: reasons = %v, want %v", got[0].MatchReasons, baseReasons)
	}
}

// issue906A and issue906B are the reproduction files of #906. b's second
// individual shares a's name, birth year and sex but was born on another day,
// elsewhere, and died 69 years earlier.
const issue906A = `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Example/
1 SEX M
1 BIRT
2 DATE 3 MAR 1850
2 PLAC Springfield, Example County
1 DEAT
2 DATE 9 OCT 1920
0 TRLR
`

const issue906B = `0 HEAD
1 GEDC
2 VERS 5.5
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Example/
1 SEX M
1 BIRT
2 DATE 3 MAR 1850
2 PLAC Springfield, Example County
1 DEAT
2 DATE 9 OCT 1920
0 @I2@ INDI
1 NAME John /Example/
1 SEX M
1 BIRT
2 DATE 25 DEC 1850
2 PLAC Othertown, Other County
1 DEAT
2 DATE 1 JAN 1851
0 TRLR
`

func decodeGEDCOM(t *testing.T, data string) *gedcom.Document {
	t.Helper()
	doc, err := decoder.Decode(strings.NewReader(data))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	return doc
}

// TestFindDuplicatesBetween_Issue906 runs the issue's reproduction: the pair
// with an identical recorded life scores 0.99 and the namesake is vetoed by
// its death date. Without that death date the namesake is reported, and
// ranks below.
func TestFindDuplicatesBetween_Issue906(t *testing.T) {
	// No confidence threshold, as the issue ran it (--min-score 0).
	detector := NewDuplicateDetector(&DuplicateConfig{MinConfidence: ZeroThreshold})
	a := decodeGEDCOM(t, issue906A)

	t.Run("as reported", func(t *testing.T) {
		pairs := detector.FindDuplicatesBetween(a, decodeGEDCOM(t, issue906B))
		if got, want := sortedPairKeys(pairs), []string{"@I1@|@I1@"}; !slices.Equal(got, want) {
			t.Fatalf("pairs = %v, want %v", got, want)
		}
		assertConfidence(t, pairs[0], 0.99)
		want := append(slices.Clone(baseReasons), "same birth date", "same death year", "same birth place")
		if !slices.Equal(pairs[0].MatchReasons, want) {
			t.Errorf("reasons = %v, want %v", pairs[0].MatchReasons, want)
		}
	})

	t.Run("namesake with no death date", func(t *testing.T) {
		b := decodeGEDCOM(t, strings.Replace(issue906B, "1 DEAT\n2 DATE 1 JAN 1851\n", "", 1))
		pairs := detector.FindDuplicatesBetween(a, b)
		if got, want := sortedPairKeys(pairs), []string{"@I1@|@I1@", "@I1@|@I2@"}; !slices.Equal(got, want) {
			t.Fatalf("pairs = %v, want %v", got, want)
		}
		conf := map[string]DuplicatePair{}
		for _, p := range pairs {
			conf[p.Individual2.XRef] = p
		}
		assertConfidence(t, conf["@I1@"], 0.99)
		assertConfidence(t, conf["@I2@"], 0.9)
		if conf["@I1@"].Confidence <= conf["@I2@"].Confidence {
			t.Errorf("identical life scored %v, not above the namesake's %v",
				conf["@I1@"].Confidence, conf["@I2@"].Confidence)
		}
		if !slices.Equal(conf["@I2@"].MatchReasons, baseReasons) {
			t.Errorf("namesake reasons = %v, want %v", conf["@I2@"].MatchReasons, baseReasons)
		}
	})
}

// TestFindDuplicates_EnglishBirthDateEarnsYearCredit: a birth date spelled
// out in English decodes to a parsed date (#907), so it earns birth-year
// credit against the same year written as GEDCOM.
func TestFindDuplicates_EnglishBirthDateEarnsYearCredit(t *testing.T) {
	doc := decodeGEDCOM(t, `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Example/
1 SEX M
1 BIRT
2 DATE June 1880
0 @I2@ INDI
1 NAME John /Example/
1 SEX M
1 BIRT
2 DATE 1880
0 TRLR
`)

	got := NewDuplicateDetector(nil).FindDuplicates(doc)
	if len(got) != 1 {
		t.Fatalf("got %d pairs, want 1", len(got))
	}
	assertConfidence(t, got[0], 0.9)
	if !slices.Equal(got[0].MatchReasons, baseReasons) {
		t.Errorf("reasons = %v, want %v", got[0].MatchReasons, baseReasons)
	}
}
