package parser

import (
	"strings"
	"testing"
)

func TestParseFlagsAndScopedReplacement(t *testing.T) {
	p, err := New(strings.NewReader(""),
		WithFlagSpecs("", "RUN", FlagSpec{Name: "mount", ValueMode: FlagRequiredValue, Repeatable: true}),
		WithFlagSpecs("onbuild", "RUN", FlagSpec{Name: "cache", ValueMode: FlagOptionalValue}),
		WithUnknownFlagPolicy(UnknownFlagError),
	)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		scope, raw string
		wantName   string
	}{
		{scope: "", raw: "--mount=type=cache echo hi", wantName: "mount"},
		{scope: "onbuild", raw: "--cache=true echo hi", wantName: "cache"},
	}
	for _, tt := range tests {
		flags, args, err := p.parseFlags(tt.scope, "RUN", tt.raw)
		if err != nil {
			t.Fatal(err)
		}
		if len(flags) != 1 || flags[0].Name != tt.wantName {
			t.Fatalf("flags = %#v", flags)
		}
		if len(args) != 2 || args[0] != "echo" {
			t.Fatalf("args = %#v", args)
		}
	}
	if _, _, err := p.parseFlags("onbuild", "RUN", "--mount=foo echo hi"); err == nil {
		t.Fatal("expected --mount to be rejected in the ONBUILD RUN scope")
	}
}

func TestSplitWordsPreservesQuotedValues(t *testing.T) {
	got, err := SplitWords("--mount=\\"type=cache,target=/root/.cache\\" echo \\"hello world\\"", '\\\\')
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0] != "--mount=\\"type=cache,target=/root/.cache\\"" || got[2] != "\\"hello world\\"" {
		t.Fatalf("unexpected words: %#v", got)
	}
}

func TestOptionalInlineFlagDoesNotConsumeFollowingArgument(t *testing.T) {
	flags, args, err := ParseFlagArguments("", "COPY", "--link source target",
		[]FlagSpec{{Name: "link", ValueMode: FlagOptionalInlineValue}}, UnknownFlagError, '\\\\')
	if err != nil {
		t.Fatal(err)
	}
	if !flags.Has("link") || len(args) != 2 || args[0] != "source" {
		t.Fatalf("flags=%#v args=%#v", flags, args)
	}
	flags, args, err = ParseFlagArguments("", "COPY", "--link=false source target",
		[]FlagSpec{{Name: "link", ValueMode: FlagOptionalInlineValue}}, UnknownFlagError, '\\\\')
	if err != nil {
		t.Fatal(err)
	}
	flag, _ := flags.First("link")
	if flag.Value != "false" || !flag.HasValue || len(args) != 2 {
		t.Fatalf("flag=%#v args=%#v", flag, args)
	}
}

func TestReplaceableCommentAndInstructionSyntax(t *testing.T) {
	p, err := New(strings.NewReader("// notes\nRUN => echo ready\n"),
		WithCommentPrefix("//"),
		WithInstructionSplitter(func(line string) (string, string, bool) {
			name, args, ok := strings.Cut(line, "=>")
			return strings.TrimSpace(name), strings.TrimSpace(args), ok
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	instruction, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if instruction.Name != "RUN" || instruction.RawArgs != "echo ready" || len(instruction.Comments) != 1 {
		t.Fatalf("unexpected custom syntax result: %#v", instruction)
	}
}

func TestReplaceableFlagParser(t *testing.T) {
	p, err := New(strings.NewReader("THING token=value rest\n"),
		WithInstructionHandler("THING", func(ctx *Context, input InstructionInput) (*Instruction, error) {
			flags, args, err := ctx.ParseFlags(input)
			return &Instruction{Flags: flags, Args: args}, err
		}),
		WithFlagParser(func(scope, instruction, raw string, specs []FlagSpec, policy UnknownFlagPolicy, escape rune) (FlagSet, []string, error) {
			parts := strings.Fields(raw)
			return FlagSet{{Name: "custom", Value: parts[0], HasValue: true}}, parts[1:], nil
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	instruction, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	flag, ok := instruction.Flags.First("custom")
	if !ok || flag.Value != "token=value" || len(instruction.Args) != 1 || instruction.Args[0] != "rest" {
		t.Fatalf("unexpected custom flag parse: %#v", instruction)
	}
}
