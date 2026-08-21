# Implementation Plan: Overseer Action Clarity

**Branch**: `013-overseer-action-clarity` | **Date**: 2026-08-20 | **Spec**: [spec.md](./spec.md)

## Summary

Reorganize the Overseer terminal controls so authored-session actions, live selection, content publication, and broadcast-state actions each live beside the state they affect. Reuse the existing activation, live-update, and clear commands; add accessible local dialogs for named terminal creation and mandatory take-off-air confirmation, and move full active-terminal reapplication into an explained secondary menu. Preserve every backend, persistence, player, and generated contract.

## Project Structure

```text
frontend/overseer/src/
├── index.html              # contextual action placement, menus, and dialogs
├── overseer.css            # responsive action groups and dialog/menu presentation
└── overseer.js             # visibility, focus, validation, confirmation, command orchestration

internal/platform/
└── assets_test.go          # static labels, ownership, accessibility, and separation contracts

tests/browser/
├── fixture-server/main.go
├── fixtures/desktop-bindings.js
└── overseer-terminal-actions.spec.mjs
```

**Structure Decision**: Keep the change inside the established Overseer source and test fixtures; do not introduce a component framework, new dependency, Wails method, player route, or persistence field.

## Constitution Check

| Principle or gate | Assessment | Evidence |
|---|---|---|
| I. Govern the Accepted Desktop Runtime | PASS | Overseer continues to call only the existing narrow desktop service. |
| II. Protobuf Contract Source of Truth | PASS | No structured boundary or serialized contract changes. |
| III. Server-Authoritative State | PASS | Activation, publication, and clear remain coordinator-owned commands; UI state follows returned coordination state. |
| IV. Separate Public and Private Capabilities | PASS | All changed actions remain private to Overseer and absent from the player surface. |
| V. Safe and Reproducible Schemas | PASS | No schema or generated-code change. |
| VI. Preserve Session JSON Version 1 | PASS | Terminal creation uses the existing session model and autosave behavior without changing shape. |
| VII. Complete Cutovers | PASS | Superseded labels and ambiguous active-action presentation are removed in the same change. |
| Dependency rules | PASS | No dependency, package, runtime, or generated-binding change. |
| Secret and credential governance | PASS | No secret-bearing surface is touched. |
| Go development tool ownership | PASS | Existing build and browser-test entrypoints remain canonical. |
| Testing and quality gates | PASS | Static UI contracts, Playwright journeys, Overseer production build, Go tests, and native build are planned. |
| Development workflow | PASS | Specification and clarifications precede design, tasks, implementation, and validation. |

The Phase 1 design introduces no constitutional exception, so no Complexity Tracking table is required.

## Phase 0: Research

The decisions and rejected alternatives are recorded in [research.md](./research.md).

## Phase 1: Design and Contracts

- [data-model.md](./data-model.md) defines the unchanged durable entities plus the UI-only action and dialog states.
- [contracts/ui-actions.md](./contracts/ui-actions.md) defines exact visible copy, contextual ownership, accessibility behavior, and existing command mapping.
- No API, protobuf, persistence, WebSocket, or HTTP contract artifact is required because none changes.

## Verification Strategy

- Extend static resource assertions to reject superseded labels and require the accepted labels, dialog semantics, action ownership, and non-generic privileged calls.
- Add a focused Playwright journey covering named creation and cancellation, active/inactive visibility, secondary reapplication, publication, mandatory take-off-air confirmation, unfinished-progress handoff, errors, duplicate-click prevention, Escape behavior, and focus restoration.
- Extend the browser fixture only where authoritative coordination results are needed; it must continue recording the exact existing desktop calls and payloads.
- Run JavaScript syntax checks, the Overseer production build, focused and full browser tests, `gofmt -l .`, `go vet ./...`, `go test ./...`, and `go run ./cmd/build build`.

## Post-Design Constitution Re-check

PASS. The final design remains UI-local, retains explicit private command boundaries, preserves server authority and version-1 persistence, introduces no dependency or schema, and adds proportionate browser/static/native verification.
