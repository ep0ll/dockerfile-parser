package instructions

import "github.com/ep0ll/dockerfile-parser/packages/go/parser"

// ParseOnbuild delegates its single nested instruction to the same registry as
// top-level statements. The onbuild scope allows a different flag schema.
func ParseOnbuild(ctx *parser.Context, input parser.InstructionInput) (*parser.Instruction, error) {
	child, err := ctx.ParseNested(input.RawArgs, "onbuild", input.Position)
	if err != nil {
		return nil, err
	}
	instruction := &parser.Instruction{Form: parser.FormRaw, Command: input.RawArgs}
	if child != nil {
		instruction.Children = []*parser.Instruction{child}
	}
	return instruction, nil
}
