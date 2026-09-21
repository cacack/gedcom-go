package gedcom

import (
	"fmt"
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

		// Leading/trailing decimal point: accepted today, explicitly protected (#504).
		{name: "leading decimal point", input: "N.5", want: 0.5},
		{name: "trailing decimal point", input: "N5.", want: 5},

		// Non-decimal spellings that strconv.ParseFloat would otherwise accept (#504).
		// Bare spellings: the first character is consumed as the direction letter,
		// so these already errored; pinned so that stays true.
		{name: "bare NaN", input: "NaN", wantErr: true},
		{name: "bare nan lowercase", input: "nan", wantErr: true},
		{name: "bare Inf", input: "Inf", wantErr: true},
		{name: "bare +Inf", input: "+Inf", wantErr: true},
		{name: "bare -Inf", input: "-Inf", wantErr: true},
		{name: "bare Infinity", input: "Infinity", wantErr: true},
		{name: "bare hex float literal", input: "0x1p3", wantErr: true},
		// Direction-prefixed spellings: these are where the defect lived.
		{name: "north NaN", input: "Nnan", wantErr: true},
		{name: "north NaN mixed case", input: "NNaN", wantErr: true},
		{name: "north inf", input: "Ninf", wantErr: true},
		{name: "east inf", input: "Einf", wantErr: true},
		{name: "north infinity", input: "Ninfinity", wantErr: true},
		{name: "south infinity", input: "Sinfinity", wantErr: true},
		{name: "north hex float literal", input: "N0x1p3", wantErr: true},
		{name: "south hex float literal uppercase", input: "S0X1P-2", wantErr: true},
		{name: "north exponent notation", input: "N1e2", wantErr: true},
		{name: "north underscore separator", input: "N1_0.5", wantErr: true},
		{name: "north multiple decimal points", input: "N1.2.3", wantErr: true},
		{name: "north decimal point only", input: "N.", wantErr: true},
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
		// Non-finite spellings strconv.ParseFloat would otherwise accept (#504).
		{
			name:    "NaN pair",
			coords:  &Coordinates{Latitude: "Nnan", Longitude: "Enan"},
			wantErr: true,
		},
		{
			name:    "infinite pair",
			coords:  &Coordinates{Latitude: "Ninf", Longitude: "Einf"},
			wantErr: true,
		},
		{
			name:    "hex float literal pair",
			coords:  &Coordinates{Latitude: "N0x1p3", Longitude: "E0x1p3"},
			wantErr: true,
		},
		{
			name:    "valid latitude with NaN longitude",
			coords:  &Coordinates{Latitude: "N42.3601", Longitude: "Enan"},
			wantErr: true,
		},
		{
			name:    "NaN latitude with valid longitude",
			coords:  &Coordinates{Latitude: "Nnan", Longitude: "W71.0589"},
			wantErr: true,
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

// TestCoordinates_AsDecimal_NeverNonFinite pins the #504 invariant: a nil error
// from AsDecimal always means both returns are finite and within their axis
// range, and a non-nil error always means both returns are the documented zero
// values. This holds regardless of how the input is spelled.
func TestCoordinates_AsDecimal_NeverNonFinite(t *testing.T) {
	candidates := []*Coordinates{
		// Valid pairs.
		nil,
		{},
		{Latitude: "N0", Longitude: "E0"},
		{Latitude: "N42.3601", Longitude: "W71.0589"},
		{Latitude: "S33.8688", Longitude: "E151.2093"},
		{Latitude: "n42.3601", Longitude: "w71.0589"},
		{Latitude: "N90.0", Longitude: "W180.0"},
		{Latitude: "S90", Longitude: "E180"},
		{Latitude: "N.5", Longitude: "E.5"},
		{Latitude: "N5.", Longitude: "E5."},

		// Whitespace variants.
		{Latitude: "  N42.3601  ", Longitude: "\tW71.0589\n"},
		{Latitude: "   ", Longitude: "   "},
		{Latitude: "   ", Longitude: "W71.0589"},

		// Empty and single-component pairs.
		{Latitude: "N42.3601"},
		{Longitude: "W71.0589"},

		// Wrong-axis directions.
		{Latitude: "E42.3601", Longitude: "W71.0589"},
		{Latitude: "N42.3601", Longitude: "N71.0589"},
		{Latitude: "X42.3601", Longitude: "W71.0589"},

		// Out of range.
		{Latitude: "N91.0", Longitude: "W71.0589"},
		{Latitude: "N42.3601", Longitude: "W181.0"},

		// Non-decimal spellings strconv.ParseFloat would otherwise accept (#504).
		{Latitude: "Nnan", Longitude: "Enan"},
		{Latitude: "NNaN", Longitude: "ENaN"},
		{Latitude: "Ninf", Longitude: "Einf"},
		{Latitude: "Ninfinity", Longitude: "Einfinity"},
		{Latitude: "Sinfinity", Longitude: "Winfinity"},
		{Latitude: "N0x1p3", Longitude: "E0x1p3"},
		{Latitude: "S0X1P-2", Longitude: "W0X1P-2"},
		{Latitude: "N1e2", Longitude: "E1e2"},
		{Latitude: "N1_0.5", Longitude: "E1_0.5"},
		{Latitude: "N-42.3601", Longitude: "W71.0589"},
		{Latitude: "N+42.3601", Longitude: "W71.0589"},
		{Latitude: "N42.3601", Longitude: "Enan"},
		{Latitude: "Nnan", Longitude: "W71.0589"},
		{Latitude: "Nabc", Longitude: "Wxyz"},
		{Latitude: "N", Longitude: "E"},
	}

	for _, c := range candidates {
		t.Run(candidateName(c), func(t *testing.T) {
			lat, long, err := c.AsDecimal()
			if err != nil {
				if lat != 0 || long != 0 {
					t.Errorf("AsDecimal() = (%v, %v) with error %v, want zero values", lat, long, err)
				}
				return
			}
			if math.IsNaN(lat) || math.IsInf(lat, 0) {
				t.Errorf("AsDecimal() lat = %v with nil error, want finite", lat)
			}
			if math.IsNaN(long) || math.IsInf(long, 0) {
				t.Errorf("AsDecimal() long = %v with nil error, want finite", long)
			}
			if lat < -90 || lat > 90 {
				t.Errorf("AsDecimal() lat = %v with nil error, want within [-90, 90]", lat)
			}
			if long < -180 || long > 180 {
				t.Errorf("AsDecimal() long = %v with nil error, want within [-180, 180]", long)
			}
		})
	}
}

// candidateName builds a readable subtest name for a coordinate pair.
func candidateName(c *Coordinates) string {
	if c == nil {
		return "nil"
	}
	return fmt.Sprintf("%q/%q", c.Latitude, c.Longitude)
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

// TestParseLatitudeAndLongitude covers the axis-aware entry points. Every input
// is run through both, because what these add over ParseCoordinate is exactly
// the check a lone component cannot make for itself: that its direction letter
// and range belong to the axis the caller asked for.
func TestParseLatitudeAndLongitude(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLat  float64
		latErr   bool
		wantLong float64
		longErr  bool
	}{
		{name: "north", input: "N42.3601", wantLat: 42.3601, longErr: true},
		{name: "south", input: "S33.8688", wantLat: -33.8688, longErr: true},
		{name: "east", input: "E151.2093", latErr: true, wantLong: 151.2093},
		{name: "west", input: "W71.0589", latErr: true, wantLong: -71.0589},
		{name: "lowercase north", input: "n42.3601", wantLat: 42.3601, longErr: true},
		{name: "lowercase west", input: "w71.0589", latErr: true, wantLong: -71.0589},
		{name: "surrounding whitespace", input: "  N42.3601  ", wantLat: 42.3601, longErr: true},
		{name: "zero", input: "N0", wantLat: 0, longErr: true},
		{name: "latitude out of range", input: "N90.1", latErr: true, longErr: true},
		{name: "longitude out of range", input: "E180.1", latErr: true, longErr: true},
		{name: "longitude at bound", input: "E180", latErr: true, wantLong: 180},
		{name: "empty", input: "", latErr: true, longErr: true},
		{name: "whitespace only", input: "   ", latErr: true, longErr: true},
		{name: "non-finite", input: "Nnan", latErr: true, longErr: true},
		{name: "hex float", input: "N0x1p3", latErr: true, longErr: true},
		{name: "invalid direction", input: "X42.3601", latErr: true, longErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLatitude(tt.input)
			switch {
			case tt.latErr && err == nil:
				t.Errorf("ParseLatitude(%q) = %v, want error", tt.input, got)
			case !tt.latErr && err != nil:
				t.Errorf("ParseLatitude(%q) unexpected error: %v", tt.input, err)
			case !tt.latErr && math.Abs(got-tt.wantLat) > 1e-9:
				t.Errorf("ParseLatitude(%q) = %v, want %v", tt.input, got, tt.wantLat)
			}

			got, err = ParseLongitude(tt.input)
			switch {
			case tt.longErr && err == nil:
				t.Errorf("ParseLongitude(%q) = %v, want error", tt.input, got)
			case !tt.longErr && err != nil:
				t.Errorf("ParseLongitude(%q) unexpected error: %v", tt.input, err)
			case !tt.longErr && math.Abs(got-tt.wantLong) > 1e-9:
				t.Errorf("ParseLongitude(%q) = %v, want %v", tt.input, got, tt.wantLong)
			}
		})
	}
}

// TestParseAxisNeverNonFinite pins, for the single-component entry points, the
// guarantee TestCoordinates_AsDecimal_NeverNonFinite pins for the pair: a nil
// error implies a finite, in-range value.
func TestParseAxisNeverNonFinite(t *testing.T) {
	for _, s := range coordinateSeeds {
		if v, err := ParseLatitude(s); err == nil {
			if math.IsNaN(v) || math.IsInf(v, 0) || v < -90 || v > 90 {
				t.Errorf("ParseLatitude(%q) = %v with nil error, want finite and within [-90, 90]", s, v)
			}
		}
		if v, err := ParseLongitude(s); err == nil {
			if math.IsNaN(v) || math.IsInf(v, 0) || v < -180 || v > 180 {
				t.Errorf("ParseLongitude(%q) = %v with nil error, want finite and within [-180, 180]", s, v)
			}
		}
	}
}
