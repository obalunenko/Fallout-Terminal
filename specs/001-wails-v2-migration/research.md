# Research: Wails v2 Runtime Migration

## Decision 1: Target the stable Wails v2 line

**Decision**: Pin `github.com/wailsapp/wails/v2` and the CLI to v2.13.0.

**Rationale**: The user explicitly requested Wails v2, the official project identifies v2 as stable, and the current application needs only one desktop window. Version pinning makes build and generated-binding behavior reproducible.

**Alternatives considered**:

- Wails v3 beta: offers first-class multi-window support but introduces a different API and prerelease risk without a current product need.
- Keep Electron: lowest immediate effort but does not achieve the requested Go/Wails migration.

**Sources**: [Wails releases](https://github.com/wailsapp/wails/releases/tag/v2.13.0), [Wails project status](https://github.com/wailsapp/wails#readme)

## Decision 2: Retain one desktop window

**Decision**: Map the existing single `BrowserWindow` to the one Wails v2 application window.

**Rationale**: The repository creates one game-master window and players use external browsers. Wails v2's single-window limit therefore causes no parity loss.

**Alternatives considered**:

- Create native player windows: conflicts with the multi-device browser workflow.
- Adopt v3 for hypothetical future windows: adds migration scope without a current requirement.

**Source**: [Wails v2-to-v3 window comparison](https://v3.wails.io/migration/v2-to-v3/#windows)

## Decision 3: Use a dedicated Go player server

**Decision**: Run a separate `net/http.Server` for player HTTP/WebSocket traffic and use Wails AssetServer only for the master frontend.

**Rationale**: Wails v2 AssetServer does not support WebSockets on any target platform. The current player client also needs a LAN-visible origin, whereas the master frontend should remain internal to the desktop webview.

**Alternatives considered**:

- Put player routes in Wails AssetServer: rejected because WebSocket upgrades are unsupported.
- Replace WebSockets with Wails events: Wails events are available only to the desktop frontend and cannot synchronize remote browsers.
- Run a separate executable: unnecessary deployment and lifecycle complexity for a small embedded service.

**Source**: [Wails AssetServer feature matrix](https://wails.io/docs/reference/options/#assetserver)

## Decision 4: Use coder/websocket with per-client writer queues

**Decision**: Pin `github.com/coder/websocket` v1.8.15 and give every registered player one reader loop, one bounded send queue, and one writer loop.

**Rationale**: The library is context-aware, has concurrent-write support, origin checks, JSON helpers, close handshakes, and a stable v1 API. A single writer per connection prevents message interleaving and allows slow clients to be disconnected without blocking canonical state. The initial supported and tested scale is four to seven simultaneous player connections.

**Alternatives considered**:

- `gorilla/websocket`: mature, but coder/websocket has a more context-oriented API and simpler graceful-shutdown integration for this design.
- Standard library only: Go has no standard WebSocket server implementation.
- An event-driven high-performance server: unjustified for the 4–7-client operating envelope.

**Sources**: [coder/websocket repository](https://github.com/coder/websocket), [package documentation](https://pkg.go.dev/github.com/coder/websocket@v1.8.15)

## Decision 5: Preserve the browser frontends

**Decision**: Move the master HTML/CSS/JavaScript into a vanilla-JavaScript Wails/Vite frontend and keep `client/` substantially unchanged. Add `desktop-api.js` as an adapter exposing the existing `window.electronAPI` shape through generated Wails bindings and runtime events.

**Rationale**: The existing UIs are browser-native and contain most presentation behavior. Preserving them minimizes visual, keyboard, audio, and authoring regressions while generated bindings avoid dependence on Wails' untyped global bridge.

**Alternatives considered**:

- Rewrite in React/Vue/Svelte: no feature value and a much larger regression surface.
- Call `window.go` directly throughout `master.js`: couples presentation code to generated names and makes future bridge changes invasive.
- Keep the Electron preload naming without an adapter: generated bindings and Wails events do not natively match that object.

**Sources**: [Wails frontend interoperability](https://wails.io/docs/introduction/#go--javascript-interoperability), [generated-binding guidance](https://wails.io/docs/guides/obfuscated/)

## Decision 6: Use bound methods for commands and events for notifications

**Decision**: Use bound Go methods for request/response operations and Wails runtime events for server-info, client-count, and hack-state notifications. Add `GetRuntimeStatus` to eliminate missed startup events.

**Rationale**: This matches the current invoke/send/listen split. The initial snapshot method makes frontend registration order deterministic, while `OnDomReady` emits the latest status for parity with Electron's `did-finish-load` behavior.

**Alternatives considered**:

- Poll all state: unnecessary latency and repeated work.
- Events for session commands: makes error/cancellation correlation harder.
- Bound callbacks: unsupported and harder to manage across reloads.

**Sources**: [Wails binding methods](https://wails.io/docs/guides/application-development/#binding-methods), [Wails events](https://wails.io/docs/reference/runtime/events/)

## Decision 7: Introduce typed validation and compatibility-preserving JSON

**Decision**: Define typed version-1 models with recursive tagged nodes, complete validation, and custom JSON handling that preserves compatible unknown fields. Reject unsupported versions and malformed known fields without changing the active session path.

**Rationale**: Go requires an explicit model and provides an opportunity to close the documented validation gap without changing valid data. Preserving unknown fields prevents a round trip from silently deleting forward-compatible user data.

**Alternatives considered**:

- Use unrestricted `map[string]any`: closest to JavaScript but weakens boundary validation and generated models.
- Drop unknown fields: simpler, but creates avoidable data-loss risk.
- Introduce session version 2: no migration-driven data requirement.

## Decision 8: Serialize and atomically replace session saves

**Decision**: Queue saves by revision in one session service and write a same-directory temporary file followed by flush, close, and atomic replace. The completion event/result carries the saved revision.

**Rationale**: Current fire-and-forget autosaves can complete out of order. One writer preserves the latest revision, and same-directory replacement reduces partial-file risk while retaining the user's selected path.

**Alternatives considered**:

- Preserve synchronous direct overwrite: parity with a known data-safety gap is not acceptable for a runtime rewrite.
- Debounce only in JavaScript: does not protect against concurrent privileged calls or crashes during overwrite.
- Add a database: incompatible with portable user-owned JSON sessions.

## Decision 9: Synchronize canonical live state in Go

**Decision**: Protect `LiveService` with a mutex, create immutable public snapshots under the lock, and broadcast snapshots after releasing the lock. Client registry operations have their own synchronization; no network write occurs while canonical state is locked.

**Rationale**: Go HTTP and WebSocket handlers are concurrent. This pattern prevents races and deadlocks without introducing an actor framework, and it is directly testable with `go test -race`.

**Alternatives considered**:

- One global mutex including socket writes: slow clients could block gameplay.
- One unbuffered actor loop for every operation: viable but more ceremony for the current state volume.
- Unsynchronized state: fails under concurrent browser actions.

## Decision 10: Port ngrok as an owned process service

**Decision**: Use `os/exec` behind a `ProcessRunner` interface, preserve credential validation and private temporary policy files, parse bounded output, wait for shutdown, and escalate termination after a timeout. On macOS, place the child in an owned process group and terminate that owned group without relying on Windows console-window behavior.

**Rationale**: It preserves the current external binary workflow while making startup, timeout, cleanup, and termination deterministic and unit-testable.

**Alternatives considered**:

- ngrok SDK dependency: changes authentication/configuration behavior and adds vendor coupling.
- Shell invocation: increases quoting and credential-exposure risk.
- Leave child termination best-effort only: retains a documented public-access risk.

**Source**: [Go os/exec documentation](https://pkg.go.dev/os/exec)

## Decision 11: Package macOS Apple Silicon first

**Decision**: Build an Apple Silicon `.app` first. For distribution, enable the hardened runtime, sign nested code and the application with a Developer ID Application identity, submit a DMG with `xcrun notarytool`, staple the accepted ticket, and validate it through Gatekeeper. Windows/WebView2 and Intel/universal macOS packaging are deferred from the initial release.

**Rationale**: The user explicitly prioritized macOS. Wails uses the system WebKit runtime on macOS, so the initial packaging risks are application-bundle composition, architecture, signing, hardened runtime, notarization, stapling, and Gatekeeper—not WebView2 installation. Apple requires Developer ID signing and notarization for trusted distribution outside the Mac App Store.

**Alternatives considered**:

- Windows-first packaging: explicitly deprioritized by the user.
- Universal binary in the first release: broader reach, but adds Intel validation and signing scope before Apple Silicon parity is established.
- Unsigned release distribution: useful only for local development and does not provide an acceptable Gatekeeper path for users.

**Sources**: [Apple: Notarizing macOS software before distribution](https://developer.apple.com/documentation/security/notarizing-macos-software-before-distribution), [Apple: Distributing your app for beta testing and releases](https://developer.apple.com/documentation/xcode/distributing-your-app-for-beta-testing-and-releases)

## Decision 12: Use Go's standard quality toolchain first

**Decision**: Add colocated table-driven and integration tests using `testing`/`httptest`; require `go test -race ./...`, `go vet ./...`, and `gofmt`. Keep browser/WebKit visual, audio, native-dialog, and signed-package validation manual for initial parity.

**Rationale**: The risk is concentrated in new Go domain, persistence, concurrency, protocol, and process boundaries. Standard tooling adds no test dependency and the race detector directly addresses the new concurrency model.

**Alternatives considered**:

- Add a JavaScript test framework immediately: useful later, but it does not validate Go ownership/concurrency and enlarges initial setup.
- Browser automation in the first slice: valuable follow-up, but native WebKit window and dialog automation is substantially more complex than the current acceptance need.
- Manual-only verification: inadequate for a full runtime replacement.

## Decision 13: Use staged cutover with unchanged public data contracts

**Decision**: Keep the Electron application runnable while Go packages and the Wails candidate are built. Remove legacy runtime files only after all automated, browser, public-access, and packaged acceptance gates pass.

**Rationale**: Session files need no transformation, so rollback is simply using the previous executable. Parallel availability reduces migration risk without maintaining two formats.

**Alternatives considered**:

- In-place rewrite: removes the behavioral oracle and rollback path too early.
- Maintain both runtimes indefinitely: duplicates security and release work after parity is established.

## Decision 14: Use hybrid macOS session storage

**Decision**: Keep user-owned session files separate from app-managed data. New/save dialogs default to `~/Documents/Fallout Terminal/Sessions/` and create that directory only after user confirmation. The sample session stays read-only in the `.app` bundle and is copied only on explicit user action. Internal metadata uses `~/Library/Application Support/com.vaulttec.fallout-terminal/`. Autosave always targets the explicitly selected session path and never relocates it.

**Rationale**: macOS application bundles should be treated as read-only, Documents is appropriate for user-visible documents, and Application Support is appropriate for app-managed files. This preserves portable JSON ownership while avoiding executable-adjacent writes.

**Alternatives considered**:

- Store sessions beside the executable: unreliable for signed/read-only `.app` bundles and unclear to users.
- Store all sessions in Application Support: hides user-owned documents and makes backup/sharing less discoverable.
- Automatically copy the demo on first launch: writes without a user decision and obscures whether the sample or a user file is active.

**Sources**: [Apple Application Support directory](https://developer.apple.com/documentation/foundation/url/applicationsupportdirectory), [Apple Documents directory](https://developer.apple.com/documentation/foundation/url/documentsdirectory), [Apple File System Programming Guide](https://developer.apple.com/library/archive/documentation/FileManagement/Conceptual/FileSystemProgrammingGuide/FileSystemOverview/FileSystemOverview.html)

## Decision 15: Make Wails the single startup orchestrator

**Decision**: Use `wails dev` from the repository root as the only development startup command after prerequisites are installed. Configure `wails.json` with `frontend:dir`, `frontend:install`/`frontend:dev:install`, `frontend:build`, `frontend:dev:watcher`, and `frontend:dev:serverUrl` so the Wails CLI installs or refreshes frontend dependencies when required, starts Vite, builds and launches the Go/Wails process, and generates bindings without a user starting another command. The packaged `.app` invokes the same Go application lifecycle directly and embeds both master and player assets.

**Rationale**: The current Electron baseline already uses one composition-root command (`npm start`) to acquire the player listener before opening the master window. Wails v2 project configuration is specifically designed to run frontend install, build, and watcher commands from `wails dev`; using that native boundary satisfies the one-command requirement without adding another wrapper or retaining Node as the final application runtime. The Go composition root then owns player-server and optional-tunnel startup, readiness, failure unwinding, and shutdown in both development and packaged modes.

**Alternatives considered**:

- Add `make dev` or a custom shell wrapper: adds a new repository tool and a second orchestration layer when the Wails CLI already owns the required lifecycle.
- Keep a root npm `start` wrapper after cutover: makes Node/npm part of the operational entry path and conflicts with removing Node as the application runtime.
- Start Vite, the Go player server, and Wails in separate terminals: directly violates the clarified startup contract and makes partial-start cleanup ambiguous.
- Use different lifecycle composition for development and packaging: increases parity risk and makes startup/shutdown acceptance harder to test.

**Sources**: [Wails project configuration](https://wails.io/docs/reference/project-config), [Wails application development](https://wails.io/docs/guides/application-development/), [Wails manual build process](https://wails.io/docs/v2.12.0/guides/manual-builds/)
