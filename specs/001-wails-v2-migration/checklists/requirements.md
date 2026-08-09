# Specification Quality Checklist: Wails v2 Runtime Migration

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-08-09
**Feature**: [spec.md](../spec.md)

## Content Quality

- [x] No implementation details beyond the user-mandated runtime migration constraint
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

## Requirement Completeness

- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Success criteria are technology-agnostic
- [x] All acceptance scenarios are defined
- [x] Edge cases are identified
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified
- [x] Personal-use and optional public-distribution acceptance profiles are explicitly separated

## Feature Readiness

- [x] All functional requirements have clear acceptance criteria
- [x] User scenarios cover primary flows
- [x] Feature meets measurable outcomes defined in Success Criteria
- [x] No unnecessary implementation details leak into specification

## Notes

- The named runtime and current component boundaries are retained only because replacing them is the feature's explicit scope; detailed technology choices belong in `plan.md` and `research.md`.
- Validation iteration 2 passed after recording the approved constitution amendment, macOS-first target, `develop`-based feature branch, 4–7-player envelope, and hybrid macOS session-storage policy.
- Validation iteration 3 passed after making the personal-use Apple Silicon `.app` the active acceptance profile and treating Developer ID signing/notarization as a conditional future public-release gate.

## Implementation Verification (2026-08-09)

All rows below were checked against the final post-Electron source tree. `PASS`
means the selected personal-use profile satisfies the requirement. Public-only
trust checks are explicitly conditional and remain `N/A`, not failed.

| Requirement | Result | Evidence |
| --- | --- | --- |
| FR-001 | PASS | Six behavioral specs plus Sections 4–7 of `quickstart.md`; no undocumented intentional parity differences. |
| FR-002 | PASS | Application partial-start/reverse-shutdown tests and final listener cleanup run. |
| FR-003 | PASS | Wails window configuration, migrated dark UI, restrictive master CSP, and narrow bridge review. |
| FR-004 | PASS | `app_test.go`, domain validation, structured bridge results, and HTTP(S)-only URL tests. |
| FR-005 | PASS | Version-1 golden decode/encode and session create/open/save tests. |
| FR-006 | PASS | Cancellation, invalid-open retention, explicit demo copy, and selected-path tests. |
| FR-007 | PASS | Ordered/coalesced save-worker tests covering 20 rapid revisions and stale completions. |
| FR-008 | PASS | Player HTTP/server tests and final `*:3690` packaged/development probes. |
| FR-009 | PASS | Separate embedded player filesystem and real HTTP/WebSocket listener checks. |
| FR-010 | PASS | Twelve protocol goldens cover all eight retained identifiers and strict compatibility tests. |
| FR-011 | PASS | Mutex-owned live/navigation/hacking tests and session runtime-field exclusion. |
| FR-012 | PASS | Hack/live/protocol serialization tests reject `secretWord` and `wordsById` leakage. |
| FR-013 | PASS | Full `go test -race ./...` and 4–7-client convergence tests passed. |
| FR-014 | PASS | RuntimeStatus plus server-info, client-count, and public hack event tests. |
| FR-015 | PASS | Browser presentation/audio evidence and BUG-001 idle/normal/hacking/blocked overflow checks. |
| FR-016 | PASS | Tunnel configuration/service tests plus authenticated public HTTP/WSS evidence. |
| FR-017 | PASS | Policy `0700`/`0600` and success/failure/timeout/shutdown cleanup tests. |
| FR-018 | PASS | Final parsed-protocol allowlist tests at the browser boundary. |
| FR-019 | PASS | Asset-manifest tests and rebuilt package inventory for master, player, fonts, sounds, demo, and icon. |
| FR-020 | PASS | Rebuilt arm64 ad-hoc-signed app passed integrity and system-only-PATH single-launch checks; public trust branch N/A. |
| FR-021 | PASS | Documents/Application Support/demo-copy/path-retention tests and package evidence. |
| FR-022 | PASS | Clean-source normal and race-enabled suites passed every tested package. |
| FR-023 | PASS | Sections 2–10 of `quickstart.md` record automated, browser, public-access, storage, package, and conditional-public procedures. |
| FR-024 | PASS | Electron oracle remained until P1/security/session/package acceptance and rollback evidence were complete. |
| FR-025 | PASS | Electron sources/root dependencies were removed only after acceptance; final asset manifest passed. |
| FR-026 | PASS | Final root `wails dev` and packaged single-launch runs owned every required runtime component. |
| FR-027 | PASS | Darwin owner-pipe regression plus the real authenticated T078 `wails dev` interrupt left zero ngrok/guardian processes, port-3690 listeners, or policy directories on the first bounded poll, without a manual kill. |
| FR-028 | PASS | `TestActiveFrontendUsesRuntimeNeutralDesktopFacade`, frontend production build, and source scan require `window.desktopAPI` and found zero active `electronAPI` definitions or consumers. |

| Success criterion | Result | Evidence |
| --- | --- | --- |
| SC-001 | PASS | Six-spec parity and full acceptance evidence; only documented storage/runtime changes remain. |
| SC-002 | PASS | 4, 5, 6, and 7 real WebSocket clients converged through 25 mixed actions and reconnect with no private fields. |
| SC-003 | PASS | Version-1 full-variant/unknown-field round trips and 20-revision ordered save tests. |
| SC-004 | PASS | Runtime readiness under one second after native launch; actionable failure and unwind tests passed. |
| SC-005 | PASS | Invalid credentials started zero processes; authenticated public HTTP/WSS rejected anonymous access. |
| SC-006 | PASS | Local, connected-player, public-tunnel, and partial-start shutdown evidence left zero owned resources. |
| SC-007 | PASS | Personal arm64 bundle integrity, assets, ad-hoc signature, P1, and tool-free launch passed; public DMG branch N/A. |
| SC-008 | PASS | Fresh source snapshot passed gofmt, vet, normal tests, and race tests; the guide required no source repair. |
| SC-009 | PASS | One root `wails dev` restored the player and one packaged launch served it with no separately invoked component. |
| SC-010 | PASS | The executable consistency test requires exactly one matching canonical record in quickstart and rollback/handoff guidance: commit `118ed8199a3a0b1c3b73a09ef98908949c2e2d75`, executable SHA-256 `d1ad65f5e5a80f3471e2d551d0ca5d1e55a8d2447cef58091a37cb35276cc121`. |
| SC-011 | PASS | Authenticated public mode returned anonymous `401`, authenticated HTTP `200`, authenticated WSS `101`, then one handled supervisor interrupt left all four resource counts at zero. |
| SC-012 | PASS | Active frontend source and built assets use the runtime-neutral facade; static contract and frontend build passed with no Electron bridge global. |

Final verification passed all 28 functional requirements and all 12 success
criteria for the selected personal-use profile. Developer ID signing,
notarization, stapling, signed DMG, and public Gatekeeper checks remain
`N/A (personal profile)` and are still required before public publication.
