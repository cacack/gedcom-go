package encoder

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/ctxcheck"
)

// threeIndividuals returns a document of three individuals, @I1@ to @I3@.
func threeIndividuals() *gedcom.Document {
	doc := &gedcom.Document{Header: &gedcom.Header{Version: gedcom.Version551}}
	for _, x := range []string{"@I1@", "@I2@", "@I3@"} {
		doc.Records = append(doc.Records, &gedcom.Record{
			XRef: x, Type: gedcom.RecordTypeIndividual,
			Tags: []*gedcom.Tag{{Level: 1, Tag: "SEX", Value: "M"}},
		})
	}
	return doc
}

// TestEncodeContextCheckedPerRecord pins that EncodeWithOptions checks
// EncodeOptions.Context before each record (#743): cancelled after the first
// check, the first record is written, the rest and the trailer are not, and
// the error wraps context.Canceled.
func TestEncodeContextCheckedPerRecord(t *testing.T) {
	var buf bytes.Buffer
	opts := DefaultOptions()
	opts.Context = ctxcheck.CancelAfter(1)

	err := EncodeWithOptions(&buf, threeIndividuals(), opts)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want one wrapping context.Canceled", err)
	}
	out := buf.String()
	if !strings.Contains(out, "0 @I1@ INDI") {
		t.Errorf("output lacks @I1@, written before the cancellation:\n%s", out)
	}
	for _, unwanted := range []string{"@I2@", "@I3@", "TRLR"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("output contains %s, past the cancellation:\n%s", unwanted, out)
		}
	}
}

// TestEncodeStreamingContextCheckedPerRecord is the streaming counterpart: it
// stops at the second record (index 1), the first after the cancellation.
// The stream is not flushed on an error, so the error names the record.
func TestEncodeStreamingContextCheckedPerRecord(t *testing.T) {
	opts := DefaultOptions()
	opts.Context = ctxcheck.CancelAfter(1)

	err := EncodeStreamingWithOptions(&bytes.Buffer{}, threeIndividuals(), opts)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want one wrapping context.Canceled", err)
	}
	if !strings.HasPrefix(err.Error(), "write record 1:") {
		t.Errorf("err = %q, want it to stop at record 1, the first past the cancellation", err)
	}
}

// TestStreamEncoderKeepsCancellation pins that a StreamEncoder whose context
// is done refuses the record and keeps the error, as it does a write error.
func TestStreamEncoderKeepsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	opts := DefaultOptions()
	opts.Context = ctx
	var buf bytes.Buffer
	enc := NewStreamEncoderWithOptions(&buf, opts)
	if err := enc.WriteHeader(&gedcom.Header{Version: gedcom.Version551}); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	cancel()

	rec := &gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual}
	if err := enc.WriteRecord(rec); !errors.Is(err, context.Canceled) {
		t.Fatalf("WriteRecord err = %v, want one wrapping context.Canceled", err)
	}
	if err := enc.Err(); !errors.Is(err, context.Canceled) {
		t.Errorf("Err() = %v, want the kept cancellation", err)
	}
	_ = enc.Close()
	if strings.Contains(buf.String(), "@I1@") {
		t.Errorf("record written after cancellation:\n%s", buf.String())
	}
}
