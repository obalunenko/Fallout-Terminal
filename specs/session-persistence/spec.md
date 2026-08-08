---
status: migrated
feature: Session Persistence
source: existing implementation
---

# Feature Specification: Session Persistence

**Migration status**: Reverse-engineered from the existing implementation on 2026-08-09  
**Scope**: Creation, opening, validation, autosave, storage location, and first-run sample seeding for filesystem-backed campaign sessions

## Purpose

Session persistence lets a game master keep terminal definitions between application runs. The Electron main process owns native file dialogs and filesystem access, while the sandboxed master renderer works with the active session through a narrow preload API. Sessions are human-readable JSON files and currently declare schema version 1.

## User Scenarios and Acceptance

### User Story 1 — Create a new session file (Priority: P1)

As a game master, I can choose a JSON file for a new session so that I have a persistent campaign workspace with one usable terminal.

**Independent verification**: Start the application, select the new-session action, choose a filename, and inspect both the loaded editor and the created JSON file.

**Acceptance scenarios**:

1. **Given** the start screen is open, **when** the game master chooses a destination in the native save dialog, **then** the application writes a version-1 session and opens it for editing.
2. **Given** the chosen file is named `vault-76.json`, **when** the session is created, **then** its session name is `vault-76`.
3. **Given** a new session is created, **when** its contents are inspected, **then** it contains one terminal named `Терминал 1`, disabled hacking, empty introductory text, and an empty `ROOT` folder.
4. **Given** the save dialog is canceled, **when** control returns to the start screen, **then** no session is loaded and no error is displayed.
5. **Given** the selected file cannot be written, **when** creation is attempted, **then** the start screen remains active and displays the filesystem error.

---

### User Story 2 — Open an existing session (Priority: P1)

As a game master, I can select a saved session and restore its terminals and content so that I can continue preparing or running a campaign.

**Independent verification**: Open the bundled demo session and a malformed or structurally invalid JSON file, then compare the resulting editor and error states.

**Acceptance scenarios**:

1. **Given** a readable JSON file whose top-level `terminals` value is an array, **when** the game master opens it, **then** the application loads it and records that file as the active save target.
2. **Given** the loaded session contains at least one terminal, **when** the editor opens, **then** the first terminal becomes the initial editing target.
3. **Given** the loaded session contains no terminals, **when** the editor opens, **then** no terminal is selected and the empty terminal state is rendered.
4. **Given** the open dialog is canceled, **when** control returns to the start screen, **then** no session is loaded and no error is displayed.
5. **Given** the selected file is invalid JSON or lacks a top-level `terminals` array, **when** opening is attempted, **then** the previous active-file state is not replaced and the error is displayed.

---

### User Story 3 — Save edits automatically (Priority: P1)

As a game master, I have content changes saved to the currently open file so that routine editing does not require a separate manual save action.

**Independent verification**: Create or open a session, perform each supported edit, observe the save status, and reopen the file to confirm the changes persisted.

**Acceptance scenarios**:

1. **Given** a session file is active, **when** the game master adds, renames, configures, or deletes a terminal, **then** the complete current session object is written to that file.
2. **Given** a session file is active, **when** the game master adds, edits, or deletes a folder, command, or entry, **then** the complete current session object is written to that file.
3. **Given** a save succeeds, **when** the IPC response returns, **then** the master interface shows a localized saved timestamp and clears its error style.
4. **Given** a save fails, **when** the IPC response returns, **then** the editor remains open and shows the returned error in its save status.
5. **Given** no session file is active, **when** a save is requested, **then** the main process rejects it with `Нет открытого файла сессии` rather than choosing a destination implicitly.

---

### User Story 4 — Use an appropriate writable session directory (Priority: P2)

As a game master, I see file dialogs start in a predictable writable location so that session files are easy to find in development and portable builds.

**Independent verification**: Run the application in development and in a packaged portable build, invoke the new/open dialogs, and inspect their default directories.

**Acceptance scenarios**:

1. **Given** the application runs from source, **when** a session dialog opens, **then** its default location is the repository's bundled `sessions/` directory.
2. **Given** the application runs as a packaged executable, **when** a session dialog opens, **then** its default location is a `sessions/` directory next to the executable.
3. **Given** the intended writable directory does not exist, **when** it is resolved, **then** the application attempts to create it recursively.
4. **Given** a user selects a file outside the default directory, **when** it is opened or created, **then** subsequent autosaves continue targeting that selected file.

---

### User Story 5 — Receive a first-run sample (Priority: P2)

As a game master using a packaged build for the first time, I receive a demo session when possible so that I can immediately explore the application without losing any existing files.

**Independent verification**: Start a packaged build with an empty writable sessions directory and then with a non-empty directory, comparing the resulting files.

**Acceptance scenarios**:

1. **Given** the packaged writable sessions directory is empty and the bundled demo exists, **when** the main window is created, **then** `demo.json` is copied into the writable directory.
2. **Given** the writable sessions directory contains any entry, **when** the application starts, **then** no bundled file is copied or overwritten.
3. **Given** the demo is missing or seeding fails, **when** the application starts, **then** startup continues without a seeded session.

## Functional Requirements

- **FR-001**: A persisted session MUST be JSON and the application-created representation MUST contain `version`, `name`, and `terminals`.
- **FR-002**: New sessions MUST declare `version: 1`.
- **FR-003**: A new session's name MUST be derived from the chosen filename without its final extension.
- **FR-004**: A new session MUST contain one terminal with a generated ID, the name `Терминал 1`, `hackLevel: 0`, empty `introText`, and an empty folder root named `ROOT`.
- **FR-005**: Session creation MUST use a native save dialog filtered to JSON files and MUST write the initial file before making it active.
- **FR-006**: Session opening MUST use a native single-file open dialog filtered to JSON files.
- **FR-007**: Opening a session MUST parse the selected file as UTF-8 JSON and MUST reject a top-level value without a `terminals` array.
- **FR-008**: A failed create or open operation MUST NOT replace the current active save path.
- **FR-009**: Canceling a create or open dialog MUST return a non-success result without treating cancellation as an error.
- **FR-010**: Successful create and open operations MUST return the selected path and session object to the master renderer and set that path as the subsequent save target.
- **FR-011**: Loading a session in the renderer MUST reset runtime-only editing, selection, expansion, live-terminal, and live-hacking state rather than persisting those values.
- **FR-012**: Saving MUST serialize the complete renderer-provided session object as indented JSON to the current active file.
- **FR-013**: Saving without an active file MUST fail explicitly and MUST NOT open a dialog or infer a path.
- **FR-014**: Filesystem and JSON errors from create, open, and save operations MUST cross the IPC boundary as non-success results with an error message.
- **FR-015**: The master renderer MUST display the selected file path after a successful create or open operation.
- **FR-016**: Successful autosave MUST display a localized completion time; failed autosave MUST display an error state without closing the editor.
- **FR-017**: Development mode MUST default session dialogs to `<application>/sessions`; packaged mode MUST default them to a `sessions` directory adjacent to the executable.
- **FR-018**: The application MUST attempt to create its default writable sessions directory recursively before using it.
- **FR-019**: On startup, the bundled demo MUST be copied to an empty writable sessions directory when available and MUST NOT overwrite or add to a non-empty directory.
- **FR-020**: Session filesystem operations MUST remain in the Electron main process and MUST be exposed to the sandboxed renderer only through the narrow preload methods `newSession`, `openSession`, and `saveSession`.
- **FR-021**: Runtime live-terminal, navigation, connection, and hacking progress MUST remain outside session JSON; only durable terminal configuration and content belong in the persisted session.

## Persisted Data Contract

The implementation creates version-1 objects with this observed shape:

```text
session
├── version: number (created as 1)
├── name: string
└── terminals: array
    └── terminal
        ├── id: string
        ├── name: string
        ├── hackLevel: number
        ├── introText: string
        └── root: folder node
            ├── id: string
            ├── type: "folder"
            ├── name: string
            └── children: node[]
```

Observed child node variants are folders with `children`, commands with `text`, and entries with `description`. This is an observational contract, not an executable schema: the loader currently validates only the top-level `terminals` array.

## IPC Contract

| Direction | Channel/API | Result or payload |
|---|---|---|
| Master renderer → main | `session:new` / `newSession()` | No payload; returns `{ok:false}` on cancellation, `{ok:false,error}` on failure, or `{ok:true,filePath,session}`. |
| Master renderer → main | `session:open` / `openSession()` | No payload; returns the same result variants as creation. |
| Master renderer → main | `session:save` / `saveSession(session)` | Carries the complete session object; returns `{ok:true}` or `{ok:false,error}`. |

## Edge Cases Observed

- Any JSON object with a `terminals` array passes load validation, even if its version or nested fields are missing or invalid.
- The `version` field is written but not checked, interpreted, upgraded, or rejected when reading.
- Creating a session at an existing path relies on native-dialog and filesystem behavior; no application-level overwrite confirmation is added.
- The active save target is process-global and changes only after a successful create or open operation.
- Autosave calls are started without awaiting them at most mutation sites, so rapid edits may produce overlapping requests.
- Writes are synchronous and replace the destination directly rather than using a temporary file and atomic rename.
- A directory containing any entry suppresses demo seeding, even when it contains no session JSON files.
- Failures while creating the writable directory or seeding the demo are intentionally ignored.
- A malformed loaded session may fail later in renderer code after passing the minimal loader check.

## Success Criteria

- **SC-001**: Creating a session produces parseable, indented version-1 JSON containing one immediately editable default terminal.
- **SC-002**: Opening the bundled demo restores its two terminals and selects the first terminal for editing.
- **SC-003**: Each supported terminal or content-tree mutation can be observed in the same JSON file after autosave completes and the file is reopened.
- **SC-004**: Canceling either native file dialog leaves the start screen and active save path unchanged without displaying an error.
- **SC-005**: Invalid JSON, a missing `terminals` array, and filesystem write failures return visible errors without switching the application to the failed file.
- **SC-006**: An empty packaged writable directory receives one demo copy, while any non-empty directory retains all of its contents unchanged.
- **SC-007**: Persisted JSON excludes runtime-only live broadcast, navigation path, client count, and active hacking-board state.
- **SC-008**: The master renderer retains Electron sandboxing and performs no direct Node.js or filesystem access during all session operations.

## Assumptions

- One master window and one active session file exist per application process.
- Session files are user-owned and may be stored outside the default `sessions/` directory.
- Human-readable formatting is intentional and files may be inspected or edited outside the application.
- Existing version-1 sample data represents the intended durable field names, but strict nested validation has not yet been specified or implemented.
- Autosave is the only implemented save workflow; there is no separate manual save or save-as action.
- Silent demo-seeding failure is intended to keep application startup non-fatal.

## Known Gaps

- No automated tests cover session defaults, validation, IPC results, filesystem failures, autosave ordering, or packaged-directory behavior.
- The loader has no complete schema validation and accepts unsupported or missing versions and malformed terminal trees.
- The main process trusts the entire session object received from the renderer without validation or size limits.
- No version migration, compatibility fallback, or future-version rejection policy exists despite the persisted `version` field.
- Direct synchronous replacement is not crash-safe or atomic and no backup or recovery copy is retained.
- Autosaves are neither serialized nor debounced, and most mutation handlers do not await completion.
- External file changes and multiple application instances are not detected, so last-writer-wins data loss is possible.
- Writable-directory creation and demo-seeding errors are suppressed and not surfaced diagnostically.
- There is no explicit Save As, manual save, dirty-state indicator, recent-file list, or recovery workflow.

