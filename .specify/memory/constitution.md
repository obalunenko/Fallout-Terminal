# Fallout Terminal Constitution

## Project Identity

Fallout Terminal is a JavaScript Electron desktop application for tabletop RPG
game masters. The desktop master interface edits and publishes Fallout-style
terminal content, while an embedded Express and WebSocket server synchronizes
that content with browser-based player clients. Saved campaign state is stored
as versioned JSON session files; live terminal, navigation, and hacking state is
held in memory by the server.

The repository is a modular monolith: Electron, the master renderer, the local
server, and player client are parts of one deployable application rather than
independent packages or services.

## Core Principles

### I. Preserve Runtime Boundaries

- `main.js` owns Electron lifecycle, native dialogs, filesystem-backed session
  persistence, server startup, and optional tunnel startup.
- `preload.js` is the only bridge between the sandboxed master renderer and
  Electron APIs. Renderer code MUST NOT gain direct Node.js access.
- `master/` owns the game-master interface and editing state.
- `server/` owns HTTP/WebSocket transport and server-authoritative live state.
  Domain logic for navigation and hacking remains separated in `nav.js` and
  `hack.js` rather than moving into transport or renderer code.
- `client/` owns the browser player experience and MUST operate without
  Electron or Node.js APIs.
- `sessions/` contains versioned JSON examples/data, not executable logic.

Changes crossing these boundaries MUST identify every affected surface in the
feature specification and implementation plan.

### II. Keep Shared State Server-Authoritative

Player navigation and hacking actions are requests, not local state changes.
The server MUST validate actions, mutate the canonical live state, and broadcast
the resulting state to all connected clients. Clients MUST converge on server
messages such as `TERMINAL_LIVE`, `TERMINAL_UPDATE`, `NAV_STATE`, and
`HACK_STATE`; optimistic client-only transitions that can diverge are prohibited.

Protocol changes MUST document message direction, payload shape, validation,
reconnection behavior, and compatibility impact. Secret hacking data MUST remain
server-side and MUST NOT be included in public state payloads.

### III. Protect Desktop and Public-Access Boundaries

Electron windows MUST retain `nodeIntegration: false`, `contextIsolation: true`,
and `sandbox: true`. New privileged operations MUST be exposed as narrow,
explicit preload methods and validated again in the main process. The master
page's Content Security Policy MUST remain restrictive. External URL handling
MUST allowlist supported protocols.

Public ngrok mode MUST fail closed without valid credentials. Credentials and
generated traffic-policy files MUST NOT be committed or persisted in project
data, and temporary policy material MUST be cleaned up on success, failure, and
shutdown paths.

### IV. Preserve Session Data Compatibility

Session files are user-owned persistent data. Specifications that change the
session shape MUST define versioning, validation, defaults for missing fields,
and migration or backward-compatibility behavior before implementation. Saving
MUST remain explicit about the target file and MUST NOT silently overwrite
unrelated user data. Bundled sample sessions may seed an empty writable session
directory but MUST NOT replace existing files.

Runtime-only live, navigation, connection, or hacking state MUST NOT be added to
session JSON unless the feature explicitly changes the persistence contract.

### V. Match Established Code Conventions

JavaScript files use lowercase filenames, camelCase variables and functions,
`UPPER_SNAKE_CASE` constants, two-space indentation, semicolons, and primarily
single-quoted strings. CSS classes use kebab-case. JSON property names use
camelCase. WebSocket message types use uppercase snake case.

Main/server modules use CommonJS; renderer files use browser scripts and globals.
There is no detected branch naming convention and no repository-wide Conventional
Commits requirement; plans MUST NOT claim either convention exists.

## Dependency Rules

- `main.js` MAY depend on Electron, Node built-ins, `server/server.js`, and the
  optional `server/ngrok.js` integration.
- `preload.js` MAY depend only on the Electron bridge APIs required to expose its
  narrow contract.
- `master/` MAY call only the API exposed by `preload.js`; it MUST NOT import
  server or Node modules.
- `server/server.js` MAY depend on Express, `ws`, Node built-ins, and domain
  modules within `server/`.
- `server/hack.js` and `server/nav.js` SHOULD remain transport-independent.
- `client/` MAY use browser APIs, the server's HTTP endpoints, static assets, and
  the WebSocket protocol; it MUST NOT depend on Electron or filesystem APIs.
- New runtime dependencies MUST have a concrete need documented in the plan and
  MUST preserve the single-package npm structure unless a structural change is
  explicitly approved.

## Testing and Quality Gates

No automated test framework, canonical test directory, coverage threshold,
linter, formatter, or CI workflow is currently configured. This absence MUST be
stated in plans and handoffs; contributors MUST NOT claim those checks passed.

Every feature specification MUST contain independently verifiable acceptance
scenarios. Every implementation plan MUST define proportionate verification for
the affected surfaces, including manual Electron/browser checks where automation
does not exist. Changes to pure server domain logic, session validation, IPC
contracts, WebSocket contracts, or ngrok credential handling SHOULD introduce
focused automated tests; the plan MUST name the proposed framework, location,
and npm command before adding them. No numeric coverage target exists until one
is explicitly adopted.

Existing applicable project commands MUST succeed:

- `npm start` for an interactive development smoke check when UI/runtime behavior
  changes.
- `npm run build:dir` for Windows packaging-sensitive changes when the required
  build environment is available.

Reviews MUST additionally verify module boundaries, session compatibility,
server/client synchronization, and Electron/public-access security as applicable.

## Development Workflow

1. Specify user-visible behavior and independently testable scenarios.
2. Identify affected runtime surfaces and contracts.
3. Plan session compatibility, synchronization, security, and packaging impacts.
4. Implement the smallest coherent vertical slice without unrelated restructuring.
5. Run the verification defined in the plan and record checks that could not run.
6. Update README or sample session documentation when setup, operation, or data
   shape changes.

## Governance

This constitution governs Spec Kit artifacts and feature work in this repository.
Amendments MUST reflect an intentional project decision, document their rationale,
and update the version below. Existing application behavior is evidence, but an
accidental inconsistency does not automatically become a new standard.

Constitution checks are required during planning and after design. Any violation
MUST be listed in the plan's Complexity Tracking table with a concrete rationale
and a rejected simpler alternative.

**Version**: 1.0.0 | **Ratified**: 2026-08-09 | **Last Amended**: 2026-08-09
