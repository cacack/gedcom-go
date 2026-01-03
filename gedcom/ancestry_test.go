package gedcom

import "testing"

func TestParseAPID(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantNil bool
		wantRaw string
		wantDB  string
		wantRec string
	}{
		{
			name:    "standard format with prefix",
			value:   "1,7602::2771226",
			wantNil: false,
			wantRaw: "1,7602::2771226",
			wantDB:  "7602",
			wantRec: "2771226",
		},
		{
			name:    "format without prefix",
			value:   "7602::2771226",
			wantNil: false,
			wantRaw: "7602::2771226",
			wantDB:  "7602",
			wantRec: "2771226",
		},
		{
			name:    "different database and record IDs",
			value:   "1,9024::12345678",
			wantNil: false,
			wantRaw: "1,9024::12345678",
			wantDB:  "9024",
			wantRec: "12345678",
		},
		{
			name:    "prefix with different number",
			value:   "2,5678::999",
			wantNil: false,
			wantRaw: "2,5678::999",
			wantDB:  "5678",
			wantRec: "999",
		},
		{
			name:    "empty string",
			value:   "",
			wantNil: true,
		},
		{
			name:    "no separator",
			value:   "123456",
			wantNil: true,
		},
		{
			name:    "single colon",
			value:   "1,7602:2771226",
			wantNil: true,
		},
		{
			name:    "missing record ID",
			value:   "1,7602::",
			wantNil: true,
		},
		{
			name:    "missing database ID",
			value:   "1,::2771226",
			wantNil: true,
		},
		{
			name:    "only separator",
			value:   "::",
			wantNil: true,
		},
		{
			name:    "only comma and separator",
			value:   ",::2771226",
			wantNil: true,
		},
		{
			name:    "complex record with spaces (edge case)",
			value:   "1,7602::abc def",
			wantNil: false,
			wantRaw: "1,7602::abc def",
			wantDB:  "7602",
			wantRec: "abc def",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAPID(tt.value)
			if tt.wantNil {
				if got != nil {
					t.Errorf("ParseAPID(%q) = %+v, want nil", tt.value, got)
				}
				return
			}

			if got == nil {
				t.Errorf("ParseAPID(%q) = nil, want non-nil", tt.value)
				return
			}

			if got.Raw != tt.wantRaw {
				t.Errorf("ParseAPID(%q).Raw = %q, want %q", tt.value, got.Raw, tt.wantRaw)
			}
			if got.Database != tt.wantDB {
				t.Errorf("ParseAPID(%q).Database = %q, want %q", tt.value, got.Database, tt.wantDB)
			}
			if got.Record != tt.wantRec {
				t.Errorf("ParseAPID(%q).Record = %q, want %q", tt.value, got.Record, tt.wantRec)
			}
		})
	}
}

func TestAncestryAPID_URL(t *testing.T) {
	tests := []struct {
		name string
		apid *AncestryAPID
		want string
	}{
		{
			name: "standard APID",
			apid: &AncestryAPID{
				Raw:      "1,7602::2771226",
				Database: "7602",
				Record:   "2771226",
			},
			want: "https://www.ancestry.com/discoveryui-content/view/2771226:7602",
		},
		{
			name: "different IDs",
			apid: &AncestryAPID{
				Raw:      "1,9024::12345678",
				Database: "9024",
				Record:   "12345678",
			},
			want: "https://www.ancestry.com/discoveryui-content/view/12345678:9024",
		},
		{
			name: "nil APID",
			apid: nil,
			want: "",
		},
		{
			name: "empty database",
			apid: &AncestryAPID{
				Raw:      "invalid",
				Database: "",
				Record:   "123",
			},
			want: "",
		},
		{
			name: "empty record",
			apid: &AncestryAPID{
				Raw:      "invalid",
				Database: "123",
				Record:   "",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.apid.URL()
			if got != tt.want {
				t.Errorf("AncestryAPID.URL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseAPIDRoundTrip(t *testing.T) {
	// Verify that parsing an APID preserves the raw value for encoding
	testCases := []string{
		"1,7602::2771226",
		"7602::2771226",
		"1,9024::12345678",
	}

	for _, value := range testCases {
		t.Run(value, func(t *testing.T) {
			apid := ParseAPID(value)
			if apid == nil {
				t.Fatalf("ParseAPID(%q) returned nil", value)
			}
			if apid.Raw != value {
				t.Errorf("ParseAPID(%q).Raw = %q, want %q", value, apid.Raw, value)
			}
		})
	}
}
