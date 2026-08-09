---
status: migrated
feature: Terminal Content Authoring
source: existing implementation
---

# Feature Specification: Terminal Content Authoring

**Migration status**: Reverse-engineered from the existing implementation on 2026-08-09  
**Scope**: Game-master terminal collection management, recursive content-tree editing, terminal introduction text, editor-local selection state, and persistence/publication integration

## Purpose

Terminal content authoring gives a game master a desktop workspace for preparing the terminals players will later explore. A session may contain multiple terminals. Each terminal has a name, optional introduction text, hacking configuration, and a recursive tree made from folders, commands, and entries. The editor keeps preparation state separate from the currently broadcast terminal and saves supported mutations through the sandboxed Wails compatibility facade.

Session file selection and filesystem behavior are specified by the migrated Session Persistence feature. Hacking rules and live transport are specified separately. This feature documents the authoring behavior that produces and changes terminal content.

## User Scenarios and Acceptance

### User Story 1 — Manage the terminal collection (Priority: P1)

As a game master, I can create, select, rename, and delete terminals so that one campaign session can contain multiple in-world computers.

**Independent verification**: Open a session, add two terminals, switch between them, rename one, delete both live and non-live terminals, and reopen the saved session.

**Acceptance scenarios**:

1. **Given** a session is open, **when** the game master adds a terminal, **then** a terminal with a generated ID, sequential display name, disabled hacking, empty introduction, and empty `ROOT` folder is appended and selected for editing.
2. **Given** multiple terminals exist, **when** the game master selects a different terminal row, **then** the editor displays that terminal and clears the previous node selection and expansion state except for `ROOT`.
3. **Given** a terminal is being renamed, **when** a non-empty trimmed name is committed by Enter or blur, **then** the stored name changes and the session is autosaved.
4. **Given** a rename input contains only whitespace, **when** it is committed, **then** the existing terminal name is retained.
5. **Given** the game master confirms terminal deletion, **when** the terminal is removed, **then** it disappears from the session and the first remaining terminal becomes selected if the deleted terminal was being edited.
6. **Given** the terminal being deleted is live, **when** deletion is confirmed, **then** the live broadcast is cleared before the editor finishes updating its local state.
7. **Given** terminal deletion is canceled, **when** control returns to the editor, **then** the session and broadcast state remain unchanged.
8. **Given** no terminals remain, **when** the collection renders, **then** the editor shows an empty-state message and disables content-authoring controls that require a terminal.

---

### User Story 2 — Navigate a terminal's content tree (Priority: P1)

As a game master, I can inspect and select nodes in a recursive tree so that I understand where content lives and where new content will be added.

**Independent verification**: Use a terminal containing nested folders, commands, and entries; expand and collapse folders; select each node type; and observe the property panel and add-target hint.

**Acceptance scenarios**:

1. **Given** an editable terminal is selected, **when** its tree renders, **then** the immutable root is labeled `ROOT` and descendants show distinct folder, command, or entry type labels.
2. **Given** a non-empty folder is collapsed, **when** its disclosure control is activated, **then** its direct children appear recursively and the disclosure indicator changes.
3. **Given** an empty folder is expanded, **when** its contents render, **then** the tree shows an explicit empty hint.
4. **Given** any node is selected, **when** the editor rerenders the tree and property panel, **then** that node is highlighted and its type-appropriate properties are shown.
5. **Given** a folder is selected, **when** the add-target hint renders, **then** new content targets that folder.
6. **Given** a command or entry is selected, **when** the add-target hint renders, **then** new content targets its parent folder.
7. **Given** no node or an invalid node reference is selected, **when** an add target is resolved, **then** the terminal root is used.

---

### User Story 3 — Create and edit terminal content (Priority: P1)

As a game master, I can build nested folders and author commands and readable entries so that players receive a navigable in-world information structure.

**Independent verification**: Add every node type at the root and inside a nested folder, edit their names and text, delete leaf and non-empty folder nodes, and inspect the saved JSON.

**Acceptance scenarios**:

1. **Given** a valid add target, **when** a folder is added, **then** it receives a generated ID, default folder name, empty `children` array, becomes selected, and its parent is expanded.
2. **Given** a valid add target, **when** a command is added, **then** it receives a generated ID, default command name, and empty `text` value.
3. **Given** a valid add target, **when** an entry is added, **then** it receives a generated ID, default entry name, and empty `description` value.
4. **Given** a non-root node is selected, **when** its non-empty trimmed name and type-specific content are applied, **then** the in-memory node changes and the complete session is autosaved.
5. **Given** the proposed node name is empty, **when** Apply is selected, **then** the mutation is rejected and focus returns to the name field.
6. **Given** a leaf node is selected, **when** deletion is confirmed, **then** the node is removed from its parent's children and selection is cleared.
7. **Given** a folder with children is selected, **when** deletion is requested, **then** confirmation warns that its content will also be removed and confirmation deletes the entire subtree.
8. **Given** the `ROOT` node is selected, **when** properties render, **then** it is described as the terminal root and no rename or delete controls are offered.
9. **Given** authored content contains HTML-significant characters, **when** the property form and tree render, **then** values are escaped or assigned as text rather than interpreted as markup.

---

### User Story 4 — Configure and safely publish authored content (Priority: P2)

As a game master, I can set terminal-wide introduction text and decide when live players receive edits so that preparation changes do not unexpectedly disrupt play.

**Independent verification**: Edit a live and non-live terminal, apply introduction text, change hacking difficulty, inspect autosave output, and compare the player view before and after explicit publication.

**Acceptance scenarios**:

1. **Given** a terminal is selected, **when** its settings render, **then** the stored hacking level and introduction text populate the controls.
2. **Given** settings are applied, **when** the editor accepts them, **then** hacking level is normalized to a number, introduction text is stored verbatim, and the session is autosaved.
3. **Given** the edited terminal is not live, **when** settings or content change, **then** no player update is emitted.
4. **Given** the edited terminal is live, **when** introduction settings are applied, **then** the current tree and introduction text are sent as a live update without restarting the hacking puzzle.
5. **Given** the edited terminal is live, **when** ordinary tree content changes, **then** changes remain in the authoring model until the game master explicitly selects the publish action.
6. **Given** the edited terminal is live, **when** Publish is selected, **then** the current tree and introduction text are sent through the narrow bound desktop API and the control briefly reports completion.
7. **Given** a terminal is renamed while live, **when** the rename is saved, **then** the active broadcast is not restarted and the new name reaches players only on the next full broadcast.
8. **Given** a hacking level changes during a live puzzle, **when** settings are applied, **then** the active puzzle is not regenerated and the new level takes effect on the next full broadcast.

## Functional Requirements

### Terminal collection

- **FR-001**: The editor MUST keep the selected editing terminal distinct from the live terminal.
- **FR-002**: Adding a terminal MUST append and select a terminal containing a generated ID, default name, `hackLevel: 0`, empty `introText`, and an empty folder root with ID `root`.
- **FR-003**: Terminal selection MUST clear the selected node and reset expansion state to the root.
- **FR-004**: Terminal rename MUST trim input, reject an empty result, and autosave an accepted name.
- **FR-005**: Terminal deletion MUST require confirmation and MUST clear the broadcast when the deleted terminal is live.
- **FR-006**: When the edited terminal is deleted, the editor MUST select the first remaining terminal or enter the no-terminal state.

### Tree navigation and node placement

- **FR-007**: The editor MUST recursively render folder, command, and entry nodes with type-identifying labels.
- **FR-008**: Folder expansion and node selection MUST remain renderer-local state and MUST NOT be persisted in session JSON.
- **FR-009**: Selecting a folder MUST make that folder the add target; selecting a leaf MUST make its parent the add target; missing selection MUST fall back to the root.
- **FR-010**: A newly added node MUST receive a generated ID and the fields required by its type.
- **FR-011**: Adding a node MUST select it, expand its parent, mutate only the current terminal tree, and autosave the session.

### Property editing and deletion

- **FR-012**: The root node MUST NOT expose rename or delete operations.
- **FR-013**: Applying a node edit MUST require a non-empty trimmed name.
- **FR-014**: Command edits MUST store their body in `text`; entry edits MUST store their body in `description`; folder nodes MUST retain `children`.
- **FR-015**: Deleting a node MUST require confirmation and remove it from its direct parent's `children` array.
- **FR-016**: A deletion prompt for a non-empty folder MUST disclose that child content will also be removed.
- **FR-017**: Authored values rendered through HTML templates MUST be escaped, while labels rendered through DOM properties SHOULD use text assignment.

### Terminal settings and integration

- **FR-018**: Applying terminal settings MUST persist numeric hacking difficulty and the authored introduction text.
- **FR-019**: Every accepted terminal, node, or settings mutation MUST request autosave through `window.electronAPI.saveSession`.
- **FR-020**: Ordinary content edits MUST NOT implicitly publish the current tree to players.
- **FR-021**: Explicit live publication MUST send the current root tree and introduction text without changing the selected editing terminal.
- **FR-022**: Applying introduction text to the live terminal MAY update players immediately, but changing hacking difficulty or terminal name MUST NOT restart an in-progress puzzle.
- **FR-023**: The master frontend MUST access filesystem and broadcast capabilities only through the bound desktop API and MUST NOT gain direct Node.js access.

## Data Model

### Terminal

```text
Terminal
├── id: string
├── name: string
├── hackLevel: number
├── introText: string
└── root: FolderNode
```

### Content nodes

```text
FolderNode  = { id, type: "folder", name, children: ContentNode[] }
CommandNode = { id, type: "command", name, text }
EntryNode   = { id, type: "entry", name, description }
ContentNode = FolderNode | CommandNode | EntryNode
```

The root uses the special ID `root` within each terminal. Generated terminal and child-node IDs combine a prefix, current timestamp, and renderer-local counter. The implementation assumes node IDs are unique within a terminal but does not validate that assumption when loading a session.

## Runtime Boundaries and Dependencies

| Boundary | Contract |
|---|---|
| Master frontend → bound Go methods | `saveSession(session)`, `updateLiveTerminal(payload)`, `clearLiveTerminal()` through the compatibility facade |
| Master frontend local state | Editing terminal, selected node, expanded folder IDs, live terminal ID, last known hacking state |
| Durable session state | Terminal IDs, names, hacking levels, introduction text, and recursive content trees |
| Server live state | Published snapshot and player navigation; not directly mutated by ordinary authoring operations |

There is no database or schema migration for this feature. The authored model is stored as versioned session JSON through the separate persistence feature.

## Success Criteria

- **SC-001**: A game master can create, rename, select, and delete terminals and observe every accepted mutation in the active session JSON.
- **SC-002**: A game master can construct at least three levels of nested folders containing both command and entry leaves, reopen the session, and recover the same hierarchy and content.
- **SC-003**: Adding content while a folder, command, entry, or no node is selected places the new node in the implemented target location without corrupting siblings.
- **SC-004**: Root deletion is unavailable, while confirmed deletion of any non-root leaf or subtree removes only that selected branch.
- **SC-005**: Text containing `<`, `>`, `&`, or quotes renders as authored text in the master editor and does not create executable markup.
- **SC-006**: Editing a live terminal's ordinary content does not change connected player views until explicit publication, while publishing sends the current tree and introduction text.
- **SC-007**: All authoring actions operate inside the Wails webview with no direct Node.js, filesystem, or process access.

## Assumptions

- A session has already been created or opened before the main authoring interface becomes available.
- Each terminal owns an independent tree even though every root currently uses the ID `root`.
- Folder, command, and entry are the complete implemented set of content-node types.
- Terminal names and authored text are intentionally free-form apart from the non-empty name checks.
- Autosave success and filesystem durability are owned by the Session Persistence feature.
- Hacking difficulty behavior and full broadcast lifecycle are owned by their respective migrated features; this feature documents only how authoring controls interact with them.

## Identified Gaps

1. No automated tests, test framework, lint command, CI workflow, or coverage target cover the editor state and recursive mutations.
2. Loaded terminals and nodes have no complete schema, type, uniqueness, depth, or size validation.
3. The final terminal can be deleted, leaving a valid UI empty state but no authorable terminal until another is added.
4. Terminals and nodes cannot be reordered, moved, duplicated, imported, or exported independently.
5. There is no undo/redo history, dirty-state indicator, or recovery path for an unintended confirmed deletion.
6. Tree expansion and selection rely primarily on pointer interaction and do not implement a complete keyboard tree pattern or ARIA semantics.
7. Names, introduction text, command text, and entry descriptions have no documented length limits.
8. Autosave requests are asynchronous and are not serialized by the renderer, so rapid mutations may complete out of order.
9. The delete prompt reports only a folder's direct child count rather than the complete descendant count.
10. The README does not document the editor workflow or content-node model.
