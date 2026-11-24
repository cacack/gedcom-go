// duplicates.go provides duplicate individual detection based on name similarity
// and birth date proximity.
//
// Duplicate detection is a common need in genealogy software where the same person
// may be entered multiple times from different sources. This package provides
// configurable matching thresholds for name comparison and date proximity.
// Contradictions veto a pair outright: exact birth years, or exact death
// years, further apart than DuplicateConfig.MaxBirthYearDiff, or two known
// sexes (M, F or X, compared case-insensitively) that differ. The sex veto has
// no switch.

package validator

import (
	"context"
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/cacack/gedcom-go/v3/gedcom"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// DefaultMaxGroupSize is the default per-surname-group ceiling applied by
// duplicate detection.
//
// The value is derived from the testdata corpus: the largest normalized-surname
// group anywhere in it is 519, in testdata/gedcom-5.5.1/longsword.ged (203,154
// individuals, 55,801 groups); the next largest is 70. 1000 therefore leaves
// roughly 2x headroom over the worst real-world observation while still bounding
// the quadratic pair sweep on a hostile document.
const DefaultMaxGroupSize = 1000

// UnlimitedGroupSize disables the per-surname-group ceiling when assigned to
// [DuplicateConfig.MaxGroupSize], restoring the unbounded pre-#530 sweep.
//
// It exists because the field's zero value means "use DefaultMaxGroupSize", not
// "no limit" — the opposite of the MaxErrors convention elsewhere in this
// package. Reaching for a named constant makes opting out deliberate and
// greppable rather than a guessed magic number.
//
// Only use it on input you control; on attacker-supplied documents it reopens
// the quadratic blowup described in docs/decisions/0009-bounded-duplicate-detection.md.
const UnlimitedGroupSize = -1

// DefaultMaxNameRunes is the default bound on the given-name length duplicate
// detection compares (#717). Given names longer than this are compared on
// their first DefaultMaxNameRunes runes.
//
// Per-pair cost is Levenshtein's O(len₁·len₂), so without this bound two
// 100,000-character names, a 200 KB file, cost 51 s; at the bound a pair
// costs at most 64² cell updates. The value was set against the testdata
// corpus, where names over it are titles stored in GIVN and comparing them
// on a prefix changes no reported pair; the measurement lives in
// docs/decisions/0009-bounded-duplicate-detection.md ("Why 64").
const DefaultMaxNameRunes = 64

// UnlimitedNameRunes disables the given-name length bound when assigned to
// [DuplicateConfig.MaxNameRunes], comparing names at their full length.
//
// As with [UnlimitedGroupSize], the field's zero value means "use
// DefaultMaxNameRunes", so opting out has to be explicit. Only use it on
// input you control; on attacker-supplied documents a pair of long names
// costs the product of their lengths.
const UnlimitedNameRunes = -1

// ZeroThreshold requests an effective value of exactly zero when assigned to
// [DuplicateConfig.MinNameSimilarity], [DuplicateConfig.MinConfidence] or
// [DuplicateConfig.MaxBirthYearDiff].
//
// It exists because those fields' zero value means "use the default", so that a
// partial struct literal such as &DuplicateConfig{MinConfidence: 0.9} keeps the
// other thresholds in force (#555). A literal 0 therefore cannot express "no
// threshold"; any negative value does, and this named constant makes that
// deliberate and greppable, in the same way [UnlimitedGroupSize] does for
// [DuplicateConfig.MaxGroupSize]. The constant is untyped, so it assigns to both
// the float64 and the int fields.
//
// Its effect per field:
//
//	MinNameSimilarity  no given-name gate; any pair of given names is accepted
//	MinConfidence      every pair that passes the other gates is reported
//	MaxBirthYearDiff   only an identical birth year earns birth-date credit;
//	                   any difference between exact birth years, or between
//	                   exact death years, vetoes the pair
const ZeroThreshold = -1

// Default thresholds applied by duplicate detection when the corresponding
// DuplicateConfig field is left zero.
const (
	defaultMinNameSimilarity = 0.8
	defaultMaxBirthYearDiff  = 2
	defaultMinConfidence     = 0.7
)

// Weights of the signals that tell a pair agreeing on the whole recorded life
// from one agreeing on name, birth year and sex alone (#906). They share the
// 0.1 those four signals leave unused, so together they cannot outweigh any
// one of them; the full table is on [DuplicatePair.Confidence].
const (
	weightSameBirthDate  = 0.04
	weightSameDeathYear  = 0.03
	weightSameBirthPlace = 0.02
	weightSameDeathPlace = 0.01
)

// Weights of the relatives signal, scored only under
// DuplicateConfig.CompareRelatives (#932). A married-name/maiden-name pair
// earns no surname weight, so these take its place: the same spouse and two
// shared children are worth 0.4.
const (
	weightSameSpouse  = 0.2
	weightSharedChild = 0.1

	// maxSharedChildrenScored is how many shared children earn weight.
	maxSharedChildrenScored = 2
)

// maxRelativesRead bounds how many spouse links, and how many child links,
// are read for one individual under CompareRelatives. Links are pointers the
// document's author controls, and many individuals can name one family of
// many children, so without it reading relatives would cost the product of
// the two. No real person has this many spouses or children; an individual
// whose relatives were cut is counted on the DUPLICATE_DETECTION_LIMITED
// notice ("truncated_relatives").
const maxRelativesRead = 64

// DuplicateConfig contains configuration options for duplicate detection.
//
// Every field's zero value is the default, so a partial struct literal such as
// &DuplicateConfig{MinConfidence: 0.9} changes only the field it names. The
// numeric fields therefore reserve negative values for "explicitly zero" or
// "unlimited" — see [ZeroThreshold], [UnlimitedGroupSize] and
// [UnlimitedNameRunes].
type DuplicateConfig struct {
	// DisableNameNormalization turns off name normalization (trim, lowercase,
	// remove accents) before surnames are grouped and given names compared.
	// With it set, "Smith" and "smith" land in different surname groups.
	// Default: false (names are normalized)
	DisableNameNormalization bool

	// MinNameSimilarity is the minimum similarity threshold for given name comparison.
	// Range: 0.0 to 1.0, where 1.0 is exact match.
	//
	//	0        use the default (0.8)
	//	negative no threshold; any given names are accepted ([ZeroThreshold])
	//	positive use that value
	//
	// Default: 0.8
	MinNameSimilarity float64

	// MaxBirthYearDiff is the maximum difference in birth years that still earns
	// partial birth-date credit. An identical birth year always earns full credit.
	// It is also a veto: when both birth dates are exact (no ABT, BEF, AFT, BET,
	// FROM or other modifier) and their years differ by more than
	// MaxBirthYearDiff, the pair is never reported, whatever its other scores,
	// so a namesake father and son are not taken for one person. An approximate
	// or range date outside the window only withholds credit. A birth date
	// with no known year (missing, a phrase, year 0) neither earns credit nor
	// vetoes (see RequireBirthDate). A large positive value effectively turns
	// the veto off.
	//
	// Death dates follow the same veto: when both death dates are exact and
	// their years differ by more than MaxBirthYearDiff, the pair is never
	// reported. Only an identical death year earns credit, and only between
	// two exact dates; an approximate, bounded, range or missing death date
	// neither earns credit nor vetoes.
	//
	// Years are compared astronomically, so 1 BCE and 1 CE are one year apart.
	// Dates in the same calendar compare Year as written; dates in different
	// calendars are compared as the Gregorian years of their first days, reading
	// a dual date in New Style.
	//
	//	0        use the default (2)
	//	negative only an identical birth year earns credit; any other exact birth or death year vetoes ([ZeroThreshold])
	//	positive use that value
	//
	// Default: 2
	MaxBirthYearDiff int

	// RequireBirthDate requires both individuals to have birth dates for a match.
	// If true, missing birth dates prevent a match.
	// Default: false
	RequireBirthDate bool

	// MinConfidence is the minimum overall confidence score for a match.
	// Range: 0.0 to 1.0
	//
	//	0        use the default (0.7)
	//	negative no threshold; report every pair that passes the other gates ([ZeroThreshold])
	//	positive use that value
	//
	// Default: 0.7
	MinConfidence float64

	// MaxGroupSize bounds how large a single normalized-surname group may be
	// before duplicate detection skips it. Comparison within a group is
	// quadratic in its size, and the surnames in a document are attacker
	// controlled, so an unbounded sweep turns a small upload into minutes of
	// CPU (#530).
	//
	// The semantics are deliberately tri-state rather than following the Go
	// zero-value convention:
	//
	//	0        use DefaultMaxGroupSize (1000)
	//	negative no cap; compare every group however large ([UnlimitedGroupSize])
	//	positive use that value
	//
	// Zero does NOT mean unlimited. A caller writing a partial struct literal
	// such as &DuplicateConfig{MinConfidence: 0.7} leaves this field zero
	// without intending to, and must stay protected rather than silently opt
	// back into unbounded work. That is a departure both from the usual Go
	// convention and from the MaxErrors precedent elsewhere in this package,
	// where zero means "no limit"; opting out here has to be explicit.
	//
	// Default: DefaultMaxGroupSize
	MaxGroupSize int

	// MaxNameRunes bounds the length of the given names duplicate detection
	// compares. Similarity is Levenshtein distance, quadratic in the name
	// lengths, and MaxGroupSize bounds how many pairs are compared but not
	// what one pair costs; two 100,000-character given names took 51 s
	// (#717). A given name longer than the bound is compared on its first
	// MaxNameRunes runes, and the individuals so compared are counted in the
	// DUPLICATE_DETECTION_LIMITED notice ("truncated_names"), since a verdict
	// reached on a prefix can differ from one on the whole name in either
	// direction, and the document's author chooses the names. A caller that
	// acts on reported pairs without a person in the loop should read
	// DuplicateReport.LimitIssues before trusting a pair.
	//
	// Tri-state, like MaxGroupSize and for the same reason:
	//
	//	0        use DefaultMaxNameRunes (64)
	//	negative no bound; compare names at full length ([UnlimitedNameRunes])
	//	positive use that value
	//
	// Default: DefaultMaxNameRunes
	MaxNameRunes int

	// CompareRelatives adds an individual's relatives to the evidence (#932):
	// the names of their spouses and of their children, read through their
	// FAMS families. A pair earns 0.2 when any spouse's name matches and 0.1
	// for each shared child's name, up to two (see [DuplicatePair.Confidence]),
	// and two individuals that share a spouse's name are compared even when
	// their surnames differ, so a woman under her married name in one file
	// and her maiden name in the other is found. A pair with different
	// surnames earns no surname weight and is still subject to every other
	// gate and veto, including MinNameSimilarity.
	//
	// Names are compared whole (given name and surname), after the same
	// normalization as the individuals' own names; a relative with no given
	// name is not compared. Individuals sharing a spouse's name form a group
	// bounded by MaxGroupSize like a surname group, and at most 64 spouse and
	// 64 child links are read per individual; both limits are reported on the
	// DUPLICATE_DETECTION_LIMITED notice.
	//
	// Default: false (relatives are not read, and surnames must match)
	CompareRelatives bool
}

// DefaultDuplicateConfig returns a new *DuplicateConfig with default values,
// matching DefaultDateLogicConfig and the DefaultOptions constructors.
//
// The result spells every default out explicitly; it is equivalent to a zero
// &DuplicateConfig{}, since each zero field selects the same default.
func DefaultDuplicateConfig() *DuplicateConfig {
	return &DuplicateConfig{
		DisableNameNormalization: false,
		MinNameSimilarity:        defaultMinNameSimilarity,
		MaxBirthYearDiff:         defaultMaxBirthYearDiff,
		RequireBirthDate:         false,
		MinConfidence:            defaultMinConfidence,
		MaxGroupSize:             DefaultMaxGroupSize,
		MaxNameRunes:             DefaultMaxNameRunes,
		CompareRelatives:         false,
	}
}

// resolveThreshold applies the zero-means-default, negative-means-zero rule
// shared by the numeric DuplicateConfig thresholds. NaN is treated like zero
// and selects the default: left as NaN it would compare false against every
// score and so silently disable the gate.
func resolveThreshold[T int | float64](value, def T) T {
	switch {
	case math.IsNaN(float64(value)):
		return def
	case value < 0:
		return 0
	case value == 0:
		return def
	default:
		return value
	}
}

// DuplicatePair represents a potential duplicate pair of individuals.
type DuplicatePair struct {
	// Individual1 is the first individual in the pair.
	Individual1 *gedcom.Individual

	// Individual2 is the second individual in the pair.
	Individual2 *gedcom.Individual

	// Confidence is the overall match confidence score (0.0 to 1.0): the sum
	// of the weights of the signals the pair agrees on, each of which also
	// adds its reason to MatchReasons.
	//
	//	0.30  surname          equal ("exact surname match")
	//	0.30  given name       0.30 × similarity ("exact given name match",
	//	                       "similar given name (N%)")
	//	0.20  birth year       same year ("same birth year"); 0.10 when within
	//	                       MaxBirthYearDiff ("birth year within N years")
	//	0.10  sex              both known and equal ("same sex"), or unknown
	//	                       on one side or both ("sex not recorded on both")
	//	0.04  full birth date  both exact, in one calendar, with the same day,
	//	                       month and year ("same birth date")
	//	0.03  death year       both exact, same year ("same death year")
	//	0.02  birth place      both recorded and equal ("same birth place")
	//	0.01  death place      both recorded and equal ("same death place")
	//
	// Under [DuplicateConfig.CompareRelatives] only, relatives add to the sum:
	//
	//	0.20  spouse           a spouse's name on each side matches
	//	                       ("same spouse name")
	//	0.10  each child       per child's name both share, at most two
	//	                       ("1 shared child name", "N shared child names")
	//
	// and the sum is capped at 1.0. A pair whose surnames differ, found
	// through a shared spouse's name, earns no surname weight.
	//
	// The maximum is 1.0. A pair that agrees on name, birth year and sex alone
	// scores 0.9, so anything above that agrees on a full birth date, a death
	// year or a place as well. Nothing is subtracted, and a value recorded on
	// one side only never costs more than one neither side records: every
	// signal but sex withholds its weight in both cases, and sex earns it in
	// both, since an unrecorded sex is no evidence of two people (#932).
	// Contradictions are vetoes instead (see [DuplicateConfig.MaxBirthYearDiff]):
	// two known sexes that differ rule the pair out.
	//
	// Places are compared whole, as recorded, after the normalization names
	// get (trim, lowercase, remove accents) unless
	// [DuplicateConfig.DisableNameNormalization] is set; "Springfield" and
	// "Springfield, Example County" are different places.
	Confidence float64

	// MatchReasons contains human-readable descriptions of why these individuals matched.
	MatchReasons []string
}

// ToIssue converts the DuplicatePair to a validation Issue.
//
// The Issue names both XRefs but not the document each came from, so for a
// pair from FindDuplicatesBetween it reads like a same-document duplicate and
// is ambiguous when the two files reuse XRefs.
func (p DuplicatePair) ToIssue() Issue {
	message := fmt.Sprintf("Potential duplicate: %s may be the same as %s (%.0f%% confidence)",
		getDisplayName(p.Individual1),
		getDisplayName(p.Individual2),
		p.Confidence*100)

	issue := NewIssue(SeverityInfo, CodePotentialDuplicate, message, p.Individual1.XRef).
		WithRelatedXRef(p.Individual2.XRef).
		WithDetail("confidence", fmt.Sprintf("%.2f", p.Confidence))

	// Add match reasons as details
	for i, reason := range p.MatchReasons {
		issue = issue.WithDetail(fmt.Sprintf("reason_%d", i+1), reason)
	}

	return issue
}

// primaryName returns the first usable name on ind, or nil when it has none.
// A hand-built document can hold a nil element in Names, so Names[0] is not
// necessarily a name and a length check is not a nil check. Skipping nils
// matches the library-wide policy in docs/decisions/0007-error-transparency.md.
func primaryName(ind *gedcom.Individual) *gedcom.PersonalName {
	if ind == nil {
		return nil
	}
	for _, name := range ind.Names {
		if name != nil {
			return name
		}
	}
	return nil
}

// getDisplayName returns a display name for an individual.
func getDisplayName(ind *gedcom.Individual) string {
	if ind == nil {
		return ""
	}
	name := primaryName(ind)
	if name == nil {
		return ind.XRef
	}
	if name.Full != "" {
		// Remove slashes from GEDCOM format
		return strings.ReplaceAll(strings.ReplaceAll(name.Full, "/", ""), "  ", " ")
	}
	if name.Given != "" || name.Surname != "" {
		return strings.TrimSpace(name.Given + " " + name.Surname)
	}
	return ind.XRef
}

// DuplicateDetector detects potential duplicate individuals in a GEDCOM document.
type DuplicateDetector struct {
	config DuplicateConfig
}

// NewDuplicateDetector creates a new DuplicateDetector with the given configuration.
// If config is nil, default configuration is used.
//
// The detector keeps its own copy of config, so later changes to the caller's
// struct have no effect. Zero thresholds are resolved to their defaults and
// negative ones to zero at this point, as documented on [DuplicateConfig]
// (a NaN threshold is treated like zero); MaxGroupSize and MaxNameRunes are
// resolved on use by maxGroupSize and maxNameRunes.
func NewDuplicateDetector(config *DuplicateConfig) *DuplicateDetector {
	if config == nil {
		config = DefaultDuplicateConfig()
	}
	resolved := *config
	resolved.MinNameSimilarity = resolveThreshold(config.MinNameSimilarity, defaultMinNameSimilarity)
	resolved.MaxBirthYearDiff = resolveThreshold(config.MaxBirthYearDiff, defaultMaxBirthYearDiff)
	resolved.MinConfidence = resolveThreshold(config.MinConfidence, defaultMinConfidence)
	return &DuplicateDetector{config: resolved}
}

// candidate carries an individual alongside the normalized names and places
// used to compare it. The keys are derived once per individual during grouping
// rather than re-derived for every pair: normalizeName runs a Unicode NFD
// transform chain, and doing that inside the O(n^2) pair loop dominated the
// cost of FindDuplicates on large documents (#529).
type candidate struct {
	ind     *gedcom.Individual
	surname string
	given   string

	// birthPlace and deathPlace are the whole place strings, normalized as
	// names are; "" when not recorded.
	birthPlace string
	deathPlace string

	// spouses and children are the sorted, distinct name keys of the
	// individual's relatives (see relativeKey); nil unless CompareRelatives.
	spouses  []string
	children []string
}

// maxGroupSize resolves DuplicateConfig.MaxGroupSize to the ceiling to compare
// group sizes against. It is the single home of the tri-state rule documented on
// the field: negative means unlimited, zero means the default, positive is taken
// as given.
//
// The unlimited case returns math.MaxInt rather than a sentinel so every call
// site stays a plain `len(group) > limit` with no special case to forget.
func (d *DuplicateDetector) maxGroupSize() int {
	switch {
	case d.config.MaxGroupSize < 0:
		return math.MaxInt
	case d.config.MaxGroupSize == 0:
		return DefaultMaxGroupSize
	default:
		return d.config.MaxGroupSize
	}
}

// maxNameRunes resolves DuplicateConfig.MaxNameRunes under the same tri-state
// rule as maxGroupSize: negative means unlimited, zero means the default,
// positive is taken as given. Unlimited is math.MaxInt so the one call site
// stays a plain length comparison.
func (d *DuplicateDetector) maxNameRunes() int {
	switch {
	case d.config.MaxNameRunes < 0:
		return math.MaxInt
	case d.config.MaxNameRunes == 0:
		return DefaultMaxNameRunes
	default:
		return d.config.MaxNameRunes
	}
}

// DuplicateReport carries the result of duplicate detection along with any
// issues describing where the analysis was deliberately incomplete.
//
// The two are returned together because a caller cannot interpret Pairs without
// knowing whether the whole document was actually examined: an empty Pairs from
// a fully swept document means "no duplicates", while an empty Pairs from a
// document whose only large surname group was skipped means "unknown".
type DuplicateReport struct {
	// Pairs contains the potential duplicates found.
	Pairs []DuplicatePair

	// LimitIssues describes any analysis that was skipped or approximated:
	// a surname group exceeding DuplicateConfig.MaxGroupSize, or given names
	// longer than DuplicateConfig.MaxNameRunes compared on a prefix. Both
	// are reported under one CodeDuplicateDetectionLimited issue; read its
	// "skipped_groups" and "truncated_names" details to tell which applied,
	// since either may be zero. Under DuplicateConfig.CompareRelatives the
	// same issue counts oversized spouse-name groups among the skipped
	// groups (named "spouse:<name>" in its "surnames" sample) and carries a
	// "truncated_relatives" detail when some individual's relatives were read
	// only in part. Empty when the entire document was compared in full.
	LimitIssues []Issue
}

// FindDuplicates analyzes all individuals in the document and returns potential duplicates.
// The algorithm groups individuals by normalized surname for efficiency, then compares
// pairs within each surname group.
//
// Contradictions veto a pair whatever its score: exact birth years, or exact
// death years, further apart than DuplicateConfig.MaxBirthYearDiff, or two
// known sexes (M, F or X, compared case-insensitively; empty and U are
// unknown) that differ. The sex veto has no switch. The signals that make up
// the score are listed on [DuplicatePair.Confidence].
//
// Complexity is O(Σ kᵢ²) over the surname groups, which
// DuplicateConfig.MaxGroupSize bounds to O(n · MaxGroupSize / 2) pair
// comparisons. Groups larger than that cap are skipped entirely; use
// FindDuplicatesReport to learn when that happened.
func (d *DuplicateDetector) FindDuplicates(doc *gedcom.Document) []DuplicatePair {
	return d.FindDuplicatesReport(doc).Pairs
}

// FindDuplicatesReport analyzes all individuals in the document and returns
// potential duplicates together with any issues describing skipped analysis.
//
// Complexity is O(Σ kᵢ²) over the surname groups, which
// DuplicateConfig.MaxGroupSize bounds to O(n · MaxGroupSize / 2) pair
// comparisons. Any group larger than the cap is skipped whole and reported in
// DuplicateReport.LimitIssues.
func (d *DuplicateDetector) FindDuplicatesReport(doc *gedcom.Document) DuplicateReport {
	return d.findDuplicatesReport(context.Background(), doc)
}

// findDuplicatesReport is FindDuplicatesReport stopping early, with the pairs
// found so far, once ctx is done; it checks ctx before each surname group.
func (d *DuplicateDetector) findDuplicatesReport(ctx context.Context, doc *gedcom.Document) DuplicateReport {
	if doc == nil {
		return DuplicateReport{}
	}

	individuals := doc.Individuals()
	if len(individuals) < 2 {
		return DuplicateReport{}
	}

	// Build surname groups for efficient comparison
	surnameGroups := d.buildSurnameGroups(individuals)

	var duplicates []DuplicatePair
	limit := d.maxGroupSize()
	nameLimit := d.maxNameRunes()
	var skipped skippedGroups
	builder := d.newCandidateBuilder(doc, nameLimit, &skipped)

	// Compare pairs within each surname group
	for surname, group := range surnameGroups {
		if ctx.Err() != nil {
			break
		}
		if len(group) < 2 {
			continue
		}

		// An oversized group is skipped whole rather than truncated to the
		// first `limit` members. Truncation depends on record order, so the
		// same document in a different order would yield different pairs, and
		// the partial result would be indistinguishable from a complete one.
		if len(group) > limit {
			skipped.record(surname, len(group))
			continue
		}

		// Derived only now that the group is known to be compared: see
		// candidateBuilder on why this is not done during grouping.
		candidates := builder.group(surname, group)

		// Compare all pairs within the group
		for i := 0; i < len(candidates); i++ {
			for j := i + 1; j < len(candidates); j++ {
				if pair, ok := d.comparePair(&candidates[i], &candidates[j]); ok {
					duplicates = append(duplicates, pair)
				}
			}
		}
	}

	if d.config.CompareRelatives {
		duplicates = d.compareSpouseGroups(ctx, individuals, builder, limit, &skipped, duplicates)
	}

	return skipped.report(duplicates, limit, nameLimit)
}

// compareSpouseGroups is the CompareRelatives pass of findDuplicatesReport:
// it compares the pairs that share a spouse's name but not a surname, which
// the surname pass never sees. Each spouse-name group is bounded by limit,
// as a surname group is, and an oversized one is skipped whole. A pair that
// shares more than one spouse's name is compared once, in the group of the
// first name it shares (see ownsPair).
func (d *DuplicateDetector) compareSpouseGroups(ctx context.Context, individuals []*gedcom.Individual, builder *candidateBuilder, limit int, skipped *skippedGroups, duplicates []DuplicatePair) []DuplicatePair {
	for key, group := range builder.spouseGroups(ctx, individuals) {
		if ctx.Err() != nil {
			break
		}
		if len(group) < 2 {
			continue
		}
		if len(group) > limit {
			skipped.record(spouseGroupLabel(key), len(group))
			continue
		}
		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				c1, c2 := builder.candidate(group[i]), builder.candidate(group[j])
				if !ownsPair(key, &c1, &c2) {
					continue
				}
				if pair, ok := d.comparePair(&c1, &c2); ok {
					duplicates = append(duplicates, pair)
				}
			}
		}
	}
	return duplicates
}

// ownsPair reports whether the spouse-name group key is where a pair is
// compared, so that each pair is compared at most once without remembering
// the pairs already seen. The surname pass owns a pair whose surname keys
// are equal and non-empty; the spouse pass skips it, so it is never reported
// twice, and a pair from a surname group skipped for its size is not
// compared behind the notice that reports the skip. Any other pair is owned
// by the first spouse name, in sorted order, the two share.
func ownsPair(key string, c1, c2 *candidate) bool {
	if c1.surname != "" && c1.surname == c2.surname {
		return false
	}
	for i, j := 0, 0; i < len(c1.spouses) && j < len(c2.spouses); {
		switch {
		case c1.spouses[i] == c2.spouses[j]:
			return c1.spouses[i] == key
		case c1.spouses[i] < c2.spouses[j]:
			i++
		default:
			j++
		}
	}
	return false
}

// spouseGroupLabel names a skipped spouse-name group in the
// DUPLICATE_DETECTION_LIMITED "surnames" sample, apart from surname groups.
func spouseGroupLabel(key string) string {
	return "spouse:" + key
}

// FindDuplicatesBetween returns the likely-same-person pairs that take one
// individual from a and one from b, for reconciling two overlapping files
// before they are combined. Pairs within a single document are not reported;
// run FindDuplicates on each document for those. For the same reason, when a
// and b are the same document (or either is nil) the result is empty rather
// than every individual paired with itself, and an individual that appears in
// both documents as the same *gedcom.Individual (as when one document is a
// shallow copy of the other) is never paired with itself.
//
// In every pair Individual1 comes from a and Individual2 from b, each the
// document's own *gedcom.Individual with its own XRef. Records are never
// looked up by XRef across documents, so two files that both use @I1@ for
// different people are not confused; a caller resolving a pair's XRefs must
// look Individual1.XRef up in a and Individual2.XRef up in b. For the same
// reason DuplicatePair.ToIssue, which names both XRefs without their
// documents, is ambiguous for these pairs. As with FindDuplicates, pairs come
// in no particular order.
//
// Scoring, thresholds, vetoes and limits are those of FindDuplicates under the
// same DuplicateConfig, and the pairs are the cross-document pairs that
// FindDuplicates would report on the two documents combined (for example by
// merge.Combine with PrefixDoc2), with their XRefs left unchanged, except
// that an individual shared by both documents is not paired with its own
// copy. In particular a surname group is the union of that surname's
// individuals in both documents, and MaxGroupSize applies to that union: a
// combined group larger than the cap is skipped whole. A shared individual
// still counts once per document toward the cap and the limit counts. Use FindDuplicatesBetweenReport to
// learn when that happened.
//
// Cost: grouping is linear in the individuals of both documents, as it is for
// one run of FindDuplicates on each. Pair comparisons are Σ aᵢ·bᵢ over the
// surnames the documents share, where aᵢ and bᵢ count that surname's
// individuals in a and in b; surnames found in only one document cost nothing
// beyond grouping. Two single-document runs compare aᵢ(aᵢ−1)/2 + bᵢ(bᵢ−1)/2
// pairs for that surname; since aᵢ·bᵢ ≤ (aᵢ² + bᵢ²)/2, the cross-document
// sweep exceeds them by at most (aᵢ+bᵢ)/2 comparisons per shared surname, and
// is far below them when one side is small. It never exceeds the
// (aᵢ+bᵢ)(aᵢ+bᵢ−1)/2 a single run on the combined documents compares. With
// the cap, each shared surname costs at most MaxGroupSize²/4 comparisons.
// Like FindDuplicates, it takes no context: MaxGroupSize is the bound on work.
func (d *DuplicateDetector) FindDuplicatesBetween(a, b *gedcom.Document) []DuplicatePair {
	return d.FindDuplicatesBetweenReport(a, b).Pairs
}

// FindDuplicatesBetweenReport is FindDuplicatesBetween returning the pairs
// together with any issues describing skipped analysis, in the same form as
// FindDuplicatesReport: one CodeDuplicateDetectionLimited issue in
// DuplicateReport.LimitIssues when a combined surname group exceeded
// MaxGroupSize or a compared given name was cut to MaxNameRunes. Only groups
// whose surname occurs in both documents are counted, since no other group can
// yield a cross-document pair.
func (d *DuplicateDetector) FindDuplicatesBetweenReport(a, b *gedcom.Document) DuplicateReport {
	if a == nil || b == nil || a == b {
		return DuplicateReport{}
	}

	groupsA := d.buildSurnameGroups(a.Individuals())
	groupsB := d.buildSurnameGroups(b.Individuals())

	var duplicates []DuplicatePair
	limit := d.maxGroupSize()
	nameLimit := d.maxNameRunes()
	var skipped skippedGroups
	builderA := d.newCandidateBuilder(a, nameLimit, &skipped)
	builderB := d.newCandidateBuilder(b, nameLimit, &skipped)

	for surname, groupA := range groupsA {
		groupB := groupsB[surname]
		if len(groupB) == 0 {
			continue
		}

		// The cap applies to the combined group, as FindDuplicates on the
		// combined documents would see it, and the group is skipped whole for
		// the same reason given there.
		if size := len(groupA) + len(groupB); size > limit {
			skipped.record(surname, size)
			continue
		}

		candidatesA := builderA.group(surname, groupA)
		candidatesB := builderB.group(surname, groupB)

		for i := range candidatesA {
			ca := &candidatesA[i]
			for j := range candidatesB {
				cb := &candidatesB[j]
				// Two distinct documents may still share an *Individual;
				// pairing it with itself would report one record as its own
				// duplicate.
				if ca.ind == cb.ind {
					continue
				}
				if pair, ok := d.comparePair(ca, cb); ok {
					duplicates = append(duplicates, pair)
				}
			}
		}
	}

	if d.config.CompareRelatives {
		duplicates = d.compareSpouseGroupsBetween(context.Background(), builderA, builderB, limit, &skipped, duplicates)
	}

	return skipped.report(duplicates, limit, nameLimit)
}

// compareSpouseGroupsBetween is compareSpouseGroups across two documents: a
// spouse-name group is the union of that name's individuals in both, bounded
// by limit as a combined surname group is, and only pairs taking one
// individual from each are compared, each once (see ownsPair).
func (d *DuplicateDetector) compareSpouseGroupsBetween(ctx context.Context, builderA, builderB *candidateBuilder, limit int, skipped *skippedGroups, duplicates []DuplicatePair) []DuplicatePair {
	groupsA := builderA.spouseGroups(ctx, builderA.doc.Individuals())
	groupsB := builderB.spouseGroups(ctx, builderB.doc.Individuals())
	for key, groupA := range groupsA {
		groupB := groupsB[key]
		if len(groupB) == 0 {
			continue
		}
		if size := len(groupA) + len(groupB); size > limit {
			skipped.record(spouseGroupLabel(key), size)
			continue
		}
		for _, indA := range groupA {
			for _, indB := range groupB {
				if indA == indB {
					continue
				}
				ca, cb := builderA.candidate(indA), builderB.candidate(indB)
				if !ownsPair(key, &ca, &cb) {
					continue
				}
				if pair, ok := d.comparePair(&ca, &cb); ok {
					duplicates = append(duplicates, pair)
				}
			}
		}
	}
	return duplicates
}

// maxSkippedSurnamesReported caps how many surnames the limit Issue names. The
// detail is a sample for diagnosis, not an inventory; an adversary can create
// up to n/MaxGroupSize oversized groups, so the message length must not scale
// with them.
const maxSkippedSurnamesReported = 5

// skippedGroups accumulates the surname groups that exceeded the cap, and the
// individuals whose given names were compared on a prefix (#717), so a single
// aggregate Issue can describe all of them.
type skippedGroups struct {
	groups         int
	individuals    int
	largest        int
	surnames       []string
	truncatedNames int

	// truncatedRelatives counts individuals whose relatives were read only up
	// to maxRelativesRead links (CompareRelatives).
	truncatedRelatives int
}

// record notes one skipped group of the given surname key and size.
//
// The surname is the group key, which is normalized (lowercased, diacritics
// folded) unless DuplicateConfig.DisableNameNormalization is set — so it is not
// necessarily the surname as it appears in the source document. The "surnames"
// detail on the emitted Issue carries the same caveat.
func (s *skippedGroups) record(surname string, size int) {
	s.groups++
	s.individuals += size
	s.largest = max(s.largest, size)
	s.surnames = append(s.surnames, surname)
}

// report wraps pairs in a DuplicateReport, adding the limit Issue when any
// group was skipped or any given name compared on a prefix.
func (s *skippedGroups) report(pairs []DuplicatePair, limit, nameLimit int) DuplicateReport {
	report := DuplicateReport{Pairs: pairs}
	if s.groups > 0 || s.truncatedNames > 0 || s.truncatedRelatives > 0 {
		report.LimitIssues = []Issue{s.toIssue(limit, nameLimit)}
	}
	return report
}

// toIssue renders the accumulated skips and truncations as one Issue.
//
// One aggregate issue is emitted rather than one per group so the output stays
// bounded regardless of how many oversized groups a document contains. The
// severity is Warning, deliberately louder than the Info carried by
// POTENTIAL_DUPLICATE: a caller running at the default strictness has to learn
// that the analysis was incomplete, since a quiet Info would let "no duplicates
// found" be read as "none exist". Prefix comparison is reported on the same
// issue, with the same severity, because a verdict reached on a prefix can
// differ from one on the whole name in either direction.
func (s *skippedGroups) toIssue(limit, nameLimit int) Issue {
	// Map iteration order is randomized, so the sample must be sorted to keep
	// the emitted Issue identical across runs of the same document.
	sort.Strings(s.surnames)
	sample := s.surnames
	if len(sample) > maxSkippedSurnamesReported {
		sample = sample[:maxSkippedSurnamesReported]
	}

	var parts []string
	if s.groups > 0 {
		parts = append(parts, fmt.Sprintf(
			"skipped %d surname group(s) totaling %d individuals "+
				"because they exceed MaxGroupSize=%d", s.groups, s.individuals, limit))
	}
	if s.truncatedNames > 0 {
		parts = append(parts, fmt.Sprintf(
			"compared %d individual(s) on the first %d characters of a given name "+
				"longer than MaxNameRunes", s.truncatedNames, nameLimit))
	}
	if s.truncatedRelatives > 0 {
		parts = append(parts, fmt.Sprintf(
			"read only the first %d spouse and child links of %d individual(s)",
			maxRelativesRead, s.truncatedRelatives))
	}
	message := "Duplicate detection " + strings.Join(parts, " and ") + "; some duplicates may be unreported"

	issue := NewIssue(SeverityWarning, CodeDuplicateDetectionLimited, message, "").
		WithDetail("skipped_groups", strconv.Itoa(s.groups)).
		WithDetail("skipped_individuals", strconv.Itoa(s.individuals)).
		WithDetail("largest_group", strconv.Itoa(s.largest)).
		WithDetail("max_group_size", strconv.Itoa(limit)).
		WithDetail("truncated_names", strconv.Itoa(s.truncatedNames)).
		WithDetail("max_name_runes", strconv.Itoa(nameLimit))
	if s.truncatedRelatives > 0 {
		issue = issue.WithDetail("truncated_relatives", strconv.Itoa(s.truncatedRelatives))
	}
	if s.groups > 0 {
		issue = issue.WithDetail("surnames", strings.Join(sample, ", "))
	}
	return issue
}

// buildSurnameGroups groups individuals by their normalized surname.
//
// Individuals with no surname are omitted rather than collected under an empty
// key. compareSurnames rejects an empty surname outright, so no pair drawn from
// such a bucket could ever match however the detector is configured. Excluding
// them at the point the key is computed keeps that rule in one place, and skips
// the pairwise sweep over them entirely (#529).
//
// Only the surname is normalized here. Given names are normalized later, per
// group, by buildCandidates — see the note there for why that split matters.
func (d *DuplicateDetector) buildSurnameGroups(individuals []*gedcom.Individual) map[string][]*gedcom.Individual {
	groups := make(map[string][]*gedcom.Individual)

	for _, ind := range individuals {
		surname := d.surnameKey(ind)
		if surname == "" {
			continue
		}

		groups[surname] = append(groups[surname], ind)
	}

	return groups
}

// surnameKey returns ind's surname as groups and comparisons key it:
// normalized unless DisableNameNormalization is set, "" when it has none.
func (d *DuplicateDetector) surnameKey(ind *gedcom.Individual) string {
	return d.nameKey(d.extractSurname(ind))
}

// nameKey normalizes a name or place for comparison unless
// DisableNameNormalization is set.
func (d *DuplicateDetector) nameKey(name string) string {
	if d.config.DisableNameNormalization {
		return name
	}
	return normalizeName(name)
}

// sortedDistinct sorts keys and drops repeats, in place.
func sortedDistinct(keys []string) []string {
	sort.Strings(keys)
	return slices.Compact(keys)
}

// countShared returns how many keys two sorted, distinct lists share.
func countShared(a, b []string) int {
	shared := 0
	for i, j := 0, 0; i < len(a) && j < len(b); {
		switch {
		case a[i] == b[j]:
			shared++
			i++
			j++
		case a[i] < b[j]:
			i++
		default:
			j++
		}
	}
	return shared
}

// candidateBuilder derives the comparison keys of one document's individuals,
// pairing each with its normalized given name, birth and death places and,
// under CompareRelatives, its relatives' names.
//
// Candidates are derived once per individual in a group that will actually
// be compared, never per pair: normalizeName runs a Unicode NFD transform
// chain, and doing that inside the O(k²) pair loop dominated the cost of
// detection on large documents (#529).
//
// They are deliberately derived after the MaxGroupSize check rather than
// during grouping. On the adversarial shape the cap exists to defend against
// — every individual sharing one surname — the whole group is skipped, so
// normalizing its given names up front would run a transform chain per
// individual only to discard every result. Deferring keeps that work
// proportional to the set actually compared rather than to the size of the
// input (#530).
//
// A given name longer than maxRunes is cut to its first maxRunes runes before
// any pair is scored, so the quadratic similarity never sees more than
// maxRunes² cells (#717), and the individual is counted in skipped.
//
// Under CompareRelatives an individual can be compared from its surname group
// and from spouse-name groups, so its candidate is remembered and derived,
// and counted, once; so are its spouse keys and every relative's name key.
// Without it no individual is in two groups and nothing is remembered.
type candidateBuilder struct {
	d         *DuplicateDetector
	doc       *gedcom.Document
	maxRunes  int
	skipped   *skippedGroups
	built     map[*gedcom.Individual]candidate
	spouses   map[*gedcom.Individual]spouseKeys
	relatives map[string]string
}

// spouseKeys is one individual's remembered spouse keys and whether its
// spouse links were cut (see candidateBuilder.spouseKeys).
type spouseKeys struct {
	keys []string
	cut  bool
}

func (d *DuplicateDetector) newCandidateBuilder(doc *gedcom.Document, maxRunes int, skipped *skippedGroups) *candidateBuilder {
	b := &candidateBuilder{d: d, doc: doc, maxRunes: maxRunes, skipped: skipped}
	if d.config.CompareRelatives {
		b.built = make(map[*gedcom.Individual]candidate)
		b.spouses = make(map[*gedcom.Individual]spouseKeys)
		b.relatives = make(map[string]string)
	}
	return b
}

// group derives the candidates of one surname group. Every member shares the
// group's surname key, so that value is passed in rather than re-derived.
func (b *candidateBuilder) group(surname string, individuals []*gedcom.Individual) []candidate {
	candidates := make([]candidate, 0, len(individuals))
	for _, ind := range individuals {
		candidates = append(candidates, b.derive(ind, surname))
	}
	return candidates
}

// candidate derives the candidate of one individual, keying its own surname.
func (b *candidateBuilder) candidate(ind *gedcom.Individual) candidate {
	if c, ok := b.built[ind]; ok {
		return c
	}
	return b.derive(ind, b.d.surnameKey(ind))
}

// derive builds a candidate, remembering it under CompareRelatives. The
// surname pass meets each individual once, so only candidate checks first.
func (b *candidateBuilder) derive(ind *gedcom.Individual, surname string) candidate {
	d := b.d
	given := d.nameKey(extractGivenName(ind))
	if cut := cutRunes(given, b.maxRunes); cut != given {
		given = cut
		b.skipped.truncatedNames++
	}
	c := candidate{
		ind:        ind,
		surname:    surname,
		given:      given,
		birthPlace: d.nameKey(ind.BirthPlace()),
		deathPlace: d.nameKey(ind.DeathPlace()),
	}
	if b.built != nil {
		// Spouse links cut here were counted when the spouse groups were
		// built, so a cut counts only when those links were read whole.
		c.spouses, _ = b.spouseKeys(ind)
		var cut bool
		c.children, cut = b.childKeys(ind)
		if cut && len(ind.SpouseInFamilies) <= maxRelativesRead {
			b.skipped.truncatedRelatives++
		}
		b.built[ind] = c
	}
	return c
}

// spouseGroups groups individuals by the name keys of their spouses, for
// the CompareRelatives pass. An individual is listed once under each distinct
// spouse name, in document order. It reads at most maxRelativesRead spouse
// links per individual and counts the individuals cut in skipped; it is
// called once per document, so each is counted once. It stops early, with
// the groups built so far, once ctx is done.
func (b *candidateBuilder) spouseGroups(ctx context.Context, individuals []*gedcom.Individual) map[string][]*gedcom.Individual {
	groups := make(map[string][]*gedcom.Individual)
	for _, ind := range individuals {
		if ctx.Err() != nil {
			break
		}
		spouses, cut := b.spouseKeys(ind)
		if cut {
			b.skipped.truncatedRelatives++
		}
		for _, key := range spouses {
			groups[key] = append(groups[key], ind)
		}
	}
	return groups
}

// spouseKeys returns the sorted, distinct name keys of ind's spouses, read
// from at most maxRelativesRead FAMS links; cut reports that more were left
// unread. The spouse in a family is the partner who is not ind, as
// [gedcom.Individual.Spouses] reads it. The keys are remembered, so the spouse
// groups and ind's candidate read its links once.
func (b *candidateBuilder) spouseKeys(ind *gedcom.Individual) (keys []string, cut bool) {
	if s, ok := b.spouses[ind]; ok {
		return s.keys, s.cut
	}
	links := ind.SpouseInFamilies
	if len(links) > maxRelativesRead {
		links, cut = links[:maxRelativesRead], true
	}
	for _, link := range links {
		fam := b.doc.GetFamily(link.FamilyXRef)
		if fam == nil {
			continue
		}
		partner := fam.Husband
		if partner == ind.XRef {
			partner = fam.Wife
		}
		if key := b.relativeKey(partner); key != "" {
			keys = append(keys, key)
		}
	}
	keys = sortedDistinct(keys)
	b.spouses[ind] = spouseKeys{keys: keys, cut: cut}
	return keys, cut
}

// childKeys returns the sorted, distinct name keys of ind's children, the
// CHIL of its FAMS families, reading at most maxRelativesRead child links;
// cut reports that more were left unread.
func (b *candidateBuilder) childKeys(ind *gedcom.Individual) (keys []string, cut bool) {
	links := ind.SpouseInFamilies
	if len(links) > maxRelativesRead {
		links, cut = links[:maxRelativesRead], true
	}
	read := 0
	for _, link := range links {
		fam := b.doc.GetFamily(link.FamilyXRef)
		if fam == nil {
			continue
		}
		for _, child := range fam.Children {
			if read == maxRelativesRead {
				return sortedDistinct(keys), true
			}
			read++
			if key := b.relativeKey(child); key != "" {
				keys = append(keys, key)
			}
		}
	}
	return sortedDistinct(keys), cut
}

// relativeKey returns the key a relative's name is compared on: given name
// and surname, normalized as an individual's own names are, or "" when xref
// resolves to no individual or the individual has no given name. A bare
// surname is shared by too many relatives to say anything.
//
// Each relative's key is derived once per document and remembered, since
// many individuals can link one family and so name one relative many times;
// normalizing it per link would cost the document's size times the length of
// a name. The given name and the surname are each cut to maxRunes runes, so
// a crafted name cannot make the key, or its comparisons, long.
func (b *candidateBuilder) relativeKey(xref string) string {
	if xref == "" {
		return ""
	}
	if key, ok := b.relatives[xref]; ok {
		return key
	}
	key := ""
	if relative := b.doc.GetIndividual(xref); relative != nil {
		if given := b.d.nameKey(extractGivenName(relative)); given != "" {
			key = cutRunes(given, b.maxRunes) + " /" + cutRunes(b.d.surnameKey(relative), b.maxRunes) + "/"
		}
	}
	b.relatives[xref] = key
	return key
}

// cutRunes returns s cut to its first n runes.
func cutRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// extractSurname extracts the surname from an individual's primary name.
func (d *DuplicateDetector) extractSurname(ind *gedcom.Individual) string {
	name := primaryName(ind)
	if name == nil {
		return ""
	}

	// Use explicit surname if available
	if name.Surname != "" {
		return name.Surname
	}

	// Extract from Full name in GEDCOM format "Given /Surname/"
	return extractSurnameFromFull(name.Full)
}

// extractSurnameFromFull extracts the surname from a GEDCOM full name format.
// GEDCOM names use the format "Given /Surname/" where surname is enclosed in slashes.
func extractSurnameFromFull(fullName string) string {
	start := strings.Index(fullName, "/")
	if start == -1 {
		return ""
	}
	end := strings.LastIndex(fullName, "/")
	if end <= start {
		return ""
	}
	return fullName[start+1 : end]
}

// extractGivenName extracts the given name from an individual's primary name.
func extractGivenName(ind *gedcom.Individual) string {
	name := primaryName(ind)
	if name == nil {
		return ""
	}

	// Use explicit given name if available
	if name.Given != "" {
		return name.Given
	}

	// Extract from Full name in GEDCOM format "Given /Surname/"
	return extractGivenFromFull(name.Full)
}

// extractGivenFromFull extracts the given name from a GEDCOM full name format.
func extractGivenFromFull(fullName string) string {
	idx := strings.Index(fullName, "/")
	if idx == -1 {
		return strings.TrimSpace(fullName)
	}
	return strings.TrimSpace(fullName[:idx])
}

// comparePair compares two candidates and returns a DuplicatePair if they match.
// Both already carry their normalized surname, given name and places from
// buildCandidates; deriving them here instead would repeat the work once per
// pair rather than once per individual.
//
//nolint:gocyclo // Complexity is appropriate for comparison logic
func (d *DuplicateDetector) comparePair(c1, c2 *candidate) (DuplicatePair, bool) {
	var confidence float64
	var reasons []string

	ind1, ind2 := c1.ind, c2.ind

	// Under CompareRelatives, a shared spouse's name stands in for a surname
	// match: it lets a married-name/maiden-name pair through (#932). Only the
	// spouse pass brings such a pair here.
	sameSpouse, sharedChildren := false, 0
	if d.config.CompareRelatives {
		sameSpouse = countShared(c1.spouses, c2.spouses) > 0
		sharedChildren = countShared(c1.children, c2.children)
	}

	// Check surname match
	if compareSurnames(c1.surname, c2.surname) {
		confidence += 0.3
		reasons = append(reasons, "exact surname match")
	} else if !sameSpouse {
		return DuplicatePair{}, false
	}

	givenSimilarity := compareGivenNames(c1.given, c2.given, d.config.MinNameSimilarity)
	if givenSimilarity < d.config.MinNameSimilarity {
		return DuplicatePair{}, false
	}

	// Given name similarity contributes to confidence
	confidence += 0.3 * givenSimilarity
	if givenSimilarity == 1.0 {
		reasons = append(reasons, "exact given name match")
	} else {
		reasons = append(reasons, fmt.Sprintf("similar given name (%.0f%%)", givenSimilarity*100))
	}

	// Compare birth dates
	birthDate1 := ind1.BirthDate()
	birthDate2 := ind2.BirthDate()

	if d.config.RequireBirthDate && (birthDate1 == nil || birthDate2 == nil) {
		return DuplicatePair{}, false
	}

	// Exact birth years further apart than MaxBirthYearDiff contradict a match,
	// so they veto the pair rather than merely withholding credit (#728). An
	// approximate, bounded or range date compares only its first year, which
	// need not contradict the other date, so it withholds credit instead.
	if yearDiff, ok := yearsApart(birthDate1, birthDate2); ok {
		switch {
		case yearDiff == 0:
			confidence += 0.2
			reasons = append(reasons, "same birth year")
		case yearDiff <= d.config.MaxBirthYearDiff:
			confidence += 0.1
			reasons = append(reasons, fmt.Sprintf("birth year within %d years", yearDiff))
		case isExactDate(birthDate1) && isExactDate(birthDate2):
			return DuplicatePair{}, false
		}
	}

	// Compare sex: two known sexes that differ veto the pair (#728). Any
	// other pair earns the weight: a sex unrecorded on one side or both is
	// no evidence of two people, so it must cost no more than a sex both
	// record alike (#932).
	switch sex1, sex2 := knownSex(ind1.Sex), knownSex(ind2.Sex); {
	case sex1 == "" || sex2 == "":
		confidence += 0.1
		reasons = append(reasons, "sex not recorded on both")
	case sex1 != sex2:
		return DuplicatePair{}, false
	default:
		confidence += 0.1
		reasons = append(reasons, "same sex")
	}

	// The remaining signals are added after the four above so that a pair
	// earning none of them keeps the score it always had, bit for bit (#906).
	if sameFullDate(birthDate1, birthDate2) {
		confidence += weightSameBirthDate
		reasons = append(reasons, "same birth date")
	}

	// Exact death years further apart than MaxBirthYearDiff contradict a match
	// as exact birth years do, and veto the pair.
	sameDeathYear, contradicted := d.compareDeathDates(ind1.DeathDate(), ind2.DeathDate())
	if contradicted {
		return DuplicatePair{}, false
	}
	if sameDeathYear {
		confidence += weightSameDeathYear
		reasons = append(reasons, "same death year")
	}

	if c1.birthPlace != "" && c1.birthPlace == c2.birthPlace {
		confidence += weightSameBirthPlace
		reasons = append(reasons, "same birth place")
	}
	if c1.deathPlace != "" && c1.deathPlace == c2.deathPlace {
		confidence += weightSameDeathPlace
		reasons = append(reasons, "same death place")
	}

	if sameSpouse {
		confidence += weightSameSpouse
		reasons = append(reasons, "same spouse name")
	}
	if sharedChildren > 0 {
		confidence += float64(min(sharedChildren, maxSharedChildrenScored)) * weightSharedChild
		if sharedChildren == 1 {
			reasons = append(reasons, "1 shared child name")
		} else {
			reasons = append(reasons, fmt.Sprintf("%d shared child names", sharedChildren))
		}
	}
	// Relatives can push the sum past 1.0; nothing else can.
	confidence = min(confidence, 1.0)

	// Check if confidence meets threshold
	if confidence < d.config.MinConfidence {
		return DuplicatePair{}, false
	}

	return DuplicatePair{
		Individual1:  ind1,
		Individual2:  ind2,
		Confidence:   confidence,
		MatchReasons: reasons,
	}, true
}

// sameFullDate reports whether two dates name the same day: both exact, in the
// same calendar, with a known day and month that agree, and zero years apart.
// Dates in different calendars are never the same full date, since their days
// and months do not correspond.
func sameFullDate(d1, d2 *gedcom.Date) bool {
	years, ok := yearsApart(d1, d2)
	if !ok || years != 0 || !isExactDate(d1) || !isExactDate(d2) {
		return false
	}
	return d1.Calendar == d2.Calendar &&
		d1.Day > 0 && d1.Month > 0 && d1.Day == d2.Day && d1.Month == d2.Month
}

// compareDeathDates compares two death dates. Only two exact dates with real
// years say anything: sameYear when they are zero years apart, contradicted
// when they are further apart than MaxBirthYearDiff. An approximate, bounded,
// range or missing death date reports neither.
func (d *DuplicateDetector) compareDeathDates(d1, d2 *gedcom.Date) (sameYear, contradicted bool) {
	years, ok := yearsApart(d1, d2)
	if !ok || !isExactDate(d1) || !isExactDate(d2) {
		return false, false
	}
	return years == 0, years > d.config.MaxBirthYearDiff
}

// isExactDate reports whether d names a single date: no modifier (ABT, BEF,
// BET, FROM and the rest) and no range end.
func isExactDate(d *gedcom.Date) bool {
	return d.Modifier == gedcom.ModifierNone && d.EndDate == nil
}

// knownSex returns sex upper-cased and trimmed, or "" when it records no
// known sex: empty or U (unknown). M, F and the GEDCOM 7.0 X are known.
func knownSex(sex string) string {
	sex = strings.ToUpper(strings.TrimSpace(sex))
	if sex == "U" {
		return ""
	}
	return sex
}

// normalizeName normalizes a name for comparison.
// Converts to lowercase, removes diacritics, and trims whitespace.
//
// All-ASCII names skip the diacritic transform: ASCII is invariant under NFD
// and NFC and holds no combining marks, so the chain would return it unchanged
// while dominating the allocation of a corpus-scale sweep (#871).
func normalizeName(name string) string {
	if name == "" {
		return ""
	}

	// Trim whitespace
	name = strings.TrimSpace(name)

	// Convert to lowercase
	name = strings.ToLower(name)

	ascii := true
	for i := 0; i < len(name); i++ {
		if name[i] >= utf8.RuneSelf {
			ascii = false
			break
		}
	}
	if ascii {
		return name
	}

	// Remove diacritics using unicode normalization
	// NFD decomposes characters, then we remove combining marks
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, err := transform.String(t, name)
	if err != nil {
		// If transform fails, return lowercase version
		return name
	}

	return result
}

// compareSurnames reports whether two surname keys match: both non-empty and
// equal. Keys are already normalized unless DisableNameNormalization is set.
// Candidates are only ever compared within one surname group, so equality is
// the grouping rule restated; there is no fuzzy surname mode.
func compareSurnames(s1, s2 string) bool {
	if s1 == "" || s2 == "" {
		// Can't match if either surname is missing
		return false
	}
	return s1 == s2
}

// compareGivenNames compares two given names and returns a similarity score.
// Returns 0.0 if either name is empty.
// Returns 1.0 for exact match, otherwise returns string similarity.
//
// minSimilarity is the caller's acceptance threshold and enables an early exit:
// when a length-based upper bound on the achievable similarity already falls
// below it, the O(n·m) Levenshtein DP is skipped. In exactly that case the
// returned value is that upper bound rather than the exact similarity — it is
// guaranteed to be below minSimilarity, which is all the caller's comparison
// needs. Pass minSimilarity <= 0 to always compute the exact value.
func compareGivenNames(g1, g2 string, minSimilarity float64) float64 {
	if g1 == "" || g2 == "" {
		return 0.0
	}

	if g1 == g2 {
		return 1.0
	}

	// Early exit. Levenshtein distance is at least the difference in rune
	// length, so 1 - |runeLen1-runeLen2|/maxRuneLen is an upper bound on what
	// stringSimilarity can return. Names that cannot possibly clear the
	// threshold skip the DP. Both sides count runes, matching stringSimilarity's
	// denominator; the bound is only sound while the two use the same unit.
	if minSimilarity > 0 {
		n1, n2 := utf8.RuneCountInString(g1), utf8.RuneCountInString(g2)
		upperBound := 1.0 - float64(absInt(n1-n2))/float64(max(n1, n2))
		if upperBound < minSimilarity {
			return upperBound
		}
	}

	return stringSimilarity(g1, g2)
}

// stringSimilarity calculates the similarity between two strings.
// Uses Levenshtein distance normalized to a 0.0-1.0 scale.
// Returns 1.0 for identical strings, 0.0 for completely different strings.
func stringSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	if s1 == "" || s2 == "" {
		return 0.0
	}

	distance := levenshteinDistance(s1, s2)
	// Measured in runes, the same unit as levenshteinDistance. A byte length
	// would inflate the score for multibyte names (#545).
	maxLen := max(utf8.RuneCountInString(s1), utf8.RuneCountInString(s2))

	// Similarity is 1 - (distance / maxLen)
	return 1.0 - float64(distance)/float64(maxLen)
}

// levenshteinDistance calculates the Levenshtein distance between two strings.
// This is the minimum number of single-character edits (insertions, deletions,
// or substitutions) needed to transform s1 into s2.
func levenshteinDistance(s1, s2 string) int {
	// Convert to rune slices for proper Unicode handling
	r1 := []rune(s1)
	r2 := []rune(s2)

	len1 := len(r1)
	len2 := len(r2)

	// Handle empty strings
	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	// Create distance matrix
	// We only need two rows since we process row by row
	prev := make([]int, len2+1)
	curr := make([]int, len2+1)

	// Initialize first row
	for j := 0; j <= len2; j++ {
		prev[j] = j
	}

	// Fill in the rest of the matrix
	for i := 1; i <= len1; i++ {
		curr[0] = i

		for j := 1; j <= len2; j++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}

			// Minimum of insert, delete, or substitute
			curr[j] = min(
				prev[j]+1,      // deletion
				curr[j-1]+1,    // insertion
				prev[j-1]+cost, // substitution
			)
		}

		// Swap rows
		prev, curr = curr, prev
	}

	return prev[len2]
}

// yearsApart returns how many years apart two dates fall. Years are
// compared astronomically, so 44 BCE and 44 CE are 87 years apart. Two
// dates in the same calendar compare Year as written, so a dual date counts
// from its Old Style year. Dates in different calendars are each placed on
// their first day and compared as Gregorian years (see gregorianStartYear), so
// Hebrew 5660 and Gregorian 1899 are the same year while Hebrew 1900 and
// Gregorian 1900 are not. Like gedcom.YearsBetween it converts only when the
// calendars differ, but it counts calendar years rather than elapsed years,
// and unlike YearsBetween it reads a same-calendar dual date in Old Style, so
// Julian 11 FEB 1731/32 is one year from Julian 1732 but the same year as
// Gregorian 1732. ok is false, and the dates earn no credit, when either is
// missing, has no real year (a phrase, "0", "0 BC", a negative year) or its
// year cannot be placed.
func yearsApart(d1, d2 *gedcom.Date) (years int, ok bool) {
	if d1 == nil || d2 == nil || !hasRealYear(d1) || !hasRealYear(d2) {
		return 0, false
	}
	if d1.Calendar == d2.Calendar {
		return absInt(gedcom.AstronomicalYear(d1.Year, d1.IsBC) - gedcom.AstronomicalYear(d2.Year, d2.IsBC)), true
	}
	y1, ok1 := gregorianStartYear(d1)
	y2, ok2 := gregorianStartYear(d2)
	if !ok1 || !ok2 {
		return 0, false
	}
	return absInt(y1 - y2), true
}

// gregorianStartYear returns the astronomical Gregorian year of d's first day,
// ignoring its modifier and any range end: a year-only or month-only date is
// placed on the first day of its year or month, and a dual date is read in New
// Style, as gedcom.Date.Bounds reads it. A day or month the calendar does not
// have, such as 31 FEB, is dropped, so the date is placed on the first day of
// its month or year instead. ok is false when Bounds cannot place even the
// year.
func gregorianStartYear(d *gedcom.Date) (year int, ok bool) {
	start := *d
	start.Modifier, start.EndDate = gedcom.ModifierNone, nil
	earliest, _, ok := start.Bounds()
	if !ok {
		start.Day = 0
		earliest, _, ok = start.Bounds()
	}
	if !ok {
		start.Month = 0
		earliest, _, ok = start.Bounds()
	}
	if !ok {
		return 0, false
	}
	year, _, _ = gedcom.JDNToGregorian(earliest)
	return year, true
}

// hasRealYear reports whether d carries a year of 1 or more, the domain
// gedcom.AstronomicalYear assumes. Phrase-only dates parse with Year 0.
func hasRealYear(d *gedcom.Date) bool {
	return !d.IsPhrase && d.Year >= 1
}

// absInt returns the absolute value of an integer.
func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
