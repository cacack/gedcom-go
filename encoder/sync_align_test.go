package encoder

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// nodes builds one level-1 structure per value, all of tag NOTE.
func nodes(values ...string) []*tagNode {
	tags := make([]*gedcom.Tag, len(values))
	for i, v := range values {
		tags[i] = &gedcom.Tag{Level: 1, Tag: "NOTE", Value: v}
	}
	return tagTree(tags)
}

// TestAlignTreesTrimKeepsLCS asserts that matching the common prefix and
// suffix directly finds a common subsequence as long as the full LCS does,
// and that every pair it returns is in order and pairs identical structures.
func TestAlignTreesTrimKeepsLCS(t *testing.T) {
	for _, tt := range []struct{ a, b string }{
		{"abcde", "abcde"},
		{"abcde", "abXde"},
		{"abcde", "aXbcde"},
		{"abcde", "abde"},
		{"aaaa", "aa"},
		{"abab", "baba"},
		{"", "abc"},
		{"abc", ""},
		{"abcabc", "cbacba"},
	} {
		a, b := nodes(strings.Split(tt.a, "")...), nodes(strings.Split(tt.b, "")...)
		if tt.a == "" {
			a = nil
		}
		if tt.b == "" {
			b = nil
		}
		got := alignTrees(a, b)
		if want := len(lcsTrees(a, b)); len(got) != want {
			t.Errorf("%q/%q: %d matches, want the LCS length %d", tt.a, tt.b, len(got), want)
		}
		last := [2]int{-1, -1}
		for _, m := range got {
			if m[0] <= last[0] || m[1] <= last[1] {
				t.Errorf("%q/%q: matches out of order: %v", tt.a, tt.b, got)
			}
			if !sameTree(a[m[0]], b[m[1]]) {
				t.Errorf("%q/%q: match %v pairs different structures", tt.a, tt.b, m)
			}
			last = m
		}
	}
}

// TestAlignTreesCap asserts that a group whose differing middle exceeds
// maxAlignCells is not aligned by LCS: the common prefix and suffix are still
// matched, and nothing between them.
func TestAlignTreesCap(t *testing.T) {
	const n = 600 // (n+1)^2 > maxAlignCells
	a := []string{"head"}
	b := []string{"head"}
	for i := 0; i < n; i++ {
		a = append(a, fmt.Sprintf("a%d", i))
		b = append(b, fmt.Sprintf("b%d", i))
	}
	a, b = append(a, "tail"), append(b, "tail")
	got := alignTrees(nodes(a...), nodes(b...))
	want := [][2]int{{0, 0}, {n + 1, n + 1}}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("matches = %v, want only the prefix and suffix %v", got, want)
	}
}

// TestSyncTagsPastAlignCap asserts that SyncTags still brings a record past
// the alignment cap in step: every note of a record with more than the cap
// allows is rewritten, and the tags read back as the typed fields.
func TestSyncTagsPastAlignCap(t *testing.T) {
	const n = 600
	var b strings.Builder
	b.WriteString(syncHead + "0 @I1@ INDI\n1 NAME A /B/\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "1 NOTE old %d\n", i)
	}
	b.WriteString("0 TRLR\n")
	doc := decodeGEDCOM(t, b.String())
	indi := doc.Individuals()[0]
	for i := range indi.InlineNotes {
		indi.InlineNotes[i] = fmt.Sprintf("new %d", i)
	}
	if err := SyncTags(doc); err != nil {
		t.Fatal(err)
	}
	again := decodeGEDCOM(t, encodeToString(t, doc)).Individuals()[0]
	if len(again.InlineNotes) != n || again.InlineNotes[0] != "new 0" || again.InlineNotes[n-1] != fmt.Sprintf("new %d", n-1) {
		t.Errorf("InlineNotes after sync: %d notes, first %q", len(again.InlineNotes), again.InlineNotes[0])
	}
	if again.Names[0].Full != "A /B/" {
		t.Errorf("NAME = %q, want it untouched", again.Names[0].Full)
	}
}
