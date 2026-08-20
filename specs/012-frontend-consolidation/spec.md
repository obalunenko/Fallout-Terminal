# Feature Specification: Frontend Consolidation

**Feature Branch**: `main`  
**Created**: 2026-08-20  
**Status**: Draft

## User Scenarios & Testing

### User Story 1 - One frontend workspace (Priority: P1)

As a maintainer, I can find both the player-facing client and the privileged Overseer application under one frontend area, so that the repository layout clearly communicates ownership and avoids parallel top-level frontend trees.

**Why this priority**: The primary value of the refactor is a single, predictable home for all browser-based application code.

**Independent Test**: Inspect a clean checkout and confirm that both runnable interfaces and all of their owned static, generated, dependency, and build inputs are contained under the frontend area, with no active top-level client application remaining.

**Acceptance Scenarios**:

1. **Given** a maintainer opens the repository root, **When** they look for user-interface code, **Then** both the player client and the Overseer application are discoverable beneath `frontend` in clearly separated role-based areas.
2. **Given** the consolidation is complete, **When** the repository is checked for active client application files, **Then** no parallel top-level `client` application remains.
3. **Given** either frontend is changed, **When** a maintainer follows its local layout, **Then** source, generated inputs, assets, dependency metadata, and build output ownership are unambiguous.

### User Story 2 - Overseer identity replaces master identity (Priority: P1)

As the person operating the privileged desktop interface, I see and maintain it as the Overseer application, so that Fallout terminology is consistent in the active frontend, its host integration, and its build and verification workflow.

**Why this priority**: The rename is an explicit product constraint and must land together with the directory consolidation to avoid creating a newly organized tree with obsolete naming.

**Independent Test**: Launch or inspect the privileged interface and verify that active frontend filenames, presentation copy, build step descriptions, host-window names, and focused test terminology identify it as Overseer rather than master or game master.

**Acceptance Scenarios**:

1. **Given** the privileged interface is opened, **When** its title and start screen are shown, **Then** the role is presented as Overseer.
2. **Given** a maintainer searches active implementation and verification surfaces for the old frontend role name, **When** historical specifications and rollback records are excluded, **Then** no obsolete master frontend identity remains.
3. **Given** stable backend concepts are not part of the frontend rename, **When** the refactor is reviewed, **Then** their behavior and externally meaningful contracts remain unchanged.

### User Story 3 - Existing workflows remain dependable (Priority: P2)

As a developer or release operator, I can use the existing repository-level dependency, generation, test, build, and packaging workflows after the move, so that the structural refactor does not change application behavior or add manual setup steps.

**Why this priority**: Consolidation is only safe if every consumer follows the new locations and clean-checkout workflows remain reproducible.

**Independent Test**: From the repository root, run the standard frontend builds and focused verification gates, then confirm both applications produce their expected artifacts and the native application embeds the correct privileged and player bundles.

**Acceptance Scenarios**:

1. **Given** a clean checkout with supported prerequisites, **When** the standard dependency and generation workflow runs, **Then** all generated client and Overseer inputs are written only to their new owned locations.
2. **Given** both frontend builds succeed, **When** the native application is compiled, **Then** the Overseer bundle is used for the private desktop window and the client bundle is served to players.
3. **Given** existing automated checks are run, **When** they resolve frontend fixtures and assets, **Then** they use the consolidated paths and retain their prior behavioral assertions.

## Edge Cases

- A clean checkout contains only placeholder build-output markers before either frontend is built.
- Generated player protocol files or desktop bindings are regenerated after the move and must not reappear at an obsolete path.
- One frontend build succeeds while the other fails; the repository workflow must identify the failing role without leaving a misleading combined result.
- Cached dependency or output directories from the old layout must not be mistaken for active source or committed artifacts.
- Historical specifications and rollback documentation may retain period-accurate master terminology and paths without being treated as active application code.

## Requirements

### Functional Requirements

- **FR-001**: The repository MUST contain all active player-client and privileged desktop frontend code beneath the top-level `frontend` directory.
- **FR-002**: The player client and privileged application MUST occupy separate, role-based subdirectories with no ambiguous ownership of source, assets, generated code, dependencies, or build output.
- **FR-003**: The privileged frontend MUST be named and presented as `overseer`, including active filenames, directory names, application copy, build descriptions, host integration identifiers, and focused verification terminology.
- **FR-004**: The completed layout MUST NOT retain an active top-level `client` application or active frontend references that depend on its former location.
- **FR-005**: Repository-level dependency installation, protocol generation, desktop-binding generation, development, test, build, packaging, and release workflows MUST resolve the consolidated locations without requiring an additional manual command.
- **FR-006**: Native composition MUST continue to embed the Overseer build for the private desktop window and the client build for the player server as distinct capability boundaries.
- **FR-007**: Existing player and Overseer behavior, security boundaries, generated-contract contents, and user data formats MUST remain unchanged except for the explicit Overseer terminology.
- **FR-008**: Automated checks MUST detect obsolete active paths and must verify that clean-checkout markers, generated outputs, frontend bundles, and embedded resources use the consolidated layout.
- **FR-009**: Historical completed specifications and rollback records MUST remain intact and are exempt from active terminology and path migration.

## Key Entities

- **Frontend workspace**: The single repository area that owns both runnable user interfaces and their build-time inputs.
- **Overseer application**: The trusted, private desktop interface used to author sessions and control player presentation.
- **Client application**: The untrusted, player-facing browser interface served by the application process.
- **Frontend artifact**: A role-specific production bundle embedded or served by the native application without combining the two trust boundaries.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A clean repository inspection finds exactly two active frontend application areas under `frontend` and zero active top-level client application areas.
- **SC-002**: One standard repository preparation workflow installs, generates, and builds both frontend applications successfully with no path-related failures.
- **SC-003**: All existing automated frontend, resource-embedding, and native integration checks pass after their paths and role terminology are updated.
- **SC-004**: Searches of active source, configuration, scripts, and focused tests find zero references to the former top-level client path and zero uses of master as the privileged frontend identity.
- **SC-005**: Functional journeys for session authoring, private control, player connection, terminal navigation, and hacking remain behaviorally unchanged.

## Assumptions

- This is a structural and terminology refactor, not a redesign of either interface.
- `overseer` is the canonical English code and directory identifier; the Russian product meaning is «смотритель».
- Stable backend domain concepts and wire/data contracts are renamed only where they directly identify the frontend or desktop host; otherwise their established identifiers remain unchanged to avoid an unrelated contract migration.
- Completed specifications and rollback documents are historical evidence and are not rewritten.

## Verbatim Constraints

- `frontend`
- `client`
- `overseer`
- `Game master - это Overseer (смотритель)`
