package instructions

import "github.com/ep0ll/dockerfile-parser/packages/go/parser"

func ParseExpose(ctx *parser.Context, input parser.InstructionInput) (*parser.Instruction, error) {
	instruction, _, err := parseBase(ctx, input, parser.FormWords)
	return instruction, err
}
