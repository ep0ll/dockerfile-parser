// Package directives contains independent parser-directive handlers.
package directives

import (
	"fmt"

	"github.com/ep0ll/dockerfile-parser/packages/go/parser"
)

func Escape(ctx *parser.DirectiveContext, directive parser.Directive) error {
	if ctx.Occurrences(directive.Name) > 1 {
		return fmt.Errorf("duplicate escape directive")
	}
	if directive.Value != "\\" && directive.Value != string(rune(96)) {
		return fmt.Errorf("invalid escape token %q: Dockerfile accepts backtick or backslash", directive.Value)
	}
	if err := ctx.SetEscapeToken([]rune(directive.Value)[0]); err != nil {
		return err
	}
	ctx.Set("dockerfile.escape", directive.Value)
	return nil
}
