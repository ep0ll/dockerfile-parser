package parser

import (
    "strings"

    "github.com/ep0ll/dockerfile-parser/packages/go/lexer"
)

// DefaultCommentParser recognizes a prefix after leading horizontal whitespace.
func DefaultCommentParser(line, prefix string) (string, bool) {
    trimmed := strings.TrimLeft(line, " \t\v\f")
    if !strings.HasPrefix(trimmed, prefix) {
        return "", false
    }
    return strings.TrimSpace(strings.TrimPrefix(trimmed, prefix)), true
}

// DefaultDirectiveParser parses the conventional name=value directive syntax.
func DefaultDirectiveParser(comment string, position Position) (Directive, bool) {
    return parseDirective(comment, position)
}

// DefaultInstructionSplitter separates the first non-whitespace word from the
// remainder of a logical line.
func DefaultInstructionSplitter(line string) (name, args string, ok bool) {
    line = strings.TrimLeft(line, " \t\v\f")
    if line == "" {
        return "", "", false
    }
    index := strings.IndexAny(line, " \t\v\f")
    if index < 0 {
        return line, "", true
    }
    return line[:index], strings.TrimLeft(line[index:], " \t\v\f"), true
}

// DefaultContinuationParser delegates to the line-continuation implementation
// in the standard-library-only lexer package.
func DefaultContinuationParser(line string, escape rune) (string, bool, error) {
    value, continued := lexer.TrimContinuation(line, escape)
    return value, continued, nil
}
