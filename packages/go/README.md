# Go parser

A standard-library-only Go implementation of a generic streaming parser and a
Dockerfile dialect.

## Install / test

```sh
go test ./...
go test -race ./...
go vet ./...
```

Module:

```text
github.com/ep0ll/dockerfile-parser/packages/go
```

## Stream one instruction at a time

```go
package main

import (
    "errors"
    "fmt"
    "io"
    "strings"

    "github.com/ep0ll/dockerfile-parser/packages/go/dockerfile"
)

func main() {
    p, err := dockerfile.NewParser(strings.NewReader(`
FROM alpine
RUN echo hello
ONBUILD RUN echo nested
`))
    if err != nil {
        panic(err)
    }

    for {
        instruction, err := p.Next()
        if errors.Is(err, io.EOF) {
            break
        }
        if err != nil {
            panic(err)
        }

        fmt.Printf("%s at lines %d-%d: %#v\\n",
            instruction.Name,
            instruction.Position.StartLine,
            instruction.Position.EndLine,
            instruction.Args,
        )

        // Decide whether to accept, reject, or apply business logic here.
        // The parser has not consumed the next instruction yet.
    }
}
```

`Next` is the primary API. `dockerfile.ParseAll` is available when eager
consumption is appropriate.

## Independent instruction handlers

Each built-in instruction has one exported handler in its own file under
`dockerfile/instructions`:

```text
add.go          ParseAdd
arg.go          ParseArg
cmd.go          ParseCmd
copy.go         ParseCopy
entrypoint.go   ParseEntrypoint
env.go          ParseEnv
expose.go       ParseExpose
from.go         ParseFrom
healthcheck.go  ParseHealthcheck
label.go        ParseLabel
maintainer.go   ParseMaintainer
onbuild.go      ParseOnbuild
run.go          ParseRun
shell.go        ParseShell
stopsignal.go   ParseStopsignal
user.go         ParseUser
volume.go       ParseVolume
workdir.go      ParseWorkdir
```

The handlers share small lexical helpers, but each function is registered and
replaceable independently. There is no central instruction `switch` and no
package-global mutable dispatch registry.

## Add a custom instruction

```go
p, err := dockerfile.NewParser(strings.NewReader(
    "PUBLISH --registry=registry.example/app:1.0 ./dist\\n",
),
    parser.WithFlagSpecs("", "PUBLISH",
        parser.FlagSpec{Name: "registry", ValueMode: parser.FlagRequiredValue},
    ),
    parser.WithInstructionHandler("PUBLISH", func(
        ctx *parser.Context,
        input parser.InstructionInput,
    ) (*parser.Instruction, error) {
        flags, args, err := ctx.ParseFlags(input)
        if err != nil {
            return nil, err
        }
        registry, _ := flags.First("registry")
        return &parser.Instruction{
            Flags: flags,
            Args:  args,
            Data:  map[string]string{"registry": registry.Value},
        }, nil
    }),
)
if err != nil {
    panic(err)
}
_ = p
```

A registration with an existing name replaces only that instruction handler.
An unknown instruction is preserved by default; use
`parser.WithUnknownInstructionHandler` or a custom handler to change that.

## Replace an instruction's complete flag schema

`WithFlagSpecs` replaces, rather than appends to, the schema for one
`(scope, instruction)` pair. Passing no specs clears it.

```go
parser.WithFlagSpecs("", "COPY",
    parser.FlagSpec{Name: "manifest", ValueMode: parser.FlagRequiredValue},
)
```

Flag value modes:

- `FlagNoValue`: `--flag`
- `FlagRequiredValue`: `--flag=value` or `--flag value`
- `FlagOptionalValue`: `--flag`, `--flag=value`, or `--flag value`
- `FlagOptionalInlineValue`: `--flag` or `--flag=value`; never consumes the
  next positional token

Unknown flags are preserved by default. Pair a custom schema with
`parser.WithUnknownFlagPolicy(parser.UnknownFlagError)` when the schema should
be a strict allow-list. Unknown-flag behavior is deliberately separate from
schema replacement so a dialect can preserve vendor-specific extensions.

## ONBUILD reuses the registered handler

`ONBUILD RUN ...` does not have a second RUN parser. `ParseOnbuild` delegates
the nested statement to the same instruction registry and changes only the
scope to `onbuild`.

```go
parser.WithFlagSpecs("onbuild", "RUN",
    parser.FlagSpec{Name: "cache", ValueMode: parser.FlagRequiredValue},
)
```

When a scoped schema exists, it takes precedence over the top-level schema.
When it does not exist, the parser falls back to the top-level schema. To
replace a schema with an empty set, call `WithFlagSpecs("onbuild", "RUN")`.

The parser intentionally does not decide which instructions may appear below
`ONBUILD`. Applications can inspect `Instruction.Children` and apply their own
policy.

## Names and directives are configurable

The default name normalizer is Dockerfile-style case-insensitive matching.
A custom language can choose exact-case matching or its own naming rules:

```go
parser.WithInstructionNameNormalizer(func(name string) string { return name })
parser.WithDirectiveNameNormalizer(strings.ToLower)
```

Configure a normalizer before custom registrations. When applied after the
Dockerfile defaults, the option re-keys existing handlers and flag schemas.
Collisions are reported during parser construction rather than silently
overwriting a registration.

Built-in Dockerfile directives live in individual files under
`dockerfile/directives` (`Escape`, `Syntax`, and `Check`). Replace or add a
handler with `parser.WithDirectiveHandler`. The parser syntax itself is
replaceable using `parser.WithDirectiveParser`.

Comments can be recognized with `parser.WithCommentParser` and delivered to
one or more independent callbacks using `parser.WithCommentHandler`.

Other replaceable lexical components are:

- `WithLineReaderFactory`
- `WithContinuationParser`
- `WithInstructionSplitter`
- `WithFlagParser`
- `WithBodyCollector`

## Multiline bodies and heredocs

The generic engine exposes `parser.WithBodyCollector`. A collector can read
following physical lines through `parser.BodyContext` and attach opaque
`parser.Body` values before an instruction handler runs.

The Dockerfile dialect registers `dockerfile/heredoc.Collect` by default. It
captures basic heredocs on `RUN`, `ADD`, and `COPY`:

```dockerfile
RUN cat <<'SCRIPT'
echo "$HOME"
SCRIPT
```

Each returned body has `Kind == "heredoc"`, its delimiter in `Name`, content
in `Content`, and source lines in `Position`. Its `Data` value is a
`heredoc.Spec`, which carries the file descriptor, whether shell expansion
was requested by the delimiter quoting, and whether `<<-` tab stripping is
enabled. The parser only captures source text; it does not interpolate variables
or execute shell code. Multiple heredocs and nested `ONBUILD RUN` bodies use
the same collector.

## Semantic validation belongs to the consumer

This parser does not require `FROM` to appear first, does not reject a top-level
`ONBUILD`, and does not decide whether instructions belong to a build stage.
It parses one statement and returns control to the caller. The caller can
implement any policy before advancing to the next statement.

## Current implementation boundaries

The Go package currently supports physical-line reading, configurable line
continuation, preamble directives (`escape`, `syntax`, `check`), comments,
quote-aware word splitting, JSON/shell/word forms for the implemented
instructions, independent flag schemas, nested `ONBUILD`, source positions,
and basic heredoc collection for `RUN`, `ADD`, and `COPY`.

Not yet implemented: full shell grammar, heredoc variable expansion, all
BuildKit frontend semantics, stage construction, Docker build validation, and
full lint/rule compatibility. These are distinct layers and will be added
without coupling them to the generic parser core.
