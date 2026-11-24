package instructions

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ep0ll/dockerfile-parser/packages/go/parser"
)

// ParseHealthcheck parses the HEALTHCHECK header and its CMD payload. It
// retains the command form but leaves duration/range validation to consumers.
func ParseHealthcheck(ctx *parser.Context, input parser.InstructionInput) (*parser.Instruction, error) {
	instruction, words, err := parseBase(ctx, input, parser.FormRaw)
	if err != nil {
		return nil, err
	}
	if len(words) == 0 {
		return instruction, nil
	}

	switch strings.ToUpper(words[0]) {
	case "NONE":
		instruction.Data = "NONE"
		instruction.Args = append([]string(nil), words[1:]...)
		return instruction, nil
	case "CMD":
		instruction.Data = "CMD"
		return parseHealthcheckCommand(instruction, words[1:])
	default:
		instruction.Command = strings.Join(words, " ")
		instruction.Args = append([]string(nil), words...)
		return instruction, nil
	}
}

func parseHealthcheckCommand(instruction *parser.Instruction, words []string) (*parser.Instruction, error) {
	command := strings.Join(words, " ")
	trimmed := strings.TrimSpace(command)
	if strings.HasPrefix(trimmed, "[") {
		var values []string
		if err := json.Unmarshal([]byte(trimmed), &values); err != nil {
			return nil, fmt.Errorf("invalid HEALTHCHECK CMD JSON: %w", err)
		}
		instruction.Form = parser.FormJSON
		instruction.JSONArgs = values
		instruction.Args = append([]string(nil), values...)
		return instruction, nil
	}

	instruction.Form = parser.FormShell
	instruction.Command = command
	instruction.Args = append([]string(nil), words...)
	return instruction, nil
}

func healthcheckFlagOptions() parser.Option {
	value := parser.FlagRequiredValue
	return parser.WithFlagSpecs("", "HEALTHCHECK",
		parser.FlagSpec{Name: "interval", ValueMode: value},
		parser.FlagSpec{Name: "timeout", ValueMode: value},
		parser.FlagSpec{Name: "start-period", ValueMode: value},
		parser.FlagSpec{Name: "start-interval", ValueMode: value},
		parser.FlagSpec{Name: "retries", ValueMode: value},
	)
}
