# Verification Quickstart: Embedded ngrok Public Access

This document defines the feature-007 verification journeys. It is not evidence that they have run.
Implementation must record each final-candidate result as `PASS`, `FAIL`, or `NOT RUN` with command,
date, build identity, and relevant non-secret artifact paths. A deterministic fake is never recorded
as real ngrok reachability.

**Bugfix**: 2026-08-15 — ANALYZE-S1/U1 adds unconditional source-bound ingress evidence and an
executable immutable checkpoint/worktree rollback drill.

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
scripts/tool-modules-check.sh
scripts/proto-check.sh
scripts/proto-drift-test.sh
scripts/proto-breaking.sh
scripts/wails-bindings-check.sh
go vet ./...
go test ./...
go test -race ./...
npm --prefix tests/browser ci
npm --prefix tests/browser test
scripts/reproducible-build-check.sh
go run ./cmd/build package
scripts/verify-macos-app.sh "build/bin/Fallout Terminal.app"
```

Also record `go list -m all`, `go mod graph`, dependency-license inventory, vulnerability review,
pre/post binary and `.app` sizes, generated/binding drift, and a package scan proving no external
ngrok executable, CLI lookup, shared credential, or runtime download is present. `cmd/build` remains
the build graph owner; an optional thin Make alias may invoke these gates but cannot replace their
canonical command or own versions.

The unconditional security run also creates real loopback TCP connections without ngrok network
access and proves: the owned `tcp4` dialer binds `127.0.0.2`, rejects every target except
`127.0.0.1:3690`, never falls back when binding fails, and causes the player boundary to classify
local/LAN/unknown Host overrides as public and return `403`. Direct `127.0.0.1`, `::1`, and allowed
private/link-local LAN source plus Host pairs must remain unauthenticated; an external source
claiming a LAN Host must be denied. Repeat the source/Host schedules under `-race` and in the
packaged macOS smoke.

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

Expected: local/LAN remains fully usable; all unknown external Hosts are denied; public failure does
not change game state.

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
4. Before the ready publication point, probe the prospective/unknown external Host and confirm no
   static or Connect request succeeds.
5. At ready, request `/`, static assets, every unary player procedure, and `Subscribe` with missing,
   wrong, and correct Basic Auth.
6. Confirm missing/wrong credentials return `401`, unknown external Host returns `403` even with the
   correct pair, and correct credentials on the exact current Host proceed.
7. Attempt a public-edge Host override to a local/LAN authority; confirm it cannot bypass public
   authentication.
8. Stop from UI, confirm URL is withdrawn before endpoint close, and verify the old URL serves zero
   player resources.

Record the real Host-override and observed-upstream-source results separately as `PASS`, `FAIL`, or
`NOT RUN`. `NOT RUN` is not real endpoint proof, but it does not replace or invalidate the
unconditional source-bound security gate: every SDK upstream connection must use `127.0.0.2`, every
such connection is public before Host classification, and bind/injection failure has no default
dialer fallback. A `FAIL` showing another upstream source or local/LAN bypass blocks cutover.

## 6. Reserved domain and account failures

Requires a real token; domain success additionally requires a domain reserved by that account.

1. Save an owned reserved domain and Start; confirm the exact normalized domain is returned—never a
   silent random fallback.
2. Stop, save an unowned/occupied domain, and Start.
3. Confirm a redacted ownership/availability error, no URL, public Host deny, and working local/LAN
   clients.
4. Repeat with invalid and revoked tokens, no network, DNS failure, provider timeout, and provider
   disconnect after ready.
5. Correct each cause and Start again without restarting the application.

Unavailable real account/domain prerequisites are `NOT RUN` individually. A fake may cover the
state/category behavior but is labelled deterministic only.

## 7. Start, stop, and reconfigure races

Use deterministic provider, Keychain, network, and clock fakes first, then repeat supported cases
against a real endpoint when available.

1. Run 100 concurrent/repeated Start, Stop, and settings-change schedules.
2. Pause provider completion at endpoint acquired, URL returned, policy activation, and event
   publication boundaries.
3. During each pause, probe static plus every Connect path for local, exact prospective, stale, and
   unknown Hosts.
4. Change domain, username, token, and password while starting and ready.
5. Deliver a late success and late failure for each superseded generation.
6. Assert one final state matching the latest valid intent, maximum one endpoint, zero stale URL
   publications, and no old/new overlap.
7. Inject endpoint close failure and retry; assert policy remains deny throughout.

Expected: policy activation is before URL publication; deactivation is before endpoint close;
stale completion can only be closed, never published.

## 8. Streaming, multi-client, and reconnect

1. Start protected access and connect four to seven authenticated browser players.
2. Verify each receives a complete initial snapshot and later non-empty `Subscribe` updates before
   stream completion.
3. Exercise character selection, navigation, hacking, sound, broadcast changes, and mixed clients.
4. Interrupt connectivity and confirm reconnect reaches current authoritative state within five
   seconds under test conditions, without duplicate actions or audio.
5. Sustain the real public stream for at least 30 minutes and record incremental updates throughout.
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
   policy-active-before-publication states.
2. Trigger normal window close, `Cmd+Q`, repeated shutdown, partial startup, and handled development
   interrupt.
3. In deterministic lifecycle tests, repeat every state 100 times and inject blocked/erroring close,
   late `Done`, and stale completion.
4. For graceful paths, verify public policy deny occurs first and endpoint, agent, goroutines,
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
| Host/Auth | HTTP integration and protected fixture | real public Host and Host-override probe | exact Host + correct Basic Auth only |
| Streaming/reconnect | in-process Connect + browser fixture | real 30-minute stream/multi-client reconnect | incremental, authoritative, ≤5s reconnect condition |
| Secrets | fake store, descriptor/adapter/leak scans | macOS Keychain packaged journey | zero forbidden exposure/readback |
| Failure isolation | provider/network/store/clock fakes | offline, invalid token/domain where available | local/LAN remains usable |
| Cleanup | 100-state schedules, race tests | close, Cmd+Q, kill, relaunch | ≤5s graceful cleanup; stale URL unusable |
| Build/package | drift, module/license, reproducible build | arm64 package/offline smoke | no CLI/runtime download; exact pins |
| Release | deterministic preflight only | Developer ID/notary/provider credentials | honest `PASS`/`FAIL`/`NOT RUN` |
