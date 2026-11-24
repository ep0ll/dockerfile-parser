package parser

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ep0ll/dockerfile-parser/packages/go/lexer"
)

// Parser incrementally parses one instruction per Next call. It is a
// single-consumer iterator and does not enforce document-level semantics.
type Parser struct {
	lines                  LineReader
	escapeToken            rune
	commentPrefix          string
	commentHandlers        []CommentHandler
	continuationParser     ContinuationParser
	commentParser          CommentParser
	directiveParser        DirectiveParser
	instructionSplitter    InstructionSplitter
	flagParser             FlagParser
	instructionNormalizer  NameNormalizer
	directiveNormalizer    NameNormalizer
	directiveHandlers      map[string]DirectiveHandler
	instructionHandlers    map[string]InstructionHandler
	flagSpecs              map[flagSchemaKey][]FlagSpec
	unknownHandler         InstructionHandler
	unknownFlagPolicy      UnknownFlagPolicy
	directivesEnabled      bool
	directivesPreambleOnly bool
	preambleOpen           bool
	lineStarted            bool
	pendingComments        []string
	directives             []Directive
	metadata               map[string]any
}

// New constructs an independent parser instance with the supplied options.
func New(r io.Reader, options ...Option) (*Parser, error) {
	if r == nil {
		return nil, fmt.Errorf("source reader cannot be nil")
	}
	cfg := defaultConfig()
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&cfg); err != nil {
			return nil, err
		}
	}
	lines := cfg.lineReaderFactory(r)
	if lines == nil {
		return nil, fmt.Errorf("line reader factory returned nil")
	}
	unknown := cfg.unknownHandler
	if unknown == nil {
		unknown = parseUnknown
	}
	return &Parser{
		lines:                  lines,
		escapeToken:            cfg.escapeToken,
		commentPrefix:          cfg.commentPrefix,
		commentHandlers:        append([]CommentHandler(nil), cfg.commentHandlers...),
		continuationParser:     cfg.continuationParser,
		commentParser:          cfg.commentParser,
		directiveParser:        cfg.directiveParser,
		instructionSplitter:    cfg.instructionSplitter,
		flagParser:             cfg.flagParser,
		instructionNormalizer:  cfg.instructionNormalizer,
		directiveNormalizer:    cfg.directiveNormalizer,
		directiveHandlers:      cfg.directiveHandlers,
		instructionHandlers:    cfg.instructionHandlers,
		flagSpecs:              cfg.flagSpecs,
		unknownHandler:         unknown,
		unknownFlagPolicy:      cfg.unknownFlagPolicy,
		directivesEnabled:      cfg.directivesEnabled,
		directivesPreambleOnly: cfg.directivesPreambleOnly,
		preambleOpen:           true,
		metadata:               make(map[string]any),
	}, nil
}

// Next returns the next parsed instruction or io.EOF. It never performs
// document-level validation such as FROM ordering or ONBUILD placement.
func (p *Parser) Next() (*Instruction, error) {
	for {
		line, err := p.lines.Next()
		if err != nil {
			return nil, err
		}
		if !p.lineStarted {
			line.Text = lexer.StripBOM(line.Text)
			p.lineStarted = true
		}
		if p.isComment(line.Text) {
			if err := p.handleComment(line, false); err != nil {
				return nil, err
			}
			continue
		}
		if strings.TrimSpace(line.Text) == "" {
			// Blank lines do not consume the parser-directive preamble. They
			// also separate comments from the next instruction, so discard any
			// pending documentation comments here.
			p.pendingComments = nil
			continue
		}

		p.closePreamble()
		logical, endLine, err := p.logicalLine(line)
		if err != nil {
			var positioned *ParseError
			if errors.As(err, &positioned) {
				return nil, err
			}
			return nil, &ParseError{Position: Position{StartLine: line.Number, EndLine: endLine}, Err: err}
		}
		position := Position{StartLine: line.Number, EndLine: endLine}
		instruction, err := p.parseInstruction(logical, "", position, p.pendingComments)
		if err != nil {
			return nil, &ParseError{Position: position, Err: err}
		}
		p.pendingComments = nil
		return instruction, nil
	}
}

// Directives returns a defensive copy of the directives accepted so far.
func (p *Parser) Directives() []Directive { return append([]Directive(nil), p.directives...) }

// Metadata returns a shallow copy of parser metadata.
func (p *Parser) Metadata() map[string]any {
	copyOf := make(map[string]any, len(p.metadata))
	for key, value := range p.metadata {
		copyOf[key] = value
	}
	return copyOf
}

// EscapeToken returns the active line-escape token.
func (p *Parser) EscapeToken() rune { return p.escapeToken }

func (p *Parser) closePreamble() {
	if p.directivesPreambleOnly {
		p.preambleOpen = false
	}
}

func (p *Parser) isComment(line string) bool {
	_, ok := p.commentParser(line, p.commentPrefix)
	return ok
}

func (p *Parser) handleComment(line lexer.Line, continuation bool) error {
	body, _ := p.commentParser(line.Text, p.commentPrefix)
	position := Position{StartLine: line.Number, EndLine: line.Number}
	if p.directivesEnabled && (!p.directivesPreambleOnly || p.preambleOpen) {
		if directive, ok := p.directiveParser(body, position); ok {
			if handler, registered := p.directiveHandlers[p.directiveNormalizer(directive.Name)]; registered {
				p.directives = append(p.directives, directive)
				if err := handler(&DirectiveContext{parser: p}, directive); err != nil {
					return &ParseError{Position: position, Err: err}
				}
				return nil
			}
		}
	}
	comment := Comment{Text: body, Raw: line.Text, Position: position, DuringContinuation: continuation}
	for _, handler := range p.commentHandlers {
		if err := handler(comment); err != nil {
			return &ParseError{Position: position, Err: err}
		}
	}
	if !continuation {
		p.pendingComments = append(p.pendingComments, body)
	}
	p.closePreamble()
	return nil
}

func (p *Parser) logicalLine(first lexer.Line) (string, int, error) {
	part, continued, err := p.continuationParser(first.Text, p.escapeToken)
	if err != nil {
		return "", first.Number, err
	}
	var builder strings.Builder
	builder.WriteString(strings.TrimRight(part, " \t"))
	endLine := first.Number
	for continued {
		next, readErr := p.lines.Next()
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				cause := fmt.Errorf("unexpected end of file after line continuation")
				return "", endLine, &ParseError{Position: Position{StartLine: first.Number, EndLine: endLine}, Err: cause}
			}
			return "", endLine, readErr
		}
		endLine = next.Number
		if p.isComment(next.Text) {
			if err := p.handleComment(next, true); err != nil {
				return "", endLine, err
			}
			continue
		}
		if strings.TrimSpace(next.Text) == "" {
			continue
		}
		part, continued, err = p.continuationParser(next.Text, p.escapeToken)
		if err != nil {
			return "", endLine, err
		}
		part = strings.TrimLeft(part, " \t")
		if part != "" {
			if builder.Len() > 0 {
				builder.WriteByte(' ')
			}
			builder.WriteString(strings.TrimRight(part, " \t"))
		}
	}
	return builder.String(), endLine, nil
}

func (p *Parser) parseInstruction(raw, scope string, position Position, comments []string) (*Instruction, error) {
	name, args, ok := p.instructionSplitter(raw)
	if !ok {
		return nil, fmt.Errorf("expected an instruction name")
	}
	input := InstructionInput{Name: name, Raw: raw, RawArgs: args, Scope: scope, Position: position}
	handler := p.instructionHandlers[p.instructionNormalizer(name)]
	if handler == nil {
		handler = p.unknownHandler
	}
	instruction, err := handler(&Context{parser: p, Scope: scope}, input)
	if err != nil {
		return nil, err
	}
	if instruction == nil {
		instruction = &Instruction{}
	}
	if instruction.Name == "" {
		instruction.Name = name
	}
	if instruction.Raw == "" {
		instruction.Raw = raw
	}
	if instruction.RawArgs == "" {
		instruction.RawArgs = args
	}
	if instruction.Position.StartLine == 0 {
		instruction.Position = position
	}
	if instruction.Comments == nil {
		instruction.Comments = append([]string(nil), comments...)
	}
	return instruction, nil
}

func parseUnknown(ctx *Context, input InstructionInput) (*Instruction, error) {
	flags, args, err := ctx.ParseFlags(input)
	if err != nil {
		return nil, err
	}
	return &Instruction{Flags: flags, Args: args, Unknown: true}, nil
}

// ParseAll eagerly consumes the stream. Prefer New+Next when instructions
// should be processed, rejected, or transformed one at a time.
func ParseAll(r io.Reader, options ...Option) ([]*Instruction, error) {
	p, err := New(r, options...)
	if err != nil {
		return nil, err
	}
	var instructions []*Instruction
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
