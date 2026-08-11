# Hacking Interface Contract

## Scope

This contract changes the player WebSocket hacking interface and the public hack projection. The static HTTP player route, navigation protocol, session JSON, and game-master desktop method name remain unchanged.

## Exact Gameplay Constants

- Allowed pattern pairs: `()`, `[]`, `{}`, `<>`.
- Valid special-pattern count per newly generated board: `3–6` inclusive.
- Dud-removal probability: `80%`.
- Attempt-restoration probability: `20%`.

## Player-to-Server Messages

### `HACK_GUESS` — retained

```json
{
  "type": "HACK_GUESS",
  "targetId": "A1"
}
```

`targetId` remains a non-blank candidate ID or `columnIndex:characterIndex` filler coordinate. Guess validation, likeness, attempt spending, success, and failure behavior are unchanged. There is no generated administrator candidate.

### `HACK_PATTERN` — added

```json
{
  "type": "HACK_PATTERN",
  "patternId": "0:17:23"
}
```

| Field | Required | Validation |
|---|---|---|
| `type` | yes | Exact string `HACK_PATTERN` |
| `patternId` | yes | Non-blank string; semantic form is `columnIndex:openingIndex:closingIndex` |
| any other field | prohibited | Strict decoder ignores the malformed request before dispatch |

The server accepts the action only when an active, unsolved, unfailed puzzle exists and production discovery finds the exact currently valid coordinate identity with `used == false`. Acceptance atomically marks the identity used and applies exactly one effect. A malformed, unknown, stale, tampered, repeated, solved-state, or failed-state action changes no board text, candidates, used state, attempts, log, or outcome and emits no `HACK_STATE`.

### `HACK_ADMIN` — removed

`HACK_ADMIN` is no longer a supported player message. The strict decoder ignores it as an unsupported type, and it can never mutate the puzzle. The player keyboard no longer translates `1` or any other typed command into a hacking action.

## Server-to-Player Messages

### `HACK_STATE` — extended

The envelope remains:

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
        "id": "0:17:23",
        "column": 0,
        "start": 17,
        "end": 23,
        "pair": "[]",
        "used": false
      }
    ]
  }
}
```

Production `columns` retain the existing two-column shape with `addresses`, `text`, and `words`. A word contains `id`, `start`, and `length`; the obsolete `isAdmin` property is removed. `patterns` is always present for a non-null puzzle, may grow after dud removal, and is sorted by `column`, `start`, then `end`. A just-consumed pattern remains present with `used: true`.

The public object MUST NOT contain `secretWord`, `wordsById`, the private used-pattern set, an outcome roll, or the selected dud before mutation.

### `TERMINAL_LIVE` — retained and transitively extended

The existing full live envelope retains `terminalId`, `terminalName`, `tree`, `hackLevel`, `introText`, `hack`, and `nav`. When `hack` is non-null, it has the exact extended public shape above. A late or reconnecting player receives the current board, current attempts and outcome, all current patterns with their `used` flags, and no regenerated puzzle.

## Publication Rules

| Event | Broadcast |
|---|---|
| Accepted first activation of a valid pattern | One `HACK_STATE` containing the fully applied effect and updated pattern state to every connected player; one detached `hack-state` event to the game-master frontend |
| Simultaneous second activation of the same pattern | None; canonical state is unchanged |
| Repeated, stale, tampered, malformed, or terminal-state activation | None; canonical state is unchanged |
| Fresh live broadcast | One `TERMINAL_LIVE` with a newly generated puzzle and no used patterns |
| Reconnection during a puzzle | One `TERMINAL_LIVE` snapshot of the existing puzzle |
| Game-master solve | Existing `ForceHackSuccess` flow publishes solved `HACK_STATE` and `hack-state` without consuming an attempt |

All mutation and acceptance decisions occur while the canonical live service holds its exclusive mutex. Network writes occur only after the detached snapshot is returned.

## Player UI Contract

- The browser consumes `hack.patterns`; it does not discover canonical patterns locally.
- Hovering the opening cell of an unused pattern highlights every character from `start` through `end`, inclusive, and shows the complete span in the existing input preview.
- Stacked patterns are selected by opening coordinate, so different openings sharing one close highlight different ranges.
- A used pattern does not highlight. Activating its opening resends its `patternId`, allowing the server to reject it without falling through to `HACK_GUESS` or consuming an attempt.
- Clicking an unused pattern opening sends `HACK_PATTERN`. Clicking candidate words or cells that are not pattern openings retains `HACK_GUESS`.
- The browser changes no canonical board, attempts, pattern, log, or outcome state until `HACK_STATE` arrives.
- The former typed administrator command and the `SUCCESS` board entry are absent.

## Game-Master Interface Contract

The trusted desktop contract remains `ForceHackSuccess()` through the generated Wails `App` binding. It is eligible only for an active unsolved, unfailed puzzle, does not consume an attempt, returns the existing command result shape, and publishes through the same public success flow. The existing `#btnHackSuccess` control remains disabled when no eligible puzzle exists.

## Compatibility Impact

The server and bundled player are released together. Existing `HACK_GUESS`, `HACK_STATE`, and `TERMINAL_LIVE` message names remain stable; `patterns` is additive. Clients that still send `HACK_ADMIN` receive no mutation or response for that unsupported request. Consumers of word placement JSON must stop expecting `isAdmin`; no replacement administrator marker exists.
