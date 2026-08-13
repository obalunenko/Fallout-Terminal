# Quickstart and Acceptance Record: Wails v3 Migration

This document describes the implementation target. It is not evidence that commands have run. Every box starts unchecked; record `PASS`, `FAIL`, or `NOT RUN` with candidate/environment provenance during implementation and acceptance.

## Candidate Identity

- [ ] Candidate Git SHA recorded: `________________`
- [ ] Baseline rollback source verified: `f1084b3df8b5630862bdf7a0f347b599156653ef`
- [ ] Go module pin is exactly `github.com/wailsapp/wails/v3 v3.0.0-beta.8`
- [ ] `tools/wails/go.mod` declares only `tool github.com/wailsapp/wails/v3/cmd/wails3` and pins `github.com/wailsapp/wails/v3 v3.0.0-beta.8`, with its own committed `go.sum`
- [ ] `tools/buf/go.mod` declares only `tool github.com/bufbuild/buf/cmd/buf` and pins `github.com/bufbuild/buf v1.72.0`, with its own committed `go.sum`
- [ ] `tools/protoc-gen-go/go.mod` declares only `tool google.golang.org/protobuf/cmd/protoc-gen-go` and pins `google.golang.org/protobuf v1.36.11`, with its own committed `go.sum`
- [ ] `tools/protoc-gen-connect-go/go.mod` declares only `tool connectrpc.com/connect/cmd/protoc-gen-connect-go` and pins `connectrpc.com/connect v1.20.0`, with its own committed `go.sum`
- [ ] Root `go.mod` and `go.sum` contain no tool declaration, tool-only dependency, or tool-only checksum
- [ ] Frontend runtime is exactly `@wailsio/runtime` `3.0.0-beta.8`, including the Vite plugin subpath
- [ ] Go/npm locks and CI/release pins agree; no Wails `latest`, caret, tilde, or unbounded range exists

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
go tool -modfile=tools/wails/go.mod wails3 dev
```

- [ ] Exactly one master window opened with title `Fallout Terminal — Master Control`
- [ ] Initial size 1200×780, minimum 900×600, and accepted dark presentation were observed
- [ ] Local player URL appeared and only one player listener was present
- [ ] Master generated calls and all four event subscriptions were usable before ready presentation
- [ ] Stopping development released the listener and any owned tunnel process within five seconds

Do not separately run Vite, the player listener, or a tunnel supervisor for the one-command acceptance check.

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

### Browser-player journeys

```sh
npm ci --prefix tests/browser
npm test --prefix tests/browser
```

- [ ] All generated ConnectRPC Playwright journeys passed
- [ ] Four-, five-, six-, and seven-client scenarios passed
- [ ] Reconnect, replay, concurrency, slow/overflow, and cancellation cases passed
- [ ] Sound manifest/playback and public/private isolation passed

## Private Desktop Acceptance

Record representative create/open/edit/copy/save behavior using safety-copy data.

- [ ] New/open/save session and ordered revision behavior matched baseline
- [ ] Bundled demo stayed read-only until explicit copy; `CopyDemo` added no new UI control
- [ ] New/open/referenced player configuration behavior matched baseline
- [ ] All roster, controller, broadcast, terminal-switch, live-update, and hack operations matched baseline
- [ ] Cancel/error/redaction/result shapes matched `contracts/desktop-bridge.md`
- [ ] Dialog titles, JSON filters, directories/filenames, aliases, creation policy, and cancel-as-empty matched
- [ ] Only absolute HTTP(S) external URLs opened; invalid schemes failed safely
- [ ] Application-owned listener/desktop startup failure produced actionable master state
- [ ] Tunnel failure preserved the local URL and showed a credential-free error

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
go tool -modfile=tools/wails/go.mod wails3 build
go tool -modfile=tools/wails/go.mod wails3 package GOOS=darwin GOARCH=arm64
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

## Local Mode and Soak

- [ ] Run a local master/player session for at least 60 minutes
- [ ] Include 4–7 concurrent players and at least 25 mixed operations
- [ ] Include at least three reconnects and two save/reopen cycles plus navigation, hacking, coordination, and sound
- [ ] Confirm convergence and expected revision after each operation, newest durable revision, and responsive clients
- [ ] Record the packaged application PID as `APP_PID`: `________`
- [ ] At minutes 15, 30, and 60, collect five `ps -o rss= -p <APP_PID>` samples ten seconds apart and record the median KiB values: `RSS15=________`, `RSS30=________`, `RSS60=________`
- [ ] Mark the local soak `FAIL` when both `RSS30 > 1.25 × RSS15` and `RSS60 > 1.25 × RSS15`; a single transient sample or one elevated median does not fail this gate
- [ ] Confirm exactly one listener during operation and zero owned listener/tunnel resources within five seconds after quit
- Result: `________`
  Duration/environment/evidence: `________________________________`

## Conditional Public Mode

Run only with real ngrok credentials/connectivity. If unavailable, write `NOT RUN` and the reason; do not count it as public-mode passing evidence.

- [ ] / [ ] `NOT RUN` — authenticated public tunnel started only after local readiness
- [ ] / [ ] `NOT RUN` — credentials and traffic policy stayed out of UI/log/session/public schemas
- [ ] / [ ] `NOT RUN` — public soak ran for at least 30 minutes with 4–7 players, at least 15 mixed public operations, and at least two reconnects
- [ ] / [ ] `NOT RUN` — one unauthorized request was rejected without private detail and every accepted operation converged at the expected revision
- [ ] / [ ] `NOT RUN` — controlled tunnel loss preserved usable local play, credentials/private fields stayed isolated, and final cleanup released owned resources within five seconds
- Result/reason: `________________________________`

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

- [ ] Source rollback identity verified
- [ ] Safety-copy hashes recorded
- [ ] Rollback opened unchanged version-1 data
- [ ] Representative master/player local journey passed
- [ ] No migration/conversion was required
- [ ] v2 artifact digest recorded only if that artifact was truly built and accepted
- Result/evidence: `________________________________`

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
