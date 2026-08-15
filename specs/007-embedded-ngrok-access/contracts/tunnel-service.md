# Internal Contract: Provider-Neutral Tunnel Lifecycle

**Bugfix**: 2026-08-15 — ANALYZE-S1 binds the embedded forwarder to a trusted dedicated loopback
source without adding a listener or provider-specific core contract.

**Bugfix**: 2026-08-15 — BUG-001 removes the unbindable dedicated source and makes endpoint Basic
Auth part of provider-neutral start input.

**Bugfix**: 2026-08-16 — BUG-002 defines the startup-to-owned-endpoint lifetime handoff and the
strictly redacted optional disconnect-failure surface.

## Interfaces

The concrete ngrok SDK is confined to an adapter in `internal/tunnel`. Core lifecycle tests depend
on these provider-neutral semantics:

```text
TunnelService.Start(context, TunnelStartRequest) -> TunnelEndpoint | error

TunnelEndpoint.URL() -> URL
TunnelEndpoint.Done() -> receive-only completion signal
TunnelEndpoint.Close(context) -> error
```

`TunnelStartRequest` contains the fixed upstream URL, optional reserved domain, scoped account-token
buffer, and scoped player username/password buffers used for endpoint Basic Auth. It is internal and
non-serializable, has no logging/string formatter, and is cleared/dropped after adapter construction.
The production target is exactly `http://127.0.0.1:3690`; there is no source address or player-policy
input. `TunnelEndpoint` does not expose SDK Agent, session, account, policy, credentials,
diagnostics, or process handles.

The production adapter uses one `golang.ngrok.com/ngrok/v2` agent and one `EndpointForwarder`. It
uses the SDK's ordinary upstream connection and attaches one in-memory Basic Auth Traffic Policy;
it never writes a policy file. `Close` owns bounded endpoint close followed by agent disconnect and is
idempotent/concurrency-safe. A deterministic fake controls start completion, returned URL,
`Done`, close blocking/error, call order, active endpoint count, and clock without network access.

## Start semantics

- Nil/cancelled contexts, missing credential, invalid upstream, and invalid reserved domain fail
  before an endpoint is published.
- Invalid upstream, missing Basic Auth input, or policy construction failure ends before readiness;
  direct local/LAN service is unaffected.
- Start is bounded by the manager's 30-second terminal deadline; the 15-second target is measured
  separately and does not shorten correctness cleanup.
- The context passed to `Start` bounds acquisition only. Cancellation before commit aborts
  acquisition and cleans partial resources; once `Start` returns a committed endpoint, completion
  or cancellation of that startup context MUST NOT close `Done` or stop the endpoint.
- The adapter may return only an acquired, policy-protected endpoint; it emits no UI events itself.
- Empty reserved domain asks the provider for a random URL. Non-empty domain requests that exact
  HTTPS host and never silently falls back.
- `URL()` is untrusted until the manager validates scheme, authority, user info, path, query,
  fragment, and requested-domain equality.
- A provider error is converted to a typed redacted category before leaving the adapter. Secret
  values and raw provider diagnostic text are discarded.

## Done and close semantics

- `Done` closing while the endpoint is current and not intentionally stopping signals public
  failure. The manager withdraws URL before cleanup.
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
settings revision, Keychain scoped use, and secret-free event publication.
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
