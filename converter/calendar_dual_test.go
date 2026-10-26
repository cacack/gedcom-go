package converter

import "testing"

func TestHasDualYear(t *testing.T) {
	tests := map[string]bool{
		"11 FEB 1731/32":             true,
		"1731/32":                    true,
		"BET 1750/51 AND 1760":       true,
		"JULIAN 1 JAN 1700/01":       true,
		"1750":                       false,
		"INT 1800 (born 1799/00)":    false,
		"(1799/00) 1800/01":          true,
		"(unclosed 1799/00":          false,
		"ABT 1/":                     false,
		"/32":                        false,
		"FROM 1700/01 TO 1702 (a/b)": true,
		"":                           false,
	}
	for value, want := range tests {
		if got := hasDualYear(value); got != want {
			t.Errorf("hasDualYear(%q) = %v, want %v", value, got, want)
		}
	}
}
