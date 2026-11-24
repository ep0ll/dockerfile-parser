package parser

import (
    "fmt"
    "io"
    "strings"

    "github.com/ep0ll/dockerfile-parser/packages/go/lexer"
)

type Option func(*config) error

type config struct {
    escapeToken rune
    commentPrefix string
    commentHandlers []CommentHandler
    lineReaderFactory LineReaderFactory
    continuationParser ContinuationParser
    commentParser CommentParser
    directiveParser DirectiveParser
    instructionSplitter InstructionSplitter
    flagParser FlagParser
    directiveHandlers map[string]DirectiveHandler
    instructionHandlers map[string]InstructionHandler
    flagSpecs map[string][]FlagSpec
    unknownHandler InstructionHandler
    unknownFlagPolicy UnknownFlagPolicy
    directivesEnabled bool
    directivesPreambleOnly bool
}

func defaultConfig() config {
    return config{
        escapeToken: '\\',
        commentPrefix: "#",
        lineReaderFactory: func(r io.Reader) LineReader { return lexer.NewReader(r) },
        continuationParser: DefaultContinuationParser,
        commentParser: DefaultCommentParser,
        directiveParser: DefaultDirectiveParser,
        instructionSplitter: DefaultInstructionSplitter,
        flagParser: ParseFlagArguments,
        directiveHandlers: make(map[string]DirectiveHandler),
        instructionHandlers: make(map[string]InstructionHandler),
        flagSpecs: make(map[string][]FlagSpec),
        unknownFlagPolicy: UnknownFlagPreserve,
        directivesEnabled: true,
    }
}

// WithEscapeToken changes the character used to escape characters and join
// physical lines. The generic parser accepts any non-newline rune.
func WithEscapeToken(token rune) Option {
    return func(c *config) error {
        if token == 0 || token == '\n' || token == '\r' {
            return fmt.Errorf("invalid escape token %q", token)
        }
        c.escapeToken = token
        return nil
    }
}

func WithCommentPrefix(prefix string) Option {
    return func(c *config) error {
        if prefix == "" {
            return fmt.Errorf("comment prefix cannot be empty")
        }
        c.commentPrefix = prefix
        return nil
    }
}

// WithLineReaderFactory replaces the physical-line reader used by the parser.
func WithLineReaderFactory(factory LineReaderFactory) Option {
    return func(c *config) error {
        if factory == nil {
            return fmt.Errorf("line reader factory cannot be nil")
        }
        c.lineReaderFactory = factory
        return nil
    }
}

func WithContinuationParser(parser ContinuationParser) Option {
    return func(c *config) error {
        if parser == nil {
            return fmt.Errorf("continuation parser cannot be nil")
        }
        c.continuationParser = parser
        return nil
    }
}

func WithCommentParser(parser CommentParser) Option {
    return func(c *config) error {
        if parser == nil {
            return fmt.Errorf("comment parser cannot be nil")
        }
        c.commentParser = parser
        return nil
    }
}

func WithDirectiveParser(parser DirectiveParser) Option {
    return func(c *config) error {
        if parser == nil {
            return fmt.Errorf("directive parser cannot be nil")
        }
        c.directiveParser = parser
        return nil
    }
}

func WithInstructionSplitter(splitter InstructionSplitter) Option {
    return func(c *config) error {
        if splitter == nil {
            return fmt.Errorf("instruction splitter cannot be nil")
        }
        c.instructionSplitter = splitter
        return nil
    }
}

func WithFlagParser(parser FlagParser) Option {
    return func(c *config) error {
        if parser == nil {
            return fmt.Errorf("flag parser cannot be nil")
        }
        c.flagParser = parser
        return nil
    }
}

func WithCommentHandler(handler CommentHandler) Option {
    return func(c *config) error {
        if handler != nil {
            c.commentHandlers = append(c.commentHandlers, handler)
        }
        return nil
    }
}

// WithInstructionHandler registers or replaces one named instruction parser.
func WithInstructionHandler(name string, handler InstructionHandler) Option {
    return func(c *config) error {
        name = normalize(name)
        if name == "" || handler == nil {
            return fmt.Errorf("instruction name and handler are required")
        }
        c.instructionHandlers[name] = handler
        return nil
    }
}

// WithInstructionMiddleware decorates a registered handler. Because options
// are applied in order, compose this after the default dialect handlers.
func WithInstructionMiddleware(name string, middleware func(InstructionHandler) InstructionHandler) Option {
    return func(c *config) error {
        key := normalize(name)
        handler := c.instructionHandlers[key]
        if key == "" || handler == nil || middleware == nil {
            return fmt.Errorf("registered instruction and middleware are required")
        }
        wrapped := middleware(handler)
        if wrapped == nil {
            return fmt.Errorf("instruction middleware returned a nil handler")
        }
        c.instructionHandlers[key] = wrapped
        return nil
    }
}

func WithDirectiveHandler(name string, handler DirectiveHandler) Option {
    return func(c *config) error {
        name = normalize(name)
        if name == "" || handler == nil {
            return fmt.Errorf("directive name and handler are required")
        }
        c.directiveHandlers[name] = handler
        return nil
    }
}

func WithDirectivePreambleOnly(enabled bool) Option {
    return func(c *config) error { c.directivesPreambleOnly = enabled; return nil }
}

func WithDirectivesEnabled(enabled bool) Option {
    return func(c *config) error { c.directivesEnabled = enabled; return nil }
}

// WithFlagSpecs replaces the complete flag schema for instruction within a
// scope. An empty slice deliberately clears that instruction's flag schema.
// WithSimpleFlags declares a list of valueless flags. Use WithFlagSpecs when
// a flag consumes a value or has repeatability constraints.
func WithSimpleFlags(scope, instruction string, names ...string) Option {
    specs := make([]FlagSpec, 0, len(names))
    for _, name := range names {
        specs = append(specs, FlagSpec{Name: name, ValueMode: FlagNoValue})
    }
    return WithFlagSpecs(scope, instruction, specs...)
}

func WithFlagSpecs(scope, instruction string, specs ...FlagSpec) Option {
    return func(c *config) error {
        key := flagKey(scope, instruction)
        if key == "/" {
            return fmt.Errorf("instruction name is required for flag specs")
        }
        copyOfSpecs := append([]FlagSpec(nil), specs...)
        for _, spec := range copyOfSpecs {
            if normalizeFlagName(spec.Name) == "" {
                return fmt.Errorf("flag name cannot be empty")
            }
        }
        c.flagSpecs[key] = copyOfSpecs
        return nil
    }
}

func WithUnknownFlagPolicy(policy UnknownFlagPolicy) Option {
    return func(c *config) error { c.unknownFlagPolicy = policy; return nil }
}

func WithUnknownInstructionHandler(handler InstructionHandler) Option {
    return func(c *config) error { c.unknownHandler = handler; return nil }
}

func normalize(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }
func normalizeFlagName(s string) string { return strings.TrimLeft(strings.TrimSpace(s), "-") }
func flagKey(scope, instruction string) string {
    return normalize(scope) + "/" + normalize(instruction)
}
