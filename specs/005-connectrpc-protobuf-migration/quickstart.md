# ConnectRPC Migration Quickstart

This is the operational path after the rollback-free cutover. There is one
public player protocol: generated ConnectRPC. Do not restore or launch a legacy
WebSocket/JSON player server as a fallback.

## Clean checkout

From the repository root with Go 1.26.x and Node 20.19+:

```bash
npm ci --prefix frontend
npm ci --prefix client
npm ci --prefix tests/browser
scripts/proto-check.sh
scripts/proto-breaking.sh --all-fixtures
go test ./...
npm run build --prefix frontend
npm run build --prefix client
npm test --prefix tests/browser
```

`proto-check.sh` formats/lints the schemas, generates Go and ECMAScript twice,
compares artifact hashes, checks generated headers/import isolation, compiles
adapters, and builds the public bundle. A reviewed schema edit must update
`proto/schema-revision.txt`; compatible history remains protected by
`proto/compatibility-baseline.binpb`.

## Local operation

Run the whole development application from the repository root:

```bash
wails dev
```

Both `wails dev` and `wails build` run the pinned full protobuf generation
before Vite: they synchronize the derived schema revision and regenerate the
Go and public ECMAScript trees. A Wails invocation therefore cannot silently
continue with stale generated contracts after a schema edit.

The native master UI and player HTTP/Connect server belong to the same process.
Players open the displayed port-3690 URL. Static files, sounds, and all six
generated procedures use that same origin.

## Protected ngrok operation

With an ngrok account token and the reserved domain configured:

```bash
NGROK_ENABLED=1 \
NGROK_USERNAME=players \
NGROK_PASSWORD='<8-to-128-character-password>' \
wails dev
```

The tunnel starts only with a complete valid credential pair. One fail-closed
Basic Auth traffic policy protects both static resources and Connect calls.
Credentials remain in process memory and a private short-lived policy file;
they are absent from schemas exposed to the browser, native status, sessions,
player configs, URLs, and diagnostics.

## Offline package

The unsigned personal smoke build is:

```bash
wails build -clean -platform darwin/arm64
```

The release entry point is `scripts/build-macos.sh`; it installs both locked
frontend dependency sets, verifies contracts, builds both applications, embeds
only `frontend/dist` and `client/dist` (including fonts and all sound folders),
then performs signing/notarization gates. A packaged player needs no CDN,
development server, Go, Node, or generation tool at runtime.

## Troubleshooting and rollback-free recovery

- Revision mismatch: review the `.proto` edit, update the revision hash, and
  regenerate. Never edit generated files.
- Breaking check failure: make the edit compatible or introduce a deliberate
  new versioned package. Do not overwrite the baseline to silence the check.
- Browser graph failure: remove private/persistence/config imports from the
  public schema or player bundle.
- Port 3690 busy: stop the known old app/process and restart the one root
  command; do not start a second protocol server on another route.
- Public tunnel failure: keep using the reported local address, correct ngrok
  credentials/domain, then restart. Do not bypass Basic Auth.
- Client incompatibility after deployment: deploy the matching app build and
  generated static bundle together. There is no supported dual-stack rollback;
  repository rollback means reverting the complete feature commit and package,
  not re-enabling a legacy endpoint inside the current build.
