# Private Desktop Contract: Wails Compatibility and Protobuf Semantics

Wails remains the trusted private transport. `fallout.terminal.private.v1` defines every structured request, result, status, and event semantically, while the bridge continues to pass native JavaScript primitives/objects with the exact current method and event identifiers. No protobuf binary, Base64, ProtoJSON, generic map envelope, or generic dispatcher crosses Wails.

## Bound method compatibility

| Go method | Exact frontend facade | Argument behavior | Result behavior | Private semantic messages |
|---|---|---|---|---|
| `GetRuntimeStatus` | internal adapter initialization | none | Native runtime-status object with server info, stream count, public hack state, startup/save status, revisions, coordination | `RuntimeStatus` |
| `NewSession` | `newSession()` | none; native destination dialog | Structured session result | `SessionOperationResult` |
| `OpenSession` | `openSession()` | none; native source dialog | Structured session result | `SessionOperationResult` |
| `CopyDemo` | no current facade method | none; native destination flow | Structured private session operation | `SessionOperationResult` |
| `SaveSession` | `saveSession(session)` | complete session-v1 compatibility object | Structured ordered-save result | `Session`, `SaveSessionResult` |
| `LoadReferencedPlayerConfig` | `loadReferencedPlayerConfig()` | none | Structured player-config operation | `PlayerConfigOperationResult` |
| `NewPlayerConfig` | `newPlayerConfig()` | none; native destination dialog | Structured player-config operation | `PlayerConfigOperationResult` |
| `OpenPlayerConfig` | `openPlayerConfig()` | none; native source dialog | Structured player-config operation | `PlayerConfigOperationResult` |
| `RequestTerminalActivation` | `requestTerminalActivation(payload)` | terminal identity, name, tree, hacking level, intro text | Structured terminal-switch result | `TerminalActivationRequest`, `TerminalSwitchResult` |
| `UpdateLiveTerminal` | `updateLiveTerminal(payload)` | content tree and optional intro text | Structured coordination result | `LiveTerminalUpdateRequest`, `CoordinationResult` |
| `RequestTerminalClear` | `requestTerminalClear()` | none | Structured terminal-switch result | `TerminalSwitchResult` |
| `ResolveTerminalSwitch` | `resolveTerminalSwitch(payload)` | opaque switch identity and allowlisted decision | Structured terminal-switch result | `TerminalSwitchDecisionRequest`, `TerminalSwitchResult` |
| `ForceHackSuccess` | `forceHackSuccess()` | none | Structured command result | `CommandResult` |
| `ResetFailedHack` | `resetFailedHack(payload)` | current terminal target payload | Structured coordination result | `ResetFailedHackRequest`, `CoordinationResult` |
| `AddCharacter` | `addCharacter(name)` | character display-name string | Structured coordination result | `AddCharacterRequest`, `CoordinationResult` |
| `RenameCharacter` | `renameCharacter(payload)` | character identity and display name | Structured coordination result | `RenameCharacterRequest`, `CoordinationResult` |
| `DeleteCharacter` | `deleteCharacter(characterId)` | character identity string | Structured coordination result | `DeleteCharacterRequest`, `CoordinationResult` |
| `RenameLogicalSession` | `renameLogicalSession(payload)` | logical-session identity and fallback name | Structured coordination result | `RenameLogicalSessionRequest`, `CoordinationResult` |
| `AssignCharacter` | `assignCharacter(payload)` | logical-session and character identities | Structured coordination result | `AssignCharacterRequest`, `CoordinationResult` |
| `ReleaseCharacter` | `releaseCharacter(sessionId)` | logical-session identity string | Structured coordination result | `ReleaseCharacterRequest`, `CoordinationResult` |
| `MoveCharacter` | `moveCharacter(payload)` | character identity and destination session identity | Structured coordination result | `MoveCharacterRequest`, `CoordinationResult` |
| `SetActiveController` | `setActiveController(sessionId)` | logical-session identity string | Structured coordination result | `SetActiveControllerRequest`, `CoordinationResult` |
| `StartBroadcast` | `startBroadcast()` | none | Structured coordination result | `CoordinationResult` |
| `EndBroadcast` | `endBroadcast()` | none | Structured coordination result | `CoordinationResult` |
| `OpenURL` | `openUrl(url)` | final HTTP(S) URL string, validated again in Go | Structured command result | `OpenUrlRequest`, `CommandResult` |

Exported lifecycle methods `Start` and `Shutdown` remain private application lifecycle boundaries, not desktop dispatch procedures and never public player capabilities.

## Event compatibility

| Exact event name | Exact facade | Native payload | Private semantic message |
|---|---|---|---|
| `server-info` | `onServerInfo(callback)` | Safe local/public server-information object | `ServerInformationEvent` |
| `client-count` | `onClientCount(callback)` | Raw active public-stream count integer | `ClientCountEvent` |
| `hack-state` | `onHackState(callback)` | Public hacking projection or null | `HackStateEvent` |
| `coordination-state` | `onCoordinationState(callback)` | Detached private game-master coordination projection | `CoordinationStateEvent` |

Every subscription returns the existing unsubscribe function. `GetRuntimeStatus` remains the synchronous initial snapshot that closes event-subscription races.

## Native compatibility shapes

The following exact native keys remain stable where applicable:

- command result: `ok`, optional `error`;
- session operation: `ok`, `canceled`, optional `error`, optional `filePath`, optional `session`;
- save result: `ok`, optional `error`, `requestedRevision`, optional `savedRevision`;
- player-config operation: `ok`, `canceled`, optional `error`, optional `playerConfig`, optional `session`, `state`;
- terminal-switch result: `ok`, optional `error`, optional `status`, optional `switchId`, `state`;
- coordination result: `ok`, optional `error`, `state`;
- runtime status: `serverInfo`, `clientCount`, `hackState`, optional `startupError`, `saveState`, `requestedRevision`, `savedRevision`, `coordinationState`.

Nested session, player-config, public-hack, server-information, and coordination shapes retain their established JavaScript keys and null/omission behavior. Adding a private protobuf field or variant without an explicit adapter case fails verification.

## Adapter rules

1. Decode/validate the Wails compatibility input at the privileged Go boundary.
2. Map every compatibility field to a generated private protobuf semantic value.
3. Map that value to the transport-independent domain/service request.
4. Invoke the existing narrow application service and authorization rule.
5. Map the domain result through generated private semantics and back to the exact compatibility DTO.
6. Emit only detached, credential-free, safe native payloads.

Descriptor tests enumerate every private message field, enum, and `oneof` variant and compare it with an explicit adapter registry. Reflection assists verification only; runtime business mapping stays typed and explicit.

## Capability separation

The public generated service and listener expose no path to `ForceHackSuccess`, `ResetFailedHack`, native dialogs, session/config file operations, URL opening, roster correction, controller reassignment, terminal switching, server information, runtime status, tunnel configuration, credentials, private candidates, secret words, future outcomes, raw physical connections, or another logical session's private state.

`ForceHackSuccess` and `ResetFailedHack` retain current trusted eligibility and names. Neither has a player procedure, generated public message, DOM control, browser global, keyboard shortcut, query parameter, or public endpoint.

## Errors and secrets

Private business and validation outcomes remain structured result objects. Wails may carry safe human-readable private errors, but credentials and public-auth configuration are redacted and absent from runtime status/events. Public errors use the stricter public contract and never reuse raw private dependency errors.
