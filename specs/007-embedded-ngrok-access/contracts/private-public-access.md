# Private Desktop Contract: Public Access

**Bugfix**: 2026-08-15 — BUG-001 replaces player-policy activation/deactivation with protected
endpoint publication, URL withdrawal, and endpoint/Agent close sequencing.

## Contract source and isolation

Add `proto/fallout/terminal/private/v1/public_access.proto`. It may import the non-secret
`fallout.terminal.config.v1.PublicAccessPreferences` message, but neither file may import public
player schemas for credentials or expose provider/Keychain implementation types. The existing
`runtime.proto` and its frozen Wails-v3 digest remain unchanged; `server-info` keeps its current safe
URL compatibility role.

All operations are trusted master-only Wails methods on the existing `desktopService`. There is no
public Connect route, generic dispatcher, player callable operation, or lifecycle method binding.

## Protobuf messages

### Enums

`PublicAccessLifecycleState` values:

- `PUBLIC_ACCESS_LIFECYCLE_STATE_UNSPECIFIED = 0`
- `PUBLIC_ACCESS_LIFECYCLE_STATE_DISABLED = 1`
- `PUBLIC_ACCESS_LIFECYCLE_STATE_STARTING = 2`
- `PUBLIC_ACCESS_LIFECYCLE_STATE_READY = 3`
- `PUBLIC_ACCESS_LIFECYCLE_STATE_STOPPING = 4`
- `PUBLIC_ACCESS_LIFECYCLE_STATE_FAILED = 5`

The master UI maps these internal contract values to the specification's observable labels exactly:
`DISABLED → stopped`, `STARTING → starting`, `READY → ready`, `STOPPING → stopping`, and
`FAILED → error`. `UNSPECIFIED` is a loading/contract-error condition and is never rendered as a
ready or stopped result.

`SecretPresence` values:

- `SECRET_PRESENCE_UNSPECIFIED = 0`
- `SECRET_PRESENCE_ABSENT = 1`
- `SECRET_PRESENCE_PRESENT = 2`
- `SECRET_PRESENCE_UNKNOWN = 3`

`PublicAccessErrorCategory` values:

- `PUBLIC_ACCESS_ERROR_CATEGORY_UNSPECIFIED = 0`
- `PUBLIC_ACCESS_ERROR_CATEGORY_VALIDATION = 1`
- `PUBLIC_ACCESS_ERROR_CATEGORY_SETTINGS_CORRUPT = 2`
- `PUBLIC_ACCESS_ERROR_CATEGORY_SECRET_STORE_LOCKED = 3`
- `PUBLIC_ACCESS_ERROR_CATEGORY_SECRET_STORE_DENIED = 4`
- `PUBLIC_ACCESS_ERROR_CATEGORY_SECRET_STORE_UNAVAILABLE = 5`
- `PUBLIC_ACCESS_ERROR_CATEGORY_CREDENTIAL_MISSING = 6`
- `PUBLIC_ACCESS_ERROR_CATEGORY_PROVIDER_AUTHENTICATION = 7`
- `PUBLIC_ACCESS_ERROR_CATEGORY_DOMAIN_UNAVAILABLE = 8`
- `PUBLIC_ACCESS_ERROR_CATEGORY_NETWORK_UNAVAILABLE = 9`
- `PUBLIC_ACCESS_ERROR_CATEGORY_TIMEOUT = 10`
- `PUBLIC_ACCESS_ERROR_CATEGORY_PROVIDER_FAILURE = 11`
- `PUBLIC_ACCESS_ERROR_CATEGORY_SHUTDOWN_TIMEOUT = 12`
- `PUBLIC_ACCESS_ERROR_CATEGORY_CONFLICT = 13`

Categories are stable, redacted UI semantics. They are not passthrough ngrok or OS status codes.

### Reusable secret-free messages

`PublicAccessStatus`:

| Field | Number | Type |
|---|---:|---|
| `state` | 1 | `PublicAccessLifecycleState` |
| `generation` | 2 | `uint64` |
| `settings_revision` | 3 | `uint64` |
| `public_url` | 4 | `optional string` |
| `error_category` | 5 | `PublicAccessErrorCategory` |
| `error_message` | 6 | `optional string` |

`PublicAccessSnapshot`:

| Field | Number | Type |
|---|---:|---|
| `preferences` | 1 | `fallout.terminal.config.v1.PublicAccessPreferences` |
| `provider_token_presence` | 2 | `SecretPresence` |
| `player_password_presence` | 3 | `SecretPresence` |
| `status` | 4 | `PublicAccessStatus` |

`GetPublicAccessRequest` is empty. `PublicAccessStatusEvent` contains one
`PublicAccessSnapshot snapshot = 1`. `PublicAccessCommandResult` contains `bool ok = 1`,
`optional string error = 2`, and `PublicAccessSnapshot snapshot = 3`. Every error is redacted.

### Narrow secret mutation input

`SavePublicAccessSettingsRequest` contains:

| Field | Number | Type / rule |
|---|---:|---|
| `expected_revision` | 1 | `uint64`; rejects stale forms. |
| `enabled_preference` | 2 | `bool`; restores only the UI preference/presentation and never means auto-start. |
| `reserved_domain` | 3 | `optional string`; blank normalizes to absent. |
| `username` | 4 | `string`; non-empty, default UI value `players`. |
| `provider_token_change` | 5/6 | `oneof`: `string replacement_provider_token = 5` or `bool delete_provider_token = 6`. |
| `player_password_change` | 7/8 | `oneof`: `string replacement_player_password = 7` or `bool delete_player_password = 8`. |

This message is legal only as the direct argument of `SavePublicAccessSettings`. It is prohibited
from status, events, persistence, logs, diagnostics, fixtures, generic serializers, and player
descriptors. The bridge clears its native input fields after invocation and does not echo them in
the result.

`GeneratePlayerPasswordRequest` contains `uint64 expected_revision = 1`.

`GeneratedPlayerPasswordResult` contains:

| Field | Number | Type / rule |
|---|---:|---|
| `ok` | 1 | `bool` |
| `error` | 2 | `optional string`; redacted. |
| `generated_password` | 3 | `optional string`; present only on successful creation and Keychain storage, once. |
| `settings_revision` | 4 | `uint64` |

It must not contain a reusable snapshot because accidental caching of the result would also cache
the secret. A separate secret-free status event reports the new revision and presence.

`PublicAccessCommandRequest` contains `uint64 expected_revision = 1`; it is used by Start and Stop.

## Exact desktop surface

| Go method | Frontend facade | Input | Output |
|---|---|---|---|
| `GetPublicAccess` | `getPublicAccess()` | no native argument; semantic `GetPublicAccessRequest` | normalized secret-free `PublicAccessSnapshot` |
| `SavePublicAccessSettings` | `savePublicAccessSettings(request)` | native compatibility object mapped to `SavePublicAccessSettingsRequest` | `PublicAccessCommandResult` |
| `GeneratePlayerPassword` | `generatePlayerPassword(request)` | expected revision | one-time `GeneratedPlayerPasswordResult` |
| `StartPublicAccess` | `startPublicAccess(request)` | expected revision | `PublicAccessCommandResult` |
| `StopPublicAccess` | `stopPublicAccess(request)` | expected revision | `PublicAccessCommandResult` |

Add exact named event `public-access-status` and facade `onPublicAccessStatus(callback)`. Its native
payload maps only from `PublicAccessStatusEvent`. Subscription is established before the initial
`GetPublicAccess` snapshot; an event received first wins over a stale snapshot for the same or lower
generation/revision. Unsubscribe and hot-disposal follow the existing exact-once rules.

No operation returns a stored token or password. There is deliberately no `RevealSecret`,
`GetSecret`, `CopyCredentials`, environment, process, or provider-generic desktop method.

## Command behavior

### GetPublicAccess

Returns persisted non-secret preferences, reconciled presence values, and current status. A locked
or denied Keychain produces `UNKNOWN`, a redacted error category, no secret readback, and no public
start.

### SavePublicAccessSettings

Validates all non-secret and ephemeral secret inputs before mutation. If public access is starting
or ready, it first advances generation, ~~deactivates public policy~~ **BUG-001** withdraws the URL,
and closes the old endpoint/Agent. Then it applies Keychain mutations and atomically writes
non-secret settings. If the old runtime was active and the complete new revision is valid, it starts
one replacement; otherwise it remains disabled/failed. No old and new endpoint may accept
concurrently.

### GeneratePlayerPassword

Generates at least 128 bits of entropy, stores the new password, advances settings revision, and
returns it only in the direct result. If active, it follows the same protected stop/change/restart
sequence as another setting change. The named event and all later calls expose presence only.

### StartPublicAccess

Requires current revision plus present token/password. It is idempotent for the same current
starting/ready intent. It publishes no URL until ~~endpoint URL validation and exact-host policy
activation both succeed~~ **BUG-001** the policy-protected endpoint has been created and its URL has
passed validation.

### StopPublicAccess

Is idempotent and joins an existing stop. It ~~atomically disables public acceptance before endpoint
close~~ **BUG-001** withdraws the URL before closing the endpoint/Agent; endpoint close is the public
admission shutdown boundary. It clears the URL before reporting disabled. Repeated calls do not
extend the shared deadline.

## Frontend handling and Copy

The master settings section has explicit labels for reserved domain, username, token replacement,
and password replacement. Secret inputs use password controls with no Reveal action and are cleared
after the call. Presence is rendered as present/absent/unavailable, never as a masked reconstruction.

Copy controls are local UI behavior:

- ready public URL and non-secret username may be copied at any time;
- a manually entered password may be copied only from its current unsaved input;
- a generated password may be copied only from its one-time direct result;
- an already saved password cannot be included in a Copy action.

The one-time result is held only in a modal/presentation-local closure and DOM node, not reusable
frontend state or storage. Copy or dismissal clears the DOM value, closure reference, request/result
reference, and any secret input. Actions are disabled during transitions, status uses polite live
regions, errors use alert semantics, and keyboard focus enters/returns from the one-time dialog.

## Verification contract

- Descriptor tests enumerate every field and prove public player descriptors import none of these
  private/config messages.
- Adapter tests reject unhandled enum/oneof values and prove reusable snapshot/result/event JSON is
  secret-free.
- Wails allowlists, generated bindings, fixture exports, and exact named events are updated together.
- Canary values are scanned across errors, event capture, generated outputs, frontend bundle,
  Application Support, session/player-config fixtures, package resources, logs, and diagnostics.
- The generated password can appear only in the direct invocation result and transient one-time UI
  presentation; a dedicated test fails if it enters a named event or later snapshot.
