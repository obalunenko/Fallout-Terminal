# Data Model: Hacking Game Evolution

## Modeling Principles

- The private hacking aggregate remains process-local and canonical; public projections are detached copies.
- A special pattern is identified by its exact column, opening position, and closing position, not by displayed text.
- Pattern validity is derived from the current visible board, while one-use history is retained in private state for the life of the puzzle.
- Candidate truth and random outcome selection remain server-only.
- A new live broadcast creates all pattern state from scratch; session JSON is unchanged.

## Constants and Verbatim Rules

| Rule | Exact value |
|---|---|
| Allowed pattern pairs | `()`, `[]`, `{}`, `<>` |
| Valid special-pattern count per newly generated board | `3–6` inclusive |
| Dud-removal probability | `80%` |
| Attempt-restoration probability | `20%` |
| Board geometry | Two columns; 16 rows per column; 12 characters per row |

## Private Runtime Entities

### HackState

`HackState` remains the aggregate mutated by `internal/hack` while the live service holds its mutex.

| Field | Type | Rules |
|---|---|---|
| `Level` | integer | Existing hacking difficulty; does not alter the special-pattern count range |
| `WordLength` | integer | Existing level-derived candidate length |
| `AttemptsMax` | integer | Existing maximum, currently 4 |
| `AttemptsLeft` | integer | Range `0..AttemptsMax` |
| `SecretWord` | string | Private; always names one current candidate and is never removed as a dud |
| `WordsByID` | map of string to `HackCandidate` | Private lookup for current candidate words only; contains no administrator entry |
| `UsedPatterns` | set of pattern ID | Private one-use history for the current puzzle; reset only with a fresh `HackState` |
| `Solved` | boolean | Terminal success state |
| `Failed` | boolean | Terminal failure state |
| `Log` | array of string | Shared public activity log |
| `Columns` | array of `HackColumn` | Current visible board, addresses, and selectable candidate placements |

The removed fields are `AdminModeUsed`, `HackCandidate.IsAdmin`, and `HackWord.IsAdmin`.

### SpecialPattern

| Field | Type | Validation |
|---|---|---|
| `ID` | string | Canonical `columnIndex:openingIndex:closingIndex`, for example `0:17:23` |
| `Column` | integer | Existing column index, currently `0` or `1` |
| `Start` | integer | Inclusive opening-bracket offset in the column text |
| `End` | integer | Inclusive closing-bracket offset; greater than `Start` |
| `Pair` | string | Exactly one of `()`, `[]`, `{}`, `<>` |
| `Used` | boolean | Derived from membership in `HackState.UsedPatterns` |

A discovered pattern is valid only when:

1. `Start / 12 == End / 12`, so both endpoints are on the same row.
2. The character at `Start` is the opener from `Pair`.
3. `End` is the first matching closer to the right of `Start` on that row.
4. No ASCII alphabetic character occurs strictly between `Start` and `End`.

Every opening position is scanned independently. Therefore two openings may have different IDs and share one closing position. A used identity remains used even if later board discovery encounters the same pair again.

### HackCandidate

| Field | Type | Rules |
|---|---|---|
| `Text` | string | Uppercase candidate word of `WordLength`; no administrator marker |

An incorrect candidate is one whose text differs from `SecretWord`. Dud removal chooses only from current incorrect candidates.

### PatternEffect

`PatternEffect` is a transition result, not persisted state.

| Value | Mutation |
|---|---|
| Dud removed | Delete exactly one incorrect candidate from `WordsByID` and its column's `Words`; replace its visible characters with periods; keep attempts unchanged; rediscover patterns |
| Attempts restored | Set `AttemptsLeft` to `AttemptsMax`; keep the board and candidates unchanged |

The domain marks the pattern used before applying either effect. If no incorrect candidate exists, the effective result is always attempts restored.

## Public Runtime Entities

### PublicHackState

The existing public fields remain: `level`, `wordLength`, `attemptsMax`, `attemptsLeft`, `solved`, `failed`, `log`, and `columns`. Add:

| JSON field | Type | Rules |
|---|---|---|
| `patterns` | array of `PublicSpecialPattern` | All currently valid spans, in deterministic column/start/end order, including used spans |

It continues to exclude `secretWord`, `wordsById`, random values, and the private used-pattern set.

### PublicSpecialPattern

| JSON field | Type | Rules |
|---|---|---|
| `id` | string | Canonical coordinate identity |
| `column` | integer | Column containing the span |
| `start` | integer | Inclusive opening offset |
| `end` | integer | Inclusive closing offset |
| `pair` | string | One exact allowed pair |
| `used` | boolean | True after the first accepted activation of this identity |

The projection contains enough information to highlight overlapping ranges without exposing which word is correct or which dud will be removed.

## Relationships

```text
LiveState 1 ── 0..1 HackState
HackState 1 ── 2 HackColumn
HackState 1 ── * HackCandidate (private lookup)
HackState 1 ── * SpecialPattern (derived from current columns)
HackState 1 ── * used coordinate identities (private history)
PublicLiveState 1 ── 0..1 PublicHackState
PublicHackState 1 ── * PublicSpecialPattern
```

## State Transitions

### Puzzle Lifecycle

```text
no puzzle
  -- set live with hackLevel > 0 --> active fresh puzzle
active fresh puzzle
  -- guess --> active / solved / failed
active fresh puzzle
  -- valid unused pattern --> active with pattern used and one effect
active
  -- game-master ForceHackSuccess --> solved
active / solved / failed
  -- fresh set live --> new active fresh puzzle with empty UsedPatterns
any puzzle
  -- clear live / shutdown --> no puzzle
```

### Pattern Activation

```text
request patternId
  → require active non-terminal puzzle
  → rediscover current valid patterns
  → require exact ID match
  → require ID absent from UsedPatterns
  → insert ID into UsedPatterns
  → no incorrect candidate? restore attempts
  → otherwise Intn(100) in 0..79? remove one random incorrect candidate
  → otherwise restore attempts
  → rediscover patterns after any board mutation
  → return detached PublicHackState
```

Malformed, unknown, stale, tampered, repeated, solved-state, and failed-state pattern requests stop before the used-state insertion and leave the entire aggregate unchanged.

## Generation Invariants

1. Place ordinary candidate words without an administrator candidate.
2. Fill remaining cells from a pool that excludes alphabetic characters and all eight allowed bracket characters.
3. Select `3 + Intn(4)` as the target valid-pattern count.
4. Insert isolated same-row pattern spans only into free, non-word cells, using the exact allowed pairs and bracket-free interiors.
5. Run production discovery over the finished board and return it only when the discovered count equals the target.

The same target rule is used at every hacking level. Tests exercise 1,000 deterministic generations across all levels and verify the `3–6` bound, row rule, first-compatible-close rule, and absence of alphabetic interiors.

## Persistence

No field in this document is added to version-1 session JSON. Only the existing durable `terminal.hackLevel` remains persisted. Board text, patterns, used identities, candidate removal, attempts, log, and outcome are discarded when live state is cleared or replaced.
