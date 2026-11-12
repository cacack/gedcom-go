package charset

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
)

// readAllTiny reads r through a p of size n, so every replacement that does
// not fit in one read has to be carried over to the next.
func readAllTiny(t *testing.T, r io.Reader, n int) string {
	t.Helper()
	var out bytes.Buffer
	buf := make([]byte, n)
	for {
		m, err := r.Read(buf)
		out.Write(buf[:m])
		if err == io.EOF {
			return out.String()
		}
		if err != nil {
			t.Fatalf("Read() error = %v after %q", err, out.String())
		}
	}
}

// TestNewReaderWithOptions_ReplacesInvalidUTF8 pins the recovery #710 adds:
// each byte outside a valid UTF-8 sequence becomes U+FFFD, is reported once
// with its own line, column and value, and reading continues to the end.
// Without OnInvalidUTF8 the same inputs fail the read (see TestNewReader_*).
func TestNewReaderWithOptions_ReplacesInvalidUTF8(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		errs  []ErrInvalidUTF8
	}{
		{
			name:  "CP1252 byte in UTF-8 header",
			input: "0 HEAD\n1 CHAR UTF-8\n0 @I1@ INDI\n1 NOTE Caf\xE9 au lait\n0 TRLR\n",
			want:  "0 HEAD\n1 CHAR UTF-8\n0 @I1@ INDI\n1 NOTE Caf\uFFFD au lait\n0 TRLR\n",
			errs:  []ErrInvalidUTF8{{Line: 4, Column: 11, Byte: 0xE9}},
		},
		{
			name:  "no CHAR, two bytes on separate CRLF lines",
			input: "0 HEAD\r\n1 NOTE \x80\r\n1 NOTE a\xFFb\r\n0 TRLR\r\n",
			want:  "0 HEAD\r\n1 NOTE \uFFFD\r\n1 NOTE a\uFFFDb\r\n0 TRLR\r\n",
			errs:  []ErrInvalidUTF8{{Line: 2, Column: 8, Byte: 0x80}, {Line: 3, Column: 9, Byte: 0xFF}},
		},
		{
			name:  "CHAR ASCII with a high byte, valid UTF-8 kept",
			input: "0 HEAD\n1 CHAR ASCII\n1 NOTE é\xE9é\n",
			want:  "0 HEAD\n1 CHAR ASCII\n1 NOTE é\uFFFDé\n",
			errs:  []ErrInvalidUTF8{{Line: 3, Column: 10, Byte: 0xE9}},
		},
		{
			// Each byte of a sequence cut off by end of input is reported.
			name:  "incomplete sequence at end of input",
			input: "abc\xF0\x9F",
			want:  "abc\uFFFD\uFFFD",
			errs:  []ErrInvalidUTF8{{Line: 1, Column: 4, Byte: 0xF0}, {Line: 1, Column: 5, Byte: 0x9F}},
		},
		{
			name:  "valid input reports nothing",
			input: "0 HEAD\n1 NOTE Café\n",
			want:  "0 HEAD\n1 NOTE Café\n",
		},
	}

	readers := []struct {
		name string
		wrap func(io.Reader) io.Reader
		pLen int
	}{
		{"whole", func(r io.Reader) io.Reader { return r }, 4096},
		{"one byte in", iotest.OneByteReader, 4096},
		{"one byte out", func(r io.Reader) io.Reader { return r }, 1},
		{"one byte in and out", iotest.OneByteReader, 1},
	}

	for _, tt := range tests {
		for _, rd := range readers {
			t.Run(tt.name+"/"+rd.name, func(t *testing.T) {
				var got []ErrInvalidUTF8
				r := NewReaderWithOptions(rd.wrap(strings.NewReader(tt.input)), ReaderOptions{
					OnInvalidUTF8: func(e *ErrInvalidUTF8) { got = append(got, *e) },
				})
				if out := readAllTiny(t, r, rd.pLen); out != tt.want {
					t.Errorf("output = %q, want %q", out, tt.want)
				}
				if !reflect.DeepEqual(got, tt.errs) {
					t.Errorf("reported = %+v, want %+v", got, tt.errs)
				}
			})
		}
	}
}

// The zero ReaderOptions is NewReader: the first invalid byte fails the read,
// and the error now names the byte.
func TestNewReaderWithOptions_ZeroValueFails(t *testing.T) {
	r := NewReaderWithOptions(strings.NewReader("1 NOTE Caf\xE9\n"), ReaderOptions{})
	_, err := io.ReadAll(r)
	var utf8Err *ErrInvalidUTF8
	if !errors.As(err, &utf8Err) {
		t.Fatalf("ReadAll() error = %v (%T), want *ErrInvalidUTF8", err, err)
	}
	if *utf8Err != (ErrInvalidUTF8{Line: 1, Column: 11, Byte: 0xE9}) {
		t.Errorf("error = %+v, want line 1, column 11, byte 0xE9", *utf8Err)
	}
}

// Recovery also applies on the fallback path taken when header probing fails:
// the probed bytes are repaired and the read error is still reported.
func TestNewReaderWithOptions_ProbeErrorFallback(t *testing.T) {
	boom := errors.New("boom")
	var got []ErrInvalidUTF8
	r := NewReaderWithOptions(
		io.MultiReader(strings.NewReader("0 HEAD\n1 NOTE \xE9x"), iotest.ErrReader(boom)),
		ReaderOptions{OnInvalidUTF8: func(e *ErrInvalidUTF8) { got = append(got, *e) }},
	)
	out, err := io.ReadAll(r)
	if !errors.Is(err, boom) {
		t.Fatalf("ReadAll() error = %v, want %v", err, boom)
	}
	if string(out) != "0 HEAD\n1 NOTE \uFFFDx" {
		t.Errorf("output = %q, want the probed bytes with the byte replaced", out)
	}
	if want := []ErrInvalidUTF8{{Line: 2, Column: 8, Byte: 0xE9}}; !reflect.DeepEqual(got, want) {
		t.Errorf("reported = %+v, want %+v", got, want)
	}
}
