# Quickstart: Validate the Wails v2 Migration

This guide becomes runnable as the migration tasks land. Until cutover, keep the Electron baseline available for comparison.

## Prerequisites

- Go 1.26.x.
- Node.js 20+ and npm for the Vite frontend build.
- Wails CLI pinned to v2.13.0.
- ngrok executable only for public-access scenarios.
- macOS 13+ on Apple Silicon with Xcode Command Line Tools.
- A Developer ID Application identity and `notarytool` Keychain profile for release-only signing/notarization checks.
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

## 9. Production and macOS packaging

Build the Apple Silicon application candidate:

```bash
wails doctor
wails build -clean -platform darwin/arm64
```

Validate `build/bin/Fallout Terminal.app` before signing:

- Launch without Go, Node, npm, or Wails on the validation host.
- Confirm master/player JS, CSS, font, every sound category, and the bundled demo are present.
- Confirm no writes occur inside or beside the `.app` bundle.
- Repeat the session, 4–7-browser reconnect, port-conflict, and shutdown scenarios.

Create the release DMG through the repository build script once T056 lands:

```bash
scripts/build-macos.sh
```

The script must use credentials by reference, never print secrets, and perform
Developer ID signing, hardened-runtime verification, DMG creation, submission
through a preconfigured `notarytool` Keychain profile, ticket stapling, and
Gatekeeper assessment. Record the exact artifact SHA-256, architecture,
`codesign --verify`, `spctl --assess`, notarization, and stapling results below
without recording credentials.

Development environments without release credentials may stop after the
unsigned `.app` smoke. They must record signing/notarization as unavailable and
must not present the artifact as a distributable release.

## 10. Cutover gate

Do not delete Electron files or publish the Wails candidate unless:

- All `go test -race ./...` checks pass.
- Every P1 acceptance scenario passes.
- Version-1 round trips show no semantic data loss.
- Invalid public configuration starts zero tunnels.
- The Apple Silicon `.app` passes P1 checks and the release DMG passes signing, hardened-runtime, notarization, stapling, and Gatekeeper checks.
- Shutdown leaves zero owned listeners, child processes, or credential directories.
- The README accurately documents the new commands and rollback release.

After legacy removal, rerun Sections 2–9 from a clean checkout.
