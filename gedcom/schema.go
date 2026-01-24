package gedcom

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
	TagMappings map[string]string
}
