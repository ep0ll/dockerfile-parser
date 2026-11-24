package parser

import (
	"errors"
	"strings"
	"testing"
)

func TestCustomDirectiveAndInstructionDialect(t *testing.T) {
	source := "! use ~\nTASK => perform ~\n  work\n"
	p, err := New(strings.NewReader(source),
		WithCommentPrefix("!"),
		WithDirectiveParser(func(comment string, position Position) (Directive, bool) {
			name, value, ok := strings.Cut(comment, " ")
			if !ok || name != "use" {
				return Directive{}, false
			}
			return Directive{Name: "DIALECT", Value: value, Position: position}, true
		}),
		WithDirectiveNameNormalizer(strings.ToLower),
		WithDirectiveHandler("dialect", func(ctx *DirectiveContext, directive Directive) error {
			if len([]rune(directive.Value)) != 1 {
				return errors.New("expected one escape rune")
			}
			if err := ctx.SetEscapeToken([]rune(directive.Value)[0]); err != nil {
				return err
			}
			ctx.Set("dialect", directive.Value)
			return nil
		}),
		WithInstructionNameNormalizer(strings.ToLower),
		WithInstructionSplitter(func(line string) (string, string, bool) {
			name, args, ok := strings.Cut(line, "=>")
			return strings.TrimSpace(name), strings.TrimSpace(args), ok
		}),
		WithInstructionHandler("task", func(_ *Context, input InstructionInput) (*Instruction, error) {
			return &Instruction{Command: input.RawArgs, Form: FormRaw}, nil
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	instruction, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if instruction.Name != "TASK" || instruction.Command != "perform work" {
		t.Fatalf("unexpected custom instruction: %#v", instruction)
	}
	if p.EscapeToken() != '~' || p.Metadata()["dialect"] != "~" {
		t.Fatalf("custom directive state not applied: escape=%q metadata=%#v", p.EscapeToken(), p.Metadata())
	}
}

func TestEmptyScopedFlagSchemaDisablesParentFlags(t *testing.T) {
	p, err := New(strings.NewReader(""),
		WithFlagSpecs("", "RUN", FlagSpec{Name: "mount", ValueMode: FlagRequiredValue}),
		WithFlagSpecs("onbuild", "RUN"),
		WithUnknownFlagPolicy(UnknownFlagError),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := p.parseFlags("", "RUN", "--mount=type=cache echo"); err != nil {
		t.Fatalf("global schema should allow mount: %v", err)
	}
	if _, _, err := p.parseFlags("onbuild", "RUN", "--mount=type=cache echo"); err == nil {
		t.Fatal("empty scoped schema should disable inherited flags")
	}
}


func TestBodyCollectorIsIndependentFromDockerfileDialect(t *testing.T) {
	source := "SCRIPT payload\nfirst line\nsecond line\nEND\nAFTER value\n"
	p, err := New(strings.NewReader(source),
		WithBodyCollector(func(ctx *BodyContext, input InstructionInput) ([]Body, error) {
			if input.Name != "SCRIPT" {
				return nil, nil
			}
			var content strings.Builder
			for {
				line, err := ctx.NextLine()
				if err != nil {
					return nil, err
				}
				if line.Text == "END" {
					break
				}
				content.WriteString(line.Text)
				content.WriteByte('\n')
			}
			return []Body{{
				Kind: "block",
				Name: "END",
				Content: content.String(),
				Position: Position{StartLine: 2, EndLine: ctx.EndLine()},
			}}, nil
		}),
		WithInstructionHandler("SCRIPT", func(_ *Context, input InstructionInput) (*Instruction, error) {
			return &Instruction{Form: FormRaw, Command: input.RawArgs}, nil
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	script, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if len(script.Bodies) != 1 || script.Bodies[0].Kind != "block" ||
		script.Bodies[0].Content != "first line\nsecond line\n" {
		t.Fatalf("custom body = %#v", script.Bodies)
	}
	if script.Position.StartLine != 1 || script.Position.EndLine != 4 {
		t.Fatalf("script position = %#v", script.Position)
	}

	next, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if next.Name != "AFTER" || next.Position.StartLine != 5 {
		t.Fatalf("next statement = %#v", next)
	}
}
