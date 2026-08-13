# Data Model: Wails v3 Runtime Migration

## Scope

This migration introduces no new application data model. Domain, private protobuf, public `fallout.terminal.player.v1` ConnectRPC, session-v1, player-config-v1, player configuration, and public/private projection models remain unchanged unless a separately specified and justified contract delta is discovered.

The records below are design and acceptance concepts for the migration. They do not enter session JSON, player-config JSON, protobuf schemas, public RPCs, or canonical gameplay state.

## Preserved Application Models

| Model family | Owner | Migration rule |
|---|---|---|
| Domain/session/terminal/content | `internal/domain`, `internal/session` | Preserve version-1 JSON names, unknown compatible fields, references, validation, revision ordering, and atomic replacement |
| Player configuration/roster | `internal/domain`, `internal/playerconfig` | Preserve version 1, strict decode, identity validation, relative references, complete-file atomic publication |
| Navigation/hacking/live state | `internal/nav`, `internal/hack`, `internal/live` | Preserve canonical server-authoritative rules and detached projections |
| Coordination/session/player state | `internal/control` | Preserve revision, authority, replay, assignment, controller, and broadcast semantics |
| Public RPC | `proto/fallout/terminal/player/v1`, generated Go/ECMAScript | Preserve the feature-005 service, messages, cardinality, limits, privacy, and reconnect behavior |
| Private desktop semantics | `proto/fallout/terminal/private/v1`, `app_contract.go` | Preserve every structured request/result/status/event and explicit native DTO adapter |
| Serializable configuration | governed `config.v1` messages and composition defaults | Preserve player, browser, lifecycle, path, tunnel, and request-limit meanings |

## Migration Design Records

### VersionPinSet

An immutable compatibility selection used by one candidate.

| Field | Type | Rule |
|---|---|---|
| `goModule` | exact module/version | `github.com/wailsapp/wails/v3 v3.0.0-beta.8` |
| `goTagCommit` | Git SHA | `81a149919f91f2149d3fe9be5a27472ae7617b8e` |
| `cli` | exact install target | `github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.8` |
| `frontendRuntime` | exact package/version | `@wailsio/runtime` `3.0.0-beta.8` |
| `frontendArtifact` | registry identity | npm gitHead, checksum/integrity, and lockfile resolution |
| `vitePlugin` | package subpath | `@wailsio/runtime/plugins/vite` from the same exact package |
| `evidenceDate` | date | Date immutable sources and registry metadata were inspected |

Validation: all members are exact, all configuration and automation agree, and a version change invalidates prior generated/package acceptance evidence.

### HostLifecycleState

An internal state-machine observation, not a new serialized runtime-status field.

| State | Entry condition | Allowed next states |
|---|---|---|
| `constructed` | Wails host, adapters, core, and service registrations exist | `starting`, `stopping` |
| `starting` | bounded core acquisition begins | `ready-local`, `failed`, `stopping` |
| `ready-local` | player listener and desktop status are available | `ready-public`, `failed`, `stopping` |
| `ready-public` | optional tunnel is acquired and validated | `ready-local` on reported tunnel loss, `failed`, `stopping` |
| `failed` | an application-owned fatal startup failure is recorded and partial acquisition unwound | `stopping` |
| `stopping` | one shutdown owner begins reverse release with a fresh bounded context | `stopped` |
| `stopped` | every acquired resource has completed or reported bounded cleanup | terminal |

The master-visible state uses the unchanged `RuntimeStatus` fields: `serverInfo`, optional `startupError`, save state/revisions, client count, hack state, and coordination state. The UI derives actionable starting/ready/failed presentation from those existing fields; it does not require a new protobuf phase field.

### OwnedRuntimeResource

| Field | Meaning |
|---|---|
| `kind` | player listener/streams, optional tunnel process, session worker, desktop adapter, or Wails window |
| `owner` | package/service with exclusive lifecycle responsibility |
| `acquired` | set immediately after successful acquisition, before validation that might fail |
| `published` | whether safe status about the resource reached the master |
| `releaseOrder` | tunnel → player listener/streams → session worker → desktop adapter; Wails owns final window destruction |
| `releaseAttempted` | guards idempotency without losing a required retry after partial cleanup failure |
| `releaseResult` | success, safe error, or timeout evidence |

Invariant: no acquired resource lacks an unwind path; each successful acquisition has at most one effective release, with repeated lifecycle calls safe.

### DesktopBindingInventory

| Field | Rule |
|---|---|
| `service` | one explicitly registered desktop service |
| `requiredMethods` | exact 25-operation allowlist in `contracts/desktop-bridge.md` |
| `authoredFacadeMethods` | 23 existing UI commands plus deliberate read-only runtime-status access used to close startup observability; `CopyDemo` remains absent |
| `internalBootstrap` | `GetRuntimeStatus` supports initialization and actionable startup presentation |
| `forbiddenMethods` | lifecycle, generic dispatch, raw filesystem/process/environment/browser managers, credentials, private player internals, public player procedures |
| `generatedDirectory` | `frontend/bindings` |
| `generationDigest` | complete inventory/content digest from two identical clean generations |

### EventSubscriptionState

| Field | Meaning |
|---|---|
| `eventName` | one of the exact four allowed names |
| `registered` | underlying Wails listener installed before snapshot resolution |
| `newerEventObserved` | per-field stale-snapshot guard |
| `snapshotPending` | cached `GetRuntimeStatus` is unresolved |
| `active` | callbacks may be delivered |
| `released` | underlying unsubscribe has been called exactly once |

State transition: `new` → `registered` → (`event-observed` or `snapshot-applied`) → `released`. Release may occur while the snapshot is pending or during callback execution; it suppresses every later delivery.

### PackageCandidate

| Field | Required value/evidence |
|---|---|
| `sourceCommit` | exact candidate Git SHA |
| `pinSet` | accepted `VersionPinSet` |
| `profile` | personal-use ad-hoc required; public release conditional |
| `target` | `darwin/arm64`, macOS 13+ |
| `appPath` | `build/bin/Fallout Terminal.app` |
| `resourceInventory` | master/player output, generated client, fonts, sounds, demo JSON, icon, plist, entitlements |
| `signature` | final ad-hoc identity for personal use, or real Developer ID evidence for public release |
| `executableDigest` | SHA-256 recorded after final mutation/signing |
| `verification` | architecture, metadata, resource, offline launch, listener, quit, and applicable trust gates |

Invariant: all resources are inserted before final signature; no accepted bundle is mutated afterward.

### RollbackReference

| Field | Rule |
|---|---|
| `sourceCommit` | canonical Wails v2 commit `f1084b3df8b5630862bdf7a0f347b599156653ef` |
| `sourceVerified` | commit exists and corresponds to accepted pre-migration `main` |
| `artifactPath` | optional; recorded only if really built and accepted |
| `artifactSHA256` | optional; never synthesized or prefilled |
| `dataCompatibility` | session-v1 and player-config-v1 need no conversion |
| `triggers` | corruption/loss, bridge drift, parity loss, leaks, public regression, missing resource, beta crash, package/signature failure |
| `record` | new `docs/wails-v3-migration-rollback.md`; historical Electron rollback remains separate |

### AcceptanceEvidence

| Field | Rule |
|---|---|
| `candidate` | exact source commit and version pin set |
| `gate` | uniquely named automated/manual/conditional gate |
| `commandOrProcedure` | reproducible command or bounded manual procedure |
| `result` | `PASS`, `FAIL`, or `NOT RUN` |
| `timestamp/environment` | enough provenance to distinguish candidates and machines |
| `artifact` | log, digest, inventory, screenshot, or observation where applicable |
| `reason` | required for `FAIL` and `NOT RUN` |

`NOT RUN` is not a pass. Credential-gated public ngrok and Developer ID/notary/DMG checks remain conditional and require real external evidence.

## Compatibility Boundary

No migration design record is written into user data or sent across the public player service. Generated Wails binding metadata stays tool-native. If implementation discovers a semantic need that cannot be represented by the preserved contracts, it must stop and produce a separate contract decision rather than silently extending these records into application schemas.
