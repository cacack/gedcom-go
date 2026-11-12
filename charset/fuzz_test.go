package charset

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"
)

// FuzzNewReader fuzzes the encoding detection cascade and the ANSEL, UTF-16
// and UTF-8 readers behind NewReader with arbitrary bytes. Errors are
// expected; panics and unbounded output are not. Seeds come from the
// encoding fixtures plus the shapes that have caused trouble: a CHAR ANSEL
// header over a long run of combining marks (#716), and marks before a line
// break or at end of stream (#431). The same input is also read with
// OnInvalidUTF8 set, whose output must always be valid UTF-8 (#710).
func FuzzNewReader(f *testing.F) {
	const maxSeedSize = 1 << 20
	files, _ := filepath.Glob("../testdata/encoding/*.ged")
	for _, file := range files {
		if info, err := os.Stat(file); err != nil || info.Size() > maxSeedSize {
			continue
		}
		data, err := os.ReadFile(file) // #nosec G304 -- path comes from the repo's own testdata
		if err != nil {
			continue
		}
		f.Add(data)
	}

	anselHeader := []byte("0 HEAD\n1 CHAR ANSEL\n0 @N1@ NOTE ")
	f.Add(append(append([]byte{}, anselHeader...), bytes.Repeat([]byte{0xE2}, 4*MaxPendingCombining)...))
	f.Add(append(append([]byte{}, anselHeader...), 0xE2, '\n', 0xE1, '\r', '\n', 0xE3))
	f.Add(append(append([]byte{}, anselHeader...), 0xE2, 'e', 0xFC, 'x'))
	f.Add([]byte("\xef\xbb\xbf0 HEAD\n1 CHAR UTF-8\n0 TRLR\n"))
	f.Add([]byte("\xff\xfe0\x00 \x00H\x00E\x00A\x00D\x00\n\x00"))
	f.Add([]byte("0 HEAD\n1 CHAR UTF-8\n1 NOTE Caf\xE9\n\xF0\x9F"))
	f.Add([]byte(""))

	f.Fuzz(func(t *testing.T, data []byte) {
		r := NewReader(bytes.NewReader(data))
		// Each input byte expands to at most one rune of UTF-8 (four bytes),
		// plus a few bytes of pending marks flushed at end of stream, so
		// output that outgrows that is a buffering bug, not a decoding one.
		n, _ := io.Copy(io.Discard, io.LimitReader(r, int64(4*len(data))+64))
		if n > int64(4*len(data))+16 {
			t.Fatalf("decoded %d bytes from %d input bytes", n, len(data))
		}

		// With OnInvalidUTF8 set, invalid UTF-8 never fails the read and
		// never reaches the output (#710). Other failures, such as an
		// invalid ANSEL byte, still can.
		reported := 0
		lr := NewReaderWithOptions(bytes.NewReader(data), ReaderOptions{
			OnInvalidUTF8: func(*ErrInvalidUTF8) { reported++ },
		})
		out, err := io.ReadAll(io.LimitReader(lr, int64(4*len(data))+64))
		var utf8Err *ErrInvalidUTF8
		if errors.As(err, &utf8Err) {
			t.Fatalf("lenient read failed with %v", err)
		}
		if !utf8.Valid(out) {
			t.Fatalf("lenient output is not valid UTF-8: %q", out)
		}
		if reported > len(data) {
			t.Fatalf("reported %d invalid bytes from %d input bytes", reported, len(data))
		}
	})
}
