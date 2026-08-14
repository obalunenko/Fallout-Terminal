# Quickstart and Acceptance Record: Wails v3 Migration

This document describes the implementation target. It is not evidence that commands have run. Every box starts unchecked; record `PASS`, `FAIL`, or `NOT RUN` with candidate/environment provenance during implementation and acceptance.

## Candidate Identity

- [x] Candidate Git SHA recorded: `658071b7011197c4f229f6a5b1f109de2764fd69`
- [x] Baseline rollback source verified: `f1084b3df8b5630862bdf7a0f347b599156653ef`
- [x] Go module pin is exactly `github.com/wailsapp/wails/v3 v3.0.0-beta.8`
- [x] `tools/wails/go.mod` declares only `tool github.com/wailsapp/wails/v3/cmd/wails3` and pins `github.com/wailsapp/wails/v3 v3.0.0-beta.8`, with its own committed `go.sum`
- [x] `tools/buf/go.mod` declares only `tool github.com/bufbuild/buf/cmd/buf` and pins `github.com/bufbuild/buf v1.72.0`, with its own committed `go.sum`
- [x] `tools/protoc-gen-go/go.mod` declares only `tool google.golang.org/protobuf/cmd/protoc-gen-go` and pins `google.golang.org/protobuf v1.36.11`, with its own committed `go.sum`
- [x] `tools/protoc-gen-connect-go/go.mod` declares only `tool connectrpc.com/connect/cmd/protoc-gen-connect-go` and pins `connectrpc.com/connect v1.20.0`, with its own committed `go.sum`
- [x] Root `go.mod` and `go.sum` contain no tool declaration, tool-only dependency, or tool-only checksum
- [x] Frontend runtime is exactly `@wailsio/runtime` `3.0.0-beta.8`, including the Vite plugin subpath
- [x] Go/npm locks and CI/release pins agree; no Wails `latest`, caret, tilde, or unbounded range exists

Candidate freeze: `658071b7011197c4f229f6a5b1f109de2764fd69`, committed 2026-08-14T16:09:18+04:00 from a clean v2-free worktree. Later evidence-only commits do not redefine this build identity.

## Clean Setup

Prerequisites: macOS 13+ Apple Silicon, Go 1.26.x, Node.js 20.19+, Xcode command-line tools, and the repository-pinned protobuf/Buf tools.

```sh
git clone <repository-url>
cd Fallout-Terminal
go mod download
go tool -modfile=tools/wails/go.mod wails3 version
npm ci --prefix client
npm ci --prefix frontend
npm ci --prefix tests/browser
scripts/proto-check.sh
go tool -modfile=tools/wails/go.mod wails3 generate bindings -clean ./...
```

- [ ] Setup completed from a clean checkout
- [ ] `go tool -modfile=tools/wails/go.mod wails3 version` identified beta.8
- [ ] Locked installs made no unexplained manifest/lock changes
- [ ] Protobuf check and clean binding generation succeeded

## Local Development

From the repository root, run exactly one development command:

```sh
go run ./cmd/build dev
```

- [ ] Exactly one master window opened with title `Fallout Terminal — Master Control`
- [ ] Initial size 1200×780, minimum 900×600, and accepted dark presentation were observed
- [ ] Local player URL appeared and only one player listener was present
- [ ] Master generated calls and all four event subscriptions were usable before ready presentation
- [ ] Stopping development released the listener and any owned tunnel process within five seconds

Do not separately run Vite, the player listener, or a tunnel supervisor for the one-command acceptance check.

#### Go-owned build entry checkpoint — 2026-08-14

- `PASS` — `env GOCACHE=/private/tmp/fallout-terminal-go-cache go run ./cmd/build build`; the standard-library-only graph verified protobuf, built the player, generated the exact pinned Wails bindings, built the master, and compiled `build/bin/Fallout Terminal` without Taskfile, Make, or a global Wails CLI.
- `PASS` — `env GOCACHE=/private/tmp/fallout-terminal-go-cache go run ./cmd/build package`; the same graph assembled `build/bin/Fallout Terminal.app`, installed plist/icon/demo before the final ad-hoc signature, and produced an arm64 executable with macOS 13.0 minimum metadata.
- `PASS` — `codesign --verify --deep --strict --verbose=2`, `lipo -archs`, plist inspection, and resource checks passed for the Go-built package.
- `PASS` — the canonical Go development entry reached the native application, and stopping the exact smoke-test PID left no `Fallout Terminal` process or port-3690 listener. The sandbox denied WebKit cache writes and emitted repeated font-service diagnostics, so this launch is process/lifecycle evidence rather than a visual acceptance journey.

## Automated Gates

### Go

```sh
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
```

- [ ] gofmt: `________` — expected no paths
- [ ] vet: `________`
- [ ] tests: `________`
- [ ] race: `________`

### Contracts and deterministic generation

```sh
scripts/proto-check.sh
scripts/proto-drift-test.sh
scripts/proto-breaking.sh --all-fixtures
go tool -modfile=tools/wails/go.mod wails3 generate bindings -clean ./...
```

Run clean Wails binding generation a second time and compare the complete generated tree/inventory using the repository-provided deterministic-binding check introduced by implementation.

- [ ] Buf format/lint and deterministic protobuf generation passed
- [ ] Compatibility baseline, negative fixtures, and public/private graph isolation passed
- [ ] Two clean Wails binding generations were identical
- [ ] Binding inventory contained exactly the 25 accepted operations
- [ ] `Start`, `Shutdown`, lifecycle/generic/native/player procedures were absent
- [ ] Generated master imports used only the v3 `frontend/bindings` path

### Locked frontend builds

```sh
npm ci --prefix client
npm run build --prefix client
npm ci --prefix frontend
go tool -modfile=tools/wails/go.mod wails3 generate bindings -clean ./...
npm run build --prefix frontend
```

- [ ] Player clean production build passed independently of Wails bindings
- [ ] Master clean production build passed only after binding generation
- [ ] No CDN or application-runtime package download exists in either bundle
- [ ] No `window.go`, `window.runtime`, `frontend/wailsjs`, Electron global, or privileged production fallback exists

#### Reproducible root-build checkpoint — 2026-08-14

- `PASS` — direct locked player and master installs/builds. `npm ci --prefix client`, `npm run build --prefix client`, `npm ci --prefix frontend`, and `npm run build --prefix frontend` completed without manifest or lock drift; the player transformed 144 modules and the master transformed 40 modules.
- `PASS` — two direct `scripts/proto-generate.sh` runs produced the same reviewed schema revision `66679f4b2f09e4e6c89f6f3921bdecf052878f98b9355954bcb1815386343ba3` without root-module drift.
- `PASS` — `scripts/reproducible-build-check.sh` ran the complete repository-owned protobuf → player → Wails bindings → master → native graph twice. Generated Go/JavaScript contracts, all 25-method/four-event bindings, both frontend distributions, and `build/bin/Fallout Terminal` were byte-identical between runs, and the full tracked/untracked repository state was unchanged.
- `PASS` — `scripts/tool-modules-check.sh` confirmed exact isolated Wails, Buf, protoc-gen-go, and protoc-gen-connect-go ownership/pins and no tool declaration, tool-only dependency, global tool command, or checksum drift in the root application module.

#### T046 binding/facade/event verification — 2026-08-14

- `PASS` — `scripts/wails-bindings-check.sh`; two clean beta.8 generations were byte-identical, with exactly 25 accepted methods and exactly four typed events (`server-info`, `client-count`, `hack-state`, `coordination-state`).
- `PASS` — forbidden generated inventory contained no lifecycle, generic dispatch, native filesystem/process/browser-manager, or player-service method.
- `PASS` — `go test . ./internal/platform -count=1`; detached status/result shapes, Go-only lifecycle phase, injected v3 event manager, all-method forwarding, and source/bundle isolation checks passed.
- `PASS` — `npx playwright test desktop-api.spec.mjs`; 3 passed, proving explicit generated calls, normalization, all-four-listeners-before-snapshot ordering, `.data` unwrapping, per-field newer-event precedence, exact-once release, pending-callback suppression, and hot disposal.
- `PASS` — `npm run build --prefix frontend`; the official `@wailsio/runtime/plugins/vite` beta.8 plugin built the master only after generated `frontend/bindings` were present.
- `PASS` — source and production-bundle scan found no `window.go`, `window.runtime`, `frontend/wailsjs`, generic dispatch, lifecycle method, or `CopyDemo` UI capability.

### Browser-player journeys

```sh
npm ci --prefix tests/browser
npm test --prefix tests/browser
```

- [ ] All generated ConnectRPC Playwright journeys passed
- [ ] Four-, five-, six-, and seven-client scenarios passed
- [ ] Reconnect, replay, concurrency, slow/overflow, and cancellation cases passed
- [ ] Sound manifest/playback and public/private isolation passed

#### T036 generated-player verification — 2026-08-14

- `PASS` — `go test ./internal/player ./internal/control ./internal/live ./internal/hack -count=1`; all four packages passed.
- `PASS` — `npm ci --prefix tests/browser`; 4 packages installed, 0 vulnerabilities. npm reported the expected blocked optional `fsevents` install script on macOS; no dependency or lock drift occurred.
- `PASS` — `npm test --prefix tests/browser`; 18 passed, 1 conditional authenticated-ngrok test skipped because `NGROK_TEST_URL`, `NGROK_USERNAME`, and `NGROK_PASSWORD` were not supplied.
- `PASS` — the generated local fixture exercised distinct 4-, 5-, 6-, and 7-player scenarios, 32 accepted navigation operations, rejected observer mutations, three reconnects, authoritative stream convergence, replay-cue suppression, typed sound discovery/failure tolerance, and opaque recognition-handle-only storage.
- `NOT RUN` — real authenticated public endpoint journey; credentials and a selected public endpoint were unavailable. This is conditional and is not counted as local public-mode evidence.

## Private Desktop Acceptance

Record representative create/open/edit/copy/save behavior using safety-copy data.

- [ ] New/open/save session and ordered revision behavior matched baseline
- [ ] Bundled demo stayed read-only until explicit copy; `CopyDemo` added no new UI control
- [ ] New/open/referenced player configuration behavior matched baseline
- [ ] All roster, controller, broadcast, terminal-switch, live-update, and hack operations matched baseline
- [ ] Cancel/error/redaction/result shapes matched `contracts/desktop-bridge.md`
- [ ] Dialog titles, JSON filters, directories/filenames, aliases, creation policy, and cancel-as-empty matched
- [ ] Only absolute HTTP(S) external URLs opened; invalid schemes failed safely
- [x] Application-owned listener/desktop startup failure produced actionable master state
- [x] Tunnel failure preserved the local URL and showed a credential-free error

#### Integrated native checkpoint — 2026-08-14 (P1 complete)

- `SUPERSEDED` — the earlier Wails-owned build checkpoint predates the repository-owned Go build command and must be rerun before acceptance.
- `PASS` — launched `build/bin/Fallout Terminal`; screenshot inspection showed one dark native window titled `Fallout Terminal — Master Control`, the start controls, and status rendered from the generated facade.
- `PASS` — process inspection found exactly one application PID and exactly one listener on TCP port 3690.
- `PASS` — a handled terminal interrupt stopped the application; immediate process/listener inspection found neither the PID nor a port-3690 listener.
- `PASS` — T024 native persistence journey. With macOS Accessibility enabled, the repository-owned `go run ./cmd/build dev` entry presented exactly one window titled `Fallout Terminal — Master Control`. The native New Session and Create Player Configuration dialogs created safety-copy files under `/private/tmp`; generated-facade operations added roster character `Vault Dweller`, terminal intro text `Native persistence checkpoint`, and a folder node. After a handled development-process interrupt released the application process and port-3690 listener, a fresh launch used the native Open Session dialog and rendered the same session path, referenced player configuration, roster character, terminal text, and folder from disk in exactly one window. The temporary files were `/private/tmp/fallout-t024-native-20260814.json` and `/private/tmp/fallout-t024-players-20260814.json`.
- `PASS` — T032 normal close. A controlled native close-button event started with one window, the application exited with status 0, and immediate checks found neither a `Fallout Terminal` process nor a port-3690 listener.
- `PASS` — T032 Cmd+Q. A controlled native Cmd+Q event started with one window, the application exited with status 0, and immediate checks again found neither the process nor a port-3690 listener.
- `PASS` — T032 handled development interrupt and repeated shutdown. `go run ./cmd/build dev` received a terminal interrupt and reported the expected `signal: interrupt`; process/listener inspection found no orphan. Together with the independent normal-close, Cmd+Q, occupied-port, and tunnel-failure shutdowns, repeated teardown remained idempotent and completed inside the five-second gate.
- `PASS` — T032 occupied port and application partial startup. A Python fixture bound `0.0.0.0:3690` exclusively before launch. The native window rendered `ЗАПУСК НЕ ЗАВЕРШЁН` with `bind: address already in use`; the app owned no listener, closing it exited with status 0, and the unrelated fixture remained the sole listener until separately stopped.
- `PASS` — T032 tunnel failure and local fallback. With valid non-secret test credentials and `NGROK_BIN=/private/tmp/fallout-t032-missing-ngrok`, the native window rendered local-ready/public-unavailable guidance without credentials or the configured password. `curl http://127.0.0.1:3690/` returned HTTP 200, and shutdown released the local listener.
- `PASS` — T037 integrated packaged-native local 4–7-player journey. The stale activation control was fixed by refreshing the tree header after `StartBroadcast`; a focused asset-contract regression test and `go run ./cmd/build build` passed. One clean `build/bin/Fallout Terminal.app` instance opened safety copies of the demo session and a seven-character roster, started the broadcast, enabled `СДЕЛАТЬ АКТИВНЫМ`, and activated terminal `t_demo1`. Real port-3690 clients then grew incrementally from four through seven simultaneous players with distinct opaque recognition handles and distinct characters. Controller/observer roles remained isolated, observers emitted no mutations, 32 navigation/back actions were accepted across four rounds per player count, three reload reconnects retained their handles, browser storage contained only the recognition token, and 80 typed `SoundManifest` requests were observed without WebSocket fallback. Exactly one application PID and one listener existed during the run; Cmd+Q left neither process nor listener immediately afterward. The temporary acceptance harness was `/private/tmp/fallout-t037-native.mjs`.
- `NOT RUN` — T037 real authenticated public mode; no selected public endpoint or real ngrok credentials were available. This conditional result is not represented as a pass.

## Event and Readiness Acceptance

- [ ] Exact events `server-info`, `client-count`, `hack-state`, and `coordination-state` were observed
- [ ] All four listeners registered before the initial status snapshot applied
- [ ] A newer event won over an older snapshot independently for each field
- [ ] Every listener released its underlying subscription exactly once
- [ ] Release during pending snapshot/callback and repeated release produced no late callback
- [ ] No duplicate effect resulted from any window-ready signal plus snapshot/event initialization
- [ ] Existing `startupError` was rendered without a protobuf phase/schema change

## Personal-Use macOS Package

```sh
go run ./cmd/build build
go run ./cmd/build package
```

Expected application:

```text
build/bin/Fallout Terminal.app
```

Inspect the final, already signed bundle. Use the repository's implementation-time package verification command/script for the complete inventory; do not modify the bundle after this point.

- [ ] App exists at the established path
- [ ] Executable is arm64 and minimum OS is macOS 13.0
- [ ] Identifier/name/version/comments/copyright metadata are correct
- [ ] Production plist, entitlements, and icon are present/correct
- [ ] Master and player assets, generated player client, fonts, and sounds are present
- [ ] `Contents/Resources/sessions/demo.json` is present with accepted read-only behavior
- [ ] Final personal-use signature is valid and no resource was copied after signing
- [ ] With external network unavailable, one app launch served master and local player successfully
- [ ] Exactly one listener existed while running and quit released all owned resources within five seconds

#### Personal-use package checkpoint — 2026-08-14

- `PASS` — `go run ./cmd/build package` produced the established arm64 app with macOS 13.0 binary/plist metadata, the reviewed entitlements, icon, and read-only bundled demo installed before the final ad-hoc signature.
- `PASS` — `scripts/verify-macos-app.sh` validated architecture, metadata, binary deployment target, entitlement values, offline frontend/player assets, resource identity/mode, resource-before-sign ordering, and the final deep/strict signature. The canonical bundle-manifest SHA-256 at this checkpoint was `7672ac4679ddb9dbfe0d1e93118f3f15250117caa59040ef6cb8fc6965365a57`.
- `PASS` — offline personal-use smoke. One packaged application PID presented exactly one native window titled `Fallout Terminal — Master Control` and owned exactly one TCP listener on port 3690. A local generated Connect player received HTTP 200, hid its connection overlay, rendered the idle terminal, made nine typed RPC requests with WebSocket disabled, and made zero requests outside `http://127.0.0.1:3690`. Cmd+Q left no application process or listener immediately afterward.
- `NOT RUN` — authenticated public/ngrok trust behavior; no real selected public endpoint or credentials were available.
- `NOT RUN` — Developer ID, notarization, staple, DMG, and Gatekeeper trust gates; no installed release identity or selected notary profile was supplied. The passing ad-hoc personal-use signature is not represented as Developer ID evidence.

## Local Mode and Soak

- [x] Run a local master/player session for at least 60 minutes
- [x] Include 4–7 concurrent players and at least 25 mixed operations
- [x] Include at least three reconnects and two save/reopen cycles plus navigation, hacking, coordination, and sound
- [x] Confirm convergence and expected revision after each operation, newest durable revision, and responsive clients
- [x] Record the packaged application PID as `APP_PID`: `63116`
- [x] At minutes 15, 30, and 60, collect five `ps -o rss= -p <APP_PID>` samples ten seconds apart and record the median KiB values: `RSS15=117936`, `RSS30=128432`, `RSS60=121312` KiB
- [x] Mark the local soak `FAIL` when both `RSS30 > 1.25 × RSS15` and `RSS60 > 1.25 × RSS15`; a single transient sample or one elevated median does not fail this gate
- [x] Confirm exactly one listener during operation and zero owned listener/tunnel resources within five seconds after quit
- Result: `PASS`
  Duration/environment/evidence: `60.67 minutes on 2026-08-14, macOS arm64; seven distinct players; 57 accepted navigation/hack operations; 28 rejected observer attempts; three retained-identity reconnects; two version-1 autosave/master-WebView-reopen cycles; 90 authoritative convergence checks; 192 typed sound requests; one listener throughout; cleanup in 0.014s. RSS samples (KiB): 15m [117936,117936,117952,117936,117936], 30m [128448,128432,128432,128416,128432], 60m [121312,121312,121328,121312,121312]. Both later medians did not exceed 125% of RSS15. Machine-readable evidence: /private/tmp/fallout-t067-result.json.`

## Conditional Public Mode

Run only with real ngrok credentials/connectivity. If unavailable, write `NOT RUN` and the reason; do not count it as public-mode passing evidence.

- [x] / [ ] `PASS` — authenticated public tunnel started only after local readiness
- [x] / [ ] `PASS` — credentials and traffic policy stayed out of UI/log/session/public schemas
- [x] / [ ] `PASS` — public soak ran for at least 30 minutes with 4–7 players, at least 15 mixed public operations, and at least two reconnects
- [x] / [ ] `PASS` — one unauthorized request was rejected without private detail and every accepted operation converged at the expected revision
- [x] / [ ] `PASS` — controlled tunnel loss preserved usable local play, credentials/private fields stayed isolated, and final cleanup released owned resources within five seconds
- Result/reason: `PASS — 32.01 minutes on 2026-08-14 through the configured authenticated https://fallout-terminal.ngrok.app endpoint; seven distinct players, HTTP 401 without credentials, 20 accepted operations, 10 rejected observer attempts, two retained-identity reconnects, 32 convergence checks, 72 sound requests, and zero observer mutations. Terminating only owned ngrok PID 2700 removed the tunnel while app PID 2695 continued returning local HTTP 200; final app/listener/tunnel cleanup completed in 0.020s. The temporary Basic Auth password file was mode 0600, never printed or persisted in application/session/public evidence, and was deleted after the run. Machine-readable evidence: /private/tmp/fallout-t067-public-result.json.`

## Conditional Developer ID Release

Run only for an explicitly selected public candidate with real installed identity and notary credentials.

```sh
scripts/build-macos.sh --preflight
scripts/build-macos.sh
```

- [ ] / [ ] `NOT RUN` — Developer ID replacement signature and hardened runtime
- [ ] / [ ] `NOT RUN` — app notarization and staple
- [ ] / [ ] `NOT RUN` — signed/notarized/stapled DMG
- [ ] / [ ] `NOT RUN` — Gatekeeper checks without bypass
- [ ] / [ ] `NOT RUN` — final SHA-256 and credential-redacted evidence
- Result/reason: `________________________________`

## Cutover Scan

Do not remove Wails v2 or run the final cutover scan until the required local soak and the rollback drill below have passed. Conditional public gates may remain `NOT RUN` only under their documented profile rules.

Run the implementation-provided source/generated/dependency/bundle/documentation scans plus a clean rebuild after v2 removal.

- [ ] No Wails v2 import or module dependency remains
- [ ] No v2 CLI command, `wails.json`, post-build hook, generated assumption, or runtime global remains active
- [ ] No permanent v2/v3 feature flag or dual desktop implementation remains
- [ ] No forbidden lifecycle/generic/native/player method appears in generated bindings
- [ ] Active README, CI, scripts, and rollback instructions use the exact isolated `go tool -modfile=tools/wails/go.mod wails3 ...` commands
- [ ] Historical completed specs and `docs/wails-migration-rollback.md` remain intact and labeled as history
- [ ] Full required matrix and personal-use package gates passed against final cutover source

## Rollback Drill

Complete this drill before Wails v2 removal and the final cutover scan.

Use `docs/wails-v3-migration-rollback.md` after it is created. Work only on safety copies of selected session-v1 and player-config-v1 files.

1. Record current candidate, selected paths, and safety-copy hashes.
2. Stop the candidate and verify no owned process/listener remains.
3. Restore/build the recorded Wails v2 source commit `f1084b3df8b5630862bdf7a0f347b599156653ef`, or use a separately recorded accepted v2 artifact.
4. Open the safety-copy version-1 files without conversion.
5. Exercise representative master and local-player behavior.
6. Record actual results and return to the candidate only according to the rollback record.

- [x] Source rollback identity verified
- [x] Safety-copy hashes recorded
- [x] Rollback opened unchanged version-1 data
- [x] Representative master/player local journey passed
- [x] No migration/conversion was required
- [x] v2 artifact digest recorded only if that artifact was truly built and accepted
- Result/evidence: `PASS` — 2026-08-14. The canonical source commit was verified and built in a clean isolated clone with the exact Wails v2.13.0 CLI. Initial safety-copy hashes matched their version-1 originals; the v2 master opened the session and seven-character player configuration, kept version 1, and saved only to the safety session copy. One app and one listener served four distinct assigned local players. The synchronized corrected harness observed one typed `Guess` RPC, accepted the master's hack-success override, navigated into a terminal row and back, reconnected with the same retained player identity, and observed 40 typed sound-manifest requests. After a clean quit, the exact safety-copy session reopened with both expected terminals and its associated seven-character configuration visible; both files remained version 1 and retained their recorded post-save hashes. Both quits released the app process/listener. The temporary accepted-for-drill v2 executable SHA-256 is `c1faf7fe4f2ed0abc5c4814b8e71805f5b57a65b817fd3a45bbcc90bdaf29530`; the immutable source commit remains canonical rollback authority. See `docs/wails-v3-migration-rollback.md` for hashes and paths.

## Acceptance Summary

| Gate group | Result (`PASS`/`FAIL`/`NOT RUN`) | Evidence |
|---|---|---|
| Exact pins and clean setup |  |  |
| Go and contract gates |  |  |
| Bindings/facade/events |  |  |
| Both frontends/player journeys |  |  |
| Lifecycle/platform |  |  |
| Personal-use package |  |  |
| Local soak |  |  |
| Rollback drill |  |  |
| Public ngrok (conditional) |  |  |
| Developer ID/notary/DMG (conditional) |  |  |
| Final v2 cutover scans |  |  |

Wails v3 becomes accepted production only when every required non-conditional row passes against the same final source and pin set. Conditional rows may be `NOT RUN` only when their release profile was not selected or real external prerequisites were unavailable.
