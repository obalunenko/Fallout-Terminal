# Implementation Plan: Player Intelligence and Hacker Perk Management

**Feature**: `011-player-intelligence-hacker` | **Date**: 2026-08-19 | **Spec**: [spec.md](./spec.md)

## Summary

Extend each durable player roster entry with an Intelligence value from 1 through 10 and an explicit Hacker perk availability flag while keeping player-config JSON at version 1. The existing coordinator remains the authoritative state owner: it validates an expected revision and inactive-broadcast precondition, writes the complete candidate roster atomically, and only then publishes a new master snapshot.

Move durable roster detail and mutation controls from the crowded master sidebar into a dedicated in-page modal dialog that becomes read-only as soon as a broadcast starts. The private desktop bridge and master projection gain the new attributes, while the public player contract and hacking behavior remain unchanged.

## Project Structure

```text
specs/011-player-intelligence-hacker/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
└── contracts/
    ├── player-config-v1.md
    ├── private-desktop.md
    └── master-ui.md

proto/fallout/terminal/
├── persistence/v1/player_config.proto    # durable roster attributes and presence
└── private/v1/
    ├── coordination.proto                # master-only detailed roster projection
    └── desktop.proto                     # add/update/delete mutation inputs

proto/schema-revision.txt                 # reviewed additive schema revision
internal/gen/fallout/terminal/            # generated Go protobuf output only
├── persistence/v1/player_config.pb.go
└── private/v1/{coordination,desktop}.pb.go

internal/
├── domain/
│   ├── model.go                          # canonical roster and master projection fields
│   ├── json.go                           # strict legacy-aware player-config JSON adapter
│   ├── validate.go                       # Intelligence and profile validation
│   └── {model,validate}_test.go
├── playerconfig/
│   ├── contract.go                       # domain/protobuf adapter
│   ├── service.go                        # content-fingerprint conditional save
│   └── {contract,service}_test.go
└── control/
    ├── service.go                        # inactive/revision guards and atomic roster mutations
    └── service_test.go

app.go                                    # trusted payload validation and coordination commands
app_contract.go                           # explicit private protobuf adapters/projections
desktop_service.go                        # narrow generated Wails method allowlist
{app,app_contract}_test.go

frontend/
├── src/
│   ├── index.html                        # player-management and delete-confirmation dialogs
│   ├── master.js                         # authoritative popup render/mutation flow
│   ├── master.css                        # responsive modal roster layout
│   └── desktop-api.js                    # typed add/update/delete facade
└── bindings/                             # Wails-generated files; never edited manually

tests/browser/
├── player-management.spec.mjs            # popup, validation, read-only and persistence journeys
├── desktop-api.spec.mjs                  # private payload normalization
├── fixture-server/main.go                # deterministic player-config state/failure endpoints
└── fixtures/desktop-bindings.js          # generated-binding test double

sessions/demo-players.json                # canonical example with explicit attributes
scripts/wails-bindings-check.sh           # exact private method inventory if names change
```

**Structure Decision**: Keep durable values in the existing domain/player-config boundary, canonical mutation ordering in `internal/control`, private transport adaptation at the root desktop bridge, and presentation in the existing single master window.

## Constitution Check

| Principle | Before research | After design | Assessment |
|---|---|---|---|
| I. Govern the Accepted Desktop Runtime | PASS | PASS | The design keeps Wails in root composition and uses the existing single registered desktop service; domain, player-config, and control packages remain runtime-independent. |
| II. Make Protobuf the Application Contract Source of Truth | PASS | PASS | Durable known fields, private mutation inputs, and master projections are defined in versioned protobuf schemas before implementation. |
| III. Use ConnectRPC and Keep State Server-Authoritative | PASS | PASS | No public RPC changes; roster state remains Go-authoritative and the browser renders returned or event-delivered snapshots without optimistic mutation. |
| IV. Separate Public and Private Capabilities | PASS | PASS | Intelligence and Hacker availability remain master-only and do not enter public player descriptors or payloads. |
| V. Evolve Schemas Safely and Reproducibly | PASS | PASS | Changes are additive at unused field numbers, use presence where absence has legacy meaning, and are regenerated only through pinned tools. |
| VI. Preserve Portable Session JSON Version 1 | PASS | PASS | Session JSON is unchanged; player-config JSON stays version 1 with defined legacy defaults, strict invalid-value handling, conditional atomic saves, and an explicit rollback caveat. |
| VII. Complete Cutovers and Remove Superseded Protocols | PASS | PASS | The bundled master UI and private bridge cut over together; the inline durable-roster editor and live name-only Wails path are removed without introducing a second protocol or runtime. |

The pre-research and post-design gates pass with no constitutional violations. No Complexity Tracking table is required.

## Implementation Strategy

### Phase 1 — Schemas, compatibility, and canonical data

1. Add the persistence and private protobuf fields defined in [contracts/](./contracts/), retain all published field numbers, regenerate Go output with repository-owned tools, and update only the reviewed schema revision.
2. Extend canonical roster and master-projection models, then enforce Intelligence 1–10 for every canonical value. Use a strict presence-aware JSON adapter so absent legacy attributes become Intelligence 1 and Hacker unavailable, while explicit zero, null, fractional, wrong-typed, out-of-range, or unknown values remain invalid.
3. Keep player-config version 1 and its ordered complete-file representation. Encode both new attributes explicitly on every successful save and update the bundled player example.
4. Extend the active player-config handle with a digest of the exact loaded/saved content. Before replacement, compare the current file with that digest; missing, unreadable, or changed content returns a conflict and leaves runtime state untouched. Reuse the existing private-temp-file, fsync, and atomic-rename storage path after the comparison succeeds.

### Phase 2 — Authoritative roster mutation and private bridge

1. Replace name-only create/update inputs with complete profile payloads. Add and update require a nonblank name, valid Intelligence, an explicitly present Hacker choice, and the expected master coordination revision; delete also carries the expected revision.
2. Check the expected revision, active player-config identity, and `Broadcast == nil` inside the coordinator transaction before allocating an ID or writing a candidate. A first successful request advances the revision, so a retry or stale dialog submission is rejected without another write or event.
3. Preserve the current persistence-before-publication order: build one detached candidate, conditionally save the complete roster once, install the returned content digest, then commit and publish one new authoritative revision. Update preserves stable player ID and list position; failure preserves canonical state, revision, file, and effects.
4. Add Intelligence and Hacker availability to the private coordination projection and explicit root adapters. Replace the exposed `RenameCharacter` Wails method with `UpdateCharacter` while reusing the compatible existing private protobuf request container with additive fields; regenerate bindings and keep the allowlist count stable.

### Phase 3 — Dedicated master popup

1. Replace the inline durable-roster editor with a `btnManagePlayers` trigger and native HTML `dialog` in the existing master window. The dialog owns detailed rows, the empty state, required add controls, per-row atomic update/delete actions, local status, and local error presentation.
2. Render only from the latest coordination snapshot. Keep the dialog available during a broadcast, but immediately mark it read-only and disable all durable mutations when a broadcast event arrives; backend guards remain final for stale or crafted calls.
3. Preserve existing live assignment correction independently from durable roster editing. Keep assign/release/controller controls in logical-session management and relocate any claimed-character transfer control needed after the inline roster rows disappear.
4. Use a dedicated delete-confirmation dialog, restore focus to the invoking control, support Escape/close without a command, identify fields with accessible labels, and use text-only insertion for player-provided names.

### Phase 4 — Verification and cutover

1. Cover legacy and canonical JSON, strict presence/type/range validation, protobuf field presence and numbers, conditional-save conflicts, atomic write failure, inactive-broadcast authorization, expected-revision replay rejection, stable identity/order, and detached private projection.
2. Add Playwright journeys for populated and empty dialogs, Intelligence boundaries, required Hacker choice, add/update/delete, failure retention, close-without-change, reload persistence, live transition to read-only, stale calls, accessible names, and focus restoration.
3. Prove public/private separation by confirming player descriptors and projections still expose only player ID/name/availability status, not Intelligence or Hacker data. Review regenerated Go/Wails artifacts and remove the old inline durable-roster editor and exposed rename method in the same cutover.
4. Run all applicable protobuf, binding, Go race, frontend, browser, and owned build gates. Begin implementation on a dedicated feature branch from `develop`; do not add or repin dependencies.

## Contract and Compatibility Impact

| Surface | Producer | Consumers | Compatibility rule |
|---|---|---|---|
| Player-config JSON v1 | `internal/playerconfig` | Reopen/install and human-authored files | Legacy absence defaults at decode; new saves emit both fields; supplied invalid/unknown data is rejected. |
| Persistence protobuf | `player_config.proto` | Player-config contract adapter and private operation result | Add fields 3–4 only; no renumbering; optional presence distinguishes legacy absence. |
| Private coordination projection | `internal/control` through `app_contract.go` | Master bootstrap, named event, and dialog | Add master-only fields 4–5 to `CharacterState`; revision ordering is unchanged. |
| Private mutation requests | Master dialog through generated Wails service | Root adapter and coordinator | Full profile plus expected revision; explicit false is distinct from missing Hacker choice. |
| Public player state | Existing coordinator/player adapter | Player browser | No schema or payload change; new attributes remain undisclosed and have no gameplay effect. |
| Player-config file identity | Player-config service | Coordinator active handle | Exact content digest advances after each successful save; external change/missing file blocks replacement. |

## Verification Strategy

| Surface | Required evidence |
|---|---|
| Domain and JSON | Legacy absence, explicit canonical output, Intelligence 1/10 acceptance, 0/11/fraction/string/null rejection, Hacker missing default, explicit false/true, unknown/trailing rejection, stable order and IDs. |
| Player-config service | Protobuf presence round trip, create/open/save/reopen, content-digest advance, missing/replaced/unreadable conflict, atomic write failure retaining old content. |
| Coordinator | Add/update/delete inactive success; active-broadcast rejection for all three; stale/duplicate expected revision; one save and revision per accepted change; no save/effect/revision on rejection; update preserves identity/order. |
| Private desktop | Complete payload validation, exact adapter mapping, detached `CharacterState`, result/event authoritative state, generated method inventory, no generic bridge surface. |
| Master UI | Modal open/close/empty/detail, required controls, bounds and explicit Hacker choice, delete confirmation, failure retention, event-driven read-only conversion, stale request refusal, assignment controls retained outside the durable editor. |
| Capability separation | Public player descriptors, snapshots, selection screen, and RPCs contain no Intelligence or Hacker fields; hacking behavior and authorization remain unchanged. |

Applicable commands:

```bash
scripts/proto-generate.sh --sync-revision
scripts/proto-check.sh
scripts/proto-breaking.sh --all-fixtures
go tool -modfile=tools/wails/go.mod wails3 generate bindings -clean ./...
scripts/wails-bindings-check.sh
gofmt -l .
go vet ./...
go test ./internal/domain ./internal/playerconfig ./internal/control .
go test -race ./...
npm ci --prefix client
npm ci --prefix frontend
npm run build --prefix frontend
npm ci --prefix tests/browser
npm test --prefix tests/browser -- desktop-api.spec.mjs player-management.spec.mjs
npm test --prefix tests/browser
go run ./cmd/build build
make check
```

`go run ./cmd/build dev` supplies the final interactive master acceptance journey. Packaging is not required because the feature does not alter native composition or packaging, but the normal build and deterministic Wails-binding gates remain required.

## Rollback and Cutover

The implementation is one bundled cutover: schemas, adapters, coordinator, generated bindings, and master assets ship together. Rollback restores the prior application, but its strict reader cannot consume player-config files already saved with the additive fields. Operational rollback therefore also requires restoring a user-held pre-feature copy or removing `intelligence` and `hackerPerkAvailable` from each entry; the remaining version-1 shape and values are unchanged.
