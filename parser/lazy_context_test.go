package parser

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/internal/ctxcheck"
)

const lazyContextInput = "0 HEAD\n0 @I1@ INDI\n1 NAME A\n0 @I2@ INDI\n0 @I3@ INDI\n0 TRLR\n"

// TestLazyParserBuildIndexContextStops pins that BuildIndexContext checks the
// context before each record (#743): cancelled after two checks it fails with
// an error wrapping context.Canceled and leaves the parser without an index.
func TestLazyParserBuildIndexContextStops(t *testing.T) {
	lp := NewLazyParser(strings.NewReader(lazyContextInput))

	err := lp.BuildIndexContext(ctxcheck.CancelAfter(2))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want one wrapping context.Canceled", err)
	}
	if lp.HasIndex() {
		t.Error("HasIndex() = true after a cancelled build, want false")
	}
}

// TestLazyParserBuildIndexContextCompletes pins that a live context indexes
// every record, as BuildIndex does.
func TestLazyParserBuildIndexContextCompletes(t *testing.T) {
	lp := NewLazyParser(strings.NewReader(lazyContextInput))
	if err := lp.BuildIndexContext(context.Background()); err != nil {
		t.Fatalf("BuildIndexContext: %v", err)
	}
	if got, want := strings.Join(lp.XRefs(), ","), "@I1@,@I2@,@I3@"; got != want {
		t.Errorf("XRefs() = %s, want %s", got, want)
	}
}
