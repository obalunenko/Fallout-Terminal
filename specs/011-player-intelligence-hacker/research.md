# Research: Player Intelligence and Hacker Perk Management

## 1. Evolve player-config JSON without a version bump

**Decision**: Keep `version: 1` and add `intelligence` plus `hackerPerkAvailable` to each roster entry. Decode through a strict presence-aware representation: missing legacy Intelligence becomes 1 and missing legacy Hacker availability becomes false, while supplied null, zero, fractional, wrong-typed, out-of-range, or unknown values remain invalid. Canonical saves always emit both fields.

**Rationale**: The attributes are additive authored data, and every existing roster entry has a safe specification-defined default. Presence must be resolved at the representation boundary so canonical domain validation never treats an explicitly invalid zero as a legacy omission. Retaining strict unknown-field rejection preserves the established player-config contract.

**Alternatives considered**: Version 2 was rejected because no incompatible shape is required. Defaulting every zero in domain validation was rejected because it would accept an explicitly invalid Intelligence value. Generic pointer fields on the canonical aggregate were rejected because absence is a legacy wire concern, not a valid runtime state.

## 2. Use additive protobuf fields with explicit legacy presence

**Decision**: Append optional Intelligence and Hacker availability fields at unused numbers 3 and 4 of persistence `RosterEntry`; map absent fields to the legacy defaults and always set presence when mapping canonical values back to protobuf. Add master-only scalar fields to private `CharacterState` at unused numbers 4 and 5.

**Rationale**: Optional persistence presence makes absent legacy data distinguishable from an explicitly invalid Intelligence zero and provides exact adapter tests. The private projection is always canonical, so its consumers need values rather than legacy presence. Existing field numbers and package versions remain stable.

**Alternatives considered**: Reusing or renumbering existing fields is prohibited. Exposing the fields through the public player schema was rejected because only the master needs them and the feature does not grant gameplay effects.

## 3. Reject external file replacement with a content fingerprint

**Decision**: Record a SHA-256 digest of the exact player-config bytes when a file is loaded or successfully saved. Before a roster replacement, read the current target and require the same digest; a missing, unreadable, or changed file returns a conflict and requires reopen/reselection. After the check, reuse the existing same-directory private temporary file, fsync, and atomic rename.

**Rationale**: The current unconditional atomic replacement could recreate a deleted file or overwrite an externally replaced configuration while the dialog is open. A digest is deterministic, needs no dependency, detects content changes independent of timestamps, and lets runtime state stay aligned with the file that was actually loaded.

**Alternatives considered**: Modification time and size can miss replacements. A background file watcher adds lifecycle and race complexity without improving the explicit save boundary. Cross-process advisory locking was rejected because the application owns an explicit single-writer workflow; the digest check provides the required stale-file refusal without adding platform-specific locking.

## 4. Make every roster mutation revision-conditional and broadcast-safe

**Decision**: Add expected coordination revision to add, update, and delete inputs. Inside the coordinator commit lock, reject a mismatched revision, missing active config, or non-nil broadcast before ID allocation or persistence. Accepted mutations save one complete candidate and then advance the canonical revision once.

**Rationale**: The same guard handles double submissions, a configuration replaced while the dialog was open, and a broadcast that starts between render and click. The coordinator—not disabled controls—is the final authority. Persistence-before-commit already gives the required atomic failure behavior and should be extended rather than replaced.

**Alternatives considered**: UI disabling alone cannot stop crafted or stale calls. A separate mutation replay cache duplicates the existing revision order. Allowing live renames while blocking only deletion contradicts the approved inactive-session requirement.

## 5. Submit one complete profile per update

**Decision**: Add uses a structured create payload, and update sends stable player ID, name, Intelligence, Hacker availability, and expected revision as one mutation. Replace the exposed Wails `RenameCharacter` method with `UpdateCharacter`, while extending and reusing the existing compatible private protobuf request container rather than deleting its published type. Delete remains a separate explicit operation.

**Rationale**: One candidate prevents partial name/attribute persistence and reports one authoritative outcome. Reusing the existing message preserves schema compatibility, while the clearer Wails method tells frontend consumers that the whole profile is updated. Regenerated bindings remove the old runtime method in the same bundled cutover.

**Alternatives considered**: Independent rename, Intelligence, and Hacker mutations could partially succeed and create extra revisions. Removing or renaming the published protobuf message would fail the established breaking-change gate. Keeping both Wails runtime methods would create an unnecessary dual mutation path.

## 6. Use an in-page modal and separate durable editing from live assignment

**Decision**: Implement the requested pop-up as a native HTML `dialog` inside the existing Wails master window. Move detailed durable roster display and add/update/delete controls into it; keep live assignment, release, transfer, and controller operations in logical-session management. During a broadcast, the dialog remains openable but its durable controls are read-only.

**Rationale**: The repository already has accessible, terminal-styled dialog lifecycles and exactly one governed master window. This preserves the existing narrow event/state flow and avoids a new native window, service registration, or routing lifecycle. Separating durable roster content from broadcast-scoped claims prevents the new read-only rule from removing existing game-master correction capability.

**Alternatives considered**: A second Wails window would expand native composition and event routing for no business benefit. Keeping the full editor inline contradicts the requested focused pop-up. Disabling assignment correction during a broadcast would regress established behavior outside this feature's scope.

## 7. Keep the attributes private and dependency-free

**Decision**: Store and display the attributes only in the player-config/domain model and master-only projection. Do not alter public player RPCs, character-selection payloads, hacking rules, or dependency manifests. Use the existing pinned protobuf, Wails-binding, Go, Vite, and Playwright toolchains.

**Rationale**: The approved specification explicitly scopes the feature to storage and master management. Existing infrastructure already supplies atomic files, authoritative events, modal UI, generated contracts, and deterministic tests.

**Alternatives considered**: Publishing attributes to player browsers increases disclosure and contract surface without a consumer. Applying automatic hacking effects would introduce unspecified gameplay semantics. New state or UI libraries would duplicate established project patterns.
