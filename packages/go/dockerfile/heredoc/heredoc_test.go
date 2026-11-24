package heredoc

import "testing"

func TestParseWord(t *testing.T) {
	quote := string(rune(34))
	tests := []struct {
		name string
		word string
		want Spec
		ok   bool
	}{
		{name: "plain", word: "<<EOF", want: Spec{Delimiter: "EOF", Expand: true}, ok: true},
		{name: "quoted", word: "<<'EOF'", want: Spec{Delimiter: "EOF", Expand: false}, ok: true},
		{name: "double quoted", word: "<<" + quote + "END" + quote, want: Spec{Delimiter: "END", Expand: false}, ok: true},
		{name: "strip tabs", word: "<<-EOF", want: Spec{Delimiter: "EOF", Expand: true, StripTabs: true}, ok: true},
		{name: "file descriptor", word: "2<<EOF", want: Spec{Delimiter: "EOF", FileDescriptor: 2, Expand: true}, ok: true},
		{name: "not heredoc", word: "echo", ok: false},
		{name: "here string", word: "<<<EOF", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok, err := ParseWord(tt.word, rune(92))
			if err != nil {
				t.Fatal(err)
			}
			if ok != tt.ok || got != tt.want {
				t.Fatalf("ParseWord() = (%#v, %v), want (%#v, %v)", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestParseWordRejectsInvalidDeclarations(t *testing.T) {
	for _, word := range []string{"<<", "<<''", "<<-"} {
		if _, _, err := ParseWord(word, rune(92)); err == nil {
			t.Errorf("ParseWord(%q) expected an error", word)
		}
	}
}
