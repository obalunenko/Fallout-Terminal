# Contract: Game-Master Desktop Bridge

## Boundary

Only registered Wails methods and runtime events cross between the master frontend and privileged Go services. `frontend/src/desktop-api.js` exposes the compatibility object `window.electronAPI`; `master.js` has no direct filesystem, process, environment, server, or arbitrary network access.

## Bound commands

| Go method | Compatibility method | Input | Result/effect |
|---|---|---|---|
| `GetRuntimeStatus` | adapter initialization | none | Current server info, count, public hack status, startup/save status |
| `NewSession` | `newSession()` | none | `SessionResult` after native save dialog and initial write |
| `OpenSession` | `openSession()` | none | `SessionResult` after native open dialog and validation |
| `SaveSession` | `saveSession(session)` | complete Session | `SaveResult`; ordered durable revision |
| `SetLiveTerminal` | `setLiveTerminal(payload)` | live payload | Validate, install canonical live state, broadcast full snapshot |
| `UpdateLiveTerminal` | `updateLiveTerminal(payload)` | tree and optional intro text | Validate, update/revalidate, broadcast update |
| `ClearLiveTerminal` | `clearLiveTerminal()` | none | Clear live state and broadcast clear |
| `ForceHackSuccess` | `forceHackSuccess()` | none | Mutate active puzzle if eligible; broadcast public state |
| `OpenURL` | `openUrl(url)` | string | Parse and open only HTTP(S); otherwise no effect/error result |

Every command validates again in Go. Expected cancellation and user-correctable errors return structured results. Panics and raw internal errors never cross the bridge.

## Events

| Event | Payload | When emitted |
|---|---|---|
| `server-info` | ServerInfo | DOM ready, tunnel success, tunnel failure |
| `client-count` | integer | DOM ready and connection registration/removal |
| `hack-state` | PublicHackState or null | DOM ready, live set/clear, accepted puzzle action/override |

The adapter's `onServerInfo`, `onClientCount`, and `onHackState` methods return unsubscribe functions. Frontend reload must not accumulate listeners.

## Live payload validation

- Terminal ID/name are non-empty bounded strings.
- `hackLevel` is an integer `0..5`.
- `introText` and recursive tree use session model bounds.
- Root must be a folder with ID `root`.
- Update payload is applied only while a live state exists.

## Security

- Keep the master CSP restrictive; add only origins/scripts required by generated Wails assets.
- Use generated Wails bindings, not arbitrary evaluation or a general command dispatcher.
- Do not expose environment variables, raw process control, or general file read/write methods.
- Validate external URL protocol in Go immediately before calling the platform browser API.

