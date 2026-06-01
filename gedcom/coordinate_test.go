package gedcom

import (
	"math"
	"testing"
)

func TestParseCoordinate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{name: "north positive", input: "N42.3601", want: 42.3601},
		{name: "south negative", input: "S33.8688", want: -33.8688},
		{name: "east positive", input: "E151.2093", want: 151.2093},
		{name: "west negative", input: "W71.0589", want: -71.0589},
		{name: "lowercase direction", input: "n42.3601", want: 42.3601},
		{name: "lowercase west", input: "w71.0589", want: -71.0589},
		{name: "surrounding whitespace", input: "  N42.3601  ", want: 42.3601},
		{name: "zero value", input: "N0", want: 0},
		{name: "integer value", input: "S5", want: -5},
		{name: "empty string", input: "", wantErr: true},
		{name: "whitespace only", input: "   ", wantErr: true},
		{name: "invalid direction", input: "X42.3601", wantErr: true},
		{name: "no direction prefix", input: "42.3601", wantErr: true},
		{name: "non-numeric value", input: "Nabc", wantErr: true},
		{name: "direction only", input: "N", wantErr: true},
		{name: "signed value rejected (negative)", input: "N-42.3601", wantErr: true},
		{name: "signed value rejected (positive)", input: "N+42.3601", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCoordinate(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseCoordinate(%q) = %v, want error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseCoordinate(%q) unexpected error: %v", tt.input, err)
			}
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("ParseCoordinate(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCoordinates_AsDecimal(t *testing.T) {
	tests := []struct {
		name     string
		coords   *Coordinates
		wantLat  float64
		wantLong float64
		wantErr  bool
	}{
		{
			name:    "nil receiver returns zero no error",
			coords:  nil,
			wantLat: 0, wantLong: 0,
		},
		{
			name:    "origin (Null Island) returns zero no error",
			coords:  &Coordinates{Latitude: "N0", Longitude: "E0"},
			wantLat: 0, wantLong: 0,
		},
		{
			name:    "latitude wrong axis direction",
			coords:  &Coordinates{Latitude: "E42.3601", Longitude: "W71.0589"},
			wantErr: true,
		},
		{
			name:    "longitude wrong axis direction",
			coords:  &Coordinates{Latitude: "N42.3601", Longitude: "N71.0589"},
			wantErr: true,
		},
		{
			name:    "signed latitude value rejected",
			coords:  &Coordinates{Latitude: "N-42.3601", Longitude: "W71.0589"},
			wantErr: true,
		},
		{
			name:     "valid pair",
			coords:   &Coordinates{Latitude: "N42.3601", Longitude: "W71.0589"},
			wantLat:  42.3601,
			wantLong: -71.0589,
		},
		{
			name:     "southern eastern hemisphere",
			coords:   &Coordinates{Latitude: "S33.8688", Longitude: "E151.2093"},
			wantLat:  -33.8688,
			wantLong: 151.2093,
		},
		{
			name:     "both empty returns zero no error",
			coords:   &Coordinates{},
			wantLat:  0,
			wantLong: 0,
		},
		{
			name:    "missing longitude",
			coords:  &Coordinates{Latitude: "N42.3601"},
			wantErr: true,
		},
		{
			name:    "missing latitude",
			coords:  &Coordinates{Longitude: "W71.0589"},
			wantErr: true,
		},
		{
			name:    "malformed latitude",
			coords:  &Coordinates{Latitude: "Nxyz", Longitude: "W71.0589"},
			wantErr: true,
		},
		{
			name:    "malformed longitude",
			coords:  &Coordinates{Latitude: "N42.3601", Longitude: "Wxyz"},
			wantErr: true,
		},
		{
			name:    "latitude out of range",
			coords:  &Coordinates{Latitude: "N91.0", Longitude: "W71.0589"},
			wantErr: true,
		},
		{
			name:    "longitude out of range",
			coords:  &Coordinates{Latitude: "N42.3601", Longitude: "W181.0"},
			wantErr: true,
		},
		{
			name:     "boundary values",
			coords:   &Coordinates{Latitude: "N90.0", Longitude: "W180.0"},
			wantLat:  90,
			wantLong: -180,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lat, long, err := tt.coords.AsDecimal()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("AsDecimal() = (%v, %v), want error", lat, long)
				}
				return
			}
			if err != nil {
				t.Fatalf("AsDecimal() unexpected error: %v", err)
			}
			if math.Abs(lat-tt.wantLat) > 1e-9 {
				t.Errorf("AsDecimal() lat = %v, want %v", lat, tt.wantLat)
			}
			if math.Abs(long-tt.wantLong) > 1e-9 {
				t.Errorf("AsDecimal() long = %v, want %v", long, tt.wantLong)
			}
		})
	}
}

func TestCoordinates_IsEmpty(t *testing.T) {
	tests := []struct {
		name   string
		coords *Coordinates
		want   bool
	}{
		{name: "nil receiver", coords: nil, want: true},
		{name: "both empty", coords: &Coordinates{}, want: true},
		{name: "whitespace only", coords: &Coordinates{Latitude: "  ", Longitude: "\t"}, want: true},
		{name: "origin coordinates not empty", coords: &Coordinates{Latitude: "N0", Longitude: "E0"}, want: false},
		{name: "populated", coords: &Coordinates{Latitude: "N42.3601", Longitude: "W71.0589"}, want: false},
		{name: "only latitude", coords: &Coordinates{Latitude: "N42.3601"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.coords.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}
