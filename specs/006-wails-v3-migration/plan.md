# Implementation Plan: Wails v3 Runtime Migration

**Bugfix**: 2026-08-14 — [ANALYZE-2026-08-14] Updated the design for isolated Go tool modules, the unchanged runtime-status schema, measurable soak gates, and dependency-ordered P1 acceptance.
**Bugfix**: 2026-08-14 — [ANALYZE-CUTOVER-2026-08-14] Ordered rollback and soak before v2 removal, protected historical artifacts, and specified deterministic local/public soak procedures.
**Bugfix**: 2026-08-14 — [ANALYZE-FINAL-MATRIX-2026-08-14] Made the post-removal verification matrix sequential, restartable, and solely responsible for final acceptance.
**Bugfix**: 2026-08-14 — [ANALYZE-FINAL-CANDIDATE-2026-08-14] Added clean browser-test installation, final-candidate native/soak reruns, and non-self-referential evidence identity.
**Bugfix**: 2026-08-14 — [ANALYZE-ATTRIBUTABLE-CANDIDATE-2026-08-14] Required a clean committed build candidate and separated final native, package, and soak attribution.
**Bugfix**: 2026-08-14 — [ANALYZE-BUNDLE-IDENTITY-2026-08-14] Defined a canonical app-bundle digest and made final active documentation verification-only.

**Branch**: `006-wails-v3-migration` | **Date**: 2026-08-13 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/006-wails-v3-migration/spec.md`
**Planning baseline**: fetched `main` / `origin/main` at `f1084b3df8b5630862bdf7a0f347b599156653ef`

## Summary

Replace only the Wails v2 desktop host, binding/event integration, build model, and macOS packaging path with a verified exact Wails v3 beta.8 set. Preserve the single master window, runtime-neutral facade, exact 25-operation trusted bridge, four event names/shapes, application-owned startup observability, feature-005 protobuf/ConnectRPC/session contracts, both Vite applications, macOS 13+ arm64 personal-use package, and the existing `build/bin/Fallout Terminal.app` location.

The implementation is a checkpointed brownfield migration rather than a one-commit rewrite. Root composition creates the Wails application first, injects a narrow host capability into platform adapters, composes the unchanged core, registers an unbound lifecycle service plus an allowlisted desktop service, registers typed events, creates one accepted window, and runs. A standard-library-only Go command owns one nonrecursive generation/build/package graph; the pinned Wails CLI is used only for binding generation. The Wails v2 baseline remains immutable rollback authority until parity and cutover scans pass.

## Technical Context

**Language/Version**: Go 1.26; browser ECMAScript modules with Node.js 20.19+ build/test tooling
**Primary Dependencies**: `github.com/wailsapp/wails/v3` `v3.0.0-beta.8`; `@wailsio/runtime` `3.0.0-beta.8` and its `plugins/vite` subpath; ConnectRPC Go `v1.20.0`; protobuf Go `v1.36.11`; Buf `v1.72.0`; exact Vite `8.1.5`; Playwright `1.62.1`
**Storage**: Unchanged version-1 local JSON session and player-configuration files; bundled read-only demo; ephemeral live/navigation/hacking/connection/coordination state in memory
**Testing**: Colocated table-driven Go tests with Testify, `t.Context()`, protobuf-aware comparison; direct JavaScript facade tests; existing Buf drift/breaking/negative/isolation gates; Playwright generated-player journeys; macOS bundle/signature inspection and manual smoke/soak
**Target Platform**: macOS 13+ Apple Silicon (`darwin/arm64`) personal use with an ad-hoc-signed app; conditional credential-backed public release
**Project Type**: Go modular desktop monolith with one Wails master frontend and a separately embedded/served ConnectRPC browser-player frontend
**Performance Goals**: Preserve bounded 30-second application startup, 20-second optional tunnel startup, five-second shutdown, two-second tunnel graceful escalation, current revision/replay limits, responsive 4–7-player operation, and no build-generation races
**Constraints**: Exact compatible prerelease pins; no `latest`/floating Wails configuration; one master window; exact bridge/event contracts; no Wails in domain/player packages; no data/protocol redesign; no runtime CDN/download; complete bundle before final signature; no permanent dual runtime
**Scale/Scope**: One game-master process/window, one player listener, zero or one owned tunnel, one active broadcast, four to seven representative browser players, approximately 35 affected files and a bounded series of attributable checkpoints

## Constitution Check

*Gate result before research and after design: PASS. No exception is required.*

| Principle / rule | Plan evidence | Result |
|---|---|---|
| I. Govern the accepted desktop runtime | Wails objects stay in root composition and `internal/platform`; dedicated lifecycle/desktop adapters; one window | PASS |
| II. Protobuf is application contract source | Existing private/public/config/persistence messages and explicit adapters remain; Wails tool metadata is correctly excluded | PASS |
| III. ConnectRPC/server authority | Public `PlayerService`, generated client, canonical state, replay, limits, and reconnect semantics are unchanged | PASS |
| IV. Public/private separation | Exact allowlisted desktop service and four events; player listener exposes neither Wails nor native capability | PASS |
| V. Safe/reproducible schemas | No schema delta; existing deterministic generation, compatibility, negative, and graph-isolation gates remain | PASS |
| VI. Portable session JSON v1 | No persistence change; paths, references, extras, strict player-config decode, saves, and demo-copy behavior preserved | PASS |
| VII. Complete cutover | Immutable v2 rollback, branch-bounded coexistence, explicit removal gate, no permanent switch | PASS |
| Dependency and tool-module rules | Exact beta.8 runtime/npm pins; every Go development tool has one isolated `tools/<tool>/` module; repository commands use `go tool -modfile`; root `go.mod` remains application-only; Wails imports stay within allowed boundaries | PASS |
| Testing conventions | Testify, tables, `t.Context()`, protobuf comparison, gofmt/vet/test/race, clean locked browser-test installation, and full browser/package gates planned | PASS |
| Evidence integrity | Required and conditional profiles are separated; unavailable gates record `NOT RUN` | PASS |

## Project Structure

### Documentation for this feature

```text
specs/006-wails-v3-migration/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── desktop-bridge.md
│   ├── desktop-events.md
│   ├── lifecycle.md
│   └── build-package.md
└── tasks.md                       # generated only after plan review
```

### Affected source and delivery surfaces

```text
main.go                             # Wails v3 application, assets, service/event/window composition
app.go                              # transport-neutral core lifecycle and accepted operations
app_contract.go                     # unchanged private protobuf ↔ native DTO adapters
go.mod / go.sum                     # exact Wails v3 module; remove v2 at cutover
tools/                              # isolated Go development-tool modules; never imported by the root app
├── wails/go.mod / go.sum           # only wails3; exact v3.0.0-beta.8 parent-module pin
├── buf/go.mod / go.sum             # only buf; exact v1.72.0 parent-module pin
├── protoc-gen-go/go.mod / go.sum   # only protoc-gen-go; exact v1.36.11 parent-module pin
└── protoc-gen-connect-go/          # only protoc-gen-connect-go; exact v1.20.0 parent-module pin
    └── go.mod / go.sum
cmd/build/main.go                   # sole root development/build/package entry
internal/buildtool/                 # standard-library-only graph and tests
build/
├── darwin/Info*.plist
├── darwin/entitlements.plist
└── appicon.png
internal/platform/
├── desktop.go                      # injected v3 dialogs/browser/event host adapter
├── paths.go                        # preserve Documents/Application Support/bundle resource semantics
├── startup_test.go                 # Go build graph/root-command assertions
└── assets_test.go                  # embedding/facade/resources/forbidden-surface assertions
frontend/
├── bindings/                       # one explicit Wails v3 generated directory
├── src/desktop-api.js              # only generated/runtime consumer; facade and event readiness
├── src/master.js                   # current UI plus actionable existing startupError presentation
├── package.json / package-lock.json
└── vite.config.js                  # exact official Wails Vite plugin
client/                             # unchanged generated ConnectRPC player app; still clean-built
proto/ and internal/gen/            # unchanged contracts/generated Go; existing gates retained
tests/browser/                      # unchanged or deliberately adapted player parity journeys
scripts/build-macos.sh              # isolated pinned wails3 release flow, existing trust controls retained
.github/workflows/wails-macos.yml   # exact pins, generations, scans, package inspection
README.md                           # active v3 setup/dev/build/package instructions
docs/
├── wails-migration-rollback.md     # preserved historical Electron→Wails v2 record
└── wails-v3-migration-rollback.md  # new v2→v3 rollback/cutover evidence
wails.json                          # retained only during branch migration; removed at cutover
production_resources_bindings.go   # remove only after v3 proof, otherwise document narrowly
```

**Structure decision**: The migration remains at existing composition/platform/front-end/build boundaries. It introduces a small Wails host/service adapter rather than moving Wails into domain, persistence, control, tunnel, or player packages. `client/`, protobuf schemas, and data codecs are verification-sensitive consumers but have no planned semantic change.

## Architecture and Contract Mapping

### Host composition

```text
application.New(options + master assets)
                │
                ├──► narrow platform host (events/dialogs/browser/lifetime)
                │                 │
                │                 ▼
                └──► compose existing Fallout Terminal core
                                  │
                    ┌─────────────┴─────────────┐
                    ▼                           ▼
          lifecycle service            allowlisted desktop service
          (never frontend-bound)        (exact 25 methods)
                    └─────────────┬─────────────┘
                                  ▼
                    register four typed events
                                  ▼
                    create one accepted window
                                  ▼
                              app.Run()
```

The construction cycle is resolved through late service registration and injected interfaces. The existing late-bound coordination effect router remains explicit. Package globals are prohibited.

### Persistent JSON

No data-model or schema change. Preserve session-v1/player-config-v1 formats, validation, relative references, compatible extras where governed, strict player-config rejection, explicit paths, ordered revisions, atomic replacement, default locations, and demo-copy behavior. No migration or conversion is required for rollback.

### Desktop bridge and events

- One generated desktop service contains exactly the 25 methods in [desktop-bridge.md](./contracts/desktop-bridge.md).
- `Start`, `Shutdown`, generic dispatch, raw native primitives, credentials, player procedures, and lifecycle helpers remain absent.
- `CopyDemo` remains bound but has no authored facade control. `GetRuntimeStatus` remains bootstrap/status access.
- The unchanged runtime-neutral facade is the only master consumer; generated modules live at `frontend/bindings`.
- Four events retain exact names and payload shapes. Listeners register before one cached status snapshot; per-field newer events suppress stale snapshot fields; unsubscribe is exact-once and idempotent.
- The existing `startupError` is deliberately rendered by the master through the facade. No protobuf `phase` field is introduced; lifecycle phase stays internal/test-observable.

### Public ConnectRPC and player application

No procedure, schema, cardinality, transport, route, projection, privacy, or behavior change. The player listener continues to serve only bundled public assets and generated `fallout.terminal.player.v1.PlayerService`. Feature-005 tests are migration gates, not targets for weakening. Wails streams and the removed WebSocket protocol remain out of scope.

### Runtime lifecycle

The lifecycle service maps Wails startup into the existing bounded core startup. Handled application failures record existing status, unwind partial acquisition, and do not return an aborting `ServiceStartup` error when a safe master UI can run. A stable application-lifetime operation context replaces the v2 retained startup/DomReady context pattern. Shutdown uses a fresh bounded background context and preserves tunnel → player → session worker → desktop cleanup.

### Platform and packaging

`internal/platform` wraps Wails v3 managers for dialogs, browser, and event emission while preserving all defaults/cancel/validation semantics. The repository Go command replaces active `wails.json` orchestration; its graph builds player, generates bindings, builds master, packages, copies all resources, then signs. Output remains `build/bin/Fallout Terminal.app`.

## Bounded Migration Checkpoints

Each checkpoint must leave an attributable test/evidence boundary. Do not collapse these into one commit.

### Checkpoint 0 — Rollback, pins, and build skeleton

- Capture/verify the immutable v2 source rollback in the new migration rollback record before removing active v2 paths.
- Add the exact beta.8 Go/npm application pins and the independently pinned beta.8 CLI under `tools/wails/`; commit every application and tool lock change; add automated pin consistency, root-module-drift, and no-floating/global-tool scans.
- Introduce `cmd/build`, a standard-library-only ordered graph, Darwin build assets, ownership comments/tests, and preserved `build/bin` output.
- Establish one nonrecursive graph and direct locked frontend checks while v2 remains the production fallback.
- Checkpoint gate: exact pins, clean tool preflight, Go build graph/source assertions, no unexplained lock drift; rollback source verified.

### Checkpoint 1 — Go host, lifecycle, and platform adapters

- Create the Wails application first, inject the narrow platform host, compose core, register lifecycle/desktop services, typed events, and one window.
- Split allowlisted desktop forwarding from unbound lifecycle methods; do not register the composition root.
- Replace retained v2 contexts/global runtime calls with application-lifetime event/dialog/browser adapters.
- Preserve bounded startup/status/local fallback/reverse unwind and fresh-context bounded shutdown; fix acquired-before-validation tunnel ownership.
- Classify pre-window failures into truly host-fatal versus master-visible application failure.
- Checkpoint gate: Go lifecycle/platform unit and host integration tests, exact service reflection inventory, startup/shutdown trigger matrix, no Wails imports outside root/platform.

### Checkpoint 2 — Generated bindings, facade, events, and master readiness

- Generate explicit v3 bindings into `frontend/bindings` before master Vite.
- Replace v2 generated lookup and globals with explicit modules and pinned runtime/plugin behind `desktop-api.js`.
- Implement four-listener readiness barrier, event `.data` unwrapping, cached snapshot, per-field stale suppression, and exact-once disposal.
- Render the existing actionable `startupError` without changing private protobuf shapes.
- Add behavioral facade tests and production fallback/bundle scans.
- Checkpoint gate: two identical clean generations, exact 25/forbidden inventory, direct master/player builds, event race suite, no `window.go`/`window.runtime`/v2 binding path.

### Checkpoint 3 — Complete v3 build graph and personal-use package

- Make protobuf verification, player build, binding generation, and master build explicit nonrecursive prerequisites for dev/build/package.
- Replace v2 post-build resources with pre-sign bundle assembly; preserve resource-root behavior.
- Package macOS 13 arm64 at the established path and inspect all assets/metadata/entitlements/icon/signature.
- Add and test one canonical bundle-manifest SHA-256 procedure using byte-sorted bundle-relative paths, entry type, POSIX mode, regular-file content digest, and symlink target while excluding timestamps and host-specific extended attributes.
- Prove one offline launch, one listener, representative master/local-player smoke, and clean quit.
- Resolve the `production_resources_bindings.go` workaround only from clean evidence.
- Checkpoint gate: clean repeated builds, final resource inventory, ad-hoc signature, offline personal-use acceptance.

### Checkpoint 4 — CI, release automation, active docs, soak, and rollback drill

- Adapt `.github/workflows/wails-macos.yml` to isolated exact-pinned Wails generation/build/package and all current gates.
- Adapt the proven macOS release script without weakening signing/notary/DMG/Gatekeeper controls.
- Update active README/quickstart commands and links to the v3 guide while treating completed specs and the Electron rollback record as immutable historical evidence.
- Create new v3 rollback record, perform safety-copy/source rollback drill, and record only real evidence.
- Run a required pre-removal 60-minute qualification soak with four to seven players, at least 25 mixed operations, three reconnects, two save/reopen cycles, navigation, hacking, coordination, sound, convergence/revision checks, one listener, and five-second post-quit cleanup. At minutes 15, 30, and 60, collect five RSS samples ten seconds apart with `ps -o rss= -p <APP_PID>`, record each median KiB value, and fail when both later medians exceed 125% of the 15-minute median. This qualification gate permits v2 removal but does not replace the same workload against the final v2-free build candidate.
- With real access, run the pre-removal 30-minute authenticated-ngrok qualification soak with four to seven players, at least 15 mixed public operations, two reconnects, one unauthorized rejection, controlled tunnel loss with local fallback, per-operation convergence, credential/private-field isolation, and final cleanup within five seconds; otherwise record `NOT RUN`. This does not replace the conditional rerun against the final v2-free build candidate.
- Checkpoint gate: CI graph and scripts pass applicable profile; soak and rollback evidence tied to exact candidate.

### Checkpoint 5 — Parity decision and irreversible source cutover

- Reconcile all required acceptance evidence for the personal-use profile; no required gate may be `FAIL` or `NOT RUN`.
- Remove Wails v2 imports/dependency/CLI/config/hooks, v2 generated/global assumptions, obsolete workaround if proven, active v2 instructions, and every temporary dual path.
- Commit all prior qualification evidence separately; complete and commit `README.md` plus every active operating document together with the v2-free application/configuration source; require a clean working tree; and freeze that commit as the immutable **build candidate SHA** with its exact pin set without accepting it. Later evidence-only documentation commits record that SHA and package digest but do not redefine the packaged candidate.
- Sequentially verify tool/protobuf state, Wails bindings, both frontends/reproducible builds, Go tests, race tests, a clean locked Playwright installation and suite, root-development/native parity, final package/offline smoke, the required final-candidate local soak, conditional public gates, and reconciled documentation/evidence as separate attributable boundaries.
- If any verification task changes a frozen tracked path, including application source, generated output, active operating documentation, a pin, or a lockfile, invalidate later evidence, commit and freeze a new build candidate SHA, and restart the final matrix from its first task. Evidence-only edits are limited to result/status fields in the feature quickstart and Wails v3 rollback record; they require a final documentation/cutover rescan but do not rebuild an otherwise unchanged package.
- Checkpoint gate: zero active v2 runtime surface, zero permanent switch, all required parity gates pass; only then designate Wails v3 production.

## Workstream Attribution

| Failure class | Primary checkpoint/files | Diagnostic boundary |
|---|---|---|
| version/tool/template incompatibility | CP0; manifests, locks, Go build graph/assets | exact pin probe, clean generate/build skeleton |
| core startup/shutdown/platform regression | CP1; `main.go`, lifecycle/service adapter, `internal/platform` | Go unit/host integration trigger matrix |
| generated surface/readiness/payload drift | CP2; `frontend/bindings`, `desktop-api.js`, Vite config | deterministic inventory + direct JS facade/event tests |
| assets/output/signing regression | CP3; build tasks, resource root, plist/entitlements | final bundle inventory/signature/offline smoke |
| automation/docs/rollback/release regression | CP4–5; CI, scripts, README/docs | clean CI, conditional evidence, rollback drill, final scans |

## Verification Plan

| Surface | Automated checks | Manual/conditional checks | Acceptance |
|---|---|---|---|
| Go formatting/quality | `gofmt -l .`; `go vet ./...`; `go test ./...`; `go test -race ./...` | N/A | no formatted paths; all commands pass with governed test conventions |
| Protobuf/contracts | existing Buf format/lint, `scripts/proto-check.sh`, two-generation drift, compatibility baseline, all negative fixtures, public/private graph isolation | inspect any intentional adapter update | no schema drift/unclassified fields; feature-005 contracts unchanged |
| Wails bindings | two clean `go tool -modfile=tools/wails/go.mod wails3 generate bindings` runs and content/inventory diff; service reflection/source scans | inspect generated service module | all 25 required; zero lifecycle/generic/native/player methods; second generation identical |
| Private bridge | table-driven Go adapter/service tests; direct JS facade call/result/error/cancel tests | representative file/session/URL journeys | exact native semantics, redaction, CopyDemo trust boundary, startup status visible |
| Events/readiness | Go payload/detachment tests; JS four-listener/snapshot/race/unsubscribe/disposal tests | launch timing/failure presentation | exact four names/shapes; no stale overwrite, duplicate effect, false readiness, or late callback |
| Dialog/browser | injected platform matrix for titles, filters, directories, filenames, aliases, create policy, cancel/errors, URL schemes | native open/save/cancel/HTTP(S) smoke | baseline semantics or explicitly recorded unavoidable difference |
| Frontends | `npm ci` and production build for both `frontend/` and `client/`; bundle scans | visual master/player parity | only v3 master binding path; no CDN/runtime download; player independent of Wails |
| Player | clean `npm ci --prefix tests/browser` followed by all feature-005 Go/Playwright gates: 4–7 clients, reconnect, replay, concurrency, overflow, sound manifest/playback, privacy | local journeys; credential-gated public journey | zero protocol/behavior/privacy regression; public unavailable evidence is `NOT RUN` |
| Startup/shutdown | lifecycle unit/host integration, occupied port, adapter/event failure, tunnel fallback/invalid URL, partial/repeat/timeout triggers | close, Cmd+Q, dev interrupt | failure actionable; local fallback; reverse cleanup within 5s; no leaks |
| Build graph | clean Go-plan source assertions, protobuf→player→bindings→master ordering, two clean native builds | `go run ./cmd/build dev` from clean setup | one root dev command; no recursion/race/stale output/floating lookup |
| Personal package | `go run ./cmd/build package`; architecture/plist/minimum/entitlements/icon/resource/signature scans; canonical bundle-manifest digest | offline single launch, one listener, master/player smoke, clean quit | final ad-hoc signed macOS 13 arm64 app at established path with stable before/after bundle identity |
| Public release | release-script preflight and credential-backed sign/notary/staple/DMG/Gatekeeper only when available | public ngrok soak and installed package check | real evidence required; otherwise explicitly `NOT RUN` |
| Soak/rollback | source/hash/data-safety assertions; v2/v3 forbidden scans | pre-removal qualification and final v2-free-candidate 60-minute local workloads with median-RSS sampling; conditional final-candidate 30-minute authenticated-ngrok workload with authorization/fallback checks; rollback drill using safety copies | rollback and qualification soak pass before v2 removal; the same required local workload passes against the final build candidate; exact thresholds and actual results are tied to the applicable candidate; no simulated pass |
| Cutover | source/generated/dependency/bundle/CI/script/docs scans plus full rebuild | owner parity review | zero active v2 runtime/dual switch; historical records intact |

## Acceptance Evidence Discipline

- [quickstart.md](./quickstart.md) begins with every evidence box unchecked.
- Every result records exact source commit, version pin set, target/profile, command/procedure, timestamp/environment, and `PASS`, `FAIL`, or `NOT RUN`.
- **Build candidate SHA**: `git rev-parse HEAD` captured only after the complete v2-free application/configuration source, `README.md`, and every active operating document are committed and `git status --porcelain` is empty. It identifies the immutable inputs used for generation, tests, packaging, native journeys, and soak.
- **Personal-use bundle identity**: the canonical bundle-manifest SHA-256 produced by `scripts/hash-macos-app.sh`, whose byte-sorted manifest includes each bundle-relative entry's type, POSIX mode, regular-file content digest, or symlink target and excludes timestamps and host-specific extended attributes. The required local and conditional public soaks verify that identity before and after exercising the app.
- **Release artifact identity**: a distinct canonical bundle-manifest digest for each Developer ID app and a distinct file SHA-256 for each DMG, always recorded with its profile.
- **Evidence commit**: documentation-only changes to result/status fields in the feature quickstart and Wails v3 rollback record. It may record the build candidate and artifact identities but cannot change them. Before each final gate, every other tracked path must remain identical to the build candidate.
- A version change invalidates generated, build, package, and acceptance evidence until research/pins/locks/CI/scripts are updated and gates rerun.
- Any frozen tracked-path change, including active documentation, restarts the final matrix. An allowlisted evidence-only documentation change requires the final active-documentation and cutover scan but does not rebuild the unchanged candidate.
- A v2 artifact digest is recorded only for an artifact actually built and accepted.
- Developer ID, notarization, stapling, DMG, Gatekeeper, and public ngrok evidence cannot be inferred from personal-use checks.

## Post-Design Constitution Recheck

PASS after the 2026-08-14 reconciliation. The contracts keep Wails at governed boundaries, preserve protobuf/ConnectRPC/session authority, define public/private exposure, use isolated exactly pinned tool modules without polluting the root application module, keep runtime state out of persistence, provide bounded coexistence/removal, and retain all required quality gates. No complexity exception is introduced.

## Complexity Tracking

No constitution violation requires justification. The separate lifecycle and desktop services, narrow platform capability, explicit event readiness state, and dependency-free Go build graph are the minimum structures needed to enforce existing ownership and exposure rules.
