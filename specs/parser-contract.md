# Language-neutral parser contract

This document defines the behavior that future language implementations
should share. It deliberately does not prescribe Go-specific types or runtime
mechanics.

## Responsibilities and boundaries

A parser implementation is composed from replaceable stages:

```text
byte source
  -> physical-line reader
  -> logical-line / continuation policy
  -> comment recognizer
  -> directive recognizer + directive handlers
  -> instruction splitter
  -> instruction-name registry
  -> instruction-local flag parser
  -> instruction handler
  -> one instruction event
```

A dialect may replace any stage. An application may replace or wrap any named
instruction handler and may choose when or whether to reject a parsed event.

## Physical input and source positions

- The physical-line reader yields one line at a time, without the line ending.
- It reports one-based physical line numbers.
- It must preserve a final line without a trailing newline.
- It should not impose an undocumented small maximum line length.
- Logical statements retain their starting and ending physical line numbers.
- A UTF-8 BOM, when supported by the implementation, is recognized only at
  the beginning of the source.

## Comments and directives

- Comment recognition is independent of instruction parsing.
- Every recognized comment can be delivered to zero or more comment handlers.
- A directive is a comment-shaped construct recognized by a configurable
  directive parser and dispatched to a handler registered by name.
- Directive handlers may update lexical state (for example, an escape rune),
  record metadata, reject duplicates, or apply any dialect-defined policy.
- The generic parser must not hard-code directive names or directive meaning.
- A dialect can decide whether directives are allowed only in a preamble or
  anywhere in the input. Blank lines do not inherently have to close a
  preamble; this is a dialect policy.
- Ordinary comments may be associated with the next instruction, but comments
  inside a continued logical statement are separately identifiable.

## Instruction iteration

- The streaming API returns exactly one instruction for each successful call.
- The caller can inspect, transform, accept, or reject the instruction before
  asking for the next one.
- End of input is distinguishable from parse failure.
- The parser must not silently impose document-level ordering rules. Examples
  include requiring `FROM` first, forbidding `ONBUILD` at the top of a file,
  or requiring instructions to appear inside a stage.
- These validations belong in explicit application-level policies, which may
  be composed around the streaming API.

## Instruction handlers

- A handler is registered by instruction name and receives exactly one
  instruction's raw text, parsed name, argument text, scope, and source range.
- A handler should not parse unrelated instruction names.
- A handler may call shared lexical helpers, but should own its grammar and
  produce an instruction result or an error.
- Unknown instruction behavior is configurable: preserve it, route it to a
  custom fallback, or reject it.
- Adding a custom instruction must not require modifying a central switch.
- Replacing a handler for an existing instruction must not affect other
  instruction handlers.

## Flags and scopes

- Each instruction has an independently replaceable flag schema.
- A flag schema defines its name, whether it accepts no value, a required
  value, an optional value, or an inline-only optional value, and whether it
  can repeat.
- A scope may override an instruction's normal schema. If no scoped schema
  exists, an implementation may fall back to the unscoped schema.
- A caller can replace the entire flag set for one instruction, including
  clearing it; it is not limited to appending flags.
- Unknown flags have a configurable policy. Preserving unknown flags must not
  guess whether the next positional token is their value.
- Flag syntax and flag semantics are separate. The parser handles lexical
  arity; a handler or application owns value validation and business meaning.

## Nested instructions

- A handler may delegate a nested instruction to the same registry.
- Nested parsing reuses the same instruction handlers, but carries a distinct
  scope so flags can be overridden for a nested context.
- `ONBUILD` is the first Dockerfile use case. Its child command is parsed with
  the registered handler for that command, not a second hard-coded parser.
- The generic parser does not itself decide which commands are permitted as
  children. That is an application policy.

## Result model

An instruction result should retain:

- instruction name and original raw statement;
- raw argument text;
- parsed flags and positional arguments where applicable;
- a representation/form marker when a dialect distinguishes word, shell,
  JSON, key/value, or raw payloads;
- zero or more nested instructions;
- source position and preceding comments;
- an opaque dialect-specific payload for language-specific extensions.

Implementations should avoid baking Dockerfile-only fields into the shared
contract. Lossless raw text takes precedence over inventing a common AST that
cannot represent other instruction languages.

## Error policy

- Lexical, handler, directive, and flag errors should carry the narrowest known
  source range.
- Errors should preserve the underlying cause for error inspection.
- Parser construction errors are distinct from errors encountered while
  advancing the input stream.
- The parser returns the first encountered error; applications may recover,
  skip, or stop according to their own policy.
