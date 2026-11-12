package charset

import (
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

// ansiInput is a CHAR ANSI header followed by body, every line LF-terminated.
func ansiInput(char, body string) string {
	return "0 HEAD\n1 CHAR " + char + "\n0 @I1@ INDI\n" + body + "0 TRLR\n"
}

func readAllWithOptions(t *testing.T, input string, opts ReaderOptions) string {
	t.Helper()
	data, err := io.ReadAll(NewReaderWithOptions(strings.NewReader(input), opts))
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	return string(data)
}

// TestReaderOptions_ANSI checks that ReaderOptions.ANSI names the code page of
// CHAR ANSI, and of nothing else (#759). The bytes are those Central European
// letters whose Windows-1250 byte means something else in Windows-1252.
func TestReaderOptions_ANSI(t *testing.T) {
	body := "1 NAME Jan\xE8 \xC8a\xE6 \xF0uro /Logari\xE8/\n"
	tests := []struct {
		name string
		char string
		ansi Encoding
		want string
	}{
		{"default ANSI is Windows-1252", "ANSI", EncodingUnknown, "1 NAME Janè Èaæ ðuro /Logariè/\n"},
		{"ANSI named Windows-1252", "ANSI", EncodingLATIN1, "1 NAME Janè Èaæ ðuro /Logariè/\n"},
		{"ANSI named Windows-1250", "ANSI", EncodingWindows1250, "1 NAME Janč Čać đuro /Logarič/\n"},
		{"lower-case ansi named Windows-1250", "ansi", EncodingWindows1250, "1 NAME Janč Čać đuro /Logarič/\n"},
		{"ANSI named CP850", "ANSI", EncodingCP850, "1 NAME JanÞ ╚aµ \u00aduro /LogariÞ/\n"},
		// Values naming no single-byte code page fall back to the default
		// rather than validating the bytes as UTF-8 or UTF-16.
		{"ANSI named UTF-8 is Windows-1252", "ANSI", EncodingUTF8, "1 NAME Janè Èaæ ðuro /Logariè/\n"},
		{"ANSI named UTF-16LE is Windows-1252", "ANSI", EncodingUTF16LE, "1 NAME Janè Èaæ ðuro /Logariè/\n"},
		{"ANSI named ANSEL is Windows-1252", "ANSI", EncodingANSEL, "1 NAME Janè Èaæ ðuro /Logariè/\n"},
		{"ANSI out of range is Windows-1252", "ANSI", Encoding(255), "1 NAME Janè Èaæ ðuro /Logariè/\n"},
		// The option is for ANSI alone: these values name their code page.
		{"WINDOWS-1252 ignores ANSI", "WINDOWS-1252", EncodingWindows1250, "1 NAME Janè Èaæ ðuro /Logariè/\n"},
		{"LATIN1 ignores ANSI", "LATIN1", EncodingWindows1250, "1 NAME Janè Èaæ ðuro /Logariè/\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readAllWithOptions(t, ansiInput(tt.char, body), ReaderOptions{ANSI: tt.ansi})
			want := ansiInput(tt.char, tt.want)
			if got != want {
				t.Errorf("decoded %q, want %q", got, want)
			}
		})
	}
}

// TestReaderOptions_OnAmbiguousANSI checks that the first byte reading
// differently in Windows-1252 and Windows-1250 is reported once, with its
// position and both readings, and that the output stays Windows-1252.
func TestReaderOptions_OnAmbiguousANSI(t *testing.T) {
	// é (0xE9), š (0x9A) and ž (0x9E) read the same in both code pages, so
	// the first report is the 0xE8 on line 5 after them.
	body := "1 NAME Ren\xE9 \x9A\x9E\r\n1 NOTE ok\r\n1 NOTE Logari\xE8 \xC8\n"
	input := ansiInput("ANSI", body)
	var got []AmbiguousANSI
	out := readAllWithOptions(t, input, ReaderOptions{OnAmbiguousANSI: func(a *AmbiguousANSI) { got = append(got, *a) }})

	want := AmbiguousANSI{Line: 6, Column: 14, Byte: 0xE8, Windows1252: 'è', Windows1250: 'č'}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("reports = %+v, want exactly [%+v]", got, want)
	}
	if wantOut := ansiInput("ANSI", "1 NAME René šž\r\n1 NOTE ok\r\n1 NOTE Logariè È\n"); out != wantOut {
		t.Errorf("decoded %q, want %q", out, wantOut)
	}
}

// TestReaderOptions_OnAmbiguousANSI_OneByteReads checks that the reported
// position survives reads that split the input, including a CRLF split
// between two reads, and that a value naming no code page still reports.
func TestReaderOptions_OnAmbiguousANSI_OneByteReads(t *testing.T) {
	input := ansiInput("ANSI", "1 NAME Ren\xE9\r\n1 NOTE ok\r\n1 NOTE Logari\xE8\n")
	var got []AmbiguousANSI
	r := NewReaderWithOptions(iotest.OneByteReader(strings.NewReader(input)), ReaderOptions{
		ANSI:            EncodingUTF8,
		OnAmbiguousANSI: func(a *AmbiguousANSI) { got = append(got, *a) },
	})
	if _, err := io.ReadAll(r); err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	want := AmbiguousANSI{Line: 6, Column: 14, Byte: 0xE8, Windows1252: 'è', Windows1250: 'č'}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("reports = %+v, want exactly [%+v]", got, want)
	}
}

// TestReaderOptions_OnAmbiguousANSI_NotCalled covers each case that must not
// report: ANSI named, bytes the code pages share, and a CHAR other than ANSI.
func TestReaderOptions_OnAmbiguousANSI_NotCalled(t *testing.T) {
	tests := []struct {
		name  string
		input string
		ansi  Encoding
	}{
		{"ANSI named Windows-1252", ansiInput("ANSI", "1 NAME Logari\xE8\n"), EncodingLATIN1},
		{"ANSI named Windows-1250", ansiInput("ANSI", "1 NAME Logari\xE8\n"), EncodingWindows1250},
		{"shared bytes only", ansiInput("ANSI", "1 NAME Ren\xE9 \x8A\x9A\x8E\x9E \xFC\n"), EncodingUnknown},
		{"WINDOWS-1252 declared", ansiInput("WINDOWS-1252", "1 NAME Logari\xE8\n"), EncodingUnknown},
		{"ISO-8859-1 declared", ansiInput("ISO-8859-1", "1 NAME Logari\xE8\n"), EncodingUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			readAllWithOptions(t, tt.input, ReaderOptions{
				ANSI:            tt.ansi,
				OnAmbiguousANSI: func(*AmbiguousANSI) { called = true },
			})
			if called {
				t.Error("OnAmbiguousANSI called, want no call")
			}
		})
	}
}

// TestNewReaderWithEncoding_Windows1250 checks the Windows-1250 table,
// including that a byte Windows-1250 leaves undefined keeps its ISO-8859-1
// code point rather than becoming U+FFFD.
func TestNewReaderWithEncoding_Windows1250(t *testing.T) {
	data, err := io.ReadAll(NewReaderWithEncoding(strings.NewReader("\xE8\xC8\xE6\xF0\x8A\x9E\xA5\x81\x98"), EncodingWindows1250))
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if want := "čČćđŠžĄ\u0081\u0098"; string(data) != want {
		t.Errorf("decoded %q, want %q", data, want)
	}
}
