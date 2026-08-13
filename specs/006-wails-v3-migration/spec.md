# Feature Specification: Wails v3 Runtime Migration

**Feature Branch**: `006-wails-v3-migration`
**Created**: 2026-08-13
**Status**: Draft
**Baseline**: `main` at `f1084b3df8b5630862bdf7a0f347b599156653ef`

**Bugfix**: 2026-08-14 — [ANALYZE-2026-08-14] Aligned active Wails commands with constitution 3.3.1, reaffirmed the unchanged feature-005 runtime-status contract, and made soak acceptance measurable.
**Bugfix**: 2026-08-14 — [ANALYZE-CUTOVER-2026-08-14] Made cutover evidence ordering, historical immutability, RSS sampling, and the conditional public-soak workload deterministic.

## Source of Truth and Scope

This feature replaces the accepted Wails v2 desktop host with Wails v3 without intentionally changing the Fallout Terminal product. The current constitution, current baseline code, and completed specifications 001–005 are the evidence set. For the private desktop bridge and public player behavior, the post-ConnectRPC contracts in feature 005 and the current code take precedence. Completed Electron behavior is historical migration evidence only and is not a current compatibility oracle.

The migration starts now against one concrete, verified-compatible set of exact Wails v3 beta dependencies selected during planning; Wails v3 general availability is not a prerequisite. The migration is limited to the native runtime, private bridge integration, development tooling, and macOS packaging. A game master and connected players must observe the same gameplay, persistence, security, visual, and public-network behavior as on the accepted baseline. Developer-visible changes are limited to the documented Wails v3 commands, explicit generated bindings, Taskfile/build assets, and pinned Wails v3 prerequisites. Wails v3 becomes the accepted production runtime only after every required migration gate passes.

## Clarifications

### Session 2026-08-13

- Q: Must the migration wait for Wails v3 general availability? → A: No. Proceed now with one exact, verified-compatible beta version set; accept it for production only after all migration gates, and permit no `@latest` in source or CI commands.
- Q: May the migration add Wails v3 product features or change the window model? → A: No. Preserve one master window and current user-visible behavior; multi-window, tray, updater, mobile, and other new v3 features are out of scope.
- Q: Which bridge and protocol boundaries must remain stable? → A: Preserve `desktop-api.js`, named desktop-operation semantics, all four event names and payloads, and feature-005 protobuf, ConnectRPC, and session contracts; keep lifecycle startup and shutdown operations out of frontend bindings.
- Q: Which development, build, and packaging model applies? → A: Adopt Wails v3 explicit bindings, configuration, Taskfile, and build assets while retaining one root development command, locked dependencies, both Vite applications, embedded resources, and the documented macOS app location unless planning justifies and consistently updates every consumer.
- Q: Which release profile is required for acceptance? → A: macOS 13+ Apple Silicon personal use with a local or ad-hoc-signed app; public Developer ID, notarized DMG, and related checks remain conditional and require real evidence.
- Q: What is the Wails v2 rollback authority? → A: The actual pre-migration `main` commit is the source rollback; preserve the Electron rollback document as history and create a separate Wails v3 migration rollback record, with no permanent v2/v3 switch.
- Q: How must application-owned startup failures behave? → A: Publish them through runtime status and actionable master UI state, preserve local mode after tunnel failure, unwind partial acquisition, and never reduce them to unexplained framework exits.

### Session 2026-08-14

- Q: Does startup observability require a new serialized lifecycle phase? → A: No. Internal phase remains host-owned and test-observable; the master derives starting, ready-local, ready-public, and failed presentation from the existing feature-005 runtime-status fields and renders the existing `startupError` without a protobuf or native-shape change.
- Q: How are repository-owned Go development tools invoked? → A: Each tool has one isolated `tools/<tool>/` module, repository commands use `go tool -modfile=tools/<tool>/go.mod <command>`, and the root application module contains no tool declaration or tool-only dependency.

## User Scenarios & Testing

### User Story 1 - Run the same native master application (Priority: P1)

A game master launches one native Fallout Terminal master application on a supported Apple Silicon Mac and continues to create, open, edit, copy, and save representative sessions and player rosters through the same dark single-window experience.

**Why this priority**: The master application is the game master's primary workspace, and any visible or semantic regression would make the runtime migration unacceptable.

**Independent Test**: Launch the migration candidate on macOS 13+ arm64, compare its title, initial and minimum size, appearance, controls, file locations, and representative session and roster workflows with the accepted baseline, then reopen the saved files with the accepted version-1 codecs.

**Acceptance Scenarios**:

1. **Given** a supported macOS 13+ Apple Silicon system, **When** the game master launches the candidate, **Then** exactly one native master window opens with the accepted title, initial dimensions, minimum dimensions, dark presentation, and master controls.
2. **Given** representative valid session and player-roster files, **When** the game master creates, opens, edits, copies, saves, and reopens them, **Then** their business meaning, selected paths, relative references, validation, and version-1 JSON compatibility remain unchanged.
3. **Given** the bundled demonstration session, **When** it is opened from the application bundle, **Then** it remains read-only until the game master explicitly chooses a writable copy destination.
4. **Given** a save target selected by the game master, **When** rapid accepted edits are saved, **Then** the newest accepted revision is the final durable revision and an older completion cannot overwrite or report itself after it.
5. **Given** an invalid session, player configuration, path, or external URL, **When** it reaches a privileged operation, **Then** the accepted validation, cancellation, redacted error, and no-partial-publication behavior is preserved.

---

### User Story 2 - Start and stop every owned runtime resource once (Priority: P1)

The application starts and owns the player listener, generated player service, optional public tunnel, session-storage worker, desktop adapter, and their shutdown cleanup as one bounded lifecycle. Local play remains available by default, and protected public mode remains opt-in and fail-closed.

**Why this priority**: Incorrect lifecycle ownership can make the application fail at launch, leak processes or ports, or lose local play when public tunneling is unavailable.

**Independent Test**: Exercise normal startup, repeated lifecycle calls, occupied-port failure, desktop-adapter failure, tunnel validation and startup failure, normal quit, and partial-start cleanup while recording acquisitions, status publications, releases, process ownership, and timeout results.

**Acceptance Scenarios**:

1. **Given** a normal local launch, **When** startup completes, **Then** the player listener is acquired once, local server information is visible to the master, and no public tunnel is started by default.
2. **Given** protected public mode with valid configuration, **When** startup completes, **Then** at most one owned tunnel process starts after local play is available and the master receives safe public status without credentials.
3. **Given** public tunneling fails after local startup, **When** the failure is handled, **Then** local play remains usable and the master receives an actionable non-secret failure instead of an unexplained process exit.
4. **Given** the player listener cannot start, **When** bounded startup fails, **Then** the failure is observable through the master runtime status or startup presentation and every partially acquired resource is released.
5. **Given** normal quit, repeated shutdown, or handled partial startup, **When** cleanup runs, **Then** tunnel, listener and streams, session worker, and desktop adapter are released at most once in the accepted order and within the accepted timeout.

---

### User Story 3 - Keep browser-player gameplay and privacy unchanged (Priority: P1)

Four to seven browser players load the same bundled player application and continue selecting characters, receiving authoritative snapshots and streamed revisions, navigating, hacking, reconnecting, and playing sounds with the same convergence, authority, replay, privacy, and security behavior as the accepted post-ConnectRPC baseline.

**Why this priority**: The native-host migration must not disturb the separately served public game experience or reopen the removed legacy protocol.

**Independent Test**: Run four-, five-, six-, and seven-browser journeys through the bundled player UI using mixed selection, navigation, hacking, reconnect, replay, slow-subscriber, and sound cases, and compare authoritative revisions and public projections with feature 005.

**Acceptance Scenarios**:

1. **Given** four to seven browser players, **When** they connect to the application-owned listener, **Then** each loads the bundled same-origin player UI and generated client and begins with one complete authoritative snapshot.
2. **Given** concurrent valid and invalid player actions, **When** the generated public procedures process them, **Then** canonical state, revision, replay, authorization, request-limit, and convergence behavior matches feature 005.
3. **Given** a player disconnects, reconnects, or loses a slow or overflowing stream, **When** it subscribes again, **Then** it recovers from a complete current snapshot without blocking canonical mutation, responsive players, or shutdown.
4. **Given** navigation, hacking, and sound-producing authoritative transitions, **When** players observe them, **Then** rendering, controls, reconnect behavior, and one-shot audio cues remain unchanged.
5. **Given** public or authenticated-ngrok access, **When** a player enumerates or crafts requests, **Then** no private desktop capability, credential, native path, secret word, private candidate, future outcome, runtime status, or other private field is exposed.
6. **Given** the final candidate, **When** its listener, source, routes, and browser bundle are inspected, **Then** the generated public service remains the only player protocol and no Wails or legacy WebSocket capability is reachable through the player listener.

---

### User Story 4 - Preserve the private desktop facade and events (Priority: P1)

The master frontend continues using its runtime-neutral desktop facade to invoke every accepted private desktop operation and to receive server, client-count, hacking, and coordination changes without polling or JavaScript-visible contract drift.

**Why this priority**: The bridge is the trusted boundary between browser-authored master UI code and native privileges; payload or exposure drift is both a product and security regression.

**Independent Test**: Generate a clean Wails v3 binding inventory, exercise every accepted private operation and event through the master compatibility boundary, compare arguments and native JavaScript values with baseline fixtures, and inspect the generated surface for unwanted capabilities.

**Acceptance Scenarios**:

1. **Given** the accepted private operation inventory, **When** the master invokes each operation, **Then** its validation, cancellation, redacted errors, result semantics, and JavaScript-visible value or object shape match the baseline.
2. **Given** any Wails v3 generated namespace or file-layout change, **When** the master UI initializes, **Then** `desktop-api.js` absorbs that change and the rest of the master UI continues using the same runtime-neutral facade.
3. **Given** a subscription to `server-info`, `client-count`, `hack-state`, or `coordination-state`, **When** the Wails v3 binding/event bridge becomes ready and an event races the initial runtime snapshot, **Then** the master is not presented as ready before calls and subscriptions are usable, and the subscriber receives current initial state without a stale snapshot overwriting a newer event.
4. **Given** an event listener is released once or more than once, **When** unsubscribe runs, **Then** exactly one underlying runtime subscription is released and no later callback is delivered to that listener.
5. **Given** a structured bridge request, result, status, or event, **When** it crosses the private boundary, **Then** its existing explicit semantic adapter is used and no protobuf binary, Base64, ProtoJSON, generic envelope, or generic dispatcher is introduced.
6. **Given** the generated Wails v3 binding inventory, **When** it is compared with the accepted desktop inventory, **Then** every required operation exists and no lifecycle method, arbitrary native primitive, generic dispatch surface, or public player capability is exposed.

---

### User Story 5 - Develop and build reproducibly from the repository root (Priority: P2)

A developer installs documented pinned prerequisites and uses one repository-root command to start the complete development system. A clean checkout can deterministically regenerate private bindings and protobuf code, build both browser applications, run verification, and build or package the native application without manually starting another frontend or server.

**Why this priority**: The beta runtime is acceptable only when its Go module, command-line tool, frontend runtime, generated outputs, and build orchestration are concrete and reproducible.

**Independent Test**: From a clean checkout with only the documented prerequisites, run the root development command, clean generation, the complete automated gate set, two clean builds, and packaging; verify that no command resolves a floating dependency and no unexplained tracked diff remains.

**Acceptance Scenarios**:

1. **Given** the documented exact prerequisites and Wails v3 Taskfile/configuration, **When** a developer runs `go tool -modfile=tools/wails/go.mod wails3 dev` from the repository root, **Then** the master Vite application, player Vite application, explicit generated bindings, embedded assets, native host, player listener, generated player service, and optional configured tunnel are prepared or started without another manual process.
2. **Given** a clean checkout, **When** Wails bindings and protobuf outputs are generated twice, **Then** the second generation is byte-stable and leaves no unexplained tracked diff.
3. **Given** two consecutive clean builds, **When** their inputs and selected profile are identical, **Then** both complete from the root without an undeclared development server, manually prebuilt asset, or floating `latest` resolution.
4. **Given** the repository verification suite, **When** it is adapted for Wails v3, **Then** existing assertions are preserved or strengthened rather than removed or weakened to obtain a pass.
5. **Given** the pinned Wails v3 Go, CLI, frontend runtime, and plugin versions, **When** their compatibility is checked, **Then** all recorded versions are exact, mutually compatible, and consistent across source, lockfiles, documentation, automation, and generated output.

---

### User Story 6 - Package a self-contained personal-use macOS app (Priority: P2)

The owner creates an arm64 application bundle for macOS 13+ that launches and runs the complete accepted experience without Go, Node, npm, Vite, Wails, a development server, or internet connectivity.

**Why this priority**: A runtime migration is not complete until the application can be used as a standalone native package on the supported target.

**Independent Test**: Package the candidate on Apple Silicon, inspect its architecture, signature, resources, and linked runtime needs, disconnect external network access, launch the bundle, exercise master and local-player smoke journeys, and confirm owned-process cleanup on quit.

**Acceptance Scenarios**:

1. **Given** the personal-use profile, **When** the owner packages the candidate, **Then** the result is a local or ad-hoc-signed macOS 13+ arm64 `.app` at `build/bin/Fallout Terminal.app` that passes the documented integrity and launch checks, unless the approved plan records a compelling path change and updates every consumer consistently.
2. **Given** no developer tools, development server, or internet connection, **When** the packaged application launches, **Then** master assets, Wails generated bindings, player assets, generated player client, fonts, sounds, and the bundled demonstration session are present and usable.
3. **Given** the packaged application is running, **When** local players connect and the master quits, **Then** the player listener starts once and the listener and any owned tunnel process are released within the accepted shutdown timeout.
4. **Given** the personal-use profile and unavailable public-release credentials, **When** acceptance is recorded, **Then** Developer ID signing, hardened runtime, notarization, stapling, DMG, and Gatekeeper-without-bypass are recorded as `NOT RUN` and do not fail personal-use acceptance.
5. **Given** an explicitly selected public-release profile with credentials available, **When** the release candidate is evaluated, **Then** every required signing, hardened-runtime, notarization, stapling, DMG, and Gatekeeper gate must pass before it is represented as a public release.

---

### User Story 7 - Cut over safely with an immutable Wails v2 rollback (Priority: P2)

Before final cutover, the owner can return to one immutable accepted Wails v2 source or accepted artifact without converting version-1 session or player-configuration data. Wails v2 remains the production fallback until the Wails v3 candidate passes every required parity, security, lifecycle, package, and rollback gate.

**Why this priority**: Wails v3 is beta at this baseline, so an explicit rollback point and cutover gate protect campaign data and live-session availability.

**Independent Test**: Verify the recorded baseline commit, optional accepted artifact digest, migration-owned Wails v3 rollback record, historical-only Electron rollback document, unchanged version-1 files, rollback-trigger decision table, temporary-coexistence expiry, and final absence of active Wails v2 source after all candidate gates pass.

**Acceptance Scenarios**:

1. **Given** implementation has not cut over production source, **When** rollback evidence is inspected, **Then** it identifies the immutable accepted Wails v2 `main` commit and, if built, the accepted executable and SHA-256 with its acceptance status.
2. **Given** a Wails v3 candidate failure matching a defined rollback trigger, **When** readiness is assessed, **Then** Wails v2 remains or becomes the production source or artifact without changing session-v1 or player-config-v1 files.
3. **Given** temporary v2/v3 coexistence on the migration branch, **When** the cutover gate is reached, **Then** the named owner removes coexistence and no permanent runtime flag or dual implementation reaches acceptance.
4. **Given** every required Wails v3 gate has passed, **When** final cutover occurs, **Then** active production source, dependencies, bindings, commands, configuration, tests, and documentation contain no Wails v2 runtime path while historical completed specifications remain unchanged.
5. **Given** external ngrok credentials or connectivity are unavailable, **When** final evidence is recorded, **Then** authenticated-ngrok soak results are `NOT RUN` and are not used as passing public-mode evidence.

## Edge Cases

- The player listener port is already occupied during startup.
- The native window or desktop adapter becomes ready after the listener starts but before an event sink can publish status.
- An event arrives between runtime subscription registration and initial status replay.
- An event listener is unsubscribed during its initial snapshot promise, during callback execution, or more than once.
- The optional tunnel validates but returns an unsafe, malformed, credential-bearing, or non-HTTPS public address after acquiring a process.
- Public tunneling fails while local play is healthy.
- Startup is canceled or times out after only some resources were acquired.
- Shutdown begins with active, blocked, reconnecting, or overflowing player streams and is called repeatedly.
- The bundled demonstration path is read-only or the installed application is launched from a different working directory.
- A session contains compatible unknown fields while a player-config file contains an unknown field that its strict codec must reject.
- Rapid save requests complete their filesystem work out of order.
- Generated bindings change namespace, module path, exported metadata, or promise behavior while application-owned payload semantics remain unchanged.
- The Wails v3 beta runtime crashes or fails to surface a startup error before the master frontend becomes interactive.
- The generated binding or event facility is not usable when the master would otherwise report ready; readiness must wait while runtime-status replay remains available after listener registration.
- A clean build accidentally relies on stale generated bindings, stale browser assets, an installed global tool with the wrong version, or a network-time `latest` lookup.
- The arm64 package has a valid directory shape but is missing generated bindings, player assets, fonts, sounds, demo data, or its expected ad-hoc signature.
- Wails v3 packaging proposes an application output path other than `build/bin/Fallout Terminal.app` while documentation, CI, validation, or launch consumers still use the established path.
- Public-release credentials are absent after the personal-use package has otherwise passed.
- Rollback is requested after a candidate has written valid version-1 files; those files must remain directly usable by the accepted Wails v2 baseline.

## Baseline Discrepancies to Resolve

- The accepted feature-005 contract states that lifecycle callbacks are private application boundaries and are not desktop dispatch surfaces, but the checked-in Wails v2 generated bindings currently export `Start` and `Shutdown` because the whole `App` value is bound. This migration treats that exposure as unintended contract drift: lifecycle semantics must remain internal, and the Wails v3 generated desktop inventory must not expose either method.
- `CopyDemo` is an accepted bound private operation with explicit writable-copy semantics, but the current `desktop-api.js` intentionally has no `copyDemo()` facade method and the current master UI has no separate control for it. The migration must retain the private operation and existing user flow without adding a new UI control; absence of a current facade method is not permission to remove the capability.
- Active setup, build, and package documentation still names Wails v2 commands and configuration because that is the accepted baseline. Those references are expected migration inputs, not evidence that Wails v2 commands should remain active after cutover.
- `docs/wails-migration-rollback.md` records the completed Electron-to-Wails migration and must remain historical; feature 006 requires its own Wails v2-to-v3 rollback record rather than repurposing or overwriting that document.

## Requirements

### Functional Requirements

- **FR-001**: The migration MUST replace the accepted Wails v2 desktop runtime with Wails v3 without intentionally changing gameplay, persistence, security, visual presentation, public network behavior, or user workflows.
- **FR-002**: The implementation MUST use the current constitution, baseline code, and completed specifications 001–005 as evidence, giving precedence to feature 005 and current code for post-ConnectRPC private and public behavior while surfacing rather than silently resolving active-contract discrepancies.
- **FR-003**: The candidate MUST retain one native master window on macOS 13+ arm64 with title `Fallout Terminal — Master Control`, initial size 1200×780, minimum size 900×600, and the accepted dark presentation.
- **FR-004**: The master experience MUST preserve every accepted control and the create, open, edit, explicit-copy, and save outcomes for representative sessions and player rosters.
- **FR-005**: Session persistence MUST preserve version-1 JSON field names, compatible unknown fields, relative references, validation, explicit selected targets, ordered revisions, atomic replacement, and zero semantic round-trip change.
- **FR-006**: Player-config persistence MUST preserve version 1, strict unknown-field and trailing-data rejection, identity validation, relative session references, complete-file atomic saves, and publication only after durable success.
- **FR-007**: macOS storage MUST preserve the Documents session default, Application Support metadata location, user-selected autosave target, and read-only bundled demonstration behavior until explicit copy.
- **FR-008**: One application process MUST compose, start, own, and stop the player listener, generated player service, optional owned ngrok process, session-storage worker, and desktop adapter exactly once.
- **FR-009**: Startup MUST remain bounded. Without changing the accepted runtime-status or event contracts, every application-owned actionable redacted startup failure MUST be recorded through existing runtime-status fields and rendered as actionable master UI state. Local server information MUST remain observable, optional tunnel failure MUST preserve local mode, partial acquisition MUST unwind, and internal lifecycle phase MUST remain host-owned and test-observable rather than serialized.
- **FR-010**: Local player access MUST remain enabled by default, while public tunneling MUST remain opt-in, authenticated, credential-redacted, and fail-closed before player capabilities when configuration is absent or invalid.
- **FR-011**: Failure of optional public tunneling MUST preserve usable local play and publish a non-secret failure status.
- **FR-012**: Player-listener startup failure MUST unwind partially acquired resources and remain observable to the master rather than appearing only as an unexplained process exit.
- **FR-013**: Shutdown MUST preserve the accepted ownership order and timeout for tunnel, player listener and streams, session worker, and desktop adapter and MUST remain safe after partial startup or repeated calls.
- **FR-014**: The player listener MUST expose only the bundled static player resources and the generated public player service and MUST expose no Wails runtime, private desktop bridge, health, reflection, generic capability discovery, or native operation.
- **FR-015**: The public service MUST preserve `fallout.terminal.player.v1.PlayerService`, every accepted procedure cardinality and meaning, same-origin binary protobuf transport, request limits, authentication boundary, player URL and port, revision ordering, replay, concurrency, cancellation, and reconnect behavior.
- **FR-016**: The final candidate MUST retain generated ConnectRPC as the only public player protocol and MUST NOT reintroduce the removed WebSocket protocol or a compatibility route.
- **FR-017**: Four to seven simultaneous browser players MUST continue to converge through character selection, authoritative snapshots and streamed revisions, navigation, hacking, reconnect, and sound behavior under the feature-005 authority rules.
- **FR-018**: Public projections, descriptors, errors, and generated player code MUST continue to expose zero desktop capabilities, credentials, native paths, runtime status, secret words, private candidates, future outcomes, or other private state.
- **FR-019**: Every private desktop operation in the accepted feature-005 inventory MUST remain available with its exact validation, cancellation, eligibility, redacted error, result semantics, and JavaScript-visible value or object shape.
- **FR-020**: `frontend/src/desktop-api.js` MUST remain the only runtime-neutral facade used by the master UI and MUST absorb Wails v3 generated namespace or module-path changes without leaking them to UI consumers.
- **FR-021**: Existing versioned private protobuf contracts and explicit adapters MUST continue to govern every application-owned bridge request, result, status, and event without becoming canonical mutable domain state.
- **FR-022**: The Wails bridge MUST NOT carry protobuf binary, Base64, ProtoJSON, a serialized protobuf envelope, or a generic map or dispatcher.
- **FR-023**: A protobuf schema change MUST occur only for a separately specified real application-contract semantic change and MUST NOT be justified solely by Wails binding metadata or namespace changes.
- **FR-024**: The event names `server-info`, `client-count`, `hack-state`, and `coordination-state` and their JavaScript-visible payload shapes MUST remain exact.
- **FR-025**: Every desktop event subscription MUST return one idempotent release function that releases exactly one underlying runtime listener and prevents callbacks after release.
- **FR-026**: The master MUST become ready only after generated desktop calls and named event subscriptions are usable, and every desktop event subscriber MUST register before runtime-status replay and receive current initial runtime, server, client, hacking, and coordination state without polling, missed initial state, duplicate effects, or a stale snapshot overwriting a newer event.
- **FR-027**: The generated Wails v3 desktop binding inventory MUST contain every required private operation and zero lifecycle methods, generic dispatchers, arbitrary filesystem, process, environment, or browser primitives, or public player procedures.
- **FR-028**: `Start` and `Shutdown` MUST remain internal lifecycle operations and MUST NOT appear in generated desktop bindings, while `CopyDemo` MUST remain a private operation with its accepted explicit-copy semantics and no newly introduced UI control.
- **FR-029**: Planning MUST select one concrete, mutually compatible, verified Wails v3 beta set for the Go module, CLI, frontend runtime, and frontend plugin, record every exact version in source and lockfiles, proceed without waiting for general availability, and permit no active source or CI command to name or resolve `@latest`.
- **FR-030**: A developer with the documented pinned prerequisites MUST be able to run `go tool -modfile=tools/wails/go.mod wails3 dev` once from the repository root to start the complete development system without separately starting a frontend, player listener, or tunnel supervisor.
- **FR-031**: The repository MUST adopt the Wails v3 configuration, explicit binding generation, Taskfile, and build-asset model so a clean checkout deterministically generates Wails bindings and protobuf code, builds both the `frontend/` and `client/` Vite applications and embedded resources, runs applicable tests, builds the native application, and packages the supported target without an undeclared manual prerequisite.
- **FR-032**: Clean binding and protobuf generation MUST be reproducible across two consecutive runs and MUST leave no unexplained tracked diff or manually edited generated file.
- **FR-033**: Two consecutive clean native builds with identical inputs MUST complete successfully without relying on stale generated output, a development server, a network-time package lookup, or a separately started process.
- **FR-034**: Existing Go, race, frontend, player, Playwright, Buf generation, drift, breaking, and relevant platform tests MUST pass unchanged or be deliberately adapted for Wails v3 without reducing their assertions.
- **FR-035**: The personal-use package MUST be a self-contained local or ad-hoc-signed macOS 13+ arm64 `.app` at the externally documented `build/bin/Fallout Terminal.app` location that launches without Go, Node, npm, Vite, Wails, a development server, or internet access, unless planning records a compelling path change and updates every source, CI, documentation, validation, and launch consumer consistently.
- **FR-036**: The packaged application MUST contain and load the master assets, Wails generated bindings, generated player client, player assets, fonts, sounds, and read-only bundled demonstration session.
- **FR-037**: The packaged application MUST start the player listener once and release it and any owned ngrok process on normal quit and handled partial startup within the accepted shutdown timeout.
- **FR-038**: Developer ID signing, hardened runtime, notarization, stapling, DMG creation, and Gatekeeper-without-bypass MUST be required only for an explicitly selected public-release candidate and MUST be recorded as `NOT RUN` when credentials are unavailable.
- **FR-039**: Before code cutover, a Wails v3 migration rollback record separate from the historical Electron rollback document MUST record the immutable accepted pre-migration `main` commit `f1084b3df8b5630862bdf7a0f347b599156653ef` as the canonical Wails v2 source rollback.
- **FR-040**: If an accepted Wails v2 baseline application is built, rollback evidence MUST record its exact executable, SHA-256 digest, build provenance, and acceptance status.
- **FR-041**: Before final cutover, the owner MUST be able to return to the recorded Wails v2 source or accepted artifact without transforming session-v1 or player-config-v1 data.
- **FR-042**: Temporary v2/v3 coexistence MUST be limited to the migration branch, have a named owner and expiry at cutover, include a removal task, and MUST NOT become a permanent runtime flag or dual implementation.
- **FR-043**: Rollback triggers MUST include session corruption or loss, bridge capability or payload drift, master or player parity loss, startup or shutdown leaks, public-access regression, missing packaged assets, unhandled beta-runtime crashes, and package or signature failure.
- **FR-044**: Wails v2 MUST remain the accepted production fallback until every required parity, security, persistence, lifecycle, generation, build, package, soak, and rollback gate for the selected profile passes.
- **FR-045**: Final cutover MUST remove every active Wails v2 import, CLI installation path, v2 command, generated binding dependency, runtime global, obsolete configuration, and permanent dual-runtime implementation while preserving historical completed specifications as history.
- **FR-046**: Final acceptance MUST include a local master/player soak lasting at least 60 minutes with four to seven concurrent players, at least 25 mixed operations, at least three reconnects, at least two save/reopen cycles, navigation, hacking, coordination, and sound. The run MUST record convergence and revision correctness after each operation, exactly one active listener during operation, and release of the listener and any owned tunnel within five seconds after quit. At minutes 15, 30, and 60, five application-RSS samples ten seconds apart MUST be collected with `ps -o rss= -p <APP_PID>` and reduced to a median KiB value; the local soak fails when both the 30-minute and 60-minute medians exceed 125% of the 15-minute median. The authenticated-ngrok soak MUST last at least 30 minutes with four to seven players, at least 15 mixed public operations, at least two reconnects, one rejected unauthorized request, one controlled tunnel-loss exercise proving local play remains usable, revision convergence after each accepted operation, credential redaction, private-field isolation, and final owned-resource cleanup within five seconds when real credentials and connectivity exist; otherwise it MUST be recorded as `NOT RUN` without serving as passing public-mode evidence.
- **FR-047**: Multi-window, mobile, system tray, updater, new desktop capabilities, persistence-format migration, public RPC redesign, ConnectRPC replacement, Wails-driven protobuf changes, new platforms, and mandatory public distribution MUST remain outside this migration.

## Key Entities

- **Migration Candidate**: One exactly versioned Wails v3 source and package state evaluated against the accepted baseline, with its commit, selected distribution profile, generated outputs, build evidence, and gate results.
- **Private Desktop Operation Inventory**: The allowlist of application-owned privileged operations available to the master compatibility boundary, including each operation's arguments, result shape, eligibility, validation, and redaction behavior.
- **Desktop Event Subscription**: One named master-runtime listener with an initial-status source, latest-event ordering rule, callback, and exactly one underlying release action.
- **Runtime Status Snapshot**: The detached master-visible state containing safe server information, active stream count, public hacking state, startup and save status, revisions, and private coordination state.
- **Owned Runtime Resource**: A player listener, generated stream set, optional tunnel process, session-storage worker, or desktop adapter whose acquisition, status, cleanup order, timeout, and idempotence belong to the application lifecycle.
- **Rollback Reference**: The immutable accepted Wails v2 source commit and, when produced, one executable digest with provenance and acceptance status that can resume operation without data conversion, recorded in a feature-006 migration rollback record separate from the preserved historical Electron rollback document.
- **Distribution Profile**: Either the required personal-use local/ad-hoc-signed arm64 application profile or an explicitly selected public-release profile with additional signing and trust gates.
- **Acceptance Evidence**: A command, test result, inventory, scan, fixture comparison, package inspection, soak record, or manual observation tied to one candidate and marked `PASS`, `FAIL`, or `NOT RUN` without promoting unavailable evidence to a pass.

## Success Criteria

### Measurable Outcomes

- **SC-001**: On macOS 13+ arm64, the candidate opens exactly one master window with the accepted title, 1200×780 initial dimensions, 900×600 minimum dimensions, dark presentation, and no missing accepted master control.
- **SC-002**: Representative session-v1 fixtures round-trip with zero semantic change, compatible unknown fields and relative references preserved, and 20 rapid accepted saves leave the newest revision on disk.
- **SC-003**: Representative player-config-v1 fixtures retain every strict validation and atomic-publication behavior with zero path or format change.
- **SC-004**: The generated desktop inventory contains all 25 accepted private operations, contains zero `Start` or `Shutdown` binding, and contains zero generic dispatcher or arbitrary native primitive.
- **SC-005**: Automated bridge compatibility tests exercise every accepted operation and all four named events with zero JavaScript-visible payload-shape, validation, cancellation, eligibility, or redaction drift.
- **SC-006**: Event tests observe the exact four names, no ready master state before calls and subscriptions are usable, listener registration before runtime-status replay, one underlying release per listener, zero callback after release, zero stale snapshot overwrites, and no missing initial server, client, hacking, or coordination state.
- **SC-007**: Normal, repeated, failed, canceled, and partial startup/shutdown tests record at most one acquisition and one release for every owned resource and complete cleanup within the accepted five-second shutdown timeout.
- **SC-008**: Player-listener failure is visible to the master in every tested failure path, while optional tunnel failure preserves a usable local URL and exposes zero credential or internal dependency detail.
- **SC-009**: Four-, five-, six-, and seven-browser journeys converge after at least 25 mixed selection, navigation, hacking, replay, rejection, sound, and reconnect operations with zero private fields exposed.
- **SC-010**: Slow-subscriber, replay, authority, request-limit, cancellation, reconnect, and concurrency gates retain every feature-005 invariant with zero reintroduced WebSocket request, route, constructor, or compatibility behavior.
- **SC-011**: Two consecutive clean Wails-binding and protobuf generations are byte-stable and leave zero unexplained tracked diff.
- **SC-012**: Two consecutive clean builds driven by the Wails v3 Taskfile/configuration complete from the repository root, build both Vite applications and embedded resources, and require zero manually started frontend, player listener, development server, or network-time `latest` resolution; source and CI scans find zero `@latest` token in active commands.
- **SC-013**: The existing Go, race, frontend, player, Playwright, Buf formatting, lint, generation-drift, breaking, and relevant platform suites pass with zero deliberately weakened assertion.
- **SC-014**: One repository-root `go tool -modfile=tools/wails/go.mod wails3 dev` launch starts the complete development system and a handled stop leaves zero owned player listeners, ngrok processes, or temporary credential-policy artifacts after the shutdown timeout.
- **SC-015**: The packaged `.app` exists at `build/bin/Fallout Terminal.app` unless one approved, consistently propagated path change is recorded, reports arm64 architecture, passes the selected signature and integrity checks, and launches offline on macOS 13+ with zero installed Go, Node, npm, Vite, Wails, or development-server dependency.
- **SC-016**: Package inspection finds every required master asset, generated binding, generated player asset, font, sound, and bundled demonstration resource, with zero required CDN or network-time asset.
- **SC-017**: Packaged smoke tests observe exactly one player listener during the application lifetime and zero listener or owned ngrok process after normal quit and handled partial startup.
- **SC-018**: The feature-006 Wails v3 migration rollback record identifies exactly one canonical Wails v2 source commit and, if a baseline executable is accepted, exactly one matching executable SHA-256 and acceptance status with zero conflicting digest, while the Electron rollback document remains unchanged historical evidence.
- **SC-019**: Before cutover, the required 60-minute local master/player soak satisfies its workload, median-RSS, convergence, listener, and cleanup thresholds, and the authenticated-ngrok soak either satisfies its complete 30-minute public workload or is explicitly `NOT RUN`, never a synthetic pass.
- **SC-020**: Final source, dependency, binding, configuration, command, bundle, test, and active-documentation scans find zero active `github.com/wailsapp/wails/v2` import, v2 CLI installation, v2 `wails` command where `wails3` is required, v2 `frontend/wailsjs` dependency, v2 runtime global, obsolete v2 configuration, or permanent dual-runtime code.
- **SC-021**: Final public-listener inspection finds exactly one generated `fallout.terminal.player.v1.PlayerService` surface and zero Wails, private desktop, legacy WebSocket, health, reflection, or generic capability-discovery exposure.
- **SC-022**: Any candidate exhibiting session corruption, bridge drift, master or player parity loss, lifecycle leakage, public-access regression, missing package assets, an unhandled beta-runtime crash, or package/signature failure is recorded as failed and does not replace the Wails v2 fallback.

## Assumptions

- The working constitution version 3.2.0 is the governing current constitution for feature 006.
- The baseline commit supplied by the owner and present at `HEAD`, local `main`, and `origin/main` is the accepted pre-migration Wails v2 source rollback.
- Planning will research, verify together, and record one mutually compatible exact Wails v3 beta set for the Go module, CLI, frontend runtime, and plugin; selecting that set is a compatibility research result, not an unresolved decision to wait for general availability.
- No application-owned protobuf schema change is expected because the migration changes framework binding metadata rather than application semantics.
- The existing 25-operation private bridge inventory remains authoritative; `CopyDemo` remains bound and private without adding a new UI control, while lifecycle methods are removed from the generated desktop surface.
- Personal-use local/ad-hoc-signed packaging is the default acceptance profile. Public-release work is conditional on an explicit selection and available credentials.
- External authenticated-ngrok credentials and connectivity may be unavailable during validation; unavailable runs are evidence gaps marked `NOT RUN`, not passing results.
- Completed specifications 001–005 remain historical records of their accepted targets and are not rewritten to claim they originally targeted Wails v3.

## Verbatim Constraints

- Feature title: `Wails v3 Runtime Migration`.
- Feature short name: `wails-v3-migration`.
- Feature directory: `specs/006-wails-v3-migration`.
- Feature branch: `006-wails-v3-migration`.
- Baseline and source rollback commit: `f1084b3df8b5630862bdf7a0f347b599156653ef`.
- Master window title: `Fallout Terminal — Master Control`.
- Master initial dimensions: `1200×780`.
- Master minimum dimensions: `900×600`.
- Supported operating system: `macOS 13+`.
- Supported architecture: `arm64`.
- Required development command: `go tool -modfile=tools/wails/go.mod wails3 dev`.
- Required clean binding command: `go tool -modfile=tools/wails/go.mod wails3 generate bindings -clean ./...`.
- Required build command: `go tool -modfile=tools/wails/go.mod wails3 build`.
- Required package command: `go tool -modfile=tools/wails/go.mod wails3 package GOOS=darwin GOARCH=arm64`.
- Required personal-use application location: `build/bin/Fallout Terminal.app`.
- Wails v3 Go module: `github.com/wailsapp/wails/v3`.
- Wails v3 frontend runtime: `@wailsio/runtime`.
- Prohibited floating version token in reproducible commands: `latest`.
- Wails v2 Go import to remove from active source: `github.com/wailsapp/wails/v2`.
- Runtime-neutral master facade: `frontend/src/desktop-api.js`.
- Accepted private operation names: `GetRuntimeStatus`, `NewSession`, `OpenSession`, `CopyDemo`, `SaveSession`, `LoadReferencedPlayerConfig`, `NewPlayerConfig`, `OpenPlayerConfig`, `RequestTerminalActivation`, `UpdateLiveTerminal`, `RequestTerminalClear`, `ResolveTerminalSwitch`, `ForceHackSuccess`, `ResetFailedHack`, `AddCharacter`, `RenameCharacter`, `DeleteCharacter`, `RenameLogicalSession`, `AssignCharacter`, `ReleaseCharacter`, `MoveCharacter`, `SetActiveController`, `StartBroadcast`, `EndBroadcast`, `OpenURL`.
- Prohibited generated desktop lifecycle methods: `Start`, `Shutdown`.
- Current Wails event names: `server-info`, `client-count`, `hack-state`, `coordination-state`.
- Public player service: `fallout.terminal.player.v1.PlayerService`.
- Public player procedures: `Subscribe`, `SelectCharacter`, `Navigate`, `Guess`, `ActivatePattern`, `SoundManifest`.
- Public player port: `3690`.
- Session Documents path: `~/Documents/Fallout Terminal/Sessions/`.
- Application Support path: `~/Library/Application Support/com.vaulttec.fallout-terminal/`.
- Conditional evidence status: `NOT RUN`.
