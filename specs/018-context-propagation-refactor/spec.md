# Feature Specification: Context Propagation and Causal Cancellation

**Feature Branch**: `develop`
**Created**: 2026-08-21
**Status**: Draft
**Input**: Refactor context handling so operations receive the application or request context instead of an absent context, owned contexts carry an explicit cancellation cause, and tests derive contexts from the active test.

**Bugfix**: 2026-08-21 — [BUG-001] Clarified that successful player-listener acquisition hands ownership to the application lifetime and added post-start streaming acceptance.

**Bugfix**: 2026-08-21 — [BUG-002] Added first-party player console and static-request cleanliness requirements while preserving HTTP-header anti-framing enforcement.

## User Scenarios & Testing

### User Story 1 - Preserve operation lifetime end to end (Priority: P1)

As an operator, I need startup, desktop actions, player requests, persistence, and public-access work to remain attached to the lifetime that initiated them so cancellation and request-scoped values are not silently lost.

**Why this priority**: Detached work can continue after its owner has stopped and can lose logging, tracing, or cancellation values needed to diagnose failures.

**Independent Test**: Start the application with a context carrying a marker, exercise representative application and request flows, and verify the same marker reaches their context-aware dependencies without an absent or newly detached context being substituted.

**Acceptance Scenarios**:

1. **Given** the application is composed from its process context, **When** startup and application-owned work acquire resources, **Then** context-aware dependencies observe a context derived from that process or lifecycle context.
2. **Given** a player or desktop operation has an available request/application context, **When** it reaches persistence, networking, or platform boundaries, **Then** the operation uses that context throughout the call chain.
3. **Given** a context-aware API receives no context, **When** the API begins work, **Then** it fails explicitly instead of silently replacing the missing context with an unrelated root.
4. **Given** the player listener is acquired under a bounded startup operation, **When** startup commits the listener and the operation context completes, **Then** the committed HTTP server and later player request contexts remain active under the application-owned lifetime until explicit server stop, serve failure, parent application cancellation, or application shutdown.
5. **Given** the committed player server is active, **When** its first-party page loads in a clean extension-free browser, **Then** application-owned documents and resources produce no console warning or error and no failed same-origin static-resource request while anti-framing remains enforced by the HTTP response policy.

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
- A bounded startup operation completes successfully after handing off an acquired listener while the application runtime remains active.
- Browser or extension-injected diagnostics that do not originate from repository-owned player assets must be separated from first-party console and request failures.

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
- **FR-009**: The first-party player page MUST load in a clean extension-free browser without application-owned console warnings or errors and without failed same-origin static-resource requests; unsupported meta-delivered CSP directives MUST be omitted, anti-framing MUST remain enforced by the HTTP `Content-Security-Policy` header, and the document MUST declare a favicon strategy that prevents an automatic missing-resource request.

**BUG-001 clarification (FR-001, FR-004, FR-008)**: A bounded startup context owns player-listener acquisition only until commit. Successful startup completion MUST NOT cancel the committed player-server lifetime or any HTTP/ConnectRPC request derived after the handoff; only the documented application/player-server owner outcomes may do so.

**BUG-002 clarification (FR-008, FR-009)**: `frame-ancestors 'none'` remains mandatory in the player HTTP response header but MUST NOT appear in the HTML meta CSP, where browsers ignore it. Diagnostics from scripts absent from repository-owned player assets are out of scope unless they reproduce in the governed extension-free browser journey.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A repository scan finds zero production calls that pass an absent value to an argument whose contract is a context, and zero non-entry-point fallbacks that create an unrelated background context.
- **SC-002**: All explicitly canceled owned lifetimes covered by the refactor expose a non-empty expected cancellation cause in automated tests.
- **SC-003**: A repository scan finds zero affected tests using background or placeholder roots for test-owned context work.
- **SC-004**: Formatting, static analysis, the complete Go test suite, and the race-enabled Go test suite all pass.
- **SC-005**: Existing application startup, shutdown, player-stream, session, and public-access tests continue to pass without contract or persistence drift.
- **SC-006**: With the production non-zero startup timeout, successful `App.Start` completion leaves the committed player-server context active, and a later `Subscribe` receives one complete snapshot plus a subsequent authoritative update before explicit shutdown supplies the documented cancellation cause.
- **SC-007**: In the governed extension-free browser journey, one initial player-page load records zero application-owned console warnings/errors, zero failed same-origin static-resource responses, no `/favicon.ico` 404, and an HTTP CSP response header containing `frame-ancestors 'none'` while the HTML meta policy omits that directive.

## Assumptions

- Executable entry points may create the initial process context; lower layers receive that root or a request-derived context.
- Cleanup may detach cancellation from an already-canceled parent only to finish bounded resource release; parent values remain available.
- Stable internal error values are sufficient cancellation causes; no public wire or persistence contract needs to change.
- The change is a cross-cutting internal lifecycle refactor and does not add user-facing behavior or dependencies.
- Browser-extension content scripts are not part of the shipped player application; only diagnostics reproducible in the governed extension-free browser context are first-party acceptance failures.

## Verbatim Constraints

- `context.WithCancelCause`
- `testing.T.Context`
