// Package parser implements a language-neutral, streaming instruction parser.
// Dialects provide instruction/directive/comment handlers and lexical policy.
package parser

import (
	"fmt"
	"io"

	"github.com/ep0ll/dockerfile-parser/packages/go/lexer"
)

// Position describes a source range using one-based physical line numbers.
type Position struct {
	StartLine int
	EndLine   int
}

// ArgumentForm identifies how an instruction's arguments were represented.
type ArgumentForm uint8

const (
	FormWords ArgumentForm = iota
	FormShell
	FormJSON
	FormKeyValues
	FormRaw
)

// KeyValue is a parsed key with an optional value (ARG supports an omitted one).
type KeyValue struct {
	Key      string
	Value    string
	HasValue bool
}

// Instruction is the dialect-neutral result of parsing one instruction.
// Data is available for dialect-specific payloads without constraining custom
// languages to a shared, prematurely fixed AST.
type Instruction struct {
	Name     string
	Raw      string
	RawArgs  string
	Command  string
	Args     []string
	Flags    FlagSet
	Form     ArgumentForm
	JSONArgs []string
	Pairs    []KeyValue
	Children []*Instruction
	Bodies   []Body
	Data     any
	Unknown  bool
	Position Position
	Comments []string
}

// InstructionInput is the input passed to one instruction handler.
type InstructionInput struct {
	Name     string
	Raw      string
	RawArgs  string
	Scope    string
	Position Position
	Bodies   []Body
}

// Body is an opaque, language-neutral block attached to an instruction.
// Kind identifies the block grammar (for example, "heredoc"); Data carries
// dialect-specific metadata without coupling the parser core to that dialect.
type Body struct {
	Kind     string
	Name     string
	Content  string
	Position Position
	Data     any
}

// LineReader is the minimal contract required from a physical-line lexer.
type LineReader interface{ Next() (lexer.Line, error) }

// BodyContext provides controlled access to subsequent source lines while a
// body collector consumes a multiline block belonging to the current statement.
type BodyContext struct {
	parser  *Parser
	endLine int
}

// NextLine consumes one physical source line and updates the current range.
func (c *BodyContext) NextLine() (lexer.Line, error) {
	line, err := c.parser.lines.Next()
	if err == nil {
		c.endLine = line.Number
	}
	return line, err
}

// EndLine returns the latest physical line consumed by the body collector.
func (c *BodyContext) EndLine() int { return c.endLine }

// EscapeToken returns the active lexical escape token.
func (c *BodyContext) EscapeToken() rune { return c.parser.escapeToken }

// BodyCollector consumes multiline bodies associated with an instruction.
// It is invoked before the instruction handler and can read following lines.
type BodyCollector func(*BodyContext, InstructionInput) ([]Body, error)

// LineReaderFactory lets a dialect replace the physical source reader.
type LineReaderFactory func(io.Reader) LineReader

// ContinuationParser joins physical lines according to a dialect's escape policy.
type ContinuationParser func(string, rune) (string, bool, error)

// CommentParser recognises a physical comment line and returns its body.
type CommentParser func(line, prefix string) (body string, isComment bool)

// DirectiveParser recognises a directive from a comment body.
type DirectiveParser func(comment string, position Position) (Directive, bool)

// InstructionSplitter separates a logical line into its instruction name and raw arguments.
type InstructionSplitter func(line string) (name, args string, ok bool)

// FlagParser is replaceable for dialects whose flag grammar differs from --name[=value].
type FlagParser func(scope, instruction, raw string, specs []FlagSpec, unknown UnknownFlagPolicy, escape rune) (FlagSet, []string, error)

// InstructionHandler parses exactly one instruction.
type InstructionHandler func(*Context, InstructionInput) (*Instruction, error)

// Comment describes a comment encountered by the parser.
type Comment struct {
	Text                string
	Raw                 string
	Position            Position
	DuringContinuation  bool
}

// CommentHandler is invoked independently for each comment.
type CommentHandler func(Comment) error

// Directive is a parsed key=value directive embedded in a comment.
type Directive struct {
	Name     string
	Value    string
	Raw      string
	Position Position
}

// DirectiveHandler is independent from instruction parsing.
type DirectiveHandler func(*DirectiveContext, Directive) error

// ParseError associates a handler or lexical error with its source position.
type ParseError struct {
	Position Position
	Err      error
}

func (e *ParseError) Error() string {
	if e.Position.EndLine > e.Position.StartLine {
		return fmt.Sprintf("parse error on lines %d-%d: %v", e.Position.StartLine, e.Position.EndLine, e.Err)
	}
	return fmt.Sprintf("parse error on line %d: %v", e.Position.StartLine, e.Err)
}

func (e *ParseError) Unwrap() error { return e.Err }
