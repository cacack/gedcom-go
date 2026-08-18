package gedcomgo

// FEATURES.md's Multi-Version Support section states measured numbers rather
// than adjectives: how many GEDCOM 7.0 structures reach the typed model, and
// how many corpus fixtures survive a round-trip. Numbers in prose rot silently
// -- they are the one thing a reader trusts and the one thing nothing checks.
//
// This checks them against the two places they are actually derived: the
// generated coverage report, and the self-cleaning lists in
// byte_roundtrip_test.go. Fixing any of the issues those lists track will fail
// this test until the prose is updated, which is the point.

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// coverageReportPath is the generated GEDCOM 7.0 inventory, which is the
// authority for every 7.0 coverage number stated anywhere else.
const coverageReportPath = "docs/reference/gedcom-7-coverage.md"

// featuresClaim is one number FEATURES.md states, with the pattern that finds
// it and the value it has to equal.
type featuresClaim struct {
	what    string
	pattern *regexp.Regexp
	want    int
}

func TestFeaturesNumbersMatchTheirSources(t *testing.T) {
	features := readFile(t, "FEATURES.md")
	typed, total := coverageReportTotals(t)

	fixtures := len(gedcomFixtures(t))
	decodable := fixtures - len(undecodable)

	claims := []featuresClaim{
		{"typed 7.0 structures", regexp.MustCompile(`\[(\d[\d,]*) of [\d,]+ structures\]`), typed},
		{"total 7.0 structures", regexp.MustCompile(`\[[\d,]+ of (\d[\d,]*) structures\]`), total},
		{"typed 7.0 structures, restated", regexp.MustCompile(`(\d[\d,]*) reach the typed model`), typed},
		{"corpus fixtures", regexp.MustCompile(`Of (\d[\d,]*) corpus\s+fixtures`), fixtures},
		{"undecodable fixtures", regexp.MustCompile(`fixtures, (\d[\d,]*) do not survive`), len(undecodable)},
		{"decodable fixtures", regexp.MustCompile(`of the (\d[\d,]*) that do`), decodable},
		{"headers that round-trip", regexp.MustCompile(`(\d[\d,]*)\s+reproduce their header`), len(headerByteIdentical)},
		{"bodies that round-trip", regexp.MustCompile(`(\d[\d,]*) reproduce their record body`), decodable - len(bodyKnownBad)},
	}

	for _, claim := range claims {
		match := claim.pattern.FindStringSubmatch(features)
		if match == nil {
			t.Errorf("FEATURES.md no longer states the %s (looked for %s); "+
				"if the wording changed, update this test with it", claim.what, claim.pattern)
			continue
		}
		got, err := strconv.Atoi(strings.ReplaceAll(match[1], ",", ""))
		if err != nil {
			t.Errorf("%s: %q is not a number: %v", claim.what, match[1], err)
			continue
		}
		if got != claim.want {
			t.Errorf("FEATURES.md says %d %s; the source says %d. Update FEATURES.md.",
				got, claim.what, claim.want)
		}
	}

	// The percentage is derived from the first two, so it cannot be checked
	// against a source -- only against arithmetic.
	wantShare := fmt.Sprintf("(%.1f%%)", 100*float64(typed)/float64(total))
	if !strings.Contains(features, wantShare) {
		t.Errorf("FEATURES.md does not state the typed share as %s (%d of %d)",
			wantShare, typed, total)
	}
}

// coverageReportTotals reads the typed count and the total from the generated
// report's summary table, so FEATURES.md and the report cannot disagree.
func coverageReportTotals(t *testing.T) (typed, total int) {
	t.Helper()

	report := readFile(t, coverageReportPath)

	// Rows are "| status | count | share | meaning |"; the total is their sum.
	rows := regexp.MustCompile(`(?m)^\| (typed|partial|raw \([a-z]+\)) \| (\d+) \|`).FindAllStringSubmatch(report, -1)
	if len(rows) == 0 {
		t.Fatalf("%s has no summary table; regenerate it with `make spec-coverage`", coverageReportPath)
	}
	for _, row := range rows {
		n, err := strconv.Atoi(row[2])
		if err != nil {
			t.Fatalf("%s: %q is not a count: %v", coverageReportPath, row[2], err)
		}
		total += n
		if row[1] == "typed" {
			typed = n
		}
	}
	return typed, total
}

// readFile reads a repository file, failing the test if it cannot.
func readFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
