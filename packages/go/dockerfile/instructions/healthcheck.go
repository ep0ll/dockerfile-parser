package instructions

import (
	"strings"

	"github.com/ep0ll/dockerfile-parser/packages/go/parser"
)

func ParseHealthcheck(ctx *parser.Context, input parser.InstructionInput) (*parser.Instruction, error) {
	instruction, words, err := parseBase(ctx, input, parser.FormRaw)
	if err != nil {
		return nil, err
	}
	instruction.Command = strings.Join(words, " ")
	if len(words) > 0 && strings.EqualFold(words[0], "NONE") {
		instruction.Data = "NONE"
	} else if len(words) > 0 && strings.EqualFold(words[0], "CMD") {
		instruction.Data = "CMD"
		instruction.Args = append([]string(nil), words[1:]...)
	}
	return instruction, nil
}
