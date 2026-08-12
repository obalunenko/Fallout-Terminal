# Validation: Player Sessions and Shared Terminal Control

Validated on 2026-08-12 from the feature working tree.

## Automated gates

| Gate | Result |
|---|---|
| `gofmt -l .` | PASS — no files reported |
| `go vet ./...` | PASS |
| `GOCACHE=/private/tmp/fallout-go-cache go test ./... -count=1` | PASS — all Go packages, including `internal/playerconfig` |
| `GOCACHE=/private/tmp/fallout-go-cache go test -race ./... -count=1` | PASS — all Go packages after reducing unrelated large-payload serialization in the slow-observer fixture; the production queue behavior was unchanged |
| `npm --prefix frontend run build` | PASS — Vite production bundle generated |
| `npm --prefix tests/browser test` | PASS — 30/30 Playwright journeys |

The browser fixture intentionally returns 404 for optional sound endpoints; tests use the static asset server and those responses are expected. Playwright also reports that `NO_COLOR` is ignored because `FORCE_COLOR` is set; this does not affect results.

## Success-criteria evidence

| Criterion | Evidence |
|---|---|
| SC-001 | `internal/control/service_test.go`: 100-way same-character claim races produce one claimant. |
| SC-002 | `internal/control/service_test.go`: concurrent first assignments establish exactly one controller. |
| SC-003 | `tests/browser/player-sessions-control.spec.mjs` and `internal/player/server_test.go`: reload, reopen, and three-tab recognition journeys. |
| SC-004 | Same files: recognized reconnect restores assignment without selection. |
| SC-005 | Browser-context, cleared-storage, stale-token, and fresh-process tests. |
| SC-006 | `internal/control/service_test.go`: disconnect retains claims until explicit release/transfer. |
| SC-007 | Control and Playwright observer tests cover navigation, guess, pattern, paging, hover, focus, and back with zero canonical mutation. |
| SC-008 | `internal/control/service_test.go`: active, observer, unassigned, disconnected, stale, unknown, and expired authorization outcomes. |
| SC-009 | Multi-tab final-close tests prove zero automatic promotion. |
| SC-010 | Reconnect-before and reconnect-after-reassignment control/server/browser tests. |
| SC-011 | 100 reassignment/action interleavings plus role-only browser convergence. |
| SC-012 | Roster correction tests compare terminal/private puzzle state before and after every correction. |
| SC-013 | `internal/control/service_test.go`: 100 forced action/reassignment interleavings. |
| SC-014 | Real-socket 4–7 client convergence, one-use action, and slow-observer isolation tests. |
| SC-015 | US7 real-socket and browser journeys follow ten terminal switches and a late assignment. |
| SC-016 | US8 live tests deep-compare private checkpoints; socket/browser tests compare restored public puzzles. |
| SC-017 | Preserve, discard, and cancel tests across live, control, server, app, and browser surfaces. |
| SC-018 | US9 control/server tests clear the broadcast epoch while retaining recognized sessions and roster. |
| SC-019 | Second-broadcast reselection and fresh-process unknown-token replacement tests. |
| SC-020 | `internal/domain/model_test.go`: byte-identical version-1 encoding across complete runtime activity. |
| SC-021 | `internal/platform/assets_test.go`, strict protocol decoding, and private Wails bridge tests prove no player `ForceHackSuccess` path. |
| SC-022 | Full Go and browser suites pass, including existing hacking generation, guesses, likeness, attempts, patterns, dud removal, lockout, content, and trusted success regressions. |
| SC-023 | `internal/playerconfig/service_test.go` creates a config, saves three stable ordered roster entries, and reloads them through both the relative session reference and shared-file open path. |
| SC-024 | `internal/domain/model_test.go` proves legacy version-1 sessions omit `playerConfig` and round-trip unchanged; the live master surface presents select/create while roster and broadcast controls remain disabled. |
| SC-025 | Player-config/domain/session/App tests cover cancel, missing/invalid data, unsupported versions, duplicate IDs, association failures, and coordinator write failures without partial installation or publication. BUG-002 regressions additionally distinguish a non-nil empty `[]` roster from nil through service result cloning, coordinator installation, session association, and the first durable roster add. |
| SC-026 | `internal/control/service_test.go` verifies successful add, rename, and unclaimed delete persist complete ordered candidates; injected save failure preserves the prior detached coordination snapshot and emits no effect. |
| SC-027 | `internal/domain/model_test.go` inspects exact session/player-config JSON field sets and rejects runtime identity, presence, claim, controller, broadcast, request, navigation, and puzzle fields. |
| SC-028 | `tests/browser/player-sessions-control.spec.mjs` runs the bundled browser against a real Go coordinator/player server: the active controller selects an ordinary filler, an unused special pattern, and a password candidate; each emits one exact request, mutates canonical state once, converges the observer, and resolves pending authoritatively, while observer clicks emit no shared action. |
| SC-029 | `tests/browser/player-sessions-control.spec.mjs` starts from the visible enabled game-master control, accepts the in-app confirmation, observes one authoritative `endBroadcast` call, applies the returned no-broadcast state, and hides the control. `internal/player/server_test.go` proves the matching coordinator transition publishes one revision of `PLAYER_STATE` plus `TERMINAL_CLEAR` to both active and observer players while retaining logical-session identities and roster entries. The production `.app` click-through additionally confirms that a real connected player immediately leaves the active terminal and returns to the unassigned waiting state. |
| SC-030 | BUG-005 live/control/App/player/asset/browser/native tests prove one private reset of the current failed active puzzle creates one fresh full-attempt generation for all assigned players while preserving broadcast, terminal, session, assignment, controller, roster, other runtime, and durable state. |
| SC-031 | Full Go, race, frontend, 30-case browser, Wails package, and native journeys pass with lockout, unfinished-puzzle preservation, broadcast lifetime, prior bugfix, and player-authority boundaries intact. |

## macOS development and package smoke gate

- `wails dev`: PASS for generation, frontend startup, development compilation, self-signing, native launch, and graceful Ctrl+C shutdown. The master start surface rendered correctly in the native window. The environment presented an unrelated Chrome local-network permission sheet while attempting deeper coordinate-driven interaction, so no claim is made for native automation beyond launch/render; the affected master/player state journeys are covered by the 24 Playwright cases and real-socket Go integration suite above.
- `wails build -clean`: PASS for bindings, frontend compilation, Go compilation, packaging, self-signing, and the Darwin post-build hook. Output: `build/bin/Fallout Terminal.app` for `darwin/arm64`.
- The Wails development build reported its expected private-API warning and is for development testing only. Distribution signing identity, hardened-runtime entitlements, Apple notarization credentials, and App Store approval were not available or attempted; the successful package is locally self-signed.

## BUG-001 development and package rerun

- `wails dev`: PASS for binding generation, Vite startup, development compilation, packaging, and self-signing. A second launcher could not claim the already-running Wails bridge on `127.0.0.1:34115`; that existing instance was left untouched. Read-only browser inspection of the live master verified visible select/create player-config actions, `НЕ ВЫБРАНА` recovery status, disabled roster-add and broadcast-start controls, and no application-script error. The live player address on `127.0.0.1:3690` rendered the authoritative waiting state.
- `wails build`: PASS after the final BUG-001 changes for generated bindings, frontend and Go compilation, Darwin packaging, self-signing, and the post-build hook. Output remains `build/bin/Fallout Terminal.app` for `darwin/arm64`.
- Browser-console noise from the Wails bridge opened directly in Chrome and missing optional sound routes in the static Playwright fixture was inspected and treated as harness-only; neither appeared in the application code path or failed a test.

## BUG-002 empty-roster regression rerun

- Targeted red tests reproduced `roster must be an array` at both the player-config service result and the real App create/associate/install boundary before the fix. After preserving non-nil empty slices at both clone boundaries, the focused player-config, domain, coordinator, and App suites pass.
- `gofmt -l .`, `go vet ./...`, `GOCACHE=/private/tmp/fallout-go-cache go test ./... -count=1`, and `GOCACHE=/private/tmp/fallout-go-cache go test -race ./... -count=1`: PASS. The networked Go suites required normal loopback bind access after the restricted sandbox rejected local listeners.
- `npm --prefix frontend run build`: PASS. `npm --prefix tests/browser test`: PASS — 25/25 journeys, including the BUG-002 master contract that treats player-config metadata, not roster length, as the active-config boundary. Optional sound-route 404s and the `NO_COLOR`/`FORCE_COLOR` warning remain expected fixture noise.
- Real `wails dev` native journey: PASS. The game master created `/private/tmp/.../session.json`, then a new `players.json` containing exactly `"roster": []`. The master activated it without `roster must be an array`, showed no player-config or coordination error, enabled add/broadcast controls despite the empty roster, and persisted the first added character to the same file. The session stored the relative association `"playerConfig": "players.json"`.
- Direct Chrome inspection of the Wails browser-development bridge logged the known `wails/ipc.js` DOM-runtime error and browser-extension asynchronous-channel errors. Bound Go commands, native dialogs, authoritative UI state, and file writes completed successfully, so this noise is treated as development-browser harness-only.
- `/Users/olegbalunenko/go/bin/wails build`: PASS for bindings, dependency installation, frontend and Go compilation, Darwin packaging, self-signing, and the post-build hook. Output: `build/bin/Fallout Terminal.app` for `darwin/arm64`.

## BUG-003 active-controller hacking regression

- The production-composed browser regression first reproduced the report at the client boundary: a real generated puzzle encoded its empty activity log as `null`; clicking a target entered pending state, but `renderHackLog()` threw on `hack.log.map(...)` before the outbound send.
- The correction preserves a non-nil empty hacking log through private/public clone boundaries and defensively treats a null or malformed browser log as empty. The real composed journey now passes for filler, unused-pattern, and password selection with exact request preconditions, one accepted mutation per click, converged active/observer surfaces, authoritative pending completion, and zero observer hacking frames.
- `gofmt -l .`, `GOCACHE=/private/tmp/fallout-go-cache go vet ./...`, `GOCACHE=/private/tmp/fallout-go-cache go test ./... -count=1`, and `GOCACHE=/private/tmp/fallout-go-cache go test -race ./... -count=1`: PASS.
- `npm --prefix frontend run build`: PASS. `npm --prefix tests/browser test`: PASS — 27/27 journeys. The `NO_COLOR`/`FORCE_COLOR` notice remains benign.
- `/Users/olegbalunenko/go/bin/wails dev`: PASS for binding generation, Vite startup, native development compilation, packaging, self-signing, launch, and graceful shutdown. Direct Chrome inspection of the development bridge again emitted the known Wails `ipc.js` DOM-runtime noise; no native file-dialog automation was claimed because macOS assistive access was unavailable.
- `/Users/olegbalunenko/go/bin/wails build`: PASS for bindings, frontend and Go compilation, Darwin packaging, self-signing, and the post-build hook. Output: `build/bin/Fallout Terminal.app` for `darwin/arm64`.

## BUG-004 game-master end-broadcast regression

- The production-shaped master regression first reproduced the failure at the confirmation boundary: `#btnEndBroadcast` was visible and enabled, but the only confirmation was native `window.confirm`; no controllable in-app confirmation existed and `EndBroadcast` was never reached when that native gate was unavailable.
- The correction replaces that gate with an accessible modal `<dialog>`, disables both confirmation actions synchronously to prevent duplicate commands, awaits the authoritative result, requires the returned state to contain no broadcast, and keeps command failures visible through the existing coordination alert.
- Focused regression evidence passes: the master journey invokes `desktopAPI.endBroadcast()` exactly once and returns to the non-live presentation; the App lifecycle test invokes the coordinator once; the real player server publishes the same no-broadcast revision and `TERMINAL_CLEAR` to active and observer clients while retaining sessions, fallback names, and roster entries. Asset-contract coverage prevents reintroduction of the native confirmation dependency.
- `gofmt -l .`, `go vet ./...`, `GOCACHE=/private/tmp/fallout-go-cache go test ./... -count=1`, and `GOCACHE=/private/tmp/fallout-go-cache go test -race ./... -count=1`: PASS.
- `npm --prefix frontend run build`: PASS. `npm --prefix tests/browser test`: PASS — 29/29 journeys, including the new BUG-004 master action and the existing US9 player lifecycle journey. The `NO_COLOR`/`FORCE_COLOR` notice remains benign.
- `/Users/olegbalunenko/go/bin/wails build`: PASS for binding generation, frontend and Go compilation, Darwin packaging, self-signing, and the post-build hook. Output: `build/bin/Fallout Terminal.app` for `darwin/arm64`.
- Production `.app` native click-through: PASS when opened through Finder. The game master loaded `sessions/demo.json`, created and activated `/private/tmp/fallout-bug004-native-players.json`, added `Mara`, started a broadcast, accepted the real Chrome player connection at `http://127.0.0.1:3690/`, assigned the character, and made the selected terminal active. The visible `ЗАВЕРШИТЬ ТРАНСЛЯЦИЮ` control opened the in-app confirmation dialog; confirming it returned the master to `ТРАНСЛЯЦИЯ ЗАВЕРШЕНА`, restored the start control, retained `Mara` and logical session `PLAYER 1`, and immediately changed the connected player from the active terminal to `ОЖИДАНИЕ ТРАНСЛЯЦИИ` with no assignment. This completes the T138/T114 native gate.

## BUG-005 failed-hack retry regression

- Focused red coverage reproduced the missing boundary: live/control/App/player/asset tests did not compile without `ResetFailedHack`, and the production-shaped Playwright journey reached `ВЗЛОМ: ЗАБЛОКИРОВАН` but could not find `#btnResetFailedHack`.
- The correction adds one trusted `ResetFailedHack` command. The coordinator accepts it only for the still-current failed, unsolved active terminal, replaces only that runtime slot from the latest validated authored payload, assigns one revision, and publishes the complete fresh terminal projection through the existing ordered effect sinks. A 32-way concurrent duplicate test accepts exactly one reset and one revision; duplicate, stale-terminal, absent, unfinished, and solved calls retain their prior revision and state.
- Fresh candidate IDs now include a private generation-derived scope, matching the existing generation-bound pattern IDs. Delayed word and pattern actions from the failed generation therefore cannot address the replacement, even when row/column placement happens to repeat. Unknown guesses are rejected without canonical mutation.
- `gofmt -l .`, `GOCACHE=/private/tmp/fallout-go-cache go vet ./...`, `GOCACHE=/private/tmp/fallout-go-cache go test ./... -count=1`, and `GOCACHE=/private/tmp/fallout-go-cache go test -race ./... -count=1`: PASS.
- `npm --prefix frontend run build`: PASS. `npm --prefix tests/browser test`: PASS — 30/30 journeys, including BUG-005 four-attempt exhaustion, visible enabled master retry, exactly one command, active/observer convergence, stable broadcast/terminal/character/role identity, and no observer action authority. The `NO_COLOR`/`FORCE_COLOR` notice remains benign.
- `wails build`: PASS for generated bindings, frontend and Go compilation, Darwin packaging, self-signing, and the post-build hook. Output: `build/bin/Fallout Terminal.app` for `darwin/arm64`.
- Production `.app` native journey: PASS. The packaged master loaded `sessions/demo.json` with the temporary `/private/tmp/fallout-bug005-players.json`, started a clean broadcast, activated the level-1 terminal, and served two real headless Chromium player contexts at `http://127.0.0.1:3690/`. The first became active and the second observer. Four authoritative filler guesses converged both screens on blocked state; the master simultaneously displayed `ВЗЛОМ: ЗАБЛОКИРОВАН` and enabled `ПОВТОРИТЬ ВЗЛОМ`. One native click removed both blocked overlays, delivered a visibly different fresh board to both players, retained active/observer roles and the same broadcast/terminal, and returned the master status to `ВЗЛОМ: осталось попыток 4/4` without restarting the broadcast. Temporary mutation of `sessions/demo.json` was reverted after the journey.

| Criterion | BUG-005 evidence |
|---|---|
| SC-030 | Live/control tests inspect the private generation change, full attempts, empty log, latest level/content, unaffected runtime slots, stable coordinator identity/ownership fields, one revision, duplicate/stale refusal, and old-generation action rejection. App/asset tests keep the exact command private; player-server and browser/native journeys prove active/observer convergence without a player reset surface. |
| SC-031 | The complete Go, race, frontend, 30-case browser, Wails package, and native active/observer journeys pass while existing lockout, unfinished-puzzle preservation, terminal switching, broadcast lifetime, player-config, BUG-001–BUG-004, and `ForceHackSuccess` gates remain green. |
