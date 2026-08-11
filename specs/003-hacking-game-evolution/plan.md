# Implementation Plan: Phase 1 Generation-Bound Hacking Patterns

**Branch**: `feature/003-hacking-game-evolution` | **Date**: 2026-08-11 | **Spec**: `specs/003-hacking-game-evolution/spec.md` | **Handoff**: `specs/003-hacking-game-evolution/planning-handoff.md`

## Summary

Correct the existing server-authoritative special-pattern implementation so every public identity is bound to one runtime puzzle generation and one rendered-row coordinate pair, every accepted activation follows the mandated weighted-selection and atomic publication order, and every rejected request leaves both canonical state and the outcome random source untouched. The final rendered board remains the sole authority for the initial `3–6` count, the player projection is reduced to identity, row, inclusive coordinates, and used status, and reconnect behavior continues through the current process-local live snapshot. Ordinary guesses and filler clicks remain unchanged, while `ForceHackSuccess` stays exclusively behind the existing private desktop/Wails boundary.

## Constitution Check

The pre-research gate passes. The Phase 1 corrections remain inside the existing modular-monolith boundaries and introduce no dependency, persistence, role, or presentation expansion.

| Principle | Assessment |
|---|---|
| I. Preserve Runtime Boundaries | PASS: discovery, generation-bound identity, outcome selection, and used history remain transport-independent in `internal/hack`; `internal/live` owns serialization and the atomic publication commit; `internal/player` owns strict WebSocket decoding and fanout; `client/` remains browser-only; and the existing Wails game-master path remains separate. |
| II. Keep Shared State Server-Authoritative | PASS: `HACK_PATTERN` remains an untrusted request. The live-service mutex validates generation and current coordinates, marks used state, selects and applies one outcome, creates a detached projection, and commits one ordered broadcast before releasing the transition. Browsers perform no optimistic mutation. |
| III. Protect Desktop and Public-Access Boundaries | PASS: strict unknown-field rejection is retained, the public pattern shape is reduced, and no player route, message, global, DOM control, shortcut, or query parameter gains access to `ForceHackSuccess`. |
| IV. Preserve Session Data Compatibility | PASS: generation IDs, patterns, used history, removed duds, attempts, logs, and outcomes remain process-local. Version-1 session JSON retains only the existing durable terminal `hackLevel`; no puzzle seed or unlocked state is added. |
| V. Match Established Code Conventions | PASS: the work extends existing Go aggregates, injectable seams, mutex-protected services, uppercase snake-case messages, camelCase JSON, and browser JavaScript conventions. No runtime dependency is added. |

The post-design re-check uses the same assessments: the data model and contract below introduce no constitutional violation, so no Complexity Tracking table is required.

## Project Structure

```text
specs/003-hacking-game-evolution/
├── spec.md                                  # Normative clarified requirements
├── planning-handoff.md                     # Mandatory no-loss planning guardrails
├── plan.md                                  # Corrective implementation plan
├── research.md                              # Superseding design decisions and rejected alternatives
├── data-model.md                            # Generation, identity, state, projection, and transition model
└── contracts/
    └── hacking-interface.md               # Strict `HACK_PATTERN` and public-state contract

internal/
├── domain/
│   ├── model.go                             # Generation-bound private identity and minimal public pattern model
│   └── model_test.go                        # Version-1 JSON remains free of runtime hacking state
├── hack/
│   ├── hack.go                              # Final-board regeneration, discovery, identity, outcomes, projections
│   └── hack_test.go                         # 100-value mapping, RNG consumption, dynamic identity, generation bounds
├── live/
│   ├── service.go                           # Generation issuance and nine-step mutex-protected activation/publication
│   └── service_test.go                      # Stale-generation, duplicate, ordering, projection-isolation coverage
├── player/
│   ├── protocol.go                          # Strict generation-bearing opaque `patternId` input
│   ├── protocol_test.go                     # Exact-field and public-envelope contract coverage
│   ├── server.go                            # Publication callback committed during the live transition
│   └── server_test.go                       # Multi-client convergence, rejection silence, reconnect, stale puzzle tests
├── platform/
│   └── assets_test.go                       # Browser pattern contract and absence of player solve authority
└── testutil/testdata/protocol/
    └── hack-state.json                      # Minimal generation-bound public pattern fixture

client/client.js                              # Row-coordinate hover/click mapping and opaque `patternId` submission
app_test.go                                   # Detached public pattern fixtures and private GM solve regression
frontend/src/index.html                       # Existing private GM solve control, verified unchanged
frontend/src/master.js                        # Existing Wails invocation and eligibility, verified unchanged
```

**Structure Decision**: Correct the established `internal/hack` → `internal/live` → `internal/player` → `client/` path in place. Keep generation and discovery in the hacking domain, serialization and publication commit in the live service, protocol ownership in the player server, and the trusted Wails override in its existing separate boundary; add no package, route, persisted field, dependency, or role model.

## Implementation Strategy

1. Replace coordinate-only string identities with a private comparable identity containing generation, flattened rendered-row ordinal, and row-local inclusive start/end offsets. Issue a new non-persisted generation ID for every fresh puzzle without consuming the pattern-outcome random source.
2. Make board construction an attempt inside a regeneration loop. Run the production discovery function on each final rendered board and publish only a board whose discovered count is within `3–6`; intended insertions remain a construction technique, not the acceptance authority.
3. Change pattern activation so every accepted request is marked used, consumes exactly one `Intn(100)` outcome value, maps `0..79` to dud removal and `80..99` to restoration, then applies restoration as the fallback if a selected dud removal has no target. A dud-selection draw remains separate and occurs only when removal has an eligible target.
4. Reduce public pattern JSON to opaque `id`, `row`, inclusive `start`/`end`, and `used`. Recompute current spans from canonical board text after mutation and retain complete private used history so a rediscovered coordinate pair stays unavailable.
5. Keep strict `HACK_PATTERN` decoding with one opaque `patternId`. The ID contains or resolves to the complete generation-bound identity; the browser treats it as opaque and uses the explicit row coordinates only for rendering.
6. Commit the detached `HACK_STATE` publication callback as step nine while the live-service mutex is still held. The callback may only enqueue the detached payload to the player fanout and game-master event paths and must not call back into live state; actual socket writes remain owned by `internal/player`.
7. Add deterministic domain, live-service, protocol, multi-client, projection-mutation, private-boundary, and version-1 persistence regressions before changing task status. Retain the existing ordinary guess and filler-click tests unchanged as behavioral sentinels.

## Verification Gates

- `gofmt -l .` reports no changed Go files.
- `go vet ./...` succeeds.
- `go test ./...` succeeds, including the controlled 100-value probability mapping, 1,000 final-board checks, stale-generation rejection, zero-RNG rejection paths, atomic duplicate publication, reconnect convergence, detached projection mutation, and version-1 session compatibility.
- `go test -race ./...` succeeds for concurrent duplicate pattern activation and publication ordering.
- `npm --prefix frontend run build` succeeds without adding or changing a player-accessible solve path.
- Browser asset-contract tests confirm row-based inclusive highlighting, opaque generation-bound submission, unchanged word/filler clicks, and absence of `ForceHackSuccess` from all player assets.
