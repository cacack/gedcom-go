package gedcom

import "testing"

func TestDate_PhraseSubstructure(t *testing.T) {
	tests := []struct {
		name  string
		date  *Date
		value string
		want  string
	}{
		{name: "nil date", date: nil, value: "1850", want: ""},
		{name: "no phrase", date: &Date{Original: "1850", Year: 1850}, value: "1850", want: ""},
		{name: "5.5.1 phrase in the value", date: &Date{Original: "(unknown)", Phrase: "unknown", IsPhrase: true}, value: "(unknown)", want: ""},
		{name: "value plus PHRASE", date: &Date{Original: "1850", Year: 1850, Phrase: "about 1850"}, value: "1850", want: "about 1850"},
		{name: "phrase-only date", date: &Date{Phrase: "while young", IsPhrase: true}, value: "", want: "while young"},
		{name: "phrase value overridden by PHRASE", date: &Date{Original: "(a)", Phrase: "b", IsPhrase: true}, value: "(a)", want: "b"},
		// A Date built in code need not set Original; the written value decides.
		{name: "hand-built value phrase without Original", date: &Date{Phrase: "unknown", IsPhrase: true}, value: "(unknown)", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.date.PhraseSubstructure(tt.value); got != tt.want {
				t.Errorf("PhraseSubstructure(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}
