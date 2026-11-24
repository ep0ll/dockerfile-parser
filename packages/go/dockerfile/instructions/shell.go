package instructions

import "github.com/ep0ll/dockerfile-parser/packages/go/parser"

func ParseShell(ctx *parser.Context, input parser.InstructionInput) (*parser.Instruction, error) {
	return parseJSONOrWords(ctx, input)
}
