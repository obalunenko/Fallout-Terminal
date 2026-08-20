# Data Model: Frontend Workspace Ownership

This refactor changes repository ownership metadata and artifact locations. It does not add or alter runtime application state, persisted session fields, or wire messages.

## Frontend Workspace

- **Identity**: the top-level `frontend` directory.
- **Attributes**: one private workspace manifest, one committed lockfile, two application members.
- **Relationships**: owns exactly one Client Application and exactly one Overseer Application.
- **Validation rules**:
  - both members must be installable from the workspace root;
  - dependency versions remain exact and reproducible;
  - no runnable frontend package exists outside this workspace.

## Client Application

- **Identity**: `frontend/client`.
- **Owned inputs**: HTML, client behavior, sound behavior, styling, font and sound resources, public generated ECMAScript contracts, local build configuration.
- **Owned output**: `frontend/client/dist`.
- **Relationships**: the player HTTP server embeds and serves only this output.
- **Validation rules**:
  - must not import Wails bindings or private desktop capabilities;
  - generated code may contain only the public player protocol tree;
  - build output preserves a committed clean-checkout marker.

## Overseer Application

- **Identity**: `frontend/overseer`.
- **Owned inputs**: private desktop HTML, `overseer.js`, `overseer.css`, desktop API facade, font resource, generated Wails bindings, local build configuration.
- **Owned output**: `frontend/overseer/dist`.
- **Relationships**: the Wails host embeds this output for the private desktop window.
- **Validation rules**:
  - privileged calls pass only through the existing narrow desktop facade and generated bindings;
  - active filenames and presentation identify the role as Overseer;
  - build output preserves a committed clean-checkout marker.

## Build Artifact Transition

```text
old source/output                    new source/output
client/*                          → frontend/client/*
frontend/src/master.js            → frontend/overseer/src/overseer.js
frontend/src/master.css           → frontend/overseer/src/overseer.css
frontend/src/{other assets}       → frontend/overseer/src/{other assets}
frontend/bindings                 → frontend/overseer/bindings
frontend/dist                     → frontend/overseer/dist
```

The transition is atomic at repository level: old locations have no retained alias or fallback state.
