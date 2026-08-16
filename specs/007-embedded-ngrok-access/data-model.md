# Data Model: Embedded ngrok Public Access

**Bugfix**: 2026-08-15 — ANALYZE-S1 adds a non-serializable trusted ingress route so local/LAN
bypass never depends on forwarded `Host` behavior.

**Bugfix**: 2026-08-15 — BUG-001 supersedes the player-bound source/Host grant with an ephemeral
ngrok Basic Auth Traffic Policy after `127.0.0.2` failed to bind on target macOS.

**Bugfix**: 2026-08-16 — BUG-003 supersedes Traffic Policy Basic Auth after the real public stream
stalled and adds an ephemeral application-owned ingress activation to the process-local lifecycle.

**Bugfix**: 2026-08-16 — BUG-003 verification follow-up reconciles lifecycle transitions and
cardinality invariants with deny-before-withdraw and one endpoint/ingress ownership.

**Bugfix**: 2026-08-16 — BUG-003 test-ergonomics follow-up models the exact dev/test environment
override as ephemeral input, not a fifth persistent lifetime or an alternate production store.

**Bugfix**: 2026-08-16 — BUG-003 second verification reconciliation applies the effective secret
source and deny-before-withdraw order to publication and settings mutation.

## Model boundaries

Public-access data is split into four deliberately different lifetimes:

1. versioned non-secret preferences in Application Support;
2. two independent secrets in macOS Keychain;
3. one process-local lifecycle snapshot and active provider endpoint handle;
4. narrow ephemeral mutation/generated-password payloads that exist only for one trusted desktop
   call; the dev/test-only environment override is resolved within this same ephemeral lifetime.

Session JSON version 1, player-config JSON version 1, public player protobuf messages, and
authoritative game state do not reference any of these entities.

## PublicAccessPreferences

Protobuf source: `fallout.terminal.config.v1.PublicAccessPreferences`.

| Field | Type | Persistence | Validation / meaning |
|---|---|---|---|
| `version` | unsigned integer | `public-access.json` | Exactly `1`; unsupported values fail safe and never start an endpoint. |
| `enabled_preference` | boolean | `public-access.json` | Restores only the UI's public-access preference/presentation; it never triggers startup, endpoint acquisition, or URL restoration on application launch. |
| `reserved_domain` | optional string | `public-access.json` | Absent/blank requests a random provider URL; otherwise a normalized DNS host with no scheme, port, path, user info, query, or fragment. |
| `username` | string | `public-access.json` | Trimmed, non-empty, no CR/LF; defaults to exact `players`. |
| `provider_token_present_hint` | boolean | `public-access.json` | Non-authoritative hint only; reconciled with Keychain metadata before use. |
| `player_password_present_hint` | boolean | `public-access.json` | Non-authoritative hint only; reconciled with Keychain metadata before use. |
| `revision` | unsigned 64-bit integer | `public-access.json` | Monotonically increases after each committed settings/secret mutation. |

The JSON adapter uses established camel-case application JSON names and an explicit version adapter;
it does not use generic ProtoJSON. The file lives at
`~/Library/Application Support/com.vaulttec.fallout-terminal/public-access.json`, with a `0700`
parent and `0600` file/temp/quarantine modes.

### Safe defaults and corruption recovery

Safe defaults are version 1, `enabled_preference=false`, no domain, username `players`, both
presence hints false, and revision 0. A missing file returns these defaults. Malformed, unsupported,
or semantically invalid content is moved to one private quarantine file, returns safe defaults plus
a redacted recovery error, and leaves runtime state `disabled`. Saving uses same-directory exclusive
temp creation, file sync, atomic rename, and cleanup on every failed path.

## SecretRef and SecretStore state

`SecretRef` is a closed internal enum, never a user-provided Keychain selector:

| Ref | Purpose | Keychain account |
|---|---|---|
| `ProviderAccountToken` | Authenticates the user's ngrok Agent session. | `ngrok-authtoken` |
| `PlayerBasicAuthPassword` | ~~BUG-001 authenticated requests through the active endpoint Traffic Policy.~~ **BUG-003** Authenticates exact-Host requests at the private application ingress. | `player-basic-auth-password` |

Both use the service selected by bundle profile. The secret value is not a field of any settings,
status, event, public descriptor, persisted model, or reusable result.

`SecretPresence` is `unspecified`, `absent`, `present`, or `unknown`. `unknown` is required when
Keychain is locked, access is denied, or the store is unavailable; it must not be flattened to
`absent` or treated as permission to overwrite.

The `SecretStore` supports only:

- metadata-only presence query;
- replace/add a caller-supplied byte buffer;
- idempotent deletion;
- trusted scoped use of one or both secret refs through a callback whose buffers are cleared after
  the call.

There is no general `Get`, string-return, export, reveal, list, or desktop-facing read operation.

### DevelopmentTestPublicAccessOverride

This process-local adapter exists only in canonical development/test composition. It reads four
exact names without enumerating or logging environment contents:

| Environment name | Effective field | Exposure rule |
|---|---|---|
| `FALLOUT_NGROK_AUTHTOKEN` | provider account token | Non-empty value overrides Keychain use for this process; reusable UI sees presence only. |
| `FALLOUT_NGROK_RESERVED_DOMAIN` | reserved domain | Non-empty validated value overrides persisted preference and may prefill the form. |
| `FALLOUT_PUBLIC_TEST_USERNAME` | player username | Non-empty validated value overrides persisted preference and may prefill the form. |
| `FALLOUT_PUBLIC_TEST_PASSWORD` | player Basic Auth password | Non-empty value overrides Keychain use for this process; reusable UI sees presence only. |

Resolution is per field: non-empty environment value first, otherwise the ordinary persisted or
Keychain source. Environment-derived secrets are passed only through the same scoped callback shape
used by `SecretStore`; they are never written into Keychain or seeded into a Save secret mutation.
The adapter itself performs no persistence; an explicit Save retains ordinary semantics for visible
non-secret domain/username fields. The effective non-secret form values
and secret presence may appear in the existing secret-free snapshot, but no new DTO, protobuf field,
desktop method, status/event payload, serialization, or persistent entity is added. Loading the
override does not save settings or start public access. Packaged production does not construct this
adapter and ignores all four names.

## PublicAccessStatus

Protobuf source: `fallout.terminal.private.v1.PublicAccessStatus`.

| Field | Type | Rule |
|---|---|---|
| `state` | `PublicAccessLifecycleState` | One of `disabled`, `starting`, `ready`, `stopping`, `failed`; zero is `UNSPECIFIED`. |
| `generation` | unsigned 64-bit integer | Increases for every start, stop, reconfigure, failure cleanup, or shutdown intent. |
| `settings_revision` | unsigned 64-bit integer | Revision whose settings the operation uses. |
| `public_url` | optional string | Present only in `ready`, only after endpoint validation and exact-Host/auth ingress activation. |
| `error_category` | enum | Redacted stable category; zero when there is no failure. |
| `error_message` | optional string | Safe corrective text, never a raw provider/Keychain error. |

`PublicAccessSnapshot` combines effective redacted preferences, reconciled secret presence, and
status. In development/test composition only, effective domain/username and presence may reflect
the ephemeral override; persisted values remain unchanged. It is
safe for the private master bridge and `public-access-status` named event. It never contains the
active password, provider token, Keychain data, provider account details, or internal endpoint ID.

## Lifecycle state machine

The state machine is serialized for intent changes but never holds its mutex over Keychain,
provider/network, event, or endpoint-close calls. Each asynchronous result carries both generation
and settings revision.

| Current | Trigger | Immediate protected action | Completion |
|---|---|---|---|
| `disabled` | Start | Increment generation; enter `starting`; start one deny-all private ingress without a published URL. | Current success validates the endpoint, atomically activates exact Host/auth, then enters `ready`; current failure denies and closes owned public resources before `failed`. |
| `failed` | Start | Clear redacted error, increment generation, enter `starting`. | Same as disabled Start. |
| `starting` | Start again | Join/return the same current intent; do not create another endpoint. | Existing operation decides result. |
| `ready` | Start with same revision | Idempotently return current snapshot. | No provider action. |
| `starting` or `ready` | Stop | Increment generation, deny ingress, withdraw URL, cancel startup/monitor, enter `stopping`. | Close endpoint/agent and ingress; enter `disabled`, URL absent. |
| `stopping` | Stop again | Join the same stop under its existing deadline. | Same terminal result; no extended budget. |
| any active/transition state | Commit changed settings | Increment generation, deny ingress, withdraw URL, cancel and close old endpoint/ingress before applying/restarting. | If it was active, start only one replacement generation; otherwise remain `disabled`. |
| `ready` | Endpoint `Done`/disconnect | If generation is current, deny ingress and clear URL immediately. | Bounded endpoint/ingress close; enter `failed`; local/LAN remains ready. |
| any | Quit/Cmd+Q | Increment generation, deny ingress, withdraw URL, cancel and close endpoint/ingress. | Continue player/session/desktop cleanup within the shared five-second deadline. |

A completion is stale if either generation or settings revision differs from the current intent, or
the expected state is no longer `starting`. A stale success closes its acquired endpoint without
publishing a URL. A stale failure cannot overwrite current status.

### Publication invariant

For every generation the only legal readiness order is:

1. an owned private loopback ingress starts in deny-all mode;
2. the provider creates an endpoint targeting that ingress without player credentials or Traffic
   Policy and no UI URL is exposed;
3. the returned URL passes strict validation;
4. ~~scoped Keychain values~~ **BUG-003 reconciliation** scoped effective secret values—production
   Keychain or the exact FR-056 dev/test override—atomically activate exact Host plus Basic Auth at
   the ingress;
5. state changes to `ready` and only then may snapshot/events expose `public_url`.

Stop, reconfigure, provider failure, and quit deny ingress admission before withdrawing the URL and
closing endpoint/ingress. Direct local/LAN behavior is outside this endpoint lifecycle and is never
changed by public failure.

## ~~PublicAccessGrant~~ Superseded by BUG-001

`PublicAccessGrant` was a process-local player-boundary value. BUG-001 removes it from active
production because the player server no longer owns public authentication.

| Field | Type | Rule |
|---|---|---|
| `generation` | unsigned 64-bit integer | Monotonic; an older activation/deactivation is rejected. |
| `external_host` | normalized authority | Exact validated Host from the ready HTTPS URL. |
| `username` | byte buffer | Compared in constant time; cleared with the grant. |
| `password` | byte buffer | Compared in constant time; cleared with the grant. |

The player policy mutex is held only while classifying Host and comparing Basic Auth headers. It is
released before static or ConnectRPC handlers run, so a long-lived stream holds no policy lock.
Deactivation takes the write lock, swaps to deny, and clears credential buffers after outstanding
header comparisons finish.

## ~~PublicIngressRoute / ProviderEndpointInput (BUG-001)~~ PublicIngressActivation (BUG-003)

The old source-bound `PublicIngressRoute` and SDK credential-bearing `ProviderEndpointInput` are
superseded. `PublicIngressActivation` is ephemeral process-local ingress input, not protobuf, JSON,
status, or reusable provider/user configuration.

| Field | Value | Rule |
|---|---|---|
| `exact_public_host` | validated host | Installed atomically only after endpoint URL validation. |
| `username` | scoped byte buffer | Used only by ingress Basic Auth comparison. |
| `password` | scoped byte buffer | Used only by ingress Basic Auth comparison. |
| `player_upstream_url` | `http://127.0.0.1:3690` | The sole authoritative player listener. |

The ingress begins deny-all, atomically installs activation after endpoint URL validation, and never
renders credential or policy text into diagnostics. The ngrok adapter receives only its private
loopback upstream plus account token/domain. The existing player listener performs ordinary
application routing only. Local/LAN clients connect directly to it and therefore do not encounter
the ingress policy; public clients reach it only through the active ingress.

## Ephemeral private payloads

### SavePublicAccessSettingsRequest

Carries expected revision, non-secret proposed preferences, and two independent `oneof` secret
mutations: no change, replacement, or deletion. A replacement token/password exists only in the
Wails call argument, protobuf adapter, manager validation, and scoped `SecretStore` write. It is
never logged, included in an error, serialized, emitted, or retained in frontend module state.

Manual player passwords require at least eight characters and no composition rule. Username and
password are validated independently. Requests that specify both replace and delete for one secret
are structurally impossible through the `oneof` contract.

### GeneratedPlayerPasswordResult

`GeneratePlayerPassword` creates at least 128 bits of entropy using the OS cryptographic random
source, stores the password in Keychain during the same trusted operation, and returns the new value
exactly once in `GeneratedPlayerPasswordResult`. The reusable snapshot/event emitted for the same
revision contains only password presence. The frontend may place the direct value only in its
one-time presentation and Copy closure; copy or dismissal clears the DOM value, closure, input, and
result reference. It is never placed in local/session storage, URL state, diagnostics, or a named
event. Failure before durable Keychain replacement returns no password.

## Settings mutation failure model

There is no false claim of a cross-Keychain/filesystem transaction. Before a mutation of active
configuration, ~~public acceptance is disabled and the old endpoint is closed~~ ~~**BUG-001** the URL
is withdrawn and the old protected endpoint is closed.~~ **BUG-003 reconciliation** the ingress is
set deny-all before URL withdrawal, then the old endpoint and ingress are closed. Keychain changes
and the atomic non-secret file write are individually durable. If a later step fails, the manager stays non-public in
`failed`, re-queries actual Keychain presence, reports a redacted partial-update recovery message,
and lets the user retry. It never restarts using a mixture that was not validated as one current
settings revision.

## Relationships and invariants

- One `PublicAccessPreferences` record references zero or one item of each fixed `SecretRef` only by
  presence hint; it never carries Keychain data.
- One `PublicAccessStatus` describes at most one owned `TunnelEndpoint` and one owned private
  ingress in the same generation.
- ~~One ready endpoint maps to exactly one active `PublicAccessGrant` and the existing player server
  at `http://127.0.0.1:3690`.~~ ~~**BUG-001**: One ready status maps to exactly one owned,
  policy-protected `TunnelEndpoint` forwarding directly to the existing player server.~~
  **BUG-003**: One ready status maps to one endpoint targeting one active ingress, which alone
  streams to the existing player server.
- ~~Every SDK upstream connection uses the one `PublicIngressRoute`; no alternate/default dialer
  path can reach ready.~~ ~~**BUG-001**: Every production endpoint uses direct upstream
  `http://127.0.0.1:3690`.~~ **BUG-003**: Every production endpoint targets only its owned private
  ingress; only that ingress targets the exact player upstream, with no alternate path to ready.
- There is never more than one production endpoint or provider runtime.
- ~~`public_url` implies `state=ready` and an active matching policy generation.~~ **BUG-001**:
  ~~`public_url` implies `state=ready` and one active matching protected endpoint generation.~~
  **BUG-003**: `public_url` implies `state=ready`, one matching endpoint, and active exact-Host/auth
  ingress policy in the same generation.
- ~~`state!=ready` implies no published URL and no external Host acceptance.~~ **BUG-001**:
  ~~`state!=ready` implies no published URL.~~ **BUG-003**: `state!=ready` implies no published URL
  and deny-all ingress admission.
- Provider or secure-store failure never changes local player, broadcast, role, session, or
  player-config state.
