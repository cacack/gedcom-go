package gedcom

import "testing"

func TestVersion_Before(t *testing.T) {
	tests := []struct {
		name  string
		v     Version
		other Version
		want  bool
	}{
		// Version ordering: 5.5 < 5.5.1 < 7.0
		{
			name:  "5.5 before 5.5.1",
			v:     Version55,
			other: Version551,
			want:  true,
		},
		{
			name:  "5.5 before 7.0",
			v:     Version55,
			other: Version70,
			want:  true,
		},
		{
			name:  "5.5.1 before 7.0",
			v:     Version551,
			other: Version70,
			want:  true,
		},
		// Reverse ordering (not before)
		{
			name:  "5.5.1 not before 5.5",
			v:     Version551,
			other: Version55,
			want:  false,
		},
		{
			name:  "7.0 not before 5.5",
			v:     Version70,
			other: Version55,
			want:  false,
		},
		{
			name:  "7.0 not before 5.5.1",
			v:     Version70,
			other: Version551,
			want:  false,
		},
		// Same version (not before)
		{
			name:  "5.5 not before 5.5",
			v:     Version55,
			other: Version55,
			want:  false,
		},
		{
			name:  "5.5.1 not before 5.5.1",
			v:     Version551,
			other: Version551,
			want:  false,
		},
		{
			name:  "7.0 not before 7.0",
			v:     Version70,
			other: Version70,
			want:  false,
		},
		// Unknown versions
		{
			name:  "unknown version not before 5.5",
			v:     Version("1.0"),
			other: Version55,
			want:  false,
		},
		{
			name:  "5.5 not before unknown version",
			v:     Version55,
			other: Version("1.0"),
			want:  false,
		},
		{
			name:  "empty version not before 5.5",
			v:     Version(""),
			other: Version55,
			want:  false,
		},
		{
			name:  "5.5 not before empty version",
			v:     Version55,
			other: Version(""),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.Before(tt.other); got != tt.want {
				t.Errorf("Version(%q).Before(%q) = %v, want %v", tt.v, tt.other, got, tt.want)
			}
		})
	}
}

func TestVersion_String(t *testing.T) {
	tests := []struct {
		v    Version
		want string
	}{
		{Version55, "5.5"},
		{Version551, "5.5.1"},
		{Version70, "7.0"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.v.String(); got != tt.want {
				t.Errorf("Version.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVersion_IsValid(t *testing.T) {
	tests := []struct {
		v    Version
		want bool
	}{
		{Version55, true},
		{Version551, true},
		{Version70, true},
		{Version(""), false},
		{Version("1.0"), false},
		{Version("6.0"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.v), func(t *testing.T) {
			if got := tt.v.IsValid(); got != tt.want {
				t.Errorf("Version(%q).IsValid() = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}
