# Validation Evidence

Date: 2026-08-13 (Asia/Tbilisi)
Revision under test: feature 005 working tree

## Outcome

All repository-controlled acceptance gates pass. The migration has one public
player protocol (ConnectRPC over HTTP), generated protobuf contracts, typed
private and persistence adapters, bounded request handling, reproducible code
generation, and no active legacy WebSocket/JSON player stack.

A literal three-to-four-hour run over a published ngrok domain is intentionally
recorded as an external release-operator observation because it depends on an
ngrok account, fixed domain, credentials, and public network availability. It
does not replace or weaken the passing scheduled three-hour-equivalent stream
soak, protected forwarding journey, tunnel-policy tests, or reconnect tests.

## Command matrix

| Gate | Command | Result |
|---|---|---|
| Schema format, lint, generation, revision, adapter and bundle checks | `scripts/proto-check.sh` | PASS; two clean generations were byte-identical and the second left no generated diff |
| Representative breaking edits | `scripts/proto-breaking.sh --all-fixtures` | PASS; field-number, field-type, enum, package, and service fixtures were all rejected |
| Go formatting | `gofmt -l .` | PASS; no files reported |
| Go static analysis | `GOCACHE=/private/tmp/fallout-go-cache go vet ./...` | PASS |
| Go suite | `GOCACHE=/private/tmp/fallout-go-cache go test ./... -count=1` | PASS |
| Go race suite | `GOCACHE=/private/tmp/fallout-go-cache go test -race ./... -count=1` | PASS |
| Player build | `npm run build --prefix client` | PASS |
| Desktop frontend build | `npm run build --prefix frontend` | PASS |
| Browser journeys | `npm test --prefix tests/browser` | PASS; 17/17 |
| Public/protected reconnect focus | `npm test --prefix tests/browser -- --grep 'protected forwarding|recognized reconnect|concurrent clean tabs'` | PASS; 3/3 |
| Three-hour-equivalent stream soak | `go test ./internal/player -run TestRepresentativeThreeHourStreamReconnectSoak -count=1 -v` | PASS; 10,800 updates, 36 reconnects |
| Contract, bundle, offline and cutover scans | focused `internal/platform` tests for protocol cutover, public generated imports, packaged player assets, and contract separation | PASS |
| macOS package | `wails build -clean -platform darwin/arm64` | PASS; arm64 application bundle produced |
| Packaged runtime smoke | launch `build/bin/Fallout Terminal.app/Contents/MacOS/Fallout Terminal` with `NGROK_ENABLED=0`, then request `/` and `SoundManifest` | PASS; HTTP 200, typed manifest, embedded assets, CSP without `ws:`/`wss:` |
| Patch hygiene | `git diff --check` | PASS |

## Convergence revalidation

The post-analysis convergence work was revalidated after closing T123–T129.
The full Go suite and race suite pass with the gap-free subscription boundary,
transaction-owned personalized updates, exhaustive private adapters, and
Connect-native HTTP errors. `scripts/proto-check.sh` passes with the pinned,
in-process compiler owned by Buf v1.72.0; all 11 generated Go protobuf files
record pinned `protoc-gen-go v1.36.11` provenance and the valid Buf-generated
`protoc (unknown)` compiler line. The five breaking fixtures, all 17 browser
journeys, the production Wails build, and a packaged-app launch smoke also pass.

The packaged executable SHA-256 observed in this validation run was
`fe0252b7237e41f1d4b03841b35f2c02491b07dbd1bdfa1aabc63e62e8103c24`.
The smoke concerns the affected unsigned offline development package; signing
and notarization are outside this feature's acceptance scope.

## Success Criteria ledger

| Criterion | Evidence | Result |
|---|---|---|
| SC-001 | Final inventory reconciliation reports zero unclassified application DTO fields and zero unclassified serializable configuration fields. | PASS |
| SC-002 | `scripts/proto-check.sh` ran Buf format and lint with no findings. | PASS |
| SC-003 | The schema check generated twice from clean staging directories, compared byte-for-byte, and detected no second-pass repository drift. | PASS |
| SC-004 | `scripts/proto-breaking.sh --all-fixtures` rejected all five representative incompatibility classes. | PASS |
| SC-005 | Generated Go and ECMAScript outputs matched `proto/schema-revision.txt` (`66679f…3ba3`). | PASS |
| SC-006 | First-value and reconnect tests require exactly one complete personalized snapshot before updates. | PASS |
| SC-007 | Snapshot/reconnect generation guards, including 100 reconnect trials, observed no puzzle regeneration. | PASS |
| SC-008 | Concurrent clean-tab browser and coordinator trials converged on one recognized handle and logical session. | PASS |
| SC-009 | Stream hub tests verified per-stream removal and logical disconnect only after the final stream closes. | PASS |
| SC-010 | The 100-trial concurrent claim test accepted at most one claimant per character. | PASS |
| SC-011 | Authority tests allow initial character selection and reject navigation/hacking until the session is assigned, eligible, connected, and controlling. | PASS |
| SC-012 | The retained-request replay matrix returned the original result/revision with no second canonical effect across at least 100 trials. | PASS |
| SC-013 | Changed procedure or fingerprint under a retained request identity returned `duplicate` with no canonical effect. | PASS |
| SC-014 | Replay-cache stress remained within its configured bound and made no claim for evicted records. | PASS |
| SC-015 | The 100-trial concurrent pattern test observed one accepted mutation, one normal accepted draw, at most one dud draw, and no rejected-contender randomness. | PASS |
| SC-016 | Coordinator/handler/stream tests observed one mutation, revision, compound update, and logical-session offer per accepted action, with at-most-once physical delivery. | PASS |
| SC-017 | Subscribers observed strictly increasing post-snapshot revisions while irrelevant revisions could be skipped. | PASS |
| SC-018 | Multi-stream browser/coordinator journeys converged after mixed selection, navigation, guess, pattern, replay, rejection, and reconnect operations. | PASS |
| SC-019 | Result-first and stream-first tests held accepted pending actions until both conditions and cleared rejected actions immediately. | PASS |
| SC-020 | Deterministic overflow tests kept mutation and responsive delivery non-blocking, closed only the blocked stream, and shut down within the bound. | PASS |
| SC-021 | Private Wails adapter tests cover every inventoried method/event and preserve native JavaScript-facing shapes. | PASS |
| SC-022 | Adapter exhaustiveness verification fails when a private protobuf field lacks a mapping. | PASS |
| SC-023 | Platform scans found no protobuf-binary/Base64/ProtoJSON Wails carrier or generic desktop dispatcher. | PASS |
| SC-024 | Public descriptor/security tests expose only the six player responsibilities and reject access to private/native/credential/other-session data. | PASS |
| SC-025 | Public generated source and transitive descriptor imports contain no private desktop, persistence, credential, native-path, tunnel, or secret hacking graph. | PASS |
| SC-026 | Session-v1 fixture round trips preserve known fields, normalized player-config reference, and compatible unknown fields at supported levels. | PASS |
| SC-027 | Player-config-v1 fixtures preserve strict validation and publish only after successful atomic save. | PASS |
| SC-028 | HTTP boundary tests reject oversized unary/stream/manifest, decoded compressed, unknown-field-expanded, and malformed bodies with the required Connect codes before any adapter or canonical effect. | PASS |
| SC-029 | Sound tests cover eight categories, five extensions, deterministic order, empty success, asynchronous failure, and authoritative one-shot cues. | PASS |
| SC-030 | Local and protected-forwarding browser journeys cover loading, snapshot, all unary responsibilities, updates, tabs, reconnect, 401, manifest, and playback; literal public ngrok observation is external-only. | PASS + external observation noted |
| SC-031 | Scheduled accelerated soak delivered 10,800 ordered revisions with 36 complete-snapshot reconnects; protected forwarding/auth semantics passed. | PASS |
| SC-032 | The packaged app served generated player code, sound assets, and the typed manifest offline without CDN/dev-server/package download. | PASS |
| SC-033 | Formatting, vet, normal/race tests, schema checks, both builds, browser journeys, and macOS package smoke all passed. | PASS |
| SC-034 | Source, dependency, route, built-asset, fixture, CSP, and active-doc scans found no active WebSocket implementation, legacy JSON envelope, direct legacy dependency, fixture, or dual stack. | PASS |

## Failure resolution and external observations

The first macOS package attempt exposed a Wails command-runner incompatibility:
compound `&&` commands in `wails.json` were forwarded as Vite arguments. The
commands now live in `frontend/package.json` as `install:all` and `build:all`,
and `wails.json` invokes those named scripts. The package build and runtime
smoke passed after that change. The startup contract test was updated to verify
both the Wails entry points and the exact compound npm scripts; the complete
race suite then passed.

No repository-controlled criterion remains failed. For a release that requires
literal public-internet observation, an operator should follow `quickstart.md`,
publish the configured authenticated ngrok endpoint, leave an idle browser open
for three to four wall-clock hours, interrupt/reconnect it, and confirm that the
first recovered application value is one complete current snapshot. This is an
additional environment-dependent observation, not a waiver of SC-030 or
SC-031.
