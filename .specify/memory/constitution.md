<!--
Sync Impact Report
- Version change: 3.0.0 -> 3.1.0
- Modified principles: None
- Added principles: None
- Added sections: None
- Removed sections: None
- Expanded guidance:
  - Dependency Rules now govern test-only assertion and protobuf comparison dependencies.
  - Testing and Quality Gates now require Testify assertions, table-driven Go tests where cases
    share a test flow, `testing.T.Context` for test-scoped contexts, and protobuf-aware `cmp`
    comparisons using helpers from `google.golang.org/protobuf/testing`.
  - Development Workflow now makes those Go test conventions part of planned verification.
- Follow-up TODOs: None
-->
# Fallout Terminal Constitution

## Project Identity

Fallout Terminal is a desktop application for tabletop RPG game masters. The native master
interface edits and publishes Fallout-style terminal content, while the embedded Go player server
synchronizes authoritative content and state with browser-based player clients. Saved campaign
state uses the portable version-1 JSON session document; live terminal, navigation, hacking,
connection, startup, and tunnel state is owned by the running application.

The production architecture is a Go 1.26 modular monolith built with Wails v2.13.0. The root Go
module owns application composition, the trusted desktop bridge, and the embedded player server.
`frontend/` is the Vite-built browser-JavaScript game-master interface, `client/` is the separately
embedded browser-JavaScript player interface, and `internal/` contains the application services,
domain logic, adapters, and platform integrations. Node.js is build, code-generation, and browser-
test tooling, not an application runtime. The supported deployment profile is macOS 13+ on Apple
Silicon (`arm64`).

The Electron-to-Wails migration is complete. Wails is the production desktop runtime, not a
migration candidate. Legacy Electron source and behavior are not active runtime boundaries,
compatibility targets, or behavioral oracles. The documented Electron rollback record MUST be
preserved only as historical documentation and MUST NOT govern current feature behavior,
architecture, dependencies, acceptance, or release decisions.

## Core Principles

### I. Govern the Current Production Architecture

- Root `main.go` and `app.go` own application composition, lifecycle, native dialogs,
  filesystem-backed persistence, player-server startup, Wails integration, and optional tunnel
  startup.
- `frontend/` MUST access privileged desktop operations only through explicitly registered,
  narrow Wails bridge methods and runtime events. It MUST NOT gain direct filesystem, process, or
  environment access.
- `client/` owns the browser player experience and MUST operate without Wails, native desktop, or
  filesystem APIs.
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

Third-party manifests and schemas are outside this rule and MUST NOT be duplicated in protobuf.
This exclusion includes `wails.json`, `package.json`, Buf configuration, GitHub Actions workflows,
and macOS plist files. Non-serializable dependency-injection values, including `fs.FS`, callbacks,
interfaces, and process handles, likewise MUST remain native implementation values rather than
protobuf fields.

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
structured request, result, event, and runtime-status payload crossing it MUST have a protobuf-
defined contract and an explicit adapter. The bridge MUST expose only narrow registered operations
and MUST NOT expose a generic RPC dispatcher.

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

## Dependency Rules

- Root `main.go` and `app.go` MAY depend on Wails v2.13.0, generated contract packages, and internal
  application services because they are composition and privileged bridge boundaries.
- Protobuf schema modules are upstream contract dependencies. Generated Go and ECMAScript outputs
  MUST depend only on pinned generators and runtimes and MUST be consumed through explicit boundary
  adapters.
- `internal/domain/`, `internal/nav/`, `internal/hack/`, `internal/live/`, and
  `internal/control/` MUST remain independent of Wails, ConnectRPC, HTTP handlers, generated
  protobuf types as mutable state owners, and browser code.
- `internal/session/` and `internal/playerconfig/` MAY depend on domain models and protobuf-defined
  contract types through explicit JSON adapters; protobuf definitions MUST NOT replace the portable
  version-1 JSON persistence format.
- `internal/player/` MAY depend on ConnectRPC, generated Go service code, HTTP asset delivery, and
  narrow application-service interfaces. It MUST NOT depend on the master frontend or expose
  private game-master services.
- `internal/platform/` contains Wails and platform adapters. `internal/tunnel/` contains optional
  public-process integration. Neither package owns domain rules, and only serializable
  application-owned configuration crossing a boundary belongs in protobuf.
- `frontend/` MAY call only narrow registered Wails bindings and consume runtime events. Every
  structured bridge payload MUST originate from a protobuf schema and pass through an explicit
  adapter; a generic dispatch surface is prohibited.
- `client/` MAY use browser APIs, generated ECMAScript Connect clients, server-streaming responses,
  and static HTTP assets. It MUST NOT depend on Wails, filesystem APIs, private services, or
  handwritten RPC envelopes.
- Third-party manifests, tool configuration, and non-serializable injected dependencies MUST remain
  native to their owning tools or language and MUST NOT acquire parallel protobuf definitions.
- Go test assertions MUST use `github.com/stretchr/testify/assert` or
  `github.com/stretchr/testify/require`. Tests involving protobuf messages or descriptors MUST use
  `github.com/google/go-cmp/cmp` with the appropriate helpers under
  `google.golang.org/protobuf/testing`. These test-only dependencies MUST remain out of production
  package APIs.
- Every runtime, generator, or build dependency MUST have a concrete need recorded in the plan and
  be pinned reproducibly in `go.mod`, Buf configuration, or the appropriate npm lockfile.

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
- `npm ci --prefix tests/browser` and `npm test --prefix tests/browser` succeed for affected player
  journeys when the required local environment is available.
- `wails dev` passes affected interactive master/player journeys.
- A clean `wails build -clean -platform darwin/arm64` produces a self-contained application for
  packaging-sensitive changes.
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

CI MUST enforce Buf formatting/linting, generation drift, and generated-code compilation when
protobuf schemas are present, and MUST add the breaking-change gate when a protobuf baseline exists.
The GitHub Actions workflow MUST continue to enforce its configured Go test, Go vet, frontend clean-
build, startup-contract, and unsigned arm64 packaging gates. Native-dialog, audio, public-tunnel,
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
   boundary, generated artifact, cutover, rollback of the feature change, and parity gate.
5. Regenerate pinned Go and ECMAScript code deterministically; never edit generated files.
6. Implement the smallest coherent vertical slice. Keep generated types at boundaries, domain logic
   transport-independent, mutations unary, live updates server-streamed, and private capabilities
   outside the public player service.
7. Run the automated and interactive verification defined in the plan. Go tests MUST follow the
   governed assertion, table-driven, `t.Context()`, and protobuf-comparison conventions. Run all
   applicable Buf, generation-drift, breaking-change, streaming, privilege-separation, and
   session-compatibility gates, and record unavailable checks.
8. Prove parity, then remove superseded transports, dependencies, fixtures, tests, and active
   documentation unless a separate compatibility requirement explicitly retains them.
9. Update README, schema documentation, fixtures, compatibility specifications, and historical
   records when setup, operation, or governed behavior changes.

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

**Version**: 3.1.0 | **Ratified**: 2026-08-09 | **Last Amended**: 2026-08-13
