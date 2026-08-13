# Contract: Reproducible Build, macOS Package, Cutover, and Rollback

## Tool and Dependency Pins

| Tool/dependency | Accepted exact value | Enforcement |
|---|---|---|
| Go | repository-governed 1.26 toolchain | developer/CI preflight |
| Node.js | 20.19+ governed baseline | package engines/CI preflight |
| Wails Go module | `github.com/wailsapp/wails/v3 v3.0.0-beta.8` | `go.mod`/`go.sum` and pin scan |
| Wails CLI | `tool github.com/wailsapp/wails/v3/cmd/wails3` plus `require github.com/wailsapp/wails/v3 v3.0.0-beta.8` in `tools/wails/go.mod` | same isolated tool declaration, module pin, committed sum, and `go tool -modfile` invocation in setup, CI, and release automation |
| Wails frontend runtime | `@wailsio/runtime` `3.0.0-beta.8` | exact `frontend/package.json` and lock resolution |
| Wails Vite plugin | `@wailsio/runtime/plugins/vite` from the same `3.0.0-beta.8` package | `frontend/vite.config.js` import and lock verification |
| Vite | retain exact repository-approved version unless compatibility evidence requires an atomic update | exact manifests/locks |
| Buf CLI | `tool github.com/bufbuild/buf/cmd/buf` plus `require github.com/bufbuild/buf v1.72.0` in `tools/buf/go.mod` | isolated committed sum, root-module-drift check, and repository-root `go tool -modfile` invocation |
| protoc-gen-go | `tool google.golang.org/protobuf/cmd/protoc-gen-go` plus `require google.golang.org/protobuf v1.36.11` in `tools/protoc-gen-go/go.mod` | isolated committed sum, deterministic generated provenance, and repository-root invocation through Buf |
| protoc-gen-connect-go | `tool connectrpc.com/connect/cmd/protoc-gen-connect-go` plus `require connectrpc.com/connect v1.20.0` in `tools/protoc-gen-connect-go/go.mod` | isolated committed sum, deterministic generated provenance, and repository-root invocation through Buf |

No reproducible configuration or command uses Go/npm `latest`, `@latest`, caret, tilde, wildcard, or unbounded range for Wails. Lockfile changes are committed with source pins.

Each `tools/<tool>/go.mod` declares exactly one direct tool command and an explicit Go language version. Tool resolution, tidying, generation, CI, Taskfiles, release automation, and documentation must not add a tool directive or tool-only dependency/checksum to the root application module.

## Visible Wails v3 Build Model

The active build model consists of:

```text
Taskfile.yml
build/
├── config.yml
├── Taskfile.yml                  # project build composition if selected by template layout
├── common/Taskfile.yml           # pinned beta.8-derived common tasks
├── darwin/Taskfile.yml           # project-customized macOS package/sign order
├── darwin/Info.plist
├── darwin/Info.dev.plist
├── darwin/entitlements.plist
└── appicon.png
```

Exact paths may follow the beta.8 build-asset generator, but responsibility must remain explicit. Generated/updatable Wails assets record their beta.8 provenance; project-owned custom tasks/config are reviewed and protected by source assertions. Running `go tool -modfile=tools/wails/go.mod wails3 update build-assets` is a code change that must not silently replace output path, deployment target, resource copy, signing, or frontend ordering.

The v2 `wails.json` and `postBuildHooks` become inactive and are removed only at final cutover after the v3 graph is proven.

## Root Command Contract

| Purpose | Repository-root command | Requirement |
|---|---|---|
| full development | `go tool -modfile=tools/wails/go.mod wails3 dev` | sole root dev command; prepares bindings, both frontends, native host, listener, and optional configured tunnel |
| clean bindings | `go tool -modfile=tools/wails/go.mod wails3 generate bindings -clean ./...` (with configured `frontend/bindings` output) | exact 25-method deterministic surface before master build |
| native build | `go tool -modfile=tools/wails/go.mod wails3 build` | runs governed dependency graph, not stale prebuilt assets |
| macOS package | `go tool -modfile=tools/wails/go.mod wails3 package GOOS=darwin GOARCH=arm64` | produces the required app at the preserved path |

Developer docs may provide focused verification commands, but they do not create a second root development workflow.

## Canonical Nonrecursive Build Graph

```text
verify exact pins and tools
        │
        ▼
verify protobuf revision + deterministic generated state (once)
        │
        ├──────────────► build client/ with npm ci + Vite (once)
        │
        ▼
generate clean Wails service bindings into frontend/bindings (once)
        │
        ▼
build frontend/ with npm ci + official Wails Vite plugin (once)
        │
        ▼
compile/package Go host with embedded master and player filesystems
        │
        ▼
copy non-embedded bundle resources
        │
        ▼
final signature → inspection → digest/artifact
```

- Protobuf generation has one owner. Do not invoke a root npm script that recursively invokes Taskfile targets.
- `client/` remains independent of Wails/private bindings and can be clean-built directly.
- Binding generation must finish before the master Vite bundle consumes `frontend/bindings`.
- Shared generation tasks use run-once/dependency semantics; no duplicate concurrent writes.
- Direct locked production checks for both frontend packages remain documented and successful.
- Generated bindings are handled deliberately: if tracked, clean generation must leave no drift; if untracked, CI still compares two clean output trees/content inventories and builds from the second.

## Generated Outputs and Embedding

| Output/resource | Producer | Consumer / final location |
|---|---|---|
| private Wails service bindings | exact beta.8 `go tool -modfile=tools/wails/go.mod wails3 generate bindings` | `frontend/bindings`, then master Vite bundle |
| protobuf Go | existing pinned Buf/protoc flow | application/private/public adapters |
| public player ECMAScript | existing pinned generation | `client/gen`, then player Vite bundle |
| master Vite output | `frontend/` build | embedded master assets configured on the Wails application/window |
| player Vite output | `client/` build | separately embedded player filesystem served only by player listener |
| fonts and sounds | frontend/client Vite/static asset rules | final embedded asset inventories; no CDN/runtime download |
| demo session | source `sessions/demo.json` | `Contents/Resources/sessions/demo.json`, copied before signature, read-only behavior preserved |

`production_resources_bindings.go` is removed only after clean static generation, package builds, and resource tests prove it unnecessary. Otherwise the exact v3 generator need and narrow build-tag behavior are documented.

## macOS Application Contract

| Property | Required value |
|---|---|
| output | `build/bin/Fallout Terminal.app` |
| target | `darwin/arm64`; executable reports arm64 |
| minimum OS | macOS 13.0 |
| identifier | `com.vaulttec.fallout-terminal` |
| display/product/executable name | `Fallout Terminal` |
| app metadata | preserve accepted version, comments/title, company, copyright, and local-network purpose text |
| icon | existing `build/appicon.png` rendered into the bundle by governed task |
| plist | production Info.plist plus accepted metadata/minimum OS |
| entitlements | current project entitlements; exact profile inspected after signing |
| window behavior | single master window; accepted macOS close/quit behavior |
| application resources | `Contents/Resources`, with `applicationResourceRoot` semantics preserved/tested |

The beta.8 template's default `bin/` output and macOS 12 minimum are overridden. Every script, workflow assertion, README instruction, quickstart, and rollback record continues to use `build/bin/Fallout Terminal.app`.

## Resource and Signature Ordering

The package must be complete before its final signature:

1. Build master/player/generated assets.
2. Compile arm64 executable.
3. Assemble `.app`, plist, icon, and executable.
4. Copy `sessions/demo.json` and any other required non-embedded resource.
5. Verify the pre-sign resource inventory and permissions.
6. Apply the final ad-hoc signature for personal use, or final Developer ID/hardened-runtime signature for a selected public release.
7. Perform no bundle mutation afterward.
8. Verify signature, metadata, resources, offline launch, listener count, quit cleanup, then compute the final digest/upload artifact.

This replaces the v2 post-build pattern that injected the demo after Wails packaging and then re-signed.

## Signing Profiles

### Required personal-use profile

- local/ad-hoc signature on the final bundle;
- macOS 13+ Apple Silicon;
- bundle integrity, architecture, entitlements, and offline launch checks;
- one listener during operation and clean owned-resource release on quit.

### Conditional public-release profile

Only with real credentials and a deliberately selected public candidate:

- Developer ID replacement signature;
- hardened runtime and intended entitlements;
- notarization and accepted result;
- stapling;
- DMG construction/signing as governed by the existing release script;
- Gatekeeper check without bypass;
- credential redaction and final SHA-256.

Unavailable public credentials/connectivity yield `NOT RUN`, never `PASS`, and do not block personal-use acceptance unless public distribution was explicitly selected.

## Release Script and CI Contract

Adapt `scripts/build-macos.sh`; do not replace its proven release controls with an opaque weaker flow. Preserve preflight, clean build, arm64 verification, Developer ID replacement signing, hardened runtime, notarization, stapling, DMG, Gatekeeper, credential redaction, and SHA-256 behavior while changing active commands/pins to Wails v3.

`.github/workflows/wails-macos.yml` must:

- install the exact beta.8 CLI;
- use locked Go/npm dependencies;
- run `gofmt`, vet, ordinary tests, race tests, all existing Buf/compatibility/negative/isolation gates;
- generate Wails bindings before standalone master compilation;
- compare two clean binding generations and enforce required/forbidden inventory;
- clean-build both frontends and prove no CDN/runtime package download;
- run all feature-005 Playwright/player/startup gates;
- package the macOS arm64 app at the established path;
- inspect architecture, plist/minimum OS, entitlements, icon, resources, signature, and final bundle;
- run source/generated/dependency/docs v2-forbidden scans; and
- upload the expected application artifact/evidence.

Credential-dependent public gates stay conditionally separate and cannot be reported passed without real evidence.

## Rollback Evidence

Create `docs/wails-v3-migration-rollback.md` without overwriting `docs/wails-migration-rollback.md`.

| Evidence | Required rule |
|---|---|
| Wails v2 source rollback | exact pre-migration `main` commit `f1084b3df8b5630862bdf7a0f347b599156653ef` |
| v2 artifact/hash | record only if actually built, manually accepted, and hashed; otherwise `NOT BUILT`/`NOT RUN` |
| data safety | make safety copies of selected session-v1/player-config-v1 files; no conversion |
| triggers | corruption/loss, bridge/capability drift, master/player parity loss, lifecycle leak, public regression, missing resource, beta crash, package/signature failure |
| drill | restore source or accepted artifact, open safety-copy data, run representative master/player local journey, record actual result |
| ownership/expiry | feature-006 implementer owns temporary coexistence; it expires at final cutover |

## Final Cutover Gate

Cut over only after every required parity, security, persistence, lifecycle, binding, frontend, player, package, soak, and rollback gate for the personal profile passes. Then remove:

- every Wails v2 import and Go dependency;
- v2 CLI install/commands, `wails.json`, post-build hooks, and generated-binding assumptions;
- `window.go`, `window.runtime`, `frontend/wailsjs`, and v2 event/runtime imports;
- obsolete binding workaround/source assertions if proven unnecessary;
- active v2 README, workflow, script, quickstart, and rollback instructions;
- any temporary runtime switch or dual implementation.

Retain completed historical specifications and the Electron→Wails v2 rollback record as labeled history. Run final source, generated-output, dependency graph, built-bundle, CI/script, and documentation scans. No permanent v2/v3 flag remains.
