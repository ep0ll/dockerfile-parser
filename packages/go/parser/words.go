package parser

import (
    "fmt"
    "strings"
    "unicode/utf8"
)

// SplitWords performs quote-aware whitespace splitting while preserving the
// original quotes and escape characters in each token. It is intentionally a
// lexical helper, not a shell evaluator.
func SplitWords(input string, escape rune) ([]string, error) {
    var words []string
    var token strings.Builder
    var quote rune
    started := false
    for i := 0; i < len(input); {
        r, size := utf8.DecodeRuneInString(input[i:])
        if r == utf8.RuneError && size == 1 {
            return nil, fmt.Errorf("invalid UTF-8 at byte %d", i)
        }
        if escape != 0 && r == escape {
            token.WriteRune(r)
            started = true
            i += size
            if i < len(input) {
                next, n := utf8.DecodeRuneInString(input[i:])
                token.WriteRune(next)
                i += n
            }
            continue
        }
        if quote != 0 {
            token.WriteRune(r)
            if r == quote {
                quote = 0
            }
            started = true
            i += size
            continue
        }
        if r == '\'' || r == '"' {
            quote = r
            token.WriteRune(r)
            started = true
            i += size
            continue
        }
        if r == ' ' || r == '\t' || r == '\v' || r == '\f' || r == '\r' || r == '\n' {
            if started {
                words = append(words, token.String())
                token.Reset()
                started = false
            }
            i += size
            continue
        }
        token.WriteRune(r)
        started = true
        i += size
    }
    if quote != 0 {
        return nil, fmt.Errorf("unterminated %q quote", quote)
    }
    if started {
        words = append(words, token.String())
    }
    return words, nil
}
