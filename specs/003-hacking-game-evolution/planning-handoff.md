# Planning Handoff: Phase 1 Server-Authoritative Hacking Patterns

**Bugfix**: 2026-08-11 — BUG-001 superseded inert delimiter input with individual filler-symbol selection and limited whole-pattern interaction to the opening coordinate.

Use `spec.md` as the normative source of truth. Rebuild the implementation plan and its research, data-model, and contract artifacts wherever the existing versions conflict with the clarified specification. Do not broaden the feature into session control, visual redesign, dictionaries, persistence, terminal switching, localization, audio, or any other Non-Goal.

## Mandatory Planning Guardrails

1. Preserve the exact activation order from FR-047 under the canonical live-service mutex. Validation of generation, current coordinates, actionable state, and used state happens before the used mark and before random selection. Every accepted activation selects one weighted outcome; invalid, stale, already-used, non-actionable, and duplicate requests consume no random value.
2. Treat the `80%`/`20%` rule as a probability mapping. Plan an injected deterministic random source whose 100 equiprobable inputs map exactly 80 values to dud removal and 20 to restoration. Do not require a production sample of 100 activations to yield an exact split.
3. When dud removal is selected but no removable incorrect candidate remains, restore attempts after the weighted selection. Do not skip the outcome selection for an otherwise accepted activation.
4. Add candidate words, ordinary filler, valid-pattern construction aids, at least one non-empty valid interior, alphabetic-interrupted spans, and standalone delimiter-decoy candidates before validation. Use the normal board rows for every category and reserve no category-specific row block.
5. Count initial patterns by running the complete final rendered board through the same discovery function used during play. Accidentally formed patterns count. Regenerate any board whose discovered count is outside `3–6`, whose standalone delimiter-decoy character count is below the discovered-pattern count, or that lacks a non-empty valid interior, an alphabetic-interrupted potential span, at least two occupied rows for each candidate-word/valid-endpoint/standalone-decoy category, pairwise overlap among those categories' inclusive occupied-row intervals, or ordinary punctuation/filler in at least two rows. Intended insertions may help generation but never satisfy publication without final-board analysis. Do not add a requirement that the count itself be randomly selected.
6. Preserve the exact discovery semantics: one of `()`, `[]`, `{}`, `<>`; same rendered row; no alphabetic interior; first compatible closer to the right; inclusive coordinates; multiple openings may share one closer. Camouflage changes generation inputs and validation only, not this algorithm.
7. Make pattern identity generation-bound: `generationId + row + inclusive start + inclusive end`. Used history belongs to the complete identity. A changed closer creates a new identity; a previously used coordinate pair remains unavailable when rediscovered in the same generation.
8. Define `HACK_PATTERN` as generation-bound. Choose either an opaque server-issued `patternId` that resolves to generation and coordinates or the explicit `generationId`, `row`, `start`, `end` payload. Whichever representation the plan selects must strictly reject missing, unknown, and invalid fields and must prevent an old-generation request from targeting coincident coordinates in a new puzzle.
9. Keep the public pattern projection minimal and detached: stable public identity, row, inclusive start/end, and available-or-used status only for spans returned by current production discovery. Standalone, mismatched, word-interrupted, later-compatible-but-unselected, and otherwise invalid delimiters receive no identity. Include projection-exclusivity and mutation-isolation tests.
10. Keep pattern progress, used identities, removed duds, attempts, and outcomes runtime-only. Reconnect within the same process receives current canonical public state. Do not change the version-1 session schema or persist a complete puzzle seed or active puzzle across application restarts.
11. Preserve ordinary password-guess and ~~non-delimiter~~ filler-click attempt behavior. Candidate words inside invalid delimiter spans remain ordinary guesses. ~~Standalone delimiter decoys must be inert in browser dispatch and canonical filler-target handling.~~ **BUG-001 correction**: standalone delimiters and non-opening filler cells inside valid spans use individual `HACK_GUESS` behavior; only a current pattern's opening coordinate receives pattern handling, and used openings retain their existing unavailable behavior. Retain `ForceHackSuccess` only through the existing private desktop/Wails boundary; expose no equivalent through the player WebSocket protocol, browser globals, DOM controls, keyboard shortcuts, or query parameters.
12. Render valid delimiter endpoints and delimiter decoys with identical static color, brightness, font, CRT effect, and classes. Treat `client/client.css` as an affected and verified surface. ~~Only normal hover, focus, or selection behavior may reveal a currently valid pattern.~~ Only hovering, focusing, or selecting a current unused pattern's opening coordinate may reveal validity through whole-span feedback.
13. Use executable browser interaction coverage for DOM events, computed-style parity, and outbound message dispatch. Keep the browser-test dependency isolated from production, pin it exactly, and commit its lockfile; Go asset-source assertions remain complementary static checks rather than substitutes for browser execution.
14. Preserve a distinct player/live-service authorization boundary as a future extension point, but do not implement controlling-player, observer, session-name, or game-master reassignment behavior.

## Existing Artifact Conflicts to Remove

The current design artifacts predate the clarification. Replace, rather than append beside, any statements that conflict with the following:

- A pattern ID based only on column/opening/closing coordinates is insufficient because it lacks the puzzle generation and the required rendered-row identity.
- A generation strategy that inserts an intended target count is insufficient unless the final board is rediscovered and rejected outside `3–6`.
- Bracket-free filler, adjacent-empty-only initial patterns, isolated pattern placement, or any category-specific row grouping conflicts with the camouflage requirements.
- Counting intended decoys before production discovery is insufficient because accidental valid spans change both the discovered-pattern count and which delimiter characters remain standalone decoys.
- Skipping the weighted outcome roll when no dud remains violates the accepted-activation order; fallback occurs after selection.
- A public projection containing `pair`, private candidate facts, or other fields beyond identity, row, inclusive coordinates, and status is too broad.
- A `patternId` that does not contain or server-resolve to the originating generation permits stale cross-generation activation and is invalid.
- Any success criterion based on a genuinely random batch producing exactly `80/20` is obsolete.
- Treating standalone delimiters as inert or treating every inclusive pattern offset as a whole-pattern hit target is obsolete under BUG-001.

## Required Plan Outputs

- `plan.md`: exact source files, domain/live/player/browser boundaries, Constitution Check, and a no-scope-expansion structure decision.
- `research.md`: decisions for generation-bound identity, camouflaged final-board regeneration, deterministic outcome selection, atomic activation, detached valid-only projections, reconnect behavior, and private game-master control.
- `data-model.md`: generation identity, complete pattern identity, final-board camouflage classification, used-history lifecycle, dynamic discovery, outcome transition, public projection, and runtime-only persistence boundary.
- `contracts/hacking-interface.md`: strict `HACK_PATTERN` request representation, rejection rules, valid-only public projection, ~~inert decoy interaction~~ individual delimiter selection with opening-coordinate-only pattern activation, broadcast/reconnect behavior, and absence of any player `ForceHackSuccess` path.

Before completing the plan, compare every requirement and success criterion in `spec.md` against those artifacts and remove all stale contradictions from the older plan set.

## Ready-to-Use Invocation

```text
$speckit-companion-plan Rebuild the existing plan artifacts from spec.md and planning-handoff.md. Treat the older plan.md, research.md, data-model.md, and contracts/hacking-interface.md as stale wherever they conflict with the clarified Phase 1 requirements. Preserve every Mandatory Planning Guardrail, remove the listed conflicts, and do not expand the Non-Goals.
```
