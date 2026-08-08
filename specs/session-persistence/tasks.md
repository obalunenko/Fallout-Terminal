---
status: migrated
feature: Session Persistence
source: existing implementation
---

# Tasks: Session Persistence

All implementation tasks below are marked complete because they reconstruct work already present in the repository. The final section records gaps; it does not claim those follow-ups were implemented.

## Phase 1 — Session Model and Sample Data

- [x] T001 Define version-1 session defaults with a filename-derived name and terminal collection in `main.js`.
- [x] T002 Create one default terminal with a generated ID, disabled hacking, empty introductory text, and an empty `ROOT` folder in `main.js`.
- [x] T003 Represent recursive folder, command, and entry content in the master editing model and `sessions/demo.json`.
- [x] T004 Keep selected editor state, live broadcast state, navigation, connections, and active hacking state outside persisted session JSON.

## Phase 2 — Main-Process Persistence

- [x] T005 Read selected session files as UTF-8 JSON and reject values without a top-level `terminals` array in `main.js`.
- [x] T006 Serialize complete sessions as human-readable, two-space-indented UTF-8 JSON in `main.js`.
- [x] T007 Track the successfully created or opened file as the process's current autosave target in `main.js`.
- [x] T008 Reject save requests when no current session path exists and return filesystem or parsing errors as structured results in `main.js`.

## Phase 3 — Storage Locations and Demo Seeding

- [x] T009 Resolve the bundled sample directory relative to application files in `main.js`.
- [x] T010 Resolve writable sessions to the repository directory in development and beside the executable in packaged builds in `main.js`.
- [x] T011 Attempt recursive creation of the writable sessions directory without making failure fatal in `main.js`.
- [x] T012 Seed `demo.json` only when the writable directory is empty and never overwrite existing user content in `main.js`.
- [x] T013 Invoke writable-directory setup and best-effort demo seeding before the main window is created in `main.js`.

## Phase 4 — Native Dialogs and IPC Boundary

- [x] T014 Add a JSON-filtered native save dialog and create the selected session before activating its path in `main.js`.
- [x] T015 Add a JSON-filtered, single-file native open dialog and validate the selected session before activating its path in `main.js`.
- [x] T016 Distinguish user cancellation from parse, validation, and filesystem errors in create/open IPC results in `main.js`.
- [x] T017 Handle whole-session autosave requests against the current path in `main.js`.
- [x] T018 Expose only promise-based `newSession`, `openSession`, and `saveSession` methods through `preload.js` while preserving renderer sandboxing.

## Phase 5 — Master Renderer Integration

- [x] T019 Wire the start-screen new/open actions to the preload API and display returned errors in `master/master.js`.
- [x] T020 Install the returned session and file path, select the first terminal when present, and reset runtime-only editor/live state in `master/master.js`.
- [x] T021 Display the active session file path and transition from the start screen to the editor after a successful create/open operation in `master/master.js`.
- [x] T022 Submit the complete in-memory session after terminal creation, deletion, rename, configuration, and content-tree mutations in `master/master.js`.
- [x] T023 Show a localized save timestamp on success and a retained error status on failure in `master/master.js` and its existing UI.

## Phase 6 — Existing Compatibility and Security Evidence

- [x] T024 Declare `version: 1` in newly created and bundled sample sessions in `main.js` and `sessions/demo.json`.
- [x] T025 Avoid replacing the active path until creation has written successfully or opening has parsed and minimally validated successfully in `main.js`.
- [x] T026 Preserve arbitrary user-selected file locations as the autosave target rather than silently redirecting writes to the default directory.
- [x] T027 Preserve existing files by skipping all demo seeding whenever the writable directory is non-empty in `main.js`.
- [x] T028 Keep native dialogs and Node.js filesystem access out of the master renderer through the Electron preload/main boundary.

## Gaps Identified During Migration

These are observations for follow-up specification and planning, not completed tasks:

1. No automated test framework or tests cover defaults, validation, IPC responses, filesystem failures, autosave ordering, packaging paths, or demo seeding.
2. Loaded files are checked only for a top-level `terminals` array; version, terminal fields, recursive nodes, IDs, value types, and bounds are not validated.
3. The version field is persisted but has no migration, backward-compatibility, or future-version rejection behavior.
4. Complete save payloads from the renderer are trusted without main-process schema validation or size limits.
5. Writes synchronously replace the destination without a temporary file, atomic rename, backup, or recovery strategy.
6. Autosave requests are not serialized or debounced, and most mutation handlers do not await their completion.
7. External changes and multi-instance conflicts are not detected; concurrent writers follow last-writer-wins behavior.
8. Writable-directory creation and demo-seeding failures are silently suppressed, limiting diagnostics.
9. No Save As, manual save, dirty-state indicator, recent-file list, or recovery workflow exists.

