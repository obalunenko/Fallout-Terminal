<!--
Sync Impact Report
- Version change: 3.3.1 -> 3.3.2
- Modified principles: None
- Added principles: None
- Added sections:
  - Go Development Tool Modules
- Removed sections: None
- Expanded guidance:
  - Repository development, build, and packaging orchestration uses the already-required Go
    toolchain and standard library; Taskfile, Make, and global Wails CLI installation are prohibited.
  - The exact pinned Wails CLI remains isolated and is invoked only where Wails-specific generation
    is required.
- Follow-up TODOs:
  - Complete feature 006 reproducible-build, CI, package verification, and native acceptance tasks
    against `go run ./cmd/build ...`.
-->
# Fallout Terminal Constitution

## Project Identity

Fallout Terminal is a desktop application for tabletop RPG game masters. The native master
interface edits and publishes Fallout-style terminal content, while the embedded Go player server
synchronizes authoritative content and state with browser-based player clients. Saved campaign
state uses the portable version-1 JSON session document; live terminal, navigation, hacking,
connection, startup, and tunnel state is owned by the running application.

The production architecture is a Go 1.26 modular monolith whose accepted desktop runtime MUST be
an exactly pinned Wails major-version implementation. The accepted baseline is Wails v2.13.0.
Feature 006 is a bounded migration from that baseline to Wails v3; Wails v3 becomes the production
runtime only after feature 006's defined parity, package, and rollback gates pass. Until then,
Wails v2.13.0 remains the production runtime and Wails v3 remains a migration candidate.

The root Go module owns application composition, the trusted desktop bridge, and the embedded
player server. `frontend/` is the Vite-built browser-JavaScript game-master interface, `client/` is
the separately embedded browser-JavaScript player interface, and `internal/` contains application
services, domain logic, adapters, and platform integrations. Node.js is build, code-generation,
and browser-test tooling, not an application runtime. The supported deployment profile is macOS
13+ on Apple Silicon (`arm64`).

The Electron-to-Wails migration is complete. Legacy Electron source and behavior are not active
runtime boundaries, compatibility targets, or behavioral oracles. The documented Electron rollback
record MUST be preserved only as historical documentation and MUST NOT govern current behavior,
architecture, dependencies, acceptance, rollback, or release decisions. Feature 006 MUST create a
separate Wails v2 rollback record from the accepted pre-migration `main` commit and, when an
accepted executable is produced, its digest. The Electron record MUST NOT substitute for that
Wails v2 rollback reference.

## Core Principles

### I. Govern the Accepted Desktop Runtime

- The Wails v3 architecture MUST use explicit application, service, window, event, dialog,
  browser, asset, and lifecycle APIs. Root `main.go` and `app.go` own their composition and also
  compose filesystem-backed persistence, player-server startup, and optional tunnel startup.
- Wails application objects, services, windows, events, dialogs, browser integration, asset
  servers, lifecycle hooks, generated bindings, and `@wailsio/runtime` imports MUST remain platform
  adapters or composition concerns. Domain, control, session, player-configuration, live, and
  player services MUST remain independent of Wails.
- `frontend/` MUST access privileged desktop operations only through one narrow, explicitly
  registered desktop service and named events. Wails v3 generated bindings and
  `@wailsio/runtime` MAY implement this private transport. They MUST NOT expose a generic
  dispatcher or arbitrary filesystem, process, or environment access.
- `client/` owns the browser player experience and MUST operate without Wails, native desktop, or
  filesystem APIs and MUST have no path to desktop capabilities.
- `internal/domain/`, `internal/nav/`, `internal/hack/`, `internal/live/`, and
  `internal/control/` own domain, navigation, hacking, live-state, and coordination behavior.
  Their canonical logic MUST remain transport-independent and server-authoritative.
- `internal/session/` and `internal/playerconfig/` own durable JSON storage and native selection
  workflows. `internal/player/` owns the generated player RPC boundary and static asset delivery.
  `internal/platform/` owns desktop adapters and platform paths. `internal/tunnel/` owns optional
  public-access process and credential lifecycles.
- `sessions/` contains versioned JSON examples or data, not executable logic.

Changes crossing these boundaries MUST identify every affected producer, consumer, state owner,
contract adapter, and verification surface in the feature specification and implementation plan.
Generated protobuf messages MUST remain application-boundary representations; they MUST NOT become
canonical mutable domain aggregates or acquire domain, navigation, live-state, or hacking logic.

### II. Make Protobuf the Application Contract Source of Truth

Protocol Buffer schemas are the sole source of truth for every application-owned externally
observable or serialized structured contract, including:

- player RPC requests, responses, server streams, events, and public state;
- desktop bridge requests, results, events, and runtime-status DTOs, even while Wails remains the
  private transport;
- every known field of the version-1 session document; and
- serializable application runtime, player-server, startup, and tunnel configuration.

Generated Go and ECMAScript types and explicit boundary adapters MUST implement those contracts.
Application code MUST NOT maintain handwritten duplicates of transport DTOs.

Third-party and tool-native manifests, schemas, and metadata are outside this rule and MUST NOT be
duplicated in protobuf. This exclusion includes repository Go build orchestration, package
manifests, npm and Go lockfiles, framework-generated binding metadata, Buf configuration, GitHub
Actions workflows, and macOS plist files. The exclusion covers tool orchestration and metadata,
not application-owned structured desktop requests, results, events, runtime statuses, or
serializable configuration values. Non-serializable dependency-injection values, including
`fs.FS`, callbacks, interfaces, application or window objects, and process handles, likewise MUST
remain native implementation values rather than protobuf fields.

Static HTML, CSS, fonts, sounds, images, and other assets MAY use normal HTTP delivery because they
are resources, not RPC contracts. Asset delivery MUST NOT be used to bypass protobuf governance for
structured application messages or state.

### III. Use ConnectRPC and Keep State Server-Authoritative

All network RPC communication MUST use ConnectRPC with code generated from the governed schemas for
Go and ECMAScript. Handwritten JSON wire envelopes, handwritten RPC routers, and duplicated network
transport DTOs are prohibited.

Browser-originated mutations MUST use unary RPCs. Authoritative live updates MUST use
server-streaming RPCs. Client-streaming and bidirectional-streaming browser request bodies MUST NOT
be required because browser clients cannot provide them portably.

Player navigation, character selection, controller actions, and hacking actions are requests, not
local state changes. Go services MUST validate requests, mutate canonical process state, and publish
detached public projections over server streams. Browser clients MUST converge on authoritative
state and MUST NOT create divergent optimistic-only transitions.

Contract changes MUST specify RPC direction, message type, validation and rejection behavior,
authorization, ordering or revision semantics, stream reconnection behavior, and compatibility
impact. Transport handlers and generated messages MUST adapt to transport-independent application
services; they MUST NOT own domain rules or authoritative mutable state.

### IV. Separate Public and Private Capabilities

Public player services and private game-master capabilities MUST remain separate at schema,
service, adapter, listener, and authorization boundaries. The player Connect service and any public
ngrok listener MUST NEVER expose native dialogs, arbitrary file access, external URL opening,
`ForceHackSuccess`, tunnel credentials, private hacking candidates, passwords, random outcomes,
secret words, or any equivalent trusted capability or secret state.

The private Wails bridge MAY remain the transport for trusted desktop-only operations, but every
structured request, result, event, runtime-status payload, and serializable configuration value
crossing it MUST have a protobuf-defined contract and an explicit adapter. The master frontend
MUST reach privileged operations only through one narrow, explicitly registered desktop service
and named events. Wails v3 generated bindings and `@wailsio/runtime` MAY implement that transport,
but the bridge MUST NOT expose a generic dispatcher, arbitrary filesystem, process, or environment
access, or any player-facing route to desktop capabilities.

Browser-controlled values, file references, runtime commands, and external URLs MUST be validated
again at the privileged Go boundary. Content Security Policy MUST remain restrictive, and external
URL handling MUST allowlist HTTP and HTTPS. Public tunnel mode MUST fail closed without valid
credentials. Credentials and generated traffic-policy files MUST NOT enter session data, public
schemas, UI output, or logs. Temporary policy material and owned processes MUST be private and
cleaned up on success, failure, and shutdown paths.

### V. Evolve Schemas Safely and Reproducibly

- Protobuf APIs MUST use versioned packages.
- Published field numbers MUST remain stable. Removed fields MUST reserve both their numbers and
  names, and field numbers or names MUST NOT be silently reused.
- Every enum MUST define an `UNSPECIFIED` zero value.
- Fields MUST use explicit presence when absence has different meaning from the scalar default.
- Variant payloads MUST use `oneof` rather than parallel optional fields or ad hoc discriminator
  strings.
- Compatible additions MUST follow protobuf evolution rules. A breaking change MUST introduce a
  new versioned package or an explicit, documented migration.

Code generation MUST be deterministic and reproducible. Generator and protobuf runtime versions
MUST be pinned. Generated files MUST NOT be edited manually, and generation MUST produce no
unexplained working-tree drift. Once the repository establishes a protobuf compatibility baseline,
CI MUST run Buf formatting and linting, generation-drift checks, and breaking-change checks against
that baseline.

### VI. Preserve Portable Session JSON Version 1

The existing portable session JSON version 1 remains a compatibility contract. Adding protobuf
schemas MUST NOT switch persistence to protobuf binary or generic ProtoJSON, change established JSON
field names, discard compatible unknown JSON fields, or otherwise make existing version-1 documents
unreadable or lossy. Session persistence MUST use an explicit adapter that maps the protobuf-defined
known fields to the established JSON representation while preserving compatible unknown fields.

Specifications that change the session or player-configuration JSON shape MUST define versioning,
validation, defaults, references, unknown-field preservation, and migration or backward-
compatibility behavior before implementation. Saving MUST remain explicit about the target file and
MUST NOT silently overwrite, relocate, or transform unrelated user data.

On macOS, user-created sessions SHOULD default to `~/Documents/Fallout Terminal/Sessions/` after
explicit confirmation. Bundled samples inside the read-only application bundle MUST be copied only
after an explicit user action. App-managed metadata belongs in
`~/Library/Application Support/com.vaulttec.fallout-terminal/`. Autosave MUST continue targeting the
explicitly selected session path. Runtime-only state MUST NOT enter persistent JSON unless a feature
explicitly changes the persistence contract compatibly.

### VII. Complete Cutovers and Remove Superseded Protocols

A final feature cutover MUST remove its superseded transports, dependencies, generated or
handwritten fixtures, adapters, tests, and active documentation after parity is proven. Historical
records MAY remain when clearly labeled as history and MUST NOT be treated as current operating
instructions or acceptance criteria.

Temporary coexistence MUST have a bounded migration plan, an owner, parity criteria, and a removal
gate. Permanent dual protocols are prohibited unless an explicit, separately specified
compatibility requirement identifies the consumers, duration, verification, and retirement policy.

For feature 006, temporary Wails v2/v3 coexistence is permitted only on its migration branch. The
plan MUST name an owner, make coexistence expire at cutover, define parity criteria, and record an
immutable Wails v2 rollback reference based on the accepted pre-migration `main` commit and, when
produced, its accepted executable digest. The final production source MUST contain no active Wails
v2 import, CLI or configuration path, generated binding, or dual-runtime switch. Completed
historical specifications MUST retain their original target and MUST NOT be rewritten as though
they had targeted Wails v3.

## Dependency Rules

- Root composition and `internal/platform/` adapters MAY depend on
  `github.com/wailsapp/wails/v3` because they are the Wails v3 composition and platform boundaries.
  During feature 006 only, the migration branch MAY also retain the exactly pinned Wails v2.13.0
  dependency under the bounded coexistence rule. No other `internal/` package MAY import Wails v2
  or v3.
- Protobuf schema modules are upstream contract dependencies. Generated Go and ECMAScript outputs
  MUST depend only on pinned generators and runtimes and MUST be consumed through explicit boundary
  adapters.
- `internal/domain/`, `internal/nav/`, `internal/hack/`, `internal/live/`, `internal/control/`,
  `internal/session/`, `internal/playerconfig/`, and `internal/player/` MUST remain independent of
  Wails. Their existing permitted domain, protobuf-adapter, ConnectRPC, HTTP, and asset dependencies
  remain governed by the package-specific rules below.
- `internal/domain/`, `internal/nav/`, `internal/hack/`, `internal/live/`, and
  `internal/control/` MUST also remain independent of ConnectRPC, HTTP handlers, generated protobuf
  types as mutable state owners, and browser code.
- `internal/session/` and `internal/playerconfig/` MAY depend on domain models and protobuf-defined
  contract types through explicit JSON adapters; protobuf definitions MUST NOT replace the portable
  version-1 JSON persistence format.
- `internal/player/` MAY depend on ConnectRPC, generated Go service code, HTTP asset delivery, and
  narrow application-service interfaces. It MUST NOT depend on the master frontend or expose
  private game-master services.
- `internal/platform/` contains Wails and platform adapters. `internal/tunnel/` contains optional
  public-process integration and MUST NOT import Wails. Neither package owns domain rules, and only
  serializable application-owned configuration crossing a boundary belongs in protobuf.
- `frontend/` MAY call only the narrow registered desktop service through generated Wails bindings
  and consume named events through `@wailsio/runtime`. Every structured bridge payload MUST
  originate from a protobuf schema and pass through an explicit adapter; a generic dispatch surface
  is prohibited.
- `client/` MAY use browser APIs, generated ECMAScript Connect clients, server-streaming responses,
  and static HTTP assets. It MUST NOT depend on Wails, filesystem APIs, private services, or
  handwritten RPC envelopes.
- Repository Go build orchestration, package manifests, plist files, framework-generated binding
  metadata, other third-party tool configuration, and non-serializable injected dependencies MUST
  remain native to their owning tools or language and MUST NOT acquire parallel protobuf
  definitions.
- Go test assertions MUST use `github.com/stretchr/testify/assert` or
  `github.com/stretchr/testify/require`. Tests involving protobuf messages or descriptors MUST use
  `github.com/google/go-cmp/cmp` with the appropriate helpers under
  `google.golang.org/protobuf/testing`. These test-only dependencies MUST remain out of production
  package APIs.
- Every runtime, generator, or build dependency MUST have a concrete need recorded in the plan and
  be pinned reproducibly in its owning production module, isolated Go development-tool module,
  Buf configuration, or appropriate npm lockfile. Feature 006 MUST use mutually compatible exact
  versions for the `github.com/wailsapp/wails/v3` runtime module, the isolated `wails3` tool module,
  and the `@wailsio/runtime` npm package and its Vite plugin subpath. All owning `go.mod`, `go.sum`,
  and npm package lockfiles MUST be committed. Reproducible builds and CI MUST reject `@latest`,
  floating prerelease versions, uncommitted tool-module resolution, and any unrecorded Go-module,
  CLI, or frontend-runtime version mismatch.

## Go Development Tool Modules

Every Go executable used for repository development, generation, validation, build, packaging, or
release automation MUST be declared and executed as a repository-owned Go tool. This includes Buf,
Wails, protobuf and Connect generators, and any future Go-based command introduced into the
development workflow. Operating-system tools and non-Go tools remain governed by their native
installation and lock mechanisms.

- Each Go development tool MUST have one independent module at `tools/<tool>/`, containing its own
  `go.mod` and committed `go.sum`. A tool module MUST declare exactly one direct tool command with a
  Go `tool` directive; unrelated tool commands MUST NOT share that module.
- Tool modules MUST pin exact module versions and an explicit Go language version. They MUST NOT
  use pseudo-install scripts, floating versions, `@latest`, or depend on whichever executable is
  first on `PATH`.
- Repository commands MUST invoke a third-party tool through its owning module from the repository
  root, using `go tool -modfile=tools/<tool>/go.mod <command> ...` directly or from the checked-in
  standard-library-only `cmd/build` command. Taskfiles and Makefiles are prohibited build entry
  points. Development documentation, code-generation scripts, CI, and release automation MUST NOT
  use `go install` or a globally installed Go tool as their executable source.
- First-party orchestration in `cmd/build` and `internal/buildtool` MAY live in the root application
  module because it is repository source rather than a separately versioned executable dependency;
  it MUST use only the Go standard library and invoke versioned third-party tools through their
  isolated modules.
- The root application `go.mod` MUST contain production/runtime dependencies only and MUST contain
  no `tool` directive or tool block. It MUST NOT contain a `require`, `replace`, or other module
  entry whose only purpose is to build, install, pin, or execute a development tool. The root
  `go.sum` MUST NOT gain entries solely from resolving development tools.
- A module used by both application code and a development tool MAY appear in the root application
  module only when application packages actually require it at runtime or compile time. Its
  application version is governed independently from the tool module. When a product runtime and
  its CLI share an upstream project, the runtime remains pinned in the application module and the
  CLI remains independently pinned in its `tools/<tool>/` module.
- Running, downloading, tidying, or upgrading a tool through `tools/<tool>/go.mod` MUST NOT modify
  the root `go.mod` or root `go.sum`. Tool checksums and transitive tool dependencies belong only to
  that tool's `go.sum` and module graph.
- Each tool module MUST be tidied, reproducible, and verified independently. A tool-version change
  MUST update that module's `go.mod` and `go.sum`, compatibility research where applicable, every
  coupled runtime/frontend pin, and the generated or acceptance evidence affected by the change.
- CI MUST verify the expected set of tool modules, exact direct tool declarations, committed sums,
  zero `tool` directives and zero tool-only dependency entries in the root `go.mod`, no root module
  drift after tool resolution, and absence of global-install or unqualified Go-tool invocations in
  active scripts and documentation.

This isolation prevents generator and build dependencies from polluting the product module, makes
the invoked executable part of the repository's reviewed dependency graph, and lets tools evolve
independently without sacrificing deterministic local and CI behavior.

## Testing and Quality Gates

Go code MUST use colocated tests and deterministic fakes at filesystem, dialog, process, random,
network, clock, stream, and event boundaries. Concurrency-sensitive code MUST pass the race
detector. Browser journeys use Playwright under `tests/browser/`. No numeric coverage threshold or
repository-wide linter is currently defined; plans MUST choose verification proportionate to the
affected behavior instead of inventing or claiming either gate.

Go tests MUST follow these conventions:

- Ordinary assertions MUST use `github.com/stretchr/testify/assert` when the test can safely
  continue after a failure, or `github.com/stretchr/testify/require` when a failure invalidates the
  remaining test steps. Handwritten equality and error assertions MUST NOT duplicate those helpers.
- Tests MUST be table-driven when multiple cases share setup, execution, and verification and
  differ primarily by inputs or expected outputs. A focused single-case test MAY be used when a
  table would obscure materially different behavior or lifecycle requirements.
- Every test needing a context MUST use `t.Context()` (`testing.T.Context`) as its root test-scoped
  context and derive cancellation, timeout, or values from it. `context.Background()` and
  `context.TODO()` MUST NOT replace `t.Context()` in tests except when the behavior of those exact
  contexts is itself the subject of the test.
- Tests involving protobuf messages or descriptors MUST compare them with
  `github.com/google/go-cmp/cmp` and protobuf-aware helpers under
  `google.golang.org/protobuf/testing`, normally `protocmp.Transform()` or more specific
  `protocmp` options. Applicable message conformance checks MUST use `prototest`. Direct
  `reflect.DeepEqual` or generic Testify equality assertions on protobuf messages are prohibited.

Applicable commands MUST succeed before a change is considered complete:

- `gofmt -l .` produces no Go source paths.
- `go vet ./...` succeeds.
- `go test ./...` succeeds.
- `go test -race ./...` succeeds for changes affecting concurrent runtime, player, live, control,
  session, stream, startup, or tunnel behavior.
- `npm ci --prefix frontend` and `npm run build --prefix frontend` succeed for frontend, bridge,
  embedding, generated ECMAScript, or packaging changes.
- `npm ci --prefix client` and `npm run build --prefix client` succeed for player-client,
  embedding, asset, or packaging changes.
- `npm ci --prefix tests/browser` and `npm test --prefix tests/browser` succeed for affected player
  journeys when the required local environment is available.
- `go run ./cmd/build dev` is the sole repository-root development entry
  and passes affected interactive master/player journeys without a separately started frontend or
  player server.
- `go tool -modfile=tools/wails/go.mod wails3 generate bindings -clean ./...` succeeds and produces
  no unexplained working-tree drift; both generated bindings and protobuf generation remain
  deterministic and MUST NOT be edited manually.
- `go run ./cmd/build build` succeeds after both `frontend/` and `client/`
  production builds succeed.
- `go run ./cmd/build package` succeeds and
  produces a self-contained macOS Apple Silicon application for packaging-sensitive changes.
- Release candidates pass signing, hardened-runtime, notarization, stapling, DMG, and Gatekeeper
  checks when release credentials are available.

Schema and RPC changes MUST additionally verify:

- Buf formatting and linting;
- deterministic regeneration and a clean generation-drift check;
- Buf breaking-change detection against the established protobuf baseline once that baseline
  exists;
- generated Go and ECMAScript compilation;
- protobuf-aware Go comparisons and applicable message conformance checks;
- unary mutation validation and rejection behavior;
- server-stream ordering, revision handling, cancellation, disconnection, and reconnection;
- public/private service separation and the absence of privileged or secret fields from public
  descriptors and payloads; and
- version-1 JSON round trips, established field names, defaults, and preservation of compatible
  unknown JSON fields for session-contract changes.

CI MUST invoke Buf and every other Go-based development command through its owning `tools/<tool>`
module. It MUST enforce Buf formatting/linting, generation drift, and generated-code compilation
when protobuf schemas are present, and MUST add the breaking-change gate when a protobuf baseline
exists.
The GitHub Actions workflow MUST continue to enforce its configured Go test, Go vet, frontend clean-
build, player-client clean-build, startup-contract, exact Wails pin-consistency, clean Wails v3
binding-generation, and unsigned arm64 packaging gates. Native-dialog, audio, public-tunnel,
multi-browser, and signed-release checks MAY remain documented manual gates where reliable
automation or credentials are unavailable; unavailable checks MUST be reported, not claimed.

Reviews MUST verify production module boundaries, protobuf contract coverage, schema evolution,
generated-file integrity, persistence compatibility, authoritative synchronization, privileged-
interface exposure, public-listener security, macOS storage behavior, final cutover cleanup, and
owned-resource shutdown.

## Development Workflow

1. Branch from `develop` into a dedicated feature branch.
2. Specify user-visible behavior, independently testable scenarios, affected public and private
   capabilities, and every application-owned structured contract.
3. Update versioned protobuf schemas first, identifying RPC cardinality, presence, variants, stable
   field numbers, compatibility, and any version-1 JSON adapter impact before implementation.
4. Plan every affected producer, consumer, adapter, state owner, persistence rule, security
   boundary, generated artifact, cutover, rollback of the feature change, parity gate, package
   gate, and dependency-pin consistency gate. Feature 006 MUST identify its coexistence owner,
   expiry, parity criteria, and immutable Wails v2 rollback reference.
5. Regenerate pinned Go and ECMAScript code deterministically through the isolated
   `tools/<tool>` modules; never edit generated files or install a global Go tool.
6. Implement the smallest coherent vertical slice. Keep generated types at boundaries, domain logic
   transport-independent, mutations unary, live updates server-streamed, and private capabilities
   outside the public player service.
7. Run the automated and interactive verification defined in the plan. Go tests MUST follow the
   governed assertion, table-driven, `t.Context()`, and protobuf-comparison conventions. Run all
   applicable Buf, generation-drift, breaking-change, streaming, privilege-separation, and
   session-compatibility gates, and record unavailable checks.
8. Prove parity and pass package and rollback gates, then remove superseded transports,
   dependencies, fixtures, tests, and active documentation unless a separate compatibility
   requirement explicitly retains them. Feature 006 cutover MUST remove every active Wails v2
   import, CLI or configuration path, generated binding, and dual-runtime switch before Wails v3
   becomes the production runtime.
9. Update README, schema documentation, fixtures, compatibility specifications, and historical
   records when setup, operation, or governed behavior changes. Development, generation, CI,
   packaging, and release commands MUST continue to resolve every Go tool through its checked-in
   isolated module.

## Governance

This constitution governs Spec Kit artifacts and feature work in this repository. Amendments MUST
reflect an intentional project decision, document their rationale and migration impact, update the
Sync Impact Report, and increment the version below. Existing behavior is evidence, but an
accidental inconsistency, a legacy Electron implementation, or a superseded transport does not
automatically become a standard.

Constitution versions follow semantic versioning: MAJOR for backward-incompatible governance,
principle removal, or principle redefinition; MINOR for a new principle or materially expanded
guidance that remains backward-compatible; and PATCH for non-semantic clarification or correction.
The original ratification date MUST remain unchanged. The Last Amended date MUST record the date of
the adopted amendment.

Constitution checks are required during specification, planning, after design, and at final review.
Every plan MUST identify applicable contract, generation, compatibility, public/private boundary,
and cutover gates. Any violation MUST be listed in the plan's Complexity Tracking table with a
concrete rationale, an owner, a bounded duration, and a rejected simpler alternative. Reviewers
MUST reject unrecorded exceptions, manually edited generated files, schema-breaking field reuse,
public capability leakage, generic bridge dispatchers, and permanent dual protocols without an
explicit compatibility requirement.

**Version**: 3.3.2 | **Ratified**: 2026-08-09 | **Last Amended**: 2026-08-14
