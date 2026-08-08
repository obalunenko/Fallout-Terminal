---
status: migrated
feature: Hacking Game
source: existing implementation
---

# Implementation Plan: Hacking Game

**Migration status**: Reconstructed from existing code; this is not a proposal to rewrite the feature.  
**Specification**: `specs/hacking-game/spec.md`

## Summary

The implemented feature is a server-authoritative, shared hacking gate within the modular Electron application. Per-terminal difficulty is persisted in session JSON. Starting a live terminal creates an in-memory puzzle in the Express/WebSocket server. Browser clients submit actions and render sanitized broadcasts; the Electron master renderer observes progress and can force success through the preload IPC bridge.

## Technical Context

| Area | Existing choice |
|---|---|
| Language | JavaScript, CommonJS in main/server modules and browser scripts in renderers |
| Desktop runtime | Electron 28 |
| Local transport | Express 4 HTTP server and `ws` 8 WebSocket server |
| Persistence | Versioned JSON session files; only `terminal.hackLevel` is persisted |
| Live state | In-memory `live.hack` object owned by `server/server.js` |
| Player UI | Static HTML/CSS/JavaScript served by Express |
| Master UI | Sandboxed Electron renderer using `preload.js` |
| Tests | No test framework, test directory, lint command, CI workflow, or coverage target is configured |
| Available commands | `npm start`; `npm run build:dir` for applicable Windows packaging checks |

## Detected Scope

### Core domain

- `server/hack.js` — board generation, secret state, guesses, administrator aid, forced success, and public-state projection.
- `server/wordbank.js` — deduplicated word buckets for lengths 4 through 8 and random selection.

### Server and protocol

- `server/server.js` — puzzle lifecycle inside live state, WebSocket action validation, public broadcasts, reconnection snapshot, and master progress callback.

### Player experience

- `client/client.js` — hacking mode, protocol dispatch, hover/click/keyboard input, attempt feedback, delayed unlock, and rendering.
- `client/index.html` — hacking board, shared log, input preview, and blocked state.
- `client/client.css` — board, log, input, and blocked-state presentation.
- `client/sound.js` and `client/sounds/` — good/bad guess and terminal interaction audio.

### Master and Electron boundaries

- `master/master.js` — difficulty configuration, active puzzle status, rebroadcast behavior, and forced success.
- `master/index.html` and `master/master.css` — difficulty controls and live status UI.
- `preload.js` — narrow hack-state listener and forced-success method.
- `main.js` — default persisted difficulty, master notification forwarding, and privileged IPC routing.

### Data example

- `sessions/demo.json` — examples of disabled and level-2 hacking configuration.

## Existing Architecture and Data Flow

1. The game master sets `terminal.hackLevel` in the master renderer and saves the session through Electron IPC.
2. On `terminal:set-live`, the main process delegates to `server.setLiveTerminal(payload)`.
3. The server creates a private puzzle with `generateBoard(level)` and stores it under the canonical in-memory `live` object.
4. The server projects the private puzzle through `publicHackState`, broadcasts it in `TERMINAL_LIVE`, and reports it to the master renderer.
5. A player selects a word/filler target or submits administrator mode; the browser sends `HACK_GUESS` or `HACK_ADMIN` without changing canonical state locally.
6. The server validates the message shape, applies the action, broadcasts sanitized `HACK_STATE`, and forwards progress to the master.
7. Player clients render the broadcast, play feedback audio, and either unlock navigation after success or retain the blocked screen after failure.
8. The game master may route `terminal:hack-force-success` through the preload and main processes to the server.

## Private and Public State

The private board contains `secretWord`, `wordsById`, public counters and flags, log entries, and rendered columns. `publicHackState` is the security boundary that returns only level, word length, counters, flags, log, and rendered columns. Runtime puzzle state is intentionally excluded from session persistence.

## Reconstructed Implementation Phases

### Phase 1 — Domain model and board generation

- Added a curated word bank bucketed by actual word length.
- Generated two 16×12 character columns with randomized filler and hexadecimal addresses.
- Placed ordinary candidates and an administrator `SUCCESS` entry while retaining a private secret lookup.
- Implemented positional match counting, attempt exhaustion, success/failure state, administrator reduction, forced success, and public projection.

### Phase 2 — Server-owned lifecycle and synchronization

- Attached a puzzle to the server's canonical live terminal state.
- Generated a fresh puzzle on live-terminal start/restart and cleared it when broadcasting stops.
- Accepted typed WebSocket actions, mutated shared state, and broadcast sanitized results.
- Included current public state in reconnection snapshots and forwarded progress to the master process.

### Phase 3 — Player experience

- Added a dedicated hacking mode that gates normal navigation.
- Rendered rows, addresses, word targets, filler targets, attempts, shared log, and failure state.
- Added mouse hover/click behavior, keyboard administrator input, audio feedback, and a delayed transition after success.

### Phase 4 — Master controls and persistence

- Added per-terminal difficulty selection and persisted it in session JSON.
- Preserved an in-progress puzzle when ordinary live edits or difficulty settings change.
- Added status reporting for attempts, success, and failure.
- Added a force-success action through the sandboxed preload and main-process IPC boundary.

## Key Technical Decisions

1. **Canonical state stays on the server** so multiple players converge after every action.
2. **The secret is projected out** with a dedicated `publicHackState` function rather than sent to clients.
3. **Puzzle state is runtime-only** while configuration (`hackLevel`) is session-owned persistent data.
4. **Starting a broadcast is the reset boundary**, producing a new random puzzle; live content updates do not reset it.
5. **The board carries rendered text and target locations** so browser clients can remain presentation-only.
6. **All player actions share the group's attempts**, matching the existing shared terminal navigation model.
7. **Administrator mode is server-side and one-time**, protecting the board mutation from client divergence.
8. **Game-master override crosses a narrow preload API**, preserving Electron isolation.
9. **Success transitions locally after 2.6 seconds**, while the solved flag itself remains authoritative and synchronized.

## Constitution Check

| Principle | Assessment |
|---|---|
| Preserve runtime boundaries | Pass: domain logic remains in `server/hack.js`; the browser has no Node access; privileged control uses preload and main IPC. |
| Keep shared state server-authoritative | Pass: attempts, logs, board changes, and outcomes are mutated on the server and broadcast. |
| Protect desktop/public boundaries | Pass for the detected feature: Electron isolation remains enabled and the secret is not published. IPC payload validation remains limited. |
| Preserve session compatibility | Partial risk: `hackLevel` defaults safely when absent, but loaded values are not explicitly validated or migrated. Runtime state is correctly excluded. |
| Match established conventions | Pass: lowercase filenames, CommonJS server modules, browser globals, two-space indentation, and uppercase message types are used. |

## Complexity Assessment

| Measure | Assessment |
|---|---|
| Files touched by the existing feature | 13 code/data files plus sound assets |
| Core domain size | 267 lines across `hack.js` and `wordbank.js` |
| Runtime boundaries crossed | Server domain, WebSocket transport, browser client, Electron master, preload/main IPC, session JSON |
| Dependency depth | Moderate; the pure domain depends only on the word bank, while integration crosses the monolith's main runtime boundaries |
| State complexity | Moderate-high due to random generation, shared multi-client mutation, private/public projections, and terminal lifecycle resets |

No constitution violation requiring a complexity exception was detected. The cross-boundary scope is inherent to the user-visible feature, and the domain logic is already separated from transport.

## Verification Strategy for the Existing Feature

No automated checks can currently be claimed. Proportionate verification for future changes should include:

1. Add focused Node tests for `server/hack.js` and `server/wordbank.js`, using an explicitly chosen test framework and npm command before implementation.
2. Test each level for expected word length, attempt count, board dimensions, and public secret redaction.
3. Test correct, incorrect, filler, malformed, repeated, administrator, forced-success, solved, and failed transitions.
4. Run a manual `npm start` smoke check with two player browsers to verify synchronized actions and reconnection.
5. Verify that difficulty changes do not reset an active puzzle and do apply after rebroadcast.
6. Save and reopen sessions containing missing, valid, and invalid `hackLevel` values to assess compatibility behavior.
7. Inspect `TERMINAL_LIVE` and `HACK_STATE` frames to confirm `secretWord` and `wordsById` are absent.
8. Verify the master status and forced-success control while Electron sandboxing remains enabled.
9. Run `npm run build:dir` only when the required Windows packaging environment is available and the change affects packaging-sensitive surfaces.

## Identified Follow-up Gaps

- Establish focused automated tests and a documented npm test command.
- Validate or normalize persisted and IPC-provided `hackLevel` values to 0 through 5.
- Decide whether repeated administrator activation should be silent or explicitly reported without duplicate log spam.
- Decide whether previously guessed candidates should become inert or visually consumed.
- Define executable schemas or explicit validators for hacking protocol payloads.
- Introduce deterministic random injection if reproducible board tests are added.
- Specify and verify accessibility behavior for the dense character-grid interaction.

