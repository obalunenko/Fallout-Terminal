---
status: migrated
feature: Terminal Content Authoring
source: existing implementation
---

# Tasks: Terminal Content Authoring

**Migration status**: Reverse-engineered from completed implementation on 2026-08-09  
**Specification**: `specs/terminal-content-authoring/spec.md`  
**Plan**: `specs/terminal-content-authoring/plan.md`

All implementation tasks below are marked complete because they describe behavior already present in the codebase. Unbuilt improvements are recorded separately as gaps and are not represented as completed work.

## Phase 1 — Authoring Workspace and Renderer State

- [x] T001 Define the start screen and three-column master workspace with terminal, tree, settings, property, live, and save-status controls in `master/index.html`.
- [x] T002 Style terminal rows, live/editing state, recursive tree rows, forms, toolbars, confirmations, empty states, and the constrained desktop layout in `master/master.css`.
- [x] T003 Define renderer state for the durable session plus editing terminal, selected node, expanded folders, live terminal, and live hacking status in `master/master.js`.
- [x] T004 Install a loaded session, select its first terminal, reset transient authoring/live state, and render the workspace in `master/master.js`.
- [x] T005 Centralize dependent UI refresh through `renderAll` while retaining focused rerenders for tree and property interactions in `master/master.js`.

## Phase 2 — Terminal Collection Management

- [x] T006 Render each terminal with distinct editing and live indicators plus rename and delete actions in `master/master.js`.
- [x] T007 Switch the editing terminal while clearing node selection and resetting expansion to `root` in `master/master.js`.
- [x] T008 Add new terminals with generated IDs, sequential names, disabled hacking, empty introduction text, and an empty `ROOT` folder in `master/master.js`.
- [x] T009 Implement inline terminal rename with focus/select behavior, trimmed non-empty validation, Enter/blur commit, Escape handling, and autosave in `master/master.js`.
- [x] T010 Require confirmation before deleting a terminal and remove only the confirmed terminal from the session in `master/master.js`.
- [x] T011 Clear the live broadcast when its terminal is deleted and select the first remaining terminal when the edited terminal is removed in `master/master.js`.
- [x] T012 Render the no-terminal state and disable terminal-dependent authoring and broadcast controls in `master/master.js`.

## Phase 3 — Recursive Tree Navigation

- [x] T013 Implement recursive ID lookup that returns both a node and its direct parent in `master/master.js`.
- [x] T014 Resolve the insertion target to a selected folder, a selected leaf's parent, or the current terminal root in `master/master.js`.
- [x] T015 Render folder, command, and entry nodes recursively with type labels, selection state, disclosure controls, and empty-folder hints in `master/master.js`.
- [x] T016 Track expanded folder IDs and selected node ID as renderer-local state and update the add-target hint after selection in `master/master.js`.
- [x] T017 Protect the root by rendering descriptive properties without rename or deletion controls in `master/master.js`.

## Phase 4 — Content Creation, Editing, and Deletion

- [x] T018 Create folder nodes with generated IDs, default names, and empty `children` arrays in `master/master.js`.
- [x] T019 Create command nodes with generated IDs, default names, and empty `text` values in `master/master.js`.
- [x] T020 Create entry nodes with generated IDs, default names, and empty `description` values in `master/master.js`.
- [x] T021 Append new nodes to the resolved folder, expand their parent, select them, autosave, and refresh the relevant UI in `master/master.js`.
- [x] T022 Render type-specific property forms and apply non-empty trimmed names plus command or entry body content in `master/master.js`.
- [x] T023 Escape authored values inserted into property-form HTML and use text assignment for tree and terminal labels in `master/master.js`.
- [x] T024 Require confirmation before node deletion, warn when a folder has children, remove the selected node from its parent's children, and clear selection in `master/master.js`.

## Phase 5 — Settings, Persistence, and Live Integration

- [x] T025 Populate hacking difficulty and introduction text controls from the selected terminal and disable them when no terminal exists in `master/master.js`.
- [x] T026 Apply numeric hacking difficulty and introduction text to the selected terminal and request autosave in `master/master.js`.
- [x] T027 Request whole-session persistence through the narrow `window.electronAPI.saveSession` preload method after every accepted terminal, node, or settings mutation in `master/master.js` and `preload.js`.
- [x] T028 Keep ordinary tree mutations in the authoring model until the game master explicitly publishes the edited live terminal in `master/master.js`.
- [x] T029 Publish the current tree and introduction text through the preload boundary without restarting the live terminal or active hacking puzzle in `master/master.js` and `preload.js`.
- [x] T030 Avoid automatically rebroadcasting live terminal renames or hacking-level changes so that an in-progress puzzle is preserved in `master/master.js`.
- [x] T031 Provide multiple example terminals containing folders, commands, entries, nested content, introduction text, and hacking configuration in `sessions/demo.json`.

## Verification Reflected by Existing Code

- [x] T032 Preserve Electron renderer isolation by keeping authoring in browser APIs and exposing filesystem/server operations only through `preload.js` and `main.js`.
- [x] T033 Keep editing selection, expansion state, file path, live identity, and hacking progress outside the persisted session object in `master/master.js`.
- [x] T034 Keep the authored content model compatible with server navigation and player presentation through the shared folder/command/entry shape in `master/master.js`, `server/nav.js`, and `client/client.js`.

## Identified Gaps — Not Implemented Tasks

1. No automated tests cover renderer state, insertion targeting, recursive traversal, mutation, deletion, escaping, or live/publication separation.
2. No complete schema validates terminals, node variants, IDs, nesting depth, collection size, or authored text at load/save and broadcast boundaries.
3. The implementation permits deletion of the final terminal.
4. Terminals and nodes cannot be reordered, moved, duplicated, or independently imported/exported.
5. No undo/redo, deletion recovery, or durable dirty-state model exists.
6. The tree does not implement full keyboard navigation or ARIA tree semantics.
7. Authored strings and recursive trees have no documented size or depth limits.
8. Asynchronous autosave calls are not serialized and can race during rapid mutations.
9. Non-empty folder deletion reports only direct child count, not total descendants.
10. The README does not explain terminal authoring or the folder/command/entry model.

## Suggested Future Task Groups

The gaps above should be specified as new work before implementation rather than marked complete in this migrated feature. Natural follow-up slices are:

- Editor model extraction and automated tests.
- Session/content schema validation and compatibility handling.
- Autosave ordering and recovery.
- Accessible keyboard authoring.
- Tree organization, undo/redo, and documentation improvements.

