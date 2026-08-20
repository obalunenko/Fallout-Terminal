# Frontend Layout and Command Contract

## Canonical identifiers

- Parent ownership directory: `frontend`
- Player application role: `client`
- Privileged application role: `overseer`
- Product terminology: `Game master - это Overseer (смотритель)`

## Required application roots

| Role | Source/package root | Generated input | Production output |
|---|---|---|---|
| Client | `frontend/client` | `frontend/client/gen` | `frontend/client/dist` |
| Overseer | `frontend/overseer` | `frontend/overseer/bindings` | `frontend/overseer/dist` |

The top-level `client` path, `frontend/src`, `frontend/bindings`, and `frontend/dist` are invalid active application locations after cutover.

## Required privileged entry assets

- `frontend/overseer/src/index.html`
- `frontend/overseer/src/overseer.js`
- `frontend/overseer/src/overseer.css`
- `frontend/overseer/src/desktop-api.js`

`index.html` must load `overseer.css` and `overseer.js` and present the trusted application as Overseer.

## Workspace commands

The workspace root must provide commands that support these repository workflows without changing directory manually:

- install all locked frontend dependencies from `frontend`;
- generate the player ECMAScript contracts into `frontend/client/gen`;
- build the client independently;
- build the Overseer independently;
- build both applications in the required preparation flow;
- run the focused Overseer browser regression subset.

The Go-owned build graph remains the canonical owner of preparation, native compilation, and packaging order.

## Embedding contract

- The native Overseer window receives only the filesystem rooted at `frontend/overseer/dist`.
- The player server receives only the filesystem rooted at `frontend/client/dist`.
- The two embed declarations and subtree selections remain distinct and are covered by static resource tests.

## Cutover contract

- Generation must write only to the new locations.
- CI and reproducibility checks must diff and digest only the new locations.
- Current documentation, governance, templates, test fixtures, and security scans must name the new locations.
- Completed historical specifications and rollback records retain their original terminology and paths.
- No symlink, duplicate source tree, fallback path, or compatibility build command may preserve the old active layout.
