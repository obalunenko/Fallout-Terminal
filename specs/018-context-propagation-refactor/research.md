# Research: Context Propagation and Causal Cancellation

## Decision 1: Keep root-context creation at executable boundaries

**Decision**: `main`, the repository build command, and the browser fixture executable remain the only places that may establish a new process root. Application composition accepts the root explicitly and passes it to constructors and loggers. Request-aware boundaries keep the request or application lifecycle context they already receive.

**Rationale**: A context represents ownership. Lower layers cannot reconstruct the caller's lifetime, values, or cancellation signal by creating a new background root.

**Alternatives considered**:

- Preserve defensive background fallbacks in services. Rejected because they hide incorrect callers and detach work.
- Store a package-global root. Rejected because it obscures ownership and makes tests interfere with each other.

## Decision 2: Make required contexts explicit at construction and operation boundaries

**Decision**: Context-owning constructors and operations accept a required context and fail or panic immediately on an absent value according to their existing return shape. Application and player constructors receive contexts directly; result-returning operations report an error rather than starting detached work.

**Rationale**: Go's context contract prohibits passing an absent context. Explicit boundaries make invalid ownership visible close to the caller and remove silent fallback behavior.

**Alternatives considered**:

- Add an optional context field to configuration structs. Rejected because optionality recreates the absent-context ambiguity.
- Keep old constructor signatures and use placeholder contexts. Rejected because it does not establish a real owner.

## Decision 3: Use causal cancellation for manually owned lifetimes

**Decision**: Application runtime, player server, player subscription, public-access start, and embedded-provider lifetime contexts use `context.WithCancelCause`. Every manual cancellation supplies a stable package-local cause for shutdown, replacement, overflow, successful handoff, failed startup, or abort. Parent cancellation is allowed to win naturally when it occurs first.

**Rationale**: The first cancellation cause is stable and observable with `context.Cause`, which makes concurrent lifecycle outcomes diagnosable without changing public contracts.

**Alternatives considered**:

- Keep bare cancellation and infer intent from surrounding state. Rejected because the state may already have changed and cannot explain races reliably.
- Export cancellation causes over RPC. Rejected because this is internal lifecycle evidence and public error disclosure is out of scope.

## Decision 4: Detach only cancellation for bounded cleanup

**Decision**: Cleanup that must run after its initiator is canceled derives from `context.WithoutCancel(parent)` so values survive, then applies the existing timeout with an explicit timeout cause. Normal early completion cancels any owned causal child with a cleanup-complete reason.

**Rationale**: Resource release must not be skipped merely because application/request work was canceled, but it must remain bounded and retain diagnostic context values.

**Alternatives considered**:

- Use a fresh background context. Rejected because it loses values and ownership lineage.
- Use the canceled parent directly. Rejected because cleanup would fail immediately and leak owned resources.

## Decision 5: Thread persistence context through the coordinator boundary

**Decision**: The coordinator command-execution decision accepts the application context and forwards it through `CommandStateStore` to session persistence. The desktop application supplies its lifecycle context; tests supply `t.Context()`.

**Rationale**: This is the existing call chain where a context-aware durable operation is currently reached through an adapter-created background context. Adding context to the narrow synchronous seam preserves real ownership without making pure in-memory domain operations context-aware.

**Alternatives considered**:

- Store the root context inside the adapter. Rejected because a per-operation context is available and more precise.
- Add context to every coordinator method. Rejected because pure synchronous state transitions do not need one and doing so would dilute the ownership signal.

## Decision 6: Enforce test ownership with both behavior and scanning

**Decision**: Every affected test passes `t.Context()` into constructors and operations; manually canceled test contexts derive with `context.WithCancelCause(t.Context())` and supply a test-specific reason. Existing AST convention coverage is expanded where needed to include the affected files and forbidden patterns.

**Rationale**: Behavioral cause assertions prove semantics, while the convention scan prevents regressions to background or placeholder roots.

**Alternatives considered**:

- Rely only on review. Rejected because this cross-cutting rule is mechanically detectable.
- Use a shared test background context. Rejected because it can outlive the test and conceal leaks.
