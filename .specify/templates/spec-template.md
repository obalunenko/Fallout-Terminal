# Feature Specification: [FEATURE NAME]

**Feature Branch**: `[###-feature-name]`

**Created**: [DATE]

**Status**: Draft

**Input**: User description: "$ARGUMENTS"

## User Scenarios & Testing *(mandatory)*

<!--
Prioritize stories as P1, P2, P3, and keep each story independently usable and
verifiable. Include both the game-master experience and player experience when
the feature crosses the Electron/WebSocket boundary.
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

- What happens when no session or live terminal exists?
- What happens when a browser connects, disconnects, or reconnects mid-action?
- How do multiple connected players remain synchronized?
- How are malformed, stale, or unexpected IPC/WebSocket inputs handled?
- What happens when session data is missing a new field or uses an older version?
- What happens when filesystem, server startup, packaging, or ngrok operations fail?
- [Feature-specific boundary or error case]

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

- **Electron main (`main.js`)**: [Affected/not affected — native lifecycle, files, dialogs, server startup]
- **Preload IPC (`preload.js`)**: [Affected/not affected — privileged bridge contract]
- **Master UI (`master/`)**: [Affected/not affected — GM editing/control workflow]
- **Server (`server/`)**: [Affected/not affected — HTTP, WebSocket, shared live state, domain logic]
- **Player UI (`client/`)**: [Affected/not affected — browser behavior and audio/visual state]
- **Session data (`sessions/`)**: [Affected/not affected — persistent JSON shape or examples]
- **Packaging/public access**: [Affected/not affected — electron-builder or ngrok behavior]

### State and Contract Requirements *(include when applicable)*

- **Session compatibility**: [Version/default/migration behavior, or N/A]
- **IPC contract**: [Channel, direction, payload, validation, error behavior, or N/A]
- **WebSocket contract**: [Message type, direction, payload, server validation, broadcast behavior, or N/A]
- **Reconnect behavior**: [State a newly connected/reconnected client receives, or N/A]
- **HTTP/static contract**: [Route/request/response behavior, or N/A]

### Security and Privacy Requirements *(include when applicable)*

- [Electron sandbox/context-isolation/CSP implications]
- [External URL, filesystem path, or untrusted payload validation]
- [ngrok authentication, credentials, and temporary-file handling]
- [Data that MUST remain server-side or local]

### Key Entities *(include if feature involves data)*

- **Session**: [Version, name, terminals, and any changed semantics]
- **Terminal**: [Identity, settings, content tree, and any changed semantics]
- **Live state**: [Ephemeral server state and lifecycle, if affected]
- **[Additional entity]**: [Meaning and relationships]

## Success Criteria *(mandatory)*

<!-- Keep outcomes measurable and observable without prescribing implementation. -->

### Measurable Outcomes

- **SC-001**: [Primary journey completes with a concrete observable result]
- **SC-002**: [All connected clients converge on the expected state under defined conditions]
- **SC-003**: [Existing compatible sessions continue to open/save correctly, if applicable]
- **SC-004**: [Failure/security/performance outcome relevant to the feature]

## Assumptions

- [Assumption about GM/player environment or local network]
- [Assumption about compatible session versions]
- [Assumption about connected-client count or supported platform]
- [Dependency on existing Electron, server, browser, or ngrok behavior]

## Out of Scope

- [Explicitly excluded related behavior]
