# Data Model: Player Intelligence and Hacker Perk Management

## CharacterRosterEntry

The durable authored player profile remains the canonical roster entry owned by `internal/domain`.

| Field | Type | Rules |
|---|---|---|
| `ID` | stable `CharacterID` | Nonblank, unique within one player configuration, retained by update, generated only after mutation preconditions pass. |
| `Name` | string | Trimmed, nonblank, at most 80 Unicode code points; duplicate visible names remain allowed. |
| `Intelligence` | integer | Canonical values are 1 through 10 inclusive. Zero is never a valid canonical value. |
| `HackerPerkAvailable` | boolean | Canonical yes/no availability; it does not grant an automatic gameplay effect. |

The ordered roster slice remains the source of presentation order. Updating a profile changes only Name, Intelligence, and Hacker perk availability. It does not replace the stable ID or move the entry.

## PlayerConfig

```text
PlayerConfig
├── Version: 1
├── Name: nonblank configuration name
└── Roster: ordered CharacterRosterEntry[]
```

Validation remains all-or-nothing:

- version must be exactly 1;
- roster must be an array and remain within the established roster-size limit;
- every ID must be nonblank and unique;
- every name must satisfy the established character-name rules;
- every Intelligence value must be 1–10;
- every canonical entry always has a boolean Hacker perk value.

Player-config JSON remains strict. Unknown fields, trailing values, wrong types, null attributes, and explicit invalid Intelligence values reject the entire file. Only missing new attributes on a legacy roster entry receive compatibility defaults during decode.

## LegacyPlayerConfigEntry

This is a representation-only decode shape, not a canonical domain entity.

| JSON member | Presence behavior |
|---|---|
| `id` | Required under existing validation. |
| `name` | Required under existing validation. |
| `intelligence` | Missing becomes 1; present value must be an integer from 1 through 10; null is invalid. |
| `hackerPerkAvailable` | Missing becomes false; present value must be boolean; null is invalid. |

After normalization, the ordinary canonical validator runs. Encoding never emits the legacy shape: both attributes are written explicitly.

## PlayerConfigHandle

The coordinator's active durable configuration handle gains one private concurrency field.

| Field | Rules |
|---|---|
| Path | Absolute trusted file path; never sent to player clients. |
| Version | Exactly 1. |
| Name | Detached configuration metadata. |
| ContentDigest | SHA-256 of the exact bytes most recently loaded or saved; private to the Go persistence/control boundary. |

The digest is not part of player-config JSON, protobuf persistence, public state, or the master browser projection. A successful save returns a handle with the digest of the new canonical bytes. A missing, unreadable, or digest-mismatched target rejects the save and retains the old handle and runtime roster.

## MasterRosterEntry

The detached private game-master projection contains:

- stable player ID;
- display name;
- Intelligence;
- Hacker perk availability;
- optional current logical-session claimant.

The projection is delivered by the existing coordination bootstrap/result/event path. It never contains the content digest. Public player projections continue to include only their established ID, name, and availability/assignment data.

## Roster Mutation Inputs

### CharacterCreatePayload

| Field | Rules |
|---|---|
| Name | Required and validated before the coordinator call. |
| Intelligence | Required whole number 1–10. Missing maps to invalid zero at the trusted boundary. |
| HackerPerkAvailable | Required presence; explicit false is valid. |
| ExpectedRevision | Must equal the current coordination revision inside the commit lock. |

### CharacterUpdatePayload

Contains the same complete profile and expected revision plus a required stable CharacterID. A successful update is one persistence operation and one canonical transition.

### CharacterDeletePayload

Contains a required stable CharacterID and ExpectedRevision. Deletion is allowed only while no broadcast exists.

## State Transitions

| Event | Preconditions | Result |
|---|---|---|
| Load legacy config | Valid v1 document; missing only the new fields | Normalize to Intelligence 1/Hacker false, calculate digest, install complete canonical roster. |
| Load canonical config | Valid v1 document with both attributes | Preserve exact values/order/IDs, calculate digest, install complete roster. |
| Add player | Active config; no broadcast; expected revision matches; complete valid profile; file digest matches | Allocate stable ID, save one complete candidate, update digest, commit one revision, publish one master snapshot. |
| Update player | Same guards; target ID exists | Replace three editable profile values, preserve ID/order, save once, commit one revision, publish once. |
| Delete player | Same guards; target ID exists | Remove target, preserve survivor order, save once, commit one revision, publish once. |
| Exact retry/stale dialog | Expected revision no longer matches | Reject with current authoritative snapshot; no ID, write, revision, or effect. |
| Broadcast starts before mutation | Broadcast is non-nil at commit time | Reject with current snapshot; dialog becomes read-only from the new event. |
| File deleted/replaced/unreadable | Current bytes cannot be read or digest differs | Reject and require reopen/reselection; runtime roster and revision stay unchanged. |
| Atomic storage failure | Candidate validated but replacement fails | Retain old file, digest, canonical roster, revision, and effects. |
| Broadcast ends | Existing broadcast lifecycle completes | Detailed dialog becomes editable again using the latest unchanged roster/revision. |

## Invariants

- The file and canonical runtime never publish different accepted roster revisions.
- No roster mutation is accepted during a broadcast, even through a stale or crafted private call.
- Expected revision is checked before ID generation and persistence, making an exact retried add unable to create a second player.
- Private master attributes do not enter public player schemas, player streams, session JSON, logical-session state, assignments, or hacking state.
- Clone and projection paths copy every new scalar field and never expose mutable canonical references.

