# Implementation Plan: Hacking Game Evolution

**Branch**: `feature/003-hacking-game-evolution` | **Date**: 2026-08-11 | **Spec**: `specs/003-hacking-game-evolution/spec.md`

## Summary

Replace every player-accessible administrator shortcut with server-authoritative, one-use special patterns while preserving normal word guesses and the game master's existing `ForceHackSuccess` control. Extend the private Go hacking aggregate with coordinate-identified pattern state, generate exactly `3–6` initially valid patterns, apply the exact `80%` dud-removal and `20%` attempt-restoration outcomes under the live-service mutex, and publish only a detached public projection. The browser will derive highlighting and input from that projection and send a typed pattern request without changing puzzle state optimistically.

## Constitution Check

The pre-research gate and the post-design re-check both pass; the final data and protocol designs keep the change inside the existing modular-monolith boundaries.

| Principle | Assessment |
|---|---|
| I. Preserve Runtime Boundaries | PASS: pattern discovery, validation, and effects remain transport-independent in `internal/hack`; `internal/live` owns canonical mutation; `internal/player` owns WebSocket validation; and `client/` remains browser-only. The existing `app.go` game-master bridge is retained rather than exposed to players. |
| II. Keep Shared State Server-Authoritative | PASS: `HACK_PATTERN` is a request, the live-service mutex atomically validates and consumes the pattern, and all clients converge on `HACK_STATE` or the current `TERMINAL_LIVE` snapshot. Secret candidates remain private. |
| III. Protect Desktop and Public-Access Boundaries | PASS: no new desktop method, filesystem access, environment access, public route, credential path, or external URL behavior is introduced. Player payloads are strictly decoded before mutation. |
| IV. Preserve Session Data Compatibility | PASS: special-pattern availability, used identities, removed duds, attempts, and outcomes remain process-local and are not added to version-1 session JSON. A fresh live broadcast creates a fresh puzzle. |
| V. Match Established Code Conventions | PASS: the design uses existing Go models, injectable randomness, mutex-protected live services, uppercase snake-case message names, camelCase JSON fields, and browser JavaScript conventions. No dependency is added. |

## Project Structure

```text
specs/003-hacking-game-evolution/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
└── contracts/
    └── hacking-interface.md

app.go                                      # Preserve GM solve publication; deep-clone added public pattern state
app_test.go                                 # Protect the GM override and detached desktop status projection
internal/
├── domain/model.go                         # Private/public special-pattern models; remove administrator-only fields
├── hack/
│   ├── hack.go                             # Exact-count generation, discovery, activation, dud removal, attempt restore
│   └── hack_test.go                        # Deterministic generation, parsing, outcomes, dynamic/stacked/no-op cases
├── live/
│   ├── service.go                          # Atomic pattern action under the canonical live-state mutex
│   └── service_test.go                     # Concurrent single-consumption, reset, and detached-snapshot coverage
├── player/
│   ├── protocol.go                         # Add `HACK_PATTERN`; remove `HACK_ADMIN`
│   ├── protocol_test.go                    # Strict payload and public-envelope contract coverage
│   ├── server.go                           # Dispatch accepted pattern requests and broadcast shared state
│   └── server_test.go                      # Multi-client convergence and reconnect coverage
├── platform/assets_test.go                 # Player interaction contract and absence of former cheats
└── testutil/testdata/protocol/
    └── hack-state.json                     # Golden public pattern projection

client/client.js                            # Pattern hover, inclusive range highlight, and typed pattern request
frontend/src/index.html                     # Existing game-master solve control retained as the trusted override
frontend/src/master.js                      # Existing `forceHackSuccess` eligibility and dispatch retained
```

**Structure Decision**: Extend the established domain → live service → player protocol → browser path, with `app.go` and the Wails master UI remaining the separate trusted game-master path; no new package, persisted schema, route, or dependency is needed.
