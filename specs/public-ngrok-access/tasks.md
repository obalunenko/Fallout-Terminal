---
status: migrated
feature: Public ngrok Access
source: existing implementation
---

# Tasks: Public ngrok Access

**Migration status**: Reconstructed from the existing implementation on 2026-08-09  
**Completion convention**: Every implementation task is marked complete because it describes behavior already present in the repository. Follow-up gaps are recorded separately and are not represented as completed work.

## Phase 1 — Opt-in and Credential Protection

- [x] T001 Add public-mode activation through `--ngrok`, `NGROK_ENABLED=1`, and the `start:ngrok` script in `main.js` and `package.json`.
- [x] T002 Resolve paired and combined Basic Auth configuration with explicit precedence and reject incomplete paired values in `server/ngrok.js`.
- [x] T003 Validate non-empty usernames, line-break constraints, and 8–128-character passwords before tunnel spawn in `server/ngrok.js`.
- [x] T004 Generate an enforced ngrok Basic Auth traffic policy in a mode-`0600` file under a fresh OS temporary directory in `server/ngrok.js`.

## Phase 2 — Tunnel Process Lifecycle

- [x] T005 Resolve configurable binary, domain, and timeout values and normalize unqualified domains to HTTPS in `server/ngrok.js`.
- [x] T006 Reject duplicate starts and spawn one tracked ngrok child with the local port, endpoint, traffic-policy, and structured logging arguments in `server/ngrok.js`.
- [x] T007 Discover an HTTPS public URL from JSON or compatible plain-text ngrok output and resolve startup only after discovery in `server/ngrok.js`.
- [x] T008 Handle policy preparation, spawn, missing-binary, premature-exit, and timeout failures while retaining bounded stderr diagnostics in `server/ngrok.js`.
- [x] T009 Remove successfully created temporary policy material on URL discovery and later handled failure, terminate a failed child when needed, and expose idempotent tunnel stopping in `server/ngrok.js`.

## Phase 3 — Electron and Game-Master Integration

- [x] T010 Start ngrok only after the embedded local server is listening and supply its resolved port in `main.js`.
- [x] T011 Forward public URL/local URL metadata or a tunnel error through `server-info` while preserving local operation in `main.js` and `preload.js`.
- [x] T012 Render local, public, and failed-tunnel address states and retain an explanatory local-address tooltip in `master/index.html`, `master/master.js`, and `master/master.css`.
- [x] T013 Open clicked addresses only after HTTP(S) validation in `main.js` and request active tunnel termination during Electron shutdown in `main.js`.

## Phase 4 — Operator Documentation

- [x] T014 Document ngrok installation, account token setup, mandatory credentials, supported environment overrides, public URL behavior, cleanup guarantees, and packaged activation in `README.md`.

## Dependencies and Reconstructed Order

1. Credential resolution and policy generation preceded safe tunnel spawn.
2. Tunnel spawn and URL discovery preceded Electron success/error status integration.
3. Main-process status payloads preceded the master renderer's public/error presentation.
4. Shutdown cleanup depended on retaining the dedicated module's stop function.
5. Package scripts and operator documentation described the completed runtime path.

## Identified Gaps

- No automated test runner or committed tests cover credential validation, traffic-policy output, URL parsing, temporary cleanup, or process lifecycle.
- No repeatable integration test verifies anonymous denial and authenticated success for both HTTP and WebSocket access.
- A tunnel that exits after successful startup clears the module reference but does not update the master UI or restart automatically.
- Non-URL ngrok stdout is not retained for diagnostics, so some failures can degrade to a generic exit-code error.
- Custom HTTP endpoints are accepted for child invocation even though public URL discovery requires HTTPS.
- The timeout error text always says 20 seconds even when `timeoutMs` is overridden internally.
- A policy-file write failure after temporary-directory creation can throw before the cleanup callback is returned, leaving the directory behind.
- Temporary cleanup failures are silent and do not retry removal of credential-bearing material.
- Tunnel shutdown does not await child exit or escalate if the child ignores the initial termination signal.

## Verification Status

- Artifact review can verify the implementation mapping and configuration contract without changing runtime behavior.
- No automated test, lint, coverage, or CI result is claimed because none is configured in the repository.
- No live ngrok or Electron smoke test is claimed as part of this documentation-only migration.
- Future runtime changes should use the verification strategy in `specs/public-ngrok-access/plan.md` and run `npm run build:dir` when packaging behavior is affected and a suitable Windows build environment is available.
