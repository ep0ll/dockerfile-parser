// Package dockerfile composes the generic parser with the Dockerfile dialect.
// Its defaults are deliberately syntax-oriented: document policy (stage order,
// ONBUILD restrictions, required arguments, and so on) belongs to consumers.
package dockerfile

import (
	"errors"
	"io"

	"github.com/ep0ll/dockerfile-parser/packages/go/dockerfile/directives"
	"github.com/ep0ll/dockerfile-parser/packages/go/dockerfile/heredoc"
	"github.com/ep0ll/dockerfile-parser/packages/go/dockerfile/instructions"
	"github.com/ep0ll/dockerfile-parser/packages/go/parser"
)

type Option = parser.Option

func DefaultOptions() []Option {
	options := []Option{
		parser.WithCommentPrefix("#"),
		parser.WithDirectivePreambleOnly(true),
		parser.WithBodyCollector(heredoc.Collect),
		parser.WithDirectiveHandler("escape", directives.Escape),
		parser.WithDirectiveHandler("syntax", directives.Syntax),
		parser.WithDirectiveHandler("check", directives.Check),
	}
	options = append(options, instructions.DefaultOptions()...)
	options = append(options, instructions.DefaultFlagOptions()...)
	return options
}

func NewParser(source io.Reader, options ...Option) (*parser.Parser, error) {
	all := DefaultOptions()
	all = append(all, options...)
	return parser.New(source, all...)
}

func New(source io.Reader, options ...Option) (*parser.Parser, error) {
	return NewParser(source, options...)
}

func ParseAll(source io.Reader, options ...Option) ([]*parser.Instruction, error) {
	p, err := NewParser(source, options...)
	if err != nil {
		return nil, err
	}
	var instructions []*parser.Instruction
	for {
		instruction, err := p.Next()
		if errors.Is(err, io.EOF) {
			return instructions, nil
		}
		if err != nil {
			return nil, err
		}
		instructions = append(instructions, instruction)
	}
}
