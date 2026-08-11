# Research: Phase 1 Generation-Bound Hacking Patterns

**Bugfix**: 2026-08-11 — BUG-001 superseded inert-delimiter input and inclusive-span hit testing with individual delimiter selection and opening-symbol-only pattern activation.

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

## Decision 4: Build camouflage first and accept only complete boards that pass production discovery and camouflage validation

**Decision**: Replace bracket-free filler and isolated adjacent-pair placement with a camouflaged construction attempt. Place candidate words, ordinary punctuation, intended valid patterns with at least one non-empty non-alphabetic interior, matching delimiters around at least one candidate word, and standalone delimiter-decoy candidates through the normal board rows before validation. Run the unchanged production discovery function used during gameplay on the complete final render. Accept only when it discovers `3–6` identities inclusive; the remaining ~~inert~~ individually selectable standalone delimiter-decoy character count is at least the valid-pattern count; at least one valid interior is non-empty; at least one potential span is invalidated by alphabetic content; candidate words, valid-pattern endpoints, and standalone decoys each occupy at least two rows; the three inclusive minimum-to-maximum occupied-row intervals overlap pairwise; and ordinary punctuation or filler remains in at least two rows. Otherwise discard the attempt and regenerate.

**Rationale**: The final board is the only reliable authority because words, intended spans, and delimiter decoys can accidentally create or remove discoverable spans and change which delimiter characters remain ~~inert~~ standalone rather than part of a valid pattern. Classifying camouflage only after production discovery keeps one gameplay definition of validity and prevents construction metadata from leaking into the public projection. Non-empty interiors and word-interrupted spans provide camouflage without altering the same-row, matching-pair, first-compatible-closer, no-alphabetic-interior discovery rules.

**Alternatives considered**:

- Trust the number of inserted pairs: rejected because intended insertions do not prove final discovery count.
- Require discovery to equal a randomly selected target: rejected because the specification requires only the final `3–6` range, not a random count distribution.
- Publish only a subset of discovered patterns: rejected because the gameplay discovery contract makes every valid current pattern selectable.
- Keep ordinary filler free of delimiters: rejected because the board must contain standalone delimiter decoys rendered among ordinary content.
- Construct every intended pattern as an adjacent empty pair: rejected because every published board must include a non-empty valid interior.
- Validate decoys before adding words or before final discovery: rejected because alphabetic interruptions and accidental pairings are properties of the complete rendered board.

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

**Decision**: Store used state as a private set keyed by the complete generation-bound coordinate identity. Re-run discovery after every dud mutation. The public `patterns` array contains only currently discovered spans, each with opaque `id`, `row`, inclusive `start`/`end`, and `used`; standalone, mismatched, word-interrupted, later-compatible-but-unselected, and otherwise invalid delimiters remain board text with no identity. Under BUG-001, `start` is also the only browser hit target for whole-pattern hover, focus, and activation; `end` still defines the inclusive highlight/effect span, not a second activation target. If dud removal turns an alphabetic-interrupted span into a valid span, it enters the next projection normally. If a historical coordinate pair disappears it remains only in private history, and if it later reappears it is immediately projected as used.

**Rationale**: This gives the browser exactly what it needs to render and interact with the current board while preserving the permanent one-use rule. Removing `column` and `pair` avoids broadening the projection, and detached copies prevent returned values from modifying canonical state.

**Alternatives considered**:

- Publish all historical used spans even when no longer valid: rejected because those coordinates no longer describe a current render target and could suppress ordinary filler interaction incorrectly.
- Forget used spans when discovery changes: rejected because a rediscovered coordinate pair would become reusable.
- Let the browser rediscover patterns: rejected because client and server discovery could diverge and the server is the canonical authority.
- Publish decoy identities with an unavailable marker: rejected because identity itself would reveal validity metadata and broaden the public projection beyond actually valid patterns.

## Decision 8: ~~Make delimiter decoys inert without marking them as valid~~ Preserve valid-only identities while keeping delimiters individually selectable (superseded by BUG-001)

**Decision**: ~~Treat a rendered delimiter cell outside every projected valid pattern range as inert input. The browser sends neither `HACK_PATTERN` nor `HACK_GUESS` for that cell, and canonical filler-target validation also ignores a direct delimiter-decoy guess.~~ **BUG-001 superseding decision**: every rendered delimiter retains its ordinary filler target unless it is a current pattern's opening coordinate. Only the opening cell at `pattern.start` resolves to pattern handling: an unused opening provides whole-pattern interaction and sends `HACK_PATTERN`, while a used opening retains its existing unavailable behavior. Standalone invalid delimiters and every non-opening filler cell within a valid span use individual preview/highlight and `HACK_GUESS`, with normal canonical filler logging and attempt handling. Candidate cells between matching delimiters retain their existing candidate IDs and ordinary guess behavior. Valid and invalid delimiter glyphs share the same static rendering in `client/client.css`; the browser adds whole-span transient pattern feedback only when a current unused pattern's opening cell is targeted. Retain Go source and stylesheet contract checks, and add an isolated executable browser suite with an exactly pinned and locked test-only dependency because source inspection cannot prove hover, focus, click, computed-style, or outbound-message behavior.

**Rationale**: The public pattern projection stays valid-only, and its existing `start` coordinate is sufficient to distinguish the sole pattern activation anchor without a new wire field. ~~The browser identifies an otherwise inert delimiter directly from its rendered character, and the canonical guard prevents a manually submitted filler target from turning an intended no-op into an attempt loss.~~ BUG-001 instead preserves the ordinary filler-target path for those characters. Keeping static rendering uniform prevents presentation metadata from exposing validity before the player targets an opening symbol.

**Alternatives considered**:

- Give decoys public IDs marked unavailable: rejected because invalid delimiters must receive no public pattern identity.
- ~~Let delimiter decoys fall through to ordinary filler guesses: rejected because the requirement makes decoy selection a no-op that consumes no attempt and changes no state.~~ **Adopted by BUG-001** so invalid delimiters and non-opening filler cells remain individually selectable with established filler semantics.
- Add a permanent valid-pattern CSS class: rejected because static styling may not reveal which delimiters are valid.
- Treat Go asset-source assertions as complete browser interaction coverage: rejected because source inspection does not execute DOM events, computed styling, or outbound WebSocket dispatch.

## Decision 9: Preserve the existing reconnect, persistence, and trusted-control boundaries

**Decision**: Continue sending the current detached hack projection through `TERMINAL_LIVE` on connection and `HACK_STATE` on accepted mutation. Keep generation, used history, removed duds, attempts, logs, and outcomes solely in the process-local live aggregate. Retain `ForceHackSuccess()` through the existing Wails `App` binding and existing success publication, with no player-protocol or browser invocation path.

**Rationale**: The repository already separates durable version-1 session content from live hacking state and already has a narrow game-master override. Reusing those boundaries satisfies reconnect and recovery behavior without schema migration, persistence, new roles, or another message family.

**Alternatives considered**:

- Persist the active puzzle or a complete seed: rejected as an explicit Non-Goal and a version-1 compatibility change.
- Add a session controller before accepting `HACK_PATTERN`: rejected because controlling-player and observer roles are explicitly out of scope; only the existing player/live-service seam remains as a future authorization extension point.
- Expose `ForceHackSuccess` as a player message or browser control: rejected because it recreates the cheat this phase removes.
