# Implementation Plan: Context Propagation and Causal Cancellation

**Bugfix**: 2026-08-21 — [BUG-001] Updated from bugfix patch.

## Summary

Refactor application-owned Go lifetimes so context values and cancellation travel from the process root or initiating request to every context-aware boundary. Composition and lifecycle constructors will require a context, the coordinator persistence seam will carry the application context instead of manufacturing a background root, and manually canceled resource contexts will use causal cancellation with stable reasons. Tests will pass `t.Context()` or a direct derivative and will assert both context propagation and cancellation causes.

## Project Structure

```text
main.go                                      # process root, composition, persistence adapter
app.go                                       # application runtime context and shutdown ownership
wails_host.go                                # Wails lifecycle root/cleanup context
internal/control/service.go                  # context-aware durable command-state seam
internal/session/service.go                  # required context validation
internal/player/
├── handler.go                               # request-context use
├── server.go                                # server lifetime and shutdown cause
└── stream.go                                # subscription lifetime and close cause
internal/platform/
├── desktop.go                               # required retained lifecycle context
├── keychain_darwin.go                       # required operation context validation
└── test_conventions_test.go                 # test-context regression gate
internal/playerconfig/service.go             # required operation context validation
internal/tunnel/
├── manager.go                               # public-access operation and cleanup ownership
├── ngrok.go                                 # embedded endpoint lifetime and cleanup
└── public_ingress.go                        # request-derived server base and close context
*_test.go and internal/**/*_test.go           # t.Context roots, propagation/cause assertions
```

**Structure Decision**: Keep ownership changes inside existing composition, service, adapter, and test files; introduce no package or dependency and do not alter generated or serialized contracts.

## Constitution Check

| Principle | Assessment | Evidence |
|---|---|---|
| I. Govern the Accepted Desktop Runtime | PASS | Root composition remains in `main.go`; Wails stays confined to root/platform adapters; shutdown remains bounded and releases owned resources. |
| II. Make Protobuf the Application Contract Source of Truth | PASS | No structured application contract changes; only native non-serializable context values and internal errors change. |
| III. Use ConnectRPC and Keep State Server-Authoritative | PASS | Connect handlers remain the player boundary and continue to invoke the canonical coordinator; request context handling does not add another transport or state owner. |
| IV. Separate Public and Private Capabilities | PASS | Context propagation does not expose private services, secrets, or cancellation diagnostics over the public player contract. |
| V. Evolve Schemas Safely and Reproducibly | PASS | No protobuf schema or generated output changes. |
| VI. Preserve Portable Session JSON Version 1 | PASS | Session data and adapters are unchanged; only the context supplied to existing persistence work changes. |
| VII. Complete Cutovers and Remove Superseded Protocols | PASS | No protocol coexistence or migration is introduced. |

Additional gates: affected tests use Testify and `t.Context()`; concurrency-sensitive application, player, session, and tunnel suites run under the race detector; public-access cancellation, bounded cleanup, stale completion, and secret-redaction behavior remain covered.

## Phase 0: Research

Research decisions are recorded in [research.md](./research.md). The load-bearing findings are that only executable entry points should establish new roots, manual ownership cancellation needs stable causes, detached cleanup must preserve parent values, and only the coordinator path that performs durable session work needs a new context parameter.

## Phase 1: Design

The ownership model and state transitions are defined in [data-model.md](./data-model.md). The internal callable contract, required-context rules, and stable cancellation outcomes are defined in [contracts/context-lifecycle.md](./contracts/context-lifecycle.md).

The design changes no public RPC, protobuf, persistence, browser, or desktop payload contract. Context values remain native dependency/lifecycle inputs as required by the constitution.

### BUG-001 Player-listener ownership handoff

The context passed to player-server `Start` bounds acquisition only. Before listener commit, its cancellation aborts startup and releases partial resources. After commit, the HTTP server base context is owned by the application/player-server lifetime established at composition; normal startup-operation completion is not a server cancellation cause. Explicit stop, serve failure, parent application cancellation, and application shutdown retain their existing causal and bounded-cleanup behavior.

## Verification Strategy

- Add focused propagation tests at composition/lifecycle, coordinator persistence, player server/subscription, and tunnel/provider boundaries.
- Add a real player-server regression with the production non-zero startup timeout: complete `App.Start`, cancel the successful acquisition operation, then prove a later `Subscribe` receives a complete snapshot and subsequent update while the server context remains active until explicit shutdown.
- Assert explicit causes with `context.Cause` and idempotent first-cause preservation.
- Run the repository convention scan to prove affected test contexts derive from `t.Context()` and production lower layers do not create replacement roots.
- Run `gofmt -l .`, `go vet ./...`, `go test ./...`, and `go test -race ./...`.
- Run the existing secret-leak and context-sensitive public-access checks when the focused suites are green; no frontend or schema build is required because their inputs and outputs are unchanged.

## Post-Design Constitution Re-check

PASS. The final design keeps Wails at the adapter boundary, keeps context native and non-serialized, preserves the single authoritative player/coordinator state owner, changes no public/private capability surface, changes no schema or session JSON, introduces no protocol coexistence, and explicitly includes governed test and race gates.
