package instructions

import "github.com/ep0ll/dockerfile-parser/packages/go/parser"

func ParseUser(ctx *parser.Context, input parser.InstructionInput) (*parser.Instruction, error) {
	instruction, args, err := parseBase(ctx, input, parser.FormRaw)
	if err != nil {
		return nil, err
	}
	instruction.Command = input.RawArgs
	instruction.Args = args
	return instruction, nil
}
