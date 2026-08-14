# Research: Wails v3 Runtime Migration

**Planning date**: 2026-08-13
**Accepted source baseline**: `main` / `origin/main` at `f1084b3df8b5630862bdf7a0f347b599156653ef`
**Selected migration candidate**: Wails `v3.0.0-beta.8`

## 1. Baseline and Release Status

### Decision

Plan directly from the fetched `main` baseline at `f1084b3df8b5630862bdf7a0f347b599156653ef`. Do not upgrade the Wails v2 baseline before migration. Record that commit as the canonical Wails v2 source rollback before active v2 removal.

Wails v3 is a prerelease on the planning date, while Wails v2 remains the stable release. The project will migrate now to one exact beta set and will accept it as production only after all required gates pass.

### Rationale

- Local `HEAD`, `main`, fetched `origin/main`, and `FETCH_HEAD` all resolved to the same commit during planning.
- The official [Wails FAQ](https://v3.wails.io/faq/) describes v2 as the current stable release and v3 as a prerelease whose CLI can coexist with v2.
- [Wails v3.0.0-beta.8](https://github.com/wailsapp/wails/releases/tag/v3.0.0-beta.8) was the newest GitHub prerelease on 2026-08-13. Its release notes explicitly retain the prerelease warning.
- An opportunistic v2 upgrade would create an unrelated behavioral delta in the rollback authority and obscure whether failures came from v2 baseline churn or v3 migration.

### Rejected alternatives

- **Wait for GA**: rejected because the specification authorizes beta adoption behind evidence-based gates.
- **Upgrade v2 first**: rejected because the accepted baseline exists only as immutable rollback source.
- **Maintain a permanent v2/v3 runtime switch**: rejected by the constitution and cutover requirement.

## 2. Exact Compatible Version Matrix

### Decision

Pin this exact matrix everywhere:

| Component | Exact version / immutable identity | Reproducible configuration |
|---|---|---|
| Wails v3 Go module | `github.com/wailsapp/wails/v3 v3.0.0-beta.8` | Exact `go.mod` requirement and committed `go.sum` |
| Wails v3 CLI | `tool github.com/wailsapp/wails/v3/cmd/wails3` plus `require github.com/wailsapp/wails/v3 v3.0.0-beta.8` in `tools/wails/go.mod` | Same exact isolated tool declaration, module pin, committed tool sum, and `go tool -modfile` invocation in developer docs, CI, and release automation |
| Frontend runtime | `@wailsio/runtime` `3.0.0-beta.8` | Exact package version, with no caret, tilde, tag, or range; committed `frontend/package-lock.json` |
| Official Vite plugin | `@wailsio/runtime/plugins/vite` from `@wailsio/runtime` `3.0.0-beta.8` | Import the plugin subpath from that exact package; it is not a separately versioned package |

The repository-owned Go development tools use one independent module per executable:

| Tool module | Sole direct `tool` declaration | Exact parent-module `require` pin |
|---|---|---|
| `tools/wails` | `github.com/wailsapp/wails/v3/cmd/wails3` | `github.com/wailsapp/wails/v3 v3.0.0-beta.8` |
| `tools/buf` | `github.com/bufbuild/buf/cmd/buf` | `github.com/bufbuild/buf v1.72.0` |
| `tools/protoc-gen-go` | `google.golang.org/protobuf/cmd/protoc-gen-go` | `google.golang.org/protobuf v1.36.11` |
| `tools/protoc-gen-connect-go` | `connectrpc.com/connect/cmd/protoc-gen-connect-go` | `connectrpc.com/connect v1.20.0` |

Each tool module declares an explicit Go version, commits its own `go.sum`, resolves independently, and leaves the root `go.mod` and `go.sum` unchanged. The root application module retains only dependencies required to compile or run the product; a module shared with a tool is pinned independently in each owning module.

The selected GitHub tag is annotated object `474778141796f74c34912db81a5b3d10e4a7d7c2`, targeting commit `81a149919f91f2149d3fe9be5a27472ae7617b8e`. The published npm `3.0.0-beta.8` artifact records git commit `86b39da5354f3c1a35a8d370f55013f334808dd0`, SHA-1 `44fe17929f1667702ce2cd0d1965418728c099a9`, and registry integrity `sha512-c9PZJcOR9z1a6cxtBS2q5cygNajlxDE8Oxv/vvcTFYRbZSoOGBszq4uzlkPTOd0Bot1xkM81jnpaBgKWRpBh2g==`.

### Compatibility finding

Matching labels alone are not the compatibility proof. The immutable beta.8 Go tag embeds `v3/internal/runtime/desktop/@wailsio/runtime/package.json` with the stale label `3.0.0-beta.7`. The official npm `3.0.0-beta.8` commit is exactly one commit after the Go tag; its only change from the tagged runtime tree is the package and lockfile version bump. Runtime source and exported plugin surface are otherwise identical. The npm package exports both `.` and `./plugins/*`, including the official Vite plugin subpath.

Therefore the exact beta.8 Go module, beta.8 CLI, and beta.8 npm artifact form the verified compatible set despite the embedded label drift. This conclusion is specific to the inspected immutable commits and artifact metadata; it must be repeated if any selected version changes.

The selected Wails source declares Go `1.25.0`; this repository's governed Go `1.26` toolchain satisfies it. The repository retains Node.js `20.19+`, subject to successful locked frontend installation and build gates.

### Reproducibility rules

- No source, Go build command, workflow, release script, quickstart, or acceptance command may use Go `@latest`, npm `latest`, a caret, a tilde, or an unbounded range for the selected Wails components.
- Commit `go.mod`, `go.sum`, `frontend/package.json`, and `frontend/package-lock.json` together.
- Pin the CLI in `tools/wails/go.mod` and invoke that owning module consistently from the Go build command, CI, documentation, and binding checks; do not install or select a global executable.
- Run npm with `npm ci` in clean and CI paths. Do not copy the upstream template's floating `latest` dependency or caret-based Vite defaults.
- If implementation changes the selected version, update this file, every pin, both lock systems, CI, release automation, generated bindings, package evidence, and acceptance records as one atomic compatibility decision.

### Evidence and risks

- The official [beta.8 release](https://github.com/wailsapp/wails/releases/tag/v3.0.0-beta.8) fixes development proxy connection churn and ordered per-window event delivery/backpressure, both relevant to this migration.
- The official [v3 release tracker](https://github.com/wailsapp/wails/issues/5844) records the beta.8 Go/frontend label mismatch, remaining release gates, and the intent to restore lockstep publishing.
- The tracker still contained unresolved blockers at planning time. A notable custom-scheme cancellation blocker could leave requests alive after an abort or window close. This project therefore requires packaged offline launch, repeated quit, listener/process cleanup, and soak evidence; beta selection is not itself acceptance.
- Streams introduced in beta.8 are out of scope and will not be adopted opportunistically.

### Rejected alternatives

- **Infer compatibility from equal version labels**: rejected because the release train has demonstrated label drift.
- **Use the tag's embedded beta.7 label for npm**: rejected because the published beta.8 artifact is a metadata-only follow-up to the inspected tag runtime and is the official artifact for the release.
- **Use floating upstream template versions**: rejected because they cannot reproduce an accepted candidate.

## 3. Wails v3 Application, Window, and Lifecycle

### Decision

Replace `wails.Run(options.App)` with the Wails v3 application model described by the official [v2-to-v3 migration guide](https://v3.wails.io/migration/v2-to-v3/) and [lifecycle documentation](https://v3.wails.io/concepts/lifecycle/):

1. Create `application.App` with master assets and application options.
2. Inject a narrow application capability into Wails-owned platform and event adapters.
3. Compose the existing Fallout Terminal core with those adapters.
4. Register an internal lifecycle service and one allowlisted desktop service before `Run`.
5. Create exactly one window with the accepted title, dimensions, minimums, dark background, assets, and macOS last-window behavior.
6. Run the application.

Use constructor injection or a narrow interface; do not introduce package globals. `application.App`, window, dialog, browser, and event APIs remain in root composition or `internal/platform`.

### Lifecycle behavior

- Wails `ServiceStartup` receives an application-lifetime context. A returned error aborts the application and triggers shutdown of already-started services.
- Existing application-owned startup failures that the accepted app represents in runtime status must be recorded, partially unwound, and returned as `nil` from the host lifecycle adapter so the master window can present an actionable state. Only failures that make the Wails host/window itself impossible to run may abort the framework.
- Create a stable application-operation context from the application lifetime. Create bounded children for acquisition. Never retain a timed startup child in the desktop adapter or command path.
- Wails `ServiceShutdown` has no context. It must call core shutdown with a newly created `context.WithTimeout(context.Background(), 5*time.Second)` (or the governed configuration value), not an already canceled application context.
- Preserve core startup and shutdown idempotency, partial acquisition tracking, and reverse cleanup. Mark tunnel acquisition before validating its returned URL so cleanup is retried if URL validation or the first stop attempt fails.
- Verify normal window close, Cmd+Q/application quit, handled development interrupt, partial startup, startup timeout/cancellation, and repeated shutdown.

The beta.8 binding collector explicitly classifies `ServiceStartup`, `ServiceShutdown`, `ServiceName`, and `ServeHTTP` as internal service methods that must not be exposed. That source behavior is useful but not sufficient: deterministic generated-inventory tests remain mandatory.

### Rationale

This mapping preserves application-owned observability while respecting the framework's abort semantics. It also removes the v2 context-lifetime hazard in which a bounded startup child was retained until a later DomReady callback replaced it.

### Rejected alternatives

- **Return every core startup failure from `ServiceStartup`**: rejected because it would turn status-visible failures into unexplained application exits.
- **Register the current composition root as one service**: rejected because it exposes lifecycle/helper methods and couples domain construction to Wails binding analysis.
- **Use globals to break the construction cycle**: rejected because late registration and narrow injected capabilities solve it explicitly.

## 4. Allowlisted Services and Generated Bindings

### Decision

Register a dedicated desktop service whose generated public method set is exactly the accepted 25-operation inventory. Keep `Start` and `Shutdown` on an unbound lifecycle/core object. Preserve `CopyDemo` as a trusted bound operation without adding an authored UI control. Preserve `GetRuntimeStatus` as bootstrap/status access. Register neither the composition root nor raw native managers.

Use one explicit generated directory, `frontend/bindings`, and clean generation. The generated namespace or file layout is an implementation detail consumed only by `frontend/src/desktop-api.js`.

The official [services](https://v3.wails.io/features/bindings/services/) and [method binding](https://v3.wails.io/features/bindings/methods/) documentation establish explicit service registration and generated frontend modules. Binding verification must run two clean generations, compare complete inventories and content, and assert required and forbidden symbols.

### Baseline discrepancy and narrow resolution

The accepted baseline stores `startupError` in the existing runtime-status contract, but `master.js` currently does not render it. The migration must deliberately expose the existing cached status through the runtime-neutral facade so authored master code can show an actionable startup state. This is not a new generated native capability or protobuf field. Internal lifecycle phase remains internal/test-observable; the UI derives starting, ready-local/ready-public, and failed presentation from the unchanged runtime status fields. Any proposal to add a serialized phase field is a separate application-contract change and is not part of this plan.

### Rejected alternatives

- **Bind all exported methods on `App`**: rejected because current generation includes unintended lifecycle methods.
- **Use `window.go` or optional runtime globals**: rejected because production builds could silently fall back to an unverified privileged surface.
- **Generate private protobuf JavaScript into the master bundle**: rejected because explicit Go adapters already produce the governed native object shapes.

## 5. Frontend Runtime, Events, and Readiness

### Decision

Use the exact `@wailsio/runtime` package and its official Vite plugin as documented in the [frontend runtime reference](https://v3.wails.io/reference/frontend-runtime/). Replace v2 `window.go` and `window.runtime.EventsOn` discovery with explicit generated-service imports and runtime event imports inside `desktop-api.js`. The rest of `master.js` continues to consume only `window.desktopAPI`.

Register the exact four typed event names and emit their current native JavaScript object shapes through `application.App.Event.Emit`. In the frontend, `Events.On` supplies an event object, so the facade unwraps `.data` before applying existing normalization. Each registration returns the underlying unsubscribe function.

The readiness protocol is:

1. Load generated service modules and event runtime successfully; absence is a production initialization failure, not snapshot-only degradation.
2. Register all four underlying listeners.
3. Request one cached `GetRuntimeStatus` snapshot.
4. For each field, apply the snapshot only if no newer event has been observed.
5. Mark the master ready only after the bindings, listeners, and initial status decision are usable.
6. Make every returned release idempotent, call the underlying release exactly once, suppress pending snapshot delivery after release, and release all prior-facade subscriptions once on hot replacement.

A Wails window-ready event may gate binding/event usability, but it cannot replace the post-subscription snapshot, which closes the pre-subscription state gap. If used, tests must prove it does not duplicate state effects. Do not recreate v2 DomReady replay as the sole initialization mechanism.

See the official [event API](https://v3.wails.io/reference/events/) and [events guide](https://v3.wails.io/guides/events-reference/).

### Test-only fallback

Tests may inject explicit service/event doubles through a test-only alias or constructor. Production resolution must not contain an optional privileged global fallback, and the production build must fail if generated bindings are missing.

## 6. Dialogs and External Browser Adapter

### Decision

Replace Wails v2 global runtime calls in `internal/platform/desktop.go` with an injected narrow adapter over Wails v3 [dialogs](https://v3.wails.io/reference/dialogs/), [browser APIs](https://v3.wails.io/reference/browser/), and [manager APIs](https://v3.wails.io/concepts/manager-api/), including `app.Browser.OpenURL`.

Preserve:

- open title `Open Fallout Terminal Session`;
- save title `Save Fallout Terminal Session`;
- JSON filters;
- open default directory reduced to the nearest existing ancestor without creating directories merely to display a dialog;
- save default directory and filename, with directory creation allowed by the save dialog;
- alias resolution where supported;
- cancel as empty path with no error;
- adapter and command error handling/redaction; and
- final privileged validation that external URLs are absolute HTTP or HTTPS only.

Use injected fakes to test defaults, cancel, errors, alias flags, creation policy, and URL schemes without a live Wails process. Record any unavoidable beta API/platform difference before accepting it.

### Rejected alternatives

- **Call dialogs or `window.open` from authored frontend code**: rejected because selection and URL opening are privileged named desktop operations.
- **Spread application manager calls into session/domain packages**: rejected because `internal/platform` owns native adaptation.

## 7. Build System and Asset Ownership

### Decision

Use the already-required Go toolchain for a repository-owned, standard-library-only build command. Do not adopt Wails Taskfiles, Make, or translate the v2 `wails.json` shape. Keep the pinned beta.8 CLI only for explicit Wails binding and icon generation, place project-owned orchestration in `internal/buildtool`, and protect ordering, output, resource copy, and final signing with plan/source tests.

The canonical nonrecursive graph is:

1. verify protobuf tool/revision and deterministic generated state once;
2. build `client/` once;
3. generate Wails bindings cleanly once;
4. build `frontend/` once with those bindings;
5. compile/package the Go/Wails host;
6. copy non-embedded resources into the unsigned bundle;
7. perform the final signature and inspection.

`go run ./cmd/build dev` is the sole repository-root development command. Direct documented clean checks for `client/` and `frontend/` remain valid. Avoid Go-command→npm→Go-command recursion, duplicate protobuf generation, and concurrent generation into shared paths. Use one sequential Go plan so converging nodes execute once.

The official beta.8 template defaults to `bin/`; this project will set `BIN_DIR`/equivalent output to `build/bin` and preserve `build/bin/Fallout Terminal.app`. The project-specific path is already consumed by scripts, CI, README, acceptance evidence, and rollback instructions; the default offers no compensating value.

### Generated build assets

Track the chosen Wails build assets deliberately. Record which files remain update-generated and which are project-owned. An update must be reviewed as a source change and rerun the full graph, configuration, resource, signing, and output-path assertions.

### Rejected alternatives

- **Keep flat `wails.json` as the active v3 model**: rejected because it hides or fights the v3 task/configuration contract.
- **Allow both npm scripts and the Go build command to own the same generation**: rejected because it creates races and makes failures unattributable.
- **Move to `bin/`**: rejected because preserving `build/bin` avoids broad consumer churn.

## 8. macOS Packaging, Resources, and Signing

### Decision

Customize the official [macOS packaging tasks](https://v3.wails.io/guides/build/macos/) for the existing product contract:

- product identifier `com.vaulttec.fallout-terminal`;
- display/product/executable name `Fallout Terminal`;
- accepted version, comments, company/copyright metadata;
- arm64-only target and macOS 13 deployment target;
- current icon, Info.plist, development plist, and entitlements;
- accepted single-window macOS behavior;
- output `build/bin/Fallout Terminal.app`.

The v2 post-build hook currently injects `sessions/demo.json` only after Wails packaging and then re-signs. Under v3, copy `sessions/demo.json` and any other non-embedded resource before the final signature. Never mutate a signed bundle. Preserve `applicationResourceRoot` behavior or replace it with a tested equivalent resolving `Contents/Resources` from the packaged executable.

Final bundle inspection covers:

- master and player generated assets;
- generated player client;
- fonts and sounds;
- `Contents/Resources/sessions/demo.json` and permissions;
- executable arm64 architecture;
- plist metadata and macOS 13 minimum;
- entitlements and icon;
- final ad-hoc signature for the required personal-use profile;
- one offline launch, one player listener, and clean quit.

Adapt `scripts/build-macos.sh` to `go run ./cmd/build package` while preserving its preflight, arm64 verification, Developer ID replacement signing, hardened runtime, notarization, stapling, DMG, Gatekeeper, credential redaction, and SHA-256 steps. Public release gates run only with real credentials and evidence; otherwise record `NOT RUN`.

## 9. Binding-Generation Resource Workaround

### Decision

Treat `production_resources_bindings.go` and the `bindings` build-tag bypass as provisionally obsolete, not automatically removable. Generate clean v3 bindings twice and run clean package builds without the workaround. Remove it only if static binding analysis and normal/package builds prove resource validation no longer runs in an asset-incomplete generator process. Otherwise retain a narrowly scoped workaround and document the exact v3 invocation that requires it.

### Rationale

The file exists to accommodate v2 binding generation before final assets and demo data exist. Wails v3 uses static binding analysis, but only observed clean generation and build evidence can justify deletion.

## 10. CI, Documentation, Cutover, and Rollback

### Decision

Update `.github/workflows/wails-macos.yml` to resolve the exact beta.8 CLI through `tools/wails/go.mod`, generate bindings before any standalone master build, preserve all existing Go/frontend/player/Buf/Playwright/startup gates, add deterministic binding and forbidden-surface scans, package arm64 with v3, inspect the final bundle, and upload the established artifact.

Replace v2-specific active assertions and instructions without rewriting completed historical specifications. Mark the old active Wails v2 migration guide as historical/superseded wherever it would otherwise direct users to current v2 commands. Preserve `docs/wails-migration-rollback.md` as the Electron→Wails v2 historical record. Create `docs/wails-v3-migration-rollback.md` for this migration and record:

- source fallback `f1084b3df8b5630862bdf7a0f347b599156653ef`;
- exact candidate pins and cutover identity;
- rollback triggers and steps;
- safety copies of session-v1 and player-config-v1 files;
- acceptance/rollback drill evidence;
- a v2 artifact digest only if that artifact was actually built and accepted.

Final cutover removes v2 imports/dependency, CLI/configuration/hooks, generated assumptions, obsolete workaround or assertions, and active v2 instructions. Source, dependency, generated-output, built-bundle, CI, script, and documentation scans must find no active v2 runtime or permanent switch. Temporary coexistence is owned by the feature-006 implementer and expires at this cutover gate.

## 11. Application Data and Public Protocols

### Decision

No application data model changes are planned. Domain models, private protobuf contracts, `fallout.terminal.player.v1` public ConnectRPC contracts, session-v1, player-config-v1, revision/replay behavior, and private/public graph isolation remain unchanged.

Wails generated binding metadata, the Go build graph, package manifests, lockfiles, and plist files are build metadata, not a reason to modify protobuf. If implementation discovers a real semantic contract delta, stop that slice, specify it separately, and update compatibility evidence before changing a schema.

### Rejected alternatives

- **Adopt Wails streams for player traffic or bridge events**: rejected as an out-of-scope protocol redesign.
- **Move sessions to protobuf binary or ProtoJSON**: rejected because session-v1 portable JSON is unchanged.
- **Use the migration to redesign ConnectRPC or public projection models**: rejected because feature 005 is the accepted oracle.
