# Application Contract Inventory

This ledger is the FR-003 through FR-005 gate. Public transport implementation may begin only when every row is assigned and the verification check reports zero unclassified application-owned structured boundaries and zero unclassified serializable configuration fields.

## Public player contracts

| Current owner/value | Producer → consumers | Classification after migration | Verification |
|---|---|---|---|
| `ClientMessage` and `SESSION_HELLO` | `client/client.js` → `internal/player/protocol.go` | `player.v1.SubscribeRequest`; legacy envelope removed | Generated descriptor and final legacy scan |
| `CHARACTER_SELECT` | player UI → coordinator selection | `player.v1.SelectCharacterRequest` / `ActionResult` | RPC adapter and authority tests |
| `NAV_ACTION` | player navigation UI → coordinator/live | `player.v1.NavigateRequest` / `ActionResult` | Variant and canonical mutation tests |
| `HACK_GUESS` | player hacking UI → coordinator/live | `player.v1.GuessRequest` / `ActionResult` | Word/filler variant tests |
| `HACK_PATTERN` and exact `patternId` | player hacking UI → coordinator/live/hack | `player.v1.ActivatePatternRequest.pattern_id` with ECMAScript/JSON name `patternId` | Generated client and concurrency/random tests |
| `SESSION_WELCOME`, `PLAYER_STATE` | coordinator/player server → browser | `player.v1.PersonalizedSnapshot.player_state` | First-message and private-field tests |
| `TERMINAL_LIVE`, `TERMINAL_UPDATE`, `TERMINAL_CLEAR` | live/control → player server/browser | `player.v1.TerminalPresentation` in snapshot/update | Complete `oneof` and reconnect tests |
| `NAV_STATE`, `HACK_STATE` | control/live → player server/browser | complete optional components in `player.v1.CompoundUpdate` | One-message-per-revision tests |
| `ACTION_RESULT` | control → initiating browser | unary `player.v1.ActionResult` | Result-first/stream-first tests |
| `domain.PlayerState`, player roster/role/phase/assignment/controller projection | control → browser | `player.v1.PlayerState` and enums | Adapter and descriptor tests |
| `domain.PublicLiveState`, `ContentNode`, `NavState` | live/control → browser | `player.v1.LiveTerminal`, `ContentNode`, `NavigationState` | Deep detached comparison |
| `domain.PublicHackState`, columns, words, patterns | hack/live/control → browser/master safe view | `player.v1.PublicHackState` family | Secret-field and generation tests |
| Sound list JSON from `GET /api/sounds/{folder}` | `internal/player/http.go` → `client/sound.js` | unary `player.v1.SoundManifestRequest/Response`; endpoint removed | Category/extension/path tests |
| Static HTML/CSS/fonts/images/scripts/sounds | embedded `client/` → browser | Ordinary same-origin HTTP resources; not RPC contracts | Asset/CSP/offline package tests |

Public service enumeration is limited to `PlayerService` and the six responsibilities in `public-player.md`. `ForceHackSuccess`, `ResetFailedHack`, native operations, URL opening, runtime/server/tunnel status, client count, credentials, private hacking data, raw connections, and other sessions have no public row because they are explicitly prohibited capabilities, not omitted classifications.

## Private Wails requests and results

| Current compatibility group | Existing owner/consumer | Private protobuf classification | Verification |
|---|---|---|---|
| `RuntimeStatus`, `ServerInfo` | `app.go` → `desktop-api.js`/master | `private.v1.RuntimeStatus`, `ServerInformation` | Exact native-key fixture and adapter exhaustiveness |
| `CommandResult` | `ForceHackSuccess`, `OpenURL` → master | `private.v1.CommandResult` | Named-method matrix |
| `SessionResult`, `SaveResult`, `ActiveSession` | `internal/session`, App → master | `private.v1.SessionOperationResult`, `SaveSessionResult`, `SessionStatus` | Dialog/save/status tests |
| `PlayerConfigCommandResult`, `playerconfig.Result`, metadata | player-config service/App → master | `private.v1.PlayerConfigOperationResult`, `PlayerConfigStatus` | Open/create/reference tests |
| `CoordinationCommandResult` | roster/session/controller/broadcast App methods → master | `private.v1.CoordinationResult` | Exact facade method matrix |
| `TerminalSwitchCommandResult` | activation/clear/decision App methods → master | `private.v1.TerminalSwitchResult` | Switch status/variant tests |
| `CharacterRenamePayload` | `renameCharacter(payload)` → App | `private.v1.RenameCharacterRequest` | Adapter validation tests |
| `LogicalSessionRenamePayload` | `renameLogicalSession(payload)` → App | `private.v1.RenameLogicalSessionRequest` | Adapter validation tests |
| `AssignmentPayload` | `assignCharacter(payload)` → App | `private.v1.AssignCharacterRequest` | Adapter validation tests |
| `MoveCharacterPayload` | `moveCharacter(payload)` → App | `private.v1.MoveCharacterRequest` | Adapter validation tests |
| `LiveTerminalPayload` | activation/reset → App/control | `private.v1.TerminalActivationRequest`, `ResetFailedHackRequest` | Tree/level/intro adapter tests |
| `LiveUpdatePayload` | live update → App/control | `private.v1.LiveTerminalUpdateRequest` | Optional intro presence tests |
| `TerminalSwitchDecisionPayload` | decision → App/control | `private.v1.TerminalSwitchDecisionRequest` | Allowlisted decision tests |
| `MasterCoordinationState` and nested roster/session/broadcast/switch values | control → App/master event/status/results | `private.v1.CoordinationState` family | Descriptor and native-object comparison |

Every exact bound method and facade name is enumerated in `private-desktop.md`, including `GetRuntimeStatus`, `NewSession`, `OpenSession`, `CopyDemo`, `SaveSession`, `LoadReferencedPlayerConfig`, `NewPlayerConfig`, `OpenPlayerConfig`, `RequestTerminalActivation`, `UpdateLiveTerminal`, `RequestTerminalClear`, `ResolveTerminalSwitch`, `ForceHackSuccess`, `ResetFailedHack`, `AddCharacter`, `RenameCharacter`, `DeleteCharacter`, `RenameLogicalSession`, `AssignCharacter`, `ReleaseCharacter`, `MoveCharacter`, `SetActiveController`, `StartBroadcast`, `EndBroadcast`, and `OpenURL`.

## Private Wails events

| Exact event | Current payload | Private protobuf classification | Verification |
|---|---|---|---|
| `server-info` | native server-information object | `private.v1.ServerInformationEvent` | Event/status parity |
| `client-count` | active public stream integer | `private.v1.ClientCountEvent` | Attach/detach/multi-tab tests |
| `hack-state` | public hacking projection or null | `private.v1.HackStateEvent` importing player-safe projection | Native event tests |
| `coordination-state` | private master coordination projection | `private.v1.CoordinationStateEvent` | Revision and deep-detach tests |

Wails lifecycle `Start`/`Shutdown` are lifecycle interfaces, not serialized bridge dispatch contracts. Their serializable status/configuration values are classified below; contexts and service handles are excluded dependencies.

## Persistence fields

| Current JSON/domain value | Exact known fields | Protobuf classification | Compatibility verification |
|---|---|---|---|
| `domain.Session` | `version`, `name`, `playerConfig`, `terminals` | `persistence.v1.Session` | Version-1 fixture round trip and top-level extras |
| `domain.Terminal` | `id`, `name`, `hackLevel`, `introText`, `root` | `persistence.v1.Terminal` | Known names, validation, terminal extras |
| recursive `domain.ContentNode` | `id`, `type`, `name`, `children`, `text`, `description` | `persistence.v1.ContentNode` | Recursive extras and variant validation |
| `domain.PlayerConfig` | `version`, `name`, `roster` | `persistence.v1.PlayerConfig` | Strict decode/trailing/unknown/version tests |
| `domain.CharacterRosterEntry` | `id`, `name` | `persistence.v1.RosterEntry` | Identity/duplicate/name tests |

Selected paths, save revisions, save state, and private active handles are runtime/status/configuration semantics, not JSON fields. Recognition, sessions, streams, assignments, controller, broadcast, live runtime, navigation, hacking, pending actions, replay, and credentials are explicitly absent from both durable files.

## Serializable configuration fields

| Current owner | Serializable fields | Protobuf classification | Native exclusions in the same owner |
|---|---|---|---|
| `AppDependencies` / application lifecycle | `TunnelEnabled`; startup/shutdown order and timeout values | `config.v1.ApplicationConfig`, `StartupConfig`, `ShutdownConfig` | service interfaces, browser, desktop, event sink |
| `player.Config` | `Address`, `QueueSize`, request/body/field/time limits | `config.v1.PlayerServerConfig` | `Assets`, `Live`, `Coordinator`, callbacks |
| `control.Config` | `RequestResultLimit` | `config.v1.CoordinationConfig` | ID source, enqueue callback, runtime/terminal/trusted-hack/roster services |
| browser constants | recognition-storage semantics, reconnect three seconds, first-tab lease/election timing | `config.v1.BrowserClientConfig` | Web Locks API, storage events, timers themselves |
| `session.Locations`, `platform.SessionLocations` | Documents default, bundled demo, Application Support paths | `config.v1.PathConfig` | dialogs, filesystem implementations |
| `tunnel.Config`, `Credentials` | enabled, binary, domain, port, local URL, startup timeout, retained policy-parent compatibility value, username/password as private ephemeral values | `config.v1.TunnelConfig`, `TunnelCredentials` | process runner/handle and log writers; application-side exact-public-Host auth enforcement |
| `tunnel.ServiceOptions`, `ProcessOptions` | grace period and configured duration values | `config.v1.ShutdownConfig` | clock/`After` callbacks |
| player request safety | 4 KiB message, 8 KiB body, semantic ID/category limits | `config.v1.PublicRequestLimits` | HTTP reader, decoder, cancellation context |

## Third-party governed schemas

| Artifact | Classification |
|---|---|
| `wails.json` | Wails configuration schema; not duplicated in protobuf |
| `frontend/package.json`, `client/package.json`, `tests/browser/package.json` and lockfiles | npm schemas; not duplicated |
| `proto/buf.yaml`, `proto/buf.gen.*.yaml` | Buf configuration schemas; not duplicated |
| `.github/workflows/wails-macos.yml` | GitHub Actions schema; not duplicated |
| `build/darwin/*.plist`, entitlements | Apple plist/entitlement schemas; not duplicated |
| ngrok CLI arguments and endpoint forwarding behavior | ngrok-owned command schema; credentials and traffic-policy files are intentionally absent |

## Non-serializable implementation dependencies

| Category | Current examples | Exclusion rationale |
|---|---|---|
| Filesystems/assets | `fs.FS`, `FileSystem`, `Store` | Capabilities and interfaces, not data values |
| Callbacks/sinks | coordination enqueue, client/hack events, Wails event sink | Executable behavior cannot be serialized safely |
| Clocks/random/IDs | `After`, `hack.Random`, word source, generation/opaque ID sources | Deterministic injected dependencies, not configuration records |
| Services/adapters | live, control, session, player-config, tunnel, desktop/browser interfaces | Runtime ownership and behavior |
| OS/network/process | contexts, listeners, `http.Server`, process runner/handle, stdout/stderr writers | Owned resources with lifecycles, not structured contracts |
| Synchronization | mutexes, atomics, wait groups, channels, cancellation functions | In-process implementation mechanisms |

## Completion rule

Automated inventory verification scans exported/current boundary structs, JSON tags, generated descriptors, Wails facade identifiers, environment/flag constants, and configured defaults. A new application-owned boundary or serializable field without a matching row/schema/exclusion fails CI. Final expected counts are zero unclassified DTOs and zero unclassified serializable configuration fields.

## Final reconciliation — 2026-08-13

- Public descriptor: exactly one `fallout.terminal.player.v1.PlayerService` with
  `Subscribe`, `SelectCharacter`, `Navigate`, `Guess`, `ActivatePattern`, and
  `SoundManifest`; the public transitive import graph contains only
  `fallout/terminal/player/v1` files.
- Generated trees: all versioned schemas generate Go under `internal/gen`; only
  the public player graph generates ECMAScript under `client/gen`.
- Active browser/server boundary: generated Connect messages only. The former
  JSON envelopes, decoder, WebSocket route/client registry, sound-list endpoint,
  test fixtures, and direct transport dependency are removed.
- Private desktop: every App request/result/status and all four events map
  through the generated `private.v1` semantic registry while preserving native
  Wails object shapes.
- Persistence: every known session-v1 and player-config-v1 field maps through
  generated persistence messages; JSON codecs remain authoritative for names,
  extras, strictness, and atomic replace behavior.
- Configuration: the composition root instantiates `config.v1.ApplicationConfig`
  defaults for port 3690, queue 32, replay 256, public request limits, browser
  timing, paths, and lifecycle bounds. Credentials and injected capabilities
  remain process-local/private as classified above.
- Explicit third-party/native exclusions remain limited to Wails/npm/Buf/GitHub
  Actions/Apple/ngrok governed files plus runtime capabilities, callbacks,
  clocks, resources, synchronization, and OS handles listed above.

Reconciliation result: **0 unclassified application-owned DTO fields and 0
unclassified serializable configuration fields**. Verification is enforced by
descriptor/adapter/asset tests plus `scripts/proto-check.sh` and the final
cutover scan.
