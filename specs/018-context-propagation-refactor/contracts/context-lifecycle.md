# Internal Contract: Context Lifecycle

**Bugfix**: 2026-08-21 — [BUG-001] Clarified player-listener acquisition-to-runtime ownership handoff.

## Required context

- Every constructor or operation that retains a context, derives a child, waits on cancellation, performs I/O, or crosses a context-aware adapter boundary receives a non-absent `context.Context`.
- Executable entry points may create a process root. Packages below those entry points do not replace an absent context with `context.Background()` or `context.TODO()`.
- A result-returning API reports an immediate error for an absent context. A pointer-only constructor treats an absent required context as a programmer error and fails immediately.

## Propagation

- `main` passes its process context into application composition, logger lookup, the application constructor, the player server, the desktop adapter, and the Wails lifecycle adapter.
- Wails startup passes its lifecycle callback context into the application runtime.
- The application passes its runtime context to desktop actions, public-access work, player startup, session operations, and the coordinator's durable command-state decision.
- The context supplied to player-server `Start` may abort acquisition before listener commit. After commit, normal completion or cancellation of that startup operation does not cancel the HTTP server base context; the committed server remains a child of the application/player-server owned lifetime.
- Connect handlers keep their request context for stream and request-aware work.
- Tests substitute `t.Context()` at the same roots and derive values, causes, or deadlines from it.

## Causal cancellation

- Manually owned lifetimes are created with `context.WithCancelCause`.
- Every explicit cancel call supplies a stable package-local error identifying the owner outcome.
- Parent cancellation retains the parent's cause when it happens first.
- Repeated cancellation and repeated close are idempotent; the first cause remains observable.

Minimum semantic outcomes:

| Lifetime | Required explicit outcome |
|---|---|
| Application runtime | normal application shutdown or startup failure |
| Player server | server shutdown or serve failure |
| Player subscription | unregister/owner close, replacement, stale revision, or queue overflow as applicable |
| Public-access start | completed handoff, timeout, request cancellation, stop, or reconfiguration |
| Embedded endpoint | aborted startup or endpoint close |
| Bounded cleanup | cleanup completed or cleanup budget expired |

Successful player-listener handoff is not a player-server cancellation outcome. Startup completion ends only the bounded acquisition operation; the committed server accepts new HTTP/ConnectRPC requests until an explicit documented server/application outcome occurs.

## Cleanup

- Cleanup that must survive an already-canceled initiator uses `context.WithoutCancel(parent)` only to detach cancellation; values remain inherited.
- The existing cleanup budget remains unchanged and has an explicit timeout cause.
- Cleanup never uses an unrelated root and never becomes unbounded.

## Verification contract

- Context propagation tests place a private marker in `t.Context()` and assert that the marker reaches the targeted fake dependency.
- Cancellation tests assert `context.Cause` against the expected semantic error and verify first-cause preservation under repeated close.
- Player-server handoff tests complete and cancel a bounded startup context, then establish a new `Subscribe`, receive its complete snapshot and a later update, and assert the committed server context remains uncanceled until explicit shutdown.
- The convention test rejects background/placeholder roots in affected tests and lower-layer production fallback creation.
