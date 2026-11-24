package lexer

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestReaderPreservesFinalLineWithoutNewline(t *testing.T) {
	reader := NewReader(strings.NewReader("first\nlast"))
	first, err := reader.Next()
	if err != nil {
		t.Fatal(err)
	}
	if first.Text != "first" || first.Number != 1 {
		t.Fatalf("first line = %#v", first)
	}
	last, err := reader.Next()
	if err != nil {
		t.Fatal(err)
	}
	if last.Text != "last" || last.Number != 2 {
		t.Fatalf("last line = %#v", last)
	}
	if _, err := reader.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("final read error = %v, want EOF", err)
	}
}

func TestReaderDoesNotImposeScannerTokenLimit(t *testing.T) {
	want := strings.Repeat("x", 2<<20)
	reader := NewReader(strings.NewReader(want + "\n"))
	line, err := reader.Next()
	if err != nil {
		t.Fatal(err)
	}
	if line.Text != want {
		t.Fatalf("line length = %d, want %d", len(line.Text), len(want))
	}
}

func TestReaderStripsCRLF(t *testing.T) {
	reader := NewReader(strings.NewReader("first\r\nsecond\r\n"))
	for index, want := range []string{"first", "second"} {
		line, err := reader.Next()
		if err != nil {
			t.Fatal(err)
		}
		if line.Text != want || line.Number != index+1 {
			t.Fatalf("line = %#v, want text=%q number=%d", line, want, index+1)
		}
	}
}

func TestStripBOMOnlyAtSourceStart(t *testing.T) {
	input := "\uFEFFFROM alpine"
	if got := StripBOM(input); got != "FROM alpine" {
		t.Fatalf("StripBOM() = %q", got)
	}
	if got := StripBOM("RUN \uFEFFecho"); got != "RUN \uFEFFecho" {
		t.Fatalf("StripBOM removed a non-leading BOM: %q", got)
	}
}
