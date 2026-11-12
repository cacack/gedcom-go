package gedcom

import "testing"

func TestAddressText(t *testing.T) {
	tests := []struct {
		name string
		addr *Address
		want string
	}{
		{
			name: "nil",
			addr: nil,
			want: "",
		},
		{
			name: "empty",
			addr: &Address{City: "Springfield", Country: "USA"},
			want: "",
		},
		{
			name: "payload only",
			addr: &Address{Full: "123 Main St\nSpringfield"},
			want: "123 Main St\nSpringfield",
		},
		{
			name: "ADR1-ADR3 only",
			addr: &Address{Line1: "123 Main St", Line2: "Apt 4", Line3: "Rear"},
			want: "123 Main St\nApt 4\nRear",
		},
		{
			name: "ADR1 and ADR3 only skips empty ADR2",
			addr: &Address{Line1: "123 Main St", Line3: "Rear"},
			want: "123 Main St\nRear",
		},
		{
			name: "ADR2 only",
			addr: &Address{Line2: "Apt 4"},
			want: "Apt 4",
		},
		{
			name: "payload and ADR1-ADR3 prefers payload",
			addr: &Address{Full: "123 Main St, Apt 4", Line1: "123 Main St", Line2: "Apt 4"},
			want: "123 Main St, Apt 4",
		},
		{
			name: "payload kept untrimmed",
			addr: &Address{Full: "  123 Main St"},
			want: "  123 Main St",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.addr.Text(); got != tt.want {
				t.Errorf("Text() = %q, want %q", got, tt.want)
			}
		})
	}
}
