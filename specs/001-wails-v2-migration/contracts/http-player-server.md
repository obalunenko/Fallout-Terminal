# Contract: Player HTTP Server

## Listener

- Default address: `0.0.0.0:3690`.
- Startup completes only after the listener is acquired.
- Port-in-use and bind errors become visible desktop startup failures and leave no partially owned server.
- The displayed local URL uses the first non-internal IPv4 address, falling back to `localhost`.

## Static player assets

`GET /` serves `client/index.html`. Other existing asset paths under `/` serve only files embedded from `client/`, including CSS, JavaScript, fonts, and sounds. Directory traversal and arbitrary filesystem access are impossible because production assets come from an embedded filesystem.

Recommended headers:

- Correct content type by extension.
- `X-Content-Type-Options: nosniff`.
- A player CSP permitting same-origin HTTP/WebSocket assets and browser audio while disallowing object embedding.
- No directory listings.

## Sound discovery

`GET /api/sounds/{folder}`:

- `{folder}` must be one of `ambient`, `hack-good`, `hack-bad`, `menu-focus`, `single`, `multiple`, `enter`, `charscroll`.
- Results contain filenames only for `.mp3`, `.wav`, `.ogg`, `.m4a`, or `.webm` files in that embedded category.
- Unknown categories, missing categories, or read errors return HTTP 200 with `[]`.
- Results never contain paths outside the selected category.

## WebSocket upgrade

A valid WebSocket upgrade on `/` registers one PlayerConnection. Same-origin local and ngrok origins are accepted according to the request host; arbitrary cross-origin browser origins are rejected. The server immediately sends the current `TERMINAL_LIVE` snapshot when one exists.

## Shutdown

Shutdown stops accepting requests, cancels all player connections, waits for reader/writer goroutines, and completes within the configured timeout. Repeated shutdown calls are safe.

