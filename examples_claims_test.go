package gedcomgo

// `make build-examples` builds every directory under examples/ by pattern, but
// `make examples` (which CI runs) and examples/README.md name each example by
// hand. A new example missing from either is never run, or never found, and
// nothing else notices.

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestEveryExampleIsRunAndDocumented(t *testing.T) {
	entries, err := os.ReadDir("examples")
	if err != nil {
		t.Fatal(err)
	}

	makefile := readFile(t, "Makefile")
	start := strings.Index(makefile, "\nexamples:")
	if start < 0 {
		t.Fatal("Makefile has no examples target")
	}
	target := makefile[start+1:]
	if end := strings.Index(target, "\n\n"); end >= 0 {
		target = target[:end]
	}
	readme := readFile(t, "examples/README.md")

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !regexp.MustCompile(`cd examples/` + regexp.QuoteMeta(name) + ` && `).MatchString(target) {
			t.Errorf("examples/%s is not run by `make examples`", name)
		}
		if !strings.Contains(readme, "("+name+"/main.go)") {
			t.Errorf("examples/README.md does not link examples/%s/main.go", name)
		}
	}
}
