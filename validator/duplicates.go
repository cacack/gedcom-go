// duplicates.go provides duplicate individual detection based on name similarity
// and birth date proximity.
//
// Duplicate detection is a common need in genealogy software where the same person
// may be entered multiple times from different sources. This package provides
// configurable matching thresholds for name comparison and date proximity.

package validator

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/cacack/gedcom-go/v2/gedcom"
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

// DuplicateConfig contains configuration options for duplicate detection.
type DuplicateConfig struct {
	// RequireExactSurname requires surnames to match exactly (after normalization).
	// Default: true
	RequireExactSurname bool

	// NormalizeNames enables name normalization (trim, lowercase, remove accents).
	// Default: true
	NormalizeNames bool

	// MinNameSimilarity is the minimum similarity threshold for given name comparison.
	// Range: 0.0 to 1.0, where 1.0 is exact match.
	// Default: 0.8
	MinNameSimilarity float64

	// MaxBirthYearDiff is the maximum allowed difference in birth years.
	// Default: 2
	MaxBirthYearDiff int

	// RequireBirthDate requires both individuals to have birth dates for a match.
	// If true, missing birth dates prevent a match.
	// Default: false
	RequireBirthDate bool

	// MinConfidence is the minimum overall confidence score for a match.
	// Range: 0.0 to 1.0
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
}

// DefaultDuplicateConfig returns a DuplicateConfig with default values.
func DefaultDuplicateConfig() DuplicateConfig {
	return DuplicateConfig{
		RequireExactSurname: true,
		NormalizeNames:      true,
		MinNameSimilarity:   0.8,
		MaxBirthYearDiff:    2,
		RequireBirthDate:    false,
		MinConfidence:       0.7,
		MaxGroupSize:        DefaultMaxGroupSize,
	}
}

// DuplicatePair represents a potential duplicate pair of individuals.
type DuplicatePair struct {
	// Individual1 is the first individual in the pair.
	Individual1 *gedcom.Individual

	// Individual2 is the second individual in the pair.
	Individual2 *gedcom.Individual

	// Confidence is the overall match confidence score (0.0 to 1.0).
	Confidence float64

	// MatchReasons contains human-readable descriptions of why these individuals matched.
	MatchReasons []string
}

// ToIssue converts the DuplicatePair to a validation Issue.
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
func NewDuplicateDetector(config *DuplicateConfig) *DuplicateDetector {
	if config == nil {
		defaultConfig := DefaultDuplicateConfig()
		config = &defaultConfig
	}
	return &DuplicateDetector{config: *config}
}

// candidate carries an individual alongside the normalized names used to
// compare it. The keys are derived once per individual during grouping rather
// than re-derived for every pair: normalizeName runs a Unicode NFD transform
// chain, and doing that inside the O(n^2) pair loop dominated the cost of
// FindDuplicates on large documents (#529).
type candidate struct {
	ind     *gedcom.Individual
	surname string
	given   string
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

	// LimitIssues describes any analysis that was skipped, for example a
	// surname group exceeding DuplicateConfig.MaxGroupSize. Empty when the
	// entire document was compared.
	LimitIssues []Issue
}

// FindDuplicates analyzes all individuals in the document and returns potential duplicates.
// The algorithm groups individuals by normalized surname for efficiency, then compares
// pairs within each surname group.
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
	var skipped skippedGroups

	// Compare pairs within each surname group
	for surname, group := range surnameGroups {
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
		// buildCandidates on why this is not done during grouping.
		candidates := d.buildCandidates(surname, group)

		// Compare all pairs within the group
		for i := 0; i < len(candidates); i++ {
			for j := i + 1; j < len(candidates); j++ {
				if pair, ok := d.comparePair(candidates[i], candidates[j]); ok {
					duplicates = append(duplicates, pair)
				}
			}
		}
	}

	report := DuplicateReport{Pairs: duplicates}
	if skipped.groups > 0 {
		report.LimitIssues = []Issue{skipped.toIssue(limit)}
	}
	return report
}

// maxSkippedSurnamesReported caps how many surnames the limit Issue names. The
// detail is a sample for diagnosis, not an inventory; an adversary can create
// up to n/MaxGroupSize oversized groups, so the message length must not scale
// with them.
const maxSkippedSurnamesReported = 5

// skippedGroups accumulates the surname groups that exceeded the cap so a
// single aggregate Issue can describe all of them.
type skippedGroups struct {
	groups      int
	individuals int
	largest     int
	surnames    []string
}

// record notes one skipped group of the given surname key and size.
//
// The surname is the group key, which is normalized (lowercased, diacritics
// folded) only when DuplicateConfig.NormalizeNames is set — so it is not
// necessarily the surname as it appears in the source document. The "surnames"
// detail on the emitted Issue carries the same caveat.
func (s *skippedGroups) record(surname string, size int) {
	s.groups++
	s.individuals += size
	s.largest = max(s.largest, size)
	s.surnames = append(s.surnames, surname)
}

// toIssue renders the accumulated skips as one Issue.
//
// One aggregate issue is emitted rather than one per group so the output stays
// bounded regardless of how many oversized groups a document contains. The
// severity is Warning, deliberately louder than the Info carried by
// POTENTIAL_DUPLICATE: a caller running at the default strictness has to learn
// that the analysis was incomplete, since a quiet Info would let "no duplicates
// found" be read as "none exist".
func (s *skippedGroups) toIssue(limit int) Issue {
	// Map iteration order is randomized, so the sample must be sorted to keep
	// the emitted Issue identical across runs of the same document.
	sort.Strings(s.surnames)
	sample := s.surnames
	if len(sample) > maxSkippedSurnamesReported {
		sample = sample[:maxSkippedSurnamesReported]
	}

	message := fmt.Sprintf(
		"Duplicate detection skipped %d surname group(s) totaling %d individuals "+
			"because they exceed MaxGroupSize=%d; some duplicates may be unreported",
		s.groups, s.individuals, limit)

	return NewIssue(SeverityWarning, CodeDuplicateDetectionLimited, message, "").
		WithDetail("skipped_groups", strconv.Itoa(s.groups)).
		WithDetail("skipped_individuals", strconv.Itoa(s.individuals)).
		WithDetail("largest_group", strconv.Itoa(s.largest)).
		WithDetail("max_group_size", strconv.Itoa(limit)).
		WithDetail("surnames", strings.Join(sample, ", "))
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
		surname := d.extractSurname(ind)
		if d.config.NormalizeNames {
			surname = normalizeName(surname)
		}
		if surname == "" {
			continue
		}

		groups[surname] = append(groups[surname], ind)
	}

	return groups
}

// buildCandidates derives the comparison keys for one surname group, pairing
// each individual with its normalized given name. Every member of a group
// shares the group's surname key, so that value is passed in rather than
// re-derived per individual.
//
// This runs once per individual in a group that will actually be compared,
// never per pair: normalizeName runs a Unicode NFD transform chain, and doing
// that inside the O(k²) pair loop dominated the cost of detection on large
// documents (#529).
//
// It is deliberately called after the MaxGroupSize check rather than during
// grouping. On the adversarial shape the cap exists to defend against — every
// individual sharing one surname — the whole group is skipped, so normalizing
// its given names up front would run a transform chain per individual only to
// discard every result. Deferring keeps that work proportional to the set
// actually compared rather than to the size of the input (#530).
func (d *DuplicateDetector) buildCandidates(surname string, individuals []*gedcom.Individual) []candidate {
	candidates := make([]candidate, 0, len(individuals))

	for _, ind := range individuals {
		given := extractGivenName(ind)
		if d.config.NormalizeNames {
			given = normalizeName(given)
		}
		candidates = append(candidates, candidate{ind: ind, surname: surname, given: given})
	}

	return candidates
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
// Both already carry their normalized surname and given name from
// buildSurnameGroups; deriving them here instead would repeat the work once per
// pair rather than once per individual.
//
//nolint:gocyclo // Complexity is appropriate for comparison logic
func (d *DuplicateDetector) comparePair(c1, c2 candidate) (DuplicatePair, bool) {
	var confidence float64
	var reasons []string

	ind1, ind2 := c1.ind, c2.ind

	// Check surname match
	surnameMatch := compareSurnames(c1.surname, c2.surname, d.config.RequireExactSurname)
	if !surnameMatch {
		return DuplicatePair{}, false
	}

	// Surname match contributes to confidence
	confidence += 0.3
	reasons = append(reasons, "exact surname match")

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

	if birthDate1 != nil && birthDate2 != nil {
		yearDiff := absInt(birthDate1.Year - birthDate2.Year)
		if yearDiff == 0 {
			confidence += 0.2
			reasons = append(reasons, "same birth year")
		} else if yearDiff <= d.config.MaxBirthYearDiff {
			confidence += 0.1
			reasons = append(reasons, fmt.Sprintf("birth year within %d years", yearDiff))
		}
	}

	// Compare sex
	if ind1.Sex != "" && ind2.Sex != "" && ind1.Sex == ind2.Sex {
		confidence += 0.1
		reasons = append(reasons, "same sex")
	}

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

// normalizeName normalizes a name for comparison.
// Converts to lowercase, removes diacritics, and trims whitespace.
func normalizeName(name string) string {
	if name == "" {
		return ""
	}

	// Trim whitespace
	name = strings.TrimSpace(name)

	// Convert to lowercase
	name = strings.ToLower(name)

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

// compareSurnames compares two surnames.
// If exact is true, returns true only for exact match.
// If exact is false, returns true for any non-empty comparison.
func compareSurnames(s1, s2 string, exact bool) bool {
	if s1 == "" || s2 == "" {
		// Can't match if either surname is missing
		return false
	}
	if exact {
		return s1 == s2
	}
	// Non-exact mode: just require non-empty surnames that were grouped together
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

	// Early exit. Levenshtein distance is at least the difference in length, so
	// 1 - |runeLen1-runeLen2|/maxLen is an upper bound on what stringSimilarity
	// can return. Names that cannot possibly clear the threshold skip the DP.
	//
	// The mixed units are deliberate, not an oversight: the numerator counts
	// RUNES to match levenshteinDistance's rune-based DP, while the denominator
	// is stringSimilarity's own BYTE-based maxLen. Using the same denominator as
	// the function being bounded is what makes the bound sound — the true value
	// is 1 - runeDistance/byteMaxLen, and runeDistance >= |runeLen1-runeLen2|,
	// so this expression is >= the true similarity for every input, multibyte
	// names included. Normalizing both sides to runes (or both to bytes) would
	// change the comparison and break that guarantee.
	if minSimilarity > 0 {
		maxLen := max(len(g1), len(g2))
		upperBound := 1.0 - float64(absInt(utf8.RuneCountInString(g1)-utf8.RuneCountInString(g2)))/float64(maxLen)
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
	maxLen := max(len(s1), len(s2))

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

// absInt returns the absolute value of an integer.
func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
