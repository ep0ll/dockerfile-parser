// Package heredoc implements Dockerfile heredoc recognition and collection.
// It is a body-collection strategy, separate from instruction handlers.
package heredoc

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/ep0ll/dockerfile-parser/packages/go/parser"
)

// Spec describes the declaration of one Dockerfile heredoc.
type Spec struct {
	Delimiter      string
	FileDescriptor uint
	Expand         bool
	StripTabs      bool
}

// ParseWord parses one shell-like word as a heredoc declaration.
// Non-heredoc words return ok=false without error.
func ParseWord(word string, escape rune) (spec Spec, ok bool, err error) {
	marker := strings.Index(word, "<<")
	if marker < 0 {
		return Spec{}, false, nil
	}
	prefix := word[:marker]
	for _, r := range prefix {
		if r < '0' || r > '9' {
			return Spec{}, false, nil
		}
	}
	var descriptor uint64
	if prefix != "" {
		descriptor, err = strconv.ParseUint(prefix, 10, strconv.IntSize)
		if err != nil {
			return Spec{}, false, fmt.Errorf("invalid heredoc file descriptor %q: %w", prefix, err)
		}
	}

	rest := word[marker+2:]
	// A triple-chevron shell here-string is not a heredoc.
	if strings.HasPrefix(rest, "<") {
		return Spec{}, false, nil
	}
	stripTabs := strings.HasPrefix(rest, "-")
	if stripTabs {
		rest = rest[1:]
	}
	if rest == "" {
		return Spec{}, false, fmt.Errorf("heredoc declaration requires a delimiter")
	}

	delimiter, quoted, err := parseDelimiter(rest, escape)
	if err != nil {
		return Spec{}, false, err
	}
	if delimiter == "" {
		return Spec{}, false, fmt.Errorf("heredoc delimiter cannot be empty")
	}
	return Spec{
		Delimiter:      delimiter,
		FileDescriptor: uint(descriptor),
		Expand:         !quoted,
		StripTabs:      stripTabs,
	}, true, nil
}

// Collect reads heredoc bodies for RUN, ADD, and COPY statements. It is
// configured by the Dockerfile facade, not hard-coded in the generic parser.
func Collect(source *parser.BodyContext, input parser.InstructionInput) ([]parser.Body, error) {
	switch strings.ToUpper(input.Name) {
	case "RUN", "ADD", "COPY":
	default:
		return nil, nil
	}

	words, err := parser.SplitWords(input.RawArgs, source.EscapeToken())
	if err != nil {
		return nil, err
	}
	if startsWithJSON(words) {
		return nil, nil
	}

	var specs []Spec
	for _, word := range words {
		spec, ok, err := ParseWord(word, source.EscapeToken())
		if err != nil {
			return nil, err
		}
		if ok {
			specs = append(specs, spec)
		}
	}
	if len(specs) == 0 {
		return nil, nil
	}

	bodies := make([]parser.Body, 0, len(specs))
	for _, spec := range specs {
		bodyStart := source.EndLine() + 1
		var content strings.Builder
		for {
			line, err := source.NextLine()
			if err != nil {
				if err == io.EOF {
					return nil, fmt.Errorf("unterminated heredoc %q", spec.Delimiter)
				}
				return nil, err
			}

			candidate := line.Text
			if spec.StripTabs {
				candidate = strings.TrimLeft(candidate, "\t")
			}
			if candidate == spec.Delimiter {
				bodies = append(bodies, parser.Body{
					Kind:     "heredoc",
					Name:     spec.Delimiter,
					Content:  content.String(),
					Position: parser.Position{StartLine: bodyStart, EndLine: line.Number},
					Data:     spec,
				})
				break
			}

			bodyLine := line.Text
			if spec.StripTabs {
				bodyLine = strings.TrimLeft(bodyLine, "\t")
			}
			content.WriteString(bodyLine)
			content.WriteByte('\n')
		}
	}
	return bodies, nil
}

func startsWithJSON(words []string) bool {
	for _, word := range words {
		if strings.HasPrefix(word, "--") {
			continue
		}
		return strings.HasPrefix(word, "[")
	}
	return false
}

func parseDelimiter(raw string, escape rune) (string, bool, error) {
	var delimiter strings.Builder
	var quote rune
	quoted := false
	runes := []rune(raw)
	for i := 0; i < len(runes); i++ {
		current := runes[i]
		if escape != 0 && current == escape && quote != '\'' {
			if i+1 >= len(runes) {
				return "", false, fmt.Errorf("heredoc delimiter ends with an escape token")
			}
			i++
			delimiter.WriteRune(runes[i])
			continue
		}
		if quote != 0 {
			if current == quote {
				quote = 0
			} else {
				delimiter.WriteRune(current)
			}
			continue
		}
		switch current {
		case '\'', '"':
			quote = current
			quoted = true
		default:
			if unicode.IsSpace(current) {
				return "", false, fmt.Errorf("heredoc delimiter %q must be a single word", raw)
			}
			delimiter.WriteRune(current)
		}
	}
	if quote != 0 {
		return "", false, fmt.Errorf("unterminated quote in heredoc delimiter %q", raw)
	}
	return delimiter.String(), quoted, nil
}
