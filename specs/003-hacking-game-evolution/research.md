# Research: Phase 1 Generation-Bound Hacking Patterns

This document supersedes the older coordinate-only and intended-count decisions. It records the corrective choices needed to bring the current implementation into alignment with `spec.md` and `planning-handoff.md` without adding any Non-Goal.

## Decision 1: Keep `patternId` opaque but make it resolve to the complete identity

**Decision**: Retain the existing `HACK_PATTERN` message with one required `patternId`. The server issues each ID from the complete tuple `generationId + row + inclusive start + inclusive end`; the browser treats the ID as opaque, and the hacking domain validates it only by resolving or matching it against patterns rediscovered from the active canonical board.

**Rationale**: This is the smallest compatible correction to the current typed protocol. Including the generation prevents a delayed request from matching coordinates in a later puzzle, while row-local coordinates distinguish stacked openings, shared closers, and changed pairings without exposing candidate truth.

**Alternatives considered**:

- Keep `columnIndex:openingIndex:closingIndex`: rejected because it lacks puzzle generation and does not implement the required rendered-row identity.
- Send explicit `generationId`, `row`, `start`, and `end`: valid under the specification but rejected for this implementation because it expands the existing request surface when an opaque generation-bearing ID already satisfies the contract.
- Assign an arbitrary ID unrelated to coordinates and store a permanent lookup: rejected because dynamic patterns would require mutable registration and stale lookup cleanup; deterministic resolution from the active generation and rediscovered coordinates is simpler.

## Decision 2: Define row coordinates independently of public column metadata

**Decision**: Model `row` as the zero-based rendered-row ordinal in canonical column render order: all rows of the first hacking column, followed by all rows of the second. Model `start` and `end` as zero-based inclusive character offsets within that row's 12-character hacking text. Internal helpers translate the tuple to the existing column and absolute text offsets; `column` and `pair` remain derivable private implementation data and leave the public projection.

**Rationale**: The tuple is stable, comparable, and sufficient for rendering while matching the exact identity fields required by the specification. A flattened row ordinal uniquely names each visible 12-character row without exposing an extra public `column` field, and row-local offsets make the same-row rule structural.

**Alternatives considered**:

- Retain public `column` with absolute column offsets: rejected because the clarified public projection permits only identity, row, inclusive coordinates, and status.
- Treat one left/right screen line as a 24-character row: rejected because it could pair delimiters across the visual gap between memory columns and would require presentation spacing or addresses to become domain data.
- Encode row and offsets only inside `patternId`: rejected because the browser also needs explicit coordinates for interaction and the public contract requires them.

## Decision 3: Issue generation IDs separately from gameplay randomness

**Decision**: Give each fresh runtime puzzle a non-blank server-issued generation ID before board construction. Use a process-local generation-ID source independent of the injected board/outcome `Random` source; production IDs are collision-resistant for the process lifetime, while live-service tests substitute a deterministic sequence.

**Rationale**: Generation identity must prevent stale activation even when coordinates repeat, including across puzzle replacement. Separating ID issuance from gameplay randomness ensures generation creation cannot disturb the exact outcome mapping or obscure assertions that rejected pattern requests consume zero outcome values.

**Alternatives considered**:

- Use only a service counter that resets on restart: rejected because a stale browser request could collide with the same counter and coordinates after a process restart.
- Draw the generation ID from the pattern-outcome `Random`: rejected because puzzle identity would consume and couple the deterministic outcome sequence.
- Persist generation IDs: rejected because active-puzzle persistence and deterministic seed persistence are explicit Non-Goals.

## Decision 4: Accept boards only after final production discovery reports `3–6`

**Decision**: Retain bracket-free ordinary filler and isolated intended pattern placement as bounded construction aids, but place construction inside a regeneration loop. After words, filler, addresses, and intended patterns form the final rendered board, run the same production discovery function used during gameplay and accept the board only when it discovers between `3–6` distinct selectable identities inclusive.

**Rationale**: The final board is the only reliable authority because interactions among placed characters can create or remove discoverable spans. The existing implementation verifies equality with a randomly selected intended target but returns `nil` instead of regenerating; replacing that with final-range acceptance and regeneration meets the clarified publication rule without making the selected target a requirement.

**Alternatives considered**:

- Trust the number of inserted pairs: rejected because intended insertions do not prove final discovery count.
- Require discovery to equal a randomly selected target: rejected because the specification requires only the final `3–6` range, not a random count distribution.
- Publish only a subset of discovered patterns: rejected because the gameplay discovery contract makes every valid current pattern selectable.

## Decision 5: Draw the weighted outcome before applying the no-dud fallback

**Decision**: After validation and used-state insertion, every accepted activation consumes exactly one outcome draw using `Intn(100)`. Values `0..79` select dud removal and `80..99` select attempt restoration. If dud removal was selected and no incorrect candidate is currently removable, apply attempt restoration as the fallback; only an actual dud removal consumes a second draw to select the dud.

**Rationale**: This preserves independent weighted selection for every accepted activation and makes the no-dud behavior a post-selection fallback. The current `len(decoys) == 0 || Intn(100) >= 80` short-circuit incorrectly skips the required outcome value when no dud remains.

**Alternatives considered**:

- Check for duds before drawing: rejected because it violates the mandated activation order and accepted-request RNG semantics.
- Preassign an effect to each public pattern: rejected because it leaks a future effect and prevents independent activation-time selection.
- Validate probability by demanding an exact random batch of 100 production activations: rejected because `80/20` defines the mapping, not a guaranteed sample result.

## Decision 6: Commit ordered publication inside the live transition without moving transport ownership

**Decision**: Extend the live pattern-activation call with a narrow publication callback. While holding the canonical live-service mutex, the service performs the nine ordered steps through detached projection creation and invokes the callback exactly once for an accepted activation. The callback, owned by `internal/player`, serializes and enqueues the detached `HACK_STATE` plus the existing game-master `hack-state` notification; it must not read or mutate live state or perform a reentrant live-service call.

**Rationale**: The current service unlocks before `internal/player` broadcasts, allowing concurrent accepted transitions to race in publication order. A narrow callback commits publication at step nine while preserving dependency direction: live state remains transport-agnostic, and the player server still owns envelopes, client queues, and WebSocket writes.

**Alternatives considered**:

- Keep broadcasting after `ApplyHackPattern` returns: rejected because it does not preserve the specified mutex-protected ordering.
- Make `internal/live` depend directly on the player server: rejected because it reverses the established boundary and violates transport independence.
- Hold the live mutex during direct network writes: rejected because slow clients would block canonical state; the existing player connection queues provide the correct non-blocking transport boundary.

## Decision 7: Project only current spans and retain complete used history privately

**Decision**: Store used state as a private set keyed by the complete generation-bound coordinate identity. Re-run discovery after every dud mutation. The public `patterns` array contains only currently discovered spans, each with opaque `id`, `row`, inclusive `start`/`end`, and `used`; if a historical coordinate pair disappears it remains only in private history, and if it later reappears it is immediately projected as used.

**Rationale**: This gives the browser exactly what it needs to render and interact with the current board while preserving the permanent one-use rule. Removing `column` and `pair` avoids broadening the projection, and detached copies prevent returned values from modifying canonical state.

**Alternatives considered**:

- Publish all historical used spans even when no longer valid: rejected because those coordinates no longer describe a current render target and could suppress ordinary filler interaction incorrectly.
- Forget used spans when discovery changes: rejected because a rediscovered coordinate pair would become reusable.
- Let the browser rediscover patterns: rejected because client and server discovery could diverge and the server is the canonical authority.

## Decision 8: Preserve the existing reconnect, persistence, and trusted-control boundaries

**Decision**: Continue sending the current detached hack projection through `TERMINAL_LIVE` on connection and `HACK_STATE` on accepted mutation. Keep generation, used history, removed duds, attempts, logs, and outcomes solely in the process-local live aggregate. Retain `ForceHackSuccess()` through the existing Wails `App` binding and existing success publication, with no player-protocol or browser invocation path.

**Rationale**: The repository already separates durable version-1 session content from live hacking state and already has a narrow game-master override. Reusing those boundaries satisfies reconnect and recovery behavior without schema migration, persistence, new roles, or another message family.

**Alternatives considered**:

- Persist the active puzzle or a complete seed: rejected as an explicit Non-Goal and a version-1 compatibility change.
- Add a session controller before accepting `HACK_PATTERN`: rejected because controlling-player and observer roles are explicitly out of scope; only the existing player/live-service seam remains as a future authorization extension point.
- Expose `ForceHackSuccess` as a player message or browser control: rejected because it recreates the cheat this phase removes.
