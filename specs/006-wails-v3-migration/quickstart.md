# Quickstart and Acceptance Record: Wails v3 Migration

This document describes the implementation target. It is not evidence that commands have run. Every box starts unchecked; record `PASS`, `FAIL`, or `NOT RUN` with candidate/environment provenance during implementation and acceptance.

## Candidate Identity

- [ ] Candidate Git SHA recorded: `________________`
- [ ] Baseline rollback source verified: `f1084b3df8b5630862bdf7a0f347b599156653ef`
- [ ] Go module pin is exactly `github.com/wailsapp/wails/v3 v3.0.0-beta.8`
- [ ] CLI pin is exactly `github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.8`
- [ ] Frontend runtime is exactly `@wailsio/runtime` `3.0.0-beta.8`, including the Vite plugin subpath
- [ ] Go/npm locks and CI/release pins agree; no Wails `latest`, caret, tilde, or unbounded range exists

## Clean Setup

Prerequisites: macOS 13+ Apple Silicon, Go 1.26.x, Node.js 20.19+, Xcode command-line tools, and the repository-pinned protobuf/Buf tools.

```sh
git clone <repository-url>
cd Fallout-Terminal
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.8
go mod download
npm ci --prefix client
npm ci --prefix frontend
npm ci --prefix tests/browser
scripts/proto-check.sh
wails3 generate bindings -clean ./...
```

- [ ] Setup completed from a clean checkout
- [ ] `wails3 version` identified beta.8
- [ ] Locked installs made no unexplained manifest/lock changes
- [ ] Protobuf check and clean binding generation succeeded

## Local Development

From the repository root, run exactly one development command:

```sh
wails3 dev
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
wails3 generate bindings -clean ./...
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
wails3 generate bindings -clean ./...
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
wails3 build
wails3 package GOOS=darwin GOARCH=arm64
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

- [ ] Run a representative long-lived local master/player session
- [ ] Include 4–7 players, at least 25 mixed operations, saves, reconnects, hacking, navigation, and sound
- [ ] Confirm newest durable revision, convergence, responsive clients, bounded memory/process behavior, and clean quit
- Result: `________`
  Duration/environment/evidence: `________________________________`

## Conditional Public Mode

Run only with real ngrok credentials/connectivity. If unavailable, write `NOT RUN` and the reason; do not count it as public-mode passing evidence.

- [ ] / [ ] `NOT RUN` — authenticated public tunnel started only after local readiness
- [ ] / [ ] `NOT RUN` — credentials and traffic policy stayed out of UI/log/session/public schemas
- [ ] / [ ] `NOT RUN` — public 4–7-player soak retained authorization, privacy, convergence, and local fallback
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

Run the implementation-provided source/generated/dependency/bundle/documentation scans plus a clean rebuild after v2 removal.

- [ ] No Wails v2 import or module dependency remains
- [ ] No v2 CLI command, `wails.json`, post-build hook, generated assumption, or runtime global remains active
- [ ] No permanent v2/v3 feature flag or dual desktop implementation remains
- [ ] No forbidden lifecycle/generic/native/player method appears in generated bindings
- [ ] Active README, CI, scripts, and rollback instructions use exact wails3 commands
- [ ] Historical completed specs and `docs/wails-migration-rollback.md` remain intact and labeled as history
- [ ] Full required matrix and personal-use package gates passed against final cutover source

## Rollback Drill

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
