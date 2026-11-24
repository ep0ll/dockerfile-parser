# dockerfile-parser

A language-agnostic parser project for Dockerfile-like instruction languages.
The first implementation is Go; additional language implementations can be
added without moving or coupling the Go module.

## Repository layout

```text
/specs/                 Language-neutral parser contracts and fixtures
/docs/                  Architecture decisions and design-pattern review
/packages/go/           Go lexer, generic parser, and Dockerfile dialect
/packages/typescript/   Reserved for a future TypeScript implementation
/packages/rust/         Reserved for a future Rust implementation
```

The implementation is split into three layers:

1. `packages/go/lexer` reads physical source lines and handles lexical mechanics.
2. `packages/go/parser` provides a language-neutral, streaming parser engine.
3. `packages/go/dockerfile` composes the engine with Dockerfile-specific directives and instruction handlers.

The parser does not enforce Dockerfile document-order or build semantics.
Those policies belong to the application consuming the stream.

See [Go usage](packages/go/README.md), [architecture](docs/architecture.md),
[pattern review](docs/pattern-review.md), and
[parser contract](specs/parser-contract.md).

## Test the Go implementation

```sh
cd packages/go
go test ./...
go test -race ./...
go vet ./...
```

The Go implementation is an extensible foundation, not yet complete BuildKit
behavioral parity. Basic RUN, ADD, and COPY heredoc collection is supported;
full shell grammar, variable expansion, frontend/build semantics, and some
Dockerfile validation remain out of scope.
