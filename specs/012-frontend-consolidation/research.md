# Research: Frontend Consolidation

## Decision 1: One npm workspace, two application packages

**Decision**: Make `frontend/` the dependency workspace and place the two runnable packages at `frontend/client/` and `frontend/overseer/`.

**Rationale**: The repository gains one canonical frontend dependency installation and lockfile while each application retains its own scripts, Vite root, assets, and production output. This expresses the user's requested ownership directly and reduces duplicated Vite dependency resolution without merging runtime capabilities.

**Alternatives considered**:

- Keep two independent lockfiles beneath `frontend/`: structurally valid, but preserves duplicate installation and lock ownership after consolidation.
- Flatten both applications into one source directory: rejected because it obscures role ownership and makes accidental private/public imports easier.

## Decision 2: Preserve two production bundles and embeds

**Decision**: Build `frontend/client/dist` and `frontend/overseer/dist` independently and embed them as separate filesystems.

**Rationale**: The player HTTP server must never receive privileged desktop assets or Wails bindings. Filesystem separation is already a tested security property and remains useful even when both sources share a parent workspace.

**Alternatives considered**:

- Produce one combined distribution tree with subdirectories: rejected because it weakens the simplest auditable guarantee that the player server is handed only player assets.
- Share runtime modules between applications during this refactor: rejected because there is no behavior requirement and cross-role imports would expand security review scope.

## Decision 3: Rename the frontend identity, not unrelated stable contracts

**Decision**: Rename privileged source assets, window/build identifiers, user-facing copy, and focused test terminology to Overseer. Preserve established backend types whose names encode stable coordination contracts unless they directly name the frontend host.

**Rationale**: This makes the active application identity unambiguous while keeping the work a behavior-neutral frontend consolidation rather than an unrelated domain and wire-contract migration.

**Alternatives considered**:

- Mechanically rename every `Master` identifier across domain state and service tests: rejected because it creates broad API churn with no user-visible or directory-ownership benefit.
- Keep `master.js` and only change visible copy: rejected because the active code structure would continue contradicting the canonical Overseer identity.

## Decision 4: Perform a complete path cutover

**Decision**: Update generators, build orchestration, embeds, scripts, CI, tests, current documentation, constitution, and Spec Kit templates in the same change, then remove the top-level `client/` tree and old privileged paths.

**Rationale**: Generated files and clean-checkout markers otherwise tend to recreate old directories after an apparently successful move. A no-alias cutover makes stale references fail loudly and keeps one source of truth.

**Alternatives considered**:

- Retain symlinks or compatibility wrapper scripts at old paths: rejected because they create a permanent dual layout and allow future drift.
- Update only the production build: rejected because CI, developer commands, security scans, and fixture servers would still encode the old architecture.
