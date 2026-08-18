package decoder

// spec7_harness_test.go checks the coverage harness itself.
//
// TestSpec7Coverage compares its output against a checked-in document, so it
// catches change but not degradation: if probe documents stopped decoding
// meaningfully, every pair would classify as raw and the fix would look like
// regenerating the document. These tests anchor the harness to facts that can
// be read directly out of the decoder, so degradation fails loudly instead.

import (
	"slices"
	"strings"
	"testing"
)

// TestSpec7SentinelIsUnknown verifies the assumption the raw (accepted) /
// raw (undiagnosed) split rests on: that the sentinel tag is not a structure
// the specification defines anywhere. A sentinel the decoder could legitimately
// recognize would silently reclassify every undiagnosed context.
func TestSpec7SentinelIsUnknown(t *testing.T) {
	for _, pair := range loadSpec7(t).pairs {
		if pair.tag == spec7SentinelTag {
			t.Fatalf("GEDCOM 7.0 defines %s under %s; the sentinel tag must be a tag "+
				"no decoder could recognize", spec7SentinelTag, spec7Term(pair.superstructure))
		}
	}
}

// TestSpec7ProbesAreWellFormed checks that each probe document differs from its
// control in exactly one line, and that the line the classifier reads
// diagnostics for is that line. Every status depends on this: a probe that
// differs in two lines attributes both to one structure.
func TestSpec7ProbesAreWellFormed(t *testing.T) {
	spec := loadSpec7(t)

	for _, pair := range spec.pairs {
		path, ok := spec.pathFor(pair)
		if !ok {
			continue // reported by TestSpec7Coverage
		}
		probe := spec.probeFor(path)

		with := strings.Split(strings.TrimSuffix(probe.with, "\n"), "\n")
		without := strings.Split(strings.TrimSuffix(probe.without, "\n"), "\n")

		name := spec7Term(pair.superstructure) + "." + pair.tag
		if len(with) != len(without)+1 {
			t.Errorf("%s: probe has %d lines, control has %d; expected exactly one more",
				name, len(with), len(without))
			continue
		}
		if probe.line < 1 || probe.line > len(with) {
			t.Errorf("%s: probe line %d is outside the document's %d lines",
				name, probe.line, len(with))
			continue
		}

		removed := with[probe.line-1]
		rest := append(append([]string{}, with[:probe.line-1]...), with[probe.line:]...)
		if strings.Join(rest, "\n") != strings.Join(without, "\n") {
			t.Errorf("%s: control is not the probe minus line %d (%q)\nprobe:\n%s\ncontrol:\n%s",
				name, probe.line, removed, probe.with, probe.without)
			continue
		}
		// The tag follows the level, except on a record line where a
		// cross-reference identifier comes between them.
		if fields := strings.Fields(removed); len(fields) < 2 ||
			(fields[1] != pair.tag && (len(fields) < 3 || fields[2] != pair.tag)) {
			t.Errorf("%s: line %d is %q, which does not carry the tag under test",
				name, probe.line, removed)
			continue
		}
		// A removed line the control still contains verbatim cannot change
		// anything, so the probe would report a confident status having
		// measured nothing. This is how HEAD.GEDC and the top-level TRLR were
		// wrong: the harness injected boilerplate identical to the line under
		// test. It is also how the next such bug will announce itself.
		if slices.Contains(without, removed) {
			t.Errorf("%s: line %d (%q) is still present in the control, so removing "+
				"it measures nothing\nprobe:\n%s", name, probe.line, removed, probe.with)
		}
	}
}

// TestSpec7AnchorStatuses pins four pairs, one per status, to what the decoder
// visibly does with them. Each is verifiable by reading the cited code, so a
// harness that stopped measuring would fail here rather than quietly reporting
// everything as unsupported.
func TestSpec7AnchorStatuses(t *testing.T) {
	anchors := []struct {
		superstructure string
		tag            string
		want           spec7Status
		because        string
	}{
		{
			superstructure: "record-INDI",
			tag:            "NAME",
			want:           spec7Typed,
			because:        "parseIndividual builds Individual.Names from it",
		},
		{
			superstructure: "record-INDI",
			tag:            "ALIA",
			want:           spec7RawAccepted,
			because:        "parseIndividual lists ALIA as recognized with no typed field (issue #375)",
		},
		{
			superstructure: "record-SNOTE",
			tag:            "CREA",
			want:           spec7RawFlagged,
			because:        "parseSharedNote falls through to addUnknownTag for CREA",
		},
		{
			superstructure: "HEAD",
			tag:            "DEST",
			want:           spec7RawUndiagnosed,
			because:        "buildHeader is given no diagnostic collector, so it reports nothing",
		},
	}

	spec := loadSpec7(t)
	prober := newSpec7Prober(spec)

	byName := map[string]spec7Pair{}
	for _, pair := range spec.pairs {
		byName[spec7Term(pair.superstructure)+"."+pair.tag] = pair
	}

	for _, anchor := range anchors {
		name := anchor.superstructure + "." + anchor.tag
		pair, ok := byName[name]
		if !ok {
			t.Errorf("%s is no longer in the specification data", name)
			continue
		}
		entry, ok := prober.measure(t, pair)
		if !ok {
			t.Errorf("%s could not be measured", name)
			continue
		}
		if entry.status != anchor.want {
			t.Errorf("%s: got %s, want %s (%s)", name, entry.status, anchor.want, anchor.because)
		}
	}
}
