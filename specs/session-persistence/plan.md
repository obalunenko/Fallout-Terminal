---
status: migrated
feature: Session Persistence
source: existing implementation
---

# Implementation Plan: Session Persistence

**Migration status**: Reconstructed from existing code; this is not a proposal to rewrite the feature.  
**Specification**: `specs/session-persistence/spec.md`

## Summary

The implemented feature persists game-master-authored terminal content as versioned JSON. Native dialogs and synchronous filesystem operations live in the Electron main process. A narrow preload bridge exposes create, open, and save requests to the sandboxed master renderer. The renderer retains the current editing model in memory and submits the entire session object after mutations. Development uses the repository `sessions/` directory; packaged builds use a directory next to the executable and opportunistically seed the bundled demo on first run.

## Technical Context

| Area | Existing choice |
|---|---|
| Language/module style | JavaScript; CommonJS in `main.js` and `preload.js`, browser script in the renderer |
| Desktop runtime | Electron 28 |
| Persistence | UTF-8, indented JSON files with an observed `version: 1` field |
| Filesystem API | Synchronous Node.js `fs` operations owned by the Electron main process |
| File selection | Electron native open/save dialogs restricted to `.json` |
| Renderer boundary | `contextBridge` methods backed by `ipcRenderer.invoke` and `ipcMain.handle` |
| Active-file state | Process-global `currentSessionPath` in `main.js` |
| Packaged storage | `sessions/` beside `app.getPath('exe')` |
| Development storage | Repository-local `sessions/` beside `main.js` |
| Tests | No test framework, test directory, lint command, CI workflow, or coverage target is configured |
| Available commands | `npm start`; `npm run build:dir` for applicable Windows packaging checks |

## Detected Scope

### Electron main process

- `main.js` — constructs default sessions, reads and minimally validates JSON, writes JSON, resolves bundled/writable directories, seeds the demo, owns the active path, opens native dialogs, and handles session IPC.

### Sandboxed renderer bridge

- `preload.js` — exposes only `newSession`, `openSession`, and `saveSession` for session persistence.

### Master renderer

- `master/master.js` — invokes session operations, installs loaded state, resets runtime-only UI state, triggers autosave after supported mutations, and renders save success/failure.
- `master/index.html` and `master/master.css` — provide the existing start actions, path label, and save-status presentation; no persistence-specific structural migration is proposed.

### Data and documentation

- `sessions/demo.json` — demonstrates the version-1 session, terminal, folder, command, and entry shapes.
- `README.md` — identifies `sessions/` as saved JSON state and documents development/packaged operation.

No database, database migration, external storage service, or session-specific runtime dependency is present.

## Existing Architecture and Data Flow

1. At window creation, the main process resolves a writable session directory, attempts to create it, and tries to seed the bundled demo if the directory is empty.
2. The master start screen invokes `newSession()` or `openSession()` through `window.electronAPI`.
3. The preload bridge sends a request over `session:new` or `session:open` without exposing Node.js APIs.
4. The main process opens a native dialog. Creation derives a session name, builds a version-1 default, writes it, and records its path. Opening reads UTF-8 JSON, verifies only `terminals` is an array, and records the selected path.
5. The result returns to the renderer as an explicit success, cancellation, or error object.
6. On success, the renderer installs the session and file path, selects the first terminal when present, clears runtime-only live/editing state, and displays the editor.
7. Supported editor mutations update the in-memory object and call `autosave()`.
8. The complete session crosses `session:save`; the main process writes it to `currentSessionPath` and returns success or an error for status display.

## Persisted and Runtime State Boundary

Durable session JSON contains campaign identity and authored terminal configuration: terminal identifiers and names, hacking difficulty, introductory text, and the folder/command/entry tree. The renderer additionally tracks the selected terminal and node, expanded folders, file path, live terminal, and live hacking status. The server owns client count, navigation path, active hacking board, and broadcast state. These renderer/server values are runtime-only and are not included in the persisted session object.

## Reconstructed Implementation Phases

### Phase 1 — Version-1 session model and sample

- Defined a default session with a name, version, and one usable terminal.
- Established terminal and content-tree fields through the editor implementation and bundled demo.
- Kept runtime live and navigation state outside the JSON representation.

### Phase 2 — Main-process filesystem ownership

- Added UTF-8 JSON read/write helpers in `main.js`.
- Added minimal top-level validation for the `terminals` array.
- Tracked one current session path as the autosave destination.
- Returned filesystem and parsing errors as structured IPC results.

### Phase 3 — Storage location and packaged seeding

- Used the repository `sessions/` directory during development.
- Used a `sessions/` directory next to the executable in packaged builds.
- Created the default directory recursively when possible.
- Copied the bundled demo only when the writable directory was empty and kept seeding failures non-fatal.

### Phase 4 — Native dialogs and isolated IPC

- Added filtered save/open dialogs in the Electron main process.
- Exposed three narrow promise-based methods through the sandboxed preload bridge.
- Distinguished cancellation from errors and changed the active path only after successful I/O and validation.

### Phase 5 — Renderer loading and autosave

- Added new/open actions to the master start screen.
- Installed loaded content and reset transient editing/live state.
- Triggered whole-document autosave after terminal, setting, and content-tree mutations.
- Displayed a timestamp on success and a retained error status on failure.

## Key Technical Decisions

1. **Filesystem access stays in the main process**, preserving Electron sandbox and context-isolation boundaries.
2. **Sessions are portable JSON files**, making campaign data inspectable and movable without a database or application account.
3. **The user selects the actual file**, and that explicit path becomes the only autosave target until another file is successfully created or opened.
4. **The whole session is rewritten after each mutation**, avoiding partial-update or object-mapping machinery for the small current data model.
5. **A version field is persisted from the first format**, although the implementation does not yet use it for validation or migration.
6. **Packaged portable builds store beside the executable**, matching the intended movable application layout.
7. **Sample seeding is conservative**, copying only into a completely empty directory and never replacing existing user data.
8. **Seeding is best-effort**, so an unavailable demo does not prevent the master application from launching.
9. **Transient play state is excluded**, ensuring that reopening a campaign restores authored content rather than stale connections, navigation, or puzzle progress.

## Constitution Check

| Principle | Assessment |
|---|---|
| Preserve runtime boundaries | Pass: native dialogs and filesystem access remain in `main.js`; the renderer uses the narrow preload bridge. |
| Keep shared state server-authoritative | Pass/not directly applicable: live navigation and hacking state remain server-owned and are deliberately excluded from persistence. |
| Protect desktop/public boundaries | Pass with a validation gap: sandboxing, context isolation, and disabled Node integration remain intact, but the save IPC payload is not schema-validated in the main process. |
| Preserve session data compatibility | Partial risk: user paths are explicit and seeding never overwrites data, but version handling and nested schema validation are absent. |
| Match established conventions | Pass: lowercase filenames, CommonJS main/preload modules, camelCase fields, two-space indentation, and JSON camelCase are used. |

The feature has no deliberate constitution exception. The validation, versioning, and write-safety limitations are documented gaps rather than endorsed standards.

## Complexity Assessment

| Measure | Assessment |
|---|---|
| Primary implementation surfaces | 4 code/UI files plus 1 sample data file and README context |
| Runtime boundaries crossed | Master renderer, preload bridge, Electron main process, local filesystem |
| External dependencies | Electron and Node.js built-ins only; no persistence-specific package |
| Data model depth | Moderate: sessions contain terminals and recursive content trees |
| State complexity | Moderate: durable authoring state must remain distinct from renderer and server runtime state |
| Compatibility risk | Moderate-high because files are user-owned and nested fields are only implicitly defined |

The cross-boundary structure is required by Electron security. No separate persistence service or database is justified for the current single-window, local-file workflow.

## Verification Strategy for the Existing Feature

No automated checks can currently be claimed. Proportionate verification for future changes should include:

1. Before adding tests, select and document a Node-capable test framework, test location, and npm command as required by the constitution.
2. Test `defaultSession` for version, filename-derived name, generated terminal ID, default hacking setting, introductory text, and empty root.
3. Test valid JSON, invalid JSON, missing/non-array `terminals`, missing/unsupported versions, and malformed nested terminal/tree data once the intended validation policy is specified.
4. Test create, open, and save IPC results for success, cancellation, missing active path, parse errors, read errors, and write errors.
5. Test that failed create/open operations do not replace the previous active path.
6. Test development and packaged writable-directory resolution with Electron path APIs isolated behind test seams.
7. Test empty, non-empty, missing-demo, and copy-failure seeding behavior without touching user directories.
8. Run `npm start` and manually create, edit, close, and reopen a session while checking the visible file path and save status.
9. Exercise rapid consecutive edits and inspect the final file for ordering or corruption until saves are explicitly serialized.
10. Confirm saved JSON excludes selected node, expanded folders, live terminal, connected clients, navigation path, and active hacking state.
11. Confirm the renderer remains sandboxed with `nodeIntegration: false`, `contextIsolation: true`, and `sandbox: true`.
12. Run `npm run build:dir` when a Windows-capable packaging environment is available, then verify executable-adjacent storage and first-run demo seeding.

## Identified Follow-up Gaps

- Define and implement an executable version-1 schema for the full recursive session tree.
- Decide compatibility rules for missing, older, and future session versions before the format changes.
- Validate renderer-provided save payloads again in the Electron main process.
- Replace direct writes with a crash-safer temporary-file and atomic-replace strategy, with platform behavior specified.
- Serialize or debounce autosaves and make the UI status represent the latest requested revision.
- Decide whether to detect external modifications and multiple-instance write conflicts.
- Surface or log writable-directory and demo-seeding failures without turning optional seeding into a startup blocker.
- Specify whether manual Save, Save As, backups, recovery, recent files, or dirty-state reporting belong in a follow-up feature.

