# Implementation Plan: Application Logging

**Branch**: `016-application-logging` | **Date**: 2026-08-21 | **Spec**: [spec.md](./spec.md)

## Summary

Add structured operational diagnostics at the root application, desktop lifecycle, and player-server boundaries using the exactly pinned `github.com/obalunenko/logger` runtime dependency. Initialize logging once at process startup, inject the logger into testable owners, record lifecycle and important trusted-command outcomes, and centralize reporting for event-delivery errors that are intentionally swallowed. Tests will capture structured records through a deterministic fake and prove both expected coverage and absence of secrets or user-authored content.

## Project Structure

```text
.
├── go.mod                              # Pin the production logger dependency
├── go.sum                              # Record reproducible module checksums
├── main.go                             # Initialize logging and handle fatal process failures
├── app.go                              # Lifecycle, command-outcome, and event-delivery records
├── app_test.go                         # Application logging and redaction coverage
├── wails_host.go                       # Record startup errors intentionally absorbed by Wails
├── wails_host_test.go                  # Verify absorbed lifecycle failures are logged
└── internal
    ├── player
    │   ├── server.go                   # Record unexpected background serving failures
    │   └── server_test.go              # Verify expected versus unexpected serve exits
    └── testutil
        └── logger.go                   # Deterministic structured logger fake for Go tests
```

**Structure Decision**: Keep logging at composition and operational boundaries; domain, persistence, control, navigation, live-state, and tunnel state owners remain independent of diagnostic policy and continue returning typed outcomes to their callers.

## Constitution Check

| Principle | Assessment | Evidence |
|---|---|---|
| I. Govern the Accepted Desktop Runtime | PASS | Logging stays in root composition, the Wails lifecycle adapter, and the player runtime boundary; no Wails dependency enters domain packages. |
| II. Make Protobuf the Application Contract Source of Truth | PASS | Records are process diagnostics, not application-owned serialized or externally consumed structured contracts; no DTO or schema is introduced. |
| III. Use ConnectRPC and Keep State Server-Authoritative | PASS | No RPC, mutation, stream, or state-ownership behavior changes. |
| IV. Separate Public and Private Capabilities | PASS | Records contain operational metadata only and expose no private capability or player-facing route. |
| V. Evolve Schemas Safely and Reproducibly | PASS | No schema or generated artifact changes. |
| VI. Preserve Portable Session JSON Version 1 | PASS | Session documents, adapters, paths, and persistence behavior are unchanged; session content is explicitly excluded from records. |
| VII. Complete Cutovers and Remove Superseded Protocols | PASS | The standard-library application logger is removed from production startup; build-tool and fixture command output remains explicitly out of scope. |
| Dependency Rules | PASS | `github.com/obalunenko/logger` is a concrete production runtime dependency pinned in the root module; no development tool dependency is added. |
| Secret and Credential Governance | PASS | Provider tokens, player passwords, generated passwords, raw provider errors, and secret-store details are prohibited from records and covered by tests. |
| Testing and Quality Gates | PASS | Colocated deterministic tests use Testify, test-scoped contexts, full Go tests, vet, formatting, race checks, and the existing secret-leak check. |

No constitution violations require Complexity Tracking.
