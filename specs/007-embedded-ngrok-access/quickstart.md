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

The unconditional security run inspects provider-neutral SDK requests without rendering secrets and
proves: the upstream is exactly `http://127.0.0.1:3690`, one non-empty Basic Auth Traffic Policy is
attached before endpoint publication, no custom source dialer/player Host policy/second server is
active, and policy construction failures publish no URL. Direct local/LAN journeys remain
unauthenticated. Repeat lifecycle and scoped-secret schedules under `-race` and in packaged smoke.

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

Expected: only the macOS Keychain contains secret values; Application Support contains valid
version-1 non-secret JSON with user-only permissions and atomic replacement.

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
6. Confirm missing/wrong credentials return `401` from ngrok and correct credentials proceed.
7. Confirm direct local/LAN access remains outside the ngrok endpoint and receives no Basic Auth
   challenge.
8. Stop from UI, confirm URL is withdrawn before endpoint close, and verify the old URL serves zero
   player resources.

Record missing/wrong/correct public Basic Auth and non-empty incremental `Subscribe` separately as
`PASS`, `FAIL`, or `NOT RUN`; neither may be inferred from a provider fake. Deterministic tests prove
only policy construction intent, lifecycle ordering, redaction, and local isolation.

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
3. Confirm a redacted ownership/availability error, no URL, ~~public Host deny~~ **BUG-001** no owned
   endpoint left active, and working local/LAN clients.
4. Repeat with invalid and revoked tokens, no network, DNS failure, provider timeout, and provider
   disconnect after ready.
5. Correct each cause and Start again without restarting the application.

Unavailable real account/domain prerequisites are `NOT RUN` individually. A fake may cover the
state/category behavior but is labelled deterministic only.

## 7. Start, stop, and reconfigure races

Use deterministic provider, Keychain, network, and clock fakes first, then repeat supported cases
against a real endpoint when available.

1. Run 100 concurrent/repeated Start, Stop, and settings-change schedules.
2. Pause provider completion at policy-configured endpoint acquisition, URL validation, and event
   publication boundaries.
3. During each pause, ~~probe static plus every Connect path for local, exact prospective, stale,
   and unknown Hosts~~ **BUG-001** inspect policy-before-publication and URL state, prove direct
   local/LAN remains without a challenge, and use the protected fixture for public static/Connect
   authentication outcomes.
4. Change domain, username, token, and password while starting and ready.
5. Deliver a late success and late failure for each superseded generation.
6. Assert one final state matching the latest valid intent, maximum one endpoint, zero stale URL
   publications, and no old/new overlap.
7. Inject endpoint close failure and retry; ~~assert policy remains deny throughout~~ **BUG-001**
   assert the URL remains withdrawn and no replacement publishes until prior endpoint ownership is
   resolved.

Expected: protected endpoint creation is before URL publication; URL withdrawal is before endpoint
close; stale completion can only be closed, never published.

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
4. For graceful paths, verify URL withdrawal occurs first and endpoint, agent, goroutines,
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
| Endpoint auth | adapter integration and protected fixture | real missing/wrong/correct Basic Auth | endpoint policy rejects accidental unauthenticated entry |
| Streaming/reconnect | in-process Connect + browser fixture | focused real incremental stream/multi-client reconnect | incremental, authoritative, ≤5s reconnect condition |
| Secrets | fake store, descriptor/adapter/leak scans | macOS Keychain packaged journey | zero forbidden exposure/readback |
| Failure isolation | provider/network/store/clock fakes | offline, invalid token/domain where available | local/LAN remains usable |
| Cleanup | 100-state schedules, race tests | close, Cmd+Q, kill, relaunch | ≤5s graceful cleanup; stale URL unusable |
| Build/package | drift, module/license, reproducible build | arm64 package/offline smoke | no CLI/runtime download; exact pins |
| Release | deterministic preflight only | Developer ID/notary/provider credentials | honest `PASS`/`FAIL`/`NOT RUN` |
