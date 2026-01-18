// registry.go provides a custom tag registry for vendor-specific GEDCOM extensions.
//
// GEDCOM files often contain vendor-specific custom tags (underscore-prefixed like
// _MILT, _PRIM, _FOOT) that are not part of the official specification. This registry
// allows users to register definitions for these tags so they can be properly validated
// instead of being flagged as unknown.
//
// # Usage
//
// Create a registry and register custom tag definitions:
//
//	registry := validator.NewTagRegistry()
//	registry.Register("_MILT", validator.TagDefinition{
//	    Tag:            "_MILT",
//	    AllowedParents: []string{"INDI"},
//	    Description:    "Military service record",
//	})
//
// Validate a tag against the registry:
//
//	if issue := registry.ValidateTag("_MILT", "INDI", "Army"); issue != nil {
//	    fmt.Printf("Validation failed: %s\n", issue.Message)
//	}

package validator

import (
	"fmt"
	"regexp"
	"sort"
)

// Common regex patterns for tag value validation.
var (
	// XRefPattern matches GEDCOM cross-reference values (e.g., "@I1@", "@F123@").
	XRefPattern = regexp.MustCompile(`^@[A-Za-z0-9_]+@$`)

	// YesNoPattern matches GEDCOM boolean-style values ("Y" or "N").
	YesNoPattern = regexp.MustCompile(`^[YN]$`)
)

// TagDefinition describes a custom GEDCOM tag for validation purposes.
type TagDefinition struct {
	// Tag is the tag name (e.g., "_MILT"). Must be uppercase and typically
	// starts with underscore for vendor extensions.
	Tag string

	// AllowedParents specifies which parent tags can contain this tag.
	// For example, []string{"INDI"} means this tag is only valid under INDI records.
	// An empty slice means the tag is allowed anywhere (no parent restriction).
	AllowedParents []string

	// ValuePattern is an optional regex pattern for validating tag values.
	// If nil, any value is accepted.
	ValuePattern *regexp.Regexp

	// Description is a human-readable explanation of the tag's purpose.
	Description string
}

// TagRegistry stores definitions for custom GEDCOM tags and provides validation.
// The registry is safe for concurrent read access after construction.
type TagRegistry struct {
	tags map[string]TagDefinition
}

// NewTagRegistry creates a new empty TagRegistry.
func NewTagRegistry() *TagRegistry {
	return &TagRegistry{
		tags: make(map[string]TagDefinition),
	}
}

// Register adds a tag definition to the registry.
// Returns an error if a tag with the same name is already registered.
func (r *TagRegistry) Register(tag string, def TagDefinition) error {
	if _, exists := r.tags[tag]; exists {
		return fmt.Errorf("tag %q is already registered", tag)
	}
	// Ensure the Tag field matches the key
	def.Tag = tag
	r.tags[tag] = def
	return nil
}

// RegisterBatch adds multiple tag definitions to the registry.
// Returns an error if any tag is already registered; in this case,
// no tags from the batch are added (atomic operation).
func (r *TagRegistry) RegisterBatch(defs map[string]TagDefinition) error {
	// First pass: check for duplicates
	for tag := range defs {
		if _, exists := r.tags[tag]; exists {
			return fmt.Errorf("tag %q is already registered", tag)
		}
	}

	// Second pass: add all definitions
	for tag, def := range defs {
		def.Tag = tag
		r.tags[tag] = def
	}
	return nil
}

// Get retrieves a tag definition by name.
// Returns the definition and true if found, or an empty definition and false if not.
func (r *TagRegistry) Get(tag string) (TagDefinition, bool) {
	def, ok := r.tags[tag]
	return def, ok
}

// IsKnown returns true if the tag is registered in this registry.
func (r *TagRegistry) IsKnown(tag string) bool {
	_, ok := r.tags[tag]
	return ok
}

// Tags returns all registered tag names in sorted order.
func (r *TagRegistry) Tags() []string {
	tags := make([]string, 0, len(r.tags))
	for tag := range r.tags {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

// Len returns the number of registered tags.
func (r *TagRegistry) Len() int {
	return len(r.tags)
}

// ValidateTag validates a tag occurrence against registry rules.
// Returns nil if the tag is valid, or an Issue describing the validation failure.
//
// Validation checks:
//   - If the tag is not registered, returns nil (unknown tags are not validated here)
//   - If AllowedParents is non-empty and parent is not in the list, returns an Issue
//   - If ValuePattern is set and value doesn't match, returns an Issue
func (r *TagRegistry) ValidateTag(tag, parent, value string) *Issue {
	def, ok := r.tags[tag]
	if !ok {
		// Unknown tags are not validated by the registry
		return nil
	}

	// Check parent restriction
	if len(def.AllowedParents) > 0 {
		allowed := false
		for _, p := range def.AllowedParents {
			if p == parent {
				allowed = true
				break
			}
		}
		if !allowed {
			issue := NewIssue(
				SeverityWarning,
				CodeInvalidTagParent,
				fmt.Sprintf("tag %s is not allowed under %s (allowed: %v)", tag, parent, def.AllowedParents),
				"",
			).WithDetail("tag", tag).
				WithDetail("parent", parent).
				WithDetail("allowed_parents", fmt.Sprintf("%v", def.AllowedParents))
			return &issue
		}
	}

	// Check value pattern
	if def.ValuePattern != nil && value != "" {
		if !def.ValuePattern.MatchString(value) {
			issue := NewIssue(
				SeverityWarning,
				CodeInvalidTagValue,
				fmt.Sprintf("tag %s value %q does not match expected pattern", tag, value),
				"",
			).WithDetail("tag", tag).
				WithDetail("value", value).
				WithDetail("pattern", def.ValuePattern.String())
			return &issue
		}
	}

	return nil
}
