package encoder

import (
	"reflect"
	"strings"
	"testing"
)

// Issue #786: a SUBM, REPO, NOTE or SNOTE record written from its entity
// writes back the CHAN, CREA, REFN, UID and NOTE-record SOUR that the decoder
// now types, so a cleared-Tags round trip keeps them.

func TestRecordMetadataClearedTagsRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		in   string
		// want are blocks the entity-written output must contain.
		want []string
	}{
		{
			name: "5.5.1",
			in: "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
				"0 @U1@ SUBM\n1 NAME Submitter\n1 _UID u-1\n1 CHAN\n2 DATE 1 JAN 2000\n3 TIME 10:20:30\n2 NOTE Changed\n" +
				"0 @R1@ REPO\n1 NAME Archive\n1 REFN R-1\n2 TYPE shelf\n1 CHAN\n2 DATE 2 FEB 2001\n" +
				"0 @S1@ SOUR\n1 TITL Register\n" +
				"0 @N1@ NOTE Note text\n1 REFN N-1\n2 TYPE note number\n1 _UID n-1\n1 SOUR @S1@\n2 PAGE p. 7\n1 CHAN\n2 DATE 3 MAR 2002\n" +
				"0 TRLR\n",
			want: []string{
				"0 @U1@ SUBM\n1 NAME Submitter\n1 _UID u-1\n1 CHAN\n2 DATE 1 JAN 2000\n3 TIME 10:20:30\n2 NOTE Changed\n",
				"0 @R1@ REPO\n1 NAME Archive\n1 REFN R-1\n2 TYPE shelf\n1 CHAN\n2 DATE 2 FEB 2001\n",
				"0 @N1@ NOTE Note text\n1 REFN N-1\n2 TYPE note number\n1 _UID n-1\n1 SOUR @S1@\n2 PAGE p. 7\n1 CHAN\n2 DATE 3 MAR 2002\n",
			},
		},
		{
			name: "7.0",
			in: "0 HEAD\n1 GEDC\n2 VERS 7.0\n" +
				"0 @U1@ SUBM\n1 NAME Submitter\n1 REFN U-1\n2 TYPE submitter number\n1 UID u-2\n" +
				"1 CHAN\n2 DATE 4 APR 2003\n1 CREA\n2 DATE 5 MAY 2004\n3 TIME 01:02:03\n" +
				"0 @R1@ REPO\n1 NAME Archive\n1 UID u-3\n1 UID u-4\n1 CHAN\n2 DATE 6 JUN 2005\n1 CREA\n2 DATE 7 JUL 2006\n" +
				"0 @N1@ SNOTE Shared text\n1 REFN S-1\n2 TYPE shared note number\n1 UID u-5\n" +
				"1 CHAN\n2 DATE 8 AUG 2007\n1 CREA\n2 DATE 9 SEP 2008\n" +
				"0 TRLR\n",
			want: []string{
				"0 @U1@ SUBM\n1 NAME Submitter\n1 REFN U-1\n2 TYPE submitter number\n1 UID u-2\n" +
					"1 CHAN\n2 DATE 4 APR 2003\n1 CREA\n2 DATE 5 MAY 2004\n3 TIME 01:02:03\n",
				"0 @R1@ REPO\n1 NAME Archive\n1 UID u-3\n1 UID u-4\n1 CHAN\n2 DATE 6 JUN 2005\n1 CREA\n2 DATE 7 JUL 2006\n",
				"0 @N1@ SNOTE Shared text\n1 REFN S-1\n2 TYPE shared note number\n1 UID u-5\n" +
					"1 CHAN\n2 DATE 8 AUG 2007\n1 CREA\n2 DATE 9 SEP 2008\n",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := decodeGEDCOM(t, tt.in)
			doc := decodeGEDCOM(t, tt.in)
			clearRecordTags(doc)
			out := encodeToString(t, doc)
			for _, want := range tt.want {
				if !strings.Contains(out, want) {
					t.Errorf("output missing\n%s\ngot:\n%s", want, out)
				}
			}

			again := decodeGEDCOM(t, out)
			if len(again.Records) != len(orig.Records) {
				t.Fatalf("record count %d, want %d", len(again.Records), len(orig.Records))
			}
			for i, rec := range orig.Records {
				if diffs := typedDiffs(reflect.ValueOf(rec.Entity), reflect.ValueOf(again.Records[i].Entity), rec.XRef); len(diffs) > 0 {
					t.Errorf("%s: typed fields differ after the round trip: %v", rec.XRef, diffs)
				}
			}
		})
	}
}
