// duplicates.go provides duplicate individual detection based on name similarity
// and birth date proximity.
//
// Duplicate detection is a common need in genealogy software where the same person
// may be entered multiple times from different sources. This package provides
// configurable matching thresholds for name comparison and date proximity.

package validator

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/cacack/gedcom-go/gedcom"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

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

// getDisplayName returns a display name for an individual.
func getDisplayName(ind *gedcom.Individual) string {
	if ind == nil {
		return ""
	}
	if len(ind.Names) == 0 {
		return ind.XRef
	}
	name := ind.Names[0]
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

// FindDuplicates analyzes all individuals in the document and returns potential duplicates.
// The algorithm groups individuals by normalized surname for efficiency, then compares
// pairs within each surname group.
func (d *DuplicateDetector) FindDuplicates(doc *gedcom.Document) []DuplicatePair {
	if doc == nil {
		return nil
	}

	individuals := doc.Individuals()
	if len(individuals) < 2 {
		return nil
	}

	// Build surname groups for efficient comparison
	surnameGroups := d.buildSurnameGroups(individuals)

	var duplicates []DuplicatePair

	// Compare pairs within each surname group
	for _, group := range surnameGroups {
		if len(group) < 2 {
			continue
		}

		// Compare all pairs within the group
		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				if pair, ok := d.comparePair(group[i], group[j]); ok {
					duplicates = append(duplicates, pair)
				}
			}
		}
	}

	return duplicates
}

// buildSurnameGroups groups individuals by their normalized surname.
// Individuals without surnames are grouped under an empty string key.
func (d *DuplicateDetector) buildSurnameGroups(individuals []*gedcom.Individual) map[string][]*gedcom.Individual {
	groups := make(map[string][]*gedcom.Individual)

	for _, ind := range individuals {
		surname := d.extractSurname(ind)
		if d.config.NormalizeNames {
			surname = normalizeName(surname)
		}
		groups[surname] = append(groups[surname], ind)
	}

	return groups
}

// extractSurname extracts the surname from an individual's primary name.
func (d *DuplicateDetector) extractSurname(ind *gedcom.Individual) string {
	if ind == nil || len(ind.Names) == 0 {
		return ""
	}

	name := ind.Names[0]

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
	if ind == nil || len(ind.Names) == 0 {
		return ""
	}

	name := ind.Names[0]

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

// comparePair compares two individuals and returns a DuplicatePair if they match.
//
//nolint:gocyclo // Complexity is appropriate for comparison logic
func (d *DuplicateDetector) comparePair(ind1, ind2 *gedcom.Individual) (DuplicatePair, bool) {
	var confidence float64
	var reasons []string

	// Get surnames
	surname1 := d.extractSurname(ind1)
	surname2 := d.extractSurname(ind2)
	if d.config.NormalizeNames {
		surname1 = normalizeName(surname1)
		surname2 = normalizeName(surname2)
	}

	// Check surname match
	surnameMatch := compareSurnames(surname1, surname2, d.config.RequireExactSurname)
	if !surnameMatch {
		return DuplicatePair{}, false
	}

	// Surname match contributes to confidence
	confidence += 0.3
	reasons = append(reasons, "exact surname match")

	// Get and compare given names
	given1 := extractGivenName(ind1)
	given2 := extractGivenName(ind2)
	if d.config.NormalizeNames {
		given1 = normalizeName(given1)
		given2 = normalizeName(given2)
	}

	givenSimilarity := compareGivenNames(given1, given2, d.config.MinNameSimilarity)
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
// The minSimilarity parameter is reserved for future use (e.g., early exit optimization).
func compareGivenNames(g1, g2 string, _ float64) float64 {
	if g1 == "" || g2 == "" {
		return 0.0
	}

	if g1 == g2 {
		return 1.0
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
	maxLen := maxOfTwo(len(s1), len(s2))

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
			curr[j] = minOfThree(
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

// minOfThree returns the minimum of three integers.
func minOfThree(a, b, c int) int {
	if a <= b && a <= c {
		return a
	}
	if b <= c {
		return b
	}
	return c
}

// maxOfTwo returns the maximum of two integers.
func maxOfTwo(a, b int) int {
	if a > b {
		return a
	}
	return b
}
