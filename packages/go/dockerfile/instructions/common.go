// Package instructions provides one independent parser handler per Dockerfile
// instruction. Handlers only parse their own instruction's lexical payload.
package instructions

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ep0ll/dockerfile-parser/packages/go/parser"
)

func parseFlagged(ctx *parser.Context, input parser.InstructionInput) (parser.FlagSet, []string, error) {
	return ctx.ParseFlags(input)
}

func parseBase(ctx *parser.Context, input parser.InstructionInput, form parser.ArgumentForm) (*parser.Instruction, []string, error) {
	flags, args, err := parseFlagged(ctx, input)
	if err != nil {
		return nil, nil, err
	}
	instruction := &parser.Instruction{Flags: flags, Form: form, Args: args}
	return instruction, args, nil
}

func parseJSONOrShell(ctx *parser.Context, input parser.InstructionInput) (*parser.Instruction, error) {
	flags, tokens, err := ctx.ParseFlags(input)
	if err != nil {
		return nil, err
	}
	rawArgs := strings.Join(tokens, " ")
	trimmed := strings.TrimSpace(rawArgs)
	instruction := &parser.Instruction{Flags: flags}
	if strings.HasPrefix(trimmed, "[") {
		var values []string
		if err := json.Unmarshal([]byte(trimmed), &values); err != nil {
			return nil, fmt.Errorf("invalid JSON instruction arguments: %w", err)
		}
		instruction.Form = parser.FormJSON
		instruction.JSONArgs = values
		instruction.Args = append([]string(nil), values...)
		return instruction, nil
	}
	instruction.Form = parser.FormShell
	instruction.Args = append([]string(nil), tokens...)
	instruction.Command = rawArgs
	return instruction, nil
}

func parseJSONOrWords(ctx *parser.Context, input parser.InstructionInput) (*parser.Instruction, error) {
	flags, tokens, err := ctx.ParseFlags(input)
	if err != nil {
		return nil, err
	}
	rawArgs := strings.Join(tokens, " ")
	trimmed := strings.TrimSpace(rawArgs)
	instruction := &parser.Instruction{Flags: flags}
	if strings.HasPrefix(trimmed, "[") {
		var values []string
		if err := json.Unmarshal([]byte(trimmed), &values); err != nil {
			return nil, fmt.Errorf("invalid JSON instruction arguments: %w", err)
		}
		instruction.Form = parser.FormJSON
		instruction.JSONArgs = values
		instruction.Args = append([]string(nil), values...)
		return instruction, nil
	}
	instruction.Form = parser.FormWords
	instruction.Args = append([]string(nil), tokens...)
	return instruction, nil
}

func parsePairs(words []string) []parser.KeyValue {
	pairs := make([]parser.KeyValue, 0, len(words))
	for _, word := range words {
		key, value, hasValue := strings.Cut(word, "=")
		pairs = append(pairs, parser.KeyValue{Key: key, Value: value, HasValue: hasValue})
	}
	return pairs
}

func parseLegacyEnv(words []string) []parser.KeyValue {
	if len(words) == 0 {
		return nil
	}
	if strings.Contains(words[0], "=") {
		return parsePairs(words)
	}
	if len(words) == 1 {
		return []parser.KeyValue{{Key: words[0]}}
	}
	return []parser.KeyValue{{Key: words[0], Value: strings.Join(words[1:], " "), HasValue: true}}
}

func unsetEnabled(flags parser.FlagSet) bool {
	flag, ok := flags.First("unset")
	return ok && (!flag.HasValue || flag.Value != "false")
}
