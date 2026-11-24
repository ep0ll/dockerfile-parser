package instructions

import "github.com/ep0ll/dockerfile-parser/packages/go/parser"

func ParseAdd(ctx *parser.Context, input parser.InstructionInput) (*parser.Instruction, error) {
	return parseJSONOrWords(ctx, input)
}

func addFlagOptions() parser.Option {
	value := parser.FlagRequiredValue
	optional := parser.FlagOptionalInlineValue
	return parser.WithFlagSpecs("", "ADD",
		parser.FlagSpec{Name: "chown", ValueMode: value},
		parser.FlagSpec{Name: "chmod", ValueMode: value},
		parser.FlagSpec{Name: "link", ValueMode: optional},
		parser.FlagSpec{Name: "exclude", ValueMode: value, Repeatable: true},
		parser.FlagSpec{Name: "checksum", ValueMode: value},
		parser.FlagSpec{Name: "keep-git-dir", ValueMode: optional},
		parser.FlagSpec{Name: "unpack", ValueMode: optional},
	)
}
