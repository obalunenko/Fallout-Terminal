# Contract: Whole-System Startup

## User entry points

| Context | Sole startup action | Required result |
|---|---|---|
| Prepared development checkout | Run `wails dev` once from the repository root | Wails manages frontend preparation/watch, launches the Go application, acquires the player listener, and opens one game-master workspace |
| Installed macOS release | Launch `Fallout Terminal.app` once | The application starts the embedded master assets and in-process player server without Go, Node, npm, Vite, Wails CLI, or a terminal |

Installing documented prerequisites is setup, not startup. After setup, users
must not run `cd frontend`, `npm run dev`, a player-server command, or a second
terminal command to reach a usable system.

Public mode may receive credentials and options through the documented process
environment or application arguments, but it uses the same single startup action.
It does not introduce a second tunnel-start command.

## Wails project orchestration

`wails.json` is the development orchestration source of truth:

| Key | Planned value/responsibility |
|---|---|
| `frontend:dir` | `frontend` |
| `frontend:install` | `npm ci` using the committed frontend lock file |
| `frontend:dev:install` | `npm ci`, or omission only when it intentionally inherits `frontend:install` |
| `frontend:build` | `npm run build` |
| `frontend:dev:watcher` | `npm run dev` |
| `frontend:dev:serverUrl` | `auto` so Wails discovers the Vite development URL |

The Wails CLI may own a frontend watcher child in development. This still
satisfies the contract because `wails dev` creates and supervises it; the user
does not launch or coordinate it. The packaged application contains compiled
master assets and never starts a frontend development server.

## Runtime acquisition order

Both development and packaged entry points use the same application composition:

1. Construct domain, session, live-state, player-server, tunnel, platform, and event services without acquiring external resources.
2. Acquire the player listener on `0.0.0.0:3690` and retain its owned shutdown handle.
3. Publish local ServerInfo and make RuntimeStatus available to the desktop bridge.
4. Allow the game-master workspace to report ready.
5. If public mode is enabled and validated, start the owned tunnel after local readiness; publish public status asynchronously.

The player listener must never be a separate executable or separately invoked
command. It runs inside the Go/Wails process in both modes.

## Readiness and failure behavior

Ready means all of the following are true:

- the master workspace is usable;
- `GetRuntimeStatus` returns non-empty local ServerInfo;
- the player listener accepts HTTP and WebSocket traffic;
- the displayed player address is operator-usable;
- startup took no more than five seconds on the supported reference machine.

A bind, asset, bridge, or desktop startup failure must produce a non-secret,
actionable error and unwind every resource already acquired. Public-tunnel
validation or startup failure must not revoke local readiness and must never
start an unprotected tunnel.

## Shutdown behavior

Shutdown is idempotent in every lifecycle phase. It stops the owned tunnel and
removes credential policy material first, then closes player connections and the
listener, waits for owned goroutines, and finally releases desktop resources.
Handled partial startup uses the same reverse-order cleanup. No owned listener,
tunnel process, or credential directory may remain after the documented timeout.

## Acceptance evidence

| Requirement | Evidence |
|---|---|
| FR-002, SC-004 | Application lifecycle tests prove player-listener acquisition precedes ready status and failures unwind partial resources; manual startup reaches workspace and player address within five seconds |
| FR-019, SC-007 | Asset-manifest and packaged-app checks prove the release contains master/player/font/sound/sample assets and launches without developer tooling |
| FR-026, SC-009 | A prepared clean checkout reaches ready state from only root `wails dev`; an installed `.app` reaches the same state from one launch; neither flow uses a separately invoked frontend or player server |
| SC-006 | Repeated normal, active-tunnel, connected-player, and partial-start shutdown tests leave zero owned resources |
