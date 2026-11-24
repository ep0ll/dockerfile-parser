package dockerfile_test

import (
	"io"
	"strings"
	"testing"

	"github.com/ep0ll/dockerfile-parser/packages/go/dockerfile"
	"github.com/ep0ll/dockerfile-parser/packages/go/parser"
)

func TestStreamsInstructionsWithoutDocumentValidation(t *testing.T) {
	p, err := dockerfile.NewParser(strings.NewReader("ONBUILD RUN echo early\nFROM alpine\n"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if first.Name != "ONBUILD" || len(first.Children) != 1 || first.Children[0].Name != "RUN" {
		t.Fatalf("unexpected first instruction: %#v", first)
	}
	second, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if second.Name != "FROM" {
		t.Fatalf("second instruction = %q", second.Name)
	}
	if _, err := p.Next(); err != io.EOF {
		t.Fatalf("Next() error = %v, want EOF", err)
	}
}

func TestEscapeDirectiveChangesContinuationToken(t *testing.T) {
	escape := string(rune(96))
	source := "# escape=" + escape + "\nFROM alpine\nRUN echo hello " + escape + "\n  world\n"
	p, err := dockerfile.NewParser(strings.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Next(); err != nil {
		t.Fatal(err)
	}
	run, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if run.Name != "RUN" || run.Command != "echo hello world" {
		t.Fatalf("RUN command = %q", run.Command)
	}
	if p.EscapeToken() != rune(96) {
		t.Fatalf("escape token = %q", p.EscapeToken())
	}
	if got := p.Metadata()["dockerfile.escape"]; got != escape {
		t.Fatalf("escape metadata = %#v", got)
	}
}

func TestIndependentCommentAndDirectiveHandlers(t *testing.T) {
	var comments []parser.Comment
	p, err := dockerfile.NewParser(strings.NewReader("# syntax=custom/frontend\n# a human comment\nFROM alpine\n"),
		parser.WithCommentHandler(func(comment parser.Comment) error { comments = append(comments, comment); return nil }),
	)
	if err != nil {
		t.Fatal(err)
	}
	instruction, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if instruction.Name != "FROM" || len(instruction.Comments) != 1 || instruction.Comments[0] != "a human comment" {
		t.Fatalf("unexpected comments on instruction: %#v", instruction.Comments)
	}
	if len(comments) != 1 || comments[0].Text != "a human comment" {
		t.Fatalf("comment handler saw %#v", comments)
	}
	if got := p.Metadata()["dockerfile.syntax"]; got != "custom/frontend" {
		t.Fatalf("syntax metadata = %#v", got)
	}
}

func TestCustomInstructionAndFlags(t *testing.T) {
	type payload struct{ Token string }
	p, err := dockerfile.NewParser(strings.NewReader("FOO --token=abc value\n"),
		parser.WithInstructionHandler("FOO", func(ctx *parser.Context, input parser.InstructionInput) (*parser.Instruction, error) {
			flags, args, err := ctx.ParseFlags(input)
			if err != nil {
				return nil, err
			}
			flag, _ := flags.First("token")
			return &parser.Instruction{Flags: flags, Args: args, Data: payload{Token: flag.Value}}, nil
		}),
		parser.WithFlagSpecs("", "FOO", parser.FlagSpec{Name: "token", ValueMode: parser.FlagRequiredValue}),
	)
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "FOO" || got.Data.(payload).Token != "abc" || len(got.Args) != 1 || got.Args[0] != "value" {
		t.Fatalf("unexpected custom instruction: %#v", got)
	}
}

func TestOnbuildUsesSameHandlerWithScopedFlags(t *testing.T) {
	source := "RUN --mount=type=cache echo top\nONBUILD RUN --cache=shared echo nested\n"
	p, err := dockerfile.NewParser(strings.NewReader(source),
		parser.WithFlagSpecs("onbuild", "RUN", parser.FlagSpec{Name: "cache", ValueMode: parser.FlagRequiredValue}),
		parser.WithUnknownFlagPolicy(parser.UnknownFlagError),
	)
	if err != nil {
		t.Fatal(err)
	}
	top, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if !top.Flags.Has("mount") {
		t.Fatalf("top-level RUN flags = %#v", top.Flags)
	}
	onbuild, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	child := onbuild.Children[0]
	if child.Name != "RUN" || !child.Flags.Has("cache") || child.Flags.Has("mount") {
		t.Fatalf("nested RUN flags = %#v", child.Flags)
	}
}

func TestAllDefaultInstructionHandlersAreRegistered(t *testing.T) {
	names := []string{"ADD", "ARG", "CMD", "COPY", "ENTRYPOINT", "ENV", "EXPOSE", "FROM", "HEALTHCHECK", "LABEL", "MAINTAINER", "ONBUILD", "RUN", "SHELL", "STOPSIGNAL", "USER", "VOLUME", "WORKDIR"}
	var source strings.Builder
	for _, name := range names {
		source.WriteString(name + " value\n")
	}
	p, err := dockerfile.NewParser(strings.NewReader(source.String()))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		instruction, err := p.Next()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if instruction.Name != name || instruction.Unknown {
			t.Fatalf("%s parsed as %#v", name, instruction)
		}
	}
}

func TestInstructionMiddlewareWrapsOnlySelectedHandler(t *testing.T) {
	calls := 0
	p, err := dockerfile.NewParser(strings.NewReader("RUN echo ok\nFROM alpine\n"),
		parser.WithInstructionMiddleware("RUN", func(next parser.InstructionHandler) parser.InstructionHandler {
			return func(ctx *parser.Context, input parser.InstructionInput) (*parser.Instruction, error) {
				calls++
				return next(ctx, input)
			}
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Next(); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Next(); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("RUN middleware called %d times", calls)
	}
}

func TestUnknownInstructionIsPreservedByDefault(t *testing.T) {
	p, err := dockerfile.NewParser(strings.NewReader("CUSTOM_THING --vendor=on payload\n"))
	if err != nil {
		t.Fatal(err)
	}
	instruction, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if !instruction.Unknown || instruction.Name != "CUSTOM_THING" || !instruction.Flags.Has("vendor") {
		t.Fatalf("unknown instruction not preserved: %#v", instruction)
	}
}

func TestBlankLinesDoNotCloseDirectivePreamble(t *testing.T) {
	escape := string(rune(96))
	p, err := dockerfile.NewParser(strings.NewReader("\n\n# escape=" + escape + "\nFROM alpine\nRUN echo a " + escape + "\n  b\n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Next(); err != nil {
		t.Fatal(err)
	}
	run, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if run.Command != "echo a b" {
		t.Fatalf("RUN command = %q", run.Command)
	}
}

func TestDuplicateDockerfileDirectivesAreRejectedByTheirHandlers(t *testing.T) {
	escape := string(rune(96))
	p, err := dockerfile.NewParser(strings.NewReader("# escape=\\\n# escape=" + escape + "\nFROM alpine\n"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Next()
	if err == nil || !strings.Contains(err.Error(), "duplicate escape directive") {
		t.Fatalf("duplicate directive error = %v", err)
	}
}

func TestContinuationErrorsIncludeSourcePosition(t *testing.T) {
	p, err := dockerfile.NewParser(strings.NewReader("RUN echo ok \\\n"), parser.WithContinuationParser(func(line string, escape rune) (string, bool, error) {
		return "", false, io.ErrUnexpectedEOF
	}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Next()
	if err == nil || !strings.Contains(err.Error(), "parse error on line 1") {
		t.Fatalf("continuation error = %v", err)
	}
}


func TestCommentsRemainAttachedAcrossBlankLines(t *testing.T) {
	p, err := dockerfile.NewParser(strings.NewReader("# useful documentation\n\n\nFROM alpine\n"))
	if err != nil {
		t.Fatal(err)
	}
	instruction, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if len(instruction.Comments) != 1 || instruction.Comments[0] != "useful documentation" {
		t.Fatalf("comments were not retained: %#v", instruction.Comments)
	}
}

func TestEmptyCommentResetsPendingComments(t *testing.T) {
	p, err := dockerfile.NewParser(strings.NewReader("# stale comment\n#\nFROM alpine\n"))
	if err != nil {
		t.Fatal(err)
	}
	instruction, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if len(instruction.Comments) != 0 {
		t.Fatalf("empty comment should reset pending comments: %#v", instruction.Comments)
	}
}


func TestLabelParsesLegacyAndKeyValueForms(t *testing.T) {
	source := "LABEL maintainer \"Example Org\"\n" +
		"LABEL org.example.name=Demo org.example.version=1\n" +
		"LABEL --unset org.example.name org.example.version\n"
	p, err := dockerfile.NewParser(strings.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}

	legacy, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if len(legacy.Pairs) != 1 || legacy.Pairs[0].Key != "maintainer" ||
		legacy.Pairs[0].Value != "\"Example Org\"" || !legacy.Pairs[0].HasValue {
		t.Fatalf("legacy LABEL pairs = %#v", legacy.Pairs)
	}

	keyValues, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if len(keyValues.Pairs) != 2 || keyValues.Pairs[0].Key != "org.example.name" ||
		keyValues.Pairs[0].Value != "Demo" || keyValues.Pairs[1].Key != "org.example.version" {
		t.Fatalf("key-value LABEL pairs = %#v", keyValues.Pairs)
	}

	unset, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if !unset.Flags.Has("unset") || len(unset.Pairs) != 2 ||
		unset.Pairs[0].HasValue || unset.Pairs[1].HasValue {
		t.Fatalf("LABEL --unset pairs = %#v", unset.Pairs)
	}
}


func TestInstructionFlagSchemasAreDeclaredForEachBuiltInCommand(t *testing.T) {
	source := "ADD --chown=1000:1000 a /b\n" +
		"COPY --from=stage --link a /b\n" +
		"FROM --platform=linux/amd64 alpine\n" +
		"RUN --network=host echo hi\n" +
		"HEALTHCHECK --interval=30s CMD true\n" +
		"ENV --unset FOO\n" +
		"LABEL --unset example.label\n" +
		"VOLUME --unset /cache\n"
	p, err := dockerfile.NewParser(strings.NewReader(source),
		parser.WithUnknownFlagPolicy(parser.UnknownFlagError),
	)
	if err != nil {
		t.Fatal(err)
	}

	expected := []struct {
		name string
		flag string
	}{
		{name: "ADD", flag: "chown"},
		{name: "COPY", flag: "from"},
		{name: "FROM", flag: "platform"},
		{name: "RUN", flag: "network"},
		{name: "HEALTHCHECK", flag: "interval"},
		{name: "ENV", flag: "unset"},
		{name: "LABEL", flag: "unset"},
		{name: "VOLUME", flag: "unset"},
	}
	for _, item := range expected {
		instruction, err := p.Next()
		if err != nil {
			t.Fatalf("%s: %v", item.name, err)
		}
		if instruction.Name != item.name || !instruction.Flags.Has(item.flag) {
			t.Fatalf("%s flags = %#v", item.name, instruction.Flags)
		}
	}
}


func TestHealthcheckSeparatesModeAndCommandForm(t *testing.T) {
	source := "HEALTHCHECK --interval=30s CMD [\\"curl\\", \\"-f\\", \\"http://localhost\\"]\\n" +
		"HEALTHCHECK CMD curl -f http://localhost\\n" +
		"HEALTHCHECK NONE\\n"
	p, err := dockerfile.NewParser(strings.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}

	jsonCommand, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if jsonCommand.Data != "CMD" || jsonCommand.Form != parser.FormJSON ||
		len(jsonCommand.JSONArgs) != 3 || jsonCommand.JSONArgs[0] != "curl" ||
		!jsonCommand.Flags.Has("interval") {
		t.Fatalf("JSON HEALTHCHECK = %#v", jsonCommand)
	}

	shellCommand, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if shellCommand.Data != "CMD" || shellCommand.Form != parser.FormShell ||
		shellCommand.Command != "curl -f http://localhost" {
		t.Fatalf("shell HEALTHCHECK = %#v", shellCommand)
	}

	none, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if none.Data != "NONE" {
		t.Fatalf("HEALTHCHECK NONE = %#v", none)
	}
}
