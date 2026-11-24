package lexer

import "testing"

func TestTrimContinuation(t *testing.T) {
    tests := []struct {
        name string
        line string
        want string
        more bool
    }{
        {name: "single trailing escape", line: "RUN echo hi \\", want: "RUN echo hi", more: true},
        {name: "escaped trailing escape pair", line: "RUN echo hi \\\\", want: "RUN echo hi \\\\", more: false},
        {name: "whitespace after escape", line: "RUN echo hi \\  ", want: "RUN echo hi", more: true},
        {name: "no escape", line: "RUN echo hi", want: "RUN echo hi", more: false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, more := TrimContinuation(tt.line, '\\')
            if got != tt.want || more != tt.more {
                t.Fatalf("TrimContinuation() = (%q, %v), want (%q, %v)", got, more, tt.want, tt.more)
            }
        })
    }
}

func TestTrimContinuationUnicodeToken(t *testing.T) {
    got, more := TrimContinuation("RUN echo hi §", '§')
    if got != "RUN echo hi" || !more {
        t.Fatalf("got (%q, %v)", got, more)
    }
}
