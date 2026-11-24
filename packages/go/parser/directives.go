package parser

import (
	"fmt"
	"strings"
)

type DirectiveContext struct{ parser *Parser }

func (c *DirectiveContext) SetEscapeToken(token rune) error {
	if token == 0 || token == '\n' || token == '\r' {
		return fmt.Errorf("invalid escape token %q", token)
	}
	c.parser.escapeToken = token
	return nil
}

func (c *DirectiveContext) EscapeToken() rune { return c.parser.escapeToken }

func (c *DirectiveContext) Occurrences(name string) int {
	count := 0
	for _, directive := range c.parser.directives {
		if c.parser.directiveNormalizer(directive.Name) == c.parser.directiveNormalizer(name) {
			count++
		}
	}
	return count
}

func (c *DirectiveContext) Set(key string, value any) {
	c.parser.metadata[key] = value
}

func (c *DirectiveContext) Get(key string) (any, bool) {
	value, ok := c.parser.metadata[key]
	return value, ok
}

func parseDirective(comment string, position Position) (Directive, bool) {
	body := strings.TrimSpace(comment)
	name, value, ok := strings.Cut(body, "=")
	if !ok {
		return Directive{}, false
	}
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, " \t") {
		return Directive{}, false
	}
	for _, r := range name {
		if !(r == '-' || r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return Directive{}, false
		}
	}
	return Directive{Name: name, Value: strings.TrimSpace(value), Raw: comment, Position: position}, true
}
