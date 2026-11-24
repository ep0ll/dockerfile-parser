package directives

import (
	"fmt"

	"github.com/ep0ll/dockerfile-parser/packages/go/parser"
)

func Syntax(ctx *parser.DirectiveContext, directive parser.Directive) error {
	if ctx.Occurrences(directive.Name) > 1 {
		return fmt.Errorf("duplicate syntax directive")
	}
	if directive.Value == "" {
		return fmt.Errorf("syntax directive requires a value")
	}
	ctx.Set("dockerfile.syntax", directive.Value)
	return nil
}
