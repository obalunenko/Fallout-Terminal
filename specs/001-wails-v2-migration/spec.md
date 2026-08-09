# Feature Specification: Wails v2 Runtime Migration

**Feature Branch**: `001-wails-v2-migration` (based on and targeted to `develop`)

**Created**: 2026-08-09

**Status**: Draft

**Input**: User description: "Prepare the migration of Fallout Terminal from its current Electron/Node desktop runtime to Go and Wails v2 while preserving the behavior documented by the existing Spec Kit artifacts."

**Bugfix**: 2026-08-09 — BUG-001 clarified that inactive player presentation states must be excluded from layout and remain unreachable through overflow scrolling.

**Scope decision**: 2026-08-09 — Current macOS acceptance targets personal use on the owner's Apple Silicon Mac. Developer ID signing, notarization, and DMG distribution remain a documented optional public-release profile and do not block personal-use migration acceptance.

## Clarifications

### Session 2026-08-09

- Q: In which context must the entire system start through one user action? → A: Both development and packaged release
- Q: Which macOS distribution profile is required for the current migration? → A: Personal use; a locally built/ad-hoc-signed `.app` is sufficient, while Developer ID distribution is deferred until credentials exist

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Run the game-master workspace without behavior loss (Priority: P1)

As a game master, I can launch the migrated desktop application, create or open an existing campaign session, author terminal content, save changes, and control the live terminal with the same observable behavior as before the migration.

**Why this priority**: The desktop workspace is the operator's entry point and owns the durable campaign workflow. A migration that cannot safely open, edit, and save existing sessions is not usable.

**Independent Test**: Launch the migrated application with no existing writable data, create a session, author representative nested content, close and reopen it, and verify that the saved content and game-master controls behave as documented.

**Acceptance Scenarios**:

1. **Given** a first launch on macOS, **When** the game master starts the application, **Then** the workspace opens at its documented size constraints, the player server is available, and the bundled demonstration session remains read-only until the game master explicitly chooses a writable copy.
2. **Given** an existing compatible version-1 session at a user-selected path, **When** the game master opens it, edits terminals and nested content, and waits for save completion, **Then** the same path contains valid, human-readable session data that reopens with the accepted edits.
3. **Given** no active session, **When** the game master cancels a new or open dialog, **Then** no error is reported and no save target or session state changes.
4. **Given** a terminal being edited separately from the live terminal, **When** the game master changes ordinary content, **Then** connected players see no change until the game master explicitly publishes it.
5. **Given** malformed session data or an unavailable destination, **When** an open or save operation fails, **Then** the game master receives a useful error and the previous active session and save target remain intact.
6. **Given** the documented development prerequisites are installed, **When** a user runs the single documented command from the repository root, **Then** the desktop workspace, player server, and required frontend assets start together without separate component-start commands.

---

### User Story 2 - Preserve synchronized player gameplay (Priority: P1)

As a player, I can open the displayed player address in a normal browser, see the Fallout-style terminal, participate in shared navigation and hacking, reconnect after interruption, and receive the same authoritative state as every other connected player.

**Why this priority**: Browser broadcast and shared state are the product's defining gameplay capabilities and must remain compatible throughout the desktop-runtime migration.

**Independent Test**: Connect between four and seven browser clients to one live terminal, exercise navigation and a hacking puzzle from multiple clients, disconnect and reconnect one client, and verify that all clients converge on the same server-owned state without exposing private puzzle data.

**Acceptance Scenarios**:

1. **Given** the application has started on a host with a local network address, **When** a player opens the displayed address, **Then** the player application and its static assets load and the game-master client count increases.
2. **Given** four to seven connected players and a live terminal, **When** any player enters a folder, opens a record, runs a command, goes back, or submits a hacking action, **Then** every player displays the next authoritative state and none applies a canonical transition before the server response.
3. **Given** a player disconnects during an active, solved, or failed hacking puzzle, **When** the browser reconnects, **Then** it receives the current sanitized live snapshot rather than a newly generated puzzle.
4. **Given** malformed, stale, unsupported, or tampered browser input, **When** the server receives it, **Then** canonical navigation and hacking state remain valid and the application continues serving other clients.
5. **Given** optional sound assets are missing or playback is denied, **When** a player uses the terminal, **Then** all visual states and gameplay inputs remain usable.
6. **Given** a live terminal menu overflows the player viewport, **When** a player scrolls from the beginning to the end of the menu, **Then** only the active presentation state occupies layout space and neither the idle waiting message nor the blocked-access message is revealed.

---

### User Story 3 - Operate protected local and public access (Priority: P2)

As a game master, I can use local-only operation by default or explicitly start protected public access, see which address is active, open that address safely, and shut down without leaving tunnel processes or credential material behind.

**Why this priority**: Public access is optional, but migration must not weaken its fail-closed authentication or cleanup guarantees.

**Independent Test**: Run once in local mode and once in public mode with valid and invalid credentials; verify visible status, authenticated HTTP and WebSocket behavior, failure isolation, and shutdown cleanup.

**Acceptance Scenarios**:

1. **Given** public mode is not requested, **When** the application starts, **Then** it exposes the local player address and does not start a public tunnel.
2. **Given** public mode is requested without valid credentials, **When** startup validates its configuration, **Then** no unprotected tunnel starts, local operation remains available, and the game master sees a non-secret error.
3. **Given** valid public configuration, **When** the tunnel becomes ready, **Then** the game master sees both public and local address context and anonymous HTTP and WebSocket access is rejected.
4. **Given** an active public tunnel, **When** the application closes, **Then** the tunnel is asked to terminate and temporary credential-bearing policy material is removed on every handled completion or failure path.
5. **Given** a displayed address containing a malformed or unsupported protocol, **When** the game master selects it, **Then** no external application is opened.

---

### User Story 4 - Install, validate, and roll back the personal-use application (Priority: P2)

As the application owner, I can build the migrated application for my macOS Apple Silicon computer, validate behavioral parity against repeatable checks, and retain the previous application until the personal-use candidate passes its acceptance gate.

**Why this priority**: A runtime replacement changes startup, packaging, filesystem, process, and browser-engine boundaries. A staged release and explicit rollback point reduce the chance of losing campaign data or live-session capability.

**Independent Test**: Produce a locally built/ad-hoc-signed `.app` on a supported macOS Apple Silicon environment, launch it without developer tooling, run the documented migration validation suite, and demonstrate that the existing application remains runnable until the new candidate passes. If Developer ID credentials are later supplied, validate the optional public-release DMG separately.

**Acceptance Scenarios**:

1. **Given** a supported macOS Apple Silicon build environment, **When** the owner builds the personal-use application, **Then** the `.app` launches without a developer runtime and includes all master, player, font, sound, and sample-session assets.
2. **Given** the owner launches a trusted local/ad-hoc-signed personal-use candidate, **When** macOS requires a one-time manual approval, **Then** the documented system Privacy & Security flow is sufficient and the absence of Developer ID credentials does not fail personal-use acceptance. **Given instead** that the application is intentionally prepared for public distribution and valid Developer ID credentials exist, **Then** the public candidate uses the hardened runtime, passes notarization, has the ticket stapled, and opens through Gatekeeper without bypass instructions.
3. **Given** representative version-1 session fixtures and protocol scenarios, **When** the parity suite runs against the migration candidate, **Then** session serialization, domain transitions, public projections, browser messages, and shutdown behavior match the documented contracts.
4. **Given** a migration candidate that fails any acceptance scenario required by the selected distribution profile, **When** readiness is assessed, **Then** the old runtime remains the rollback application and user session files require no rollback transformation.
5. **Given** an installed personal-use application, **When** the owner launches it once, **Then** the desktop workspace and player server start together without requiring a terminal or a separately launched frontend or server process.

### Edge Cases

- The default player port is already in use or server startup fails before the desktop becomes ready.
- No non-internal IPv4 address is available, a network interface changes, or a player uses an IPv6-only environment.
- A browser connects, disconnects, or reconnects while live content is being replaced or a hacking action is processed.
- Four to seven browsers submit valid navigation or hacking requests in rapid succession.
- A session is valid JSON but has an unsupported version, malformed recursive nodes, duplicate identifiers, excessive nesting, or fields outside documented bounds.
- Autosaves arrive faster than disk writes complete, the file is externally modified, or the destination becomes unavailable.
- The player server or public tunnel is still active when the desktop window closes or startup partially fails.
- The tunnel executable is missing, exits early, writes mixed log formats, times out, or ignores the first termination request.
- The `.app` bundle is read-only, the user declines Documents access, or the selected session destination becomes unavailable.
- Fonts, sounds, demonstration data, or generated frontend bindings are missing from a production package.
- The host WebKit engine has rendering or audio differences, or Developer ID credentials are unavailable; credential absence must be reported as public-distribution checks not applicable rather than as a personal-use blocker.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The migrated application MUST preserve every accepted user-visible behavior and public contract in the six existing feature specifications unless this specification explicitly changes it.
- **FR-002**: The desktop application MUST start its player server before reporting the game-master workspace ready and MUST stop owned server and tunnel resources during orderly shutdown and handled partial-startup failure.
- **FR-003**: The game-master workspace MUST retain the current minimum and initial window dimensions, dark presentation, restrictive content policy, and absence of direct privileged filesystem or process access from authored browser scripts.
- **FR-004**: Privileged desktop capabilities MUST be exposed through a narrow, explicitly registered interface that validates untrusted session, broadcast, path, and URL inputs before performing privileged work.
- **FR-005**: The migrated application MUST create, open, and save version-1 session JSON without changing the documented field names, content-tree variants, runtime-state exclusions, indentation, or explicit save-target behavior.
- **FR-006**: The migration MUST NOT overwrite, relocate, or transform an existing user-selected session file without the game master's explicit action.
- **FR-007**: Save operations MUST be ordered so completion status cannot report an older revision after a newer accepted revision and the final file represents the latest accepted session state.
- **FR-008**: The player server MUST continue binding to all interfaces on port `3690` by default, serve the player application and allowlisted sound discovery, and report an operator-usable player address.
- **FR-009**: The player HTTP and WebSocket service MUST remain available to other devices independently of the desktop webview's internal asset-loading mechanism.
- **FR-010**: Existing WebSocket message names, directions, required payload fields, validation effects, reconnect behavior, and broadcast semantics MUST remain backward-compatible for the bundled player client.
- **FR-011**: Live terminal, shared navigation, connection, and hacking state MUST remain server-authoritative, process-local, and absent from session JSON.
- **FR-012**: Private hacking fields, including the secret word and private candidate lookup, MUST never appear in player-facing live or hacking projections.
- **FR-013**: Concurrent player actions and desktop control requests MUST be serialized or synchronized so canonical state, client registration, and outbound broadcasts remain race-free.
- **FR-014**: The game-master workspace MUST receive current player-address, client-count, and public hacking-status changes without polling and without direct access to server internals.
- **FR-015**: The existing master and player visual, keyboard, pointer, reveal-animation, and optional-audio behaviors MUST remain usable on the supported macOS Apple Silicon target.
  - **Clarification (BUG-001)**: Player presentation states MUST be layout-exclusive: inactive connection, idle, normal list, record, hacking, and blocked containers MUST occupy no layout space and MUST NOT become visible through scrolling, regardless of class-level `display` declarations.
- **FR-016**: Public tunneling MUST remain opt-in, fail closed before process creation when credentials are absent or invalid, enforce authentication for both HTTP and WebSocket access, and avoid exposing credentials in UI or diagnostic output.
- **FR-017**: Temporary credential policy material MUST use private filesystem permissions where supported and MUST be removed after successful tunnel discovery, handled startup failure, and application shutdown.
- **FR-018**: External addresses MUST be opened only after validating that their final parsed protocol is HTTP or HTTPS.
- **FR-019**: Production packages MUST contain the desktop frontend, player frontend, fonts, sounds, sample session, and all runtime integration assets required for offline application startup.
- **FR-020**: The current macOS process MUST provide an Apple Silicon `.app` that is locally built or ad-hoc signed, passes bundle integrity and single-launch checks, and is acceptable for personal use on the owner's Mac. Developer ID signing, notarization, stapling, DMG creation, and Gatekeeper-without-bypass validation MUST be required only when the optional public-distribution profile is selected; unavailable credentials MUST NOT block personal-use acceptance.
- **FR-021**: On macOS, new user session dialogs MUST default to `~/Documents/Fallout Terminal/Sessions/`; bundled demo data MUST remain read-only until explicitly copied; app-managed metadata MUST use `~/Library/Application Support/com.vaulttec.fallout-terminal/`; and autosave MUST retain the explicitly selected path.
- **FR-022**: The migration MUST provide automated tests for session compatibility, navigation and hacking domain transitions, player protocol projections, multi-client convergence, privileged-interface validation, tunnel credential handling, and owned-resource shutdown.
- **FR-023**: The migration MUST provide a runnable end-to-end validation guide covering game-master authoring, four-to-seven-browser synchronization, reconnect, optional audio degradation, local/public access, session reopen, macOS storage, personal-use app packaging, and the conditional public-distribution path.
- **FR-024**: The existing runtime and build path MUST remain available until all P1 scenarios, security checks, session-compatibility checks, and package checks required by the selected distribution profile pass for the migration candidate. Public signing/notarization checks are not part of the current personal-use gate.
- **FR-025**: Removing the old runtime, its bridge, and its dependency tree MUST be the final migration action and MUST NOT remove browser assets still used by the game-master or player experiences.
- **FR-026**: The migration MUST provide one documented repository-root command that starts the complete development system after prerequisites are installed, and the packaged release MUST start all required runtime components from one application launch; neither path may require separate frontend or player-server start commands.

### Impacted Application Surfaces *(mandatory)*

- **Electron main (`main.js`)**: Affected — its lifecycle, native dialogs, filesystem persistence, player-server startup, tunnel startup, external URL opening, and renderer notifications move behind the replacement desktop runtime.
- **Preload IPC (`preload.js`)**: Affected — the existing narrow contract is replaced by an equivalently narrow privileged interface and event bridge; a compatibility adapter may preserve the renderer-facing method names during migration.
- **Master UI (`master/`)**: Affected — behavior and presentation remain stable, while startup integration and privileged calls are adapted and revalidated under the replacement webview.
- **Server (`server/`)**: Affected — HTTP, WebSocket, authoritative live state, navigation, hacking, sound discovery, and tunnel integration are replaced while retaining their public behavior.
- **Player UI (`client/`)**: Affected by compatibility validation — its protocol, rendering, audio, and reconnect behavior remain stable; changes are limited to defects required for parity or packaging.
- **Session data (`sessions/`)**: Compatibility-critical but shape-unaffected — version-1 examples and user-owned files remain readable and writable without migration.
- **Packaging/public access**: Affected — the build toolchain, application-bundle layout, personal-use ad-hoc signing, optional public signing/notarization and DMG output, session locations, and tunnel process ownership change.

### State and Contract Requirements

- **Session compatibility**: Version-1 session fields and recursive node variants remain unchanged. Runtime state remains excluded. Missing, unsupported, or malformed data must fail without replacing the active save target; stricter validation may reject previously malformed files but must not alter valid files silently. As an explicit migration behavior change, packaged macOS builds no longer auto-seed beside the executable: bundled demo data is copied only by user choice.
- **Privileged desktop contract**: The master can create, open, and save sessions; start, update, and clear a live terminal; force hacking success; open a validated player URL; and subscribe to server information, client count, and public hacking status. Calls return structured success, cancellation, or non-secret error results.
- **WebSocket contract**: `TERMINAL_LIVE`, `TERMINAL_UPDATE`, `TERMINAL_CLEAR`, `NAV_ACTION`, `NAV_STATE`, `HACK_GUESS`, `HACK_ADMIN`, and `HACK_STATE` retain their documented direction, payload, validation, sanitization, and broadcast behavior.
- **Reconnect behavior**: A newly connected or reconnected player receives the current sanitized live snapshot when one exists and receives no stale snapshot when no terminal is live.
- **HTTP/static contract**: `/` and bundled player asset paths serve the browser client; `/api/sounds/:folder` returns only supported files from allowlisted categories and degrades to an empty list on optional asset errors.

### Security and Privacy Requirements

- Browser-authored code MUST NOT gain arbitrary filesystem, process, environment, or network privileges through the desktop bridge.
- Session, broadcast, WebSocket, and URL inputs MUST be validated again at their privileged or canonical boundary.
- The master page MUST retain a restrictive content policy compatible with only the resources and bridge operations it needs.
- Public access MUST require valid credentials before starting and MUST not persist or disclose those credentials in project data, session data, logs, UI status, or long-lived files.
- Private hacking state MUST remain server-side and local campaign sessions MUST not be transmitted externally by the migration.
- Owned network listeners and child processes MUST have deterministic shutdown paths to avoid unintentionally leaving public access active.

### Key Entities

- **Session**: A user-owned version-1 JSON document with a name and terminal collection; it retains its existing portable representation and explicit active save path.
- **Terminal**: A durable terminal identity, display name, hacking level, introduction text, and recursive content root.
- **Content node**: A folder with child nodes, a command with output text, or an entry with descriptive text; node identifiers and root semantics remain compatible.
- **Live state**: The process-local authoritative terminal snapshot, shared navigation, optional private hacking puzzle, and public hacking projection.
- **Player connection**: A registered browser WebSocket that receives canonical broadcasts and contributes validated action requests.
- **Server information**: The local address and optional public address or non-secret tunnel failure displayed to the game master.
- **Tunnel configuration**: Ephemeral, validated public-access settings and short-lived credential policy material used to start one owned tunnel process.
- **Distribution candidate**: A macOS Apple Silicon `.app` containing all required runtime and browser assets, with recorded architecture, integrity, ad-hoc signature, asset, single-launch, and personal-use evidence; when the optional public profile is selected, it additionally includes a DMG and Developer ID, hardened-runtime, notarization, stapling, and Gatekeeper evidence.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every acceptance scenario in the six existing migrated feature specifications passes against the migration candidate, with no intentional user-visible behavior difference left undocumented.
- **SC-002**: In tests with four, five, six, and seven simultaneous browsers covering at least 25 mixed navigation and hacking actions plus one reconnect, all browsers converge on identical authoritative state after every accepted action and expose zero private hacking fields.
- **SC-003**: A fixture set containing all supported version-1 terminal and content-node variants opens and round-trips without semantic changes, and 20 rapid accepted edits leave the final saved file at the latest revision.
- **SC-004**: Application startup either presents a usable workspace and player address within 5 seconds on the supported reference machine or presents a clear actionable failure without leaving a listener or child process running.
- **SC-005**: Invalid public credentials result in zero tunnel process starts; valid credentials reject all anonymous HTTP and WebSocket attempts in the public-access validation scenario.
- **SC-006**: Closing the application during local-only operation, active player connections, active public access, and partial startup leaves zero owned player listeners, tunnel processes, or credential-policy directories after the documented shutdown timeout.
- **SC-007**: The personal-use macOS `.app` launches on the owner's Apple Silicon Mac, contains every required asset category, passes bundle integrity and ad-hoc-signature checks, and completes the P1 smoke scenarios without a separately installed developer toolchain. If the optional public profile is selected, its DMG also passes Developer ID, notarization, stapling, and Gatekeeper checks.
- **SC-008**: All required automated migration checks pass repeatedly on a clean checkout, and the end-to-end validation guide can be completed without undocumented setup or manual file repair.
- **SC-009**: On a clean checkout with documented prerequisites, one documented repository-root command reaches a usable development workspace and player address; in an installed release, one application launch reaches the same state, with zero separately started frontend or player-server processes in either case.

### Implementation Verification

Implementation verification completed on 2026-08-09. All FR-001–FR-026 and
SC-001–SC-009 pass for the selected personal-use profile; the conditional
Developer ID/notarization/DMG/Gatekeeper branch is correctly recorded as
`N/A (personal profile)` and remains closed for public distribution. The
requirement-by-requirement evidence index is maintained in
[`checklists/requirements.md`](checklists/requirements.md), with executable and
manual results in [`quickstart.md`](quickstart.md).

## Assumptions

- macOS 13 or later on Apple Silicon is the primary supported release target during this migration. Intel/universal macOS and Windows distributions are deferred.
- The migration targets the stable Wails v2 line requested by the user; adopting a later major version is a separate decision.
- The application continues to own one desktop game-master window, one player server, and at most one live terminal and public tunnel per process.
- Player devices continue using standards-compliant browsers with WebSocket, Web Audio, and current CSS support.
- Valid version-1 session files remain the compatibility baseline; no new session fields are required by this runtime migration.
- Existing player message names and payloads remain internal to the bundled player client but are preserved to minimize simultaneous frontend changes.
- The current acceptance owner has no Developer ID identity and needs the application only for personal use; a locally built/ad-hoc-signed `.app` is therefore a valid installed candidate, not merely a development artifact.
- If public distribution is selected later, the release operator can supply appropriate Developer ID and notarization credentials and run the preserved public-release profile before publishing.
- The previous application remains available on a separate branch or release artifact until migration acceptance is complete.

## Out of Scope

- New game-master authoring capabilities, player gameplay, terminal node types, or hacking rules.
- A session-format version increase, database introduction, cloud synchronization, or account system.
- Multiple independent desktop windows, system tray behavior, mobile applications, or a browser-only game-master workspace.
- Replacing ngrok with another public-access provider.
- Visual redesign, frontend framework adoption, localization, or broad accessibility remediation unrelated to migration parity.
- Changing the shared-navigation model into per-player navigation.
- Supporting Intel/universal macOS, Windows, Linux, or other desktop architectures in the initial migration release.
- Public distribution to unrelated users while Developer ID credentials are unavailable; the documented signing/notarization automation is retained for a future profile switch.
