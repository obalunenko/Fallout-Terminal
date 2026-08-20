# Research: Immersive Controller Status Log

## Decision 1: Reuse the authoritative player projection

**Decision**: Format the lower status line entirely from the existing session fallback name, assigned character, and role already delivered to the player client.

**Rationale**: The server already owns all identity and controller-authority state. A presentation-only derivation preserves convergence and avoids unnecessary contract or persistence work.

**Alternatives considered**:

- Add a server-composed display string: rejected because it would turn view copy into a network contract and duplicate client presentation concerns.
- Track controller ownership locally: rejected because it could diverge from authoritative state during reassignment or reconnect.

## Decision 2: Move the existing semantic status surface

**Decision**: Relocate and reshape the existing identity section, preserving `playerIdentity`, `playerFallbackName`, `playerCharacterName`, and `roleBadge` as stable identifiers.

**Rationale**: Existing browser journeys and assistive semantics already observe these elements. Reusing them gives a clean cutover without a second rendering path or duplicated announcements.

**Alternatives considered**:

- Create a new status component and keep the old one hidden: rejected because it leaves duplicate semantics and a superseded UI path.
- Render a CSS pseudo-element only: rejected because dynamic names and role changes must remain accessible and testable as real text.

## Decision 3: Keep the line persistent while session state is available

**Decision**: Treat the selected concept as a persistent low-priority system log line rather than a timed toast.

**Rationale**: Controller ownership affects whether player actions are permitted. Persistent role visibility preserves the current usability and accessibility contract while the quieter placement restores immersion.

**Alternatives considered**:

- Hide the line after a short timeout: rejected because observers and newly reassigned controllers could lose the only visible authority indicator.
- Show it only when authority changes: rejected because a freshly opened or reconnected client would lack durable context.

## Decision 4: Compact only conventional default player labels

**Decision**: Convert a default `PLAYER N` fallback label to `PN` for the input-channel segment and leave custom fallback labels recognisable.

**Rationale**: This produces the approved `P1` copy without hard-coding one player or destroying meaningful custom session names.

**Alternatives considered**:

- Always display `P1`: rejected because multiple and renamed sessions require distinct identities.
- Display the full `PLAYER 1`: rejected because it does not match the approved concept and is visually heavier.

## Decision 5: Verify semantics and geometry through existing browser journeys

**Decision**: Add exact-copy and bounding-box assertions to the existing Playwright coverage and update all intentional active-role copy assertions.

**Rationale**: The risk is visual placement plus role-state regression. Browser journeys can observe live server projections, accessibility-facing text, and actual layout geometry without introducing a new test stack.

**Alternatives considered**:

- Rely only on a production build: rejected because compilation cannot detect incorrect copy or overlap.
- Add bitmap snapshot baselines: rejected because the repository does not currently govern visual-regression assets and geometry assertions are sufficient for this focused layout change.
