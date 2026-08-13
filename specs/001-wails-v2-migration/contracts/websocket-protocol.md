# Contract: Player WebSocket Protocol

> **SUPERSEDED HISTORICAL RECORD — NON-AUTHORITATIVE.** This completed
> feature-001 contract documents the removed pre-Connect implementation. The
> application exposes no supported WebSocket/JSON player route. The current
> public contract is
> [`specs/005-connectrpc-protobuf-migration/contracts/public-player.md`](../../005-connectrpc-protobuf-migration/contracts/public-player.md).

## Compatibility

Message names and JSON field names remain compatible with the bundled `client/client.js`. The protocol remains unversioned for this migration; adding protocol negotiation is separate feature work.

## Server-to-player messages

### `TERMINAL_LIVE`

```json
{
  "type": "TERMINAL_LIVE",
  "terminalId": "t1",
  "terminalName": "Terminal",
  "tree": { "id": "root", "type": "folder", "name": "ROOT", "children": [] },
  "hackLevel": 0,
  "introText": "",
  "hack": null,
  "nav": { "path": ["root"], "mode": "list", "viewEntryId": null, "commandNodeId": null }
}
```

Sent on set/restart and directly to a newly connected player when live state exists. `hack`, when present, is sanitized.

### `TERMINAL_UPDATE`

Fields: `type`, `tree`, `introText`, `nav`. Sent after content replacement and navigation revalidation. Does not replace identity or puzzle state.

### `TERMINAL_CLEAR`

Fields: `type` only. Sent after canonical live state becomes absent.

### `NAV_STATE`

Fields: `type`, `nav`. Sent after each syntactically eligible navigation request, including validated no-op requests retained for compatibility.

### `HACK_STATE`

Fields: `type`, `hack: PublicHackState`. Sent after each syntactically eligible hacking request/override. Never includes `secretWord` or `wordsById`.

## Player-to-server messages

### `NAV_ACTION`

```json
{ "type": "NAV_ACTION", "action": "enter|back|command|entry", "nodeId": "optional" }
```

Requires live state and string action. Targets must be direct children of the current folder and match the requested type. Back never escapes root.

### `HACK_GUESS`

```json
{ "type": "HACK_GUESS", "targetId": "word-id-or-column:index" }
```

Requires active unfinished puzzle and string target. Unknown, out-of-range, stale, and in-word filler references are ignored without mutation.

### `HACK_ADMIN`

```json
{ "type": "HACK_ADMIN" }
```

Requires active unfinished puzzle and applies at most once.

## General validation

- Text frames contain one JSON object within a configured read limit.
- Binary frames, malformed JSON, arrays, null, missing/non-string `type`, and unknown types are ignored or closed with an appropriate policy status without crashing the server.
- No player input directly supplies canonical state.
- State mutation and snapshot creation are synchronized; outbound writes are serialized per connection.
- A slow/closed client cannot block other clients or canonical mutation.

## Reconnection

The browser retains its current three-second reconnect behavior. On success it receives the current full live snapshot if one exists. Reconnection never creates or resets a puzzle.
