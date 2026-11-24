package mediatype

import "testing"

func TestIANA(t *testing.T) {
	tests := []struct {
		in, want string
		ok       bool
	}{
		{"JPG", "image/jpeg", true},
		{"JPEG", "image/jpeg", true},
		{"jpg", "image/jpeg", true},
		{" Tif ", "image/tiff", true},
		{"PNG", "image/png", true},
		{"GIF", "image/gif", true},
		{"PDF", "application/pdf", true},
		{"heic", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		got, ok := IANA(tt.in)
		if got != tt.want || ok != tt.ok {
			t.Errorf("IANA(%q) = %q, %v, want %q, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}
