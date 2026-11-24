package directives

import (
	"fmt"
	"strings"

	"github.com/ep0ll/dockerfile-parser/packages/go/parser"
)

func Check(ctx *parser.DirectiveContext, directive parser.Directive) error {
	if ctx.Occurrences(directive.Name) > 1 {
		return fmt.Errorf("duplicate %s directive", strings.ToLower(directive.Name))
	}
	if strings.TrimSpace(directive.Value) == "" {
		return fmt.Errorf("check directive requires a value")
	}
	ctx.Set("dockerfile.check", directive.Value)
	return nil
}
