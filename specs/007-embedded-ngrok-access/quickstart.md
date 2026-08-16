# Verification Quickstart: Embedded ngrok Public Access

This document defines the feature-007 verification journeys. It is not evidence that they have run.
Implementation must record each final-candidate result as `PASS`, `FAIL`, or `NOT RUN` with command,
date, build identity, and relevant non-secret artifact paths. A deterministic fake is never recorded
as real ngrok reachability.

**Bugfix**: 2026-08-15 — ANALYZE-S1/U1 adds unconditional source-bound ingress evidence and an
executable immutable checkpoint/worktree rollback drill.

**Bugfix**: 2026-08-15 — BUG-001 replaces the impossible source-bound gate with deterministic ngrok
Traffic Policy construction plus a focused opt-in real Basic Auth/`Subscribe` journey.

**Bugfix**: 2026-08-15 — BUG-001 analysis follow-up installs locked frontend dependencies before
dependent generation, adds the canonical dev smoke, and makes vulnerability evidence attributable.

**Bugfix**: 2026-08-16 — BUG-003 second verification reconciliation replaces active Traffic Policy,
direct-upstream, and withdraw-before-deny instructions with the private-ingress lifecycle while
retaining dated evidence as history.

**Bugfix**: 2026-08-16 — BUG-003 non-secret username reconciliation limits value-confinement checks
to token/password and makes domain/username verification explicitly observable but non-loggable.

## Recorded implementation evidence

### 2026-08-15 — US1 secure settings and Keychain checkpoint (T028)

- Build identity: base commit `6eea0cba5d06638d0c5e7cee253083e5b74c1c6e`; working-tree
  feature candidate before the required T074 immutable checkpoint.
- `PASS` — focused root descriptor/allowlist adapters:
  `go test ./ -run 'TestPublicAccess|TestDesktopServiceInventoryAndNativeEventsAreExactlyAllowlisted|TestMasterPublicAccess' -count=1`.
- `PASS` — settings, secret, Keychain, and master asset tests:
  `go test ./internal/tunnel ./internal/platform -run 'TestPublicAccess|TestKeychain|TestDarwinKeychain|TestMasterPublicAccess' -count=1`.
- `PASS` — the same focused tunnel/platform selection under the race detector:
  `go test -race ./internal/tunnel ./internal/platform -run 'TestPublicAccess|TestKeychain|TestDarwinKeychain|TestMasterPublicAccess' -count=1`.
- `PASS` — nine focused facade/settings/relaunch browser journeys:
  `npm test --prefix tests/browser -- desktop-api.spec.mjs public-access-settings.spec.mjs`.
  The run covered event-before-snapshot ordering, stale completion suppression, exact-once disposal,
  replace-without-echo, immediate input clearing, one-time generated-password Copy/dismissal, and
  secret-free preference/presence restoration without auto-start.
- `PASS` — `scripts/secret-leak-check.sh --self-test` and
  `scripts/secret-leak-check.sh`; no forbidden persistent, public, status, event, frontend-storage,
  generated-output, log, or diagnostic surface was detected by this checkpoint's deterministic scan.
- `NOT RUN` — Darwin Keychain OS integration. Command:
  `go test ./internal/platform -run TestDarwinKeychainAdapterOptInRoundTripWithoutReadbackSurface -count=1 -v`.
  The test reported its explicit opt-in skip because `FALLOUT_KEYCHAIN_INTEGRATION=1` was not
  provided. Deterministic Keychain adapter tests passed; they are not recorded as OS integration.
- `NOT RUN` — real ngrok reachability/random or reserved endpoint. No explicit opt-in provider
  credentials were supplied. No fake result is recorded as real service evidence.
- Non-blocking toolchain diagnostic: Darwin link steps warned that ngrok SDK object files were built
  for a newer macOS version than the current macOS 11 link target. All focused binaries linked and
  tests passed; package/deployment-target compatibility remains an explicit later package gate.

Evidence paths: `frontend/src/desktop-api.js`, `frontend/src/master.js`,
`tests/browser/public-access-settings.spec.mjs`, `internal/tunnel/settings_test.go`,
`internal/platform/keychain_test.go`, and `scripts/secret-leak-check.sh`.

### 2026-08-15 — US2 protected embedded endpoint checkpoint (T041)

- Build identity: Darwin 25.5.0 arm64, Go 1.26.5; base commit
  `6eea0cba5d06638d0c5e7cee253083e5b74c1c6e`; working-tree feature candidate.
- `PASS` — focused manager and pinned SDK adapter tests:
  `go test ./internal/tunnel -run 'TestEmbeddedNgrok|TestPublicAccessManager' -count=1`.
  The deterministic seam proves the exact direct `http://127.0.0.1:3690` upstream, one in-memory
  enforced Basic Auth Traffic Policy supplied to `Forward` before an endpoint can be returned,
  empty/invalid endpoint credentials rejected before forwarding, random URL and exact reserved
  domain requests, strict HTTPS validation before publication, 15-second target/30-second bound,
  stale acquisition disposal, URL withdrawal before endpoint close, maximum one active endpoint,
  and redacted provider failures. It does not claim external reachability.
- `PASS` — the same focused manager/adapter selection under the race detector:
  `go test -race ./internal/tunnel -run 'TestEmbeddedNgrok|TestPublicAccessManager' -count=1`.
- `PASS` — root composition and local-fallback tests:
  `go test . -run 'TestEmbeddedPublicAccess|TestPublicAccessCompositionUsesDirectExistingPlayerTarget|TestPublicAccess' -count=1`.
  The player listener becomes locally ready before manager initialization; public access never
  auto-starts, explicit Start/Stop is routed only through the embedded core, ready publication
  preserves safe local/public `server-info`, and a public failure leaves authoritative local mode
  usable.
- `PASS` — twelve facade and browser settings/lifecycle journeys:
  `npm test --prefix tests/browser -- desktop-api.spec.mjs public-access-settings.spec.mjs`.
  These cover exact `stopped/starting/ready/stopping/error` presentation, no pre-ready URL,
  random/reserved outcomes, URL/username Copy, transition-disabled actions, current
  generation/revision propagation, and redacted URL-free failure state.
- `PASS` — `scripts/secret-leak-check.sh`; endpoint inputs remain confined to private ephemeral
  paths and no forbidden persistent, public, event, log, diagnostic, generated, or frontend surface
  was detected.
- `NOT RUN` — real ngrok startup and Basic Auth acceptance. Command:
  `go test ./internal/tunnel -run TestEmbeddedNgrokSDKOptInProtectedDirectUpstream -count=1 -v`.
  The harness reported `NOT RUN: explicit real-ngrok integration opt-in was not provided`. No
  external credential was read or emitted; deterministic fake results above are not recorded as
  real random/reserved URL, reachability, or edge-auth evidence.
- Non-blocking toolchain diagnostic: Darwin link steps again warned that ngrok SDK object files
  were built for a newer macOS version than the current macOS 11 link target. Focused binaries
  linked and passed; deployment-target compatibility remains owned by the later package gates.

Evidence paths: `internal/tunnel/manager_test.go`, `internal/tunnel/ngrok_test.go`,
`internal/tunnel/ngrok_integration_test.go`, `app_test.go`,
`tests/browser/public-access-settings.spec.mjs`, and `scripts/secret-leak-check.sh`.

### 2026-08-15 — US3 authenticated remote-player parity checkpoint (T046)

- Build identity: Darwin 25.5.0 arm64, Go 1.26.5; base commit
  `6eea0cba5d06638d0c5e7cee253083e5b74c1c6e`; working-tree feature candidate.
- `PASS` — focused unchanged-player endpoint seam, request-boundary, stream-soak, and reconnect tests
  under the race detector:
  `go test -race ./internal/player -run 'TestEndpointAuthSeamProtectsStaticUnaryAndStreamingBeforeUnchangedPlayerBoundary|TestAuthenticatedForwardingStillAppliesOriginAndBodyLimitsInsidePlayer|TestRepresentativeThreeHourStreamReconnectSoak' -count=1`.
  The deterministic external seam rejected missing/wrong Basic Auth for static, unary, and non-empty
  streaming requests, accepted correct credentials, stripped authorization before the unchanged
  player handler, preserved same-origin/body limits, and delivered a complete snapshot plus a later
  update. This is deterministic policy/forwarding evidence, not real ngrok reachability.
- `PASS` — complete player Go package plus fixture compilation:
  `go test ./internal/player ./tests/browser/fixture-server`.
- `PASS` — complete ConnectRPC player browser file:
  `npm test --prefix tests/browser -- connectrpc-player.spec.mjs`; 11 journeys passed and the single
  credential-gated real-ngrok journey was skipped with explicit `NOT RUN` semantics. The protected
  fixture kept five clients on one recognition, converged selection/navigation/hacking/action and
  streamed updates, loaded sound manifests from the protected same origin, reconnected every client
  after forced stream closure within the five-second condition using the unchanged three-second
  retry loop, and returned `410` after endpoint disable.
- `NOT RUN` — real ngrok missing/wrong/correct Basic Auth and incremental `Subscribe`. Command:
  `go test ./internal/tunnel -run TestEmbeddedNgrokSDKOptInProtectedDirectUpstream -count=1 -v`.
  The harness reported `NOT RUN: explicit real-ngrok integration opt-in was not provided`; no
  credential value was read or emitted. The browser real-endpoint journey was skipped for the same
  absent opt-in prerequisites. No deterministic result is reported as external ingress proof.

Evidence paths: `internal/player/public_stream_test.go`, `internal/player/stream_test.go`,
`tests/browser/connectrpc-player.spec.mjs`, and `tests/browser/fixture-server/main.go`.

### 2026-08-15 — US4 protected reconfigure checkpoint (T054)

- Build identity: Darwin 25.5.0 arm64, Go 1.26.5; base commit
  `6eea0cba5d06638d0c5e7cee253083e5b74c1c6e`; working-tree feature candidate.
- `PASS` — focused manager/root race checkpoint:
  `go test -race ./internal/tunnel . -run 'TestPublicAccessManager|TestRedactedPublicAccess|TestRedactionSurvives|TestActivePublicAccessMutations|TestPartialPublicAccessMutation|TestEmbeddedPublicAccess' -count=1`.
  The 100-schedule gate converged each set of repeated concurrent same-revision mutations to one
  revision and one replacement endpoint, observed maximum active endpoint count one, and proved
  stopping/no-URL publication before old close and old close before replacement Start. Separate
  schedules disposed late success and late failure before replacement; injected close failure kept
  ownership and blocked persistence/replacement until a successful retry.
- `PASS` — partial durable-mutation acceptance reconciled actual Keychain presence after a second
  Keychain mutation failure and after settings commit failure, retained the old settings revision,
  published only stable redacted categories, and performed no mixed-revision restart.
- `PASS` — master and player production frontend builds:
  `npm run build --prefix frontend` and `npm run build --prefix client`.
- `PASS` — 16 focused desktop facade/settings browser journeys:
  `npm test --prefix tests/browser -- public-access-settings.spec.mjs desktop-api.spec.mjs`.
  Active edits require explicit confirmation, cancellation sends no mutation, replacement renders
  stopped/stopping then starting/ready with disabled transition actions, and a newer event wins over
  a late command result. Token/password inputs are cleared immediately and saved values are never
  reconstructed.
- `PASS` — `scripts/secret-leak-check.sh`; no secret-bearing value escaped the narrow private
  mutation/one-time result surfaces after reconfigure and retry coverage.
- Non-blocking toolchain diagnostic: Darwin link steps continued to warn that ngrok SDK objects
  target a newer macOS version than the current macOS 11 link target. Tests passed; the mandatory
  later macOS 13+ package/linkage gate remains authoritative.

Evidence paths: `internal/tunnel/manager_test.go`, `internal/tunnel/redaction_test.go`, `app_test.go`,
`tests/browser/public-access-settings.spec.mjs`, and `scripts/secret-leak-check.sh`.

### 2026-08-15 — US5 local/LAN failure-isolation checkpoint (T060)

- Build identity: Darwin 25.5.0 arm64, Go 1.26.5; base commit
  `6eea0cba5d06638d0c5e7cee253083e5b74c1c6e`; working-tree feature candidate.
- `PASS` — focused tunnel/app failure matrix under the race detector:
  `go test -race ./internal/tunnel . -run 'TestPublicAccessManager|TestEmbeddedNgrok|TestRedactedPublicAccess|TestRedactionSurvives|TestEmbeddedPublicAccess|TestUnexpectedPublicEndpointFailure|TestActivePublicAccess|TestPartialPublicAccess' -count=1`.
  Deterministic cases covered invalid/revoked token categories, no network, DNS/timeout, domain
  conflict, Keychain locked/denied/unavailable, policy/provider construction failure, unexpected
  endpoint completion, failed Close, settings/store failure, stale completion, and a later Start
  without application restart. Unexpected completion withdrew the URL first; failed Close retained
  ownership, and successful retry closed the prior endpoint before replacement with maximum active
  endpoint count one. All diagnostics remained fixed and redacted.
- `PASS` — protected/local player plus fallback browser checkpoint:
  `npm test --prefix tests/browser -- public-access-fallback.spec.mjs connectrpc-player.spec.mjs`;
  12 journeys passed and one real-ngrok journey was skipped. Across thirteen injected public
  failure states, the local generated-Connect player retained its character, alternated navigation,
  entered hacking, emitted an action, requested sound manifests, reconnected after forced stream
  closure within five seconds, suppressed a stale ready URL, and observed a later ready generation.
  The protected fixture separately retained missing/wrong/correct Basic Auth coverage for static,
  unary, and streaming paths and five-client convergence. No second player or proxy runtime was
  introduced.
- `PASS` — `scripts/secret-leak-check.sh`; no forbidden secret-bearing persistent, public, event,
  status, log, diagnostic, generated-output, frontend, or fixture surface was detected.
- `NOT RUN` — real invalid/revoked token, reserved-domain conflict, provider disconnect, and offline
  ngrok cases. No explicit real-ngrok opt-in credentials were supplied, so no provider request was
  attempted and no deterministic result is reported as real external evidence.
- Non-blocking toolchain diagnostic: Darwin link steps warned that ngrok SDK objects target a newer
  macOS version than the current macOS 11 link target. All focused binaries linked and tests passed;
  deployment compatibility remains owned by the mandatory package gates.

Evidence paths: `app_test.go`, `internal/tunnel/ngrok_test.go`, `internal/tunnel/manager.go`,
`tests/browser/public-access-fallback.spec.mjs`, `tests/browser/fixture-server/main.go`, and
`scripts/secret-leak-check.sh`.

### 2026-08-15 — US6 packaged lifecycle checkpoint attempt (T067, INCOMPLETE)

- Build identity: Darwin 25.5.0 arm64, Go 1.26.5; base commit
  `6eea0cba5d06638d0c5e7cee253083e5b74c1c6e`; working-tree feature candidate.
- `PASS` — deterministic manager/SDK/app/Wails lifecycle race gate after adding an explicit
  shutdown-before-late-acquisition barrier:
  `go test -race ./internal/tunnel . -run 'TestPublicAccessManagerShutdown|TestEmbeddedNgrok(StartCancellation|EndpointConcurrentClose|EndpointCloseDoesNotTrust|EndpointDone|EndpointCloseFailure)|TestApplicationShutdown|TestWailsLifecycle' -count=1`.
  The gate includes 100 post-cancel acquisition schedules with zero active endpoints, concurrent
  reconfigure/shutdown revision convergence, bounded context-ignoring Close, concurrent SDK Close,
  retained cleanup ownership, and repeated App/Wails retry.
- `PASS` — `scripts/secret-leak-check.sh`.
- `PASS` — canonical `go run ./cmd/build package` and
  `scripts/verify-macos-app.sh 'build/bin/Fallout Terminal.app'`. The verified arm64/macOS 13 bundle
  had canonical bundle-manifest SHA-256
  `9829db2f7612bf2bf56fd553e510a64325c9b6efc7970b8d7509da194aad3ea6`.
- `FAIL` — mandatory credential-free packaged lifecycle harness:
  `scripts/public-access-macos-smoke.sh 'build/bin/Fallout Terminal.app'` stopped at the first normal
  window-close assertion with the exact redacted diagnostic
  `public-access-macos-smoke: FAIL: normal window close exceeded five seconds`.
  The harness trap removed the packaged process; a follow-up process query returned no matching
  application process. T067 remains incomplete and no later implementation/cutover task is
  authorized by this checkpoint.
- `NOT RUN` — packaged `Cmd+Q`, partial startup, forced owner loss, stopped relaunch, and real stale
  public URL probe were not reached after the mandatory normal-close failure. No real-ngrok opt-in
  credentials or public URL were supplied, read, or printed.

Evidence paths: `internal/tunnel/manager_test.go`, `internal/tunnel/ngrok_test.go`, `app_test.go`,
`wails_host_test.go`, `scripts/public-access-macos-smoke.sh`, and
`build/bin/Fallout Terminal.app`.

### 2026-08-15 — US6 packaged lifecycle resumed attempt (T067, INCOMPLETE)

- Build identity: Darwin 25.5.0 arm64, Go 1.26.5; base commit
  `6eea0cba5d06638d0c5e7cee253083e5b74c1c6e`; working-tree feature candidate.
- `PASS` — test-first native close ownership. The focused test was first observed RED because no
  explicit quit request was registered, then GREEN after the master window began joining
  `common:WindowClosing` and Darwin `WindowWillClose` into one application quit intent:
  `go test . -run '^TestMasterWindowCloseExplicitlyRequestsApplicationQuit$' -count=1`.
- `PASS` — the post-change deterministic lifecycle race gate:
  `go test -race ./internal/tunnel . -run 'TestPublicAccessManagerShutdown|TestEmbeddedNgrok(StartCancellation|EndpointConcurrentClose|EndpointCloseDoesNotTrust|EndpointDone|EndpointCloseFailure)|TestApplicationShutdown|TestWailsLifecycle|TestMasterWindowClose' -count=1`.
  Both packages passed; the manager schedules still reported zero leaked active endpoints.
- `PASS` — `scripts/secret-leak-check.sh`.
- `PASS` — canonical `go run ./cmd/build package`, followed by
  `scripts/verify-macos-app.sh 'build/bin/Fallout Terminal.app'`. The rebuilt verified
  arm64/macOS 13 bundle has canonical bundle-manifest SHA-256
  `2f5c39dbf201f37ef7dedc9b605029c75f55d6ac9b9eb971fe655efd77f529af`.
- `FAIL` — the mandatory credential-free packaged lifecycle command
  `scripts/public-access-macos-smoke.sh 'build/bin/Fallout Terminal.app'` again returned the exact
  redacted diagnostic `public-access-macos-smoke: FAIL: normal window close exceeded five seconds`.
  The original `System Events` keyboard route was proven unavailable with macOS error `-25211`
  (`osascript is not allowed assistive access`). Direct application and exact-bundle Apple Events
  were also exercised without Accessibility; they either bypassed the interactive Wails close path
  or waited on the Apple Event reply and did not make the packaged process exit inside five
  seconds. The harness cleanup removed its target process after each failed attempt.
- `NOT RUN` — packaged `Cmd+Q`, partial startup, forced owner loss, stopped relaunch, and the real
  stale-public-URL probe remain unreached because the first mandatory close assertion failed. No
  real-ngrok opt-in credentials or URL were supplied, read, logged, or requested.
- Blocking result: T067 remains unchecked. T068 and every later cutover task remain unauthorized;
  no weaker close-plus-forced-quit substitute was accepted as packaged normal-close evidence.

Evidence paths: `wails_host.go`, `wails_host_test.go`,
`scripts/public-access-macos-smoke.sh`, and `build/bin/Fallout Terminal.app`.

### 2026-08-15 — US6 packaged lifecycle resumed checkpoint (T067, PASS with one manual follow-up)

- Build identity: Darwin 25.5.0 arm64, Go 1.26.5; base commit
  `6eea0cba5d06638d0c5e7cee253083e5b74c1c6e`; working-tree feature candidate.
- `PASS` — test-first synchronous native close ownership. The focused test was observed RED while
  the callback returned before the application entered termination, then GREEN after both
  `common:WindowClosing` and Darwin `WindowWillClose` began synchronously joining one exact-once
  quit intent:
  `go test . -run '^TestMasterWindowCloseExplicitlyRequestsApplicationQuit$' -count=1`.
- `PASS` — a race run exposed that the scheduled lifecycle fake could publish `closed` before its
  active-resource decrement and let a retry return early. The fake now publishes its idempotent
  closed state only after resource accounting and `Done` settle. The exact regression passed ten
  race iterations:
  `go test -race ./internal/tunnel -run '^TestPublicAccessManagerShutdownBoundsContextIgnoringCloseAndRetainsRetryOwnership$' -count=10`.
- `PASS` — complete focused manager/SDK/App/Wails lifecycle race subset:
  `go test -race ./internal/tunnel . -run 'TestPublicAccessManagerShutdown|TestEmbeddedNgrok(StartCancellation|EndpointConcurrentClose|EndpointCloseDoesNotTrust|EndpointDone|EndpointCloseFailure)|TestApplicationShutdown|TestWailsLifecycle|TestMasterWindowClose' -count=1`.
  Both packages passed; all deterministic shutdown schedules finished with zero active endpoints
  and maximum active endpoint count one.
- `PASS` — canonical `go run ./cmd/build package`, `scripts/secret-leak-check.sh`, and
  `scripts/verify-macos-app.sh 'build/bin/Fallout Terminal.app'`. The verified arm64/macOS 13
  package has canonical bundle-manifest SHA-256
  `580f26281f7252258f513fbed9c33651cb33af77400ef59241a54f2a36399de5`.
- `PASS` — remaining credential-free packaged lifecycle journeys:
  `env FALLOUT_PUBLIC_ACCESS_MANUAL_CLOSE_NOT_RUN=1 scripts/public-access-macos-smoke.sh 'build/bin/Fallout Terminal.app'`.
  LaunchServices startup used no application arguments or provider credentials. Graceful
  application quit completed in `0s` against a `5s` limit; partial second-instance startup left
  the original local server usable; forced owner loss left no owned packaged process; relaunch
  restored local mode without automatic endpoint reuse; final relaunch cleanup completed in `0s`
  against the same `5s` limit.
- `NOT RUN` — interactive red-button/`Cmd+W` automation was explicitly deferred for manual
  follow-up by the user. macOS CoreGraphics preflight reported `denied`; this item is not described
  as a PASS and the default harness still fails rather than silently skipping it.
- `NOT RUN` — both real stale-public-URL probes require explicit opt-in plus a non-secret active
  URL. No real-ngrok credentials or URL were supplied, read, logged, or requested.

Evidence paths: `internal/tunnel/manager_test.go`, `wails_host.go`, `wails_host_test.go`,
`scripts/public-access-macos-smoke.sh`, and `build/bin/Fallout Terminal.app`.

### 2026-08-15 — Embedded-only pre-removal parity checkpoint (T073, PASS)

- Build identity: Darwin 25.5.0 arm64, Go 1.26.5; base commit
  `6eea0cba5d06638d0c5e7cee253083e5b74c1c6e`; working-tree pre-removal candidate. The verified
  canonical bundle-manifest SHA-256 is
  `c25868f31ec25e78e5d385bc6c8a2cbdbcaa0ab9073e85c83babe1494f7a9318`.
- `PASS` — empty `gofmt -l .`, locked installs in `frontend/`, `client/`, and `tests/browser/`,
  tool-module, protobuf format/lint/drift/breaking, Wails binding/cutover, exact dependency/license,
  long-canary leak, `go vet ./...`, `go test ./...`, and `go test -race ./...` gates. Session JSON
  version 1 and player-config version 1 remain unchanged.
- `PASS` — the full Playwright gate: `npm test --prefix tests/browser`; 36 journeys passed and the
  credential-gated real-ngrok journey was skipped. Protected static, unary, and non-empty streaming
  requests rejected missing/wrong Basic Auth, accepted correct credentials, while direct local/LAN
  paths remained outside the challenge. Five-client convergence, navigation, hacking, sound,
  updates, stale shutdown, and reconnect all passed.
- `PASS` — focused race proof:
  `go test -race ./internal/tunnel ./internal/player . -run '^(TestEmbeddedNgrokAdapterAttachesBasicAuthPolicyToDirectUpstreamBeforeReturningEndpoint|TestEndpointAuthSeamProtectsStaticUnaryAndStreamingBeforeUnchangedPlayerBoundary|TestRepresentativeThreeHourStreamReconnectSoak|TestUnexpectedPublicEndpointFailureRetainsCleanupOwnershipBeforeRetryWithoutAppRestart|TestPublicAccessCompositionUsesDirectExistingPlayerTarget)$' -count=1`.
  It proves the only upstream is `http://127.0.0.1:3690`, a non-empty endpoint Basic Auth Traffic
  Policy is passed to `Forward` before URL publication, scoped secret use does not escape, and an
  unexpected `Done` withdraws the URL and joins owned cleanup before retry. The exact unexpected
  `Done` regression also passed 20 race iterations before the full race gate.
- `PASS` — `scripts/reproducible-build-check.sh`; two complete protobuf/player/binding/native/app
  package builds were byte-identical at the digest above, with zero repository drift. Separate
  direct `go run ./cmd/build build`, `go run ./cmd/build package`,
  `scripts/verify-macos-app.sh 'build/bin/Fallout Terminal.app'`, and
  `scripts/hash-macos-app.sh 'build/bin/Fallout Terminal.app'` runs passed and reproduced the same
  digest. The signed arm64/macOS 13 package contains reviewed notices and native
  Security/CoreFoundation linkage, and contains no provider binary, PATH lookup, runtime download,
  or provider credential.
- `PASS` — credential-free packaged lifecycle:
  `env FALLOUT_PUBLIC_ACCESS_MANUAL_CLOSE_NOT_RUN=1 scripts/public-access-macos-smoke.sh 'build/bin/Fallout Terminal.app'`.
  LaunchServices startup used no application arguments or provider credentials; Cmd+Q-equivalent
  quit, partial startup, forced owner loss, and relaunch retained local mode, with each measured
  cleanup completing in `0s` against the five-second limit.
- `PASS` — `scripts/legacy-public-access-check.sh --diagnose` reported the bounded rollback-only CLI
  source inventory without finding a provider executable in the candidate package. This is not the
  strict final legacy gate: the default command remains intentionally failing until T075 removes
  those sources, and must pass afterward.
- The first broad unit attempt exposed two convention seams and one verifier-contract seam; the
  exact failures were repaired without weakening coverage. The first full race attempt then exposed
  a real retry-vs-monitor-cleanup race. `Start`, `Stop`, and `Reconfigure` now join the single owned
  unexpected-`Done` cleanup before retry, after URL withdrawal. Focused regressions, full unit, and
  full race all passed afterward.
- `NOT RUN` — real ngrok random/reserved URL, real endpoint Basic Auth/streaming, invalid token,
  network/domain failures, and stale public URL probes. No explicit opt-in credentials were
  supplied and no provider request was made.
- `NOT RUN` — interactive red-button/`Cmd+W` packaged close automation, explicitly deferred by the
  user for manual verification. It is not counted as a PASS; the default harness remains strict.
- Non-blocking diagnostic: test-only Darwin link steps warn that SDK objects were built for macOS
  26 while the generic test linker targets macOS 11. The authoritative packaged binary independently
  passed its exact arm64/macOS 13 minimum-version and native-framework inspection.

Evidence paths: `internal/tunnel/manager.go`, `internal/tunnel/manager_test.go`,
`internal/tunnel/ngrok_test.go`, `internal/player/public_stream_test.go`,
`scripts/secret-leak-check.sh`, `scripts/reproducible-build-check.sh`,
`scripts/verify-macos-app.sh`, `tests/browser/connectrpc-player.spec.mjs`, and
`build/bin/Fallout Terminal.app`.

### 2026-08-16 — Immutable checkpoint rollback drill (T074, FAIL)

- User-approved checkpoint commit:
  `558c84780687c2de81b4457216f397865b68939e` (`feat: checkpoint embedded ngrok public access`).
  The primary worktree was clean immediately after the commit. T073 package digest:
  `c25868f31ec25e78e5d385bc6c8a2cbdbcaa0ab9073e85c83babe1494f7a9318`.
- `PASS` — `/tmp/fallout-terminal-007-rollback.DEiePK` was created as a detached task-owned
  worktree. `git worktree list --porcelain`, `git rev-parse HEAD`, and
  `git status --porcelain=v1 --untracked-files=all` proved the exact 40-hex checkpoint identity and
  an initially clean checkout.
- `FAIL` — the first required canonical command, `go run ./cmd/build build`, stopped in
  `verify protobuf and generated clients` before any package build. Exact non-secret cause:
  `Cannot find module './client/node_modules/@bufbuild/protoc-gen-es/package.json'` followed by
  `protoc-gen-es version is ; expected 2.13.0`. The clean worktree has no `client/node_modules`, and
  the canonical graph currently verifies protobuf before its later locked player dependency
  install. Node reported version `v26.7.0`; no credential or secret was read or printed.
- `NOT RUN` — canonical package build, packaged app verification, rollback digest calculation, and
  byte-for-byte comparison were not reached after the mandatory build failure.
- Per the task contract, the unvalidated detached worktree is preserved for inspection and was not
  removed. T074 remains incomplete; T075 CLI deletion is forbidden. Installing dependencies by hand
  inside the drill would not satisfy the exact canonical-command proof and was not used as a
  workaround.

### 2026-08-16 — Clean-build order correction and T073 requalification

- `PASS` — test-first canonical-order regression. The focused buildtool suite was first observed
  RED for `prepare`, `build`, `dev`, `run`, and `package`, because locked player dependency install
  followed protobuf verification. `preparePlan` now runs exactly `npm ci --prefix client` before
  `scripts/proto-check.sh` for every action; protobuf generation, version enforcement, build
  ownership, and all later nodes are unchanged. The complete `internal/buildtool` suite passed.
- `PASS` — direct `go run ./cmd/build build` began with `install locked player dependencies`, then
  completed protobuf verification/generation, player build, bindings, master build, and native
  compilation without any manual dependency installation.
- `PASS` — empty `gofmt -l .`, `go vet ./...`, `go test ./...`, `go test -race ./...`, and
  `scripts/secret-leak-check.sh` on the corrected candidate.
- `PASS` — `scripts/reproducible-build-check.sh`; both corrected canonical package runs installed
  locked player dependencies before protobuf verification, produced zero repository drift, passed
  package inspection, and reproduced the T073 digest byte-for-byte:
  `c25868f31ec25e78e5d385bc6c8a2cbdbcaa0ab9073e85c83babe1494f7a9318`.
- The failed checkpoint `558c84780687c2de81b4457216f397865b68939e` remains immutable and its
  unvalidated detached worktree remains preserved. It is historical failed evidence and cannot
  satisfy T074; a new user-authorized checkpoint and clean detached drill are required.

### 2026-08-16 — Immutable checkpoint rollback drill resumed (T074, PASS)

- User-authorized corrected checkpoint commit:
  `ba32d1a57bb0a33227841c6c189939c91dec6bf3`
  (`fix: install player tools before protobuf verification`). The primary working tree was clean
  immediately after the commit.
- `PASS` — `/tmp/fallout-terminal-007-rollback-fixed.ycaZYH` was created as a detached task-owned
  worktree. `git worktree list --porcelain`, `git rev-parse HEAD`, and empty
  `git status --porcelain=v1 --untracked-files=all` output proved the exact checkpoint identity and
  clean tracked state.
- `PASS` — without a preliminary manual install, canonical `go run ./cmd/build build` began with
  locked player dependency installation and completed protobuf generation, both frontends,
  bindings, and native compilation. Canonical `go run ./cmd/build package` independently repeated
  the locked install and completed the full signed package graph.
- `PASS` — `scripts/verify-macos-app.sh 'build/bin/Fallout Terminal.app'` verified arm64, macOS 13,
  native frameworks, reviewed resources/notices, signature, and absence of provider executable or
  PATH runtime. `scripts/hash-macos-app.sh 'build/bin/Fallout Terminal.app'` returned
  `c25868f31ec25e78e5d385bc6c8a2cbdbcaa0ab9073e85c83babe1494f7a9318`, exactly equal to the
  requalified T073 digest. The detached checkout remained free of tracked/untracked repository
  drift after both builds.
- `PASS` — the exact validated worktree appeared in `git worktree list --porcelain` immediately
  before cleanup and `git worktree remove /tmp/fallout-terminal-007-rollback-fixed.ycaZYH`
  completed successfully. The earlier failed worktree at
  `/tmp/fallout-terminal-007-rollback.DEiePK` remains preserved as recorded failure evidence; it was
  not confused with or used by this successful drill.
- T074 now unblocks sequential T075. The checkpoint is a rollback reference only; it does not add a
  production runtime switch.

### 2026-08-16 — Final post-CLI-removal deterministic candidate (T078, PASS)

- Build identity: base commit `ba32d1a57bb0a33227841c6c189939c91dec6bf3` plus the recorded
  T075–T077 working-tree cutover. The candidate has one embedded SDK runtime and no external
  process/config/parser implementation.
- `PASS` — exact clean-checkout-safe deterministic sequence from section 1, in order: empty
  `gofmt -l .`; locked `npm ci` in `frontend/`, `client/`, and `tests/browser/`; tool-module,
  protobuf format/lint/generation/drift/breaking, Wails binding/cutover, dependency/license,
  strict legacy-runtime, and secret-leak gates; `go vet ./...`; `go test ./...`; and
  `go test -race ./...`. The negative protobuf drift fixture printed its expected rejection and
  returned success. The resolved inventory contained 171 modules and 244 module-graph edges.
- `PASS` — both production frontend builds and the complete Playwright gate:
  `npm run build --prefix frontend`, `npm run build --prefix client`, and
  `npm test --prefix tests/browser`. The browser run passed 36 deterministic journeys and skipped
  exactly one real authenticated public-endpoint journey with explicit `NOT RUN` semantics because
  no opt-in credentials were provided.
- `PASS` — `scripts/reproducible-build-check.sh`; its two full package graphs agreed. The separate
  mandatory `go run ./cmd/build build`, `go run ./cmd/build package`, and
  `scripts/verify-macos-app.sh "build/bin/Fallout Terminal.app"` also passed. The final verified
  arm64/macOS-13 ad-hoc bundle contains reviewed notices, native Keychain frameworks, offline
  resources, no provider executable/PATH runtime, and has canonical bundle-manifest SHA-256
  `1992f2fedf2d84e356cc3067383d72d3f4f43abfc2492e2276c949b4358b49a5` (17,844 KiB bundle;
  15,975,824-byte main executable).
- `PASS` — credential-free packaged lifecycle:
  `env FALLOUT_PUBLIC_ACCESS_MANUAL_CLOSE_NOT_RUN=1 scripts/public-access-macos-smoke.sh "build/bin/Fallout Terminal.app"`.
  LaunchServices/double-click semantics reached local mode without Terminal, application arguments,
  provider credentials, or an installed provider executable. Deferred-close cleanup, Cmd+Q-equivalent
  cleanup, partial second-instance startup, forced owner loss, offline/local relaunch, and final Quit
  all met the five-second budget; measured cleanup results were `0s`.
- `PASS` — separate canonical `go run ./cmd/build dev` master/player smoke with no separately
  started frontend or player server. The native master window visibly reached
  `ЛОКАЛЬНЫЙ РЕЖИМ ГОТОВ · http://127.0.0.1:3690`; static `/` and live typed `SoundManifest`
  returned `200`; a generated Connect client received a non-empty initial `Subscribe` snapshot at
  revision 1; interrupt released listener 3690 on the first 100ms probe. The in-app Browser backend
  was unavailable (`[]`), so visual public-settings coverage remains attributable to the passing
  Playwright settings journeys rather than being misreported as Browser automation.
- `NOT RUN` — interactive red-button/`Cmd+W` native close remains the user-deferred manual follow-up;
  the harness reported it explicitly and did not count it as `PASS`.
- `NOT RUN` — real stale-public-URL probes and all real provider reachability/authentication checks
  lacked explicit opt-in credentials and a non-secret active URL; no external request was made.
- Non-blocking diagnostic: Go unit link steps repeated the known SDK object minimum-version warning
  under their macOS-11 test link target. The authoritative packaged executable verification passed
  exact arm64 architecture, `LSMinimumSystemVersion=13.0`, Mach-O minimum 13.0, native framework,
  entitlements, signature, and resource gates.

### 2026-08-16 — Conditional real-provider evidence (T079, NOT RUN)

- Opt-in prerequisite presence check (values were never read or printed):
  `FALLOUT_NGROK_INTEGRATION`, `FALLOUT_NGROK_AUTHTOKEN`,
  `FALLOUT_NGROK_RESERVED_DOMAIN`, `FALLOUT_PUBLIC_TEST_URL`,
  `FALLOUT_PUBLIC_TEST_USERNAME`, and `FALLOUT_PUBLIC_TEST_PASSWORD` were all absent.
- `NOT RUN` — real random URL startup and the 20-attempt SC-001 sample: no personal token or
  explicit opt-in was provided.
- `NOT RUN` — real reserved-domain success, unavailable-domain conflict, invalid token, revoked
  token, no-network/provider timeout, and disconnect-after-ready recovery: the required personal
  account/token/domain/network prerequisites were absent.
- `NOT RUN` — missing, wrong, and correct Basic Auth against real public static, unary, and
  non-empty incremental `Subscribe` requests: no active public URL or credential opt-in existed.
- `NOT RUN` — real public multi-client convergence, reconnect, navigation, hacking, sound,
  stop/reconfigure, failure recovery, and stale-URL probes: no active opted-in endpoint existed.
- The focused Go harness command
  `go test ./internal/tunnel -run '^TestEmbeddedNgrokSDKOptInProtectedDirectUpstream$' -count=1 -v`
  reported `NOT RUN: explicit real-ngrok integration opt-in was not provided` and made no provider
  request. The focused browser command
  `npm test --prefix tests/browser -- connectrpc-player.spec.mjs` passed 11 deterministic journeys
  and skipped exactly the real authenticated endpoint journey. Those deterministic passes are not
  reported as external evidence.

### 2026-08-16 — Packaged UI/Keychain/public smoke (T080, PARTIAL PASS / public NOT RUN)

- `PASS` — unconditional package verification and credential-free LaunchServices smoke:
  `scripts/verify-macos-app.sh "build/bin/Fallout Terminal.app"` and
  `env FALLOUT_PUBLIC_ACCESS_MANUAL_CLOSE_NOT_RUN=1 scripts/public-access-macos-smoke.sh "build/bin/Fallout Terminal.app"`.
  The exact digest remained
  `1992f2fedf2d84e356cc3067383d72d3f4f43abfc2492e2276c949b4358b49a5`; architecture,
  macOS-13 minimum, signature, native frameworks, entitlements, offline resources, absence of a
  bundled provider executable/PATH lookup, local readiness, partial startup, forced-owner-loss
  relaunch, Cmd+Q-equivalent cleanup, and the five-second lifecycle budget all passed.
- `PASS` — deterministic Keychain adapter selection:
  `go test ./internal/platform -run 'Test.*Keychain' -count=1 -v`. Stable service/account names,
  attribute-only presence, replace/update/add/delete semantics, scoped read clearing, and
  locked/denied/unavailable/cancelled redaction passed.
- `NOT RUN` — real isolated macOS Keychain round-trip because
  `FALLOUT_KEYCHAIN_INTEGRATION` was absent. The harness reported its explicit skip and no temporary
  Keychain item was created.
- `NOT RUN` — packaged UI personal-token save/presence, public Start, authenticated player static
  and streaming access, Stop, and public Quit because no provider-token opt-in was supplied. The
  user retained this real packaged public journey for manual completion; deterministic UI/adapter
  evidence is not promoted to a packaged public `PASS`.
- `NOT RUN` — literal host-without-installed-provider-binary observation. A provider CLI is present
  on this development host, so it was not claimed absent or uninstalled. Source/package scans prove
  that the application has no CLI lookup, subprocess, bundled executable, environment launch path,
  or runtime download; the actual no-installed-binary public UI journey remains part of the same
  manual follow-up.
- `NOT RUN` — interactive red-button/`Cmd+W` and real stale-URL probes remain the separately recorded
  user-deferred/credential-gated cases. All unconditional local cleanup paths still passed in `0s`.

### 2026-08-16 — Conditional distribution and provider-plan gates (T081, NOT RUN)

- Prerequisite presence check found no `DEVELOPER_ID_APPLICATION` reference, no
  `NOTARYTOOL_KEYCHAIN_PROFILE` reference, zero installed Developer ID Application identities, and
  no release DMG. No signing identity or Keychain profile value was printed.
- `NOT RUN` — Developer ID signing and hardened-runtime distribution signature: no matching identity
  or configured identity reference was available. The local ad-hoc personal package verification is
  not reported as Developer ID evidence.
- `NOT RUN` — Apple notarization, stapling, release DMG construction/stapling, and Gatekeeper release
  assessment: the Developer ID and notary profile prerequisites were absent, so
  `scripts/build-macos.sh --preflight` and the mutating release pipeline were not invoked.
- `NOT RUN` — paid/reserved-domain and provider-account capability gates: no personal provider token,
  reserved-domain prerequisite, or explicit real-provider opt-in was supplied. No provider API or
  endpoint request was made.

### 2026-08-16 — Final requalification and vulnerability disposition (T082, PASS)

- Build identity: immutable rollback reference
  `ba32d1a57bb0a33227841c6c189939c91dec6bf3`; post-CLI-removal working-tree candidate using the
  module-selected Go 1.26.6 toolchain. `go list -m all` resolved 171 modules and `go mod graph`
  contained 244 edges. The preserved detached rollback worktree remains registered at
  `/private/tmp/fallout-terminal-007-rollback.DEiePK`; the immutable commit resolves successfully.
- `PASS` — the exact clean-checkout-safe sequence in section 1: empty `gofmt -l .`, locked installs
  in all three npm projects before dependent generation, clean diff check, tool-module, protobuf
  format/lint/drift/breaking, Wails binding/cutover, dependency/license, strict legacy, secret-leak,
  vet, full unit, full race, both frontend builds, and full Playwright. Playwright reported 36
  passed and one explicit credential-gated real-endpoint skip. The negative protobuf drift fixture
  emitted its expected rejection diagnostic and exited successfully.
- `PASS` — `scripts/reproducible-build-check.sh` produced two byte-identical inspected packages.
  Separate direct `go run ./cmd/build build`, `go run ./cmd/build package`, package verification,
  and bundle hashing reproduced canonical bundle-manifest SHA-256
  `dcc6d5af31e996e2db2bf84634521207d5b8922c64b01c65c4642852ed59f023`.
  The bundle is 17,844 KiB and its main executable is 15,975,824 bytes; verifier evidence is arm64,
  macOS 13.0, native frameworks, reviewed notices, complete resources/entitlements, valid final
  signature, and no provider executable or PATH runtime.
- `PASS` — bounded canonical `go run ./cmd/build dev` smoke without a second player/frontend server:
  local static `/` returned 200, typed `SoundManifest` returned one ambient asset, and `Subscribe`
  began with a complete snapshot at revision 1. Interrupt released the sole port-3690 listener
  immediately within the five-second budget.
- `PASS` — final credential-free packaged lifecycle smoke. Cmd+Q-equivalent quit, partial startup,
  forced owner loss, local-server preservation, and stopped relaunch completed with zero-second
  measured cleanup. Interactive red-button/`Cmd+W` remains the user-deferred `NOT RUN`; both real
  stale-public-URL probes remain credential-gated `NOT RUN`.
- `PASS` — dated vulnerability review on 2026-08-16. The first official Go vulnerability database
  scan against Go 1.26.5 found five reachable standard-library advisories: `GO-2026-6218`,
  `GO-2026-6090`, `GO-2026-6089`, `GO-2026-5972`, and `GO-2026-5026`, all fixed in Go 1.26.6.
  The root module minimum was raised to 1.26.6; the repeated official `govulncheck ./...` returned
  `No vulnerabilities found`. Official npm registry audits of the committed `frontend/`, `client/`,
  and `tests/browser/` lockfiles each returned zero vulnerabilities. No applicable finding remains
  without disposition.
- Non-blocking diagnostic: test-only Darwin link steps still warn that SDK objects target macOS 26
  while the generic test linker targets macOS 11. Unit/race binaries link successfully; the final
  packaged executable independently passed the authoritative arm64/macOS 13 minimum-version and
  native-framework verifier.

Success-criteria reconciliation for this candidate:

- `SC-001` — deterministic timeout/cancellation schedules `PASS`; 20 real starts `NOT RUN`.
- `SC-002` — deterministic static/unary/streaming auth and unaffected local/LAN boundary `PASS`;
  real-edge portion `NOT RUN`.
- `SC-003` — deterministic protected browser parity/reconnect `PASS`; real public-address portion
  `NOT RUN`.
- `SC-004` — simulated provider/network/token/domain/timeout/secure-store fallback matrix `PASS`.
- `SC-005` — deterministic 100-schedule shutdown and available packaged cleanup paths `PASS`; real
  stale-URL probes and deferred interactive close remain `NOT RUN`.
- `SC-006` — 100 repeated concurrent start/stop/reconfigure schedules `PASS` with one endpoint and
  no stale publication.
- `SC-007` — final secret-leak scan `PASS` with zero forbidden secret surfaces.
- `SC-008` — offline packaged launch/local lifecycle `PASS`; packaged public UI configuration,
  authenticated access, and no-installed-host-binary observation `NOT RUN`.
- `SC-009` — deterministic 100-save/relaunch non-secret restoration and non-readback `PASS`.
- `SC-010` — session JSON v1, player-config v1, local broadcast, roles, authoritative state, and all
  compatibility gates `PASS` without migration.
- `SC-011` — evidence labeling `PASS`: all unavailable provider, network, signing, notarization,
  plan, interactive, and host-prerequisite gates remain explicit `NOT RUN`.
- `SC-012` — complete password length boundary tests `PASS`.
- `SC-013` — real-edge reject/retry/no-cooldown acceptance `NOT RUN`; deterministic edge-policy
  acceptance is not substituted for it.

### 2026-08-16 — Final source/generated/package/documentation audit (T083, PASS)

- `PASS` — production composition contains exactly one tunnel construction:
  `main.go` creates one `PublicAccessManager` with `NewEmbeddedNgrokService`; the sole provider
  adapter is `internal/tunnel/ngrok.go`, forwarding only to `http://127.0.0.1:3690`. The surviving
  `service.go` contains provider-neutral lifecycle interfaces only. The external process/config,
  Darwin guardian, subprocess lifecycle, log parser, and process-only tests are deleted.
- `PASS` — strict active source/build/package/documentation scan:
  `scripts/legacy-public-access-check.sh` found no external provider execution, PATH lookup,
  `NGROK_BIN`, guardian, log-derived URL, shared token/domain, packaged env/argument instruction,
  bundled provider executable, runtime download, or second production implementation. Historical
  migration specs remain history and are not active runtime or operating guidance. README and
  templates describe UI configuration with a personal token and Keychain storage.
- `PASS` — final generated/package/security scan: protobuf generation and drift verification retained
  schema revision `aff02dd5920ecde1e3f682b06201a509dde354e6216d6a8f1ced652b4ab35112`
  and deterministic contract digest
  `f712f68df524d04fe1c30754818d6911c6b0c0d48f1b8dad1372b65ef00ab507`;
  Wails exposes exactly 30 allowlisted desktop methods; dependency/license, secret-leak, package
  verifier, and diff-whitespace gates passed. Session JSON and player-config remain version 1.
- Final candidate package digest:
  `dcc6d5af31e996e2db2bf84634521207d5b8922c64b01c65c4642852ed59f023`.
  Immutable pre-removal rollback package digest:
  `c25868f31ec25e78e5d385bc6c8a2cbdbcaa0ab9073e85c83babe1494f7a9318` at commit
  `ba32d1a57bb0a33227841c6c189939c91dec6bf3`. They intentionally differ because the final candidate
  removes the legacy implementation and uses the security-patched Go 1.26.6 toolchain.
- Final FR evidence is attached through the complete `tasks.md` traceability matrix: FR-001–FR-009
  map to settings/Keychain/private-input/non-readback tests; FR-010–FR-017 to one-server,
  protobuf/bindings, local/LAN, and version-1 compatibility gates; FR-018–FR-025 to embedded policy,
  exact upstream, public auth, streaming, browser parity, and reconnect tests; FR-026–FR-036 to
  failure isolation, lifecycle, concurrency, shutdown, and secret scans; FR-037–FR-044 to CLI
  removal, canonical build graph, reproducibility, package, docs, and evidence labeling; and
  FR-045–FR-051 to real-provider/release conditional gates and explicit `NOT RUN` handling. The
  per-SC disposition immediately above records every SC-001–SC-013 result without upgrading fake or
  unavailable evidence.
- Remaining gaps are explicit and unchanged: real provider random/reserved/auth/streaming/failure
  journeys, packaged public UI with no installed host provider binary, Darwin OS Keychain opt-in,
  Developer ID/notary/DMG/provider-plan checks, real stale URLs, and the user-deferred interactive
  red-button/`Cmd+W` close are `NOT RUN`. No deterministic or local result is claimed as those
  external acceptances.

### 2026-08-15 — Historical US2 source-bound checkpoint that triggered BUG-001

- Build identity: Darwin 25.5.0 arm64, Go 1.26.5; base commit
  `6eea0cba5d06638d0c5e7cee253083e5b74c1c6e`; uncommitted feature candidate.
- `PASS` — deterministic embedded Agent/Forward adapter tests and the injected no-fallback bind
  failure test. The adapter uses the pinned SDK constructor, omits Traffic Policy, distinguishes
  random from exact reserved URL requests, validates returned HTTPS authorities, redacts provider
  failures, and owns idempotent Forwarder close plus Agent disconnect.
- `NOT RUN` — real ngrok SDK integration. The focused opt-in harness reported
  `NOT RUN: explicit real-ngrok integration opt-in was not provided`; no credential value was read
  or emitted, and this result is not treated as endpoint evidence.
- `FAIL` — the mandatory real-loopback source-binding acceptance on the target macOS host. Command:
  `env GOCACHE=/private/tmp/fallout-go-build-cache go test internal/tunnel/ngrok.go internal/tunnel/upstream_dialer.go internal/tunnel/model.go internal/tunnel/secret.go internal/tunnel/service.go internal/tunnel/config.go internal/tunnel/process.go internal/tunnel/process_darwin.go internal/tunnel/ngrok_test.go internal/tunnel/upstream_dialer_test.go`.
  Exact non-secret diagnostic:

  ```text
  TestSourceBoundUpstreamDialerUsesDedicatedTCP4SourceAndSolePlayerTarget:
  dial tcp4 127.0.0.2:0->127.0.0.1:3690: bind: can't assign requested address
  ```

  `ifconfig lo0` showed only `127.0.0.1` for IPv4. No ordinary/default-dialer fallback hidden behind
  the old player policy, Host-trust, second player server, privileged interface mutation, or
  alternate-source fallback was introduced. BUG-001 now supersedes that contract with ngrok
  endpoint Basic Auth; T037 remains unchecked until the corrective adapter/tests pass.

## Preconditions

- macOS 13+ on Apple Silicon (`arm64`)
- repository-pinned Go, Node, protobuf, Wails v3, and tool modules
- no ngrok executable required in `PATH` and no provider binary inside the `.app`
- optional personal ngrok token and optional owned reserved domain for real-network journeys
- optional Developer ID/notary credentials only for their conditional release gates

Never place a real token or player password in this file, shell history, process arguments, URLs,
test fixtures, screenshots, logs, or evidence output. A test harness may read dedicated credentials
outside the application and submit them through the same trusted UI/private mutation path; redact
the harness output.

## Migration ownership and evidence stages

Root composition and `internal/tunnel` own the temporary CLI/SDK coexistence. It expires in feature
007 immediately after the embedded-only pre-removal package, security, lifecycle, and rollback gates
pass. No accepted candidate may expose a runtime switch between CLI and SDK paths.

Evidence is collected in two deterministic stages:

1. **Pre-removal candidate**: composition and packaged UX use only the embedded path while legacy
   source remains available solely for rollback. Run the complete package/reproducibility/security
   matrix, record the immutable user-approved checkpoint commit plus package digest, and perform a
   clean rollback/rebuild through separate canonical `go run ./cmd/build build` and
   `go run ./cmd/build package` commands.
2. **Final post-removal candidate**: after deleting CLI/process/config/documentation artifacts, the
   legacy scan and every deterministic gate run again against a new candidate digest. Conditional
   real-service and release evidence runs only after this stage passes.

The rollback reference is recovery evidence, not active architecture. Restoring it means reverting
the complete feature-007 cutover and rebuilding a coherent candidate; it must never produce or ship
a dual-runtime switch.

### Pre-removal checkpoint and rollback drill

T074 requires a user-approved checkpoint commit created only after T073 passes. The implementer
MUST NOT create, amend, or infer that commit without user approval. Record its full 40-hex SHA and
the pre-removal package digest. A mutable branch/tag name, working tree, index, stash, or archive is
not an acceptable rollback reference. If the commit is unavailable, record T074 as `BLOCKED`; do
not start T075.

Using task-specific variables, create a detached temporary worktree from exactly that commit and
run the canonical commands inside it:

```bash
set -e
checkpoint_sha="$(git rev-parse --verify 'HEAD^{commit}')"
test "${#checkpoint_sha}" -eq 40
case "$checkpoint_sha" in *[!0-9a-f]*) exit 1 ;; esac
rollback_worktree="$(mktemp -d /tmp/fallout-terminal-007-rollback.XXXXXX)"
git worktree add --detach "$rollback_worktree" "$checkpoint_sha"
(
  cd "$rollback_worktree"
  test "$(git rev-parse HEAD)" = "$checkpoint_sha"
  test -z "$(git status --porcelain)"
  go run ./cmd/build build
  go run ./cmd/build package
  scripts/verify-macos-app.sh "build/bin/Fallout Terminal.app"
  scripts/hash-macos-app.sh "build/bin/Fallout Terminal.app"
)
git worktree remove "$rollback_worktree"
```

Before the final command, verify the path appears as the detached task-owned worktree in
`git worktree list --porcelain`; if any build or identity check fails, preserve it for inspection
and record `FAIL` rather than deleting an unvalidated path. Compare the detached rebuild's package
digest byte-for-byte with the T073 pre-removal candidate digest; a mismatch is `FAIL` even when both
packages verify independently. Attach the full SHA, both digests, comparison, verification output,
and cleanup result without credentials. T075 is unblocked only by a complete `PASS` record.

## 1. Deterministic quality gates

Run in the canonical build order and preserve separate attributable results:

```bash
gofmt -l .

npm ci --prefix frontend
npm ci --prefix client
npm ci --prefix tests/browser

scripts/tool-modules-check.sh
scripts/proto-check.sh
scripts/proto-drift-test.sh
scripts/proto-breaking.sh
scripts/wails-bindings-check.sh
scripts/wails-v3-cutover-check.sh
scripts/dependency-license-check.sh
scripts/legacy-public-access-check.sh
scripts/secret-leak-check.sh
go vet ./...
go test ./...
go test -race ./...

npm run build --prefix frontend
npm run build --prefix client
npm test --prefix tests/browser

scripts/reproducible-build-check.sh
go run ./cmd/build build
go run ./cmd/build package
scripts/verify-macos-app.sh "build/bin/Fallout Terminal.app"
```

The `gofmt -l .` result MUST be empty. Run this exact sequence after T075–T077 on the final
post-CLI-removal tree; the T074 pre-removal build does not substitute for either direct final build
command.

### 1.1 Canonical development-entrypoint smoke

After the deterministic sequence, start the integrated development runtime from the repository root
with exactly:

```bash
go run ./cmd/build dev
```

Do not start a separate frontend or player server. Confirm the master UI and public-access controls
open, then exercise local-player static delivery, unary operations, and a non-empty `Subscribe`.
With provider credentials and networking absent, public Start must fail redacted and bounded while
local play remains usable. Close the application normally, confirm the owned dev runtime exits, and
record `PASS` or `FAIL`; this mandatory canonical-entrypoint gate is not conditional `NOT RUN`.

Also record `go list -m all`, `go mod graph`, dependency-license inventory, vulnerability review,
pre/post binary and `.app` sizes, generated/binding drift, and a package scan proving no external
ngrok executable, CLI lookup, shared credential, or runtime download is present. `cmd/build` remains
the build graph owner; an optional thin Make alias may invoke these gates but cannot replace their
canonical command or own versions.

### 1.2 Dependency vulnerability review

Use the resolved `go list -m all`/`go mod graph` and the committed `frontend/`, `client/`, and
`tests/browser/` lockfiles as the review inputs. Record the review date, external advisory sources,
applicable advisory identifiers, and a non-secret disposition for every finding. Record `PASS` only
when the sources were checked and every applicable finding has a documented disposition; record
`FAIL` for any applicable finding left without one. If external advisory sources are unavailable,
record `NOT RUN` and do not claim that the candidate has no known vulnerabilities. T082 owns the
final-candidate result; this review adds no second build or dependency-orchestration path.

~~The unconditional security run inspects provider-neutral SDK requests without rendering secrets
and proves: the upstream is exactly `http://127.0.0.1:3690`, one non-empty Basic Auth Traffic Policy
is attached before endpoint publication, no custom source dialer/player Host policy/second server is
active, and policy construction failures publish no URL.~~ **BUG-003**: The unconditional security
run proves the SDK targets only an owned loopback ingress, the ingress alone targets
`http://127.0.0.1:3690`, it begins deny-all, and exact Host/Basic Auth activates before publication
without Traffic Policy, buffering, or a second player server. Activation failures publish no URL.
Direct local/LAN journeys remain unauthenticated. Repeat lifecycle and scoped-secret schedules under
`-race` and in packaged smoke.

## 2. Local-only and offline packaged launch

1. Ensure no provider credential is available to the test application and temporarily disconnect
   external networking.
2. Launch `Fallout Terminal.app` by double click, not through Terminal.
3. Confirm the master UI reaches local readiness and public status is stopped/disabled.
4. Open the local and a reachable LAN player address without Basic Auth.
5. Exercise `Subscribe`, character selection, navigation, hacking, sound discovery/playback, and
   reconnect.
6. Attempt public Start and confirm a bounded redacted unavailable/network failure while local/LAN
   clients continue.
7. Quit normally and repeat with `Cmd+Q`; confirm cleanup is at most five seconds.
8. Relaunch offline and confirm there is no endpoint auto-start, provider download, CLI/PATH lookup,
   stale URL, or terminal requirement.

Expected: local/LAN remains fully usable; ~~all unknown external Hosts are denied~~ **BUG-001** no
public URL is published or retained when the endpoint is unavailable; public failure does not change
game state.

## 3. Saved settings and Keychain non-readback

1. Start from safe defaults and open Public Access settings.
2. Confirm username is exactly `players`, domain is blank, and token/password presence is absent.
3. Enter canary token and password values (manual password at least eight characters) and save.
4. Confirm both inputs are cleared and only presence is shown.
5. Search Application Support, session JSON, player-config JSON, events, runtime status, logs,
   diagnostics, frontend storage/state snapshots, generated outputs, and package resources for both
   canaries; expect zero matches.
6. Quit and relaunch by double click. Confirm domain, username, enabled preference, and presence are
   restored, but neither secret can be revealed or copied.
7. Replace each secret independently; then delete each independently. Confirm presence and Start
   eligibility track actual Keychain state.
8. Remove a Keychain item outside the app and relaunch. Confirm the persisted hint is reconciled to
   actual absence rather than treated as a usable credential.
9. Repeat save/relaunch 100 times in deterministic adapter tests and record zero displayed/read-back
   secrets.

Expected: in packaged production, only the macOS Keychain contains secret values; Application
Support contains valid version-1 non-secret JSON with user-only permissions and atomic replacement.
The separate FR-056 dev/test process override remains transient and is not part of this journey.

## 4. Generated password one-time journey

1. Request Generate Password.
2. Confirm the direct presentation contains a newly generated password, an explicit Copy action,
   and no Reveal control.
3. Copy it once and close the presentation.
4. Confirm the DOM presentation/input is cleared, later snapshots/events show presence only, and no
   second operation can recover it.
5. Run descriptor/adaptor/leak tests with a generated canary and prove it appears only in the direct
   result plus transient presentation.
6. Inject cryptographic randomness failure and Keychain write failure. Confirm no password is
   returned and no partial presence is claimed.

Expected: generated value has at least 128 bits of source entropy, is stored before successful
return, and is not reusable after Copy/dismissal.

## 5. Random HTTPS URL

Requires a real personal token and responsive network. If either is unavailable, record this entire
journey `NOT RUN`—do not substitute the fake endpoint.

1. In the packaged UI, save the personal token and player password, leave domain blank, and Start.
2. Observe disabled → starting → ready within the 30-second terminal bound; record whether it meets
   the 15-second target.
3. Confirm the displayed address is HTTPS and provider-assigned.
4. Before the ready publication point, confirm the UI exposes no reusable public URL.
5. At ready, request `/`, static assets, every unary player procedure, and `Subscribe` with missing,
   wrong, and correct Basic Auth.
6. ~~Confirm missing/wrong credentials return `401` from ngrok and correct credentials proceed.~~
   **BUG-003**: Confirm the application-owned ingress returns `401` for missing/wrong credentials
   and correct credentials proceed for static, unary, and streaming requests.
7. Confirm direct local/LAN access remains outside the ngrok endpoint and receives no Basic Auth
   challenge.
8. Stop from UI, ~~confirm URL is withdrawn before endpoint close~~ **BUG-003** confirm ingress
   admission is denied before URL withdrawal and endpoint/ingress close, then verify the old URL
   serves zero player resources.

Record missing/wrong/correct public Basic Auth and non-empty incremental `Subscribe` separately as
`PASS`, `FAIL`, or `NOT RUN`; neither may be inferred from a provider fake. Deterministic tests prove
only ingress activation/auth/streaming intent, lifecycle ordering, redaction, and local isolation.

For SC-001, repeat this explicitly opted-in real start 20 times under predeclared
responsive-network and valid-account prerequisites. At least 19 attempts must reach `ready` or a
clear terminal error within 15 seconds, and all 20 must finish within 30 seconds. If the
prerequisites are unavailable, record the real sample `NOT RUN`. Record the separate 100-schedule
deterministic timeout/cancellation result as lifecycle evidence only, never as provider-performance
evidence.

## 6. Reserved domain and account failures

Requires a real token; domain success additionally requires a domain reserved by that account.

1. Save an owned reserved domain and Start; confirm the exact normalized domain is returned—never a
   silent random fallback.
2. Stop, save an unowned/occupied domain, and Start.
3. Confirm a redacted ownership/availability error, no URL, ~~public Host deny~~ ~~**BUG-001** no
   owned endpoint left active~~ **BUG-003** deny-all ingress admission plus no owned endpoint/ingress
   left active, and working local/LAN clients.
4. Repeat with invalid and revoked tokens, no network, DNS failure, provider timeout, and provider
   disconnect after ready.
5. Correct each cause and Start again without restarting the application.

Unavailable real account/domain prerequisites are `NOT RUN` individually. A fake may cover the
state/category behavior but is labelled deterministic only.

## 7. Start, stop, and reconfigure races

Use deterministic provider, Keychain, network, and clock fakes first, then repeat supported cases
against a real endpoint when available.

1. Run 100 concurrent/repeated Start, Stop, and settings-change schedules.
2. ~~Pause provider completion at policy-configured endpoint acquisition, URL validation, and event
   publication boundaries.~~ **BUG-003**: Pause at deny-all ingress start, endpoint acquisition,
   URL validation, exact-Host/auth activation, and event publication boundaries.
3. During each pause, ~~probe static plus every Connect path for local, exact prospective, stale,
   and unknown Hosts~~ ~~**BUG-001** inspect policy-before-publication and URL state~~ **BUG-003**
   inspect deny-all/active ingress state and URL publication, prove direct local/LAN remains without
   a challenge, and use the protected fixture for public static/Connect authentication outcomes.
4. Change domain, username, token, and password while starting and ready.
5. Deliver a late success and late failure for each superseded generation.
6. Assert one final state matching the latest valid intent, maximum one endpoint, zero stale URL
   publications, and no old/new overlap.
7. Inject endpoint close failure and retry; ~~assert policy remains deny throughout~~ ~~**BUG-001**
   assert the URL remains withdrawn~~ **BUG-003** assert ingress admission remains deny-all, the URL
   remains withdrawn, and no replacement publishes until prior endpoint/ingress ownership is
   resolved.

Expected: deny-all ingress and protected endpoint acquisition precede exact-Host/auth activation and
URL publication; ingress denial precedes URL withdrawal and endpoint/ingress close; stale completion
can only be closed, never published.

## 8. Streaming, multi-client, and reconnect

1. Start protected access and connect four to seven authenticated browser players.
2. Verify each receives a complete initial snapshot and later non-empty `Subscribe` updates before
   stream completion.
3. Exercise character selection, navigation, hacking, sound, broadcast changes, and mixed clients.
4. Interrupt connectivity and confirm reconnect reaches current authoritative state within five
   seconds under test conditions, without duplicate actions or audio.
5. Observe an initial snapshot and at least one later non-empty update before the real public stream
   completes; longer soak testing is optional for personal-use evidence.
6. Stop/reconfigure while streams are active; old streams/URLs must close and local streams must
   continue.

The deterministic browser fixture proves UI/Connect behavior only. Steps using real public ingress
are credential-gated and must be `NOT RUN` when unavailable.

## 9. Keychain locked, denied, and unavailable

1. Inject each condition with the deterministic `SecretStore` fake for presence, save, delete, and
   scoped use.
2. Confirm status distinguishes unknown from absent, errors are redacted, Start stays fail-closed,
   and no plaintext fallback file is created.
3. Exercise the corresponding OS result on the packaged app where safely reproducible, including a
   user-denied prompt and locked/unavailable Keychain condition.
4. Confirm correcting access restores presence/use without application restart.
5. Run the same tests under the development bundle namespace and packaged production namespace;
   confirm they do not silently share credentials.

OS cases that cannot be safely induced are reported separately as `NOT RUN`; fake cases do not
upgrade them to a real packaged pass.

## 10. Quit, crash, and cleanup budget

1. Quit during disabled, starting, ready, stopping, failed, reconfiguring, endpoint-acquired, and
   protected-endpoint-before-publication states.
2. Trigger normal window close, `Cmd+Q`, repeated shutdown, partial startup, and handled development
   interrupt.
3. In deterministic lifecycle tests, repeat every state 100 times and inject blocked/erroring close,
   late `Done`, and stale completion.
4. For graceful paths, ~~verify URL withdrawal occurs first~~ **BUG-003** verify ingress admission is
   denied first, then URL withdrawal and endpoint/ingress close occur; endpoint, agent, goroutines,
   listeners, scoped secret buffers, player streams, session worker, and desktop resources finish
   within the existing single five-second budget.
5. Force-kill the packaged process after endpoint readiness; verify the previous URL serves zero
   player resources, then relaunch and confirm stopped state with no stale URL.

Expected: repeated shutdown joins existing cleanup rather than extending the deadline. Unexpected
termination cannot guarantee an in-process callback, so proof is external URL unavailability and a
clean stopped next launch.

## 11. Final package and conditional release evidence

The final-candidate package smoke includes double-click UI configuration, Keychain save/presence,
Start, authenticated static/streaming player access, Stop, and Quit without Terminal, env/args,
external CLI, or PATH. With no real token/connectivity this full public package journey is
`NOT RUN`; the unconditional offline/local package journey still runs but is not a substitute.

Developer ID signing, notarization, stapling, DMG, Gatekeeper, paid/reserved-domain features, and
real ngrok reachability each keep independent evidence. Missing external prerequisites always mean
`NOT RUN`, never synthetic `PASS`.

## Evidence matrix

| Surface | Deterministic evidence | Real/package evidence | Required outcome |
|---|---|---|---|
| State/generation/order | unit, race, lifecycle integration | packaged start/stop/reconfigure | no unprotected window or stale publication |
| Endpoint auth | private-ingress integration and protected fixture | real missing/wrong/correct Basic Auth | exact-Host ingress rejects accidental unauthenticated entry without buffering |
| Streaming/reconnect | in-process Connect + browser fixture | focused real incremental stream/multi-client reconnect | incremental, authoritative, ≤5s reconnect condition |
| Secrets | fake store, descriptor/adapter/leak scans | macOS Keychain packaged journey | zero forbidden exposure/readback |
| Failure isolation | provider/network/store/clock fakes | offline, invalid token/domain where available | local/LAN remains usable |
| Cleanup | 100-state schedules, race tests | close, Cmd+Q, kill, relaunch | ≤5s graceful cleanup; stale URL unusable |
| Build/package | drift, module/license, reproducible build | arm64 package/offline smoke | no CLI/runtime download; exact pins |
| Release | deterministic preflight only | Developer ID/notary/provider credentials | honest `PASS`/`FAIL`/`NOT RUN` |

## 12. BUG-003 corrective execution and dev/test override (T087–T095)

This section supersedes the active Traffic Policy/direct-upstream instructions above for BUG-003
work; dated evidence remains historical. Execute T087–T095 in order. The current path is ngrok SDK
endpoint → owned loopback-only deny-all/exact-Host Basic Auth ingress → the sole player service on
port 3690. Real closure requires an initial non-empty `Subscribe` snapshot, a later update, and
reconnect through the packaged endpoint; `NOT RUN` cannot close BUG-003.

For canonical development/test launches only, the application may consume these exact names:

- `FALLOUT_NGROK_AUTHTOKEN`
- `FALLOUT_NGROK_RESERVED_DOMAIN`
- `FALLOUT_PUBLIC_TEST_USERNAME`
- `FALLOUT_PUBLIC_TEST_PASSWORD`

Set credentials outside recorded commands and output. ~~Never print, inspect, echo, snapshot, or
capture their values.~~ **BUG-003 username reconciliation**: Never print, inspect, echo, snapshot,
or capture token/password values. Domain and username may be inspected only in the approved
non-secret master form/snapshot surfaces and must not enter logs or diagnostics. For each non-empty
name, the environment source wins for that development/test process; domain and username prefill the
form, while token/password show only presence and are used transiently after explicit Start. Empty
or unset names fall back independently to persisted settings or Keychain. Loading never saves or
starts public access.

Run the RED matrix in T093 before T094. It must cover every name alone and together, empty/unset
fallback, invalid non-secret validation, explicit-start use, no implicit persistence, no secret Save
mutation, ordinary explicit Save of visible non-secrets, no auto-start, and canary absence from UI
values/events/status for token/password plus absence of every secret canary from logs, diagnostics,
JSON, game data, and artifacts. Domain/username canaries must appear only in the expected secret-free
master form/snapshot surfaces.
Then rebuild through the canonical build graph and prove the packaged production profile ignores all
four names. Configure the T095 packaged real run through its UI/Keychain; it must not depend on a
Terminal environment.

### T087 real-stream baseline — 2026-08-16

- `FAIL` — the available real ngrok endpoint served the player HTML but the generated
  `PlayerService/Subscribe` request did not deliver its initial snapshot; the browser remained on
  `УСТАНОВКА СВЯЗИ...` until public access was disabled and localhost was used.
- No Authorization header, cookie, token, password, request body, raw provider diagnostic, or
  secret-bearing environment value was captured. The pre-correction observation therefore proves
  only static success plus missing first-frame delivery; it does not assign a provider status or
  buffering cause.
- The opt-in Go harness now records only response status/content type, upstream-arrival time,
  response-header time, first-snapshot time, and later-update time for the actual generated
  `Subscribe` request. The browser journey requires the connection overlay to clear within five
  seconds and verifies reconnect. Without explicit opt-in credentials these checks report `NOT RUN`.

### T092 qualification checkpoint — 2026-08-16

- `PASS` — empty `gofmt -l .`, `git diff --check`, focused ingress/tunnel/player/root tests,
  deterministic protected Playwright journeys, local-fallback Playwright journey, and `go vet ./...`.
- `FAIL` — the first broad `go test ./... -count=1` gate stopped qualification. Exact non-secret
  causes: `TestActiveFrontendUsesRuntimeNeutralDesktopFacade` still requires the obsolete exact
  substring `import { Events } from '@wailsio/runtime'` while the active facade imports both
  `Clipboard` and `Events`; the convention gate rejects the new `context.Background()` cleanup in
  `internal/player/public_stream_test.go`; and it rejects direct `t.Fatal` in
  `internal/tunnel/ngrok_test.go`.
- No credential value or secret-bearing diagnostic was captured. T092 remains unchecked; race,
  protobuf/bindings, full frontend/browser, reproducibility, package, scan, and packaged-smoke gates
  were not run after this mandatory failure, and T093–T095 were not started.

### T092 qualification rerun — 2026-08-16

- `PASS` — the obsolete runtime-import assertion now requires the active `Clipboard, Events`
  contract, the public-stream fixture cancels its stream before bounded ingress shutdown without a
  root context, and the endpoint-lifetime assertion uses semantic Testify polling. The focused
  platform/player/tunnel checks pass, including the test-convention gate.
- `PASS` — `gofmt -l .` is empty, `git diff --check` and `go vet ./...` pass, and fresh bounded
  `go test ./... -count=1 -timeout=2m` plus `go test -race ./... -count=1 -timeout=3m` runs pass.
  The only warnings are macOS linker target-version warnings from system objects.
- `PASS` — locked installs for `frontend`, `client`, and `tests/browser`; tool-module, protobuf,
  generated-contract drift, breaking, Wails bindings/v3, dependency-license, legacy-runtime, and
  secret-leak gates; and both production frontend builds pass. The drift gate's synthetic dirty
  generation is rejected as intended before the clean deterministic result passes.
- `PASS` — full Playwright reports 37 passed and one real-provider test skipped because this task
  makes no real-endpoint claim. Its first sandboxed attempt could not bind the loopback fixture;
  the identical suite passed when allowed to bind only its local test ports.
- `PASS` — two canonical package runs are byte-reproducible with zero repository drift. The rebuilt
  `.app` passes signature, architecture, entitlements, offline-resource, dependency/license, and
  no-provider-executable/PATH verification with bundle-manifest SHA-256
  `048e413a94b2662337bc5bae19b699caf3741db030f609c4e347b408efc7cc00`.
- `PASS` — packaged local/offline lifecycle smoke preserves localhost after partial startup,
  releases owned resources on deferred-close cleanup and Cmd+Q-equivalent quit within one second,
  and retains local mode after forced owner loss without stale endpoint reuse. Per the recorded user
  deferral, interactive normal-window close is `NOT RUN`; real stale-public-URL probes are also
  `NOT RUN` without explicit opt-in. No provider request was made and no credential or secret value
  was read, printed, or persisted.

### T093 environment-override RED checkpoint — 2026-08-16

- Added table-driven settings and secret tests for each of the four exact FR-056 names alone and
  together, empty/unset per-field fallback, invalid visible values, presence-only secret use,
  callback clearing, ordinary visible Save, no implicit persistence, and no secret-store mutation.
- Added root-composition coverage for explicit Start only, secret-free snapshot serialization,
  underlying Keychain fallback remaining untouched, and zero environment lookup in the packaged
  profile. Added the browser prefill/no-auto-save/no-auto-start journey and leak/package source
  gates using generated synthetic canaries only.
- `RED` is confirmed: tunnel/root tests fail to compile because the approved environment constants,
  `NewDevelopmentTestPublicAccessOverride`, and `publicAccessStoresForProfile` do not exist yet;
  the package-profile test cannot find `internal/tunnel/test_override.go`; and the leak gate reports
  `development/test public-access override is missing`. The browser projection test already passes
  because it exercises the existing secret-free snapshot surface. No environment value was printed.

### T094 environment-override GREEN checkpoint — 2026-08-16

- `PASS` — one `DevelopmentTestPublicAccessOverride` implements the existing settings and
  `SecretStore` contracts. It queries only the four exact FR-056 names, overlays non-empty
  domain/username during normalized `Load`, reports token/password as presence, and copies them only
  into a cleared `SecretUse` callback for explicit Start. Empty/unset fields independently fall back
  to persisted settings or Keychain.
- `PASS` — adapter `Save`, Replace, and Delete delegate ordinary explicit mutations without ever
  seeding an environment secret. Initialization performs no write or Start; the focused composition
  test proves an explicit Start succeeds through the existing embedded manager/ingress path while
  the underlying secret store remains absent.
- `PASS` — packaged profile composition returns the original settings/Keychain stores before any
  environment lookup. Focused tunnel/root/package-source tests, the browser presence-only prefill
  journey, secret-leak self-test and active scan, and legacy single-runtime scan all pass. No Wails
  method, protobuf field, process argument, external CLI, second server, or second build path was
  added, and no environment value was logged or returned.

### T095 final closure — 2026-08-16 (PASS)

- `PASS` — focused environment/ingress/tunnel/player/root tests after correcting explicit endpoint
  teardown order. The first real run delivered HTTP 200 `application/connect+proto`, its initial
  snapshot in 140 ms, and a later update in 315 ms, but exposed that endpoint cleanup cancelled the
  committed lifetime before SDK close. A regression test now requires SDK close before owned-
  lifetime cancellation; the corrected real rerun delivered its initial snapshot in 147 ms, a later
  update in 247 ms, and clean teardown. Missing and wrong Basic Auth returned `401`; correct auth
  returned the player resource.
- `PASS` — empty `gofmt -l .`, `git diff --check`, all three locked npm installs, tool-module,
  protobuf format/lint/drift/breaking, Wails bindings/v3/cutover, dependency-license, legacy-runtime,
  secret-leak, `go vet ./...`, fresh full unit and race suites, both production frontend builds, and
  full deterministic Playwright. Playwright reported 38 passed and one real-browser test skipped in
  the broad run. The known test-only macOS linker target warnings remained non-blocking.
- `PASS` — two canonical package graphs were byte-reproducible with zero repository drift. Separate
  direct build/package commands and package verification reproduced arm64/macOS-13 bundle-manifest
  SHA-256 `a25d1e9d6a1ac920385671b5eee03b40f1fe86d6da0c37ec5fc04ba159b7cbad` with a valid final
  signature, reviewed resources, native frameworks, and no provider executable/PATH runtime.
- `PASS` — credential-free packaged lifecycle smoke kept localhost usable and met the five-second
  cleanup budget for deferred-close cleanup, Cmd+Q-equivalent quit, partial startup, forced owner
  loss, and stopped relaunch. Interactive normal-window close remains the prior user-deferred
  `NOT RUN`.
- `PASS` — the Finder-semantics packaged production profile restored Keychain presence, ignored all
  four development/test variables, loaded a writable copy of the bundled session, created a player
  configuration and character, started broadcast, activated a terminal, reached a real provider-
  assigned HTTPS URL, then stopped with the stale URL returning `404` while localhost returned
  `200`.
- `PASS` — after the user aligned the packaged Keychain password through the secure UI, status-only
  probes against the real endpoint returned static `401`/`401`/`200` and unary
  `401`/`401`/`200` for missing/wrong/correct Basic Auth. The provider-assigned exact Host reached
  the application, while an unknown Host was rejected with `421`. No credential value was read,
  shown, printed, copied, or persisted outside Keychain.
- `PASS` — the credential-gated real Playwright journey requires HTTP 200, a visible player
  `#screen`, and the first authoritative snapshot within five seconds. Five simultaneous packaged
  clients converged on character selection and the streamed navigation update from folder `2` to
  `запись`, requested sound manifests from the exact public origin, and each reconnected with HTTP
  200 `application/connect+proto`; the run passed in 12.3 seconds. A separate live level-2-terminal
  journey rendered the hack board, submitted a real guess, observed the non-empty streamed hack
  log, and reconnected; it passed in 4.5 seconds.
- `PASS` — Stop removed the public endpoint: the stale URL returned `404` while
  `http://127.0.0.1:3690/` remained `200`. The post-change full deterministic browser suite reports
  38 passed and two explicitly credential-gated real-provider tests skipped. The packaged
  production composition continued to ignore all four development/test variables without reading
  or exposing their values.
