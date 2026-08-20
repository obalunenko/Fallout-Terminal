# Implementation Plan: Frontend Consolidation

## Summary

Consolidate the two browser applications into one npm workspace rooted at `frontend/`, with the public player interface in `frontend/client/` and the trusted desktop interface in `frontend/overseer/`. Move each application's source, generated inputs, static assets, and output marker intact; rename the privileged entry assets and immediate Wails host/build terminology from master to Overseer; then update every active producer and consumer of the old paths. The native process will continue embedding two distinct production filesystems, so the directory consolidation does not weaken the public/private capability boundary.

## Project Structure

```text
frontend/
├── package.json                 # workspace commands and shared dependency install
├── package-lock.json            # single pinned frontend dependency graph
├── client/
│   ├── package.json
│   ├── vite.config.js
│   ├── index.html
│   ├── client.js
│   ├── client.css
│   ├── sound.js
│   ├── gen/                     # generated public player ECMAScript contracts
│   ├── fonts/
│   ├── sounds/
│   └── dist/.keep
└── overseer/
    ├── package.json
    ├── vite.config.js
    ├── src/
    │   ├── index.html
    │   ├── overseer.js
    │   ├── overseer.css
    │   ├── desktop-api.js
    │   └── Fixedsys.ttf
    ├── bindings/                # generated private Wails bindings
    └── dist/.keep

main.go                          # embeds the two nested production outputs
internal/buildtool/              # owns install → generate → client → bindings → Overseer ordering
proto/buf.gen.es.yaml            # writes player code into frontend/client/gen
scripts/                         # path-aware generation, reproducibility, security, and package checks
tests/browser/                   # fixtures and journeys consume the nested apps
.github/workflows/               # clean install/generation/build gates use the workspace lock
.specify/                        # current governance and templates describe the accepted layout
```

**Structure Decision**: Use one dependency workspace with two role-owned application directories; do not introduce shared runtime modules or a combined production bundle during this structural cutover.

## Constitution Check

| Principle | Assessment |
|---|---|
| Project Identity | PASS — the accepted architecture remains a Wails-hosted private UI plus a separately served browser client; current wording and paths will be amended to the new names. |
| I. Govern the Accepted Desktop Runtime | PASS — `frontend/overseer` retains the only generated Wails bridge and `frontend/client` remains free of native capabilities. |
| II. Make Protobuf the Application Contract Source of Truth | PASS — schemas are unchanged and generated ECMAScript output moves without manual edits. |
| III. Use ConnectRPC and Keep State Server-Authoritative | PASS — client RPC behavior and authoritative state ownership are unchanged. |
| IV. Separate Public and Private Capabilities | PASS — two source roots and two embedded filesystems preserve the trust boundary explicitly. |
| V. Evolve Schemas Safely and Reproducibly | PASS — no schema change; generator destinations and drift gates move together. |
| VI. Preserve Portable Session JSON Version 1 | PASS — persistence and adapters are untouched. |
| VII. Complete Cutovers and Remove Superseded Protocols | PASS — old active directories and paths are removed in one cutover with no compatibility alias. |
| Dependency Rules | PASS — dependency versions remain pinned in one committed npm workspace lock; no production dependency is added. |
| Secret and Credential Governance | PASS — only paths and Overseer identity change; leak checks follow the moved trusted sources. |
| Go Development Tool Modules | PASS — the existing dependency-free Go build owner remains authoritative and invokes pinned tools exactly as before. |
| Testing and Quality Gates | PASS — focused path contracts, clean builds, deterministic generation, Go tests, and browser regressions are updated and rerun. |

**Post-design re-check**: PASS. The layout contract retains distinct runtime artifacts, the data model introduces no application state, and every moved generated/output owner has an explicit verification path.
