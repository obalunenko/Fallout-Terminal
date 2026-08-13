# Contract: Private Desktop Bridge

## Boundary

Wails v3 is the trusted transport behind `frontend/src/desktop-api.js`. The generated service surface is allowlisted, while `fallout.terminal.private.v1` and the explicit adapters in `app_contract.go` continue to define application-owned request/result/status semantics. The bridge passes native JavaScript primitives and objects, never protobuf bytes, Base64, ProtoJSON, serialized envelopes, or generic maps.

## Registered Service Inventory

The generated Wails v3 desktop service MUST contain exactly these 25 methods. Generated module names or paths may change only behind the facade.

| # | Generated service method | Facade consumer | Arguments | Native result / cancellation | Semantic adapter |
|---:|---|---|---|---|---|
| 1 | `GetRuntimeStatus` | bootstrap plus deliberate read-only startup-status presentation | none | Runtime status; no `ok` on successful transport call | `RuntimeStatus` |
| 2 | `NewSession` | `newSession()` | none; native destination dialog | Session result; cancel is `{ok:false,canceled:true}` with no fabricated error | `SessionOperationResult` |
| 3 | `OpenSession` | `openSession()` | none; native source dialog | Session result; same cancel rule | `SessionOperationResult` |
| 4 | `CopyDemo` | trusted binding only; no authored facade control | none; explicit native destination flow | Session result; copy never mutates bundled sample | `SessionOperationResult` |
| 5 | `SaveSession` | `saveSession(session)` | complete session-v1 compatibility object | Ordered save result with requested/saved revisions | `Session`, `SaveSessionResult` |
| 6 | `LoadReferencedPlayerConfig` | `loadReferencedPlayerConfig()` | none | Player-config operation result | `PlayerConfigOperationResult` |
| 7 | `NewPlayerConfig` | `newPlayerConfig()` | none; native destination dialog | Player-config operation; cancel preserved | `PlayerConfigOperationResult` |
| 8 | `OpenPlayerConfig` | `openPlayerConfig()` | none; native source dialog | Player-config operation; cancel preserved | `PlayerConfigOperationResult` |
| 9 | `RequestTerminalActivation` | `requestTerminalActivation(payload)` | terminal ID/name/tree, hacking level, intro text | Terminal-switch result | `TerminalActivationRequest`, `TerminalSwitchResult` |
| 10 | `UpdateLiveTerminal` | `updateLiveTerminal(payload)` | content tree and optional intro text | Coordination result | `LiveTerminalUpdateRequest`, `CoordinationResult` |
| 11 | `RequestTerminalClear` | `requestTerminalClear()` | none | Terminal-switch result | `TerminalSwitchResult` |
| 12 | `ResolveTerminalSwitch` | `resolveTerminalSwitch(payload)` | opaque switch ID and allowlisted decision | Terminal-switch result | `TerminalSwitchDecisionRequest`, `TerminalSwitchResult` |
| 13 | `ForceHackSuccess` | `forceHackSuccess()` | none | Command result; trusted eligibility unchanged | `CommandResult` |
| 14 | `ResetFailedHack` | `resetFailedHack(payload)` | current terminal target | Coordination result | `ResetFailedHackRequest`, `CoordinationResult` |
| 15 | `AddCharacter` | `addCharacter(name)` | display-name string | Coordination result | `AddCharacterRequest`, `CoordinationResult` |
| 16 | `RenameCharacter` | `renameCharacter(payload)` | character ID and display name | Coordination result | `RenameCharacterRequest`, `CoordinationResult` |
| 17 | `DeleteCharacter` | `deleteCharacter(characterId)` | character ID string | Coordination result | `DeleteCharacterRequest`, `CoordinationResult` |
| 18 | `RenameLogicalSession` | `renameLogicalSession(payload)` | logical-session ID and fallback name | Coordination result | `RenameLogicalSessionRequest`, `CoordinationResult` |
| 19 | `AssignCharacter` | `assignCharacter(payload)` | logical-session and character IDs | Coordination result | `AssignCharacterRequest`, `CoordinationResult` |
| 20 | `ReleaseCharacter` | `releaseCharacter(sessionId)` | logical-session ID | Coordination result | `ReleaseCharacterRequest`, `CoordinationResult` |
| 21 | `MoveCharacter` | `moveCharacter(payload)` | character ID and destination session ID | Coordination result | `MoveCharacterRequest`, `CoordinationResult` |
| 22 | `SetActiveController` | `setActiveController(sessionId)` | logical-session ID | Coordination result | `SetActiveControllerRequest`, `CoordinationResult` |
| 23 | `StartBroadcast` | `startBroadcast()` | none | Coordination result | `CoordinationResult` |
| 24 | `EndBroadcast` | `endBroadcast()` | none | Coordination result | `CoordinationResult` |
| 25 | `OpenURL` | `openUrl(url)` | final URL string, validated again in Go | Command result; only absolute HTTP(S) is eligible | `OpenUrlRequest`, `CommandResult` |

`GetRuntimeStatus` remains the single snapshot/bootstrap operation. The master may consume it through the facade to render the already-governed optional `startupError`; this closes a current baseline visibility gap without adding a Wails capability or protobuf field. `CopyDemo` remains intentionally unavailable to authored UI.

## Native Result Shapes

Exact native keys and existing nested shapes remain stable:

| Result | Required/optional keys |
|---|---|
| Command | `ok`, optional `error` |
| Session operation | `ok`, `canceled`, optional `error`, optional `filePath`, optional `session` |
| Save | `ok`, optional `error`, `requestedRevision`, optional `savedRevision` |
| Player configuration | `ok`, `canceled`, optional `error`, optional `playerConfig`, optional `session`, `state` |
| Terminal switch | `ok`, optional `error`, optional `status`, optional `switchId`, `state` |
| Coordination | `ok`, optional `error`, `state` |
| Runtime status | `serverInfo`, `clientCount`, `hackState`, optional `startupError`, `saveState`, `requestedRevision`, `savedRevision`, `coordinationState` |

Nested session, player-config, public-hack, server-information, and coordination keys retain their feature-005 null/omission behavior. The migration does not add a serialized lifecycle phase.

## Error and Cancellation Semantics

- Business, validation, eligibility, and native-operation outcomes remain structured results.
- A rejected generated-service promise is normalized by the facade to `{ok:false,error:<safe string>}` for command calls.
- Cancellation remains operation-specific and non-exceptional: native file-dialog cancel produces the established empty-path/no-native-error path and the structured result sets `canceled`.
- Save ordering and requested/saved revision semantics are unchanged.
- Private errors may be actionable but must redact credentials, tunnel policy, dependency internals, and secrets.
- `OpenURL` is parsed and allowlisted again at the privileged Go boundary immediately before `app.Browser.OpenURL`; the facade does not call `window.open` or expose a raw browser manager.
- `ForceHackSuccess`, `ResetFailedHack`, roster corrections, controller changes, and terminal-switch actions preserve trusted eligibility and have no public player route.

## Adapter Ownership

The host service is a narrow forwarding adapter; it does not own domain rules. For every structured call:

1. The generated Wails method receives the established native argument.
2. `app_contract.go` maps every field to a generated private protobuf semantic value.
3. The adapter maps that value into the existing transport-neutral service/domain request.
4. The existing service validates, authorizes, mutates canonical state, and returns a domain result.
5. The result maps through generated private semantics to a detached native DTO.
6. The facade preserves the established JavaScript normalization and freezes where it already does.

Private generated Go messages do not enter `frontend/`; private generated ECMAScript is not created. Domain packages do not import Wails.

## Wails v3 Binding Integration

- Register one dedicated desktop service before `application.App.Run`.
- Generate clean modules into `frontend/bindings` before the master Vite bundle.
- Import those modules only from `desktop-api.js`.
- Use exact `@wailsio/runtime` `3.0.0-beta.8` and the same package's official Vite plugin.
- A missing production binding is a build/initialization failure. Test doubles enter through a test-only injection or alias that production configuration cannot resolve.
- Two clean generations must produce identical complete content and identical 25-method inventory.

## Forbidden Generated or Frontend Capabilities

The generated inventory, facade, source, and production bundle MUST contain zero:

- `Start`, `Shutdown`, `ServiceStartup`, or `ServiceShutdown` callable bindings;
- generic dispatch, reflection/capability discovery, or raw application-manager access;
- arbitrary filesystem, process, shell, environment, credential, dialog, or browser primitives;
- player listener/service procedures or player-private state;
- protobuf binary/Base64/ProtoJSON serialization or private generated JavaScript;
- `window.go`, `window.runtime`, `frontend/wailsjs`, Wails v2 imports, Electron globals, or optional privileged production fallbacks;
- direct privileged call from `master.js` or any authored UI module outside `desktop-api.js`.

The Wails generator's beta.8 internal-method exclusion is verified rather than trusted. Reflection tests target the allowlisted service itself and compare it with this ledger.

## Verification

- Descriptor and adapter exhaustiveness retain zero unclassified private fields/enums/variants.
- Table-driven Go tests cover all 25 operations, native shapes, validation, errors, cancellation, redaction, and detachment using Testify and `t.Context()` where context is needed.
- JavaScript facade tests cover all generated call names/arguments, rejection normalization, result normalization, `GetRuntimeStatus` caching/status visibility, and `CopyDemo` absence.
- Source and bundle scans enforce every forbidden item above.
- Clean standalone `frontend` compilation proves the chosen generated path is present and no runtime package is downloaded at application runtime.
