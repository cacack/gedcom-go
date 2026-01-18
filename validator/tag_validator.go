// tag_validator.go provides validation for custom (underscore-prefixed) GEDCOM tags.
//
// Custom tags are vendor-specific extensions that start with an underscore (e.g., _MILT,
// _PRIM). This validator checks custom tags against a TagRegistry to ensure they appear
// under valid parent tags and have correctly formatted values.

package validator

import (
	"fmt"
	"strings"

	"github.com/cacack/gedcom-go/gedcom"
)

// TagValidator validates custom (underscore-prefixed) tags against a registry.
type TagValidator struct {
	registry        *TagRegistry
	validateUnknown bool
}

// NewTagValidator creates a new TagValidator.
//
// Parameters:
//   - registry: The TagRegistry containing custom tag definitions
//   - validateUnknown: If true, report unknown custom tags as warnings
//
// If registry is nil, validation will only report unknown custom tags (if validateUnknown is true).
func NewTagValidator(registry *TagRegistry, validateUnknown bool) *TagValidator {
	return &TagValidator{
		registry:        registry,
		validateUnknown: validateUnknown,
	}
}

// Validate scans all tags in the document and validates custom tags.
//
// Validation logic:
//   - Only underscore-prefixed tags (custom/vendor tags) are checked
//   - If a tag is in the registry, validate parent and value constraints
//   - If a tag is NOT in the registry and validateUnknown is true, report as warning
//   - Standard GEDCOM tags (non-underscore) are not checked
func (v *TagValidator) Validate(doc *gedcom.Document) []Issue {
	if doc == nil {
		return nil
	}

	var issues []Issue

	for _, record := range doc.Records {
		// The record type tag (INDI, FAM, etc.) is the parent for level 1 tags
		recordTypeTag := string(record.Type)

		// Scan all tags in the record
		v.scanTags(record.Tags, recordTypeTag, record.XRef, &issues)
	}

	return issues
}

// scanTags recursively scans tags and validates custom tags.
// Tags are stored in a flat list with Level indicating hierarchy.
// We use the level to track parent context.
func (v *TagValidator) scanTags(tags []*gedcom.Tag, recordType, recordXRef string, issues *[]Issue) {
	if len(tags) == 0 {
		return
	}

	// Build a stack to track parent tags at each level
	// Index corresponds to level, value is the tag name at that level
	parentStack := make([]string, 0, 10)
	parentStack = append(parentStack, recordType) // Level 0 parent is the record type

	for _, tag := range tags {
		// Determine the parent tag for this tag
		// For a tag at level N, the parent is at level N-1
		parent := ""
		if tag.Level > 0 && tag.Level-1 < len(parentStack) {
			parent = parentStack[tag.Level-1]
		} else if tag.Level == 1 {
			// Level 1 tags have the record type as parent
			parent = recordType
		}

		// Validate if it's a custom tag (underscore-prefixed)
		if strings.HasPrefix(tag.Tag, "_") {
			v.validateCustomTag(tag, parent, recordXRef, issues)
		}

		// Update parent stack for subsequent tags
		// Ensure stack has enough capacity
		for len(parentStack) <= tag.Level {
			parentStack = append(parentStack, "")
		}
		parentStack[tag.Level] = tag.Tag

		// Truncate stack to current level + 1 to clear deeper levels
		if tag.Level+1 < len(parentStack) {
			parentStack = parentStack[:tag.Level+1]
		}
	}
}

// validateCustomTag validates a single custom tag against the registry.
func (v *TagValidator) validateCustomTag(tag *gedcom.Tag, parent, recordXRef string, issues *[]Issue) {
	// Check if tag is known in the registry
	if v.registry != nil && v.registry.IsKnown(tag.Tag) {
		// Tag is registered - validate against definition
		if issue := v.registry.ValidateTag(tag.Tag, parent, tag.Value); issue != nil {
			// Upgrade severity based on the requirement
			// INVALID_TAG_PARENT and INVALID_TAG_VALUE should be Error severity
			if issue.Code == CodeInvalidTagParent || issue.Code == CodeInvalidTagValue {
				issue.Severity = SeverityError
			}
			issue.RecordXRef = recordXRef
			issue.Details["line_number"] = fmt.Sprintf("%d", tag.LineNumber)
			*issues = append(*issues, *issue)
		}
		return
	}

	// Tag is not in registry
	if v.validateUnknown {
		issue := NewIssue(
			SeverityWarning,
			CodeUnknownCustomTag,
			fmt.Sprintf("unknown custom tag %s", tag.Tag),
			recordXRef,
		).WithDetail("tag", tag.Tag).
			WithDetail("parent", parent).
			WithDetail("line_number", fmt.Sprintf("%d", tag.LineNumber))

		*issues = append(*issues, issue)
	}
}
