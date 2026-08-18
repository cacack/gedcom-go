package decoder

// spec7_harness_test.go checks the GEDCOM 7.0 coverage harness itself.
//
// TestSpec7Coverage compares its output against a checked-in document, so it
// catches change but not degradation: if probe documents stopped decoding
// meaningfully, every pair would classify as raw and the fix would look like
// regenerating the document. These tests anchor the harness to facts that can
// be read directly out of the decoder, so degradation fails loudly instead.

import (
	"testing"
)

// TestSpec7SentinelIsUnknown verifies the assumption the raw (accepted) /
// raw (undiagnosed) split rests on: that the sentinel tag is not a structure
// the specification defines anywhere. A sentinel the decoder could legitimately
// recognize would silently reclassify every undiagnosed context.
func TestSpec7SentinelIsUnknown(t *testing.T) {
	for _, pair := range loadSpec7(t).pairs {
		if pair.tag == specSentinelTag {
			t.Fatalf("GEDCOM 7.0 defines %s under %s; the sentinel tag must be a tag "+
				"no decoder could recognize", specSentinelTag, spec7Term(pair.superstructure))
		}
	}
}

// TestSpec7ProbesAreWellFormed checks that each 7.0 probe document differs from
// its control in exactly one line, and that the line the classifier reads
// diagnostics for is that line.
func TestSpec7ProbesAreWellFormed(t *testing.T) {
	spec := loadSpec7(t)
	specProbesAreWellFormed(t, spec.specGraph, spec.specForm, func(pair specPair) string {
		return spec7Term(pair.superstructure) + "." + pair.tag
	})
}

// TestSpec7AnchorStatuses pins four pairs, one per status, to what the decoder
// visibly does with them. Each is verifiable by reading the cited code, so a
// harness that stopped measuring would fail here rather than quietly reporting
// everything as unsupported.
func TestSpec7AnchorStatuses(t *testing.T) {
	anchors := []struct {
		superstructure string
		tag            string
		want           specStatus
		because        string
	}{
		{
			superstructure: "record-INDI",
			tag:            "NAME",
			want:           specTyped,
			because:        "parseIndividual builds Individual.Names from it",
		},
		{
			superstructure: "record-INDI",
			tag:            "ALIA",
			want:           specRawAccepted,
			because:        "parseIndividual lists ALIA as recognized with no typed field (issue #375)",
		},
		{
			superstructure: "record-SNOTE",
			tag:            "CREA",
			want:           specRawFlagged,
			because:        "parseSharedNote falls through to addUnknownTag for CREA",
		},
		{
			superstructure: "HEAD",
			tag:            "DEST",
			want:           specRawUndiagnosed,
			because:        "buildHeader is given no diagnostic collector, so it reports nothing",
		},
	}

	spec := loadSpec7(t)
	prober := newSpecProber(spec.specGraph, spec.specForm)

	byName := map[string]specPair{}
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
		_, status, ok := prober.measure(t, pair)
		if !ok {
			t.Errorf("%s could not be measured", name)
			continue
		}
		if status != anchor.want {
			t.Errorf("%s: got %s, want %s (%s)", name, status, anchor.want, anchor.because)
		}
	}
}
