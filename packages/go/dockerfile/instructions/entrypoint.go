package instructions

import "github.com/ep0ll/dockerfile-parser/packages/go/parser"

func ParseEntrypoint(ctx *parser.Context, input parser.InstructionInput) (*parser.Instruction, error) {
	return parseJSONOrShell(ctx, input)
}
