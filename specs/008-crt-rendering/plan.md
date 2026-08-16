# Implementation Plan: CRT Rendering and Motion Effects

**Branch**: `008-crt-rendering` | **Date**: 2026-08-16 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/008-crt-rendering/spec.md`

**Bugfix**: 2026-08-16 — BUG-001 Updated from bugfix patch

**Bugfix**: 2026-08-17 — BUG-002 Updated from bugfix patch

## Summary

Preserve the historical Fallout-style CRT shell, palette, scanlines, vignette, six-second flicker, hard-step indicators, and 40ms ordered row/line reveal in the current player client. Add a browser-local reveal controller so any key pressed during an active reveal completes the visible page within 100ms and is consumed before navigation, activation, paging, back, or hacking handling; a later physical key press behaves normally. Persistent CRT effects remain always on, with no player-facing or operating-system-driven reduced-motion branch, and the feature changes no RPC, authoritative state, persistence, Wails bridge, or dependency.

BUG-001 extends the same browser-local lifecycle to the first presentation of a hacking generation: complete code rows reveal at the 40ms cadence in deterministic DOM source order, unchanged hacking updates and fit work do not replay them, replacement generations cancel stale work, and a skip key completes the board without also becoming hacking input.

BUG-002 separates stable puzzle-generation identity from mutable board content. An authoritative dud-removal update reconciles only the affected row content and used-pattern/log presentation, preserves unaffected row nodes and active reveal progress, and never starts a second whole-board reveal; only a different generation replaces the board.

## Project Structure

The implementation stays within the existing player presentation and its established verification surfaces.

```text
client/
├── index.html                         # Existing CRT shell and decorative-layer semantics
├── client.css                         # Historical palette, effects, keyframes, and responsive layout
└── client.js                          # Reveal controller, cancellation, and consumed-key guard
internal/platform/
└── assets_test.go                     # Source and embedded-player asset contract checks
tests/browser/
├── crt-rendering.spec.mjs             # CRT, reveal, keyboard, safety, and viewport journeys
├── crt-rendering.spec.mjs-snapshots/  # Approved historical-color visual baselines
└── fixture-server/
    └── main.go                        # Deterministic CRT content and transition fixtures
specs/008-crt-rendering/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── crt-presentation.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

**Structure Decision**: CRT behavior is browser-local presentation owned by `client/`. Go changes are limited to static-asset assertions and deterministic browser fixtures; no domain, transport, persistence, master UI, generated contract, tunnel, or application-composition path changes.

## Constitution Check

The gate passes before research and remains passed after the Phase 1 design.

| Principle | Status | Assessment |
|---|---|---|
| I. Govern the Accepted Desktop Runtime | PASS | The player remains browser-only under `client/`; no Wails, native, filesystem, lifecycle, or composition capability is added. |
| II. Make Protobuf the Application Contract Source of Truth | PASS | No application-owned structured contract changes; existing generated player projections remain the only network inputs. |
| III. Use ConnectRPC and Keep State Server-Authoritative | PASS | Reveal progress and consumed-key bookkeeping are transient presentation state and cannot mutate navigation, hacking, identity, or broadcast state. |
| IV. Separate Public and Private Capabilities | PASS | The restrictive CSP, same-origin assets, literal authored-content rendering, and player/private capability boundary remain intact. |
| V. Evolve Schemas Safely and Reproducibly | PASS | No Protobuf schema, generated file, field, package, or compatibility baseline changes. |
| VI. Preserve Portable Session JSON Version 1 | PASS | No session or player-configuration field is added; reveal state is never persisted or streamed. |
| VII. Complete Cutovers and Remove Superseded Protocols | PASS | Historical Electron-era presentation is used only as an experiential visual baseline; the active implementation stays in the accepted Wails v3 architecture. |

## Technical Context

**Runtime**: Go 1.26 modular monolith with the pinned Wails v3.0.0-beta.8 desktop runtime; browser ECMAScript modules, HTML, and CSS in the separately embedded player client.

**Existing dependencies**: Vite 8.1.5 for the player build, Playwright 1.62.1 for browser verification, and generated ConnectRPC/Protobuf player clients. No dependency or version change is required.

**State ownership**: Existing authoritative projections choose the visible player state and puzzle generation. Reveal timers, active reveal controllers, generation-local board snapshots, and consumed-key repeat suppression are ephemeral per browser tab and are cleared on completion, generation replacement, key release, or terminal teardown. Same-generation board mutations reconcile against the local snapshot without becoming a new reveal identity.

**Performance and viewport targets**: Keep the 40ms row/line cadence; complete 25 uninterrupted rows within 1.2 seconds; reveal hacking rows at the same cadence; complete all remaining visible rows or lines, including hacking code rows, within 100ms of a skip key; remain operable without page scrolling at 360×640, 768×720, and 1440×900.

**Security constraints**: Continue constructing authored rows and lines with `textContent` or the existing escaping boundary, preserve the restrictive CSP and same-origin resource policy, and keep optional sound failures isolated from rendering and input.

## Design

### Historical CRT shell

The current `.screen`, `.scanlines`, `.vignette`, `.blink`, selection, focus, active, and hacking-highlight rules already contain the intended historical treatment. The implementation protects these declarations instead of redesigning them. The six-second `flicker` keyframes remain opacity 1 through 96%, .92 at 97%, 1 at 98%, .96 at 99%, and 1 at 100%. Scanlines and vignette remain full-screen, pointer-transparent, and explicitly decorative.

### Reveal lifecycle

`revealInto` will own an explicit container-scoped controller rather than only an opaque timer. The controller retains the ordered nodes, next index, generation, pending timer, and terminal state, and exposes idempotent complete and cancel operations. Completing appends every remaining node synchronously in source order and unregisters the controller; cancelling invalidates the generation before clearing or replacing the container so a stale callback cannot append.

Content identity continues to decide whether a reveal is new. Unchanged authoritative updates, responsive repagination, font fitting, and navigation among pages of already-opened text render immediately. A newly opened folder, record, or command identity starts its own reveal; skipping one identity does not persist a preference or suppress a later identity.

~~For BUG-001, the authoritative hacking generation plus rendered board text forms one reveal identity.~~ BUG-002 narrows hacking reveal identity to the authoritative generation because dud removal mutates rendered board text without replacing the puzzle. The board is constructed through safe DOM boundaries as complete row units and revealed in deterministic DOM source order. Only fully appended rows enter the interaction surface. Attempt, activity-log, hover, reconnect, viewport, font-readiness, `scheduleHackFit()`, and same-generation dud-removal updates preserve active or completed progress; only a different generation cancels the old controller before the replacement reveal starts.

The client retains a generation-local board snapshot and reconciles changed rows by stable column/row coordinates. For a revealed row, reconciliation updates the existing row's affected cell content and target metadata without replacing the row or any unaffected row. For a pending row, it updates the controller's queued row descriptor so the authoritative content appears at the original reveal position and remains noninteractive until appended. Used-pattern state and the activity log update normally, stale hover is cleared, and no new controller is registered.

### Consumed-key guard

A capture-phase key guard runs before the existing normal terminal keyboard handler. If the visible page has an active reveal, any key completes all reveal controllers belonging to that page, calls `preventDefault`, stops further handling for that event, and records the consumed physical key until key release so auto-repeat from the same press cannot navigate or activate content. Another physical key press after completion follows existing keyboard behavior; there is no stored skip mode.

The guard also owns an active hacking-code reveal, so completion occurs before the existing hacking keyboard handler can submit a guess or pattern, append typed input, or navigate away.

A same-generation dud update does not create a new reveal controller, so it cannot unexpectedly arm the consumed-key guard or require the player to skip the board again.

### Verification strategy

Go asset assertions protect the required DOM layers, pointer transparency, absence of reduced-motion overrides, exact historical keyframe declarations, safe text construction, and the consumed-key/reveal-controller contract. Playwright proves rendered geometry, overlay hit testing, 40ms ordering, completion latency, event consumption, repeat suppression, normal handling on the next key, cancellation, identity replay suppression, audio-failure isolation, literal authored content, and viewport containment.

Approved screenshots cover selection, focus, active, and hacking-hover historical colors with animations frozen at a defined stable phase. Flicker timing and opacity checkpoints are verified through the browser animation model rather than timing a screenshot against a live animation phase.

BUG-001 browser checks observe the first hacking frame, row-by-row DOM order and cadence, unrevealed-target non-interactivity, sub-100ms completion, consumed hacking input, stable progress across same-identity updates/reconnect/fit, cancellation on replacement, and one fresh reveal for the new generation.

BUG-002 browser checks force a deterministic dud-removal outcome and observe child-list mutations, row-node identity, reveal timings, pending-row interaction, affected candidate text, used-pattern hover/focus state, and later hacking input. They prove zero full-board teardown and zero second reveal for both revealed-row and pending-row dud removal.

## Implementation Phases

### Phase 0: Research and decisions

- Retain the DOM/CSS composition and exact historical color/flicker baseline.
- Model reveal completion as an explicit, idempotent controller operation.
- Consume a skip key before normal player input and suppress repeats from the same physical press until key release.
- Keep pagination and layout recalculation immediate for already-opened content while new content identities reveal independently.
- Separate deterministic animation-contract checks from stable historical-color screenshots.

### Phase 1: Presentation contract and state design

- Define the observable shell, effect, reveal, keyboard, responsive, safety, and visual-snapshot contracts in `contracts/crt-presentation.md`.
- Model transient presentation mode, reveal sequence, active-page registry, and consumed-key guard in `data-model.md`.
- Confirm no RPC, generated binding, persistent JSON, Wails bridge, public access, or capability change.
- Re-check the Constitution Check against the final design; all seven principles remain PASS.

### Phase 2: Player integration

- Mark scanline and vignette nodes explicitly decorative without changing stacking or hit testing.
- Refactor row/line reveal around an idempotent controller with generation-safe cancellation and immediate completion.
- Extend that controller to a hacking-generation identity and complete code-row units without exposing partial or unrevealed targets.
- Reconcile same-generation dud-removal deltas into existing revealed or queued hacking rows without replacing unaffected nodes or restarting the controller.
- Install the reveal-skip guard before normal keyboard actions and isolate one consumed physical key from auto-repeat.
- Preserve current identity suppression, pagination, resize/font recalculation, same-generation hacking updates/fitting, authored-text safety, CSP, and optional-audio degradation.
- Do not add `prefers-reduced-motion`, a setting, persistence, or any path that disables persistent CRT effects.

### Phase 3: Focused verification and integration

- Extend `internal/platform/assets_test.go` with source and embedded contract assertions.
- Add a deterministic fixture surface with 25-row content, multiline record/command content, replacement content, hacking/blocked states, and literal markup-like authored text.
- Add focused Playwright coverage and approved visual snapshot baselines.
- Add BUG-001 asset and Playwright coverage for initial hacking-row reveal, interaction gating, skip consumption, identity stability, and replacement cancellation.
- Add BUG-002 fixture, asset, and Playwright coverage for deterministic dud removal, revealed/pending row reconciliation, zero unrelated row removals, and zero reveal replay.
- Run focused checks while developing, then the full browser suite as the required browser regression gate.
- Run repository Go checks, the player production build, the current runtime journey, and the unsigned application build.

## Verification Plan

| Surface | Command or method | Expected evidence |
|---|---|---|
| Player asset contract | `go test ./internal/platform -run 'CRT|Player.*Asset'` | Required DOM/CSS/JS contracts exist; reduced-motion overrides and unsafe authored-content paths are absent. |
| Player production build | `npm ci --prefix client` then `npm run build --prefix client` | The embedded player bundle builds without dependency or asset drift. |
| Focused browser journey | `npm test --prefix tests/browser -- crt-rendering.spec.mjs` | Fast feedback for CRT geometry, exact animation configuration, reveal lifecycle, consumed keys, snapshots, audio failure, safety, and viewports. |
| BUG-001 hacking reveal | Focused cases in `crt-rendering.spec.mjs` | A new puzzle reveals complete code rows at 40ms in deterministic order; same-identity updates do not replay; replacement cancels stale rows; a key completes within 100ms without a hacking action. |
| BUG-002 dud reconciliation | Focused cases in `crt-rendering.spec.mjs` | Dud removal within one generation updates only affected content, preserves unaffected row nodes and reveal progress, exposes no pending target early, and starts no new reveal. |
| Full browser regression gate | `npm ci --prefix tests/browser` then `npm test --prefix tests/browser` | Every browser journey passes, not only the new focused specification. |
| Repository Go checks | `gofmt -l .`, `go vet ./...`, `go test ./...` | No Go formatting paths; vet and all Go tests succeed. |
| Generated bindings | `go tool -modfile=tools/wails/go.mod wails3 generate bindings -clean ./...` | No unexplained generated-file drift; no generated contract was edited manually. |
| Current runtime | `go run ./cmd/build dev` | New folder, record, and command identities reveal; any key completes only the active page; its repeats cause no action; the next press works; persistent CRT effects continue. |
| Unsigned application | `go run ./cmd/build build` | The accepted runtime embeds the verified player assets. |

`go test -race ./...`, packaging, signing, notarization, and real public-endpoint checks are not feature-specific because the design changes no concurrent runtime surface, packaging contract, release policy, or public-access behavior. They remain governed by the repository’s normal conditional gates.
