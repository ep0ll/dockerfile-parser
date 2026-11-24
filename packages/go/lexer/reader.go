// Package lexer provides language-neutral, line-oriented source reading.
// It deliberately does not know about Dockerfile instructions, comments, or
// parser directives; those policies belong to the parser/dialect layers.
package lexer

import (
    "bufio"
    "io"
    "strings"
)

// Line is a physical source line. Text excludes the trailing newline and CR.
type Line struct {
    Text string
    Number int
}

// Reader reads physical lines without imposing bufio.Scanner's token-size
// limit. It preserves the final unterminated line and reports line numbers.
type Reader struct {
    r *bufio.Reader
    lineNo int
}

func NewReader(r io.Reader) *Reader {
    return &Reader{r: bufio.NewReader(r)}
}

func (r *Reader) Next() (Line, error) {
    text, err := r.r.ReadString('\n')
    if err != nil && err != io.EOF {
        return Line{}, err
    }
    if err == io.EOF && text == "" {
        return Line{}, io.EOF
    }
    r.lineNo++
    text = strings.TrimSuffix(text, "\n")
    text = strings.TrimSuffix(text, "\r")
    return Line{Text: text, Number: r.lineNo}, nil
}

// StripBOM removes a UTF-8 byte-order mark from the beginning of a source.
func StripBOM(s string) string {
    return strings.TrimPrefix(s, "\uFEFF")
}
