# Data Model: Application Logging

This feature introduces no persisted domain entity and no application contract. It emits transient operational records only.

## Operational Record

Represents one diagnostic event emitted by a production runtime boundary.

### Required attributes

- **Severity**: debug, information, warning, error, or fatal.
- **Message**: a stable human-readable summary of the event.
- **Timestamp**: assigned by the logging runtime when the record is emitted.

### Optional structured attributes

- **Operation**: stable action name such as application startup, session save, broadcast start, or public-access stop.
- **Outcome**: started, succeeded, failed, cancelled, rejected, or completed.
- **Phase**: safe application lifecycle phase.
- **Event**: the fixed desktop event name when delivery fails.
- **Revision**: a monotonic persistence or settings revision when it helps correlate an operation.
- **Count**: a non-sensitive aggregate such as connected-client count.
- **Port**: the bound local player port; addresses and URLs are not required.
- **Public-access state**: the safe lifecycle state already exposed by the redacted status contract.
- **Error category**: an existing redacted category for public access.
- **Error**: the wrapped Go error only at boundaries where the error is known not to carry credentials or user-authored content.

### Validation rules

- A record must never contain full command requests, results, domain aggregates, session or terminal documents, file contents, player or character names, provider tokens, player passwords, generated passwords, or raw provider and secure-store errors.
- Public-access records may use only the redacted lifecycle state, error category, and safe message already present in the public-access snapshot.
- Cancellation is an outcome, not an error.
- Expected validation rejection is not fatal and must not cause process termination.
- A normal player-server close is not an unexpected-serving error.

## Logger Dependency

Represents the process-scoped logger implementation injected into runtime owners.

### Relationships

- Process startup initializes exactly one logger.
- The root application and player server each receive that logger through their existing dependency structure.
- Tests substitute a concurrency-safe recording logger implementing the same interface.
- Domain and persistence owners do not acquire a logger dependency; their typed results are recorded at the application boundary.

## State transitions

Operational records observe existing transitions but do not own them:

- Application: constructed → starting player server → desktop loading → ready local → stopping → stopped, with failure variants preserved.
- Public access: disabled → starting → ready, stopping, or failed, using redacted snapshots.
- Important command: requested → succeeded, failed, cancelled, or rejected.
- Player serving: active → normal closed or unexpected failed.
