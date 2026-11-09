package charset

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// Issue #716: a run of combining marks with no base character must not be
// buffered without bound.

func TestAnselReader_CombiningRunIsFlushedInGroups(t *testing.T) {
	const acute = 0xE2 // ANSEL acute accent -> U+0301
	run := bytes.Repeat([]byte{acute}, 2*MaxPendingCombining+8)
	input := append(append([]byte{}, run...), 'a')

	got, err := io.ReadAll(newAnselReader(bytes.NewReader(input)))
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	// The first two groups of MaxPendingCombining marks are written out as
	// they stand, before the base; the remaining 8 are reordered after it.
	want := strings.Repeat("́", 2*MaxPendingCombining) + "a" + strings.Repeat("́", 8)
	if string(got) != want {
		t.Errorf("got %q (%d runes), want %q", got, len([]rune(string(got))), want)
	}
}

func TestAnselReader_ShortRunStillReordered(t *testing.T) {
	// Up to the bound, behaviour is unchanged: every mark follows its base.
	input := append(bytes.Repeat([]byte{0xE2}, MaxPendingCombining), 'e')
	got, err := io.ReadAll(newAnselReader(bytes.NewReader(input)))
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if want := "e" + strings.Repeat("́", MaxPendingCombining); string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// combiningRun is an io.Reader yielding n copies of one byte without holding
// them in memory, so the test can stream far more than it allocates.
type combiningRun struct {
	b byte
	n int
}

func (r *combiningRun) Read(p []byte) (int, error) {
	if r.n == 0 {
		return 0, io.EOF
	}
	n := min(len(p), r.n)
	for i := range p[:n] {
		p[i] = r.b
	}
	r.n -= n
	return n, nil
}

// TestAnselReader_PendingStaysBounded streams a long run through the reader
// and checks the pending buffer never grows past the bound. The output is
// consumed in small reads so the output buffer stays small too.
func TestAnselReader_PendingStaysBounded(t *testing.T) {
	const marks = 1 << 20
	r := newAnselReader(&combiningRun{b: 0xE2, n: marks}).(*anselReader)
	buf := make([]byte, 4096)
	var total int
	for {
		n, err := r.Read(buf)
		total += n
		if cap(r.pending) > 2*MaxPendingCombining {
			t.Fatalf("pending buffer grew to cap %d", cap(r.pending))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
	}
	if want := marks * 2; total != want { // U+0301 is two bytes of UTF-8
		t.Errorf("read %d bytes, want %d", total, want)
	}
}
