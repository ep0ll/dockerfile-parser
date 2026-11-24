package instructions

import "github.com/ep0ll/dockerfile-parser/packages/go/parser"

func ParseCopy(ctx *parser.Context, input parser.InstructionInput) (*parser.Instruction, error) {
	return parseJSONOrWords(ctx, input)
}

func copyFlagOptions() parser.Option {
	value := parser.FlagRequiredValue
	optional := parser.FlagOptionalInlineValue
	return parser.WithFlagSpecs("", "COPY",
		parser.FlagSpec{Name: "from", ValueMode: value},
		parser.FlagSpec{Name: "chown", ValueMode: value},
		parser.FlagSpec{Name: "chmod", ValueMode: value},
		parser.FlagSpec{Name: "link", ValueMode: optional},
		parser.FlagSpec{Name: "parents", ValueMode: optional},
		parser.FlagSpec{Name: "exclude", ValueMode: value, Repeatable: true},
	)
}
