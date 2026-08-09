# Quickstart: Validate the Wails v2 Migration

This guide becomes runnable as the migration tasks land. Until cutover, keep the Electron baseline available for comparison.

## Prerequisites

- Go 1.26.x.
- Node.js 20+ and npm for the Vite frontend build.
- Wails CLI pinned to v2.13.0.
- ngrok executable only for public-access scenarios.
- macOS 13+ on Apple Silicon with Xcode Command Line Tools.
- No Apple Developer membership is required for the active personal-use profile. A Developer ID Application identity and `notarytool` Keychain profile are required only when explicitly validating the optional public-release profile.
- Between four and seven current browser profiles or devices for convergence/reconnect checks.

Install the pinned CLI:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0
wails doctor
```

Do not remove or globally overwrite user session files during validation. Use a temporary test directory and copies of `sessions/demo.json`.

## 1. Baseline before migration cutover

```bash
npm install
npm start
```

Record the master layout, player URL, demo session contents, four-browser navigation, one hacking puzzle, reconnect behavior, optional audio, and shutdown. This is the behavioral oracle until the final cutover.

Baseline evidence recorded on 2026-08-09:

- `npm start` launched the Electron application and one process-owned listener on `0.0.0.0:3690`.
- `GET /` returned the bundled player page and `GET /api/sounds/ambient` returned `obj_computerzax_hum_lp.wav`.
- A root WebSocket upgrade returned HTTP 101, confirming that the same startup command exposed both HTTP and WebSocket player surfaces.
- Interrupting the application removed the port-3690 listener.
- Master authoring, multi-browser convergence, visual/audio parity, and native-dialog behavior remain manual baseline checks and are not claimed by this command-line smoke.

## 2. Automated Go gates

```bash
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
```

Expected:

- `gofmt -l .` prints no Go source paths.
- Vet and all tests exit successfully.
- Race-enabled live/player tests report no data races.
- Tests cover version-1 fixtures, navigation, hacking projections, multi-client protocol behavior, ordered saves, process failures, and shutdown.

T068 clean-source gate recorded on 2026-08-09 after Electron removal:

- A fresh temporary source tree was assembled from repository `HEAD`, the
  complete migration patch, and the source placeholder
  `frontend/dist/.keep`; ignored dependency directories and build outputs were
  absent.
- `gofmt -l .` passed with no output.
- `go vet ./...` passed with no diagnostics.
- `go test ./...` passed for the root package and every internal package.
- `go test -race ./...` passed for the root package and every internal package
  with no reported races. `internal/testutil` correctly reported no test files.

## 3. Develop the Wails candidate

After the prerequisites above are installed, startup is exactly one command from
the repository root:

```bash
wails dev
```

Do not run a separate frontend install, Vite watcher, or player-server command.
The checked-in `wails.json` must make Wails install or refresh frontend
dependencies when needed, start the frontend watcher, generate bindings, launch
the Go application, and acquire the embedded player listener.

Expected within five seconds on the reference host: one game-master window opens,
a player address appears, and the UI reports an actionable error instead if port
3690 cannot be acquired. Confirm the player address serves HTTP and WebSocket
traffic before marking startup ready.

Wails-candidate evidence recorded on 2026-08-09:

- After installing the pinned CLI prerequisite, one repository-root `wails dev`
  invocation owned `go mod tidy`, binding generation, `npm ci`, the Vite watcher,
  Go compilation, native application launch, and the player listener. No second
  frontend or player-server command was used.
- The first prepared launch spent about 50 seconds in Wails-owned dependency and
  build preparation; a cached relaunch spent about 26 seconds rebuilding and
  packaging. Once the native process launched, runtime readiness and the player
  listener were available in under one second. The five-second runtime target is
  met; clean development build wall-clock time is recorded separately and is not
  represented as application startup time.
- The native process owned `*:3690`. `GET /` returned the player application with
  HTTP 200, and a root WebSocket handshake returned HTTP 101.
- Renderer inspection showed the master start screen with Open/New actions, the
  published player URL `http://192.168.0.195:3690`, and client count `1` after a
  player renderer connected. The player showed `ОЖИДАНИЕ ТРАНСЛЯЦИИ_`, and its
  console had no errors.
- Session cancellation, invalid-open retention, explicit demo copying, atomic
  replacement, unknown-field preservation, and 20-revision ordering were
  exercised by `go test -race ./internal/session`; the migrated master bundle
  passed `npm run build --prefix frontend`.
- Interrupting the sole Wails command removed the player listener, Wails dev
  server, and Vite watcher. A diagnostic Chrome connection to Wails' browser-dev
  endpoint logged one internal `runtime:ready`/IPC warning; it did not prevent
  RuntimeStatus, client-count events, native launch, HTTP, or WebSocket readiness.
  Native-dialog visual acceptance remains part of the full acceptance pass.

## 4. Game-master and session parity

1. Start without an active session and confirm no file is written beside the `.app`.
2. Choose the bundled demo, cancel its copy dialog, and confirm no file or active path is created.
3. Choose the bundled demo again, save a writable copy under a temporary test directory, and confirm the bundle copy remains unchanged.
4. Cancel New and Open dialogs; confirm no error or active-path change.
5. Create a session and confirm the dialog initially suggests `~/Documents/Fallout Terminal/Sessions/` without creating it before confirmation.
6. Create, rename, select, and delete terminals.
7. Create a three-level folder hierarchy with command and entry leaves.
8. Edit names and multiline bodies containing `<`, `>`, `&`, quotes, and Cyrillic text.
9. Apply intro text and each hacking level.
10. Perform 20 rapid accepted edits and wait for the newest save revision.
11. Close, reopen, and compare semantic JSON content with the last visible edit.
12. Attempt malformed and unsupported-version fixtures; confirm a useful error and unchanged active session.
13. Confirm app-managed metadata, if any, is under `~/Library/Application Support/com.vaulttec.fallout-terminal/` and contains no session content.

## 5. Local player parity

1. Open the displayed local URL in four browsers and repeat the scale case with five, six, and seven browsers.
2. Verify client count reaches the expected value in every scale case and returns to zero on close.
3. Make a non-hacking terminal live.
4. Alternate at least 25 folder, entry, command, and back actions among browsers.
5. After each action, confirm every browser shows the same canonical screen.
6. Edit live content in the master; confirm players do not change before Publish.
7. Publish removal of the active node; confirm navigation revalidates safely.
8. Disconnect one browser, navigate with the remaining clients, and wait for reconnect; confirm the reconnected browser immediately converges.
9. Stop broadcast; confirm every browser returns to idle.

Player convergence evidence recorded on 2026-08-09:

- `go test -race ./internal/player` opened real WebSocket connections for four,
  five, six, and seven simultaneous clients. At every scale, actions originated
  from alternating clients and all clients received byte-equivalent public
  state through 25 accepted transitions: administrator hacking, a candidate
  guess, and 23 alternating navigation actions.
- The same race-enabled integration suite disconnected a client after shared
  navigation, advanced the remaining state, and verified a new connection
  received the current snapshot. Clear-state connections received no stale
  live message, and shutdown closed all sockets and the listener idempotently.
- Protocol tests rejected malformed, oversized, duplicate-field, unknown-type,
  and invalid-target input and confirmed public envelopes contain neither
  `secretWord` nor `wordsById`.

## 6. Hacking parity and privacy

1. Start each hacking level 1 through 5 and verify word lengths 4 through 8 and four attempts.
2. Inspect `TERMINAL_LIVE` and `HACK_STATE`; confirm `secretWord` and `wordsById` never appear.
3. Submit valid word, filler, stale, malformed, and administrator actions from alternating browsers.
4. Confirm every browser and master status converge after accepted actions.
5. Reconnect during active, solved, and failed puzzles; confirm no puzzle regeneration.
6. Force success from the master and verify navigation unlocks after the existing delay.

## 7. Player presentation and audio degradation

1. Exercise pointer and documented keyboard controls in list and record modes.
2. Verify CRT styling, font, scrolling, reveal animation, hacking board, log, and prompt.
3. Temporarily build a candidate without one optional sound category.
4. Confirm discovery returns `[]` for that category and all visual/input behavior remains usable.
5. Restore assets before packaging.

Player presentation/audio evidence recorded on 2026-08-09:

- The retained player was opened from the real in-process listener started by
  the sole repository-root Wails command. Browser inspection showed the green
  Fixedsys CRT frame, scanline treatment, idle prompt, stylesheet, and both
  external scripts with no player application console errors before the
  deliberate disconnect.
- Stopping the owning Wails command exposed the existing Russian reconnect
  overlay. Restarting the same one command removed the overlay automatically
  and restored the idle authoritative state without reloading the tab.
- Live HTTP checks returned the restrictive CSP and `nosniff` header, discovered
  the ambient and hacking-success WAV files, returned `[]` for an unavailable
  category, and served the ambient asset as `audio/x-wav`. The player audio
  wrapper treats missing assets, unsupported Web Audio, decode failures,
  autoplay rejection, and device errors as non-fatal; static and HTTP tests
  exercise the same degradation boundary.
- BUG-001 regression verification first reproduced the missing authoritative
  `[hidden]` rule with a failing player-asset contract test, then passed after
  the retained player stylesheet made hidden state containers layout-inert.
  A real Chromium render received protocol-valid idle, normal, active-hacking,
  and failed-hacking snapshots from temporary local WebSocket fixtures. In the
  normal state, 80 revealed menu rows produced `scrollHeight=3368` with
  `clientHeight=427`; after scrolling to `scrollTop=2941/2941`, both
  `termIdle` and `hackBlocked` still had computed `display:none`, zero height,
  and neither `ОЖИДАНИЕ ТРАНСЛЯЦИИ` nor `Вход заблокирован` appeared in rendered
  text. Idle showed only the waiting state, active hacking showed only the board
  and hacking header, and failed hacking showed only the blocked state; every
  inactive state container remained hidden with zero layout height.

## 8. Local and public access security

Local default:

```bash
wails dev
```

Confirm no ngrok child starts.

Invalid public mode: start without credentials and confirm local operation remains usable and zero tunnel processes start.

Valid public mode: provide test-only credentials through the documented
environment variables on the same `wails dev` invocation; do not run a separate
tunnel command. Then confirm:

- The public HTTPS URL and retained local context appear.
- Anonymous HTTP and WebSocket attempts receive authentication failure.
- Authenticated HTTP loads assets and authenticated WSS converges with a local client.
- Logs/UI never contain the password.
- Closing the app removes the policy directory and tunnel process within the shutdown timeout.

Public-access evidence recorded on 2026-08-09 (credentials redacted):

- Local-only startup used the sole repository-root Wails command and started no
  ngrok child. Repeating the same command with public mode enabled but all
  credential variables unset kept local HTTP ready with status `200`, started
  zero ngrok processes, and created zero policy directories. Application tests
  also verified the desktop retained the local URL and received a non-secret,
  actionable error.
- With disposable test credentials, the same single Wails invocation started
  exactly one ngrok 3.39.10 child for the in-process port 3690 listener. The
  discovered fixed endpoint used HTTPS, while the credential-bearing policy
  file and its temporary directory had already been removed.
- Anonymous public HTTP and WebSocket upgrade attempts returned `401`.
  Authenticated HTTP returned `200`, and authenticated WSS upgraded with `101`.
  Neither credentials nor the password appeared in application status or
  repository evidence.
- A normal native application Quit invoked reverse-order shutdown: the ngrok
  child/inspector and player listener were gone and no policy directory
  remained. The original BUG-003 reproduction showed that the Wails development
  supervisor could bypass that callback and orphan the isolated ngrok child;
  the guarded run below supersedes that failed observation.
- T078 reran the same authenticated public profile through the sole root
  `wails dev` command after adding the Darwin owner guard. Anonymous HTTP and
  WebSocket requests returned `401`; authenticated HTTP returned `200`; and an
  authenticated same-origin HTTP/1.1 WebSocket request upgraded with `101`.
  The generated traffic-policy directory had already been removed before the
  interrupt. Sending one terminal `Ctrl+C` to the Wails supervisor produced
  `Caught quit` and `Development mode exited`. On the first cleanup poll, port
  3690 had zero listeners, ngrok had zero processes, the owner guardian had zero
  processes, and zero `fallout-terminal-ngrok-*` directories remained. No
  manual process kill was used and all credential values remained redacted.

## 9. Personal-use and optional public macOS packaging

Build the active personal-use Apple Silicon application candidate:

```bash
wails doctor
wails build -clean -platform darwin/arm64
```

Validate `build/bin/Fallout Terminal.app` as the current acceptance candidate:

- Launch without Go, Node, npm, or Wails on the validation host.
- Confirm master/player JS, CSS, font, every sound category, and the bundled demo are present.
- Confirm no writes occur inside or beside the `.app` bundle.
- Repeat the session, 4–7-browser reconnect, port-conflict, and shutdown scenarios.

The Wails build is locally/ad-hoc signed. For personal use on the owner's Mac,
record the architecture, bundle integrity, assets, signature kind, hashes,
single-launch behavior, P1 journeys, storage boundaries, and clean shutdown. If
macOS applies quarantine after the bundle is transferred, use the documented
one-time **System Settings → Privacy & Security → Open Anyway** flow. Do not
disable Gatekeeper globally and do not describe this candidate as a public
Developer ID release.

### Optional public-release profile

The release script accepts only references to an installed Developer ID
Application identity and a `notarytool` Keychain profile. It does not accept an
Apple ID, app-specific password, private key, or API key value. Create the
Keychain profile separately using Apple's credential-storage workflow, then set
the two non-secret references:

```bash
export DEVELOPER_ID_APPLICATION='Developer ID Application: Organization Name (TEAMID)'
export NOTARYTOOL_KEYCHAIN_PROFILE='fallout-terminal-notary'
scripts/build-macos.sh --preflight
```

`--preflight` verifies macOS/arm64, Go 1.26.x, Wails 2.13.0, the entitlements,
the installed signing identity, and Keychain-profile authentication. It creates
no build and makes no notarization submission. Missing release references fail
immediately with an actionable message.

After preflight succeeds, run the complete release path from any directory:

```bash
scripts/build-macos.sh
```

The script resolves the repository root, performs a clean
`darwin/arm64` Wails build with deployment target macOS 13, applies the Developer
ID signature with hardened runtime and `build/darwin/entitlements.plist`, verifies
the signature and architecture, notarizes and staples the app, creates and signs
`build/bin/Fallout-Terminal-arm64.dmg`, then notarizes, staples, and Gatekeeper-
assesses the DMG. Temporary archives and staging directories are trap-owned and
removed on success, failure, or interruption.

The command prints notarization IDs/statuses and final SHA-256 values but never
credential values.

T060 personal-use packaging evidence recorded on 2026-08-09:

- A fresh `wails build -clean -platform darwin/arm64` completed in 2.856 s.
  The executable is a thin Mach-O `arm64` binary. The rendered plist is valid,
  identifies `com.vaulttec.fallout-terminal` version `1.0.0`, and requires
  macOS 13.0.
- `codesign --verify --deep --strict --verbose=2` passed. This development
  candidate is explicitly ad-hoc signed with hardened-runtime flag `runtime`
  and the required JIT, network-client, and network-server entitlements; it has
  no TeamIdentifier and is not a distributable Developer ID build.
- The physical bundle contains `iconfile.icns` and a read-only
  `Contents/Resources/sessions/demo.json` byte-identical to the source demo.
  Master/player JavaScript, CSS, fonts, and all eight sound categories are
  compiled into the arm64 executable and covered by the asset-manifest tests.
- App executable SHA-256:
  `ec013929e5962c79edbdd633bb3986cbe271ac9f5d35fa43c5e6efe63b90509a`.
  Bundled demo SHA-256:
  `c15baf6195a2a07cb7ed7985693c21bc910ae83092656483c94861ba39692e9c`.
  Packaged icon SHA-256:
  `5c59a39c245e7c47c076fbae0eb8b66746bc4a70c767a4cb1114f820756822a4`.
- `security find-identity -v -p codesigning` reported zero valid identities.
  Therefore Developer ID signing, app/DMG notarization, stapling, Gatekeeper
  public-release assessment, and a signed DMG/hash are **N/A for the selected
  personal-use profile**, not passed.
  `scripts/build-macos.sh` was syntax/preflight tested only and made no
  notarization submission. This prevents public publication but does not block
  personal-use acceptance or the migration cutover.

T061 packaged single-launch evidence recorded on 2026-08-09:

- The unsigned `Fallout Terminal.app` executable was launched once with a
  system-only `PATH` (`/usr/bin:/bin:/usr/sbin:/sbin`) while retaining the real
  macOS user environment required for Documents/Application Support paths.
  Go, Node, npm, Vite, and Wails were therefore unavailable to the application.
- That one packaged process opened the native workspace and owned the only
  product listener, `*:3690`. Player HTTP returned `200` and a WebSocket upgrade
  returned `101`. No listeners existed on the Wails/Vite development ports
  `34115` or `5173`, and no frontend or player-server command was invoked.
- A normal application Quit removed the packaged process and port-3690
  listener. A deliberately empty environment failed early because `$HOME` was
  absent; this is not a developer-tool dependency, and installed macOS app
  launches always provide the user environment.

T062 package storage/failure evidence recorded on 2026-08-09:

- Race-enabled session/platform tests passed for the Documents default,
  Application Support metadata boundary, explicit demo-copy cancellation and
  destination, atomic autosave replacement, unknown-field retention, and
  bundled-demo validation. The packaged demo remained byte-identical and
  read-only after application launch; `build/bin` contained only the `.app`,
  with no session file written beside it.
- The first conflict probe bound loopback only and exposed a Darwin-specific
  IPv4/wildcard coexistence case. The player listener now chooses `tcp4` for
  configured IPv4 addresses and has a regression test. With a temporary process
  owning `0.0.0.0:3690`, the rebuilt app stayed open for its actionable startup
  status but acquired no second listener. Normal Quit and holder cleanup left
  port 3690 unused.
- The ad-hoc app was expectedly rejected by `spctl --assess`; this is acceptable
  for the selected personal-use profile and is not claimed as Gatekeeper public
  acceptance. The signed DMG candidate is N/A while Developer ID credentials are
  unavailable, so its storage, conflict, and shutdown repetitions are not
  claimed. The personal-use app's normal startup and conflict paths left no
  listener or child process after Quit.

T063 compatibility comparison recorded on 2026-08-09:

- The retained Electron `sessions/demo.json` and the migrated
  `internal/testutil/testdata/session-v1.json` are semantically identical and
  share SHA-256
  `c15baf6195a2a07cb7ed7985693c21bc910ae83092656483c94861ba39692e9c`.
  Go decode/encode tests preserve every version-1 field, nested Cyrillic and
  multiline content, and unknown extension fields.
- The 12 checked-in protocol goldens cover all eight retained identifiers and
  their client/server directions: `TERMINAL_LIVE`, `TERMINAL_UPDATE`,
  `TERMINAL_CLEAR`, `NAV_STATE`, `HACK_STATE`, `NAV_ACTION`, `HACK_GUESS`, and
  `HACK_ADMIN`. A retained-source comparison found every identifier in the
  Electron server/player oracle, while the Go constructor/decoder tests matched
  every golden semantically and rejected private hacking fields.
- Zero intentional version-1 or WebSocket contract differences were found.
  The Go boundary is intentionally stricter only for malformed, oversized,
  duplicate-field, unknown-type, and cross-origin input; accepted legacy
  messages and public outputs are unchanged.

For the active personal-use profile, environments without Developer ID
credentials stop after the complete local `.app` acceptance suite. Record
signing/notarization/DMG/Gatekeeper public checks as `N/A (personal profile)` and
do not present the artifact as a public distributable release.

## 10. Cutover gate

The current gate authorizes personal-use cutover and Electron removal when:

- All `go test -race ./...` checks pass.
- Every P1 acceptance scenario passes.
- Version-1 round trips show no semantic data loss.
- Invalid public configuration starts zero tunnels.
- The Apple Silicon personal-use `.app` passes bundle integrity, architecture, ad-hoc-signature, asset, single-launch, storage, shutdown, and P1 checks.
- Shutdown leaves zero owned listeners, child processes, or credential directories.
- The README and rollback guide accurately document the personal-use profile, new commands, and retained rollback artifact.

Developer ID signing, hardened-runtime verification, notarization, stapling,
signed DMG, and Gatekeeper-without-bypass checks become mandatory only before a
future public release is published. Missing credentials keep that public gate
closed but do not reopen or invalidate an accepted personal-use cutover.

After legacy removal, rerun Sections 2–9 from a clean checkout.

T069 final post-Electron acceptance recorded on 2026-08-09:

- A fresh `wails build -clean -platform darwin/arm64` completed in 3.009 s
  after the Electron sources, root npm metadata, and dependency tree were
  removed. The rebuilt app is a thin Mach-O `arm64` bundle, passed
  `codesign --verify --deep --strict`, and has executable SHA-256
  `84a99810993a952706c43a099f26be4d18c33390a0d1023096b6b09bc6eb2e29`.
  The packaged demo and icon retained their recorded hashes.
- The packaged executable launched once with only the system `PATH`, owned
  `*:3690`, served player HTTP with status `200`, and had no listeners on the
  Wails/Vite development ports. HTTP retained the restrictive CSP and
  `nosniff`; allowlisted ambient and hack-success discovery returned the
  expected WAV names and an unknown category returned `[]`.
- A real Chrome render showed only `ОЖИДАНИЕ ТРАНСЛЯЦИИ_` in the
  idle state. Both inactive hacking containers had computed `display:none`
  and zero height. Together with the recorded 80-row full-scroll normal,
  active-hacking, and blocked fixture pass above, every BUG-001 state remained
  layout-exclusive at both overflow extremes.
- After packaged shutdown the existing browser displayed its reconnect
  overlay. One repository-root `wails dev` invocation then performed frontend
  install/build/watch, application compilation/launch, and player-listener
  startup; the browser reconnected to the authoritative idle state without a
  reload. Its only logged errors were Chrome-extension message-channel notices,
  not player application errors.
- Interrupting that local-only Wails supervisor produced its handled
  `Ctrl+C detected. Shutting down` path and released ports `3690`, `34115`, and
  `5173`; no ngrok process existed. T078 subsequently repeated the handled
  supervisor interrupt with authenticated public mode active and verified zero
  listener, ngrok/guardian process, or policy-directory residue on the first
  bounded cleanup poll, closing the earlier BUG-003 caveat.
- The complete session, 4–7-client convergence/reconnect, public-authentication,
  storage, port-conflict, audio-degradation, and single-launch evidence remains
  recorded in Sections 4–9 and passed the final source/test review. Developer
  ID signing, notarization, stapling, signed DMG, and Gatekeeper public-release
  checks remain `N/A (personal profile)` and public publication stays closed.

## 10. Canonical personal-use acceptance record

The final post-Electron candidate is the source at the current cutover commit
and the executable produced from it by the clean personal-use command in
Section 9. This is the only artifact identity that acceptance, rollback, and
handoff documentation may label canonical.

Canonical candidate commit: `d95e144b4f8b5629968e8c28c43eb0b0b9ff2d86`

Canonical executable SHA-256: `84a99810993a952706c43a099f26be4d18c33390a0d1023096b6b09bc6eb2e29`

Verified on 2026-08-09 with:

```bash
git rev-parse HEAD
wails build -clean -platform darwin/arm64
shasum -a 256 'build/bin/Fallout Terminal.app/Contents/MacOS/Fallout Terminal'
codesign --verify --deep --strict --verbose=2 'build/bin/Fallout Terminal.app'
go test ./internal/platform -run TestAcceptanceEvidenceUsesOneCanonicalPostElectronCandidate
```

The consistency test deliberately fails if either acceptance document omits
its canonical record, records it more than once, uses malformed identifiers,
or disagrees on the commit or executable digest. The optional public-release
profile remains a separate future gate and must establish a new public artifact
record before publication; it does not replace this personal-use record.

### BUG-003 handled-supervisor regression harness

Run the Darwin real-process lifecycle check before and after the ownership fix:

```bash
go test ./internal/tunnel -run TestDevelopmentSupervisorInterruptCleansRealOwnedResources -count=1
```

The fixture acquires the documented player port, keeps an ngrok-policy-shaped
private directory, and runs as a real supervised child. It then simulates the
development application process disappearing along the handled supervisor path.
The check rejects any result that leaves the child running, port 3690 occupied,
or policy material present after the bounded cleanup interval.
