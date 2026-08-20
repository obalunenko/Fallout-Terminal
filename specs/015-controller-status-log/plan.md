# Implementation Plan: Immersive Controller Status Log

**Branch**: `develop` | **Date**: 2026-08-20 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `specs/015-controller-status-log/spec.md`

## Summary

Replace the player client's framed identity/role panel with a low-priority ROBCO system line anchored in the lower terminal chrome immediately above the prompt. The browser will derive the compact input label and terminal-native role wording from the existing authoritative player projection, preserve the established DOM identifiers and live-status semantics, and update affected Playwright journeys to verify the new exact copy, role transitions, and non-overlapping placement. No RPC, protobuf, persistence, Overseer, or server-authority change is required.

## Project Structure

```text
frontend/client/
├── index.html                                  # move and reshape the identity status markup
├── client.css                                  # lower-chrome layout and terminal-native styling
└── client.js                                   # derive compact input and role presentation

tests/browser/
├── terminal-navigation.spec.mjs               # focused copy, placement, role-change assertions
├── state-changing-command-approval.spec.mjs   # canonical active-role wording
├── state-changing-command-sync.spec.mjs       # canonical active-role wording
└── player-sessions-control.spec.mjs            # canonical active-role wording

scripts/
└── state-changing-reset-native-player-smoke.mjs # native smoke role assertion
```

**Structure Decision**: Keep the change inside the existing player-client presentation boundary and its current browser/native-smoke verification surfaces; reuse authoritative session state and existing stable DOM identifiers instead of creating a parallel status component or transport contract.

## Constitution Check

| Principle | Assessment |
|---|---|
| I. Govern the Accepted Desktop Runtime | PASS — the change stays inside `frontend/client/`, uses browser APIs only, and does not introduce Wails or desktop access. |
| II. Make Protobuf the Application Contract Source of Truth | PASS — no structured application contract changes; the line is a local projection of existing generated player state. |
| III. Use ConnectRPC and Keep State Server-Authoritative | PASS — role, character, and fallback identity continue to come exclusively from the authoritative stream; the client adds presentation formatting only. |
| IV. Separate Public and Private Capabilities | PASS — no private Overseer capability or privileged data is added to the public player client. |
| V. Evolve Schemas Safely and Reproducibly | PASS — no schema or generated-file changes are planned. |
| VI. Preserve Portable Session JSON Version 1 | PASS — no persisted data or adapter changes are planned. |
| VII. Complete Cutovers and Remove Superseded Protocols | PASS — the old framed presentation is removed in the same change; there is no dual presentation path. |

### Applicable Gates

- **Contract and generation**: no protobuf, RPC, Wails binding, or generated-file change; generation gates are not applicable.
- **Compatibility**: retain existing player-state meanings and stable DOM identifiers used by browser journeys; update only intentional visible wording assertions.
- **Public/private boundary**: player-only presentation; no Overseer or desktop bridge changes.
- **Secret handling and provider lifecycle**: not applicable.
- **Build ownership**: use the existing frontend workspace command `npm run build:client --prefix frontend`.
- **Cutover**: remove superseded framed/badge styling and assertions; do not retain a hidden legacy rendering.
- **Verification**: run focused Playwright journeys plus the player-client production build, then broaden to the full browser suite when its fixture environment is available.

## Phase 0: Research

The design decisions and rejected alternatives are recorded in [research.md](./research.md).

## Phase 1: Design & Contracts

- [data-model.md](./data-model.md) defines the derived player-status-line view and its state transitions.
- [contracts/player-status-line.md](./contracts/player-status-line.md) defines the visible copy, stable identifiers, accessibility semantics, and layout contract consumed by browser tests.

## Implementation Strategy

1. Reshape and relocate the existing identity section into lower terminal chrome while retaining stable element identifiers.
2. Derive a compact input-channel label and terminal-native role text from the existing authoritative player state, omitting absent character segments.
3. Replace framed, coloured badge styling with a dim, wrapping single-line log treatment that remains a fixed flex child above the prompt.
4. Update role assertions and add focused checks for exact active copy, authority changes, and lower-chrome geometry.
5. Build the player client and run the affected browser journeys; record any environment-dependent full-suite checks honestly.

## Post-Design Constitution Re-check

PASS — Phase 1 keeps every field derived from the existing authoritative player projection, retains stable test/accessibility identifiers, removes the superseded framed rendering in the same cutover, and adds no contract, generated file, dependency, persistence, privilege, or authority boundary. No Complexity Tracking exception is required.
