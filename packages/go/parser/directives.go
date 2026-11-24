package parser

import (
    "fmt"
    "strings"
)

// DirectiveContext exposes only state changes relevant to lexical parsing and
// directive metadata. Business validation remains in the directive handler.
type DirectiveContext struct{ parser *Parser }

func (c *DirectiveContext) SetEscapeToken(token rune) error {
    if token == 0 || token == '\n' || token == '\r' {
        return fmt.Errorf("invalid escape token %q", token)
    }
    c.parser.escapeToken = token
    return nil
}

func (c *DirectiveContext) EscapeToken() rune { return c.parser.escapeToken }

// Occurrences reports how many directives with the given name have been
// recognized up to and including the directive currently being handled. A
// directive handler can use it to opt into duplicate rejection or override
// semantics without imposing a global policy on every dialect.
func (c *DirectiveContext) Occurrences(name string) int {
    count := 0
    for _, directive := range c.parser.directives {
        if strings.EqualFold(directive.Name, name) {
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
    return Directive{Name: normalize(name), Value: strings.TrimSpace(value), Raw: comment, Position: position}, true
}
