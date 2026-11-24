package parser

import (
	"strings"
	"testing"
)

func TestNameNormalizersRekeyHandlersDirectivesAndSchemas(t *testing.T) {
	var directiveSeen string
	upperHandler := func(_ *Context, input InstructionInput) (*Instruction, error) {
		flags, args, err := ParseFlagArguments(input.Scope, input.Name, input.RawArgs,
			[]FlagSpec{{Name: "Token", ValueMode: FlagRequiredValue}}, UnknownFlagError, '\\')
		return &Instruction{Flags: flags, Args: args, Data: "custom-handler"}, err
	}

	p, err := New(strings.NewReader("# MODE=enabled\nthing --Token=value payload\n"),
		WithDirectiveHandler("MODE", func(ctx *DirectiveContext, directive Directive) error {
			directiveSeen = directive.Value
			ctx.Set("mode", directive.Value)
			return nil
		}),
		WithInstructionHandler("THING", upperHandler),
		WithFlagSpecs("", "THING", FlagSpec{Name: "Token", ValueMode: FlagRequiredValue}),
		WithDirectiveNameNormalizer(strings.ToLower),
		WithInstructionNameNormalizer(strings.ToLower),
	)
	if err != nil {
		t.Fatal(err)
	}

	instruction, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if directiveSeen != "enabled" || p.Metadata()["mode"] != "enabled" {
		t.Fatalf("directive handler did not run: seen=%q metadata=%#v", directiveSeen, p.Metadata())
	}
	if instruction.Data != "custom-handler" || instruction.Name != "thing" {
		t.Fatalf("unexpected case-normalized instruction: %#v", instruction)
	}
	flag, ok := instruction.Flags.First("Token")
	if !ok || flag.Value != "value" || len(instruction.Args) != 1 || instruction.Args[0] != "payload" {
		t.Fatalf("custom flag schema was not re-keyed: %#v", instruction)
	}
}

func TestNameNormalizerRejectsRegistryCollisions(t *testing.T) {
	first := func(*Context, InstructionInput) (*Instruction, error) { return &Instruction{}, nil }
	second := func(*Context, InstructionInput) (*Instruction, error) { return &Instruction{}, nil }
	_, err := New(strings.NewReader(""),
		WithInstructionNameNormalizer(func(name string) string { return name }),
		WithInstructionHandler("Thing", first),
		WithInstructionHandler("THING", second),
		WithInstructionNameNormalizer(strings.ToLower),
	)
	if err == nil || !strings.Contains(err.Error(), "duplicate key") {
		t.Fatalf("expected a normalizer collision error, got %v", err)
	}
}
