# Design-pattern review

This architecture was selected after reviewing the pattern catalogs at
[Go Patterns](https://go-patterns.dev/) and
[Refactoring.Guru's Go design-pattern catalog](https://refactoring.guru/design-patterns/go),
including the [Go examples repository](https://github.com/RefactoringGuru/design-patterns-go).
The goal is not to force every named pattern into the code. A parser is a
small, sequential data-processing system; patterns are useful only where they
reduce coupling or make extension behavior explicit.

## Go Patterns catalog

The Go Patterns catalog groups concurrency/generative patterns, creational
patterns, stability patterns, and database patterns. The following is the
implementation decision for every listed item.

### Generative and concurrency patterns

| Pattern | Decision | Reason |
| --- | --- | --- |
| Generator | Used conceptually by `Parser.Next`. | Each call lazily produces one instruction; no goroutine or channel is needed for a synchronous parser. |
| Fan-In | Not used in the parser core. | Combining multiple source streams is a caller-level concern and changes source-order semantics. |
| Fan-Out | Not used in the parser core. | One source is parsed in strict order; downstream work may fan out after an instruction is emitted. |
| Pipeline | Used as package boundaries, not as concurrent channels. | Physical reading, continuation, comments/directives, instruction splitting, flags, and handlers are replaceable stages. |
| Mutex | Not used. | A parser instance is single-consumer and not concurrently mutated. A future shared immutable registry can be safely copied at construction. |
| Semaphore | Not used. | Parsing does not create a bounded set of parallel tasks. |
| Singleflight | Not used. | The parser performs no shared remote or expensive deduplicated lookup. |
| Worker Pool | Not used. | Instruction order and parser state are sequential; worker pools belong downstream. |
| Queuing | Not used. | `Next` supplies backpressure naturally by not reading the next instruction until requested. |
| Parallel For Loop | Not used. | One instruction stream cannot be parallelized without losing ordered lexical state. |
| Map-Reduce | Not used. | Aggregation and whole-document analysis are consumer-level operations. |
| Future / Promise | Not used. | Handlers are synchronous and return a result or error directly. |
| Error Group | Not used. | There are no concurrent parser subtasks to coordinate. |

### Creational and configuration patterns

| Pattern | Decision | Reason |
| --- | --- | --- |
| Functional Options | Used. | `New`, `dockerfile.NewParser`, and `With...` options make registrations and policies composable without a long constructor. |
| Configuration Struct + Factory | Used internally. | Construction validates options into a private config and then creates a parser instance. |
| Configurable Object | Limited use. | Runtime lexical state such as the escape rune can change through a directive handler; public configuration is otherwise applied at construction. |
| Lazy Initialization | Not needed. | No expensive resource is required to instantiate parser registries. |
| Fluent Interfaces | Not used. | Functional options are clearer than mutable setter chaining for this API. |

### Stability patterns

| Pattern | Decision | Reason |
| --- | --- | --- |
| Retry | Not used. | Retrying deterministic syntax failures is incorrect; retrying I/O is a source adapter concern. |
| Timeout | Not used. | The generic parser accepts an `io.Reader`; cancellation and deadlines should be supplied by the source or caller. |
| Drop | Not used by default. | Silently dropping source lines would undermine parser correctness. A caller can implement explicit recovery around `Next`. |
| Circuit Breaker | Not used. | There are no remote dependencies or failing services in the parser core. |
| Caching with `map` and `sync.RWMutex` | Not used. | No repeated computation currently warrants a shared cache. |
| Caching with `sync.Map` | Not used. | No concurrent, dynamically populated cache exists. |
| Caching with automatic cleanup | Not used. | The parser owns no expiring cache. |

### Database pattern

| Pattern | Decision | Reason |
| --- | --- | --- |
| Unit of Work / Transaction Manager | Not used. | Parsing does not coordinate database writes. If a consumer persists parsed instructions transactionally, that belongs outside the parser. |

## Refactoring.Guru Go catalog

### Creational patterns

| Pattern | Decision | Reason |
| --- | --- | --- |
| Abstract Factory | Not used. | There is no family of interdependent product objects; handler registries and small factories suffice. |
| Builder | Not used in the public API. | Functional options provide a lighter construction DSL. |
| Factory Method | Used in a Go-idiomatic form. | `dockerfile.NewParser` creates a generic parser preconfigured with a Dockerfile dialect. |
| Prototype | Not used. | Parser instances are constructed from options; cloning mutable streams would be unsafe and ambiguous. |
| Singleton | Deliberately avoided. | Global mutable registries would prevent isolated dialects and per-parser customization. |

### Structural patterns

| Pattern | Decision | Reason |
| --- | --- | --- |
| Adapter | Supported by interfaces. | A custom `LineReaderFactory` adapts an alternate reader or lexer to the parser contract. |
| Bridge | Used architecturally. | The generic parser engine and the Dockerfile dialect are separate abstractions that vary independently. |
| Composite | Used for nested instructions. | `Instruction.Children` represents `ONBUILD`'s nested instruction tree. |
| Decorator | Used. | `WithInstructionMiddleware` wraps one registered instruction handler without changing the handler implementation. |
| Facade | Used. | The `dockerfile` package supplies a focused constructor over lexer, generic parser, directives, handlers, and default flags. |
| Flyweight | Not needed. | There is no measured memory bottleneck from repeated immutable objects that justifies shared mutable state. |
| Proxy | Not needed. | There is no remote or protected parser service requiring access mediation. |

### Behavioral patterns

| Pattern | Decision | Reason |
| --- | --- | --- |
| Chain of Responsibility | Not used for instruction dispatch. | Exact normalized-name lookup is deterministic and easier to inspect than a chain of handlers that each decline unrelated names. Middleware remains composable. |
| Command | Used. | Each registered instruction handler is a first-class function that represents one instruction parse operation. |
| Iterator | Used. | `Next` exposes ordered traversal without exposing the reader's internal state. |
| Mediator | Not used as a separate object. | The parser context coordinates lexical services and nested dispatch; a second mediator would add indirection without reducing dependencies. |
| Memento | Not used. | Rollback of parser stream position is not supported; callers can retain returned results instead. |
| Observer | Used narrowly. | Comment handlers observe comments independently of instruction parsing. A general event bus is intentionally avoided. |
| State | Used implicitly. | The preamble and escape token are parser state; explicit fields and directive handlers are simpler than a full state-object hierarchy. |
| Strategy | Used. | Comment parsing, directive parsing, continuation, instruction splitting, flag parsing, and instruction handlers are replaceable functions. |
| Template Method | Not used. | Instruction grammars are intentionally independent; a common rigid template would constrain custom dialects. |
| Visitor | Not used. | Consumers can traverse the simple instruction tree directly; a visitor framework would be premature. |

## Guiding rule

Use interfaces for meaningful boundaries, function types for replaceable
algorithms, and ordinary structs for parsed data. Prefer explicit registration
over global initialization, and prefer a small helper over a named pattern when
the helper already solves the problem.
