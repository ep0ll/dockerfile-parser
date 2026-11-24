package main

import (
    "errors"
    "fmt"
    "io"
    "strings"

    "github.com/ep0ll/dockerfile-parser/packages/go/dockerfile"
    "github.com/ep0ll/dockerfile-parser/packages/go/parser"
)

func main() {
    source := "PUBLISH --registry=registry.example/app:1.0 ./dist\\n" +
        "ONBUILD PUBLISH --target=release ./dist\\n"

    p, err := dockerfile.NewParser(strings.NewReader(source),
        parser.WithFlagSpecs("", "PUBLISH",
            parser.FlagSpec{Name: "registry", ValueMode: parser.FlagRequiredValue},
        ),
        parser.WithFlagSpecs("onbuild", "PUBLISH",
            parser.FlagSpec{Name: "target", ValueMode: parser.FlagRequiredValue},
        ),
        parser.WithUnknownFlagPolicy(parser.UnknownFlagError),
        parser.WithInstructionHandler("PUBLISH", parsePublish),
    )
    if err != nil {
        panic(err)
    }

    for {
        instruction, err := p.Next()
        if errors.Is(err, io.EOF) {
            return
        }
        if err != nil {
            panic(err)
        }
        fmt.Printf("%s: %#v\\n", instruction.Name, instruction)
    }
}

func parsePublish(ctx *parser.Context, input parser.InstructionInput) (*parser.Instruction, error) {
    flags, args, err := ctx.ParseFlags(input)
    if err != nil {
        return nil, err
    }
    return &parser.Instruction{Flags: flags, Args: args}, nil
}
