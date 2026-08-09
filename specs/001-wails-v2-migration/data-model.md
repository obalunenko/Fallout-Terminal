# Data Model: Wails v2 Runtime Migration

## Modeling principles

- Durable session data and runtime live data are different aggregate roots.
- JSON field names remain camelCase and version 1 remains the only accepted persisted version.
- Player-facing structures never contain private puzzle fields.
- Bound desktop inputs and WebSocket inputs are validated before mutation.
- Services return immutable snapshots across goroutine and frontend boundaries.

## Durable entities

### Session

| Field | Type | Rules |
|---|---|---|
| `version` | integer | Required; exactly `1` |
| `name` | string | Required; non-empty after trimming; bounded to 256 UTF-8 bytes |
| `terminals` | array of Terminal | Required; bounded to 1,000 terminals; zero remains readable for parity but the UI may show an empty state |
| extra fields | raw JSON map | Preserved on round trip; may not shadow known fields |

### Terminal

| Field | Type | Rules |
|---|---|---|
| `id` | string | Required; non-empty; unique within the session |
| `name` | string | Required; non-empty after trimming; bounded to 256 UTF-8 bytes |
| `hackLevel` | integer | Required/default `0`; range `0..5` |
| `introText` | string | Required/default empty; bounded to 64 KiB |
| `root` | FolderNode | Required; ID `root`, type `folder` |
| extra fields | raw JSON map | Preserved on round trip |

### ContentNode

Tagged union selected by `type`:

| Variant | Required fields | Rules |
|---|---|---|
| FolderNode | `id`, `type: "folder"`, `name`, `children[]` | Root is `root`; child IDs are unique per terminal; maximum depth 64; maximum total nodes 100,000 |
| CommandNode | `id`, `type: "command"`, `name`, `text` | Leaf; text bounded to 1 MiB |
| EntryNode | `id`, `type: "entry"`, `name`, `description` | Leaf; description bounded to 1 MiB |

Known fields must have the documented type. Unknown node types are rejected because navigation and rendering cannot interpret them. Compatible unknown fields on known variants are preserved.

### ActiveSession

Process-only persistence metadata:

| Field | Type | Meaning |
|---|---|---|
| `path` | absolute path or empty | Current explicit autosave target |
| `session` | Session or nil | Last accepted durable model |
| `requestedRevision` | unsigned integer | Latest save request accepted |
| `savedRevision` | unsigned integer | Latest revision durably replaced |
| `saveState` | idle/saving/saved/failed | User-visible completion state |

`ActiveSession` is never serialized into session JSON.

### SessionLocations (macOS process configuration)

| Field | Meaning |
|---|---|
| `documentsDefault` | `~/Documents/Fallout Terminal/Sessions/`; suggested for New/Save only and created after user confirmation |
| `bundledDemo` | Read-only embedded sample inside the `.app`; never an autosave target |
| `applicationSupport` | `~/Library/Application Support/com.vaulttec.fallout-terminal/`; app-managed metadata only |

An explicitly opened or created session path always takes precedence over these
locations. Choosing the demo creates a writable copy through a native save
dialog; cancellation changes no session or filesystem state.

## Runtime entities

### ApplicationRuntime

Process-only composition state shared by development and packaged launch paths:

| Field | Type | Meaning |
|---|---|---|
| `mode` | `development` or `packaged` | Selects asset delivery and diagnostic behavior, never service ownership |
| `phase` | lifecycle state | Current startup/readiness/shutdown phase |
| `serverInfo` | ServerInfo or nil | Present only after the player listener is acquired |
| `tunnelState` | TunnelState | Optional public-access lifecycle; local readiness does not depend on tunnel success |
| `startupError` | non-secret string or empty | Actionable failure safe for the game-master UI |

`ApplicationRuntime` is created once per process. Development mode is entered by
the repository-root `wails dev` command, which also owns the configured frontend
tooling. Packaged mode is entered by one application launch and never requires
Node, npm, Vite, Wails CLI, or a separately started player server. Both modes use
the same service acquisition and cleanup order.

### LiveState (private)

| Field | Type | Persistence |
|---|---|---|
| `terminalId` | string | memory only |
| `terminalName` | string | memory only |
| `tree` | FolderNode snapshot | memory only |
| `hackLevel` | integer `0..5` | memory only |
| `introText` | string | memory only |
| `nav` | NavState | memory only |
| `hack` | HackState or nil | memory only, private |

### NavState

| Field | Type | Rules |
|---|---|---|
| `path` | array of node IDs | Begins with `root`; every later ID is a direct folder child |
| `mode` | `list` or `entry` | `entry` requires a valid `viewEntryId` |
| `viewEntryId` | string or null | Direct entry child of current folder |
| `commandNodeId` | string or null | Direct command child of current folder |

### HackState (private)

Contains level, word length, maximum/remaining attempts, solved/failed/admin flags, log, two public columns, generated addresses, `secretWord`, and `wordsById`. `secretWord` and `wordsById` never enter a public model.

### PublicHackState

Contains only `level`, `wordLength`, `attemptsMax`, `attemptsLeft`, `solved`, `failed`, `log`, and public `columns`. It is generated from `HackState` after every accepted mutation.

### PublicLiveState

Contains `terminalId`, `terminalName`, `tree`, `hackLevel`, `introText`, `nav`, and `hack: PublicHackState|null`.

### PlayerConnection

| Field | Type | Meaning |
|---|---|---|
| `id` | opaque generated string | Internal connection identity |
| `socket` | WebSocket connection | Owned network resource |
| `send` | bounded message channel | Single-writer queue |
| `closed` | cancellation signal | Idempotent shutdown |

Connections are not persisted. Queue overflow closes the slow connection; it never blocks live-state mutation.

### ServerInfo

| Field | Type | Meaning |
|---|---|---|
| `ip` | string | Selected non-internal IPv4 or `localhost` |
| `port` | integer | Listening port, default 3690 |
| `url` | HTTP(S) URL | Address emphasized to players |
| `localUrl` | HTTP URL or empty | Retained local address when public mode succeeds |
| `tunnel` | boolean | Public URL active |
| `tunnelError` | string or empty | Non-secret actionable failure |

### TunnelConfig

Validated ephemeral values: enabled flag, executable path, endpoint domain, Basic Auth username/password, and startup timeout. Password is never placed in status, logs, or errors. On macOS the owned tunnel process is placed in a process group that can be terminated deterministically at shutdown.

### DistributionCandidate

| Field | Type | Meaning |
|---|---|---|
| `architecture` | `arm64` | Initial supported macOS architecture |
| `appPath` | path | Built `.app` validation candidate |
| `dmgPath` | path or empty | Distribution image after packaging |
| `signed` | boolean | Developer ID signature verified |
| `hardenedRuntime` | boolean | Hardened-runtime flag verified |
| `notarizationId` | string or empty | Non-secret submission identifier |
| `stapled` | boolean | Accepted notarization ticket attached |
| `sha256` | string | Recorded artifact digest |

### TunnelState

States: `disabled`, `preparing`, `starting`, `active`, `stopping`, `stopped`, `failed`.

Process-only fields include child handle, public URL, temporary directory cleanup function, bounded diagnostics, and cancellation context.

## Desktop result models

### SessionResult

`ok`, `canceled`, optional `error`, optional `filePath`, optional `session`.

### SaveResult

`ok`, optional `error`, `requestedRevision`, optional `savedRevision`.

### RuntimeStatus

Current `ServerInfo`, `clientCount`, public `hackState`, optional startup error, and save revision state. Returned on master startup so events emitted before listener registration cannot be lost.

## State transitions

### Application lifecycle

```text
constructed → starting-player-server → desktop-loading → ready-local
       └──────────── failure ─────────→ failed/error-visible → stopped
ready-local → starting-tunnel → ready-public
ready-local/ready-public → stopping-tunnel → stopping-player-server → stopped
```

The desktop may report readiness only after `serverInfo` exists and the master can
retrieve `RuntimeStatus`. Tunnel failure returns to `ready-local` with a non-secret
error. Every transition after `constructed` registers cleanup immediately after
acquiring a resource. Shutdown and partial-start unwinding are idempotent and use
reverse acquisition order.

### Live terminal

```text
none --set-live--> live(root nav, optional fresh puzzle)
live --update-live--> live(revalidated nav, same puzzle)
live --player action--> live(mutated nav or puzzle)
live --set-live--> live(reset nav, optional fresh puzzle)
live --clear/shutdown--> none
```

### Session save

```text
idle --request N--> queued/saving N
saving N --request N+1--> saving N, queued N+1
saving N --success--> saved N (then start newest queued revision)
saving N --failure--> failed N (newer queued revision remains eligible)
```

Only a successful create/open changes the active path. Save writes always target a captured active path and revision.
