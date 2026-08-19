# Player-Config JSON Version 1 Contract

## Canonical JSON shape

The document version remains 1. Every newly saved roster entry contains both attributes explicitly:

```json
{
  "version": 1,
  "name": "vault-crew",
  "roster": [
    {
      "id": "player_mara",
      "name": "Mara",
      "intelligence": 8,
      "hackerPerkAvailable": true
    },
    {
      "id": "player_boone",
      "name": "Boone",
      "intelligence": 4,
      "hackerPerkAvailable": false
    }
  ]
}
```

| JSON path | Type | Validation |
|---|---|---|
| `roster[].intelligence` | integer | 1–10 inclusive; fraction, string, null, zero, and values outside the range are invalid. |
| `roster[].hackerPerkAvailable` | boolean | `true` or `false`; null and non-boolean values are invalid. |

Existing ID, name, roster-size, ordering, strict unknown-member, one-document, indentation, and final-newline rules remain unchanged.

## Legacy decode

A valid version-1 entry that lacks `intelligence` loads as Intelligence 1. A valid entry that lacks `hackerPerkAvailable` loads as Hacker unavailable. Either field may be independently absent; a supplied invalid value never receives a default.

Opening a legacy file does not write it immediately. Its first successful roster mutation replaces the complete file atomically and emits both fields for every surviving entry. Merely opening and closing the player-management dialog performs no save.

## Persistence protobuf

```proto
message RosterEntry {
  string id = 1;
  string name = 2;
  optional int32 intelligence = 3;
  optional bool hacker_perk_available = 4;
}
```

Field numbers 1 and 2 and their names remain unchanged. Fields 3 and 4 are additive. Domain-to-protobuf mapping always sets both optional fields. Protobuf-to-domain mapping applies legacy defaults only when presence is absent; present invalid Intelligence is rejected by canonical validation.

`PlayerConfig.version`, `name`, and `roster` retain field numbers 1–3. The protobuf compatibility baseline remains the older breaking reference and is not rewritten for this additive change.

## Conditional atomic replacement

1. Load returns the canonical configuration plus SHA-256 of the exact source bytes.
2. A mutation encodes and validates the complete candidate before replacement.
3. The persistence service re-reads the target and requires its digest to equal the active handle's digest.
4. On mismatch, missing file, or read error, it returns a safe conflict and does not call atomic replacement.
5. On match, existing storage writes a private mode-0600 temporary file in the target directory, syncs it, and renames it over the target.
6. Success returns the digest of the canonical replacement bytes; only then may the coordinator install and publish the candidate.

This detects ordinary external deletion or replacement while the configuration is open. The application remains the intended single writer; no background watcher or cross-process editing protocol is introduced.

## Privacy and lifetime

Only stable ID, name, Intelligence, and Hacker perk availability are durable. Content digests, logical sessions, claims, controller identity, broadcast IDs, active terminal, revisions, and hacking state never enter this file. The file path and digest remain private and never enter player payloads.

## Rollback note

The pre-feature strict reader rejects the two new members. Rolling the application back after a new-version save therefore also requires restoring a pre-feature copy of that player-config file or deliberately removing the two additive members. Normal forward compatibility requires no manual migration.

