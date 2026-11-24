package main

import (
    "errors"
    "fmt"
    "io"
    "os"

    "github.com/ep0ll/dockerfile-parser/packages/go/dockerfile"
)

func main() {
    p, err := dockerfile.NewParser(os.Stdin)
    if err != nil {
        fatal(err)
    }

    for {
        instruction, err := p.Next()
        if errors.Is(err, io.EOF) {
            return
        }
        if err != nil {
            fatal(err)
        }
        fmt.Printf("%s %q (lines %d-%d)\\n",
            instruction.Name,
            instruction.RawArgs,
            instruction.Position.StartLine,
            instruction.Position.EndLine,
        )
    }
}

func fatal(err error) {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(1)
}
