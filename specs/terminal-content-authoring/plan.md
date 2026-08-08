---
status: migrated
feature: Terminal Content Authoring
source: existing implementation
---

# Implementation Plan: Terminal Content Authoring

**Migration status**: Reverse-engineered from the existing implementation on 2026-08-09  
**Specification**: `specs/terminal-content-authoring/spec.md`

## Summary

The implemented feature is a dependency-free authoring interface inside the sandboxed Electron master renderer. Static HTML defines a three-column workspace, CSS presents terminal and node state, and `master/master.js` owns the in-memory session model plus transient editor state. Recursive helpers locate and render content nodes. Mutations update the active session object, request whole-session autosave through the preload bridge, and reach live players only through explicit broadcast controls or the existing settings integration.

## Technical Context

| Area | Implemented choice |
|---|---|
| Language | Browser JavaScript, HTML, and CSS |
| Desktop runtime | Electron 28 |
| Renderer security | `nodeIntegration: false`, `contextIsolation: true`, `sandbox: true` |
| UI architecture | Static DOM plus imperative rendering and event listeners; no frontend framework |
| State | One renderer-local `state` object containing durable session data and transient editor/live references |
| Persistence integration | Whole-session IPC request through `window.electronAPI.saveSession` |
| Broadcast integration | Narrow preload calls for live update and broadcast clearing |
| Content model | Recursive tagged objects: folder, command, and entry |
| Runtime dependencies | No authoring-specific package; browser DOM and the existing Electron preload bridge |
| Tests | No test framework, test directory, lint command, CI workflow, or coverage target is configured |
| Available checks | `npm start`; `npm run build:dir` for applicable Windows packaging checks |

## Detected Project Structure

### Primary implementation

- `master/index.html` — start screen and the terminal list, settings, tree, property, status, and authoring controls.
- `master/master.css` — three-column layout, tree states, forms, buttons, status treatments, and responsive presentation.
- `master/master.js` — authoring state, rendering, recursive traversal, mutations, autosave requests, and live-publication integration.
- `sessions/demo.json` — an executable example of multiple terminals and all supported content-node variants.

### Supporting boundaries

- `preload.js` — exposes narrow session-save and broadcast methods to the sandboxed renderer.
- `main.js` — owns session filesystem persistence and forwards broadcast operations to the server.
- `server/server.js` — consumes explicitly published content but does not own authoring state.

### Related migrated artifacts

- `specs/session-persistence/` owns session creation, loading, validation, storage paths, and autosave durability.
- `specs/live-broadcast-shared-navigation/` owns live snapshots, explicit updates, and navigation revalidation.
- `specs/hacking-game/` owns hacking difficulty semantics and puzzle lifecycle.
- `specs/player-terminal-presentation/` owns player rendering of authored content.

## Feature Boundaries

### Included

- Terminal collection creation, selection, rename, and deletion.
- Recursive tree visualization and expansion.
- Folder, command, and entry creation.
- Node property editing and subtree deletion.
- Terminal introduction and hacking-setting form integration.
- Editor-local selection and expansion state.
- Autosave request initiation after accepted mutations.
- Explicit publication behavior from the editor.

### Excluded

- Native file dialogs, filesystem paths, JSON parsing, and disk-write guarantees.
- Hacking board generation, guesses, and success/failure behavior.
- WebSocket protocol implementation and server-authoritative navigation.
- Player-side rendering, keyboard navigation, reveal animation, and audio.
- Public ngrok tunnel startup and authentication.

## Reconstructed Architecture

1. A successfully opened or created session is installed in renderer state by `loadSession`.
2. The first terminal becomes the editing target; node selection is cleared and only the root starts expanded.
3. `renderAll` derives the terminal list, header, settings, toolbar, recursive tree, property form, add-target hint, and hacking status from current state.
4. Terminal actions mutate `state.session.terminals`; tree actions locate a node recursively and mutate its parent or type-specific fields.
5. Each accepted durable mutation calls `autosave`, which submits the complete session object through the preload bridge.
6. Renderer-only selection, expansion, live-terminal identity, file path, and hacking progress stay outside serialized session data.
7. Ordinary content mutations do not update the server's live snapshot. The game master explicitly publishes the current tree, except that applying settings to the live terminal immediately refreshes tree and introduction text without restarting hacking.
8. Renderer-authored strings are placed with `textContent` or escaped before use in generated form markup.

## Implementation Phases

### Phase 1 — Master authoring workspace and state

- Defined terminal-list, content-tree, settings, toolbar, properties, and status regions in the master HTML.
- Styled the fixed authoring workspace and visual editing/live/selected states.
- Established durable session references and transient editor state in the master renderer.
- Installed a loaded session, selected its first terminal, and rendered all dependent regions.

### Phase 2 — Terminal collection management

- Rendered the terminal collection with separate editing and live indicators.
- Added new terminals with the implemented defaults and selected them immediately.
- Added inline terminal renaming with trimmed non-empty input and autosave.
- Added confirmed deletion, fallback selection, and live-broadcast clearing for a deleted live terminal.

### Phase 3 — Recursive content-tree authoring

- Added recursive node lookup with parent recovery.
- Derived add targets from folder, leaf, or missing selection.
- Rendered expandable recursive folders plus command and entry leaves.
- Added type-specific default nodes, unique generated IDs, parent expansion, and selection.
- Added property forms for names, command text, and entry descriptions.
- Added confirmed leaf and subtree deletion while protecting the root.

### Phase 4 — Settings, persistence, and publication integration

- Populated and applied hacking-level and introduction controls for the selected terminal.
- Requested whole-session autosave after each accepted authoring mutation.
- Kept ordinary authoring changes separate from the current live server snapshot.
- Added explicit tree/introduction publication for the edited live terminal.
- Preserved active hacking state when applying settings or renaming a live terminal.
- Escaped renderer-authored values at HTML insertion boundaries.

## Technical Decisions Reconstructed

1. **Keep authoring in the master renderer** because it is local UI state and does not require server authority before publication.
2. **Store content as a recursive tagged-object tree** to map folders and leaves directly to player navigation and JSON persistence.
3. **Use one `state` object for durable and transient renderer data** while passing only `state.session` to persistence.
4. **Use terminal-scoped root IDs** so every terminal can use the stable navigation root value `root`.
5. **Resolve leaf insertion to its parent folder** so an add action always creates a sibling or child in a valid folder container.
6. **Protect the root structurally in the property UI** by omitting edit and delete controls rather than adding a special deletion handler.
7. **Autosave the whole session after each mutation** to keep the persistence boundary simple at the cost of serialization and write-order risks.
8. **Require explicit live publication for ordinary content edits** so preparation does not unexpectedly change the player experience.
9. **Apply introduction updates without a full rebroadcast** so live text can refresh while server-owned hacking progress survives.
10. **Keep terminal rename out of live refresh** because a full `setLiveTerminal` call would regenerate an active hacking board.
11. **Render authored values as text or escape them** to prevent terminal content from becoming executable master-page markup.
12. **Avoid authoring-specific dependencies** because the existing DOM operations and preload contract cover the implemented scale.

## Data and Contract Considerations

The content model is persisted in version-1 session JSON, but this feature does not introduce a separate database or migration. A terminal contains its identity, presentation configuration, and one root folder. Folder nodes recursively contain children; command and entry nodes are leaves with distinct body fields.

The master renderer assumes valid terminal and node shapes after load. It does not enforce global ID uniqueness, maximum depth, maximum child count, string-size limits, or exhaustive type validation. Those concerns cross into the Session Persistence contract and are recorded as gaps rather than inferred guarantees.

The preload bridge remains intentionally narrow. Authoring invokes persistence and publication methods but never imports Node modules, touches the filesystem, or calls server modules directly.

## Constitution Check

| Principle | Assessment |
|---|---|
| Preserve runtime boundaries | Pass: authoring stays in `master/`; persistence and server operations cross only the preload/main boundary. |
| Keep shared state server-authoritative | Pass: authoring changes are local until explicit publication; player navigation and hacking remain server-owned. |
| Protect desktop/public boundaries | Pass with existing validation gaps: Electron isolation and the restrictive master CSP remain enabled, but renderer-provided session and broadcast payloads are not fully validated in main/server code. |
| Preserve session data compatibility | Pass for observed behavior: authoring uses the established version-1 shape and excludes transient editor/live state. Complete schema and migration validation are absent. |
| Match established code conventions | Pass: lowercase files, camelCase JavaScript, two-space indentation, semicolons, browser globals, and kebab-case CSS are used. |
| Testing and quality gates | Gap: no automated framework or canonical checks exist; the implementation requires manual Electron verification. |

No intentional constitution violation was detected. The validation and testing limitations are existing gaps, not approved exceptions.

## Complexity Assessment

| Measure | Assessment |
|---|---|
| Primary source | 3 files, 925 lines (`master/master.js`, `master/index.html`, `master/master.css`) |
| Example model | 1 JSON file, 76 lines |
| Supporting boundary | 2 files, 253 lines (`preload.js`, `main.js`), mostly shared with other features |
| Runtime boundaries crossed | Master renderer → preload → Electron main; optionally Electron main → local server for publication |
| External dependencies | Electron bridge only; no authoring-specific runtime dependency |
| Data complexity | Moderate: recursively nested heterogeneous nodes plus terminal-scoped transient selection state |
| Behavioral complexity | Moderate: destructive subtree operations, live/non-live separation, and whole-session autosave integration |
| Automated coverage | None detected |

The single-renderer implementation matches the repository's modular-monolith structure. Introducing a frontend framework, separate authoring service, or database would add complexity without evidence that the current feature requires it.

## Verification Strategy

Because no automated test framework is configured, the implemented feature can currently be verified only through code inspection and manual runtime checks:

1. Run `npm start` and create or open a session.
2. Add several terminals; switch between them and confirm selection and expansion reset behavior.
3. Rename terminals with valid, whitespace-only, Enter, blur, and Escape interactions.
4. Create folders, commands, and entries at root, within folders, and while leaf nodes are selected.
5. Build a tree at least three folders deep and verify expand/collapse and add-target behavior.
6. Edit names and multiline command/entry bodies, including HTML-significant characters.
7. Cancel and confirm deletion of leaf nodes, nested non-empty folders, non-live terminals, and the live terminal.
8. Delete every terminal and confirm the empty state remains usable and allows creation of a new terminal.
9. Reopen the session and compare every supported authoring mutation with saved JSON.
10. Broadcast a terminal, edit ordinary content, and confirm players remain unchanged until Publish.
11. Apply introduction text to a live terminal and confirm the current puzzle is not regenerated.
12. Confirm authoring never exposes Node globals and the master CSP remains effective.
13. Run `npm run build:dir` only when a Windows-capable packaging environment is available and changes affect packaged master assets.

Before adding automated tests, a future plan must name the framework, test directory, npm command, and intended boundaries as required by the constitution. Pure recursive model operations should first be extracted or exposed through a testable module rather than tested only through DOM integration.

## Identified Follow-up Work

- Establish focused tests for recursive lookup, insertion targeting, mutations, subtree deletion, and escaping.
- Validate the complete terminal/content schema and identifiers at the Electron boundary.
- Serialize or version autosave requests to prevent stale writes winning during rapid edits.
- Add undo/redo or a recoverable deletion workflow.
- Add ordering and move/duplicate operations if campaign authoring needs them.
- Implement an accessible keyboard tree pattern with appropriate roles and state attributes.
- Define reasonable depth, node-count, and authored-text limits.
- Document terminal authoring and node semantics in the README.

