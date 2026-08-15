# Implementation Plan: Embedded ngrok Public Access

**Branch/Feature**: `007-embedded-ngrok-access`  
**Specification**: `specs/007-embedded-ngrok-access/spec.md`  
**Target**: macOS 13+ Apple Silicon (`arm64`)  
**Plan date**: 2026-08-15

**Bugfix**: 2026-08-15 — ANALYZE-S1/U1 adds a trusted source-bound public-ingress discriminator and
an executable user-approved checkpoint/worktree rollback gate.

**Bugfix**: 2026-08-15 — BUG-001 supersedes the source-bound discriminator after the target Darwin
host proved `127.0.0.2` unavailable. Personal-game public admission moves to ngrok endpoint Basic
Auth; the SDK forwards directly to the existing `127.0.0.1:3690` service.

**Bugfix**: 2026-08-15 — BUG-001 follow-up makes startup sampling, task-owned RED/GREEN gates, and
the exact final constitution command sequence executable.

**Bugfix**: 2026-08-15 — BUG-001 analysis follow-up makes the final sequence clean-checkout safe,
adds the canonical bounded dev smoke, and assigns vulnerability-review evidence.

**Bugfix**: 2026-08-16 — BUG-002 adds an explicit startup-to-endpoint context ownership handoff and
safe provider disconnect-code propagation after the embedded SDK forwarder proved dependent on the
context passed to `Forward`.

## Summary

Replace the startup-only external ngrok CLI path with one UI-controlled embedded ngrok runtime that
forwards to the existing player server and remains optional to local/LAN play. The implementation
adds a generation-aware lifecycle manager, an ngrok endpoint Basic Auth Traffic Policy, native
macOS Keychain storage, atomic non-secret Application Support settings, and strictly typed private
protobuf/Wails operations. The official `golang.ngrok.com/ngrok/v2` SDK is pinned at `v2.1.4` and
the native Security.framework wrapper `github.com/keybase/go-keychain` at `v0.0.1`; the CLI process,
PATH/config/parser runtime is removed only after SDK parity tests pass.

## Project Structure

```text
.
├── go.mod / go.sum                         # exact SDK, Keychain, and transitive module graph
├── main.go                                 # compose stores, manager, and SDK adapter
├── app.go                                  # private commands/events and shutdown ordering
├── app_contract.go                         # protobuf ↔ native desktop adapters
├── app_contract_test.go                    # descriptor, redaction, method/event allowlists
├── desktop_service.go                      # five explicit public-access master methods
├── wails_host.go / wails_host_test.go      # typed event registration and 5s shutdown owner
├── internal/
│   ├── tunnel/
│   │   ├── service.go                      # provider-neutral TunnelService/Endpoint boundary
│   │   ├── manager.go                      # generation/revision state machine
│   │   ├── settings.go                     # versioned atomic non-secret settings store
│   │   ├── secret.go                       # narrow SecretStore contract and secret refs
│   │   ├── ngrok.go                        # pinned official SDK adapter
│   │   ├── *_test.go                       # fake, lifecycle, race, redaction, settings tests
│   │   ├── config.go                       # delete CLI/env parsing or reduce to non-production seam
│   │   ├── process*.go                     # DELETE after embedded parity
│   │   ├── process_test.go                 # DELETE after lifecycle guarantees migrate
│   │   └── process_darwin_integration_test.go # DELETE after packaged cleanup parity
│   ├── player/
│   │   ├── http.go / http_test.go          # unchanged static/Connect application routing
│   │   └── server.go / server_test.go       # one authoritative listener on port 3690
│   ├── platform/
│   │   ├── keychain_darwin.go              # native production SecretStore adapter
│   │   ├── keychain.go                     # safe construction/error mapping
│   │   ├── keychain_test.go                # deterministic adapter/error tests
│   │   ├── paths.go / paths_test.go        # public-access settings path
│   │   ├── assets_test.go                   # master UI/contract/security asset checks
│   │   └── test_conventions_test.go         # register new Go test surfaces
│   └── gen/fallout/terminal/{config,private}/v1/
│                                               # regenerated Go contracts
├── proto/fallout/terminal/
│   ├── config/v1/
│   │   ├── config.proto                     # reserve removed CLI/plaintext tunnel fields
│   │   └── public_access.proto              # persisted secret-free preferences source
│   └── private/v1/public_access.proto       # master settings/status/commands/event contracts
├── proto/compatibility-baseline.binpb       # reviewed explicit internal-config migration baseline
├── proto/schema-revision.txt                # attributable schema revision
├── frontend/
│   ├── src/index.html                       # accessible Fallout-style settings section/dialog
│   ├── src/master.css                       # loading/empty/error/ready/transition presentation
│   ├── src/master.js                        # UI state, transient secret inputs, Copy feedback
│   ├── src/desktop-api.js                   # explicit methods/event and stale snapshot protection
│   └── bindings/                            # regenerated Wails v3 allowlisted bindings/events
├── tests/browser/
│   ├── fixtures/desktop-bindings.js         # explicit private test facade
│   ├── desktop-api.spec.mjs                 # snapshot/event/secret non-retention behavior
│   └── connectrpc-player.spec.mjs           # auth, streaming, reconnect, real opt-in endpoint
├── scripts/
│   ├── proto-*.sh                           # preserved format/lint/drift/breaking gates
│   ├── wails-bindings-check.sh              # updated exact method/event allowlist
│   ├── secret-leak-check.sh                 # canary scan over prohibited surfaces/artifacts
│   ├── dependency-license-check.sh          # pinned graph/license inventory check
│   ├── reproducible-build-check.sh          # unchanged two-build ownership, expanded inputs
│   └── verify-macos-app.sh                  # no CLI/PATH binary plus packaged/offline assertions
├── .github/workflows/wails-macos.yml         # run deterministic gates; real service remains opt-in
├── THIRD_PARTY_NOTICES.md                    # reviewed notices for new runtime graph
├── README.md                                 # packaged UI workflow; remove CLI/PATH/default-domain docs
└── .specify/templates/{plan,spec,tasks}-template.md
                                                # provider-neutral active lifecycle wording only
```

**Structure Decision**: Keep public-access policy and provider lifecycle in `internal/tunnel`, place
Keychain only in `internal/platform`, and route all UI behavior through the one existing desktop
service. The player server remains unchanged; do not create a new server, frontend, root command,
or production provider path.

## Constitution Check

| Constitution gate | Pre-design | Post-design assessment |
|---|---|---|
| I. Govern the Accepted Desktop Runtime | PASS | PASS — Wails v3 remains only composition/transport; tunnel core has no Wails/domain dependency and shutdown releases policy/SDK work within the existing budget. |
| II. Make Protobuf the Application Contract Source of Truth | PASS | PASS — new preferences, private inputs/results/status/event are versioned protobuf contracts; only the two approved narrow payloads can carry a new secret. |
| III. Use ConnectRPC and Keep State Server-Authoritative | PASS | PASS — the existing generated player service and one server remain unchanged; endpoint admission precedes forwarding and does not create client-owned state. |
| IV. Separate Public and Private Capabilities | PASS | PASS — only the master bridge mutates settings/secrets; player descriptors stay secret-free and the endpoint policy is never exposed as a player capability. |
| V. Evolve Schemas Safely and Reproducibly | PASS | PASS — new fields use versioned packages/enums/oneofs; removal of internal CLI config is explicitly documented, reserves fields, regenerates deterministically, and updates—not disables—the reviewed baseline. |
| VI. Preserve Portable Session JSON Version 1 | PASS | PASS — settings use a separate Application Support file; session and player-config v1 schemas/adapters/fixtures remain unchanged. |
| VII. Complete Cutovers and Remove Superseded Protocols | PASS | PASS — SDK parity precedes deletion of every CLI/process mechanism; no permanent dual runtime or fallback remains. |
| Dependency Rules | PASS | PASS — SDK and Keychain are exact root runtime pins; SDK stays behind tunnel adapter, Security.framework behind platform adapter, and core/domain packages import neither. |
| Secret and Credential Governance | PASS | PASS — Keychain-only persistence, no readback, scoped use, one-time generated result, redacted failures, and canary leak tests are explicit. |
| Go Development Tool Modules | PASS | PASS — no global tool or Taskfile is introduced; any future license tool would require its own exact `tools/<tool>/` module, while this plan uses repository scripts. |
| Testing and Quality Gates | PASS | PASS — unit/race/integration/browser/drift/repro/package gates remain and expand; real external evidence is opt-in and honestly `NOT RUN`. |
| Development Workflow / build ownership | PASS | PASS — `cmd/build`/`internal/buildtool` remain graph owners; direct `go run ./cmd/build ...` stays canonical and Make remains optional/thin. |

There are no constitution violations and no Complexity Tracking exception.

## Technical Context

**Runtime**: Go 1.26 modular monolith, exactly pinned Wails v3.0.0-beta.8, plain Vite/JavaScript
master and player frontends  
**New runtime dependencies**: `golang.ngrok.com/ngrok/v2 v2.1.4`,
`github.com/keybase/go-keychain v0.0.1`  
**Network**: existing `0.0.0.0:3690` HTTP/ConnectRPC player listener; embedded SDK forwards only to
`http://127.0.0.1:3690`  
**Persistence**: macOS Keychain for two secrets; version-1 atomic JSON in the existing Application
Support directory for non-secret preferences  
**Startup target**: ~~95% ready/terminal-error within 15s on responsive network; every attempt
bounded to 30s~~ **BUG-001 follow-up**: at least 19 of 20 explicitly opted-in real starts under
declared responsive-network and valid-account prerequisites reach ready/terminal-error within 15s,
and all finish within 30s; otherwise record the real sample `NOT RUN`. A separate 100-schedule
deterministic timeout/cancellation gate is not provider-performance evidence.

**Shutdown target**: ~~policy deny plus all owned cleanup~~ **BUG-001**: URL withdrawal plus
endpoint/Agent and all owned cleanup within the existing single 5s application budget
**Scale**: one master window, one player server, zero/one endpoint, four to seven representative
players, concurrent start/stop/reconfigure schedules  
**Conditional gates**: real ngrok/account/domain, Developer ID, notarization, stapling, DMG,
Gatekeeper, and provider-plan features

## Phase 0: Research

The decisions and rejected alternatives are recorded in [research.md](research.md). Load-bearing
conclusions are:

- the selected SDK pin supports explicit Agent connect, `Forward` to the existing loopback server,
  URL retrieval, terminal `Done`, and bounded explicit close;
- cancellation must trigger explicit endpoint close/disconnect rather than being treated as cleanup;
- ~~application Basic Auth remains at the player HTTP boundary until separate real streaming
  evidence supports any edge-policy move~~ **BUG-001**: Basic Auth is configured on the ngrok Agent
  Endpoint because the feature's purpose is casual sharing and direct local/LAN traffic must remain
  unchanged; a focused opt-in non-empty `Subscribe` is retained as honest compatibility evidence;
- ~~the owned `WithUpstreamDialer` path binds SDK connections to `127.0.0.2`~~ **BUG-001**: the SDK
  uses its ordinary upstream connection to the fixed `http://127.0.0.1:3690`; no source/Host
  discriminator or custom upstream dialer remains;
- the selected Keychain adapter calls Security.framework directly and uses isolated dev/prod service
  namespaces;
- every launch loads settings but starts public access only after explicit UI intent;
- CLI/env/process mechanisms are removed rather than retained as fallback.

## Phase 1: Design and contracts

- [data-model.md](data-model.md) defines the four data lifetimes, settings/status entities,
  generation/revision state machine, ephemeral provider-endpoint input, and mutation rules.
- [private-public-access.md](contracts/private-public-access.md) defines exact protobuf messages,
  Wails methods, named event, UI facade, redacted errors, and one-time generated result.
- [public-host-auth.md](contracts/public-host-auth.md) retains the superseded Host/source design as
  history and defines the current ngrok endpoint Basic Auth boundary plus local/LAN separation.
- [tunnel-service.md](contracts/tunnel-service.md) defines provider-neutral SDK/fake lifecycle and
  deletion of the process runtime.
- [quickstart.md](quickstart.md) defines deterministic, real-network, Keychain, race, streaming,
  cleanup, reproducibility, and packaged double-click evidence.

## Implementation sequence

### Migration ownership, expiry, and rollback

Root composition and `internal/tunnel` own the temporary CLI/SDK coexistence. It exists only while
feature 007 establishes the embedded replacement and expires immediately after the embedded-only
security, streaming, failure-isolation, full package, reproducibility, and rollback gates pass. The
CLI removal task MUST execute in this same feature; no production switch may select the old path.

Immediately before removal, record an immutable pre-removal checkpoint commit and packaged
candidate digest in feature 007 `quickstart.md`. The reference MUST be a full 40-hex SHA of a
user-approved checkpoint commit, not the mutable working tree, index, branch name, tag, stash, or
untracked archive. Create a detached temporary worktree at that SHA, require it to be clean, run
canonical `go run ./cmd/build build` and `go run ./cmd/build package`, verify the packaged app, and
record its digest before removing only that validated temporary worktree. If no approved checkpoint
commit exists, the rollback drill is `BLOCKED` and CLI deletion cannot begin. The reference is
recovery evidence only, not an accepted architecture and not permission to ship both runtimes.

### 1. Establish contracts and compatibility migration

1. Add secret-free `config/v1/public_access.proto` and private
   `private/v1/public_access.proto` exactly as the contracts specify.
2. Remove active plaintext/CLI configuration from `config/v1/config.proto`: reserve the removed
   field numbers and names inside legacy messages and reserve `ApplicationConfig.tunnel_enabled`
   and `ApplicationConfig.tunnel`. Document this as an internal, non-persisted config cutover.
3. Regenerate Go descriptors, advance `schema-revision.txt`, and update
   `compatibility-baseline.binpb` in one reviewed checkpoint. Keep Buf format/lint/breaking and
   generation-drift checks enabled before and after the baseline update.
4. Add explicit adapters/tests proving every new field/enum/oneof is handled and public player
   descriptors remain free of private/config/secret imports.

This phase does not change session or player-config v1 files or schemas.

### 2. Build the stores and deterministic boundaries

1. Add `SecretStore` with fixed refs, metadata-only presence, replace, delete, and scoped-use
   callback. Implement a deterministic fake before the OS adapter.
2. Implement the Keychain adapter with fixed service/account names, non-synchronizing generic
   passwords, update/add/delete semantics, and redacted OSStatus classification. Never expose a
   generic Keychain query or shell command.
3. Implement `PublicAccessSettingsStore` with explicit protobuf/JSON adapter, `0700` directory,
   `0600` atomic temp/rename, injected filesystem/clock, version validation, safe defaults, presence
   hint reconciliation, and private corruption quarantine.
4. Add cryptographic generated-password creation and failure injection. Store before returning;
   never publish the returned value through a reusable status or event.

Unit tests cover missing/corrupt/future settings, modes, atomic failure points, Keychain
locked/denied/unavailable/not-found/update/delete, secret buffer lifetime, 8-character manual
password rule, ≥128-bit generator source, and zero secret formatting.

### 3. Protect the ngrok endpoint without changing the player server

~~The pre-BUG-001 design injected a `127.0.0.2` source/Host policy into the player handler.~~ The
target macOS host cannot bind that unassigned address, and the personal-use requirement does not
need a hostile-client transport discriminator.

1. Keep the existing player HTTP/Connect handler and the one listener on port 3690 unchanged.
2. Construct an in-memory ngrok Basic Auth Traffic Policy from the scoped username/password and
   attach it while creating the Agent Endpoint; never write a policy file or expose policy text.
3. Forward only to the fixed `http://127.0.0.1:3690` upstream using the SDK's ordinary connection
   path. Do not configure `WithUpstreamDialer`, a source alias, Host admission, or a second server.
4. Publish the URL only after `Forward` returns a validated HTTPS endpoint with the policy attached.
5. Direct local/LAN requests never traverse ngrok and therefore remain unauthenticated.

Deterministic tests inspect a provider-neutral policy request rather than its rendered secret text,
prove empty credentials fail before endpoint creation, and cover static/unary/stream intent. One
explicit opt-in real test checks `401` for missing/wrong credentials and incremental `Subscribe`
with correct credentials; without prerequisites it is `NOT RUN`.

### 4. Implement the embedded provider and lifecycle manager

1. Add the provider-neutral `TunnelService`/`TunnelEndpoint` contract and deterministic fake with
   controlled Start, URL, `Done`, Close, clock, failures, and active-count observation.
2. Implement `ngrok.go` using an explicit `v2.1.4` Agent, scoped token, fixed loopback upstream,
   omitted URL for random assignment, exact `WithURL` for a reserved domain, and an in-memory Basic
   Auth Traffic Policy created from scoped Keychain values. No custom upstream dialer or policy file
   exists.
3. Implement `PublicAccessManager` states `disabled`, `starting`, `ready`, `stopping`, `failed` with
   generation and settings revision. Network/store/event calls occur outside locks; every completion
   revalidates its generation/revision/state.
4. Enforce the no-window sequence: acquire the already protected endpoint privately → validate its
   HTTPS URL → mark ready/publish. Withdraw the reusable URL before Close on stop, reconfigure,
   `Done`, failure, and shutdown; endpoint close is the public admission shutdown boundary.
5. Make repeated/concurrent Start and Stop join one intent, make reconfigure close before restart,
   and make stale successes close themselves without publication.
6. Monitor `Done` and disconnect events; map failures to redacted categories and preserve local/LAN.
7. Create the SDK forwarder with an endpoint-owned context derived without inheriting startup
   cancellation. Before commit, a watcher propagates startup cancellation and aborts acquisition;
   after URL validation and commit, only endpoint `Close` owns cancellation of that context.
   Capture disconnect events only as fixed application categories plus a strictly validated
   `ERR_NGROK_<digits>` code, never as raw SDK diagnostic text.

Lifecycle/race tests use 100 schedules, transition probes, timeout/cancellation, partial acquisition,
late completion, close failure/retry, unexpected `Done`, maximum-one-endpoint assertions, local
fallback, shared five-second shutdown budget, and a fake forwarder that binds `Done` to the exact
context supplied to `Forward`. SDK integration tests are explicit opt-in and report `NOT RUN`
without real credentials/connectivity.

### 5. Integrate private desktop operations and UX

1. Compose the settings path/store, platform Keychain adapter, SDK service, and manager in `main.go`.
   Disconnect startup args/env/process helpers from production composition so
   the pre-removal package has an embedded-only runtime; retain their unreachable source only as the
   bounded rollback reference until the removal gate. Application startup loads only
   settings/presence and remains disabled.
2. Add five transparent `desktopService` methods and the typed `public-access-status` event. Route
   every native DTO through the protobuf adapters; keep `server-info` for safe current local/public
   address compatibility. Map internal lifecycle values to observable UI labels exactly:
   `disabled→stopped`, `starting→starting`, `ready→ready`, `stopping→stopping`, and `failed→error`.
3. Update method/event allowlists, Wails bindings, event types, browser fixtures, and initial
   snapshot/event-wins logic together. Do not bind lifecycle, provider, Keychain, environment, or
   process operations.
4. Add the existing Fallout-style Public Access section with labelled fields, default `players`,
   optional domain, presence-only secrets, replacement/deletion, Generate, Start/Stop, redacted
   states/errors, Copy feedback, transition-disabled actions, live regions, and keyboard-safe dialog.
   The enabled preference restores only that section's preference/presentation and never starts an
   endpoint or restores a URL.
5. Keep password input/result out of reusable module state and all storage. Clear manual inputs after
   call; clear generated DOM/closure/result on Copy or dismissal. Never offer Reveal or reconstruct
   full credentials from a saved password.
6. ~~Change shutdown ownership so manager deny/endpoint cleanup occurs first~~ **BUG-001**: Change
   shutdown ownership so manager URL withdrawal and endpoint/Agent cleanup occur first, then
   existing player, session, and desktop cleanup within the same fresh five-second Wails context.

Browser tests cover loading/empty/error/starting/ready/stopping/failed rendering, event/snapshot
races, stale UI completion, keyboard navigation, secure input clearing, one-time Copy, no Reveal,
dynamic start/stop/reconfigure, multi-client player behavior, and local recovery.

### 6. Prove packaged parity, record rollback, and remove the CLI runtime

Before deletion, implement and run the dependency/license, leak, reproducibility, package, offline
double-click, security, streaming, failure-isolation, lifecycle, and rollback gates against an
embedded-only composition candidate. The pre-removal package gate MUST include architecture,
minimum OS, native framework linkage, resources, signing, no bundled provider executable, no PATH
or runtime-download dependency, local fallback, Keychain behavior, and the five-second shutdown
budget. Record the immutable rollback reference and successful clean rebuild. Only after those
gates pass:

- remove `main.go` `configureTunnel`, `publicModeRequested`, `configurationErrorTunnel`, all
  `os.Args`/production tunnel-env behavior, and static player public-access construction;
- remove CLI-only `internal/tunnel/config.go` fields/parsers/defaults: `Binary`, `Port`, `LocalURL`,
  `PolicyParent`, `DefaultBinary`, hard-coded `DefaultDomain`, `NGROK_BIN`, `NGROK_ENABLED`,
  `NGROK_DOMAIN`, `NGROK_USERNAME`, `NGROK_PASSWORD`, `NGROK_BASIC_AUTH`, timeout variables, and
  all `--ngrok*` arguments;
- delete `internal/tunnel/process.go`, `process_darwin.go`, `process_other.go`,
  `process_test.go`, and `process_darwin_integration_test.go`;
- delete log URL parsing, stderr-tail/process diagnostics, process groups, guardian shell, owner pipe,
  temp policy cleanup, terminate/kill escalation, and executable selection;
- rewrite `internal/tunnel/service_test.go` around SDK/fake URL validation, timeout, `Done`, close,
  redaction, concurrency, and cleanup; retain every lifecycle guarantee without process assertions;
- update `internal/platform/test_conventions_test.go` for the new test inventory;
- replace active CLI/PATH/authtoken/default-domain/process cleanup instructions in `README.md` and
  provider-neutralize active `.specify/templates` wording;
- leave completed feature-006 specifications, Electron/Wails rollback records, and their CLI/process
  evidence untouched and clearly historical;
- scan active docs, code, package resources, generated output, and tests to prove no second
  production tunnel mechanism or `NGROK_BIN` survives.

The narrow automation seam is constructor/test-harness injection into the same SDK manager. It has
no external process option and is unavailable to packaged UI.

After deletion, rerun the legacy, dependency/license, leak, reproducibility, package, offline smoke,
and lifecycle gates against the final tree before any conditional external acceptance or completion
claim. The legacy scan is expected to identify remaining files before deletion and MUST pass after
deletion.

### 7. Final reproducibility, package, and release qualification

1. Review root module graph and every new license; add deterministic notice/inventory checks and
   package required notices without altering build ownership. Record a dated external advisory
   review with explicit `PASS`/`FAIL`/`NOT RUN` semantics and no secret-bearing inputs.
2. Record stripped binary/`.app` size delta, cgo Security/CoreFoundation linkage, arm64/minimum OS,
   entitlements, signatures, and no-bundled-provider-executable evidence.
3. ~~Run protobuf/binding drift and breaking gates, vet, unit, full race, browser multi-client,
   secret-leak canaries, reproducible two-build comparison, package build, package verification, and
   offline double-click local smoke in canonical order.~~ **BUG-001 follow-up**: Run the exact final
   deterministic sequence recorded in `quickstart.md` on the post-CLI-removal tree, including empty
   `gofmt -l .` output, protobuf/binding gates, vet, unit/race, clean locked builds for both
   `frontend/` and `client/`, full Playwright, secret/leak/reproducibility gates, direct
   `go run ./cmd/build build`, package build/verification, and offline double-click local smoke.
   Install locked npm dependencies before any protobuf gate that reads `client/node_modules`, then
   run the separate bounded canonical `go run ./cmd/build dev` master/player smoke without a
   separately started frontend or player server.
4. With real credentials, run random URL, reserved domain, invalid/revoked token,
   ~~Host override~~ **BUG-001** missing/wrong/correct Basic Auth, authenticated
   static/unary/non-empty incremental stream, reconnect, stop/reconfigure, crash/quit, and packaged
   UI journeys. Without credentials/connectivity, record each as `NOT RUN`; do not count the fake
   fixture.
5. Run Developer ID/notary/staple/DMG/Gatekeeper gates only with their real prerequisites; otherwise
   record separate `NOT RUN` outcomes.
6. Reconcile the full evidence matrix in `quickstart.md` against the final candidate digest. No
   earlier or historical feature evidence substitutes for feature 007.

## Testing strategy

| Layer | Main files | Required proof |
|---|---|---|
| Unit | `internal/tunnel/*_test.go`, `internal/player/http_test.go`, `internal/platform/keychain_test.go`, contract tests | validation, state transitions, generation/revision, redaction, stores, ~~Host/Auth~~ **BUG-001** endpoint-policy input, **BUG-002** startup/owned-lifetime handoff, generator, idempotence |
| Race | `go test -race ./...` plus focused 100-schedule tests | ~~no mixed grant, stale activation~~ **BUG-001** no mixed revision or stale publication, duplicate endpoint, event/order, or stop/reconfigure race |
| Lifecycle integration | `app_test.go`, `wails_host_test.go`, fake endpoint/store/network/clock | local-first, policy-before-publish, ~~deny-before-close~~ **BUG-001** URL-withdrawal-before-endpoint-close, `Done`, partial startup, five-second cleanup |
| HTTP/ConnectRPC | real in-process server and protected fixture | all static/RPC paths, ~~exact/unknown/local Host and constant-time auth~~ **BUG-001** endpoint-auth fixture outcomes and local/LAN no-challenge, non-empty streaming, reconnect |
| Contract/drift | Buf/proto scripts, `app_contract_test.go`, Wails checks | public/private isolation, reserved removals, explicit adapters, deterministic generated outputs |
| Browser | Playwright desktop and multi-client player journeys | UI-only management, one-time Copy, full player behavior, stale UI rejection, accessibility |
| Leak | canary script plus Go/browser tests | zero token/password exposure across every prohibited surface and shipped artifact |
| Build/package | canonical build tool, reproducibility/package scripts | exact graph, licenses, offline launch, arm64/macOS13, no CLI/PATH/download, package cleanup |
| External | opt-in SDK/Playwright/package journeys | real endpoint/domain/stream/~~Host~~ **BUG-001** auth/quit evidence or explicit `NOT RUN` |

## Risk controls

- ~~Unknown external Host currently passes: make all-path dynamic policy foundational and test it
  before connecting the SDK.~~ **BUG-001**: The player application no longer owns public Host
  admission. The ngrok endpoint Basic Auth policy is active before URL publication, while direct
  local/LAN traffic remains outside the endpoint.
- **SDK cancellation is not cleanup**: always call bounded endpoint close then Agent disconnect; use
  `Done` only as a signal. **BUG-002**: the context passed to SDK `Forward` belongs to the acquired
  endpoint after commit; a completed manager startup operation cannot own or cancel that lifetime.
- **Go strings cannot be reliably zeroed**: keep provider token inside the shortest SDK adapter
  lifetime, keep Basic password as locked byte buffers, drop all SDK references on stop, and verify
  no observable leaks rather than claiming impossible heap erasure.
- **Cross-store mutation is not one transaction**: disable public access first, make each store
  individually durable, reconcile actual presence after failure, and never restart a partially
  validated revision.
- ~~**Host-based local/public distinction is unsafe**: classify the dedicated `127.0.0.2` SDK
  source before Host and fail closed when source binding fails.~~ **BUG-001**: application
  Host/source classification is removed; the personal-use endpoint is protected at ngrok and
  local/LAN traffic is separated by topology. A real streaming/auth check remains opt-in evidence,
  not an unconditional source-binding gate.
- **Keychain behavior varies with identity/state**: isolate dev/prod services and keep real packaged
  locked/denied/signing evidence separate from fakes.
- **Schema cleanup is intentionally incompatible internally**: reserve removed fields, document the
  migration, update the baseline once, and continue running the breaking gate afterward.
- **Conditional gates can be unavailable**: label them `NOT RUN`; never convert missing credentials
  or a deterministic fake into release/public-endpoint proof.

## Completion gate

Feature 007 is ready for task generation only after this plan is approved. Implementation cutover is
complete only when the embedded SDK path has parity, the CLI/process path and active documentation
are gone, direct canonical build commands pass without Make, package smoke proves no external
binary/PATH dependency, secrets have zero forbidden leaks, and every unavailable external gate is
honestly recorded as `NOT RUN`.
