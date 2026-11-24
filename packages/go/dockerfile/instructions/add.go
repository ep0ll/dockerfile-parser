package instructions

import "github.com/ep0ll/dockerfile-parser/packages/go/parser"

func ParseAdd(ctx *parser.Context, input parser.InstructionInput) (*parser.Instruction, error) {
	instruction, err := parseJSONOrWords(ctx, input)
	return instruction, err
}
