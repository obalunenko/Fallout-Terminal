# Research: Go 1.27 Upgrade

## Decision 1: Use Go 1.27.0 as the module baseline

**Decision**: Set the root and all isolated tool module `go` directives to `1.27.0`.

**Rationale**: The repository's active local toolchain reports `go1.27.0`, and a full patch-form directive expresses the precise stable baseline requested while permitting later Go 1.27 patch toolchains. Applying the same baseline to every module prevents language-version drift between application and tool execution.

**Alternatives considered**: Retaining `1.26.6` in the root or `1.26` in tool modules would not complete the requested upgrade. Using only `1.27` would identify the release family but lose the repository's explicit stable patch baseline.

## Decision 2: Keep CI patch-floating within Go 1.27

**Decision**: Configure CI with `1.27.x` and require the `go1.27.*` stable version pattern in release preflight.

**Rationale**: CI should automatically receive security and bug-fix patches within the accepted release line, while the release preflight must reject release candidates, development builds, older lines, and future unqualified lines.

**Alternatives considered**: Pinning CI to `1.27.0` would miss later patch fixes. Accepting any `go1.27` prefix could accidentally admit prerelease labels.

## Decision 3: Update active fixtures and governance, not historical evidence

**Decision**: Replace Go 1.26 references in active modules, automation, tests, README guidance, the planning template, and the constitution; preserve completed specs, rollback records, quickstarts that capture dated evidence, and their context journals.

**Rationale**: Active owners must agree on the current requirement, while the constitution explicitly requires historical migration evidence to retain the target that was true when recorded.

**Alternatives considered**: A repository-wide textual replacement would falsify historical records. Updating modules alone would leave CI, preflight, fixtures, tests, and contributor guidance inconsistent.

## Decision 4: Preserve dependency and generated-output state

**Decision**: Run Go 1.27 module maintenance for each module, accept only deterministic directive/layout changes, and reject unrelated dependency, checksum, or generated-source drift.

**Rationale**: This feature changes the language/toolchain baseline, not dependency versions or application contracts. Existing module-isolation, generation, reproducibility, and package checks provide direct compatibility evidence.

**Alternatives considered**: Combining dependency upgrades with the toolchain change would broaden risk and obscure the cause of failures. Skipping module maintenance would leave reproducibility under the new toolchain unverified.
