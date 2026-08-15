# Data Model: Embedded ngrok Public Access

**Bugfix**: 2026-08-15 — ANALYZE-S1 adds a non-serializable trusted ingress route so local/LAN
bypass never depends on forwarded `Host` behavior.

**Bugfix**: 2026-08-15 — BUG-001 supersedes the player-bound source/Host grant with an ephemeral
ngrok Basic Auth Traffic Policy after `127.0.0.2` failed to bind on target macOS.

## Model boundaries

Public-access data is split into four deliberately different lifetimes:

1. versioned non-secret preferences in Application Support;
2. two independent secrets in macOS Keychain;
3. one process-local lifecycle snapshot and active provider endpoint handle;
4. narrow ephemeral mutation/generated-password payloads that exist only for one trusted desktop
   call.

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
| `PlayerBasicAuthPassword` | ~~Authenticates requests for the active public Host.~~ **BUG-001** Authenticates requests through the active public endpoint. | `player-basic-auth-password` |

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

## PublicAccessStatus

Protobuf source: `fallout.terminal.private.v1.PublicAccessStatus`.

| Field | Type | Rule |
|---|---|---|
| `state` | `PublicAccessLifecycleState` | One of `disabled`, `starting`, `ready`, `stopping`, `failed`; zero is `UNSPECIFIED`. |
| `generation` | unsigned 64-bit integer | Increases for every start, stop, reconfigure, failure cleanup, or shutdown intent. |
| `settings_revision` | unsigned 64-bit integer | Revision whose settings the operation uses. |
| `public_url` | optional string | Present only in `ready`, only after protected endpoint creation. |
| `error_category` | enum | Redacted stable category; zero when there is no failure. |
| `error_message` | optional string | Safe corrective text, never a raw provider/Keychain error. |

`PublicAccessSnapshot` combines redacted preferences, reconciled secret presence, and status. It is
safe for the private master bridge and `public-access-status` named event. It never contains the
active password, provider token, Keychain data, provider account details, or internal endpoint ID.

## Lifecycle state machine

The state machine is serialized for intent changes but never holds its mutex over Keychain,
provider/network, event, or endpoint-close calls. Each asynchronous result carries both generation
and settings revision.

| Current | Trigger | Immediate protected action | Completion |
|---|---|---|---|
| `disabled` | Start | Increment generation; enter `starting` without a published URL. | Current success validates an already protected endpoint, then enters `ready`; current failure enters `failed`. |
| `failed` | Start | Clear redacted error, increment generation, enter `starting`. | Same as disabled Start. |
| `starting` | Start again | Join/return the same current intent; do not create another endpoint. | Existing operation decides result. |
| `ready` | Start with same revision | Idempotently return current snapshot. | No provider action. |
| `starting` or `ready` | Stop | Increment generation, withdraw URL, cancel startup/monitor, enter `stopping`. | Close endpoint/agent; enter `disabled`, URL absent. |
| `stopping` | Stop again | Join the same stop under its existing deadline. | Same terminal result; no extended budget. |
| any active/transition state | Commit changed settings | Increment generation, withdraw URL, cancel and close old endpoint before applying/restarting. | If it was active, start only one replacement generation; otherwise remain `disabled`. |
| `ready` | Endpoint `Done`/disconnect | If generation is current, clear URL immediately. | Bounded close/disconnect; enter `failed`; local/LAN remains ready. |
| any | Quit/Cmd+Q | Increment generation, withdraw URL, cancel and close endpoint. | Continue player/session/desktop cleanup within the shared five-second deadline. |

A completion is stale if either generation or settings revision differs from the current intent, or
the expected state is no longer `starting`. A stale success closes its acquired endpoint without
publishing a URL. A stale failure cannot overwrite current status.

### Publication invariant

For every generation the only legal readiness order is:

1. scoped Keychain values are used to construct the ngrok Basic Auth Traffic Policy in memory;
2. the provider creates the endpoint with that policy and no UI URL is exposed;
3. the returned URL passes strict validation;
4. state changes to `ready` and only then may snapshot/events expose `public_url`.

Stop, reconfigure, provider failure, and quit withdraw the URL before closing the endpoint. Direct
local/LAN behavior is outside this endpoint lifecycle and is never changed by public failure.

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

## ~~PublicIngressRoute~~ ProviderEndpointInput (BUG-001)

The old source-bound `PublicIngressRoute` is superseded. `ProviderEndpointInput` is ephemeral
process-local adapter input, not protobuf, JSON, status, or reusable provider/user configuration.

| Field | Value | Rule |
|---|---|---|
| `upstream_url` | `http://127.0.0.1:3690` | The one authoritative player listener. |
| `username` | scoped byte buffer | Used only to construct the endpoint Basic Auth policy. |
| `password` | scoped byte buffer | Used only to construct the endpoint Basic Auth policy. |

The adapter attaches the Basic Auth policy while creating the endpoint, clears/drops its scoped
input after construction, and never renders policy text into diagnostics. The existing player
listener performs ordinary application routing only. Local/LAN clients connect directly to it and
therefore do not encounter the ngrok policy; public clients use the ngrok URL and do.

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
configuration, ~~public acceptance is disabled and the old endpoint is closed~~ **BUG-001** the URL
is withdrawn and the old protected endpoint is closed. Keychain changes and the atomic non-secret
file write are individually durable. If a later step fails, the manager stays non-public in
`failed`, re-queries actual Keychain presence, reports a redacted partial-update recovery message,
and lets the user retry. It never restarts using a mixture that was not validated as one current
settings revision.

## Relationships and invariants

- One `PublicAccessPreferences` record references zero or one item of each fixed `SecretRef` only by
  presence hint; it never carries Keychain data.
- One `PublicAccessStatus` describes at most one owned `TunnelEndpoint`.
- ~~One ready endpoint maps to exactly one active `PublicAccessGrant` and the existing player server
  at `http://127.0.0.1:3690`.~~ **BUG-001**: One ready status maps to exactly one owned,
  policy-protected `TunnelEndpoint` forwarding to the existing player server.
- ~~Every SDK upstream connection uses the one `PublicIngressRoute`; no alternate/default dialer
  path can reach ready.~~ **BUG-001**: Every production endpoint uses the exact upstream
  `http://127.0.0.1:3690`; no custom dialer or alternate upstream can reach ready.
- There is never more than one production endpoint or provider runtime.
- ~~`public_url` implies `state=ready` and an active matching policy generation.~~ **BUG-001**:
  `public_url` implies `state=ready` and one active matching protected endpoint generation.
- ~~`state!=ready` implies no published URL and no external Host acceptance.~~ **BUG-001**:
  `state!=ready` implies no published URL.
- Provider or secure-store failure never changes local player, broadcast, role, session, or
  player-config state.
