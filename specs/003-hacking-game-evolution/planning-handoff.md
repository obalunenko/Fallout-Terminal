# Planning Handoff: Phase 1 Server-Authoritative Hacking Patterns

Use `spec.md` as the normative source of truth. Rebuild the implementation plan and its research, data-model, and contract artifacts wherever the existing versions conflict with the clarified specification. Do not broaden the feature into session control, visual redesign, dictionaries, persistence, terminal switching, localization, audio, or any other Non-Goal.

## Mandatory Planning Guardrails

1. Preserve the exact activation order from FR-047 under the canonical live-service mutex. Validation of generation, current coordinates, actionable state, and used state happens before the used mark and before random selection. Every accepted activation selects one weighted outcome; invalid, stale, already-used, non-actionable, and duplicate requests consume no random value.
2. Treat the `80%`/`20%` rule as a probability mapping. Plan an injected deterministic random source whose 100 equiprobable inputs map exactly 80 values to dud removal and 20 to restoration. Do not require a production sample of 100 activations to yield an exact split.
3. When dud removal is selected but no removable incorrect candidate remains, restore attempts after the weighted selection. Do not skip the outcome selection for an otherwise accepted activation.
4. Count initial patterns by running the final rendered board through the same discovery function used during play. Regenerate any board whose discovered count is outside `3–6`. Intended insertions may help generation but never satisfy the requirement without final-board discovery. Do not add a requirement that the count itself be randomly selected.
5. Use the exact discovery semantics: one of `()`, `[]`, `{}`, `<>`; same rendered row; no alphabetic interior; first compatible closer to the right; inclusive coordinates; multiple openings may share one closer.
6. Make pattern identity generation-bound: `generationId + row + inclusive start + inclusive end`. Used history belongs to the complete identity. A changed closer creates a new identity; a previously used coordinate pair remains unavailable when rediscovered in the same generation.
7. Define `HACK_PATTERN` as generation-bound. Choose either an opaque server-issued `patternId` that resolves to generation and coordinates or the explicit `generationId`, `row`, `start`, `end` payload. Whichever representation the plan selects must strictly reject missing, unknown, and invalid fields and must prevent an old-generation request from targeting coincident coordinates in a new puzzle.
8. Keep the public pattern projection minimal and detached: stable public identity, row, inclusive start/end, and available-or-used status only. Do not expose the password, dud identities, future effects, private candidate metadata, delimiter metadata derivable from the board, or canonical mutable references. Include mutation-isolation tests.
9. Keep pattern progress, used identities, removed duds, attempts, and outcomes runtime-only. Reconnect within the same process receives current canonical public state. Do not change the version-1 session schema or persist a complete puzzle seed or active puzzle across application restarts.
10. Preserve ordinary password-guess and filler-click attempt behavior. Retain `ForceHackSuccess` only through the existing private desktop/Wails boundary; expose no equivalent through the player WebSocket protocol, browser globals, DOM controls, keyboard shortcuts, or query parameters.
11. Preserve a distinct player/live-service authorization boundary as a future extension point, but do not implement controlling-player, observer, session-name, or game-master reassignment behavior.

## Existing Artifact Conflicts to Remove

The current design artifacts predate the clarification. Replace, rather than append beside, any statements that conflict with the following:

- A pattern ID based only on column/opening/closing coordinates is insufficient because it lacks the puzzle generation and the required rendered-row identity.
- A generation strategy that inserts an intended target count is insufficient unless the final board is rediscovered and rejected outside `3–6`.
- Skipping the weighted outcome roll when no dud remains violates the accepted-activation order; fallback occurs after selection.
- A public projection containing `pair`, private candidate facts, or other fields beyond identity, row, inclusive coordinates, and status is too broad.
- A `patternId` that does not contain or server-resolve to the originating generation permits stale cross-generation activation and is invalid.
- Any success criterion based on a genuinely random batch producing exactly `80/20` is obsolete.

## Required Plan Outputs

- `plan.md`: exact source files, domain/live/player/browser boundaries, Constitution Check, and a no-scope-expansion structure decision.
- `research.md`: decisions for generation-bound identity, final-board regeneration, deterministic outcome selection, atomic activation, detached projections, reconnect behavior, and private game-master control.
- `data-model.md`: generation identity, complete pattern identity, used-history lifecycle, dynamic discovery, outcome transition, public projection, and runtime-only persistence boundary.
- `contracts/hacking-interface.md`: strict `HACK_PATTERN` request representation, rejection rules, public projection, broadcast/reconnect behavior, and absence of any player `ForceHackSuccess` path.

Before completing the plan, compare every requirement and success criterion in `spec.md` against those artifacts and remove all stale contradictions from the older plan set.

## Ready-to-Use Invocation

```text
$speckit-companion-plan Rebuild the existing plan artifacts from spec.md and planning-handoff.md. Treat the older plan.md, research.md, data-model.md, and contracts/hacking-interface.md as stale wherever they conflict with the clarified Phase 1 requirements. Preserve every Mandatory Planning Guardrail, remove the listed conflicts, and do not expand the Non-Goals.
```
