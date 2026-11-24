package instructions

import "github.com/ep0ll/dockerfile-parser/packages/go/parser"

func ParseRun(ctx *parser.Context, input parser.InstructionInput) (*parser.Instruction, error) {
	return parseJSONOrShell(ctx, input)
}

func runFlagOptions() parser.Option {
	value := parser.FlagRequiredValue
	return parser.WithFlagSpecs("", "RUN",
		parser.FlagSpec{Name: "mount", ValueMode: value, Repeatable: true},
		parser.FlagSpec{Name: "network", ValueMode: value},
		parser.FlagSpec{Name: "security", ValueMode: value},
		parser.FlagSpec{Name: "device", ValueMode: value, Repeatable: true},
	)
}
