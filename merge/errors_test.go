package merge_test

import (
	"errors"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/merge"
)

// TestArgumentErrorSentinels checks each argument failure of Combine and
// RemapXRefs wraps its sentinel, wraps no other, and keeps its message.
func TestArgumentErrorSentinels(t *testing.T) {
	doc := func() *gedcom.Document {
		d := &gedcom.Document{Header: &gedcom.Header{}, XRefMap: map[string]*gedcom.Record{}}
		r := &gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual}
		d.Records = []*gedcom.Record{r}
		d.XRefMap["@I1@"] = r
		return d
	}
	identity := func(s string) string { return s }
	tests := []struct {
		name    string
		call    func() error
		want    error
		message string
	}{
		{"combine nil doc1", func() error { _, _, err := merge.Combine(nil, doc(), merge.CombineOptions{}); return err },
			merge.ErrNilDocument, "merge: doc1 is nil"},
		{"combine nil doc2", func() error { _, _, err := merge.Combine(doc(), nil, merge.CombineOptions{}); return err },
			merge.ErrNilDocument, "merge: doc2 is nil"},
		{"combine unknown strategy", func() error {
			_, _, err := merge.Combine(doc(), doc(), merge.CombineOptions{CollisionStrategy: 99})
			return err
		}, merge.ErrInvalidOptions, "merge: unknown CollisionStrategy 99"},
		{"combine missing prefix", func() error {
			_, _, err := merge.Combine(doc(), doc(), merge.CombineOptions{CollisionStrategy: merge.PrefixDoc2})
			return err
		}, merge.ErrInvalidOptions, "merge: PrefixDoc2 strategy requires non-empty Prefix"},
		{"remap nil doc", func() error { _, _, err := merge.RemapXRefs(nil, identity); return err },
			merge.ErrNilDocument, "merge: doc is nil"},
		{"remap nil transform", func() error { _, _, err := merge.RemapXRefs(doc(), nil); return err },
			merge.ErrInvalidOptions, "merge: transform is nil"},
	}
	sentinels := []error{merge.ErrNilDocument, merge.ErrInvalidOptions, merge.ErrInvalidRemap,
		merge.ErrIncompatibleHeader, merge.ErrXRefCollision}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil {
				t.Fatalf("error = nil, want %v", tt.want)
			}
			for _, s := range sentinels {
				if got := errors.Is(err, s); got != (s == tt.want) {
					t.Errorf("errors.Is(%q, %v) = %t, want %t", err, s, got, s == tt.want)
				}
			}
			if err.Error() != tt.message {
				t.Errorf("error = %q, want %q", err, tt.message)
			}
		})
	}
}
