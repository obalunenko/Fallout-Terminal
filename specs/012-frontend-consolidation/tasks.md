# Tasks: Frontend Consolidation

## Phase 1: Setup

**Wave 1 — failing contracts for the target layout:**

- [x] **T001** Add failing static and build-plan assertions for the nested `client`/`overseer` roots, renamed privileged assets, distinct embeds, and single workspace install · `internal/platform/assets_test.go`, `production_resources_test.go`, `internal/buildtool/buildtool_test.go`

**⟶ Wait for Wave 1 to finish, then continue to the foundational cutover.**

## Phase 2: Foundational — role-owned workspace

**Wave 1 — filesystem cutover:**

- [x] **T002** Move the complete player package, generated code, static assets, and output marker from the top-level client tree into its role-owned application root · `client/`, `frontend/client/`

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — privileged filesystem cutover:**

- [x] **T003** Move the privileged package, bindings, sources, and output marker into the Overseer root and rename the master JavaScript/CSS entry assets · `frontend/src/`, `frontend/bindings/`, `frontend/dist/`, `frontend/overseer/`

**⟶ Wait for Wave 2 to finish, then:**

**Wave 3 — shared dependency ownership:**

- [x] **T004** Define the private npm workspace, role-member manifests, exact shared dependencies, scripts, and one regenerated lockfile · `frontend/package.json`, `frontend/package-lock.json`, `frontend/client/package.json`, `frontend/overseer/package.json`

**Checkpoint**: Both applications and all owned inputs now have canonical locations beneath `frontend`, and dependency ownership is ready for downstream consumers.

## Phase 3: User Story 1 — One frontend workspace (Priority: P1) 🎯 MVP

**Goal**: Make every producer and consumer use the consolidated role-based paths, with no active top-level client tree.

**Independent Test**: From a clean dependency install, generate player and desktop bindings, build both applications, and inspect the two distinct outputs under `frontend`.

### Implementation

**Wave 1 — independent producer updates:**

- [x] **T005** [P] [US1] Point protobuf generation, version checks, public-contract scans, and generated-code drift checks at the nested client · `proto/buf.gen.es.yaml`, `scripts/proto-generate.sh`, `scripts/proto-check.sh`
- [x] **T006** [P] [US1] Update the Go-owned preparation graph to install the workspace once, build each member by role, and generate bindings into Overseer · `internal/buildtool/buildtool.go`, `internal/buildtool/buildtool_test.go`
- [x] **T007** [P] [US1] Embed and subtree the nested Overseer and client distributions as two explicit filesystems · `main.go`, `production_resources_test.go`
- [x] **T008** [P] [US1] Update repository convenience dependency targets and tracked/ignored artifact rules for the workspace layout · `Makefile`, `.gitignore`, `.gitattributes`

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — consumer/path contract integration:**

- [x] **T009** [US1] Migrate active frontend asset, generated-binding, clean-checkout, embed, and privilege-separation assertions to the canonical nested paths · `internal/platform/assets_test.go`, `internal/platform/startup_test.go`, `internal/player/http.go`

**Checkpoint**: User Story 1 is independently functional when both nested applications build and native composition selects the correct distinct bundles.

## Phase 4: User Story 2 — Overseer identity replaces master identity (Priority: P1)

**Goal**: Establish Overseer as the canonical name of the trusted frontend in active presentation, entry assets, host code, and focused verification.

**Independent Test**: Open the privileged application and search active frontend/host/build verification surfaces; the UI and role identifiers say Overseer while stable backend contract names remain untouched.

### Implementation

**Wave 1 — independent role surfaces:**

- [x] **T010** [P] [US2] Load the renamed entry assets and replace visible master-control copy with Overseer terminology without changing DOM behavior · `frontend/overseer/src/index.html`, `frontend/overseer/src/overseer.js`, `frontend/overseer/src/overseer.css`
- [x] **T011** [P] [US2] Rename the Wails window constructor, options, close-registration interfaces, variables, and tests from master to Overseer · `wails_host.go`, `wails_host_test.go`, `main.go`
- [x] **T012** [P] [US2] Rename active browser-fixture routes, source loading, variables, and assertion messages that identify the privileged frontend · `tests/browser/fixture-server/main.go`, `tests/browser/*.spec.mjs`, `tests/browser/fixtures/desktop-bindings.js`

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — terminology integration:**

- [x] **T013** [US2] Update active Go comments, build step names, plist descriptions, and smoke-check terminology that directly identifies the trusted frontend as Overseer · `app.go`, `app_contract.go`, `internal/buildtool/`, `build/darwin/`, `scripts/`

**Checkpoint**: User Story 2 is independently functional when the private desktop role is presented and maintained as Overseer with unchanged privileges and behavior.

## Phase 5: User Story 3 — Existing workflows remain dependable (Priority: P2)

**Goal**: Complete the no-alias cutover across automation, current guidance, generation, builds, and behavioral verification.

**Independent Test**: Run the governed clean-generation, frontend, Go, browser, native-build, and obsolete-path checks from the repository root.

### Implementation

**Wave 1 — independent automation and guidance updates:**

- [x] **T014** [P] [US3] Migrate CI caches, generation diffs, reproducibility digests, package verification, security scans, and cutover fixtures to the workspace paths · `.github/workflows/wails-macos.yml`, `scripts/`
- [x] **T015** [P] [US3] Update current setup guidance, architecture governance, and future Spec Kit templates to the `frontend/client` and `frontend/overseer` model · `README.md`, `.specify/memory/constitution.md`, `.specify/templates/`

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — deterministic generated artifacts:**

- [x] **T016** [US3] Install the locked workspace, regenerate public ECMAScript contracts and private Wails bindings at their canonical locations, and confirm no obsolete output tree is recreated · `frontend/package-lock.json`, `frontend/client/gen/`, `frontend/overseer/bindings/`

**Checkpoint**: User Story 3 is independently functional when a clean root workflow regenerates and builds both applications without old paths or extra manual commands.

## Phase 6: Polish & Cross-Cutting Validation

**Wave 1 — focused validation:**

- [x] **T017** Validate JavaScript syntax, both production frontend builds, focused browser journeys, Go formatting/vet/tests, and static resource contracts against SC-002, SC-003, and SC-005 · `frontend/`, `tests/browser/`, `internal/platform/`, `internal/buildtool/`

**⟶ Wait for Wave 1 to finish, then:**

**Wave 2 — release-shape and cleanup gate:**

- [x] **T018** Run deterministic generation, native build/package-appropriate checks, and an active-tree scan proving zero obsolete top-level client or master-frontend identities against SC-001 and SC-004 · `scripts/`, `main.go`, `frontend/`, `.github/workflows/wails-macos.yml`

## Dependencies & Execution Order

- Phase 1 establishes failing contracts before any move.
- Phase 2 performs the ordered filesystem and workspace cutover and blocks all user stories.
- User Story 1 updates producers first, then path consumers; it establishes the buildable MVP.
- User Story 2 can proceed after Phase 2 and joins its independent UI, host, and fixture changes before cross-cutting terminology cleanup.
- User Story 3 updates automation and governance independently, then regenerates only after all destinations are stable.
- Polish runs focused validation before the final deterministic, native, and obsolete-path gates.
