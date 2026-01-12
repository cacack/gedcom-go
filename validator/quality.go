// quality.go provides comprehensive data quality reporting for GEDCOM documents.
//
// The QualityAnalyzer aggregates results from all validators (date logic, references,
// duplicates) and calculates data completeness metrics to provide a comprehensive
// quality overview of a GEDCOM file.

package validator

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/cacack/gedcom-go/gedcom"
)

// QualityReport contains aggregated validation results and data completeness statistics.
type QualityReport struct {
	// Summary counts
	TotalIndividuals int `json:"total_individuals"`
	TotalFamilies    int `json:"total_families"`
	TotalSources     int `json:"total_sources"`

	// Data completeness counts
	IndividualsWithBirthDate int `json:"individuals_with_birth_date"`
	IndividualsWithDeathDate int `json:"individuals_with_death_date"`
	IndividualsWithSources   int `json:"individuals_with_sources"`
	IndividualsWithPlaces    int `json:"individuals_with_places"`

	// Percentages (calculated) - 0.0 to 1.0
	BirthDateCoverage float64 `json:"birth_date_coverage"`
	DeathDateCoverage float64 `json:"death_date_coverage"`
	SourceCoverage    float64 `json:"source_coverage"`

	// Issues by severity
	Errors   []Issue `json:"errors"`
	Warnings []Issue `json:"warnings"`
	Info     []Issue `json:"info"`

	// Issues by category
	DateLogicIssues    []Issue `json:"date_logic_issues"`
	ReferenceIssues    []Issue `json:"reference_issues"`
	DuplicateIssues    []Issue `json:"duplicate_issues"`
	CompletenessIssues []Issue `json:"completeness_issues"`

	// Summary counts
	TotalIssues  int `json:"total_issues"`
	ErrorCount   int `json:"error_count"`
	WarningCount int `json:"warning_count"`
	InfoCount    int `json:"info_count"`
}

// String returns a human-readable summary of the quality report.
func (r *QualityReport) String() string {
	var sb strings.Builder

	sb.WriteString("GEDCOM Quality Report\n")
	sb.WriteString("=====================\n")
	sb.WriteString(fmt.Sprintf("Records: %d individuals, %d families, %d sources\n",
		r.TotalIndividuals, r.TotalFamilies, r.TotalSources))

	sb.WriteString("\nData Completeness:\n")
	sb.WriteString(fmt.Sprintf("- Birth dates: %.0f%% (%d/%d)\n",
		r.BirthDateCoverage*100, r.IndividualsWithBirthDate, r.TotalIndividuals))
	sb.WriteString(fmt.Sprintf("- Sources: %.0f%% (%d/%d)\n",
		r.SourceCoverage*100, r.IndividualsWithSources, r.TotalIndividuals))

	sb.WriteString(fmt.Sprintf("\nIssues Found: %d total\n", r.TotalIssues))
	sb.WriteString(fmt.Sprintf("- Errors: %d\n", r.ErrorCount))
	sb.WriteString(fmt.Sprintf("- Warnings: %d\n", r.WarningCount))
	sb.WriteString(fmt.Sprintf("- Info: %d\n", r.InfoCount))

	// Top issues by code
	if r.TotalIssues > 0 {
		sb.WriteString("\nTop Issues:\n")
		issueCounts := r.countIssuesByCode()
		// Sort by count descending
		type codeCount struct {
			code  string
			count int
		}
		var counts []codeCount
		for code, count := range issueCounts {
			counts = append(counts, codeCount{code, count})
		}
		sort.Slice(counts, func(i, j int) bool {
			return counts[i].count > counts[j].count
		})
		// Show top 5
		for i, cc := range counts {
			if i >= 5 {
				break
			}
			sb.WriteString(fmt.Sprintf("- %s: %d\n", cc.code, cc.count))
		}
	}

	return sb.String()
}

// countIssuesByCode returns a map of issue codes to their counts.
func (r *QualityReport) countIssuesByCode() map[string]int {
	counts := make(map[string]int)
	allIssues := append(append(append([]Issue{}, r.Errors...), r.Warnings...), r.Info...)
	for _, issue := range allIssues {
		counts[issue.Code]++
	}
	return counts
}

// JSON returns the report in JSON format.
func (r *QualityReport) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// IssuesForRecord returns all issues affecting a specific record.
func (r *QualityReport) IssuesForRecord(xref string) []Issue {
	var result []Issue
	allIssues := append(append(append([]Issue{}, r.Errors...), r.Warnings...), r.Info...)
	for _, issue := range allIssues {
		if issue.RecordXRef == xref || issue.RelatedXRef == xref {
			result = append(result, issue)
		}
	}
	return result
}

// IssuesByCode returns all issues with a specific error code.
func (r *QualityReport) IssuesByCode(code string) []Issue {
	var result []Issue
	allIssues := append(append(append([]Issue{}, r.Errors...), r.Warnings...), r.Info...)
	for _, issue := range allIssues {
		if issue.Code == code {
			result = append(result, issue)
		}
	}
	return result
}

// QualityAnalyzer aggregates validation results from multiple validators.
type QualityAnalyzer struct {
	dateLogic  *DateLogicValidator
	references *ReferenceValidator
	duplicates *DuplicateDetector
}

// QualityOption is a functional option for configuring QualityAnalyzer.
type QualityOption func(*QualityAnalyzer)

// WithDateLogicConfig returns a QualityOption that sets the date logic configuration.
func WithDateLogicConfig(config *DateLogicConfig) QualityOption {
	return func(a *QualityAnalyzer) {
		a.dateLogic = NewDateLogicValidator(config)
	}
}

// WithDuplicateConfig returns a QualityOption that sets the duplicate detection configuration.
func WithDuplicateConfig(config *DuplicateConfig) QualityOption {
	return func(a *QualityAnalyzer) {
		a.duplicates = NewDuplicateDetector(config)
	}
}

// NewQualityAnalyzer creates a new QualityAnalyzer with the given options.
// By default, it creates validators with their default configurations.
func NewQualityAnalyzer(opts ...QualityOption) *QualityAnalyzer {
	a := &QualityAnalyzer{
		dateLogic:  NewDateLogicValidator(nil),
		references: NewReferenceValidator(),
		duplicates: NewDuplicateDetector(nil),
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

// Analyze runs all validators and calculates completeness metrics for the document.
func (a *QualityAnalyzer) Analyze(doc *gedcom.Document) *QualityReport {
	report := &QualityReport{
		Errors:             []Issue{},
		Warnings:           []Issue{},
		Info:               []Issue{},
		DateLogicIssues:    []Issue{},
		ReferenceIssues:    []Issue{},
		DuplicateIssues:    []Issue{},
		CompletenessIssues: []Issue{},
	}

	if doc == nil {
		return report
	}

	// Calculate summary counts
	individuals := doc.Individuals()
	families := doc.Families()
	sources := doc.Sources()

	report.TotalIndividuals = len(individuals)
	report.TotalFamilies = len(families)
	report.TotalSources = len(sources)

	// Run validators
	a.runDateLogicValidator(doc, report)
	a.runReferenceValidator(doc, report)
	a.runDuplicateDetector(doc, report)

	// Calculate completeness metrics and generate completeness issues
	a.calculateCompleteness(individuals, report)

	// Aggregate issues by severity
	a.aggregateIssues(report)

	return report
}

// runDateLogicValidator runs date logic validation and adds issues to the report.
func (a *QualityAnalyzer) runDateLogicValidator(doc *gedcom.Document, report *QualityReport) {
	issues := a.dateLogic.Validate(doc)
	report.DateLogicIssues = issues
}

// runReferenceValidator runs reference validation and adds issues to the report.
func (a *QualityAnalyzer) runReferenceValidator(doc *gedcom.Document, report *QualityReport) {
	issues := a.references.Validate(doc)
	report.ReferenceIssues = issues
}

// runDuplicateDetector runs duplicate detection and converts results to issues.
func (a *QualityAnalyzer) runDuplicateDetector(doc *gedcom.Document, report *QualityReport) {
	pairs := a.duplicates.FindDuplicates(doc)
	for _, pair := range pairs {
		report.DuplicateIssues = append(report.DuplicateIssues, pair.ToIssue())
	}
}

// calculateCompleteness calculates data completeness metrics and generates completeness issues.
func (a *QualityAnalyzer) calculateCompleteness(individuals []*gedcom.Individual, report *QualityReport) {
	for _, ind := range individuals {
		// Check birth date
		if ind.BirthDate() != nil {
			report.IndividualsWithBirthDate++
		} else {
			// Generate MISSING_BIRTH_DATE issue
			issue := NewIssue(
				SeverityInfo,
				CodeMissingBirthDate,
				"individual has no birth date recorded",
				ind.XRef,
			)
			report.CompletenessIssues = append(report.CompletenessIssues, issue)
		}

		// Check death date
		if ind.DeathDate() != nil {
			report.IndividualsWithDeathDate++
		}

		// Check sources
		if len(ind.SourceCitations) > 0 {
			report.IndividualsWithSources++
		} else {
			// Generate NO_SOURCES issue
			issue := NewIssue(
				SeverityInfo,
				CodeNoSources,
				"individual has no source citations",
				ind.XRef,
			)
			report.CompletenessIssues = append(report.CompletenessIssues, issue)
		}

		// Check places (any event with a place)
		if a.hasPlace(ind) {
			report.IndividualsWithPlaces++
		}

		// Check names
		if len(ind.Names) == 0 {
			issue := NewIssue(
				SeverityInfo,
				CodeMissingName,
				"individual has no name recorded",
				ind.XRef,
			)
			report.CompletenessIssues = append(report.CompletenessIssues, issue)
		}
	}

	// Calculate percentages
	if report.TotalIndividuals > 0 {
		report.BirthDateCoverage = float64(report.IndividualsWithBirthDate) / float64(report.TotalIndividuals)
		report.DeathDateCoverage = float64(report.IndividualsWithDeathDate) / float64(report.TotalIndividuals)
		report.SourceCoverage = float64(report.IndividualsWithSources) / float64(report.TotalIndividuals)
	}
}

// hasPlace checks if an individual has any event with a place.
func (a *QualityAnalyzer) hasPlace(ind *gedcom.Individual) bool {
	for _, event := range ind.Events {
		if event.Place != "" || (event.PlaceDetail != nil && event.PlaceDetail.Name != "") {
			return true
		}
	}
	return false
}

// aggregateIssues collects all issues and sorts them by severity.
func (a *QualityAnalyzer) aggregateIssues(report *QualityReport) {
	// Collect all issues
	var allIssues []Issue
	allIssues = append(allIssues, report.DateLogicIssues...)
	allIssues = append(allIssues, report.ReferenceIssues...)
	allIssues = append(allIssues, report.DuplicateIssues...)
	allIssues = append(allIssues, report.CompletenessIssues...)

	// Sort by severity (Errors first, then Warnings, then Info)
	sort.Slice(allIssues, func(i, j int) bool {
		return allIssues[i].Severity < allIssues[j].Severity
	})

	// Categorize by severity
	for _, issue := range allIssues {
		switch issue.Severity {
		case SeverityError:
			report.Errors = append(report.Errors, issue)
			report.ErrorCount++
		case SeverityWarning:
			report.Warnings = append(report.Warnings, issue)
			report.WarningCount++
		case SeverityInfo:
			report.Info = append(report.Info, issue)
			report.InfoCount++
		}
	}

	report.TotalIssues = report.ErrorCount + report.WarningCount + report.InfoCount
}
