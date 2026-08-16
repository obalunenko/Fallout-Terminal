# Internal Contract: Provider-Neutral Tunnel Lifecycle

**Bugfix**: 2026-08-15 — ANALYZE-S1 binds the embedded forwarder to a trusted dedicated loopback
source without adding a listener or provider-specific core contract.

**Bugfix**: 2026-08-15 — BUG-001 removes the unbindable dedicated source and makes endpoint Basic
Auth part of provider-neutral start input.

**Bugfix**: 2026-08-16 — BUG-002 defines the startup-to-owned-endpoint lifetime handoff and the
strictly redacted optional disconnect-failure surface.

**Bugfix**: 2026-08-16 — BUG-003 removes player credentials and Traffic Policy from the provider
adapter; the SDK endpoint now forwards only to an owned private application ingress.

**Bugfix**: 2026-08-16 — BUG-003 second verification reconciliation makes unexpected completion
deny-before-withdraw and recognizes the FR-056 effective secret source.

## Interfaces

The concrete ngrok SDK is confined to an adapter in `internal/tunnel`. Core lifecycle tests depend
on these provider-neutral semantics:

```text
TunnelService.Start(context, TunnelStartRequest) -> TunnelEndpoint | error

TunnelEndpoint.URL() -> URL
TunnelEndpoint.Done() -> receive-only completion signal
TunnelEndpoint.Close(context) -> error
```

`TunnelStartRequest` contains the owned private-ingress upstream URL, optional reserved domain, and
scoped account-token buffer. ~~BUG-001 also supplied player username/password for endpoint Basic
Auth.~~ **BUG-003** removes those fields from provider input. The request is internal and
non-serializable, has no logging/string formatter, and is cleared/dropped after adapter construction.
Only the application ingress targets exactly `http://127.0.0.1:3690`; the SDK target is its
loopback-only address. `TunnelEndpoint` does not expose SDK Agent, session, account, policy, credentials,
diagnostics, or process handles.

The production adapter uses one `golang.ngrok.com/ngrok/v2` agent and one `EndpointForwarder`. It
uses the SDK's ordinary upstream connection without Traffic Policy; player authentication belongs
to the separate application ingress contract. `Close` owns bounded endpoint close followed by agent disconnect and is
idempotent/concurrency-safe. A deterministic fake controls start completion, returned URL,
`Done`, close blocking/error, call order, active endpoint count, and clock without network access.

## Start semantics

- Nil/cancelled contexts, missing credential, invalid upstream, and invalid reserved domain fail
  before an endpoint is published.
- Invalid upstream or missing provider account credential ends before readiness; ingress credential
  validation and exact-Host activation are owned by the lifecycle/ingress contract, and direct
  local/LAN service is unaffected.
- Start is bounded by the manager's 30-second terminal deadline; the 15-second target is measured
  separately and does not shorten correctness cleanup.
- The context passed to `Start` bounds acquisition only. Cancellation before commit aborts
  acquisition and cleans partial resources; once `Start` returns a committed endpoint, completion
  or cancellation of that startup context MUST NOT close `Done` or stop the endpoint.
- The adapter may return only an acquired endpoint targeting the owned private ingress; it emits no
  UI events and does not decide public authentication readiness itself.
- Empty reserved domain asks the provider for a random URL. Non-empty domain requests that exact
  HTTPS host and never silently falls back.
- `URL()` is untrusted until the manager validates scheme, authority, user info, path, query,
  fragment, and requested-domain equality.
- A provider error is converted to a typed redacted category before leaving the adapter. Secret
  values and raw provider diagnostic text are discarded.

## Done and close semantics

- `Done` closing while the endpoint is current and not intentionally stopping signals public
  failure. ~~The manager withdraws URL before cleanup.~~ **BUG-003 reconciliation** The manager
  first sets the owned ingress to deny-all, then withdraws the URL and closes endpoint/ingress.
- An adapter MAY expose an internal `Failure() error` extension after `Done`; it may contain only a
  fixed application category and a validated `ERR_NGROK_<digits>` code. The base provider-neutral
  interface, status, events, and diagnostics never expose raw SDK error text.
- `Done` closing for an old generation has no status effect; any remaining endpoint is still closed.
- Cancellation of an active lifecycle intent causes the manager to call `Close`; cancellation
  alone is not accepted as proof of endpoint cleanup. Endpoint `Close`, not the completed startup
  context, owns cancellation of the committed SDK forwarder lifetime.
- `Close` can be called before Start completes, after `Done`, after partial acquisition, and
  concurrently. Calls join one close result rather than acquiring or leaking another endpoint.
- Cleanup drops SDK endpoint/agent references, goroutines, credential buffers, and monitor work
  within the caller's remaining budget.
- A close failure retains ownership for bounded retry during the same overall shutdown, but does not
  republish the URL.

## Manager sequencing

The `PublicAccessManager` is the sole owner of one `TunnelEndpoint`, lifecycle generation,
settings revision, ~~Keychain scoped use~~ **BUG-003 reconciliation** scoped effective-secret use
(production Keychain or exact FR-056 dev/test override), and secret-free event publication.
It performs no Wails, Keychain, or provider-specific operation except through injected interfaces.

Network calls occur outside the manager lock. On return, generation/revision/state are checked
before publication. Event callbacks also run outside the lock and receive detached secret-free
snapshots. Start/Stop/Reconfigure are safe under concurrent calls and preserve latest valid user
intent.

## Cutover constraint

After parity, no implementation of this contract may launch an external ngrok process. Delete
`ProcessRunner`, `ProcessHandle`, `OwnedProcess`, Darwin guardian/process-group code, log URL parser,
`NGROK_BIN`, CLI executable selection, and their process-specific tests. The SDK adapter and fake
inherit the old timeout, early completion, concurrent stop, partial acquisition, redaction, and
cleanup guarantees; the old production mechanism does not remain as fallback.
