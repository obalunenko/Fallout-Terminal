---
status: migrated
feature: Hacking Game
source: existing implementation
---

# Tasks: Hacking Game

All implementation tasks below are marked complete because they reconstruct work already present in the repository. The final section records gaps; it does not claim those follow-ups were implemented.

## Phase 1 — Word Bank and Board Domain

- [x] T001 Create length-indexed, deduplicated hacking word pools for 4–8 character candidates in `server/wordbank.js`.
- [x] T002 Implement random word selection without duplicates in `server/wordbank.js`.
- [x] T003 Generate two 16-row, 12-character board columns with randomized filler and hexadecimal addresses in `server/hack.js`.
- [x] T004 Place difficulty-scaled candidate words and the administrator `SUCCESS` entry with unique public target IDs in `server/hack.js`.
- [x] T005 Retain the secret answer and ID lookup in private server state while exposing a sanitized public projection in `server/hack.js`.

## Phase 2 — Puzzle Rules

- [x] T006 Implement four shared attempts, positional character matching, success logs, failure logs, and terminal outcomes in `server/hack.js`.
- [x] T007 Handle validated filler-coordinate selections and reject malformed, out-of-range, or stale word-overlap targets in `server/hack.js`.
- [x] T008 Implement the one-time administrator board reduction while preserving the answer and at most one decoy in `server/hack.js`.
- [x] T009 Implement game-master forced success and block mutations after solved or failed outcomes in `server/hack.js`.

## Phase 3 — Server Lifecycle and Shared Protocol

- [x] T010 Attach runtime puzzle state to the canonical live terminal and generate a fresh board on broadcast start/restart in `server/server.js`.
- [x] T011 Accept typed `HACK_GUESS` and `HACK_ADMIN` WebSocket actions and apply them only when a live puzzle exists in `server/server.js`.
- [x] T012 Broadcast sanitized `HACK_STATE` after player or master actions and notify the master process of progress in `server/server.js`.
- [x] T013 Include current sanitized state in `TERMINAL_LIVE` for connected and reconnecting players in `server/server.js`.
- [x] T014 Clear puzzle state when broadcasting stops and preserve it during ordinary live tree/text updates in `server/server.js`.

## Phase 4 — Player Hacking Experience

- [x] T015 Add player hacking and blocked-state markup in `client/index.html` and presentation styles in `client/client.css`.
- [x] T016 Render board addresses, candidate spans, filler targets, remaining attempts, shared log, and input preview in `client/client.js`.
- [x] T017 Send word/filler guesses from pointer input and administrator requests from keyboard input in `client/client.js`.
- [x] T018 Apply `TERMINAL_LIVE` and `HACK_STATE` broadcasts without mutating canonical puzzle state on the client in `client/client.js`.
- [x] T019 Gate normal navigation until success, show the blocked screen after failure, and transition after the success delay in `client/client.js`.
- [x] T020 Add hover, character-entry, incorrect-guess, correct-guess, and ambient audio behavior in `client/sound.js` and `client/sounds/`.

## Phase 5 — Master Controls and Electron Integration

- [x] T021 Add difficulty levels 0–5 and live hacking status controls in `master/index.html` and `master/master.css`.
- [x] T022 Persist per-terminal `hackLevel`, apply configuration, preserve active puzzles during updates, and render live status in `master/master.js`.
- [x] T023 Add sanitized master progress notifications and a narrow forced-success API in `preload.js`.
- [x] T024 Forward server progress to the master renderer and route forced success from Electron IPC to the server in `main.js`.
- [x] T025 Default new terminals to disabled hacking and demonstrate persisted level configuration in `main.js` and `sessions/demo.json`.

## Phase 6 — Existing Verification Evidence

- [x] T026 Keep private `secretWord` and `wordsById` out of the object returned by `publicHackState` in `server/hack.js`.
- [x] T027 Keep runtime puzzle state out of JSON session persistence while retaining the backward-compatible `hackLevel || 0` fallback.
- [x] T028 Preserve Electron `nodeIntegration: false`, `contextIsolation: true`, and `sandbox: true` while exposing only narrow preload methods.

## Gaps Identified During Migration

These are observations for follow-up specification and planning, not completed tasks:

1. No automated test framework or tests cover domain rules, random generation, redaction, session compatibility, protocol validation, or multi-client synchronization.
2. Loaded session and live IPC payloads do not constrain `hackLevel` to an integer from 0 through 5.
3. Repeated administrator requests append duplicate activation log entries.
4. Previously guessed candidates remain active and can consume more attempts.
5. Hacking messages and payloads have no executable schema or protocol version.
6. Random board generation has no deterministic seed or injectable random source for tests.
7. Accessibility requirements and verification are absent for the character-grid interface.

