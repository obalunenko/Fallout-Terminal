---
status: migrated
feature: Public ngrok Access
source: existing implementation
---

# Implementation Plan: Public ngrok Access

**Migration status**: Reconstructed from the existing implementation on 2026-08-09  
**Specification**: `specs/public-ngrok-access/spec.md`

## Summary

The implemented feature optionally starts an external ngrok child process after the embedded local server is ready. A dedicated CommonJS module resolves and validates Basic Auth credentials, writes an ephemeral ngrok traffic policy, spawns one tunnel, recognizes an HTTPS public URL from process output, and cleans up startup resources. The Electron main process forwards either public-address information or a tunnel error to the sandboxed game-master renderer and asks the child to stop during application shutdown.

This is a documentation-only migration. It does not change source, dependencies, runtime behavior, secrets, session data, or packaging configuration.

## Technical Context

| Area | Existing choice |
|---|---|
| Language/runtime | JavaScript, CommonJS, Node.js APIs inside Electron 28 |
| Local application | Existing Express 4 and `ws` 8 server on port 3690 by default |
| Public ingress | Separately installed ngrok binary spawned as a child process |
| Authentication | Mandatory ngrok Basic Auth traffic policy |
| Credential inputs | Paired or combined environment variables; internal function options for tests/callers |
| Tunnel activation | `--ngrok`, `NGROK_ENABLED=1`, or `npm run start:ngrok` |
| Endpoint | Reserved domain default with environment/internal override |
| Secret material | Mode-`0600` JSON file inside a fresh OS temporary directory |
| Renderer integration | Existing `server-info` preload listener; renderer never receives credentials |
| Persistent storage | None; configuration and process state are runtime-only |
| Automated tests | No test framework, test directory, coverage threshold, lint command, or CI workflow is configured |

## Detected Scope

### Dedicated tunnel module

- `server/ngrok.js` — resolves and validates configuration, builds the traffic policy, creates and cleans temporary files, spawns and tracks ngrok, parses public URLs, captures startup errors, enforces a timeout, and stops the child.

### Electron orchestration and renderer boundary

- `main.js` — starts the local server, detects public mode, lazy-loads the tunnel module, maps tunnel results into displayed server information, and invokes shutdown cleanup.
- `preload.js` — exposes the narrow `onServerInfo` listener and `openUrl` request used by the feature; it does not expose credentials or tunnel controls.

### Game-master status presentation

- `master/index.html` — contains the player-address element.
- `master/master.js` — renders local, public, and error states; retains a local click target on failure; and asks the main process to open the selected address.
- `master/master.css` — supplies the existing address and error presentation styles.

### Configuration and documentation

- `package.json` — supplies the `start:ngrok` convenience script; no ngrok npm dependency is installed.
- `README.md` — documents ngrok installation, activation, credentials, domain and binary overrides, fail-closed behavior, and packaged invocation.

### Related but out of scope

- `server/server.js` owns the local HTTP/WebSocket service and returns its port but has no ngrok-specific route, policy, or message.
- `client/` is served unchanged through the tunnel. Its `wss` selection and reconnection behavior are documented in the player-presentation and live-broadcast specs.
- `sessions/` contains no tunnel configuration or runtime state.
- General Electron sandboxing, CSP, and preload hardening predate this feature; only the narrow status/open-address integration is relevant here.

## Existing Architecture and Lifecycle

```text
Electron startup
  → start local Express/WebSocket server
  → create sandboxed master window and show local URL
  → detect --ngrok or NGROK_ENABLED=1
  → validate external credential configuration
  → create private temporary traffic-policy file
  → spawn ngrok for the local port and configured endpoint
  → parse HTTPS public URL
  → delete temporary policy
  → send public URL + retained local URL to master UI

failure after policy creation but before public URL
  → delete temporary policy on a best-effort basis
  → terminate child if necessary
  → retain local server
  → send tunnel error to master UI

Electron before-quit
  → request active ngrok child termination
```

`server/ngrok.js` stores one module-level child reference. Startup completion means the endpoint was recognized, not that the child lifecycle is continuously supervised. Once the promise resolves, later child exit is logged and clears the module reference but does not update the renderer.

## Configuration and Process Contract

### Configuration resolution

1. Internal `options.basicAuth`, when defined, is the direct credential source.
2. Otherwise, if either paired username/password option or environment value is truthy, both paired values are required and joined with the first colon separator.
3. Otherwise, `NGROK_BASIC_AUTH` supplies the combined credential.
4. Internal binary and domain options precede `NGROK_BIN` and `NGROK_DOMAIN`, which precede the implemented defaults.
5. A domain without an HTTP(S) prefix receives `https://` before it is passed to ngrok.

### Child invocation

The effective invocation is:

```text
<binary> http <local-port>
  --url <endpoint-url>
  --traffic-policy-file <temporary-policy-path>
  --log stdout
  --log-format json
```

The process inherits the Electron main process environment, has ignored stdin and piped stdout/stderr, and uses `windowsHide: true`.

### Status contract

| State | Main → master payload additions | Master presentation |
|---|---|---|
| Local/pending | Existing `{ url, port, ... }` | Local URL and local-link tooltip |
| Public | `{ url: publicUrl, localUrl, tunnel: true }` | Public URL and tooltip retaining local URL |
| Failed | `{ tunnelError }` on the local server object | `NGROK: ОШИБКА`, error/local tooltip, local URL retained as click target |

The preload listener forwards these objects from the trusted main process. Clicking the address sends the string back through the preload bridge; `main.js` reparses it and allows only HTTP(S) before calling Electron's external opener.

## Reconstructed Implementation Phases

### Phase 1 — Opt-in configuration and fail-closed authentication

- Added CLI/environment activation while preserving ordinary local startup.
- Added paired and combined Basic Auth credential inputs with explicit precedence.
- Required complete credentials and validated username, password length, and line-break constraints.
- Added configurable binary/domain inputs with stable defaults.

### Phase 2 — Ephemeral ngrok policy

- Generated one enforced Basic Auth request policy as JSON.
- Created a feature-specific OS temporary directory and mode-`0600` policy file.
- Added best-effort recursive cleanup after success and failures occurring after the policy helper returns successfully.
- Kept credentials out of renderer messages, sessions, and repository configuration.

### Phase 3 — Tunnel process lifecycle

- Enforced one tracked tunnel process at a time.
- Spawned ngrok against the running local server with structured stdout logging.
- Parsed JSON log lines and provided a compatible plain-text fallback for HTTPS URL discovery.
- Captured bounded stderr, translated missing-binary errors, handled premature exit, and enforced a startup timeout.
- Added explicit child termination for failure and Electron shutdown paths.

### Phase 4 — Electron and game-master integration

- Started the tunnel only after local server startup, keeping local service available on tunnel failure.
- Forwarded public URL, retained local URL, tunnel marker, or error through existing `server-info` IPC.
- Updated the master address state and tooltip for public and failed modes.
- Reused main-process HTTP(S) validation for clickable addresses.

### Phase 5 — Operation documentation

- Added the `start:ngrok` npm script.
- Documented ngrok account setup, mandatory credentials, public URL behavior, overrides, security properties, and packaged activation in `README.md`.

## Key Technical Decisions

1. **Public access is opt-in** so ordinary local-network operation does not depend on ngrok.
2. **Authentication is mandatory before spawn** so invalid configuration cannot accidentally create an open endpoint.
3. **ngrok remains an external binary** rather than an npm runtime dependency, preserving the existing single-package dependency set.
4. **Traffic policy is ephemeral** so credentials are consumable by ngrok without becoming project or session data.
5. **The local server starts first and survives tunnel failure** so a public-access problem does not remove local-table play.
6. **The renderer receives status only** so environment access and child-process authority remain in trusted Node/Electron boundaries.
7. **One process-global tunnel is allowed** because the application exposes one embedded server and one displayed player address.
8. **Public URL discovery is output-driven** because the spawned CLI reports the allocated/reserved endpoint asynchronously.
9. **External opening is validated again in main** so renderer-provided strings are not trusted merely because they came through preload.

## Constitution Check

| Principle | Assessment |
|---|---|
| Preserve runtime boundaries | Pass: Electron orchestration stays in `main.js`, tunnel/process logic stays in `server/ngrok.js`, preload exposes status/opening only, and the master renderer only presents state. |
| Keep shared state server-authoritative | Not materially changed: the tunnel forwards the existing HTTP/WebSocket service without adding client-owned state or messages. |
| Protect desktop and public-access boundaries | Pass with recorded gaps: public mode validates mandatory credentials, uses a temporary policy, keeps secrets out of the renderer, and retains HTTP(S) external-URL validation. |
| Preserve session data compatibility | Pass: no tunnel configuration, credential, address, or child state is stored in session JSON. |
| Match established code conventions | Pass: the module is CommonJS, uses lowercase filenames/camelCase functions, two-space indentation, and existing main/preload boundaries. |
| Testing and quality gates | Gap recorded: no automated framework exists and no automated or live integration checks can be claimed from repository state. |

No constitutional violation requiring a complexity exception was found. Post-start monitoring, diagnostics, endpoint validation, timeout text, and test omissions are follow-up gaps rather than accepted future design choices.

## Complexity Assessment

| Dimension | Existing scope |
|---|---|
| Dedicated implementation | 1 file, 203 lines (`server/ngrok.js`) |
| Adjacent integration | `main.js`, `preload.js`, `master/index.html`, `master/master.js`, `master/master.css` |
| Configuration/documentation | `package.json`, `README.md` |
| Runtime boundaries crossed | Electron main → external child; main → preload → master renderer; remote HTTP/WebSocket → ngrok → local server |
| External runtime requirement | Separately installed/configured ngrok binary |
| New npm dependency | None |
| Persistent schema impact | None |
| Existing automated coverage | None detected |

The feature is moderate in operational complexity because it crosses process, filesystem, network-ingress, and renderer-status boundaries despite having one dedicated module.

## Verification Strategy for the Existing Feature

The repository has no configured automated test framework, lint command, coverage threshold, CI workflow, or canonical test command. This migration therefore records checks that should be performed and does not claim they passed.

### Focused automated tests recommended for follow-up

Adopt Node's built-in `node:test` runner before adding the tests, avoiding a new runtime dependency:

- Location: `test/server/ngrok.test.js` for pure configuration, validation, policy, and URL-parsing behavior.
- Location: `test/server/ngrok-process.test.js` for child-process lifecycle using an injectable or fixture executable.
- Command: add `"test": "node --test"` to `package.json`, then run `npm test`.
- Cases: credential precedence, paired-variable completeness, password boundaries, line breaks, policy structure, HTTPS JSON/plain-text parsing, malformed output, duplicate start, missing binary, timeout, premature exit, bounded diagnostics, cleanup, and stop behavior.

### Manual/local integration checks

1. Run `npm start`; confirm only the local URL appears and no ngrok process starts.
2. Run public mode without credentials, with one paired variable, and with invalid length/newline values; confirm a visible error and usable local URL.
3. Run public mode with a missing `NGROK_BIN`; confirm the selected binary appears in the error and no temporary policy remains.
4. Configure a valid ngrok account, reserved domain, and credential, then run `npm run start:ngrok`.
5. Confirm the master changes from the local URL to HTTPS public URL and retains the local URL in its tooltip.
6. Confirm anonymous HTTP and WebSocket attempts receive `401 Unauthorized` while valid Basic Auth permits the player application and socket connection.
7. Click the displayed public address and confirm Electron opens it; confirm unsupported schemes are ignored by the main process.
8. Close Electron and confirm the ngrok child exits.
9. Force ngrok to exit after successful startup and observe the known stale-link gap.
10. Repeat with `NGROK_BASIC_AUTH`, `NGROK_DOMAIN`, `NGROK_BIN`, and `NGROK_ENABLED=1` to verify supported alternatives.

`npm run build:dir` is not required for this documentation-only migration. For future packaging-sensitive changes, run it where the Windows build environment is available and verify `--ngrok` plus external binary discovery in the packaged application.

## Identified Follow-up Gaps

1. **No automated verification** — credential, policy, parsing, cleanup, and child-process behavior have no regression suite.
2. **No committed ingress integration check** — HTTP and WebSocket Basic Auth enforcement is documented but not continuously verified.
3. **Stale UI after later child exit** — the success promise has settled, so a later exit is logged without a replacement `server-info` event or retry.
4. **Incomplete stdout diagnostics** — only public URLs are consumed from stdout; other structured ngrok errors may be lost when stderr is empty.
5. **Endpoint scheme mismatch** — an explicit HTTP endpoint is passed to ngrok even though URL recognition accepts HTTPS only.
6. **Misleading custom-timeout message** — the rejection text says 20 seconds regardless of the internal `timeoutMs` option.
7. **Partial policy-creation cleanup gap** — if directory creation succeeds but the policy write throws, the helper never returns its cleanup callback and may leave the directory behind.
8. **Best-effort cleanup is silent** — inability to remove a credential-bearing temporary directory is neither retried nor surfaced.
9. **No graceful termination acknowledgement** — `stopNgrok()` sends the default kill signal and clears the reference without waiting for exit or escalating a stuck child.
