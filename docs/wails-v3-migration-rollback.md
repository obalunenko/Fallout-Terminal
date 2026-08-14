# Wails v3 Migration Rollback and Cutover Record

This record governs the bounded Wails v2-to-v3 migration in feature 006. The
accepted pre-migration Wails v2 source remains the production fallback until
every required personal-use parity, package, soak, and rollback gate passes.
The Electron-to-Wails record in `docs/wails-migration-rollback.md` is immutable
historical evidence and is not a rollback authority for this migration.

## Ownership and Expiry

| Field | Value |
| --- | --- |
| Temporary coexistence owner | Feature-006 implementer; repository owner approves cutover |
| Coexistence scope | `006-wails-v3-migration` branch only |
| Expiry | Final feature-006 cutover, immediately after all required gates pass |
| Permanent v2/v3 switch | Prohibited |

## Canonical Wails v2 Source Rollback

| Field | Value |
| --- | --- |
| Source commit | `f1084b3df8b5630862bdf7a0f347b599156653ef` |
| Source verification | `PASS` — `git cat-file -t f1084b3df8b5630862bdf7a0f347b599156653ef` returned `commit`; `git show -s` identified `feat: Migrate player protocol to ConnectRPC and Protobuf (#8)` on 2026-08-13 |
| Application runtime | Wails v2.13.0 accepted pre-migration baseline |
| Session compatibility | session-v1; no conversion permitted or required |
| Player configuration compatibility | player-config-v1; no conversion permitted or required |

The source commit is the only canonical rollback reference unless a Wails v2
application is genuinely built, manually accepted, and recorded below. Never
invent or prefill an artifact digest.

## Optional Accepted Wails v2 Artifact

| Field | Value |
| --- | --- |
| Artifact status | `BUILT FOR DRILL — ACCEPTED FOR THIS DRILL` |
| Artifact path | `/private/tmp/fallout-t066-v2-20260814/build/bin/Fallout Terminal.app` (temporary, non-canonical) |
| Executable SHA-256 | `c1faf7fe4f2ed0abc5c4814b8e71805f5b57a65b817fd3a45bbcc90bdaf29530` |
| Build provenance | Clean isolated clone at `f1084b3df8b5630862bdf7a0f347b599156653ef`; exact `go run github.com/wailsapp/wails/v2/cmd/wails@v2.13.0 build -clean -platform darwin/arm64` |
| Acceptance result | `PASS` |
| Acceptance evidence | Built, packaged, and ad-hoc signed successfully. A synchronized corrected rerun served four distinct players, observed the typed `Guess` RPC, accepted the master hack-success override, completed terminal navigation/back and reconnect with retained identity, reopened the saved version-1 safety copy, and released its process/listener cleanly. The temporary artifact passed this drill, while the immutable source commit remains rollback authority |

## Candidate and Cutover Identity

These fields were recorded only after the v2-free source, active operating
documentation, pins, and locks were committed and the working tree was clean.
Evidence-only commits record the frozen candidate but do not redefine it.

| Field | Value |
| --- | --- |
| Build candidate commit | `658071b7011197c4f229f6a5b1f109de2764fd69` |
| Wails Go/runtime/CLI pin | `github.com/wailsapp/wails/v3 v3.0.0-beta.8`; isolated `wails3` tool at the same parent-module version |
| Frontend runtime/plugin pin | `@wailsio/runtime` `3.0.0-beta.8`, including `@wailsio/runtime/plugins/vite` |
| Personal-use app path | `build/bin/Fallout Terminal.app` |
| Canonical bundle-manifest SHA-256 |  |
| Cutover result | `NOT RUN` |
| Cutover timestamp/environment | Build candidate frozen 2026-08-14T16:09:18+04:00 on macOS arm64; final acceptance pending Phase 11 |

The pre-removal qualification worktree was branch `006-wails-v3-migration`,
based on commit `bcb207704657a92f9902f4ac04ef11765b18f031`. Its migration changes and
evidence were intentionally uncommitted during the rollback drill and soak, so
that base commit remains provenance only—not the build candidate and not an
accepted artifact identity. T069 froze the first v2-free clean commit,
`658071b7011197c4f229f6a5b1f109de2764fd69`, as the immutable build candidate.

## Rollback Triggers

| Trigger | Required action | Decision owner |
| --- | --- | --- |
| Session corruption, loss, incompatible round trip, or an older save replacing the newest accepted revision | Stop immediately; preserve originals and safety copies; return to canonical v2 source | Repository owner |
| Private bridge capability, inventory, payload, validation, cancellation, or redaction drift | Halt cutover; retain evidence; restore canonical v2 | Feature implementer, then repository owner |
| Master/player visual, persistence, gameplay, authority, replay, reconnect, convergence, sound, or privacy regression | Halt the affected journey and cutover; restore v2 unless a migration-only correction passes the full matrix | Feature implementer |
| Startup/shutdown listener, stream, session-worker, tunnel, temporary-policy, or child-process leak | Stop candidate and clean only its exact owned resources; restore v2 | Feature implementer |
| Local/public access regression, credential exposure, or loss of local fallback after tunnel failure | Disable public mode, preserve redacted evidence, restore v2 | Repository owner |
| Missing master/player/binding/font/sound/icon/plist/entitlement/demo resource | Reject package and restore v2 | Feature implementer |
| Unhandled Wails v3 beta-runtime crash | Reject v3 candidate and restore v2 | Repository owner |
| Architecture, minimum-OS, integrity, signature, notarization, staple, DMG, or Gatekeeper failure for the selected profile | Reject only that profile; restore v2 for any required personal-use gate | Repository owner |

## Data-Safe Rollback Procedure

1. Stop the Wails v3 candidate and verify port 3690, owned ngrok processes, and
   temporary policy material are gone within five seconds.
2. Record the candidate identity and make safety copies of the selected
   session-v1 and player-config-v1 files outside the repository.
3. Record SHA-256 values for the originals and safety copies. Never include
   credentials or user file contents in repository evidence.
4. Create a separate maintenance worktree or clone at
   `f1084b3df8b5630862bdf7a0f347b599156653ef`; do not overwrite the migration
   worktree or revert unrelated work.
5. Build and run that source with its recorded Wails v2 toolchain, or run the
   separately accepted artifact recorded above.
6. Open only the safety-copy version-1 data without migration or conversion.
7. Exercise representative master create/open/edit/save/reopen behavior and a
   four-player local selection/navigation/hacking/reconnect journey.
8. Quit and verify port 3690 and every owned resource are released.
9. Record the real result below. A missing prerequisite or unperformed journey
   is `NOT RUN`, never `PASS`.

## Rollback Drill Evidence

| Field | Value |
| --- | --- |
| Candidate/source tested | Pre-removal feature-006 worktree based on `bcb207704657a92f9902f4ac04ef11765b18f031`; rollback source `f1084b3df8b5630862bdf7a0f347b599156653ef` |
| Safety-copy paths or redacted identifiers | `/private/tmp/fallout-t066-rerun-20260814/session-v1.json`; `/private/tmp/fallout-t066-rerun-20260814/player-config-v1.json` |
| Original/safety-copy SHA-256 values | Session original/copy before drill: `c15baf6195a2a07cb7ed7985693c21bc910ae83092656483c94861ba39692e9c`; player-config original/copy: `07d5d888c58d65c6a3b8769988081215e299bbdf5a206e64d5639e5a5eff06b7`; post-save session safety copy: `0c448064f81665c4d99f0ccf721270227027fc28cc163693a664ebcaa129a36f` |
| Wails v2 source or artifact used | Clean isolated clone at canonical source commit; temporary executable digest `c1faf7fe4f2ed0abc5c4814b8e71805f5b57a65b817fd3a45bbcc90bdaf29530` (accepted for this drill; immutable source commit remains canonical rollback authority) |
| Session-v1 open/save/reopen result | `PASS` — opened and saved as version 1 with the relative player-config association, quit cleanly, then reopened the exact safety-copy path with both expected terminals and the associated seven-character configuration visible; post-reopen SHA-256 remained `0c448064f81665c4d99f0ccf721270227027fc28cc163693a664ebcaa129a36f` |
| Player-config-v1 result | `PASS` — opened unchanged as version 1 with seven roster characters; post-drill SHA-256 remained identical |
| Representative local master/player result | `PASS` — one v2 app/listener served four distinct assigned players; the synchronized harness observed one typed `Guess` RPC, accepted the master's hack-success override, navigated into a terminal row and back, reconnected with the same retained player identity, and observed 40 typed sound-manifest requests |
| Post-quit cleanup result | `PASS` — no `Fallout Terminal` process or port-3690 listener remained on the immediate post-quit check |
| Overall drill result | `PASS` |
| Timestamp/environment/evidence | 2026-08-14, macOS arm64, Go 1.26.5, Node 26.7.0, Wails v2.13.0; harness `/private/tmp/fallout-t066-v2-drill.mjs` |

## Qualification and Final Evidence

| Gate | Candidate/artifact identity | Result | Evidence/reason |
| --- | --- | --- | --- |
| Pre-removal 60-minute local soak | Pre-removal feature-006 worktree based on `bcb207704657a92f9902f4ac04ef11765b18f031`; packaged app PID 63116 | `PASS` | 60.67 minutes; seven players; 57 accepted operations; 28 rejected observer attempts; three reconnects; two durable version-1 save/reopen cycles; 90 convergence checks; RSS medians 117936/128432/121312 KiB; one listener; 0.014s cleanup |
| Pre-removal authenticated-ngrok soak | Same pre-removal worktree; app PID 2695; owned ngrok PID 2700; configured protected endpoint | `PASS` | 32.01 minutes; seven players; unauthenticated HTTP 401; 20 accepted operations; 10 rejected observer attempts; two reconnects; 32 convergence checks; controlled tunnel loss preserved local HTTP 200; 0.020s final cleanup; temporary credential deleted |
| Final-candidate native journeys |  | `NOT RUN` |  |
| Final personal-use package/offline smoke |  | `NOT RUN` |  |
| Final-candidate 60-minute local soak |  | `NOT RUN` |  |
| Final authenticated-ngrok soak |  | `NOT RUN` |  |
| Developer ID/notary/staple/DMG/Gatekeeper |  | `NOT RUN` |  |
| Final active-v2/cutover scan |  | `NOT RUN` |  |

Wails v3 is accepted only after every required non-conditional gate is `PASS`
against the frozen v2-free build candidate and its recorded personal-use
bundle identity. Conditional public gates may remain `NOT RUN` only with an
honest reason and cannot serve as passing public-mode evidence.
