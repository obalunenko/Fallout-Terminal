# Toolchain Baseline Contract

## Version ownership matrix

| Consumer | Required value | Contract |
|---|---|---|
| Root application module | `go 1.27.0` | Defines the minimum application language and toolchain baseline. |
| `tools/buf` module | `go 1.27.0` | Executes the pinned Buf CLI under the shared baseline without entering the root graph. |
| `tools/protoc-gen-connect-go` module | `go 1.27.0` | Executes the pinned Connect generator under the shared baseline. |
| `tools/protoc-gen-go` module | `go 1.27.0` | Executes the pinned protobuf generator under the shared baseline. |
| `tools/wails` module | `go 1.27.0` | Executes the pinned Wails CLI under the shared baseline. |
| GitHub Actions | `1.27.x` | Selects the newest stable patch in the accepted release line. |
| macOS release preflight | `go1.27.<patch>` | Accepts stable Go 1.27 patch versions and rejects every other version form. |
| Active fixtures and assertions | `go 1.27.0` | Exercise the same module declaration emitted by active modules. |
| Active guidance and governance | Go 1.27 | Communicates the supported release family to contributors and reviewers. |

## Invariants

- Dependency and tool versions remain exactly as pinned before the upgrade.
- Tool modules remain isolated and independently reproducible.
- Root module maintenance cannot acquire tool-only requirements or sums.
- Generated source and protobuf schema revisions remain unchanged.
- Completed specs, rollback records, dated evidence, and context journals retain their historical Go versions.

## Verification contract

- All five module files expose `go 1.27.0` and tidy cleanly with Go 1.27.
- Active non-historical version scans find no Go 1.26 requirement.
- Module isolation, Wails contract/cutover, formatting, vet, unit, race, generation, reproducibility, and package checks pass.
- The final diff contains no unrelated dependency or generated-source changes.
