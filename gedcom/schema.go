package gedcom

import "strings"

// SchemaDefinition represents GEDCOM 7.0 schema mappings (SCHMA structure).
// Maps custom/extension tags to their URI definitions, enabling interoperability
// of vendor-specific extensions.
//
// GEDCOM 7.0 structure:
//
//	0 HEAD
//	  1 SCHMA
//	    2 TAG <TagName> <URI>
//
// Example:
//
//	0 HEAD
//	  1 SCHMA
//	    2 TAG _SKYPEID http://xmlns.com/foaf/0.1/skypeID
//	    2 TAG _FACEBOOK https://www.facebook.com/
type SchemaDefinition struct {
	// TagMappings maps tag names to URIs (e.g., "_SKYPEID" -> "http://xmlns.com/foaf/0.1/skypeID")
	TagMappings map[string]string `json:",omitempty"`
}

// headerSchema returns the schema the header h declares: its level-1 SCHMA
// structure's TAG lines, each "<TagName> <URI>", read from h.Tags. It is nil
// unless h.Version is GEDCOM 7.0 and h has a SCHMA; a later SCHMA replaces an
// earlier one, and a TAG with no URI is skipped.
func headerSchema(h *Header) *SchemaDefinition {
	if h == nil || h.Version != Version70 {
		return nil
	}
	var schema *SchemaDefinition
	inSchema := false
	for _, tag := range h.Tags {
		if tag == nil {
			continue
		}
		if tag.Level <= 1 {
			inSchema = tag.Level == 1 && tag.Tag == "SCHMA"
			if inSchema {
				schema = &SchemaDefinition{TagMappings: make(map[string]string)}
			}
			continue
		}
		if !inSchema || tag.Level != 2 || tag.Tag != "TAG" {
			continue
		}
		// Split on the first space, then trim, to allow extra spaces or tabs.
		if name, uri, ok := strings.Cut(tag.Value, " "); ok {
			schema.TagMappings[strings.TrimSpace(name)] = strings.TrimSpace(uri)
		}
	}
	return schema
}
