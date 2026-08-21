# Feature Specification: Go 1.27 Upgrade

**Feature Branch**: `017-go-1-27-upgrade`

**Created**: 2026-08-21

**Status**: Draft

**Input**: Update the repository to Go 1.27.

**Bugfix**: 2026-08-21 — BUG-001 requires warning-free macOS Go validation through the repository-governed deployment environment.

## User Scenarios & Testing

### User Story 1 - Build with the Current Go Release (Priority: P1)

As a maintainer, I can build and test every application and repository-owned Go tool module with Go 1.27 so that local development and automation use the same supported language baseline.

**Why this priority**: A single, consistent toolchain baseline is required before any other upgrade documentation or release validation is trustworthy.

**Independent Test**: Use Go 1.27 to load, tidy-check, test, and validate the root module and each isolated tool module, then confirm that no active module still declares the previous Go release.

**Acceptance Scenarios**:

1. **Given** the repository root and isolated tool modules, **When** their version declarations are inspected, **Then** every active module requires Go 1.27.
2. **Given** Go 1.27 is active, **When** the root test and static-analysis suites run, **Then** they complete successfully without dependency-graph drift.
3. **Given** each isolated tool module, **When** its module graph is resolved with Go 1.27, **Then** it remains reproducible and does not modify the root module graph.
4. **Given** a supported macOS host and a fresh Go build cache, **When** the canonical repository test command runs, **Then** it uses the governed macOS 13 deployment environment and emits no linker warnings.

---

### User Story 2 - Keep Automation and Guidance Consistent (Priority: P2)

As a contributor or release operator, I receive one consistent Go 1.27 requirement from CI, release preflight checks, repository validation fixtures, setup guidance, and project governance.

**Why this priority**: Conflicting requirements cause avoidable local-versus-CI failures and can allow an unsupported release toolchain into packaging.

**Independent Test**: Search active configuration, scripts, tests, templates, guidance, and governance for the previous Go baseline and verify that all operational references now require Go 1.27 while historical evidence remains unchanged.

**Acceptance Scenarios**:

1. **Given** a CI run, **When** the Go setup step executes, **Then** it selects a Go 1.27 patch release.
2. **Given** a release operator runs the macOS preflight with a non-1.27 toolchain, **When** the version gate executes, **Then** it fails with a Go 1.27 requirement.
3. **Given** repository validation fixtures and their assertions, **When** they create or inspect temporary modules, **Then** they consistently expect the Go 1.27 language version.
4. **Given** a contributor reads active setup or governance material, **When** they look for the supported Go version, **Then** the stated requirement is Go 1.27.

## Edge Cases

- A Go 1.27 release candidate or development build must not satisfy the release preflight intended for stable Go 1.27 patch releases.
- Isolated tool modules must not retain an older language version after the root module is upgraded.
- Module tidy operations must not move tool-only dependencies or checksums into the root module.
- Historical specifications, migration records, and recorded build evidence must retain the versions that were true when those records were produced.
- Unrelated dependency versions and generated source must not change merely because the toolchain baseline changes.
- A raw Go command that bypasses the repository's macOS deployment environment may select an older linker target; canonical validation must not inherit that mismatch.

## Requirements

### Functional Requirements

- **FR-001**: The root application module MUST declare Go 1.27.0 as its minimum language and toolchain baseline.
- **FR-002**: Every repository-owned isolated Go tool module MUST declare Go 1.27.0 as its minimum language and toolchain baseline.
- **FR-003**: Continuous integration MUST select the latest stable Go 1.27 patch release.
- **FR-004**: The macOS release preflight MUST accept stable Go 1.27 patch releases and reject other Go release lines.
- **FR-005**: Active validation fixtures and assertions MUST create and expect Go 1.27 module declarations consistently.
- **FR-006**: Active contributor guidance, planning templates, and project governance MUST identify Go 1.27 as the supported baseline.
- **FR-007**: Historical specifications, completed migration records, and dated evidence MUST remain unchanged.
- **FR-008**: The upgrade MUST preserve existing dependency versions, generated outputs, module ownership boundaries, and application behavior except for changes produced deterministically by Go 1.27 module maintenance.
- **FR-009**: The root and isolated module graphs, repository quality checks, and application package build MUST succeed with Go 1.27.
- **FR-010**: Canonical local and CI Go validation on macOS MUST use the repository-governed macOS 13 deployment target and matching CGO flags without emitting linker target-version warnings.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A scan of active, non-historical repository files finds zero requirements for Go 1.26.
- **SC-002**: All five active Go modules declare Go 1.27.0 and resolve without cross-module ownership drift.
- **SC-003**: The repository's Go formatting, static analysis, unit, race, tool-module, contract, generation, and reproducible-build checks pass with Go 1.27.
- **SC-004**: The application package build completes successfully with Go 1.27.
- **SC-005**: The final change contains no unrelated dependency upgrade or generated-source drift.
- **SC-006**: A fresh-cache canonical Go test run on the supported macOS host emits zero `ld: warning` lines.

## Assumptions

- Go 1.27.0 is the stable patch baseline available to this repository; CI remains on the 1.27 patch line so later security and bug-fix releases are selected automatically.
- Go module declarations use the full 1.27.0 baseline, while human-facing guidance may use the Go 1.27 release family where patch-level wording is unnecessary.
- Dated historical documents and completed feature artifacts are evidence, not active setup instructions, and are intentionally excluded from the replacement scope.
- No application source or serialized contract behavior is expected to change as part of this toolchain-only upgrade.
- Local macOS validation uses the documented Make targets, or an explicitly equivalent environment, so the deployment target matches CI and packaging.
