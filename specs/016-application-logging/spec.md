# Feature Specification: Application Logging

**Feature Branch**: `016-application-logging`  
**Created**: 2026-08-21  
**Status**: Draft  
**Input**: Add operational logs to the application using the required Go logger.

## User Scenarios & Testing

### User Story 1 - Diagnose Application Lifecycle (Priority: P1)

As an operator or maintainer, I can observe the application's startup, readiness, and shutdown progress so that failures can be diagnosed without reproducing them under a debugger.

**Why this priority**: Lifecycle failures can prevent the application or player service from becoming usable, so they need the clearest operational evidence.

**Independent Test**: Start and stop the application with successful and failing lifecycle dependencies, then verify that the emitted records identify each completed stage and any absorbed failure without changing the visible runtime status.

**Acceptance Scenarios**:

1. **Given** all required runtime dependencies are available, **When** the application starts, **Then** the records show startup beginning, the local player service becoming available, and the application reaching readiness.
2. **Given** a startup dependency fails, **When** the host keeps the application window available to present the failure, **Then** an error record captures the failed operation while the existing user-visible failure status remains available.
3. **Given** the application is running, **When** shutdown completes, **Then** the records show shutdown beginning and completing after owned resources are released.
4. **Given** the player listener stops unexpectedly, **When** the background serving operation exits, **Then** an error record identifies the unexpected listener failure.

---

### User Story 2 - Trace Important Operator Actions (Priority: P2)

As a maintainer supporting an Overseer, I can trace important persistence, broadcast, player-configuration, and public-access outcomes so that operational problems can be correlated with the action that triggered them.

**Why this priority**: These actions are important once the runtime is available, but they do not block initial startup diagnostics.

**Independent Test**: Exercise successful, cancelled, and failed commands at the trusted application boundary and verify that concise records describe the operation and outcome without recording content or credentials.

**Acceptance Scenarios**:

1. **Given** a session or player configuration operation completes, **When** the application returns its existing result, **Then** a record identifies the operation and whether it succeeded, failed, or was cancelled.
2. **Given** a broadcast starts or ends, **When** the command completes, **Then** a record identifies the lifecycle transition and outcome.
3. **Given** public access starts, stops, or changes configuration, **When** its state changes, **Then** a record identifies the safe lifecycle state and outcome without including credentials or credential-derived values.
4. **Given** an event cannot be delivered to the Overseer interface, **When** the application intentionally continues, **Then** a warning or error record identifies the event name and delivery failure.

---

### User Story 3 - Keep Diagnostics Safe and Stable (Priority: P3)

As an Overseer, I can use the application normally with logging enabled and know that private gameplay content and public-access secrets are not exposed through diagnostics.

**Why this priority**: Diagnostic value is acceptable only when existing behavior and secret boundaries remain intact.

**Independent Test**: Run automated logging checks with distinctive secret and content markers, then confirm that expected operational records are present, forbidden values are absent, and existing application tests still pass.

**Acceptance Scenarios**:

1. **Given** commands contain session content, player identities, provider credentials, or generated passwords, **When** records are emitted, **Then** none of those raw values appear in the records.
2. **Given** the application runs normally, **When** logging is enabled, **Then** command results, lifecycle ordering, shutdown behavior, and user-visible status remain unchanged.
3. **Given** a record includes operational context, **When** it is inspected, **Then** stable field names and appropriate severity make the operation and outcome understandable without parsing message text.

## Edge Cases

- Repeated startup or shutdown calls must not create misleading duplicate completion records.
- User-cancelled native dialogs must be distinguishable from failures without being reported as errors.
- Expected player-server closure during normal shutdown must not be reported as an unexpected serving failure.
- Event-delivery failures must be recorded even though the application intentionally continues.
- Public-access failures must expose only existing safe categories and messages, never raw provider or secure-store errors that may contain sensitive material.
- Logging must remain available before the desktop host supplies its lifecycle context and during late shutdown after that context is cleared.

## Requirements

### Functional Requirements

- **FR-001**: The application MUST emit structured records for startup beginning, local player availability, application readiness, shutdown beginning, and successful shutdown completion.
- **FR-002**: The application MUST emit an error record for a startup failure that the desktop host absorbs so the existing user-visible failure status can remain available.
- **FR-003**: The player service MUST emit an error record when its background serving operation exits unexpectedly, while normal server closure MUST remain silent.
- **FR-004**: The application MUST emit outcome records for session creation, session opening, demo copying, session saving, player-configuration loading, broadcast lifecycle commands, and public-access lifecycle or configuration commands.
- **FR-005**: User-cancelled operations MUST be recorded as cancellations rather than failures, and expected validation rejections MUST not be escalated to fatal process errors.
- **FR-006**: Event-delivery errors that are intentionally not returned to callers MUST be recorded with the affected event name.
- **FR-007**: Records MUST use severity and structured fields to identify the operation, outcome, safe lifecycle state, revision, count, or error as applicable.
- **FR-008**: Records MUST NOT contain provider tokens, player passwords, generated passwords, session content, terminal content, character names, or unredacted secret-store and provider error details.
- **FR-009**: Logging MUST be initialized exactly once for the production application and MUST remain available throughout startup and shutdown.
- **FR-010**: Logging MUST NOT change existing command results, runtime status, lifecycle ordering, resource cleanup, public/private capability boundaries, or persistence behavior.
- **FR-011**: Automated tests MUST verify representative lifecycle, command-outcome, unexpected-serving, swallowed-event-error, and secret-redaction records.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Automated lifecycle tests observe every required startup and shutdown milestone and the absorbed-startup-failure record in the correct operational path.
- **SC-002**: Automated command tests observe success, cancellation, and failure outcomes for each required action category without changing the returned result.
- **SC-003**: A diagnostic capture containing distinctive credential and gameplay markers contains zero forbidden raw values.
- **SC-004**: Normal player-server shutdown produces zero unexpected-serving error records, while an injected serving failure produces exactly one.
- **SC-005**: All applicable Go formatting, static analysis, unit, and race checks pass after logging is enabled.

## Assumptions

- Production records default to human-readable informational output on the process error stream; runtime log-file persistence, rotation, and a settings UI are outside this feature.
- The logging scope is the production application and its runtime services. Repository build tooling and browser-test fixture processes retain their existing command-line output behavior.
- Operational fields use identifiers and counts only when they are safe; user-authored content, names, file contents, and credentials are excluded.
- Existing safe public-access categories and messages may be recorded, but underlying provider and secure-store errors remain redacted.

## Verbatim Constraints

- Go application logging MUST use `github.com/obalunenko/logger`.
