package merge

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/specgraph"
)

// TestOnceTagsMatchSpec holds onceTags to the vendored specification tables:
// for each version, the level-1 tags of a SOUR and a REPO record whose
// cardinality allows at most one.
func TestOnceTagsMatchSpec(t *testing.T) {
	versions := []struct {
		version gedcom.Version
		dir     string
		prefix  string
	}{
		{gedcom.Version55, "gedcom-5.5", ""},
		{gedcom.Version551, "gedcom-5.5.1", ""},
		{gedcom.Version70, "gedcom-7.0", specgraph.V7Prefix},
	}
	for _, v := range versions {
		dir := filepath.Join("..", "testdata", "spec", v.dir)
		graph, err := specgraph.Load(filepath.Join(dir, "substructures.tsv"), v.prefix)
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(dir, "cardinalities.tsv"))
		if err != nil {
			t.Fatal(err)
		}
		// repeats[super][structure] is whether the structure may repeat.
		repeats := map[string]map[string]bool{}
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n")[1:] {
			fields := strings.Split(line, "\t")
			if len(fields) != 3 {
				t.Fatalf("%s: malformed cardinality row %q", v.dir, line)
			}
			super, structure := strings.TrimPrefix(fields[0], v.prefix), strings.TrimPrefix(fields[1], v.prefix)
			if repeats[super] == nil {
				repeats[super] = map[string]bool{}
			}
			repeats[super][structure] = !strings.HasSuffix(strings.Trim(fields[2], "{}"), ":1")
		}

		for typ, tag := range map[gedcom.RecordType]string{gedcom.RecordTypeSource: "SOUR", gedcom.RecordTypeRepository: "REPO"} {
			records := graph[""][tag]
			if len(records) != 1 {
				t.Fatalf("%s: %d record structures for %s, want 1", v.dir, len(records), tag)
			}
			var want []string
			for child, structures := range graph[records[0]] {
				if !slices.ContainsFunc(structures, func(s string) bool { return repeats[records[0]][s] }) {
					want = append(want, child)
				}
			}
			slices.Sort(want)
			if got := onceTags[typ][v.version]; !reflect.DeepEqual(got, want) {
				t.Errorf("onceTags[%s][%s] = %v, the specification tables give %v", typ, v.version, got, want)
			}
		}
	}
}
