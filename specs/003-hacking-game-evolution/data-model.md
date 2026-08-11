# Data Model: Phase 1 Generation-Bound Hacking Patterns

## Modeling Principles

- The private hacking aggregate remains the process-local source of truth and is mutated only while the canonical live-service mutex is held.
- Pattern identity is the complete tuple `generationId + row + inclusive start + inclusive end`; public `patternId` values contain or resolve to that tuple and are opaque to clients.
- Pattern validity is derived from the current final rendered board by one discovery function used both before publication and during gameplay.
- Used history survives discovery changes for the lifetime of one puzzle generation, while public patterns describe only spans currently present on the board.
- Outcome selection and private candidate truth remain server-only. Public projections are detached and contain no mutable reference to canonical data.
- All state introduced here is runtime-only and does not alter version-1 session JSON.

## Constants and Verbatim Rules

| Rule | Exact value |
|---|---|
| Allowed pattern pairs | `()`, `[]`, `{}`, `<>` |
| Initial valid special-pattern count | `3–6` inclusive on the final rendered board before the first player action |
| Dud-removal probability mapping | `80%` |
| Attempt-restoration probability mapping | `20%` |
| Pattern identity | `generationId + row + inclusive start + inclusive end` |
| Persistence boundary | Runtime-only; no version-1 session schema change |
| Existing board geometry | Two columns, 16 rows per column, 12 hacking characters per row |

## Coordinate Model

`row` is the zero-based rendered-row ordinal in canonical render order. Rows `0..15` address the first hacking column and rows `16..31` address the second. `start` and `end` are zero-based inclusive character offsets within that row's 12-character hacking text.

Internal code translates without exposing a public column field:

```text
columnIndex = row / 16
rowInColumn = row % 16
absoluteStart = rowInColumn * 12 + start
absoluteEnd = rowInColumn * 12 + end
```

Validation requires `0 <= row < 32`, `0 <= start < end < 12`, both translated offsets in the same row, and the current canonical text at those offsets to satisfy discovery. The current two-column geometry remains an implementation constraint rather than a public pattern field.

## Private Runtime Entities

### PuzzleGenerationID

| Property | Type | Rules |
|---|---|---|
| value | non-blank opaque string | Unique enough to prevent reuse for another puzzle accepted by the same or a restarted process |
| lifetime | runtime-only | Created before constructing a fresh puzzle; retained through updates and reconnects; discarded when live state is replaced or cleared |
| source | independent generation-ID source | Production uses a collision-resistant process-local source; tests may inject a deterministic sequence; never consumes the pattern-outcome `Random` source |

### PatternIdentity

`PatternIdentity` is a comparable private value and the key type for used history.

| Field | Type | Validation |
|---|---|---|
| `GenerationID` | string | Equals the active `HackState.GenerationID` |
| `Row` | integer | Valid rendered-row ordinal |
| `Start` | integer | Inclusive opening-character offset within `Row` |
| `End` | integer | Inclusive closing-character offset within `Row`; greater than `Start` |

The server-issued public ID is a stable opaque encoding or resolver key for this complete value. Its wire representation is not parsed or synthesized by the browser.

### HackPattern

`HackPattern` is a currently discovered private span.

| Field | Type | Rules |
|---|---|---|
| `Identity` | `PatternIdentity` | Complete active-generation identity |
| `Pair` | string | Private derived value; exactly one of `()`, `[]`, `{}`, `<>` |
| `ColumnIndex` | integer | Private coordinate translation for current board storage |
| `AbsoluteStart` | integer | Private inclusive offset into `HackColumn.Text` |
| `AbsoluteEnd` | integer | Private inclusive offset into `HackColumn.Text` |

A span is discovered only when the opening delimiter and first compatible closer to its right are within one rendered row and the exclusive interior has no ASCII alphabetic character. The board is byte-based and contains uppercase ASCII candidate words, so the existing `A-Z`/`a-z` classification remains sufficient without adding localization behavior.

Every opening is evaluated independently. Two openings may therefore have different identities and the same closing offset. If mutation changes the first compatible closer for one opening, the new `End` produces a new identity.

### HackState

`HackState` remains the canonical aggregate mutated by `internal/hack` while `internal/live.Service` owns the mutex.

| Field | Type | Rules |
|---|---|---|
| `GenerationID` | string | Non-blank active puzzle generation; private except as part of opaque pattern IDs |
| `Level` | integer | Existing hacking difficulty; does not alter the initial `3–6` rule |
| `WordLength` | integer | Existing level-derived candidate length |
| `AttemptsMax` | integer | Existing configured maximum, currently 4 |
| `AttemptsLeft` | integer | Range `0..AttemptsMax` |
| `SecretWord` | string | Private and never eligible for dud removal |
| `WordsByID` | map of string to `HackCandidate` | Private current candidate lookup |
| `UsedPatterns` | set of `PatternIdentity` | Complete accepted-history set for this generation; never cleared by board mutation |
| `Solved` | boolean | Existing terminal success state |
| `Failed` | boolean | Existing terminal failure state |
| `Log` | array of string | Existing process-local shared activity log |
| `Columns` | array of `HackColumn` | Existing current rendered board and selectable word placements |

Starting a fresh puzzle creates a new `GenerationID` and empty `UsedPatterns`. Updating published terminal content preserves both. Clearing live state or restarting the process discards both.

### HackCandidate

| Field | Type | Rules |
|---|---|---|
| `Text` | string | Existing uppercase candidate word of the configured length |

An incorrect candidate is a currently placed candidate whose text differs from `SecretWord`. Dud removal selects only from that current private set.

### PatternOutcome

`PatternOutcome` is selected after used-state insertion and before effect application. It is never preassigned to a pattern or included in an unused public projection.

| Selected outcome | Random mapping | Applied mutation |
|---|---|---|
| Dud removal | `Intn(100)` returns `0..79` | If a removable incorrect candidate exists, use a separate candidate-selection draw, replace exactly that word with periods, remove its private/public placement, preserve the secret and attempts, then rediscover. If none exists, apply attempt restoration instead. |
| Attempt restoration | `Intn(100)` returns `80..99` | Set `AttemptsLeft` to `AttemptsMax`; board and candidates remain unchanged. |

Every accepted activation consumes exactly one outcome value. Invalid, stale-generation, non-current, terminal-state, already-used, and concurrent duplicate requests consume zero values. A selected dud removal consumes a second value only when at least one eligible dud exists.

## Public Runtime Entities

### PublicHackPattern

The public pattern projection contains only:

| JSON field | Type | Rules |
|---|---|---|
| `id` | string | Stable opaque public identity containing or resolving to the complete `PatternIdentity` |
| `row` | integer | Rendered-row ordinal |
| `start` | integer | Inclusive opening offset within the row |
| `end` | integer | Inclusive closing offset within the row |
| `used` | boolean | True when the complete identity exists in private used history; false when currently available |

It deliberately excludes `column`, `pair`, `generationId` as a separately editable field, the password, dud identities, future effects, outcome values, candidate truth, and private maps or slices.

### PublicHackState

The existing public fields remain: `level`, `wordLength`, `attemptsMax`, `attemptsLeft`, `solved`, `failed`, `log`, and `columns`. The `patterns` field is an array of detached `PublicHackPattern` values for every currently discovered span, sorted by `row`, `start`, then `end`.

Public columns retain existing addresses, text, and word placements because ordinary board rendering and guessing are unchanged. Word placements reveal no correct/dud classification. Every public slice and nested mutable value is copied before leaving the canonical boundary.

## Relationships

```text
LiveState 1 ── 0..1 HackState
HackState 1 ── 1 PuzzleGenerationID
HackState 1 ── 2 HackColumn
HackState 1 ── * HackCandidate (private current lookup)
HackState 1 ── * PatternIdentity (private used history)
HackState 1 ── * HackPattern (derived from current columns)
PublicLiveState 1 ── 0..1 PublicHackState
PublicHackState 1 ── * PublicHackPattern (current derived spans only)
```

## State Transitions

### Puzzle Generation and Publication

```text
issue fresh generation ID without using outcome Random
  → construct candidate words and final rendered board attempt
  → run production discovery on final board
  → discovered count < 3 or > 6? discard attempt and regenerate
  → discovered count in 3..6? publish fresh active HackState
```

Intended bracket insertion and bracket-free filler may reduce retries, but only final production discovery decides publication. No target-count distribution becomes part of the contract.

### Atomic Pattern Activation

While the canonical live-service mutex remains held:

```text
1. require request identity generation == active HackState.GenerationID
2. rediscover current board and require the exact row/start/end identity
3. require identity absent from UsedPatterns
4. insert complete identity into UsedPatterns
5. consume one Intn(100) outcome value
6. apply dud removal, attempt restoration, or no-dud restoration fallback
7. rediscover patterns affected by any board mutation
8. create a fully detached PublicHackState
9. invoke the ordered publication callback once with that detached state
```

Any failure in steps 1–3 exits before used insertion, random selection, log changes, board/candidate mutation, projection, or publication. Concurrent duplicate requests serialize at the mutex: exactly one reaches step 4.

### Dynamic Discovery and Used History

```text
current opening pairs with closer A → identity G/R/S/A
accept activation → store G/R/S/A in UsedPatterns
board mutation makes closer B first → identity G/R/S/B is new and available
later mutation restores closer A → identity G/R/S/A is rediscovered as used
fresh puzzle generation H at same R/S/A → H/R/S/A is a distinct available identity
```

### Reconnect and Restart

```text
reconnect to same running process → TERMINAL_LIVE carries current detached board, attempts, log, outcome, and current pattern statuses
fresh Set in same process → new generation and empty used history
server process restart → active puzzle and all pattern runtime state are discarded
```

## Persistence Boundary

Version-1 session JSON remains limited to the existing campaign and terminal fields, including durable `terminal.hackLevel`. It gains no generation ID, board text, pattern projection, used history, candidate removal, attempt count, log, outcome, unlocked state, or complete puzzle seed.
