# Wails Migration Acceptance and Rollback

This document assigns separate personal-use and public-release gates and
preserves a tested return path to the Electron release after the Wails source
cutover. The personal-use macOS candidate has the evidence required for T064;
public publication remains independently prohibited until its trust gates pass.

## Acceptance ownership

| Gate | Evidence owner | Required evidence |
| --- | --- | --- |
| Version-1 session compatibility | Application maintainer | Golden fixture round trips, malformed/unsupported rejection, unknown-field retention, and rapid-save ordering |
| Master and player parity | Game-master acceptance tester | Authoring journey, 4–7-client convergence, reconnect, hacking privacy, CRT/input behavior, and optional-audio degradation |
| Local and protected public access | Application maintainer | Local-only zero-tunnel run, invalid-credential fail-closed run, authenticated HTTP/WSS run, and owned-resource cleanup |
| Personal-use macOS application bundle | Repository owner | Clean arm64 build, ad-hoc signature, embedded-asset inventory, single-launch smoke without developer tooling, storage-path check, port-conflict error, normal-shutdown cleanup, and SHA-256 |
| Personal-use source cutover | Repository owner | Automated gates green, all P1 journeys passed, personal-use bundle accepted, rollback reference recorded, and no unresolved data-loss/security blocker |
| Optional public DMG | Public release engineer | Developer ID signature, hardened runtime, notarization acceptance, stapled ticket, Gatekeeper assessment, architecture, and SHA-256 |

The locally built/ad-hoc-signed `.app` is the accepted active profile for use on
the owner's Mac. Missing Developer ID credentials make public-release checks
`N/A (personal profile)` and do not block source cutover. The personal package
must not be published as a Developer ID release or offered to unrelated users.

## Cutover record

The immutable Electron rollback source is commit
`9b2dd022a724202f95766d8196cc3fdf88be9084` (`docs: Add specs for migrating to
go server`). No independently archived Electron binary was produced for this
checkout; that commit and its lock file are therefore the canonical rebuild
source.

The accepted Wails artifact is the final rebuild after Electron source and
dependency removal. These two fields are the release handoff identity and must
match the canonical personal-use acceptance record in
`specs/001-wails-v2-migration/quickstart.md`:

Canonical candidate commit: `118ed8199a3a0b1c3b73a09ef98908949c2e2d75`

Canonical executable SHA-256: `d1ad65f5e5a80f3471e2d551d0ca5d1e55a8d2447cef58091a37cb35276cc121`

The earlier pre-cutover package digest is historical T060 evidence only and is
not an accepted cutover or rollback decision artifact. Run the executable
consistency test before handoff; it fails on a missing, duplicate, malformed,
or conflicting canonical commit or digest:

```bash
go test ./internal/platform -run TestAcceptanceEvidenceUsesOneCanonicalPostElectronCandidate -count=1
```

The release notes or pull request must link this final record and include:

- the immutable Electron rollback tag or commit;
- the last accepted Electron artifact and its SHA-256;
- the canonical Wails candidate commit and executable SHA-256 above;
- the completed sections of
  `specs/001-wails-v2-migration/quickstart.md`;
- the location of user-owned session files used for the semantic round-trip
  comparison (never commit those files).

The Electron rollback reference must predate deletion of `main.js`,
`preload.js`, `server/`, the root Electron package files, and duplicate legacy
assets. The Wails acceptance record must instead identify the final
post-deletion rebuild; a pre-cutover Wails digest cannot be promoted at handoff.

## Roll back before source cutover

If the Wails candidate fails before T064, stop it normally, confirm port 3690
and any ngrok child are gone, and start the retained Electron oracle from the
repository root:

```bash
npm ci
npm start
```

For protected public access, use the same test-only environment variables and
the retained Electron public-mode command documented by the rollback release.
Do not run Electron and Wails simultaneously because both own port 3690 and the
same user-selected session documents.

## Roll back after source cutover

Do not reconstruct deleted Electron files manually and do not revert unrelated
work. Create a maintenance branch from the recorded immutable rollback tag or
commit, build it with its documented Node/Electron toolchain, and publish that
known artifact as the temporary rollback release. Keep the Wails line on a
separate fix-forward branch.

Version-1 session JSON is the shared durable contract. A rollback requires no
format conversion: close the current application, make a safety copy of each
user-owned session, then open the same version-1 document in the rollback
release. Never replace a newer user file with the bundled demo or a repository
fixture.

## Rollback triggers

Roll back or halt cutover for any of these conditions:

- semantic session data loss or a save revision older than the last accepted
  edit;
- private hacking fields crossing the player or desktop boundary;
- anonymous access to a public HTTP or WebSocket endpoint;
- repeatable 4–7-client divergence or reconnect puzzle regeneration;
- an installed app that requires Go, Node, npm, Wails, Vite, or a separately
  started player server;
- a personal-use artifact that fails ad-hoc signature, architecture,
  single-launch, or embedded-asset checks;
- a candidate intended for public distribution that fails Developer ID
  signing, notarization, stapling, DMG, or Gatekeeper checks;
- an orderly application quit that leaves the player listener, ngrok child, or
  credential-policy directory behind.

## Recovery verification

After switching to the rollback release:

1. Launch it once and confirm one Electron process owns the master and player
   server.
2. Open safety copies of representative version-1 sessions and compare their
   terminal trees and multiline text.
3. Connect four players, exercise navigation and one hacking puzzle, and
   reconnect one client.
4. Verify local-only mode starts no tunnel; if public access is required,
   verify anonymous HTTP/WSS is rejected.
5. Quit normally and confirm port 3690, ngrok, and temporary policy material are
   gone.

Record the rollback artifact, commit, checks, and reason. Preserve the failed
Wails artifact and redacted diagnostics for fix-forward analysis; never include
credentials or user session contents.
