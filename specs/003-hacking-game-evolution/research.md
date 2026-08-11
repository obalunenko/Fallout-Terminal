# Research: Hacking Game Evolution

## Decision 1: Replace the player administrator path with a typed pattern action

**Decision**: Remove the generated `SUCCESS` candidate, keyboard command `1`, `HACK_ADMIN`, `ApplyAdmin`, and administrator-only model fields. Add player request `HACK_PATTERN` with one `patternId`, while leaving the Wails `ForceHackSuccess` method and its normal solved-state publication unchanged.

**Rationale**: The current implementation exposes the same bulk-dud-removal shortcut as both a board word and a keyboard command. A distinct typed action makes the replacement mechanic explicit at the untrusted protocol boundary, while retaining the already narrow and trusted game-master recovery path required by the specification.

**Alternatives considered**:

- Reuse `HACK_ADMIN`: rejected because the identifier preserves a removed capability and blurs compatibility behavior.
- Encode patterns as `HACK_GUESS` targets: rejected because guesses spend attempts while patterns have one-use random effects and require different validation.
- Remove `ForceHackSuccess`: rejected because the game-master override is explicitly required and already isolated behind the desktop bridge.

## Decision 2: Identify patterns by immutable board coordinates and rediscover them from current text

**Decision**: Identify a pattern as `columnIndex:openingIndex:closingIndex`, scan every row from each opening bracket to the first matching closing bracket on its right, and accept the span only when the interior contains no ASCII alphabetic character. Store consumed identities privately and expose every currently valid span with its coordinate fields and `used` flag.

**Rationale**: Coordinates make stacked openings that share a close distinct, are deterministic from the board already held by the server, and cannot reveal secret candidate data. Re-running the same pure discovery function after a dud becomes periods immediately reveals newly valid spans; retaining used coordinate identities prevents an earlier span from becoming reusable.

**Alternatives considered**:

- Use the bracket text as identity: rejected because duplicate and stacked patterns can have identical text.
- Trust client-supplied start/end coordinates without rediscovery: rejected because stale and tampered spans could bypass current-board validation.
- Assign opaque random IDs during generation only: rejected because dynamically created patterns would require a second identity mechanism.

## Decision 3: Construct the initial valid-pattern count instead of relying on incidental filler

**Decision**: Generate ordinary filler without `()`, `[]`, `{}`, or `<>`, choose the target with `3 + Intn(4)`, and insert exactly that many non-interacting same-row spans into free board cells after candidate placement. Cycle or randomly select among the exact allowed pairs `()`, `[]`, `{}`, `<>`; fill interiors only with non-alphabetic, non-bracket filler and verify the board with the production discovery function before returning it.

**Rationale**: Incidental brackets in the current filler pool make the number of valid spans uncontrolled and can create cross-pair interactions. Constructing isolated spans and verifying the final board guarantees `3–6` inclusive for every difficulty while keeping the rule deterministic under the existing injectable `Random` seam.

**Alternatives considered**:

- Retry fully random boards until the count happens to be in range: rejected because it has an unbounded tail and makes deterministic tests brittle.
- Publish only a random subset of incidental valid patterns: rejected because an apparently valid bracket span would become inexplicably unselectable.
- Vary the target by difficulty: rejected by the explicit difficulty-independent requirement.

## Decision 4: Consume and resolve a pattern atomically in the live service

**Decision**: Add `ApplyHackPattern(patternId)` to the hacking domain and invoke it while `internal/live.Service` holds its existing exclusive mutex. The domain rediscovery check, used check, used-state insertion, random outcome, board mutation, pattern recalculation, and returned public projection occur as one transition; rejected actions return no accepted result and trigger no broadcast.

**Rationale**: The existing live mutex is already the serialization boundary for shared hacking. Keeping the whole transition within it guarantees that simultaneous requests for one coordinate pair can produce at most one effect and that every emitted snapshot represents a complete state.

**Alternatives considered**:

- Deduplicate in the WebSocket server: rejected because other callers of the live service could bypass it and transport code would own domain rules.
- Mark a pattern used after choosing its outcome: rejected because concurrent callers could both pass validation before consumption.
- Let each browser disable a pattern locally: rejected because reconnects and multiple clients would diverge.

## Decision 5: Use the existing random boundary for exact effect selection

**Decision**: When at least one incorrect candidate remains, values `0..79` from `Intn(100)` remove one randomly selected incorrect candidate and values `80..99` restore attempts. Dud removal deletes exactly one non-secret candidate from the private lookup and public word placements and replaces its visible characters with periods. With no incorrect candidate, the transition restores attempts without making an outcome roll.

**Rationale**: The integer threshold implements the exact `80%` and `20%` contract without floating-point ambiguity or a new dependency. The current injectable random interface supports an exact controlled sequence test and deterministic dud selection.

**Alternatives considered**:

- Use floating-point probabilities: rejected because integer buckets are simpler to test exactly.
- Allow a dud-removal outcome to do nothing: rejected because the specification requires attempt restoration as the useful fallback.
- Preselect removable duds in public state: rejected because candidate truth, including the secret, must remain private.

## Decision 6: Drive browser interaction entirely from the public pattern projection

**Decision**: Extend the public hack state with current pattern metadata. The browser maps each opening coordinate to its pattern, highlights the inclusive opening-through-closing range on hover, and sends `HACK_PATTERN` on activation. Used openings remain mapped but do not highlight; reactivation sends the same pattern request so the server can reject it without spending an attempt. All other word and filler cells retain `HACK_GUESS` behavior.

**Rationale**: The browser needs exact range metadata to render overlapping spans but must not rediscover or own canonical availability. Keeping used openings distinguishable also prevents a second click from falling through to an ordinary filler guess and unexpectedly consuming an attempt.

**Alternatives considered**:

- Discover patterns independently in JavaScript: rejected because two implementations could disagree after board changes.
- Render one wrapper element per pattern: rejected because overlapping and stacked ranges cannot be represented reliably as independent nested DOM spans.
- Remove filler guessing: rejected because normal hacking behavior remains unchanged outside the former cheat paths.

## Decision 7: Extend `HACK_STATE` compatibly and reject the removed request

**Decision**: Keep `TERMINAL_LIVE`, `HACK_STATE`, and the existing hack fields, add a camelCase `patterns` array, and remove the obsolete public `isAdmin` field from word placements. Old `HACK_ADMIN` requests become unsupported strict-protocol input and leave state unchanged; reconnects receive the current board and pattern projection through the existing full snapshot.

**Rationale**: Additive server fields are tolerated by the current browser dispatch, and the existing reconnect envelope already carries the full public puzzle. Explicit rejection of the removed mutation path ensures stale clients cannot retain a cheat.

**Alternatives considered**:

- Add a protocol version solely for this change: rejected because the application ships server and client together and the current decoder has no negotiation mechanism.
- Keep accepting `HACK_ADMIN` as a no-op: rejected because a silently accepted removed command obscures validation and makes testing the absence of cheats weaker.
- Add a separate reconnect message for patterns: rejected because it would duplicate state already carried by `TERMINAL_LIVE`.
