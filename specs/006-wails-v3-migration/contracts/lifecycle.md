# Contract: Wails v3 Composition and Lifecycle

## Composition and Ownership

The migration preserves one Fallout Terminal core and one master window. Wails v3 objects remain host/platform concerns; domain and application services remain testable without Wails.

### Required construction order

1. Create the Wails `application.App` with master assets and application options, initially without the Fallout Terminal service registrations that depend on composed core values.
2. Create a narrow injected host capability for event emission, dialogs, external browser opening, and application-lifetime context access. `internal/platform` implements the native adapters over it.
3. Compose the existing player server, session worker, tunnel service, control/live/hack services, desktop adapter, event sink, and core `App`. Preserve the existing explicit late binding of the coordination effect router.
4. Create/register a lifecycle service that owns core start/stop but exposes no frontend callable methods.
5. Create/register the dedicated 25-method desktop service.
6. Register the exact four typed events.
7. Create one window and configure the accepted master properties.
8. Call `application.App.Run`.

No package global or hidden singleton may bridge this order. Late `RegisterService` before `Run`, constructor injection, and narrow interfaces are the accepted cycle resolution.

## Window Contract

| Property | Accepted value |
|---|---|
| count | exactly one master window |
| title | `Fallout Terminal — Master Control` |
| initial size | 1200×780 |
| minimum size | 900×600 |
| background | accepted dark RGB 11/13/10 presentation |
| assets | bundled `frontend/dist` master filesystem |
| macOS behavior | accepted single-window close and application termination behavior; no tray/background-only product |

Multi-window, tray, updater, mobile, and other new v3 capabilities are not registered or configured.

## Resource Ownership

| Resource | Owner | Acquire / publish | Release responsibility |
|---|---|---|---|
| Wails application/window | Wails v3 host in root composition | host creation, then one window | framework after lifecycle shutdown |
| Application operation context | lifecycle host | application-lifetime context before core start | cancel on application shutdown |
| Desktop adapter | `internal/platform` | ready after listener/local status can be presented | core shutdown final step |
| Session save worker | `internal/session.Service` | starts during service construction; accepts ordered saves | drain/stop after player release |
| Player listener, HTTP server, public Connect streams | `internal/player.Server` | first runtime acquisition; publish local safe server info | stop listener/streams/waitgroup after tunnel |
| Coordination/live state | `internal/control`, `internal/live` | existing canonical mutation/publication | clear process-local state before owned resource release |
| Optional ngrok process/policy | `internal/tunnel.Service` | only after local readiness; mark acquired before returned-URL validation | first shutdown release; process group/guardian cleanup |
| Event sink | `internal/platform` injected capability | usable for local status before ready-local | follows desktop/application lifetime; retains no startup child context |

## Startup Phases

Startup remains bounded by the governed 30-second overall deadline and 20-second tunnel acquisition deadline unless configuration is separately changed.

| Phase | Action | Failure handling |
|---|---|---|
| constructed | Host/adapters/core/services/window exist | unrecoverable Wails host/window/resource-construction failures may abort with a clear framework error |
| starting-player | Start player listener/generated public service once | record redacted `startupError`; unwind any partial acquisition; retain master-capable failed state when host can run |
| local-published | Store and emit safe local server information | an initial required event-sink failure is fatal application startup, visible in status, and unwinds player |
| desktop-ready | Make native adapter/operations available without retaining the bounded startup context | record actionable failure and unwind player/desktop partial state |
| ready-local | Master can call bindings, subscribe, and use local player URL | this is successful acceptance when tunnel is disabled |
| starting-tunnel | If explicitly enabled and configured, start one tunnel with its own bound | failure is nonfatal; record redacted `tunnelError`, keep ready-local |
| validating-tunnel | Validate safe HTTPS public URL and credential-free status | acquisition was already recorded; invalid/unsafe URL triggers stop and remains eligible for shutdown retry |
| ready-public | Publish safe public status while retaining local fallback | later tunnel loss falls back to actionable ready-local behavior |

Core `Start` remains idempotent. A repeated startup call does not reacquire resources or republish duplicate effects.

## Startup Error Classification

| Failure | Return a Wails `ServiceStartup` error? | Required master/runtime behavior | Cleanup |
|---|---:|---|---|
| Wails application/window cannot be created or run | yes, when no safe UI can exist | clear host error/log | framework cleanup |
| irrecoverably invalid embedded master assets before window construction | yes | clear host error/log; package gate must catch | no acquired core resources |
| player listener occupied/fails | no when master host can run | unchanged `RuntimeStatus.startupError`, actionable master state | reverse partial cleanup |
| initial event sink/local publication fails | no when master host can run | actionable startup failure, no false ready | release player and partial desktop |
| desktop adapter readiness fails | no when master host can run | actionable startup failure | release desktop partial and player |
| tunnel absent/invalid/fails | no | local URL remains usable; redacted tunnel error | stop acquired tunnel/policy, retain local player |
| startup timeout/cancel after acquisition | no when host can run | actionable bounded failure | unwind every acquired resource |

The lifecycle service catches handled core-start failures after the core has recorded status and unwound resources, then returns `nil` so Wails does not convert them into unexplained exits. Tests distinguish these from a framework-host failure.

Current baseline pre-window `log.Fatal` paths (asset sub-filesystems, default locations, resource validation, player construction) must be classified. Application-owned cases that can be represented by a safe failed core/service must reach the master; only failures that make host/window operation impossible remain host-fatal.

## Context Contract

- `ServiceStartup`'s context represents application lifetime; derive a stable operation context/cancel for adapters and commands.
- Derive bounded startup/tunnel children for acquisition only and cancel them on completion.
- Do not retain a 30-second startup child in `App`, desktop, event sink, or frontend command handling.
- On shutdown, ignore cancellation state of the application context for cleanup and create a new five-second bounded context from `context.Background()`.
- Test contexts use `t.Context()` as their root.

## Shutdown Contract

One idempotent shutdown owner handles Wails `ServiceShutdown` or an explicit application shutdown hook. It performs:

1. Atomically transition to stopping; repeated callers join/return the same result and do not release twice.
2. Clear live publication and process-local coordinator state so no new effects are accepted.
3. Stop the owned tunnel/process/policy, including an acquired-but-invalid tunnel.
4. Stop the player listener, active/reconnecting/overflow streams, HTTP server, and waitgroups.
5. Drain and stop the session save worker so accepted saves are not lost.
6. Release the desktop adapter/application operation context.
7. Report safe cleanup errors/timeouts, then allow Wails to destroy the window/application.

The configured overall deadline is five seconds. Component graceful escalation (including the tunnel's two-second process grace) must fit within it. A component failure does not skip later cleanup.

## Required Trigger Matrix

| Trigger | Expected result |
|---|---|
| master window close under accepted macOS behavior | one shutdown sequence, one clean app exit |
| Cmd+Q/application quit | same sequence and deadline |
| handled `go run ./cmd/build dev` interrupt | same sequence; no lingering listener/ngrok process |
| normal core shutdown | tunnel → player → session worker → desktop |
| startup failed after player acquisition | acquired subset released in reverse; master failure remains explainable while host lives |
| tunnel acquired but URL invalid | tunnel stop attempted immediately and retried at shutdown if still owned |
| repeated/concurrent shutdown | at most one effective release per resource; callers complete safely |
| active and slow/overflow/reconnecting players | listener/streams terminate without blocking beyond deadline |

## Observable Status

The existing runtime status remains detached and credential-free. It exposes local server info when available, optional redacted `startupError`, tunnel failure through safe server status, client count, public hack state, save state/revisions, and private coordination state. Internal phase is test-observable but not added to protobuf. `desktop-api.js` and `master.js` must render application-owned startup failure rather than presenting an empty or falsely ready UI.

## Verification

- Unit tests preserve exact acquisition/publication/unwind order, idempotency, timeouts, and local-tunnel fallback.
- Host integration tests cover construction order, late registration before Run, lifecycle method non-exposure, application-operation context lifetime, and host-vs-core failure classification.
- Injected platform tests cover dialog/browser/event adapter behavior independent of Wails.
- Process/listener checks prove normal close, Cmd+Q, dev interrupt, partial startup, invalid public URL, and repeated shutdown leave one listener during operation and zero owned resources after quit.
- Go tests use Testify, table-driven cases, `t.Context()`, and protobuf comparison rules from the constitution.
