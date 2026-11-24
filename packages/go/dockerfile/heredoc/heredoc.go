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
	descriptor, err := parseDescriptor(prefix)
	if err != nil {
		return Spec{}, false, err
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
		FileDescriptor: descriptor,
		Expand:         !quoted,
		StripTabs:      stripTabs,
	}, true, nil
}

// Declarations finds unquoted heredoc operators in a shell-like command.
// It recognises both "cat <<EOF" and the adjacent form "cat<<EOF", as well
// as optional file descriptors, tab stripping, and quoted delimiters.
func Declarations(raw string, escape rune) ([]Spec, error) {
	runes := []rune(raw)
	var specs []Spec

	for i := 0; i < len(runes); {
		current := runes[i]
		if escape != 0 && current == escape {
			i += 2
			continue
		}
		if current == '\'' || current == '"' {
			quote := current
			i++
			for i < len(runes) {
				if escape != 0 && runes[i] == escape && quote != '\'' {
					i += 2
					continue
				}
				if runes[i] == quote {
					i++
					break
				}
				i++
			}
			continue
		}
		if current == '#' && (i == 0 || unicode.IsSpace(runes[i-1])) {
			break
		}
		if current != '<' || i+1 >= len(runes) || runes[i+1] != '<' ||
			i > 0 && runes[i-1] == '<' ||
			i+2 < len(runes) && runes[i+2] == '<' {
			i++
			continue
		}

		prefixStart := i - 1
		for prefixStart >= 0 && !unicode.IsSpace(runes[prefixStart]) &&
			!strings.ContainsRune(";&|()<>", runes[prefixStart]) {
			prefixStart--
		}
		prefix := string(runes[prefixStart+1 : i])
		descriptor := uint64(0)
		if prefix != "" {
			allDigits := true
			for _, r := range prefix {
				if r < '0' || r > '9' {
					allDigits = false
					break
				}
			}
			if allDigits {
				parsed, err := parseDescriptor(prefix)
				if err != nil {
					return nil, err
				}
				descriptor = uint64(parsed)
			}
		}

		cursor := i + 2
		stripTabs := cursor < len(runes) && runes[cursor] == '-'
		if stripTabs {
			cursor++
		}
		for cursor < len(runes) && unicode.IsSpace(runes[cursor]) {
			cursor++
		}
		delimiterStart := cursor
		delimiterRaw, end, err := scanDelimiter(runes, delimiterStart, escape)
		if err != nil {
			return nil, err
		}
		if delimiterRaw == "" {
			return nil, fmt.Errorf("heredoc declaration requires a delimiter")
		}
		delimiter, quoted, err := parseDelimiter(delimiterRaw, escape)
		if err != nil {
			return nil, err
		}
		if delimiter == "" {
			return nil, fmt.Errorf("heredoc delimiter cannot be empty")
		}
		specs = append(specs, Spec{
			Delimiter:      delimiter,
			FileDescriptor: uint(descriptor),
			Expand:         !quoted,
			StripTabs:      stripTabs,
		})
		i = end
	}
	return specs, nil
}

// Collect reads heredoc bodies for RUN, ADD, and COPY statements. It is
// configured by the Dockerfile facade, not hard-coded in the generic parser.
func Collect(source *parser.BodyContext, input parser.InstructionInput) ([]parser.Body, error) {
	switch strings.ToUpper(input.Name) {
	case "RUN", "ADD", "COPY":
	default:
		return nil, nil
	}

	specs, err := Declarations(input.RawArgs, source.EscapeToken())
	if err != nil {
		return nil, err
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

func parseDescriptor(prefix string) (uint, error) {
	if prefix == "" {
		return 0, nil
	}
	descriptor, err := strconv.ParseUint(prefix, 10, strconv.IntSize)
	if err != nil {
		return 0, fmt.Errorf("invalid heredoc file descriptor %q: %w", prefix, err)
	}
	return uint(descriptor), nil
}

func scanDelimiter(runes []rune, start int, escape rune) (string, int, error) {
	var quote rune
	cursor := start
	for cursor < len(runes) {
		current := runes[cursor]
		if escape != 0 && current == escape && quote != '\'' {
			if cursor+1 >= len(runes) {
				return "", cursor, fmt.Errorf("heredoc delimiter ends with an escape token")
			}
			cursor += 2
			continue
		}
		if quote != 0 {
			if current == quote {
				quote = 0
			}
			cursor++
			continue
		}
		switch current {
		case '\'', '"':
			quote = current
		default:
			if unicode.IsSpace(current) || strings.ContainsRune(";|&()<>", current) {
				return string(runes[start:cursor]), cursor, nil
			}
		}
		cursor++
	}
	if quote != 0 {
		return "", cursor, fmt.Errorf("unterminated quote in heredoc delimiter")
	}
	return string(runes[start:cursor]), cursor, nil
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
