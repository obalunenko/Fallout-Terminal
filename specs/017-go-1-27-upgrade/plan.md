# Implementation Plan: Go 1.27 Upgrade

**Branch**: `017-go-1-27-upgrade` | **Date**: 2026-08-21 | **Spec**: [spec.md](./spec.md)

**Bugfix**: 2026-08-21 — BUG-001 updated from bugfix patch.

## Summary

Raise the root application module and all four isolated development-tool modules to the stable Go 1.27.0 baseline, then align CI, release preflight, active fixtures, assertions, contributor guidance, planning defaults, and project governance with that release line. Preserve all dependency pins, generated outputs, application behavior, and dated historical evidence; use Go 1.27.0 module maintenance and the repository's existing quality/package gates to prove compatibility. On macOS, run local Go validation through `make test`, `make test-race`, or an explicitly equivalent environment so the link target remains aligned with the governed macOS 13 deployment target.

## Project Structure

```text
.
├── go.mod                                      # Root application Go 1.27.0 baseline
├── tools
│   ├── buf/go.mod                              # Isolated Buf tool baseline
│   ├── protoc-gen-connect-go/go.mod            # Isolated Connect generator baseline
│   ├── protoc-gen-go/go.mod                    # Isolated protobuf generator baseline
│   └── wails/go.mod                            # Isolated Wails CLI baseline
├── .github/workflows/wails-macos.yml           # CI Go 1.27 patch-line selection
├── scripts
│   ├── build-macos.sh                          # Stable Go 1.27 release preflight
│   ├── tool-modules-check.sh                   # Tool-module validation fixtures
│   ├── wails-v3-contract-check.sh              # Contract fixtures and reviewed baselines
│   └── wails-v3-cutover-check.sh               # Cutover validation fixture
├── Makefile                                     # Canonical quality-gate prerequisites
├── internal/platform/startup_test.go            # Tool-module baseline assertion
├── README.md                                    # Active contributor prerequisite
└── .specify
    ├── memory/constitution.md                   # Governed production baseline
    └── templates/plan-template.md               # Default plan runtime baseline
```

**Structure Decision**: Change every active owner of the Go version in place and leave completed specs, rollback records, dated evidence, dependency versions, and generated artifacts untouched.

## Constitution Check

| Principle | Assessment | Evidence |
|---|---|---|
| I. Govern the Accepted Desktop Runtime | PASS | The accepted Wails v3 runtime and application boundaries are unchanged; only their Go baseline advances. |
| II. Make Protobuf the Application Contract Source of Truth | PASS | Go module and build-tool metadata are explicitly outside protobuf governance; no application contract changes. |
| III. Use ConnectRPC and Keep State Server-Authoritative | PASS | RPC transport, state ownership, and generated messages are unchanged. |
| IV. Separate Public and Private Capabilities | PASS | No capability surface or authorization boundary changes. |
| V. Evolve Schemas Safely and Reproducibly | PASS | Generator pins and schemas stay fixed, and deterministic generation checks guard against drift under Go 1.27. |
| VI. Preserve Portable Session JSON Version 1 | PASS | Persistence formats and adapters are untouched. |
| VII. Complete Cutovers and Remove Superseded Protocols | PASS | Active toolchain references move together; historical records retain their original evidence. |
| Dependency Rules | PASS | The root and isolated tool module ownership model remains intact, with exact dependency versions preserved. |
| Secret and Credential Governance | PASS | No secret-handling path or diagnostic content changes. |
| Go Development Tool Modules | PASS | Every isolated tool module receives the same explicit Go 1.27.0 baseline and remains independently tidy and reproducible. |
| Testing and Quality Gates | PASS | The plan runs formatting, vet, unit, race, module-isolation, contract, generation, reproducibility, and package gates with Go 1.27, including a fresh-cache warning-free test under the governed macOS 13 environment. |

No constitution violations require Complexity Tracking.

The final design re-check remains PASS: the upgrade adds no runtime dependency, application contract, state transition, or second build path.

## Bugfix Validation

For BUG-001, create a fresh writable Go build cache and run the canonical `make test` target while capturing combined output. The task passes only when the test suite succeeds and the capture contains zero `ld: warning` lines. A comparison run confirmed that raw `go test` without the repository environment reproduces the mismatch, while the existing Makefile exports `MACOSX_DEPLOYMENT_TARGET=13.0`, `CGO_CFLAGS=-mmacosx-version-min=13.0`, and `CGO_LDFLAGS=-mmacosx-version-min=13.0` and eliminates it.
