package lexer

import (
    "strings"
    "unicode/utf8"
)

// TrimContinuation removes an unescaped trailing continuation token and
// reports whether the logical line continues. A run of N trailing escape
// tokens continues only when N is odd; preceding pairs remain literal.
func TrimContinuation(line string, escape rune) (string, bool) {
    if escape == 0 || line == "" {
        return line, false
    }
    trimmed := strings.TrimRight(line, " \t\v\f")
    if trimmed == "" {
        return line, false
    }
    count := 0
    for i := len(trimmed); i > 0; {
        r, size := runeAtEnd(trimmed[:i])
        if r != escape {
            break
        }
        count++
        i -= size
    }
    if count%2 == 0 {
        return line, false
    }
    // Remove the continuation token and trailing horizontal whitespace.
    _, size := runeAtEnd(trimmed)
    return strings.TrimRight(trimmed[:len(trimmed)-size], " \t"), true
}

func runeAtEnd(s string) (rune, int) {
    r, size := utf8.DecodeLastRuneInString(s)
    return r, size
}
