# Feature Specification: [FEATURE NAME]

**Feature Branch**: `[###-feature-name]`

**Created**: [DATE]

**Status**: Draft

**Input**: User description: "$ARGUMENTS"

## User Scenarios & Testing *(mandatory)*

<!--
Prioritize stories as P1, P2, P3, and keep each story independently usable and
verifiable. Include both the game-master and player experience when the feature
crosses the Wails/Go, HTTP, or WebSocket boundary.
-->

### User Story 1 - [Brief Title] (Priority: P1)

[Describe the game-master or player journey in plain language]

**Why this priority**: [Explain its value and priority]

**Independent Test**: [Describe how this story can be verified by itself]

**Acceptance Scenarios**:

1. **Given** [initial state], **When** [action], **Then** [observable result]
2. **Given** [initial state], **When** [action], **Then** [observable result]

---

### User Story 2 - [Brief Title] (Priority: P2)

[Describe this user journey]

**Why this priority**: [Explain its value and priority]

**Independent Test**: [Describe how this story can be verified by itself]

**Acceptance Scenarios**:

1. **Given** [initial state], **When** [action], **Then** [observable result]

---

### User Story 3 - [Brief Title] (Priority: P3)

[Describe this user journey]

**Why this priority**: [Explain its value and priority]

**Independent Test**: [Describe how this story can be verified by itself]

**Acceptance Scenarios**:

1. **Given** [initial state], **When** [action], **Then** [observable result]

[Add or remove stories as needed]

### Edge Cases

- What happens when no session, player configuration, broadcast, or live terminal exists?
- What happens when a browser connects, opens another tab, disconnects, or reconnects mid-action?
- How do multiple connected players remain synchronized and respect controller authority?
- How are malformed, stale, duplicate, oversized, or unexpected Wails/WebSocket inputs handled?
- What happens when persistent JSON is missing a new field, references another file, or uses an older version?
- What happens when filesystem, player-server startup, packaging, or an optional public provider fails?
- [Feature-specific boundary, concurrency, or error case]

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST [specific observable capability]
- **FR-002**: System MUST [validation or failure behavior]
- **FR-003**: System MUST [persistence or synchronization behavior]
- **FR-004**: Users MUST be able to [key interaction]

Mark material ambiguity inline, for example:

- **FR-005**: System MUST retain [NEEDS CLARIFICATION: duration or lifecycle not specified]

### Impacted Application Surfaces *(mandatory)*

<!-- Mark each as affected or not affected and explain why. -->

- **Composition and Wails bridge (`main.go`, `app.go`)**: [Affected/not affected — lifecycle, native operations, bound commands, or runtime events]
- **Domain and canonical state (`internal/domain/`, `internal/nav/`, `internal/hack/`, `internal/live/`, `internal/control/`)**: [Affected/not affected — models, validation, state transitions, coordination, or public projections]
- **Persistence (`internal/session/`, `internal/playerconfig/`, `sessions/`)**: [Affected/not affected — JSON shape, validation, file references, storage behavior, or examples]
- **Player transport (`internal/player/`)**: [Affected/not affected — HTTP routes, WebSocket protocol, validation, broadcast, or reconnect behavior]
- **Platform and public access (`internal/platform/`, `internal/tunnel/`)**: [Affected/not affected — macOS paths, desktop operations, embedded endpoint ownership, Keychain secrets, or provider behavior]
- **Master UI (`frontend/src/`)**: [Affected/not affected — game-master editing/control workflow or Wails bridge consumer]
- **Player UI (`client/`)**: [Affected/not affected — browser behavior, presentation, audio, or WebSocket consumer]
- **Tests and fixtures (`internal/**/*_test.go`, `tests/browser/`, `internal/testutil/`)**: [Affected/not affected — automated coverage or fixtures]
- **Build and packaging (`go.mod`, `frontend/`, `wails.json`, `build/`, `scripts/`)**: [Affected/not affected — dependencies, embedding, macOS build, signing, or release behavior]

### State and Contract Requirements *(include when applicable)*

- **Session/player-config compatibility**: [Version, default, reference, migration, and backward-compatibility behavior, or N/A]
- **Wails bridge and event contract**: [Bound method/event, direction, payload, validation, error behavior, and public projection, or N/A]
- **WebSocket contract**: [Message type, direction, payload, server validation, ordering/revision, broadcast behavior, and rejection result, or N/A]
- **Reconnect and multi-tab behavior**: [Authoritative state a new/reconnected tab receives and identity rules, or N/A]
- **HTTP/static contract**: [Route, method, request/response behavior, origin policy, or N/A]
- **Runtime-state lifecycle**: [Creation, mutation, publication, clearing, shutdown, and persistence boundary, or N/A]

### Security and Privacy Requirements *(include when applicable)*

- [Wails method exposure, CSP, external URL, filesystem path, or untrusted payload validation]
- [WebSocket origin, input-size, authorization, or public-projection implications]
- [Public-endpoint authentication, Keychain-backed credentials, embedded resource ownership, and ephemeral-secret handling]
- [Data that MUST remain server-side, process-local, or user-controlled]

### Verification Requirements *(mandatory)*

- **Go tests**: [Affected packages and observable behavior, or N/A]
- **Race testing**: [Affected concurrent services and required command, or N/A]
- **Browser tests**: [Playwright journey under `tests/browser/`, or N/A]
- **Interactive verification**: [Affected `go run ./cmd/build dev` master/player journey]
- **Packaging/release verification**: [arm64 app, signing/notarization, or N/A]

No numeric coverage threshold or repository-wide linter is currently defined.
Specify concrete behavioral checks rather than inventing either gate.

### Key Entities *(include if feature involves data)*

- **Session**: [Version, name, terminals, player-config reference, and changed semantics]
- **Player configuration**: [Stable character identities, names, and changed semantics]
- **Terminal**: [Identity, settings, content tree, and changed semantics]
- **Live/coordination state**: [Ephemeral state, authority, revisions, and lifecycle, if affected]
- **[Additional entity]**: [Meaning and relationships]

## Success Criteria *(mandatory)*

<!-- Keep outcomes measurable and observable without prescribing implementation. -->

### Measurable Outcomes

- **SC-001**: [Primary journey completes with a concrete observable result]
- **SC-002**: [All connected clients converge on the expected state under defined conditions]
- **SC-003**: [Existing compatible persistent files continue to open/save correctly, if applicable]
- **SC-004**: [Failure, security, responsiveness, or packaging outcome relevant to the feature]

## Assumptions

- [Assumption about the game-master/player environment or local network]
- [Assumption about compatible session or player-configuration versions]
- [Assumption about connected-player count or supported macOS profile]
- [Dependency on existing Wails, Go service, browser, or optional public-provider behavior]

## Out of Scope

- [Explicitly excluded related behavior]
