package parser

import (
	"fmt"
	"io"
	"strings"

	"github.com/ep0ll/dockerfile-parser/packages/go/lexer"
)

type Option func(*config) error

// NameNormalizer maps a source-level command or directive name to its registry key.
// The default Dockerfile dialect is case-insensitive; custom languages can use
// identity or their own normalization rules.
type NameNormalizer func(string) string

type flagSchemaKey struct {
	scope       string
	instruction string
}

type config struct {
	escapeToken            rune
	commentPrefix          string
	commentHandlers        []CommentHandler
	lineReaderFactory      LineReaderFactory
	continuationParser     ContinuationParser
	commentParser          CommentParser
	directiveParser        DirectiveParser
	instructionSplitter    InstructionSplitter
	flagParser             FlagParser
	bodyCollector          BodyCollector
	instructionNormalizer  NameNormalizer
	directiveNormalizer    NameNormalizer
	directiveHandlers      map[string]DirectiveHandler
	instructionHandlers    map[string]InstructionHandler
	flagSpecs              map[flagSchemaKey][]FlagSpec
	unknownHandler         InstructionHandler
	unknownFlagPolicy      UnknownFlagPolicy
	directivesEnabled      bool
	directivesPreambleOnly bool
}

func defaultConfig() config {
	return config{
		escapeToken:           '\\',
		commentPrefix:         "#",
		lineReaderFactory:     func(r io.Reader) LineReader { return lexer.NewReader(r) },
		continuationParser:    DefaultContinuationParser,
		commentParser:         DefaultCommentParser,
		directiveParser:       DefaultDirectiveParser,
		instructionSplitter:   DefaultInstructionSplitter,
		flagParser:            ParseFlagArguments,
		instructionNormalizer: DefaultNameNormalizer,
		directiveNormalizer:   DefaultNameNormalizer,
		directiveHandlers:     make(map[string]DirectiveHandler),
		instructionHandlers:   make(map[string]InstructionHandler),
		flagSpecs:             make(map[flagSchemaKey][]FlagSpec),
		unknownFlagPolicy:     UnknownFlagPreserve,
		directivesEnabled:     true,
	}
}

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

// WithBodyCollector replaces the multiline-body consumer for this dialect.
// A nil collector is rejected; use a collector that returns no bodies to disable it.
func WithBodyCollector(collector BodyCollector) Option {
	return func(c *config) error {
		if collector == nil {
			return fmt.Errorf("body collector cannot be nil")
		}
		c.bodyCollector = collector
		return nil
	}
}

// WithInstructionNameNormalizer replaces instruction registry-key
// normalization. Existing registrations and flag schemas are re-keyed.
// Configure it before registering custom handlers when source case matters.
func WithInstructionNameNormalizer(normalizer NameNormalizer) Option {
	return func(c *config) error {
		if normalizer == nil {
			return fmt.Errorf("instruction name normalizer cannot be nil")
		}
		handlers := make(map[string]InstructionHandler, len(c.instructionHandlers))
		for name, handler := range c.instructionHandlers {
			key := normalizer(name)
			if strings.TrimSpace(key) == "" {
				return fmt.Errorf("instruction name normalizer produced an empty key for %q", name)
			}
			if _, exists := handlers[key]; exists {
				return fmt.Errorf("instruction name normalizer produced a duplicate key %q", key)
			}
			handlers[key] = handler
		}
		schemas := make(map[flagSchemaKey][]FlagSpec, len(c.flagSpecs))
		for key, specs := range c.flagSpecs {
			normalized := flagSchemaKey{
				scope:       normalizer(key.scope),
				instruction: normalizer(key.instruction),
			}
			if _, exists := schemas[normalized]; exists {
				return fmt.Errorf("instruction name normalizer produced a duplicate flag schema key for %q", key.instruction)
			}
			schemas[normalized] = specs
		}
		c.instructionNormalizer = normalizer
		c.instructionHandlers = handlers
		c.flagSpecs = schemas
		return nil
	}
}

// WithDirectiveNameNormalizer replaces directive registry-key normalization
// and re-keys handlers already registered in the current option sequence.
func WithDirectiveNameNormalizer(normalizer NameNormalizer) Option {
	return func(c *config) error {
		if normalizer == nil {
			return fmt.Errorf("directive name normalizer cannot be nil")
		}
		handlers := make(map[string]DirectiveHandler, len(c.directiveHandlers))
		for name, handler := range c.directiveHandlers {
			key := normalizer(name)
			if strings.TrimSpace(key) == "" {
				return fmt.Errorf("directive name normalizer produced an empty key for %q", name)
			}
			if _, exists := handlers[key]; exists {
				return fmt.Errorf("directive name normalizer produced a duplicate key %q", key)
			}
			handlers[key] = handler
		}
		c.directiveNormalizer = normalizer
		c.directiveHandlers = handlers
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

func WithInstructionHandler(name string, handler InstructionHandler) Option {
	return func(c *config) error {
		name = c.instructionNormalizer(name)
		if strings.TrimSpace(name) == "" || handler == nil {
			return fmt.Errorf("instruction name and handler are required")
		}
		c.instructionHandlers[name] = handler
		return nil
	}
}

func WithInstructionMiddleware(name string, middleware func(InstructionHandler) InstructionHandler) Option {
	return func(c *config) error {
		key := c.instructionNormalizer(name)
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
		name = c.directiveNormalizer(name)
		if strings.TrimSpace(name) == "" || handler == nil {
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

// WithFlagSpecs replaces the complete schema for (scope, instruction). An
// empty list clears the schema, even when a parent scope has a default schema.
func WithSimpleFlags(scope, instruction string, names ...string) Option {
	specs := make([]FlagSpec, 0, len(names))
	for _, name := range names {
		specs = append(specs, FlagSpec{Name: name, ValueMode: FlagNoValue})
	}
	return WithFlagSpecs(scope, instruction, specs...)
}

func WithFlagSpecs(scope, instruction string, specs ...FlagSpec) Option {
	return func(c *config) error {
		if strings.TrimSpace(instruction) == "" {
			return fmt.Errorf("instruction name is required for flag specs")
		}
		key := flagKey(c.instructionNormalizer, scope, instruction)
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

// DefaultNameNormalizer implements Dockerfile-style case-insensitive keys.
func DefaultNameNormalizer(s string) string { return normalize(s) }
func normalizeFlagName(s string) string     { return strings.TrimLeft(strings.TrimSpace(s), "-") }
func flagKey(normalizer NameNormalizer, scope, instruction string) flagSchemaKey {
	return flagSchemaKey{scope: normalizer(scope), instruction: normalizer(instruction)}
}
