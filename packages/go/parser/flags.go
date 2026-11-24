package parser

import (
	"fmt"
	"strings"
)

type FlagValueMode uint8

const (
	FlagNoValue FlagValueMode = iota
	FlagRequiredValue
	FlagOptionalValue
	FlagOptionalInlineValue
)

type FlagSpec struct {
	Name       string
	ValueMode  FlagValueMode
	Repeatable bool
}

type Flag struct {
	Name     string
	Value    string
	HasValue bool
	Raw      string
	Unknown  bool
}

type FlagSet []Flag

func (f FlagSet) Values(name string) []string {
	name = normalizeFlagName(name)
	values := make([]string, 0)
	for _, item := range f {
		if strings.EqualFold(item.Name, name) && item.HasValue {
			values = append(values, item.Value)
		}
	}
	return values
}

func (f FlagSet) Has(name string) bool {
	name = normalizeFlagName(name)
	for _, item := range f {
		if strings.EqualFold(item.Name, name) {
			return true
		}
	}
	return false
}

func (f FlagSet) First(name string) (Flag, bool) {
	name = normalizeFlagName(name)
	for _, item := range f {
		if strings.EqualFold(item.Name, name) {
			return item, true
		}
	}
	return Flag{}, false
}

type UnknownFlagPolicy uint8

const (
	UnknownFlagPreserve UnknownFlagPolicy = iota
	UnknownFlagAsArgument
	UnknownFlagError
)

type Context struct {
	parser *Parser
	Scope  string
}

func (c *Context) EscapeToken() rune { return c.parser.escapeToken }

func (c *Context) ParseFlags(input InstructionInput) (FlagSet, []string, error) {
	return c.parser.parseFlags(input.Scope, input.Name, input.RawArgs)
}

func (c *Context) Words(raw string) ([]string, error) {
	return SplitWords(raw, c.parser.escapeToken)
}

func (c *Context) ParseNested(raw, scope string, position Position) (*Instruction, error) {
	return c.parser.parseInstruction(raw, scope, position, nil)
}

func (c *Context) FlagSpecs(instruction string) []FlagSpec {
	return c.parser.flagSpecsFor(c.Scope, instruction)
}

func ParseFlagArguments(scope, instruction, raw string, specs []FlagSpec, unknownPolicy UnknownFlagPolicy, escape rune) (FlagSet, []string, error) {
	tokens, err := SplitWords(raw, escape)
	if err != nil {
		return nil, nil, err
	}
	byName := make(map[string]FlagSpec, len(specs))
	for _, spec := range specs {
		byName[normalizeFlagName(spec.Name)] = spec
	}
	seen := make(map[string]bool)
	flags := make(FlagSet, 0)
	args := make([]string, 0, len(tokens))
	parsingFlags := true
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		if !parsingFlags || !strings.HasPrefix(token, "--") || token == "--" {
			if token == "--" && parsingFlags {
				args = append(args, tokens[i+1:]...)
				break
			}
			parsingFlags = false
			args = append(args, token)
			continue
		}
		nameValue := strings.TrimPrefix(token, "--")
		name, value, hasEqual := strings.Cut(nameValue, "=")
		key := normalizeFlagName(name)
		spec, known := byName[key]
		if !known {
			switch unknownPolicy {
			case UnknownFlagError:
				return nil, nil, fmt.Errorf("unsupported flag --%s for %s", key, normalize(instruction))
			case UnknownFlagAsArgument:
				args = append(args, token)
			default:
				flags = append(flags, Flag{Name: key, Value: value, HasValue: hasEqual, Raw: token, Unknown: true})
			}
			continue
		}
		if seen[key] && !spec.Repeatable {
			return nil, nil, fmt.Errorf("flag --%s may not be repeated", key)
		}
		flag := Flag{Name: key, Raw: token}
		if hasEqual {
			if spec.ValueMode == FlagNoValue {
				return nil, nil, fmt.Errorf("flag --%s does not accept a value", key)
			}
			flag.Value, flag.HasValue = value, true
		} else {
			switch spec.ValueMode {
			case FlagRequiredValue:
				if i+1 >= len(tokens) || strings.HasPrefix(tokens[i+1], "--") {
					return nil, nil, fmt.Errorf("flag --%s requires a value", key)
				}
				i++
				flag.Value, flag.HasValue = tokens[i], true
			case FlagOptionalValue:
				if i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "--") {
					i++
					flag.Value, flag.HasValue = tokens[i], true
				}
			case FlagOptionalInlineValue:
			}
		}
		seen[key] = true
		flags = append(flags, flag)
	}
	return flags, args, nil
}

func (p *Parser) flagSpecsFor(scope, instruction string) []FlagSpec {
	key := flagKey(p.instructionNormalizer, scope, instruction)
	if specs, ok := p.flagSpecs[key]; ok {
		return specs
	}
	if scope != "" {
		if specs, ok := p.flagSpecs[flagKey(p.instructionNormalizer, "", instruction)]; ok {
			return specs
		}
	}
	return nil
}

func (p *Parser) parseFlags(scope, instruction, raw string) (FlagSet, []string, error) {
	return p.flagParser(scope, instruction, raw, p.flagSpecsFor(scope, instruction), p.unknownFlagPolicy, p.escapeToken)
}
