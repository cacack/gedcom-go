package gedcom

import "testing"

func TestDate_PhraseSubstructure(t *testing.T) {
	tests := []struct {
		name string
		date *Date
		want string
	}{
		{name: "nil date", date: nil, want: ""},
		{name: "no phrase", date: &Date{Original: "1850", Year: 1850}, want: ""},
		{name: "5.5.1 phrase in the value", date: &Date{Original: "(unknown)", Phrase: "unknown", IsPhrase: true}, want: ""},
		{name: "value plus PHRASE", date: &Date{Original: "1850", Year: 1850, Phrase: "about 1850"}, want: "about 1850"},
		{name: "phrase-only date", date: &Date{Phrase: "while young", IsPhrase: true}, want: "while young"},
		{name: "phrase value overridden by PHRASE", date: &Date{Original: "(a)", Phrase: "b", IsPhrase: true}, want: "b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.date.PhraseSubstructure(); got != tt.want {
				t.Errorf("PhraseSubstructure() = %q, want %q", got, tt.want)
			}
		})
	}
}
