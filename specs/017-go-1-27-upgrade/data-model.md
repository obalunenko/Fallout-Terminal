# Data Model: Go 1.27 Upgrade

This feature introduces no application data or persistence changes. Its governed configuration model is the repository toolchain baseline described below.

## Toolchain Baseline

Represents the supported Go release line across the repository.

| Attribute | Value | Validation |
|---|---|---|
| Stable module baseline | `1.27.0` | Root and four isolated tool modules declare the same value. |
| CI selector | `1.27.x` | Resolves to a stable Go 1.27 patch release. |
| Release preflight selector | `go1.27.*` | Accepts stable patch-form `go version` output and rejects other release lines or prerelease labels. |
| Human-facing release family | Go 1.27 | README, planning template, and constitution identify the active release family. |

## Relationships

- The root application module and each isolated tool module independently own their dependency graph while sharing the Toolchain Baseline.
- CI, release preflight, validation fixtures, and acceptance assertions enforce that baseline.
- Active guidance and governance communicate the same baseline without rewriting historical evidence.

## State Transition

`Go 1.26 active baseline` → update every active owner → validate all module and repository gates with Go 1.27 → `Go 1.27 active baseline`.

There is no migration of session data, application state, protobuf contracts, or generated messages.
