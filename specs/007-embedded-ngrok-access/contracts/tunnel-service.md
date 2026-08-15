# Internal Contract: Provider-Neutral Tunnel Lifecycle

**Bugfix**: 2026-08-15 — ANALYZE-S1 binds the embedded forwarder to a trusted dedicated loopback
source without adding a listener or provider-specific core contract.

## Interfaces

The concrete ngrok SDK is confined to an adapter in `internal/tunnel`. Core lifecycle tests depend
on these provider-neutral semantics:

```text
TunnelService.Start(context, TunnelStartRequest) -> TunnelEndpoint | error

TunnelEndpoint.URL() -> URL
TunnelEndpoint.Done() -> receive-only completion signal
TunnelEndpoint.Close(context) -> error
```

`TunnelStartRequest` contains the fixed upstream URL, fixed `tcp4` source address, optional reserved
domain, and a scoped account credential buffer. It is an internal non-serializable value. It has no
logging/string formatter and must be cleared/dropped after adapter construction. The production
route is exactly source `127.0.0.2` to target `127.0.0.1:3690`; root composition gives the player
policy the same trusted source. `TunnelEndpoint` does not expose SDK Agent, session, account,
diagnostics, or process handles.

The production adapter uses one `golang.ngrok.com/ngrok/v2` agent and one
`EndpointForwarder`. Its `WithUpstreamDialer` implementation forces `tcp4`, binds local address
`127.0.0.2`, rejects every target except `127.0.0.1:3690`, and never falls back to the SDK default
dialer. `Close` owns bounded endpoint close followed by agent disconnect and is
idempotent/concurrency-safe. A deterministic fake controls start completion, returned URL,
`Done`, close blocking/error, call order, active endpoint count, and clock without network access.

## Start semantics

- Nil/cancelled contexts, missing credential, invalid upstream, and invalid reserved domain fail
  before an endpoint is published.
- Invalid source/target, unavailable `127.0.0.2` binding, or any attempt to use another/default
  dialer fails closed before readiness; direct local/LAN service is unaffected.
- Start is bounded by the manager's 30-second terminal deadline; the 15-second target is measured
  separately and does not shorten correctness cleanup.
- The adapter may return only an acquired endpoint; it never activates player policy or emits UI
  events itself.
- Empty reserved domain asks the provider for a random URL. Non-empty domain requests that exact
  HTTPS host and never silently falls back.
- `URL()` is untrusted until the manager validates scheme, authority, user info, path, query,
  fragment, and requested-domain equality.
- A provider error is converted to a typed redacted category before leaving the adapter. Secret
  values and raw provider diagnostic text are discarded.

## Done and close semantics

- `Done` closing while the endpoint is current and not intentionally stopping signals public
  failure. The manager deactivates policy and withdraws URL before cleanup.
- `Done` closing for an old generation has no status effect; any remaining endpoint is still closed.
- Context cancellation causes the manager to call `Close`; cancellation alone is not accepted as
  proof of endpoint cleanup.
- `Close` can be called before Start completes, after `Done`, after partial acquisition, and
  concurrently. Calls join one close result rather than acquiring or leaking another endpoint.
- Cleanup drops SDK endpoint/agent references, goroutines, credential buffers, and monitor work
  within the caller's remaining budget.
- A close failure retains ownership for bounded retry during the same overall shutdown, but does not
  reactivate public policy or republish URL.

## Manager sequencing

The `PublicAccessManager` is the sole owner of one `TunnelEndpoint`, lifecycle generation,
settings revision, Keychain scoped use, player-policy mutations, and secret-free event publication.
It performs no Wails, Keychain, or provider-specific operation except through injected interfaces.

Network calls occur outside the manager lock. On return, generation/revision/state are checked
before activation. Event callbacks also run outside the lock and receive detached secret-free
snapshots. Start/Stop/Reconfigure are safe under concurrent calls and preserve latest valid user
intent.

## Cutover constraint

After parity, no implementation of this contract may launch an external ngrok process. Delete
`ProcessRunner`, `ProcessHandle`, `OwnedProcess`, Darwin guardian/process-group code, log URL parser,
`NGROK_BIN`, CLI executable selection, and their process-specific tests. The SDK adapter and fake
inherit the old timeout, early completion, concurrent stop, partial acquisition, redaction, and
cleanup guarantees; the old production mechanism does not remain as fallback.
