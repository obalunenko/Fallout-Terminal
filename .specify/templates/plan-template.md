# Implementation Plan: [FEATURE]

**Branch**: `[###-feature-name]` | **Date**: [DATE] | **Spec**: [link]

**Input**: Feature specification from `/specs/[###-feature-name]/spec.md`

**Note**: This template is filled in by the `$speckit-plan` command.

## Summary

[Extract the primary requirement and technical approach from the feature spec]

## Technical Context

**Language/Version**: JavaScript on Node.js/Electron 28

**Primary Dependencies**: Electron, Express 4, `ws` 8; electron-builder for packaging

**Storage**: Versioned local JSON session files; ephemeral live state in server memory

**Testing**: No automated framework or coverage threshold currently configured; define feature-specific automated and manual verification

**Target Platform**: Electron desktop application packaged for Windows x64, with modern browser clients on the local network or authenticated ngrok endpoint

**Project Type**: Modular desktop monolith with embedded HTTP/WebSocket server

**Performance Goals**: [Feature-specific responsiveness, synchronization, startup, or client-count goal]

**Constraints**: Preserve renderer sandbox/context isolation, server-authoritative shared state, session compatibility, and single-package npm structure

**Scale/Scope**: One GM desktop process, one active broadcast state, and [expected connected player-client count or NEEDS CLARIFICATION]

## Constitution Check

*GATE: Must pass before Phase 0 research and be re-checked after Phase 1 design.*

- [ ] Runtime ownership remains within `main.js`/`preload.js`, `master/`, `server/`, `client/`, and `sessions/` boundaries.
- [ ] Cross-boundary IPC, HTTP, and WebSocket contracts are documented with validation and failure behavior.
- [ ] Shared navigation/hacking behavior remains server-authoritative and reconnect-safe.
- [ ] Electron isolation, CSP, external URL handling, and ngrok credential protections are preserved where applicable.
- [ ] Session schema changes define versioning, defaults, migration, and backward compatibility.
- [ ] New dependencies or structural changes have a concrete, documented need.
- [ ] Verification is proportionate and does not claim absent lint, coverage, test, or CI gates.
- [ ] Naming and code style match the conventions of the affected files.

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── spec.md
├── plan.md
├── research.md          # Include only when research decisions are needed
├── data-model.md        # Include for session/state model changes
├── quickstart.md        # Feature verification instructions
├── contracts/           # Include for IPC/HTTP/WebSocket/session contracts
└── tasks.md
```

### Source Code (repository root)

```text
main.js                  # Electron lifecycle, native APIs, persistence, orchestration
preload.js               # Narrow contextBridge API for the master renderer
master/
├── index.html
├── master.js            # GM session editor and broadcast controls
└── master.css
server/
├── server.js            # Express, WebSocket transport, canonical live state
├── hack.js              # Hacking domain logic
├── nav.js               # Navigation domain logic
├── wordbank.js          # Hacking word data/selection
└── ngrok.js             # Optional authenticated tunnel integration
client/
├── index.html
├── client.js            # Player state/rendering and WebSocket client
├── sound.js             # Browser audio behavior
├── client.css
└── sounds/
sessions/
└── demo.json            # Versioned example session
package.json             # npm scripts, dependencies, electron-builder configuration
```

**Structure Decision**: [List affected paths and explain why the feature belongs in each]

## Contract and State Design

### Session JSON

[Document changed fields, versioning, validation, defaults, migration, and sample updates, or N/A]

### IPC

[Document preload API and main/renderer channels, directions, payloads, validation, and errors, or N/A]

### HTTP and WebSocket

[Document routes/message types, directions, payloads, server validation, broadcasts, and reconnect state, or N/A]

### Live-State Lifecycle

[Describe creation, mutation, clearing, client synchronization, and persistence boundary, or N/A]

## Implementation Phases

### Phase 0: Research and Decisions

- [Resolve actual unknowns; omit generic research]
- [Choose test tooling only if the feature introduces automated tests]
- [Confirm platform, protocol, or compatibility decisions]

### Phase 1: Contracts and Data Design

- [Define session/IPC/WebSocket/HTTP contracts as applicable]
- [Define compatibility and reconnection behavior]
- [Update Constitution Check after design]

### Phase 2: Server and Desktop Foundations

- [Changes to `server/`, `main.js`, or `preload.js` required before UI work]

### Phase 3: Master and Player Experiences

- [Vertical user-story slices across `master/` and `client/`]

### Phase 4: Integration and Packaging

- [End-to-end synchronization, session, security, ngrok, and packaging checks]

## Verification Plan

| Surface | Automated check | Manual check | Expected result |
|---|---|---|---|
| Domain/session logic | [Command or not configured] | [Scenario] | [Result] |
| Electron master | [Command or not configured] | `npm start` + [journey] | [Result] |
| Player browser(s) | [Command or not configured] | [multi-client/reconnect journey] | [Result] |
| Packaging | `npm run build:dir` when applicable/available | [packaged smoke test] | [Result] |

## Complexity Tracking

> Fill only when a Constitution Check violation requires justification.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|---|---|---|
| [Constitution rule] | [Concrete need] | [Why a compliant approach is insufficient] |
