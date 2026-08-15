# Public HTTP Contract: Host Activation and Basic Auth

**Bugfix**: 2026-08-15 — ANALYZE-S1 replaces unresolved Host-only ingress inference with an owned
source-bound public ingress class.

**Bugfix**: 2026-08-15 — BUG-001 supersedes the source/Host contract with endpoint Basic Auth after
the required `127.0.0.2` bind failed on target macOS.

## Current contract (BUG-001)

Public authentication is owned by the one ngrok Agent Endpoint, not by `internal/player`.

1. The provider adapter receives scoped account-token, username, and password buffers.
2. It constructs one in-memory Basic Auth Traffic Policy and attaches it to `Agent.Forward` while
   the endpoint URL remains private.
3. The endpoint forwards directly to the existing `http://127.0.0.1:3690` service. It uses no
   custom source-bound dialer and starts no second player server or proxy.
4. Only after `Forward` succeeds and its HTTPS URL is validated may lifecycle state become `ready`
   and expose that URL.
5. Missing or incorrect public Basic Auth is rejected by ngrok with `401`; correct credentials are
   forwarded unchanged to the existing static/Connect application.
6. Direct local/LAN clients connect to the player listener without traversing ngrok and therefore
   receive no Basic Auth challenge.
7. The player handler performs no public `RemoteAddr`, forwarding-header, or Host classification.

The Traffic Policy is ephemeral provider configuration. It is never written to disk, protobuf,
JSON, environment, arguments, events, status, logs, diagnostics, fixtures, or frontend storage.
The username/password may exist only in the private mutation path, Keychain scoped-use buffers,
SDK policy construction, and the standard browser Basic Authorization exchange.

Endpoint creation with its policy is the atomic readiness boundary. Stop, reconfigure, failure,
and quit withdraw the URL and close the endpoint/Agent within the shared deadline. There is no
separate player-policy activation/deactivation sequence.

Deterministic tests prove policy configuration intent, secret confinement, URL validation,
lifecycle ordering, and unchanged local behavior without rendering the policy secret. An explicit
credential-gated real run checks missing/wrong/correct Basic Auth and one non-empty incremental
`Subscribe`; unavailable prerequisites are `NOT RUN` and do not invalidate deterministic gates.

The threat model is casual personal-game sharing and prevention of accidental entry. It does not
claim protection against a malicious local process, provider compromise, or deliberate hostile
protocol manipulation.

<details>
<summary>Superseded pre-BUG-001 source/Host contract (non-normative history)</summary>

## Scope

This contract wraps the existing player application's static and generated ConnectRPC handler on
the one authoritative listener at port `3690`. It changes request admission only. Public player
protobuf schemas, procedure names, request limits, authoritative state, streaming messages, and
browser reconnect behavior remain unchanged.

## Policy boundary

`internal/player` owns a provider-neutral, thread-safe policy for the lifetime of the player server.
The privileged lifecycle side exposes only:

```text
Activate(generation, exactExternalHost, username, password) -> applied
Deactivate(generation) -> applied
```

Generation is monotonic. An activation or deactivation older than the current policy generation is
ignored, so a late provider completion cannot revive a stale Host. Host, username, and password are
installed as one locked grant; no request can observe a partially changed tuple.

The request side first classifies the accepted connection source, then validates Host and Basic
Auth while holding a short read lock, and releases the lock before invoking any static or ConnectRPC
handler. It never wraps or buffers the response writer and never holds policy synchronization over
a `Subscribe` stream.

Root composition configures the SDK adapter and player policy with the same immutable public source
`127.0.0.2`. The adapter reaches the existing `127.0.0.1:3690` listener only through an owned
`tcp4` dialer bound to that source. The request boundary reads the actual `RemoteAddr`; forwarding
headers cannot create direct ingress or local/LAN privilege.

## Host classes

Normalize an authority by trimming optional trailing DNS dot, lowercasing DNS, and validating one
legal host plus optional valid port. Reject whitespace, control characters, user info, path, query,
fragment, malformed IPv6, or ambiguous colon forms.

| Class | Definition | Admission |
|---|---|---|
| direct local/LAN | Transport source is loopback/private/link-local but not `127.0.0.2`, and authority is exact `localhost`, loopback, or an actual private/link-local interface authority recorded for the running listener; port is the player port when present. | Allowed without Basic Auth in every public lifecycle state. |
| public ingress | Transport source is exactly `127.0.0.2`, regardless of Host or forwarding headers. | Never eligible for local/LAN bypass; only exact active public Host plus correct Basic Auth proceeds. |
| unknown direct/external | Every valid direct authority that is neither local/LAN nor the active public Host. | Rejected fail-closed. |
| malformed | Any authority that fails normalization. | Rejected fail-closed. |

Local/LAN membership is an immutable concrete allow set, not “anything other than the public Host,”
not a suffix wildcard, and not inferred from an `X-Forwarded-*` header. Provider-supplied forwarding
headers are untrusted for authorization. A real endpoint acceptance test must confirm that attempts
to override Host at the public edge cannot obtain local/LAN admission.

~~Local/LAN bypass is valid only for traffic that is trustworthy as direct local/LAN ingress.
Because the embedded forwarder targets the same loopback server, `Host` alone is not such proof.
Before this design can be implemented, authoritative documentation/source for the pinned provider
runtime must establish that remote requests cannot reach the upstream with `localhost`, loopback,
or LAN authority; a real edge probe must verify it when prerequisites exist. If that property is
not guaranteed, this contract requires a revised, trustworthy ingress discriminator; the endpoint
remains fail-closed and cannot enter `ready` in its absence.~~

Local/LAN bypass is valid only for direct ingress whose accepted transport source is itself
loopback/private/link-local and is not the dedicated public source. A public-ingress request carrying
`localhost`, loopback, LAN, stale,
arbitrary, or malformed Host is denied before any static/RPC handler. Failure to bind or inject the
owned source-bound dialer prevents endpoint readiness rather than falling back to Host inference or
a default dialer.

## Request decision table

Ingress and Host admission run before both static and RPC routing.

| Ingress | Request authority | Policy state | Credentials | Result |
|---|---|---|---|---|
| direct | local/LAN | any | none/any | Continue unchanged; no Basic challenge. |
| public (`127.0.0.2`) | exact active public Host | ready grant | correct pair | Continue unchanged. |
| public (`127.0.0.2`) | exact active public Host | ready grant | missing/malformed/wrong | HTTP `401`, `WWW-Authenticate: Basic realm="Fallout Terminal Players"`, no protected body. |
| direct non-local | exact active public Host | ready grant | correct pair | Continue unchanged as public authority. |
| direct non-local | exact active public Host | ready grant | missing/malformed/wrong | HTTP `401`, `WWW-Authenticate: Basic realm="Fallout Terminal Players"`, no protected body. |
| public (`127.0.0.2`) | local/LAN, prospective, old, unknown, or malformed Host | any | even current correct pair | HTTP `403`, no static/RPC handler call. |
| direct | prospective, old, unknown external, or malformed Host | any | any | HTTP `403`, no handler call. |

Username and password comparisons are constant-time over byte buffers. The application adds no
attempt counter, cooldown, or account lockout; a correct request immediately after invalid attempts
succeeds unless ngrok independently throttles it.

## Atomic readiness and teardown

Required readiness order:

1. policy has trusted public source `127.0.0.2`, direct local/LAN allow set, and default deny;
2. provider creates an endpoint forwarding from owned source `127.0.0.2` to
   `http://127.0.0.1:3690` while URL is private;
3. lifecycle validates HTTPS URL and reserved-domain match;
4. `Activate` binds its exact Host and Basic Auth pair at the current generation;
5. lifecycle marks ready and only then emits/publishes the URL.

Required stop/reconfigure/failure/quit order:

1. advance generation and `Deactivate` so every non-local Host is denied;
2. withdraw URL/status;
3. cancel monitor/start work and close endpoint/agent within its deadline;
4. continue existing player/session/desktop shutdown as applicable.

If endpoint acquisition succeeds after cancellation or for an old generation, it is closed without
calling `Activate`. If policy activation fails, URL is never published and endpoint is closed.

## Static and ConnectRPC coverage

The same admission decision applies to:

- `/` and SPA fallback;
- every JavaScript, CSS, font, image, and sound/static resource;
- `fallout.terminal.player.v1.PlayerService/Subscribe`;
- `SelectCharacter`, `Navigate`, `Guess`, `ActivatePattern`, and `SoundManifest`;
- unsupported paths in the public player service namespace.

After admission, existing CSP, `nosniff`, same-Host Origin validation, encoded/decoded message
limits, Connect error mapping, static traversal protection, and procedure allowlist remain in their
current order. Unsupported RPCs still return Connect `unimplemented` only after admission.

## Streaming and reconnect

An authenticated `Subscribe` retains server-streaming cardinality. The middleware checks only Host
and headers before delegation and does not buffer, replace, or await the response. Acceptance
requires observing the initial complete snapshot and at least one later non-empty update before
stream completion through:

1. deterministic in-process policy/server integration;
2. the protected browser fixture (not real-endpoint evidence);
3. an explicit credential-gated real ngrok URL, or `NOT RUN` when unavailable.

The same real run separately records attempts to override the public-edge Host with every local/LAN
authority and verifies the observed upstream source is `127.0.0.2`, as `PASS`, `FAIL`, or `NOT RUN`.
`NOT RUN` is honest lack of real-service evidence and cannot be called real endpoint proof. The
unconditional security gate is instead the pinned SDK API/source review plus deterministic adapter,
real TCP source-binding, handler, race, and packaged macOS tests proving that every dedicated-source
request is public and that source-bind failure has no default-dialer fallback.

Reconnect must use the same current origin and credentials and converge within the specification's
five-second test condition. A stopped/stale URL must not reconnect successfully.

## Failure isolation

Provider, network, account, domain, Keychain, policy activation, or endpoint completion failure
changes only public status and admission. Local/LAN static, unary, streaming, selection, navigation,
hacking, sound, and reconnect journeys remain available without Basic Auth or application restart.

</details>
