---
status: migrated
feature: Player Terminal Presentation
source: existing implementation
---

# Implementation Plan: Player Terminal Presentation

**Migration status**: Reconstructed from existing code; this is not a proposal to rewrite the feature.  
**Specification**: `specs/player-terminal-presentation/spec.md`

## Summary

The implemented feature is a dependency-free browser presentation layer served by the application's Express server. Static markup defines mutually exclusive terminal states, CSS supplies a responsive CRT treatment, and browser JavaScript converts authoritative WebSocket snapshots into visible content. Local pointer and keyboard state provides immediate highlighting, but shared navigation and hacking changes wait for server broadcasts. A separate browser audio module discovers allowlisted assets, prefetches them, and plays event-specific feedback without blocking the visual experience.

## Technical Context

| Area | Existing choice |
|---|---|
| Language | Browser JavaScript with globals and two-space indentation |
| Markup and styling | Static HTML5 and CSS; bundled Fixedsys font with monospace fallback |
| Hosting | Express static middleware from `client/` |
| Live transport | Native browser WebSocket using `ws:` or `wss:` according to page protocol |
| Audio | Native Fetch, Web Audio API, and `HTMLAudioElement` |
| Rendering | Direct DOM updates; no UI framework or template dependency |
| Presentation state | Local browser variables mirroring server state plus local selection/hover/reveal state |
| Tests | No test framework, browser harness, CI workflow, linter, formatter, or coverage target is configured |
| Available commands | `npm start`; `npm run build:dir` for applicable Windows packaging checks |

## Detected Scope

### Player markup and assets

- `client/index.html` — CRT shell, connection overlay, normal header, idle/list/entry/output states, hacking state, back control, and script loading.
- `client/fonts/Fixedsys.ttf` — bundled terminal font.
- `client/sounds/` — ambient loop and event-specific WAV assets, plus folder convention documentation.

### Presentation and interaction

- `client/client.css` — frame, glow, scanlines, vignette, responsive font sizing, state layouts, highlights, scrolling, and animation.
- `client/client.js` — connection lifecycle, presentation state, WebSocket dispatch, pointer/keyboard interaction, text-safe rendering, reveal sequencing, and hacking-board projection.

### Audio integration

- `client/sound.js` — sound-folder discovery, prefetch, decode/cache, random selection, event volumes, browser gesture gate, and ambient lifecycle.
- `server/server.js` — static client hosting and the allowlisted `GET /api/sounds/:folder` discovery endpoint.

### Related but out of scope

- `server/nav.js` owns canonical navigation validation and mutation.
- `server/hack.js` and `server/wordbank.js` own hacking rules and private puzzle data.
- `master/`, `preload.js`, and `main.js` own game-master controls, Electron boundaries, persistence, and broadcast initiation.
- The migrated `specs/hacking-game/` artifacts specify the hacking domain; this feature specifies only its player-facing projection and interaction affordances.

## Existing Architecture and Data Flow

1. Express serves `client/index.html`, CSS, scripts, fonts, and audio files from the static client directory.
2. On boot, `client.js` renders the idle state and opens a same-host WebSocket, selecting `ws:` or `wss:` from the page protocol.
3. The connection overlay disappears on open, returns on close, and remains visible while a three-second reconnect timer is pending.
4. `TERMINAL_LIVE`, `TERMINAL_UPDATE`, `NAV_STATE`, `HACK_STATE`, and `TERMINAL_CLEAR` messages replace or clear local mirrors of server state and invoke the relevant renderer.
5. Pointer and keyboard handlers update local highlight, hover, and text-preview state, then send navigation or hacking requests; shared mode/path/content changes wait for server messages.
6. Render functions show one compatible state at a time and build content with `textContent` or explicit escaping where HTML assembly is used.
7. Folder rows, record lines, and command-output lines reveal progressively only when their identity key changes; a container cancels any prior reveal timer before replacement.
8. `sound.js` asks the server for allowlisted filenames, prefetches raw data, lazily decodes one-shot buffers, and degrades silently if audio is unavailable.
9. A document click unlocks the ambient loop. A live terminal starts it when ready, and `TERMINAL_CLEAR` pauses it.

## Reconstructed Implementation Phases

### Phase 1 — Static terminal shell and CRT presentation

- Added the player document with separate normal, hacking, idle, blocked, connection, output, footer, and prompt regions.
- Bundled Fixedsys and applied a green phosphor palette, glow, scanlines, vignette, blink, and flicker effects.
- Used flexible sizing, viewport-relative spacing, `clamp()` typography, bounded containers, and themed scrollbars.

### Phase 2 — Connection and state-driven rendering

- Opened a same-origin WebSocket with secure-protocol selection and reconnect feedback.
- Added dispatch handlers for the server's terminal, navigation, hacking, and clear messages.
- Projected current state into mutually exclusive idle, list, record, hacking, and blocked views.
- Rendered untrusted authored text through text nodes or escaping helpers.

### Phase 3 — Player interaction and reveal feedback

- Added pointer selection, hover highlighting, clicking, back-button handling, and document-level keyboard controls.
- Kept only selection and hover local while emitting requests for authoritative navigation and hacking changes.
- Added identity-keyed progressive reveal for folders, records, and command output.
- Added hacking target grouping, input preview, attempt display, shared log, and delayed solved transition.

### Phase 4 — Optional atmospheric audio

- Established allowlisted server discovery for named sound categories and supported file extensions.
- Prefetched raw sound data and lazily cached decoded Web Audio buffers.
- Mapped focus, character, entry, failure, success, and reveal events to category-specific playback and volume.
- Added a click-gated looping ambient track and stopped it with the live terminal lifecycle.
- Treated unavailable audio as a non-blocking enhancement failure.

## Key Technical Decisions

1. **The browser client is framework-free** to keep the player payload and runtime dependencies small.
2. **One static DOM shell contains all states**, with render functions controlling visibility instead of page navigation.
3. **Server messages own shared state**, while highlight, hover, typed preview, and animation bookkeeping remain local.
4. **The page protocol selects the WebSocket protocol**, allowing the same client to work over local HTTP and public HTTPS.
5. **Text safety is handled at render boundaries** with `textContent` for DOM construction and a shared escape helper for generated board HTML.
6. **Reveal identity keys suppress unnecessary replay**, preserving atmosphere without restarting animation on every server update.
7. **Sound categories are folder-driven and server-allowlisted**, allowing asset variation without exposing arbitrary filesystem paths.
8. **One-shot audio is prefetched and decoded lazily**, while ambient playback uses a looping `Audio` element.
9. **Audio failures are deliberately non-fatal**, so presentation and interaction remain available when media support or assets fail.
10. **Autoplay restrictions are accepted as a platform boundary**, with ambient playback gated on the first document click.

## Constitution Check

| Principle | Assessment |
|---|---|
| Preserve runtime boundaries | Pass: all presentation logic remains under `client/`; the browser uses only browser APIs, static assets, HTTP, and WebSocket contracts. |
| Keep shared state server-authoritative | Pass: local input sends requests and waits for `NAV_STATE`, `TERMINAL_UPDATE`, or `HACK_STATE` before shared transitions. |
| Protect desktop/public boundaries | Pass for the detected scope: the player page receives no Electron or Node.js access, and sound discovery uses an allowlist. The page has no explicit Content Security Policy. |
| Preserve session compatibility | Not applicable: the feature reads runtime projections and does not alter session JSON. |
| Match established conventions | Pass: browser scripts use globals, camelCase, single quotes, semicolons, and two-space indentation; CSS classes use kebab-case. |

## Complexity Assessment

| Measure | Assessment |
|---|---|
| Primary source size | 1,055 lines across `index.html`, `client.css`, `client.js`, and `sound.js` |
| Supporting assets | One font, 20 WAV files, and sound-folder documentation |
| Server integration | One static mount and one allowlisted sound-discovery route |
| Runtime boundaries crossed | Browser presentation and server HTTP/WebSocket contracts |
| State complexity | Moderate: mutually exclusive modes, reconnect lifecycle, server mirrors, local highlights, reveal timers, and audio caches |
| Dependency depth | Low: native browser APIs plus the application's existing Express/WebSocket server |

No constitution violation requiring a complexity exception was detected. Presentation of server states and a small allowlisted asset endpoint are inherent to the browser experience and stay within the modular monolith's declared boundaries.

## Verification Strategy for the Existing Feature

No automated checks can currently be claimed. Proportionate verification for future changes should include:

1. Run `npm start` and open the player URL in at least two modern browsers or browser profiles.
2. Verify initial connection, idle, live, clear, forced disconnect, three-second reconnect, and restored-live-state presentation.
3. Broadcast a tree containing nested folders, an empty folder, long names, a multiline record, and multiline command output; exercise pointer and supported keyboard controls.
4. Confirm that player input sends requests without visible shared transitions until authoritative messages arrive, and compare both clients after navigation.
5. Broadcast hacking-enabled terminals and inspect active, solved, failed, hover, split-word, log, and delayed-unlock presentation.
6. Use authored text containing HTML-like markup and confirm it is displayed literally without creating elements or executing script.
7. Verify reveal animation plays for newly selected content, does not restart for unchanged content, and cancels cleanly when the view changes quickly.
8. Test with one missing or invalid sound file and one rejected sound folder; confirm the visual interface remains usable.
9. Test a click before and after live publication to verify ambient start, loop, and clear-time pause behavior under browser autoplay rules.
10. Inspect narrow phone-sized and large desktop viewports, content overflow, zoom, and scrollbar behavior; record the current hacking-layout limitations.
11. Perform keyboard-only and browser accessibility inspection, explicitly recording the current semantic, focus, and reduced-motion gaps.
12. Run `npm run build:dir` only when the required Windows packaging environment is available and changes affect bundled player assets or packaging-sensitive paths.

Before adding automated tests, a future plan must name the browser or DOM test framework, test location, and npm command because the repository currently defines none.

## Identified Follow-up Gaps

- Establish a browser/DOM testing strategy, visual checks, a focused sound-endpoint test, and a documented npm test command.
- Define supported viewport widths and adapt the hacking layout explicitly for narrow screens.
- Replace or augment clickable `div`/`span` elements with semantic controls, focus behavior, and accessible names.
- Honor reduced-motion preferences for flicker, cursor blink, and progressive reveal.
- Decide whether keyboard input should also satisfy the ambient user-gesture gate and provide a visible mute control.
- Add development diagnostics or an observable muted state without making optional audio a blocking dependency.
- Decide whether Russian-only copy is a permanent product constraint or should move behind a localization mechanism.
