# Hacking Interface Contract: Phase 1 Generation-Bound Patterns

**Bugfix**: 2026-08-11 — BUG-001 retained the `HACK_GUESS` wire shape while extending filler eligibility to delimiters and restricting pattern interaction to `start`.

## Scope

This contract governs the player WebSocket pattern request, the public pattern projection carried by existing hack snapshots, ordered publication, reconnect behavior, and the separation of the trusted game-master solve control. It does not add a route, protocol version, session role, persisted puzzle, terminal switch, or presentation redesign.

## Exact Gameplay Constraints

- Allowed pattern pairs: `()`, `[]`, `{}`, `<>`.
- Initial valid special-pattern count: `3–6` inclusive on the final rendered board before the first player action.
- Initial standalone delimiter-decoy count: at least the number of initially valid special patterns.
- Every initial board contains at least one valid pattern with a non-empty non-alphabetic interior and at least one matching-delimiter span invalidated by alphabetic content.
- Candidate words, valid-pattern endpoints, and standalone delimiter decoys each occupy at least two rendered rows; their inclusive minimum-to-maximum occupied-row intervals overlap pairwise; ordinary punctuation or filler remains present in at least two rows.
- Dud-removal probability mapping: `80%`.
- Attempt-restoration probability mapping: `20%`.
- Pattern identity: `generationId + row + inclusive start + inclusive end`.
- Persistence boundary: runtime-only; no version-1 session schema change.

## Player-to-Server Messages

### `HACK_GUESS` — retained wire shape; delimiter eligibility expanded by BUG-001

```json
{
  "type": "HACK_GUESS",
  "targetId": "A1"
}
```

`targetId` remains a non-blank candidate ID or the existing filler coordinate. Candidate validation, likeness, attempt spending, log messages, success, failure, and filler-click behavior do not change. Under BUG-001, an existing filler coordinate is valid for standalone delimiter glyphs and non-opening filler glyphs inside a valid pattern span; a current pattern's opening coordinate remains reserved for pattern handling. There is no generated administrator candidate.

### `HACK_PATTERN` — generation-bound opaque identity

```json
{
  "type": "HACK_PATTERN",
  "patternId": "opaque-server-issued-pattern-identity"
}
```

| Field | Required | Validation |
|---|---|---|
| `type` | yes | Exact string `HACK_PATTERN` |
| `patternId` | yes | Non-blank server-issued string that contains or resolves to the originating `generationId`, `row`, inclusive `start`, and inclusive `end` |
| any other field | prohibited | Strict decoding rejects the complete request before live-service dispatch |

The browser must echo `patternId` exactly as received and must not parse, synthesize, shorten, or replace it with coordinates from another projection.

### Semantic Acceptance

The canonical live service accepts `HACK_PATTERN` only when all conditions are true in this order:

1. A puzzle exists and is active, unsolved, and unfailed.
2. The identity resolves to the active puzzle generation.
3. Production discovery against the current canonical rendered board contains the exact `row`, `start`, and `end` coordinate pair.
4. The complete generation-bound identity is not in used history.

Acceptance then marks the identity used, selects one weighted outcome, applies the effect or no-dud fallback, recomputes current patterns, creates a detached public state, and commits one ordered publication under the same live-service mutex.

### Rejection Contract

| Rejection | Required result |
|---|---|
| Missing, blank, non-string, duplicate, or unknown field | Decoder rejects before dispatch |
| Unsupported message type, including removed `HACK_ADMIN` | Decoder rejects before dispatch |
| Generation differs from the active puzzle | No canonical mutation, random-source advancement, or broadcast |
| Coordinates are malformed or do not identify a currently discovered span | No canonical mutation, random-source advancement, or broadcast |
| Complete identity is already used | No canonical mutation, random-source advancement, or broadcast |
| No puzzle, solved puzzle, failed puzzle, or otherwise non-actionable puzzle | No canonical mutation, random-source advancement, or broadcast |
| Concurrent duplicate arriving after the accepted request | Rejected as used; exactly one request total advances the random source and broadcasts |

A delayed ID from an older puzzle can never activate coincident coordinates in a newer puzzle because the opaque identity includes or resolves to the older `generationId`.

### `HACK_ADMIN` — removed

`HACK_ADMIN` is unsupported player input. The player has no keyboard command, query parameter, board entry, DOM control, browser global, or alternate message that maps to it or to `ForceHackSuccess`.

## Server-to-Player Messages

### `HACK_STATE` — minimal pattern projection

The existing envelope remains:

```json
{
  "type": "HACK_STATE",
  "hack": {
    "level": 2,
    "wordLength": 5,
    "attemptsMax": 4,
    "attemptsLeft": 4,
    "solved": false,
    "failed": false,
    "log": [],
    "columns": [],
    "patterns": [
      {
        "id": "opaque-server-issued-pattern-identity",
        "row": 4,
        "start": 1,
        "end": 6,
        "used": false
      }
    ]
  }
}
```

Existing hack and column fields retain their current behavior. For each currently discovered special pattern, the `patterns` array contains only:

| Field | Meaning |
|---|---|
| `id` | Stable opaque identity containing or resolving to `generationId + row + inclusive start + inclusive end` |
| `row` | Zero-based rendered-row ordinal in canonical column render order |
| `start` | Zero-based inclusive opening-character offset within that row and sole coordinate eligible for pattern handling |
| `end` | Zero-based inclusive closing-character offset within that row; completes the highlight/effect span but is not a pattern hit target |
| `used` | `false` when currently available; `true` when this complete identity has already been accepted |

The array is sorted by `row`, `start`, then `end`. It may exceed six after the first player action. A currently discovered used span remains present with `used: true`; a used span that is not currently valid remains only in private history and reappears as used if later rediscovered.

The pattern object must not contain `column`, `pair`, a separately editable `generationId`, password or dud facts, a future outcome, random values, private candidate metadata, delimiter-decoy metadata, or any reference to canonical slices, maps, or objects. Standalone, mismatched, word-interrupted, later-compatible-but-unselected, and otherwise invalid delimiters remain individually selectable ordinary rendered characters and receive no pattern object or identity.

### `TERMINAL_LIVE` — reconnect snapshot retained

The existing full live envelope retains `terminalId`, `terminalName`, `tree`, `hackLevel`, `introText`, `hack`, and `nav`. When `hack` is non-null, its `patterns` use the exact shape above.

A player connecting or reconnecting to the same running process receives the current canonical public puzzle: rendered board, remaining attempts, log and outcome, removed-dud board changes, and all current pattern statuses. The server does not regenerate the puzzle for a reconnect.

No contract promises restoration after application restart. A fresh puzzle has a new generation identity and empty used history; version-1 session data contains neither.

## Atomic Publication Contract

For an accepted `HACK_PATTERN`, the canonical live-service mutex covers this order:

1. Validate active puzzle generation.
2. Rediscover or validate the requested coordinates against canonical board state.
3. Verify unused state.
4. Mark the complete identity used.
5. Select the weighted outcome.
6. Apply dud removal, restoration, or the no-dud restoration fallback.
7. Recompute patterns affected by board mutation.
8. Produce a detached public projection.
9. Invoke one publication callback with that projection.

The publication callback is owned by the player boundary. It serializes and enqueues one `HACK_STATE` to the existing client fanout and invokes the existing detached `hack-state` game-master notification. It performs no reentrant live-service call. Actual WebSocket writes remain outside the canonical domain and use the existing player connection queues.

| Event | Publication |
|---|---|
| First accepted activation | Exactly one complete post-effect `HACK_STATE` committed for all connected players and one detached game-master `hack-state` notification |
| Concurrent duplicate | None |
| Invalid, stale-generation, non-current, already-used, or terminal-state request | None |
| Fresh live publication | Existing `TERMINAL_LIVE` with a new generation, `3–6` complete-final-board-discovered initial patterns, and all camouflage publication gates satisfied |
| Reconnect while process remains active | Existing `TERMINAL_LIVE` with current state and generation-bound pattern IDs |
| Trusted game-master solve | Existing success publication; no attempt spent |

## Player UI Contract

- The browser consumes `hack.patterns`; it never performs canonical discovery or effect selection.
- Each rendered hacking cell exposes its canonical rendered-row ordinal and row-local character offset. Pattern lookup resolves only at `start`; the resolved pattern still supplies inclusive `start`-through-`end` highlighting.
- Hovering an unused pattern opening highlights `start` through `end` on `row`, inclusive, and previews that board substring.
- Different openings sharing one closer remain different targets because their `start` values and opaque IDs differ.
- A used current pattern does not highlight and does not fall through to `HACK_GUESS` when clicked; the server remains authoritative if a repeated request is sent.
- Clicking an available opening sends only `{type: "HACK_PATTERN", patternId: pattern.id}`. Clicking an ordinary candidate, including a candidate inside an alphabetic-interrupted delimiter span, retains existing `HACK_GUESS` behavior.
- ~~Hovering, focusing, or clicking a rendered delimiter outside every current projected pattern range produces no highlight, preview, `HACK_PATTERN`, `HACK_GUESS`, attempt consumption, or puzzle-state change. Canonical filler-target handling also rejects such a direct delimiter target without mutation.~~ **Superseded by BUG-001**: standalone delimiters and non-opening filler cells inside a current pattern span receive ordinary individual highlight/preview; clicking sends `HACK_GUESS`, and canonical filler handling applies the established log and attempt behavior. They never send `HACK_PATTERN`.
- Clicking ordinary ~~non-delimiter~~ filler retains existing `HACK_GUESS` behavior unless the cell is a current pattern's opening coordinate. A used current pattern opening retains its existing unavailable behavior and does not fall through to `HACK_GUESS`.
- Valid delimiter endpoints and delimiter decoys use identical static color, brightness, font, CRT effect, and persistent classes. Only transient whole-span feedback from hovering, focusing, or selecting a current unused pattern's opening coordinate may distinguish a valid pattern.
- After dud removal replaces an alphabetic-interrupted candidate with periods, the browser treats the span as actionable only after the server's next valid-only pattern projection includes it.
- The browser changes no board, attempts, log, pattern status, or outcome until a server snapshot arrives.

## Game-Master Interface Contract

The trusted desktop contract remains `ForceHackSuccess()` through the generated Wails `App` binding. It is eligible only for an active unsolved and unfailed puzzle, does not consume an attempt, and publishes through the existing shared success flow. The existing private control remains disabled when no eligible puzzle exists.

No player WebSocket type, browser global, DOM control, keyboard shortcut, query parameter, static asset, or public endpoint exposes `ForceHackSuccess` or an equivalent operation.

## Compatibility Impact

The server and bundled player are released together. Existing `HACK_GUESS`, `HACK_STATE`, and `TERMINAL_LIVE` message names remain stable. `HACK_PATTERN` retains `patternId`, but coordinate-only IDs from the older implementation are stale and are rejected because they do not resolve to the active generation. Public pattern objects replace `column` and `pair` with `row`; bundled browser code and golden fixtures change in the same release.

No session migration, protocol negotiation, terminal switch, role assignment, persistent unlock, or active-puzzle restart recovery is introduced.
