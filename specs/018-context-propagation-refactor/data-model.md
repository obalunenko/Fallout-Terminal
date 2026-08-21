# Data Model: Context Ownership

This refactor introduces no persisted entities. The following runtime-only ownership concepts define the implementation and tests.

## Process Root

- **Represents**: The lifetime and values created by an executable entry point.
- **Attributes**: context values, parent cancellation, logging/tracing metadata.
- **Relationships**: Parent of application composition and lifecycle-service state; never reconstructed below the entry point.
- **Validation**: Must be non-absent at constructors that retain or derive application-owned work.

## Operation Context

- **Represents**: One startup, desktop, player, persistence, provider, or cleanup operation.
- **Attributes**: parent/root lineage, deadline when applicable, current error, cancellation cause.
- **Relationships**: Derived from a process/lifecycle context or from an initiating request; passed unchanged through pure forwarding seams.
- **Validation**: A context-aware operation rejects an absent context before acquiring resources or mutating state.

## Owned Lifetime

- **Represents**: A cancellable resource scope such as the application runtime, player server, player subscription, public-access start, or embedded endpoint.
- **Attributes**: derived context, causal cancel function, stable close reason, idempotence guard, owned resources.
- **Relationships**: Child of a process/request operation; its cancellation is observed by goroutines, servers, streams, and provider work.
- **Validation**: Manual cancellation always supplies a non-empty cause; repeated cancellation preserves the first cause.

## Cleanup Scope

- **Represents**: Bounded resource release that may need to continue after ordinary application/request cancellation.
- **Attributes**: parent values, cancellation detached from the parent, existing timeout budget, timeout cause, completion cause.
- **Relationships**: Derived from the initiating context with cancellation removed but values retained; owns endpoint, ingress, HTTP server, or desktop cleanup.
- **Validation**: Cannot be unbounded and cannot use a new unrelated root.

## State Transitions

```text
process/request active
        |
        v
owned lifetime active
   | parent canceled --------> canceled(parent cause)
   | owner closes -----------> canceled(owner close cause)
   | startup fails ----------> canceled(failure cause)
   | timeout ----------------> canceled(timeout cause)
   | overflow/replacement ---> canceled(specific cause)
        |
        v
bounded cleanup active
   | resources released -----> canceled(cleanup complete cause)
   | budget expires ---------> canceled(cleanup timeout cause)
```

The first completed cancellation transition wins. Later repeated or racing close calls do not replace its cause.
