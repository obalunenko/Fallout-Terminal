---
status: migrated
feature: Player Terminal Presentation
source: existing implementation
---

# Tasks: Player Terminal Presentation

All implementation tasks below are marked complete because they reconstruct work already present in the repository. The final section records gaps; it does not claim those follow-ups were implemented.

## Phase 1 — Static Shell and CRT Styling

- [x] T001 Create separate connection, idle, normal terminal, record, command-output, hacking, blocked, footer, and prompt regions in `client/index.html`.
- [x] T002 Bundle Fixedsys and implement the green CRT frame, glow, scanlines, vignette, blink, flicker, highlights, and themed scrolling in `client/client.css` and `client/fonts/`.
- [x] T003 Add flexible viewport sizing, `clamp()` typography, bounded content panes, and overflow handling for the player shell in `client/client.css`.

## Phase 2 — Connection and Presentation State

- [x] T004 Open a same-host `ws:` or `wss:` connection, expose connection-loss feedback, and schedule reconnects after three seconds in `client/client.js`.
- [x] T005 Dispatch terminal, navigation, hacking, and clear messages into local presentation mirrors in `client/client.js`.
- [x] T006 Render mutually exclusive idle, list, record, hacking, and blocked states from the current live mode in `client/client.js`.
- [x] T007 Render terminal headings, optional introduction text, folder rows, empty directories, record bodies, command output, prompts, and back-control visibility in `client/client.js`.
- [x] T008 Escape HTML assembled for the hacking board and use text nodes or `textContent` for authored terminal data in `client/client.js`.

## Phase 3 — Pointer, Keyboard, and Reveal Interaction

- [x] T009 Add local pointer hover selection, click activation, and back-button handling for normal terminal views in `client/client.js`.
- [x] T010 Add bounded Arrow Up/Down selection and Enter/Escape/Backspace behaviors for list and record modes in `client/client.js`.
- [x] T011 Send `NAV_ACTION`, `HACK_GUESS`, and `HACK_ADMIN` requests without mutating canonical server-owned navigation or puzzle state in `client/client.js`.
- [x] T012 Render hacking attempts, addresses, grouped word/filler targets, shared log, input preview, and blocked state in `client/client.js`.
- [x] T013 Add identity-keyed progressive reveal for new folder rows, record lines, and command-output lines while suppressing replay for unchanged content in `client/client.js`.

## Phase 4 — Atmospheric Audio

- [x] T014 Add an allowlisted sound-folder discovery endpoint and supported media-extension filter in `server/server.js`.
- [x] T015 Prefetch sound assets, lazily decode and cache Web Audio buffers, and select category files in `client/sound.js`.
- [x] T016 Map menu focus, character hover, entry, incorrect guess, correct guess, and reveal events to sound categories and configured volumes in `client/sound.js` and `client/client.js`.
- [x] T017 Add a click-gated looping ambient track that starts with live presentation when permitted and pauses on terminal clear in `client/sound.js` and `client/client.js`.
- [x] T018 Treat missing folders, failed fetches, decode errors, and rejected playback as non-blocking optional-audio failures in `client/sound.js`.

## Gaps Identified During Migration

These are observations for follow-up specification and planning, not completed tasks:

1. No automated browser, DOM, accessibility, visual-regression, or sound-endpoint tests exist.
2. No explicit narrow-screen breakpoint adapts the two-column hacking board and proportional log panel for phones.
3. Terminal rows and hacking cells lack semantic interactive roles, accessible names, and native focus behavior.
4. Flicker, blink, and reveal animations do not honor `prefers-reduced-motion`.
5. Keyboard-only interaction does not unlock the ambient audio loop because the gesture flag is click-only.
6. Optional-audio failures have no diagnostic or visible muted state.
7. Player-facing strings are hardcoded and no localization mechanism exists.
