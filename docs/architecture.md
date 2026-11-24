# Architecture

## Package placement

The Go code lives in `/packages/go`, rather than at repository root. This is a
language-neutral repository, so each implementation needs an independent
module, toolchain, test commands, and dependency graph. The root is reserved
for shared specifications, design notes, fixtures, CI orchestration, and
cross-language documentation. Future implementations can live beside Go:
`/packages/typescript`, `/packages/rust`, and so on.

The Go module path is:

```text
github.com/ep0ll/dockerfile-parser/packages/go
```

## Dependency direction

```text
                    dockerfile (dialect facade)
                      /                 \
           dockerfile/instructions   dockerfile/directives
                      \                 /
                       generic parser
                            |
                           lexer
```

- `lexer` depends only on the standard library. It reads physical lines and
  exposes continuation mechanics without knowing comments, directives, or
  Dockerfile instruction names.
- `parser` depends on `lexer`. It owns iteration, source positions, generic
  instruction results, flag tokenization/schema handling, directive/comment
  dispatch, metadata, and nested parsing.
- `dockerfile/instructions` depends on the generic parser API. Every default
  instruction handler is an independent exported function in its own file.
- `dockerfile/directives` contains one handler per Dockerfile parser directive.
- `dockerfile` composes those pieces and supplies convenience constructors.

The generic parser never imports the Dockerfile dialect. There is no mutable
global dispatch map and no package `init` registration. Every parser instance
receives its own copied configuration during construction.

## Streaming lifecycle

The primary API is `Next()`:

```go
for {
    instruction, err := p.Next()
    if errors.Is(err, io.EOF) {
        break
    }
    if err != nil {
        return err
    }
    if err := applicationPolicy(instruction); err != nil {
        return err // the parser is not advanced to the next instruction
    }
    if err := executeBusinessLogic(instruction); err != nil {
        return err
    }
}
```

`ParseAll` is a convenience for consumers that explicitly want eager parsing.
`Next` does not enforce Dockerfile build semantics. Therefore `ONBUILD RUN`
can be parsed before any `FROM`, and a top-level `RUN` can be parsed without a
stage. The consuming application decides whether such sequences are valid.

## Handler and flag extension model

A command is a string-keyed registry lookup. A handler gets a narrow context
with lexical helpers, its scope, and the ability to parse nested instructions.
It can parse only its own instruction. Registration replaces the handler with
the same normalized name; middleware decorates a specific registered handler.

Flags are a separate schema registry indexed by `(scope, instruction)`. A
scoped schema takes precedence over the default schema. This is used by
`ONBUILD`: `ONBUILD RUN` resolves the same `RUN` handler as top-level `RUN`,
but uses the `onbuild/RUN` schema when one is registered.

Example: replace the nested `RUN` schema entirely:

```go
parser.WithFlagSpecs("onbuild", "RUN",
    parser.FlagSpec{Name: "cache", ValueMode: parser.FlagRequiredValue},
)
```

To reject any flag not included in the new schema, also set
`parser.WithUnknownFlagPolicy(parser.UnknownFlagError)`. This policy is
intentionally separate because some applications want to preserve vendor
extensions rather than reject them.

## Directives are handlers too

The parser does not special-case `escape`, `syntax`, or `check`. The Dockerfile
facade registers handlers for these names. The escape handler updates the
parser's active continuation rune; the others store metadata. A custom dialect
can register an entirely different directive grammar with
`WithDirectiveParser` and its own named handlers.

Comment recognition and comment delivery are independent of directive
recognition. Comment handlers can observe comments without having to parse
instructions.

## Error and state ownership

The lexer owns physical line numbers. The generic parser owns parse positions
and parser state. A directive handler owns the policy for the directive it
handles (including whether duplicate occurrences are valid). An instruction
handler owns only that instruction's syntax. Application code owns global
semantics, such as stage order, legal nested instructions, or required fields.

The parser is stateful and should be consumed by one goroutine at a time. A
single stream is inherently ordered: parallelizing its parsing would complicate
directive state and `ONBUILD` context without creating useful independent
work. Applications can process already parsed instructions concurrently if
those operations are safe and do not require order.

## Explicitly deferred concerns

- Heredoc body collection needs a separate body-consumption contract because
  heredoc terminators temporarily suspend ordinary instruction iteration.
- Full shell parsing belongs in a shell package or a replaceable dialect
  component, not in the generic parser.
- Dockerfile stage interpretation, variable expansion, LLB generation,
  platform checks, and build execution are out of scope for syntax parsing.
- A lossless concrete syntax tree, formatter, and source rewrite engine can be
  added separately if preserving comments and exact whitespace becomes a
  primary use case.
