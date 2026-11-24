package instructions

import "github.com/ep0ll/dockerfile-parser/packages/go/parser"

func ParseVolume(ctx *parser.Context, input parser.InstructionInput) (*parser.Instruction, error) {
	return parseJSONOrWords(ctx, input)
}

func volumeFlagOptions() parser.Option {
	return parser.WithFlagSpecs("", "VOLUME",
		parser.FlagSpec{Name: "unset", ValueMode: parser.FlagOptionalInlineValue},
	)
}
