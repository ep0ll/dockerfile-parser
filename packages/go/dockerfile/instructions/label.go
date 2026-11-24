package instructions

import "github.com/ep0ll/dockerfile-parser/packages/go/parser"

func ParseLabel(ctx *parser.Context, input parser.InstructionInput) (*parser.Instruction, error) {
	instruction, words, err := parseBase(ctx, input, parser.FormKeyValues)
	if err != nil {
		return nil, err
	}
	if unsetEnabled(instruction.Flags) {
		instruction.Pairs = parsePairs(words)
	} else {
		instruction.Pairs = parseLegacyKeyValues(words)
	}
	return instruction, nil
}
