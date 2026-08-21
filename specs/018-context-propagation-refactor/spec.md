# Feature Specification: Context Propagation and Causal Cancellation

**Feature Branch**: `develop`
**Created**: 2026-08-21
**Status**: Draft
**Input**: Refactor context handling so operations receive the application or request context instead of an absent context, owned contexts carry an explicit cancellation cause, and tests derive contexts from the active test.

## User Scenarios & Testing

### User Story 1 - Preserve operation lifetime end to end (Priority: P1)

As an operator, I need startup, desktop actions, player requests, persistence, and public-access work to remain attached to the lifetime that initiated them so cancellation and request-scoped values are not silently lost.

**Why this priority**: Detached work can continue after its owner has stopped and can lose logging, tracing, or cancellation values needed to diagnose failures.

**Independent Test**: Start the application with a context carrying a marker, exercise representative application and request flows, and verify the same marker reaches their context-aware dependencies without an absent or newly detached context being substituted.

**Acceptance Scenarios**:

1. **Given** the application is composed from its process context, **When** startup and application-owned work acquire resources, **Then** context-aware dependencies observe a context derived from that process or lifecycle context.
2. **Given** a player or desktop operation has an available request/application context, **When** it reaches persistence, networking, or platform boundaries, **Then** the operation uses that context throughout the call chain.
3. **Given** a context-aware API receives no context, **When** the API begins work, **Then** it fails explicitly instead of silently replacing the missing context with an unrelated root.

---

### User Story 2 - Explain why owned work stopped (Priority: P1)

As a maintainer, I need every explicitly canceled application-owned context to retain a meaningful cause so shutdown, replacement, timeout, overflow, and failed-start paths can be distinguished during diagnosis.

**Why this priority**: A bare canceled signal reveals that work ended but not the owner action or failure that ended it.

**Independent Test**: Close each representative owned lifecycle and assert that its context finishes with the expected non-empty semantic cause while parent cancellation continues to preserve the parent's cause.

**Acceptance Scenarios**:

1. **Given** an application-owned resource is running, **When** its owner closes or replaces it, **Then** the resource context is canceled with the owner's explicit reason.
2. **Given** an owned operation ends because its parent is canceled, **When** the child observes cancellation, **Then** the parent cancellation cause remains observable.
3. **Given** cleanup must continue after an initiating request is already canceled, **When** cleanup starts, **Then** it preserves the initiating context's values while applying its own bounded lifetime and cleanup reason.

---

### User Story 3 - Keep test lifetimes isolated (Priority: P2)

As a contributor, I need context-sensitive tests to be rooted in the running test so leaked work is canceled automatically and no test accidentally outlives its owner.

**Why this priority**: Test-owned contexts make concurrency failures deterministic and keep the repository's testing convention enforceable.

**Independent Test**: Scan and run affected Go tests, confirming every test context is the active test context or a direct derivative and that cancellation-cause assertions pass.

**Acceptance Scenarios**:

1. **Given** a Go test needs a context, **When** it constructs the subject or an operation context, **Then** the root is the active test context.
2. **Given** a test needs explicit cancellation or a deadline, **When** it derives that context, **Then** the derived context remains owned by the active test.

## Edge Cases

- A caller passes an absent context to a public context-aware boundary.
- A parent is already canceled before a child lifetime is created.
- Explicit close races with parent cancellation, timeout, queue overflow, or provider failure.
- Shutdown cleanup starts after the normal application/request context has already been canceled.
- Repeated close and shutdown calls must remain idempotent and must not overwrite the first meaningful cancellation cause.
- A partial startup failure must cancel acquired resource lifetimes with the startup failure while still allowing bounded cleanup.

## Requirements

### Functional Requirements

- **FR-001**: The application MUST propagate a non-absent process or lifecycle context from composition through every application-owned context-aware dependency.
- **FR-002**: Request-triggered context-aware work MUST use the initiating request or application context through persistence, networking, and platform boundaries.
- **FR-003**: Context-aware production APIs MUST reject an absent context rather than silently substitute an unrelated background context below an executable entry point.
- **FR-004**: Every explicitly canceled application-owned context MUST be created with causal cancellation and canceled with a stable, non-empty semantic reason.
- **FR-005**: Cleanup that must survive an already-canceled initiating context MUST preserve that context's values, apply the existing bounded cleanup duration, and expose the cleanup termination reason.
- **FR-006**: Repeated or concurrent close paths MUST remain safe and MUST preserve the first applicable cancellation cause.
- **FR-007**: Every affected Go test that needs a context MUST use the active test context as its root and derive values, cancellation, or deadlines from it.
- **FR-008**: The refactor MUST preserve existing user-visible behavior, persistence formats, network contracts, public/private boundaries, and configured timeout budgets.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A repository scan finds zero production calls that pass an absent value to an argument whose contract is a context, and zero non-entry-point fallbacks that create an unrelated background context.
- **SC-002**: All explicitly canceled owned lifetimes covered by the refactor expose a non-empty expected cancellation cause in automated tests.
- **SC-003**: A repository scan finds zero affected tests using background or placeholder roots for test-owned context work.
- **SC-004**: Formatting, static analysis, the complete Go test suite, and the race-enabled Go test suite all pass.
- **SC-005**: Existing application startup, shutdown, player-stream, session, and public-access tests continue to pass without contract or persistence drift.

## Assumptions

- Executable entry points may create the initial process context; lower layers receive that root or a request-derived context.
- Cleanup may detach cancellation from an already-canceled parent only to finish bounded resource release; parent values remain available.
- Stable internal error values are sufficient cancellation causes; no public wire or persistence contract needs to change.
- The change is a cross-cutting internal lifecycle refactor and does not add user-facing behavior or dependencies.

## Verbatim Constraints

- `context.WithCancelCause`
- `testing.T.Context`
