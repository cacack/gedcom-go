package gedcom

import "testing"

// Dedicated contract test for schema.go. SchemaDefinition is a pure data
// type with no methods, so this pins its shape and usage semantics in
// isolation: the zero value has a nil map, and TagMappings models the
// GEDCOM 7.0 SCHMA "TAG <name> <URI>" structure as a name -> URI lookup.
//
// Deep-copy behavior is deliberately not retested here — it is covered by
// clone_test.go's "Schema is deep copied" case.

func TestSchemaDefinition_ZeroValue(t *testing.T) {
	var s SchemaDefinition
	if s.TagMappings != nil {
		t.Errorf("zero-value TagMappings = %v, want nil", s.TagMappings)
	}
	// Reading from a nil map is safe and yields the zero value.
	if got := s.TagMappings["_ANY"]; got != "" {
		t.Errorf("lookup on nil map = %q, want empty", got)
	}
}

func TestSchemaDefinition_TagMappings(t *testing.T) {
	s := SchemaDefinition{
		TagMappings: map[string]string{
			"_SKYPEID":  "http://xmlns.com/foaf/0.1/skypeID",
			"_FACEBOOK": "https://www.facebook.com/",
		},
	}

	tests := []struct {
		name string
		tag  string
		want string
	}{
		{"mapped extension tag", "_SKYPEID", "http://xmlns.com/foaf/0.1/skypeID"},
		{"second mapped tag", "_FACEBOOK", "https://www.facebook.com/"},
		{"unmapped tag yields empty", "_UNKNOWN", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.TagMappings[tt.tag]; got != tt.want {
				t.Errorf("TagMappings[%q] = %q, want %q", tt.tag, got, tt.want)
			}
		})
	}

	if len(s.TagMappings) != 2 {
		t.Errorf("len(TagMappings) = %d, want 2", len(s.TagMappings))
	}
}
