# Contract: Version-1 Session JSON

## Compatibility promise

The migration does not change the session version or JSON field names. A valid version-1 file written by the Electron release must open in the Wails release, and a file written by the Wails release must remain readable by the Electron rollback release when it contains only version-1 fields.

## Top-level shape

```json
{
  "version": 1,
  "name": "Campaign name",
  "terminals": []
}
```

Known fields use the rules in [../data-model.md](../data-model.md). Compatible unknown fields on known objects are preserved. Unknown node types, unsupported versions, invalid known field types, duplicate IDs, invalid root semantics, or configured size/depth limits are rejected.

## Terminal shape

```json
{
  "id": "t123",
  "name": "Терминал 1",
  "hackLevel": 0,
  "introText": "",
  "root": {
    "id": "root",
    "type": "folder",
    "name": "ROOT",
    "children": []
  }
}
```

## Node variants

```json
{ "id": "f1", "type": "folder", "name": "DOCS", "children": [] }
{ "id": "c1", "type": "command", "name": "STATUS", "text": "ONLINE" }
{ "id": "e1", "type": "entry", "name": "LOG", "description": "..." }
```

## macOS storage policy

- User-owned session files remain portable JSON documents, not application metadata.
- New and Save dialogs default to `~/Documents/Fallout Terminal/Sessions/`.
- The default directory is created only after the user confirms a create/save operation.
- The demo inside the `.app` is read-only and is copied only through an explicit user action and destination choice.
- App-managed metadata belongs in `~/Library/Application Support/com.vaulttec.fallout-terminal/` and is never mixed into session JSON.
- Autosave always targets the current explicitly selected session path; it never moves a file into Documents or Application Support.
- Canceling a copy/create/open dialog changes no active session, path, or filesystem content.

This replaces the Electron release's automatic executable-adjacent demo seeding
for packaged macOS builds. It is an intentional storage behavior change, not a
session-format change.

## Operations

### Create

- Native save dialog filtered to `*.json`, initially suggesting the macOS Documents default when no active path exists.
- Cancellation returns `{ok:false,canceled:true}` and changes no state.
- Derive session name from the final filename extension.
- Write the initial session before activating the path.

### Open

- Native single-file open dialog filtered to `*.json`.
- Parse UTF-8 JSON and validate the complete known schema.
- Failure returns a non-secret error and retains the previous active path/session.

### Save

- Requires an active path; never opens a dialog implicitly.
- Accepts a complete validated session and a monotonically increasing revision.
- Serializes with two-space indentation and a final newline.
- Writes a private same-directory temporary file and atomically replaces the target where supported.
- Returns the durable revision so stale completions cannot overwrite the UI status.

## Runtime exclusions

File path, selected editor terminal/node, expanded folders, live terminal, navigation, player connections, current puzzle, client count, tunnel state, and save revisions are not serialized.
