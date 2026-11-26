package tagtree

import (
	"slices"
	"testing"
)

// level reads a test tag: a level, or nil for an absent tag.
func level(l *int) (int, bool) {
	if l == nil {
		return 0, false
	}
	return *l, true
}

func levels(ls ...int) []*int {
	out := make([]*int, len(ls))
	for i, l := range ls {
		if l >= 0 {
			out[i] = &ls[i]
		}
	}
	return out
}

func TestParents(t *testing.T) {
	tests := []struct {
		name   string
		levels []int // -1 is an absent tag
		want   []int
	}{
		{"empty", nil, []int{}},
		{"flat", []int{1, 1, 1}, []int{-1, -1, -1}},
		{"nested by one", []int{1, 2, 3, 2, 1}, []int{-1, 0, 1, 0, -1}},
		{"jump nests under the nearest shallower tag", []int{1, 3, 2}, []int{-1, 0, 0}},
		{"a shallower tag after a jump is not the jumped tag's child", []int{1, 4, 3, 4}, []int{-1, 0, 0, 2}},
		{"first tag deeper than 1 is top level", []int{2, 3, 1}, []int{-1, 0, -1}},
		{"absent tags are skipped and parent nothing", []int{1, -1, 2, -1}, []int{-1, -1, 0, -1}},
		{"level 0 parents deeper tags", []int{1, 0, 2}, []int{-1, -1, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Parents(levels(tt.levels...), level); !slices.Equal(got, tt.want) {
				t.Errorf("Parents(%v) = %v, want %v", tt.levels, got, tt.want)
			}
		})
	}
}
