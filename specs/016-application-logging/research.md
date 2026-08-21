# Research: Application Logging

## Required logging dependency

**Decision**: Pin `github.com/obalunenko/logger` v1.2.0 as a direct root-module runtime dependency.

**Rationale**: The user selected this package explicitly. v1.2.0 is its latest tagged stable release, exposes structured fields and errors through a small `Logger` interface, supports context lookup, and is built on the standard structured logging implementation already shipped with Go.

**Alternatives considered**: The standard-library logger is rejected because it does not satisfy the pinned package constraint. Direct use of another structured logger is rejected because it would create a second application logging policy.

## Initialization and dependency flow

**Decision**: Initialize the package once in `main`, keep the package default available for the earliest and latest process boundaries, and inject the returned `logger.Logger` into `App` and `player.Server` through their existing dependency/config structures.

**Rationale**: One production initialization satisfies the package contract, while interface injection keeps tests deterministic and prevents global logger mutation from making parallel tests race. The root application and player server are already the owners of the lifecycle events being recorded.

**Alternatives considered**: Global-only calls are rejected because tests would need to replace shared process state. Threading logger calls into domain services is rejected because it would spread diagnostic policy through transport-independent state owners.

## Logging surface and levels

**Decision**: Emit informational records for lifecycle milestones and successful important commands, informational cancellation records for native-dialog cancellation, warnings for expected command failures or rejected outcomes, and errors only for unexpected background failures or errors deliberately absorbed at the host boundary.

**Rationale**: This preserves the existing return/result ownership of errors and avoids duplicate logging. It also keeps expected validation feedback from appearing as a fatal or infrastructure failure.

**Alternatives considered**: Logging every desktop method and domain transition is rejected as noisy and likely to expose payload details. Logging only fatal process exits is rejected because it misses the absorbed Wails startup failure and operational command outcomes requested by the feature.

## Safe structured fields

**Decision**: Limit fields to stable operational metadata such as `operation`, `outcome`, `phase`, `event`, `revision`, `count`, `port`, public-access lifecycle state, and already-redacted error category. Never attach full request/result objects, paths, URLs, session or terminal content, character/player names, credentials, or raw public-access provider and secure-store errors.

**Rationale**: These fields make records queryable while satisfying the constitution's secret and private-content boundaries. Safe values are selected at each application boundary instead of attempting generic reflection-based redaction after the fact.

**Alternatives considered**: Generic payload logging plus a redaction filter is rejected because future fields could bypass the filter. Raw error logging for public access is rejected because provider and secure-store errors are explicitly sensitive boundaries.

## Swallowed event errors

**Decision**: Route non-fatal desktop event publication through one application helper that logs a structured delivery error and continues. Preserve startup's first server-info publication as a returned fatal startup error because it is part of the existing readiness contract.

**Rationale**: The helper removes repeated ignored-error sites while preserving the one publication whose failure must unwind startup.

**Alternatives considered**: Returning all event errors is rejected because it would change established command behavior. Continuing to discard them is rejected because it removes useful operational evidence.

## Test strategy

**Decision**: Add a concurrency-safe recording implementation of `logger.Logger` under `internal/testutil`, then test lifecycle milestones, absorbed startup errors, command outcomes, event-delivery failures, unexpected player serving exits, normal shutdown silence, and forbidden-value absence.

**Rationale**: Interface-level capture verifies observable logging behavior without asserting formatting or mutating the package-global logger. The existing secret-leak script remains an additional repository-wide check.

**Alternatives considered**: Matching text written to standard error is rejected as formatting-dependent and unsafe for parallel tests. Snapshotting entire records is rejected because tests should assert meaningful invariants, not freeze incidental wording.
