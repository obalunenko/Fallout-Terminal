# Validation: Player Intelligence and Hacker Perk Management

**Date**: 2026-08-19  
**Automated validation owner**: T030

## Automated gates

| Gate | Result | Evidence |
|---|---|---|
| Repository quality gate | PASS | `make check` completed: formatting, `go vet ./...`, full `go test -race ./...`, protobuf format/lint/deterministic generation, breaking fixtures, and Wails binding drift all passed. |
| Protobuf compatibility | PASS | Schema revision `70a35fd47fb285170ff2b17f8f6fcccdce4bea3773d56d33082866eecce12ed1`; all five negative breaking fixtures were rejected. |
| Wails private surface | PASS | Deterministic generation reported 34 methods; `UpdateCharacter` replaced `RenameCharacter` and the allowlist check passed. |
| Focused player-management browser/API | PASS | `npm test --prefix tests/browser -- desktop-api.spec.mjs player-management.spec.mjs`: 20/20 passed. |
| Full browser regression | PASS | `npm test --prefix tests/browser`: 112 passed, 2 skipped. The skipped cases are the existing opt-in tests for a real authenticated ngrok endpoint; all deterministic local journeys ran. |
| Master frontend build | PASS | `npm run build --prefix frontend` passed; the owned build repeated the production Vite build successfully. |
| Owned application build | PASS | `env GOCACHE=/private/tmp/fallout-root-gocache go run ./cmd/build build` completed and produced `build/bin/Fallout Terminal`, a macOS arm64 Mach-O executable. |
| Diff hygiene | PASS | `git diff --check` passed after implementation and generated output review. |

## Success criteria

| Criterion | Result | Evidence |
|---|---|---|
| SC-001: add a complete player in under 60 seconds | PASS (automated) | The end-to-end Playwright add journeys completed in well under one second, submitted the complete profile, and rendered the authoritative result. Native human-timed confirmation is recorded separately by T031. |
| SC-002: retain Intelligence boundaries and both Hacker choices | PASS | Browser tests accepted Intelligence 1 and 10 with Hacker false and true, reloaded the fixture, and verified the same detailed values. Domain/player-config tests verify canonical reopen. |
| SC-003: reject every active-broadcast roster mutation | PASS | Control race tests and the crafted browser add/update/delete calls all returned refusal while preserving revision and roster. |
| SC-004: identify every player's complete details in the popup | PASS | Accessible populated-list coverage verifies name, Intelligence, and Hacker availability for every row, including read-only broadcast mode and mobile scrolling. |
| SC-005: open legacy player configurations without repair | PASS | Strict JSON and protobuf tests default independently missing Intelligence to 1 and Hacker availability to false while still rejecting explicit invalid/null values. |
| SC-006: storage failures produce no partial roster changes | PASS | Player-config, coordinator, App, and browser failure tests cover digest conflict and atomic-write failure for add/update/delete; file, digest, roster, order, revision, and effects remain unchanged. |

## Warnings and scope notes

- Linker output warns that some cached Wails objects were built for a newer macOS version than the test link target. Tests and the owned arm64 build still completed successfully; this is a toolchain-cache warning, not a feature failure.
- The two skipped full-browser cases require real ngrok credentials and are intentionally opt-in. Public-access deterministic regressions passed.
- Rolling back to the pre-feature strict reader after saving a canonical file requires restoring a pre-feature copy or removing the two additive roster members, as documented in the plan.

## Native interactive validation

**Result: PASS.** T031 launched the owned development application with `env GOCACHE=/private/tmp/fallout-root-gocache go run ./cmd/build dev` and exercised the real macOS/Wails master window through native computer control.

- Created a new session and writable player configuration through the native save panels.
- Opened the dedicated **УПРАВЛЕНИЕ ИГРОКАМИ** popup and added `Native Courier` with Intelligence 7 and Hacker available. From beginning field entry to the authoritative rendered row, the interaction took about 4 seconds, comfortably below SC-001's 60-second target.
- Edited the same stable row to `Native Courier Prime`, Intelligence 10, Hacker unavailable; the popup reported **ПРОФИЛЬ ИГРОКА СОХРАНЁН**.
- Added a second profile, invoked its custom destructive confirmation, confirmed deletion, and observed **ИГРОК УДАЛЁН** with the first profile preserved.
- Closed and reopened the popup and observed the saved `Native Courier Prime / 10 / НЕДОСТУПЕН` details. A direct read of the native-created configuration after the journey confirmed the canonical durable JSON values and one surviving stable ID.
- Started a native broadcast, reopened the popup, and observed **ТРАНСЛЯЦИЯ АКТИВНА · ПРОСМОТР БЕЗ РЕДАКТИРОВАНИЯ**. Name, Intelligence, and Hacker details remained visible while Save, Delete, and Add controls were disabled.
- Ended the interactive run and stopped the development process. The pre-feature rollback caveat remains the documented one: restore a pre-feature configuration copy or remove the two additive roster members before using the old strict reader.

Native test artifacts were intentionally left in `~/Documents/fallout-feature-011-native.json` and `~/Documents/fallout-feature-011-players.json` so the exact durable result remains inspectable.
