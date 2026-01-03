package gedcom

import "testing"

func TestIndividual_BirthEvent(t *testing.T) {
	tests := []struct {
		name   string
		events []*Event
		want   *Event
	}{
		{
			name: "has birth event",
			events: []*Event{
				{Type: EventBirth, Date: "1 JAN 1850"},
				{Type: EventDeath, Date: "1 JAN 1920"},
			},
			want: &Event{Type: EventBirth, Date: "1 JAN 1850"},
		},
		{
			name: "multiple birth events returns first",
			events: []*Event{
				{Type: EventBirth, Date: "1 JAN 1850"},
				{Type: EventBirth, Date: "2 JAN 1850"},
				{Type: EventDeath, Date: "1 JAN 1920"},
			},
			want: &Event{Type: EventBirth, Date: "1 JAN 1850"},
		},
		{
			name: "no birth event",
			events: []*Event{
				{Type: EventDeath, Date: "1 JAN 1920"},
				{Type: EventBaptism, Date: "15 JAN 1850"},
			},
			want: nil,
		},
		{
			name:   "no events",
			events: []*Event{},
			want:   nil,
		},
		{
			name:   "nil events slice",
			events: nil,
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Individual{Events: tt.events}
			got := i.BirthEvent()

			if tt.want == nil {
				if got != nil {
					t.Errorf("BirthEvent() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("BirthEvent() = nil, want %v", tt.want)
				return
			}

			if got.Type != tt.want.Type || got.Date != tt.want.Date {
				t.Errorf("BirthEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIndividual_DeathEvent(t *testing.T) {
	tests := []struct {
		name   string
		events []*Event
		want   *Event
	}{
		{
			name: "has death event",
			events: []*Event{
				{Type: EventBirth, Date: "1 JAN 1850"},
				{Type: EventDeath, Date: "1 JAN 1920"},
			},
			want: &Event{Type: EventDeath, Date: "1 JAN 1920"},
		},
		{
			name: "multiple death events returns first",
			events: []*Event{
				{Type: EventBirth, Date: "1 JAN 1850"},
				{Type: EventDeath, Date: "1 JAN 1920"},
				{Type: EventDeath, Date: "2 JAN 1920"},
			},
			want: &Event{Type: EventDeath, Date: "1 JAN 1920"},
		},
		{
			name: "no death event",
			events: []*Event{
				{Type: EventBirth, Date: "1 JAN 1850"},
				{Type: EventBaptism, Date: "15 JAN 1850"},
			},
			want: nil,
		},
		{
			name:   "no events",
			events: []*Event{},
			want:   nil,
		},
		{
			name:   "nil events slice",
			events: nil,
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Individual{Events: tt.events}
			got := i.DeathEvent()

			if tt.want == nil {
				if got != nil {
					t.Errorf("DeathEvent() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("DeathEvent() = nil, want %v", tt.want)
				return
			}

			if got.Type != tt.want.Type || got.Date != tt.want.Date {
				t.Errorf("DeathEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIndividual_BirthDate(t *testing.T) {
	birthDate := mustParseDate("1 JAN 1850")
	deathDate := mustParseDate("1 JAN 1920")

	tests := []struct {
		name   string
		events []*Event
		want   *Date
	}{
		{
			name: "has birth date",
			events: []*Event{
				{Type: EventBirth, Date: "1 JAN 1850", ParsedDate: birthDate},
				{Type: EventDeath, Date: "1 JAN 1920", ParsedDate: deathDate},
			},
			want: birthDate,
		},
		{
			name: "birth event without parsed date",
			events: []*Event{
				{Type: EventBirth, Date: "1 JAN 1850", ParsedDate: nil},
				{Type: EventDeath, Date: "1 JAN 1920", ParsedDate: deathDate},
			},
			want: nil,
		},
		{
			name: "no birth event",
			events: []*Event{
				{Type: EventDeath, Date: "1 JAN 1920", ParsedDate: deathDate},
				{Type: EventBaptism, Date: "15 JAN 1850"},
			},
			want: nil,
		},
		{
			name:   "no events",
			events: []*Event{},
			want:   nil,
		},
		{
			name:   "nil events slice",
			events: nil,
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Individual{Events: tt.events}
			got := i.BirthDate()

			if tt.want == nil {
				if got != nil {
					t.Errorf("BirthDate() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("BirthDate() = nil, want %v", tt.want)
				return
			}

			if got.Original != tt.want.Original {
				t.Errorf("BirthDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIndividual_DeathDate(t *testing.T) {
	birthDate := mustParseDate("1 JAN 1850")
	deathDate := mustParseDate("1 JAN 1920")

	tests := []struct {
		name   string
		events []*Event
		want   *Date
	}{
		{
			name: "has death date",
			events: []*Event{
				{Type: EventBirth, Date: "1 JAN 1850", ParsedDate: birthDate},
				{Type: EventDeath, Date: "1 JAN 1920", ParsedDate: deathDate},
			},
			want: deathDate,
		},
		{
			name: "death event without parsed date",
			events: []*Event{
				{Type: EventBirth, Date: "1 JAN 1850", ParsedDate: birthDate},
				{Type: EventDeath, Date: "1 JAN 1920", ParsedDate: nil},
			},
			want: nil,
		},
		{
			name: "no death event",
			events: []*Event{
				{Type: EventBirth, Date: "1 JAN 1850", ParsedDate: birthDate},
				{Type: EventBaptism, Date: "15 JAN 1850"},
			},
			want: nil,
		},
		{
			name:   "no events",
			events: []*Event{},
			want:   nil,
		},
		{
			name:   "nil events slice",
			events: nil,
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Individual{Events: tt.events}
			got := i.DeathDate()

			if tt.want == nil {
				if got != nil {
					t.Errorf("DeathDate() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("DeathDate() = nil, want %v", tt.want)
				return
			}

			if got.Original != tt.want.Original {
				t.Errorf("DeathDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIndividual_FamilySearchURL tests the FamilySearchURL helper method.
// This returns the FamilySearch.org URL for the individual's record.
// Ref: Issue #80
func TestIndividual_FamilySearchURL(t *testing.T) {
	tests := []struct {
		name           string
		familySearchID string
		want           string
	}{
		{
			name:           "typical ID",
			familySearchID: "KWCJ-QN7",
			want:           "https://www.familysearch.org/tree/person/details/KWCJ-QN7",
		},
		{
			name:           "another ID",
			familySearchID: "ABCD-123",
			want:           "https://www.familysearch.org/tree/person/details/ABCD-123",
		},
		{
			name:           "empty ID returns empty string",
			familySearchID: "",
			want:           "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Individual{FamilySearchID: tt.familySearchID}
			got := i.FamilySearchURL()

			if got != tt.want {
				t.Errorf("FamilySearchURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
