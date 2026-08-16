# Quickstart: Verify CRT Rendering and Motion Effects

## Prerequisites

- Go 1.26 toolchain available to the repository.
- Node.js 20.19+.
- Current branch: `008-crt-rendering`.
- Use repository-owned commands; do not start a separate frontend or player server.

## 1. Build the player assets

```bash
npm ci --prefix client
npm run build --prefix client
```

Expected:

- Vite builds `client/dist/` with the player bundle, font, and sounds.
- No dependency version, generated RPC contract, or content-policy source changes.

## 2. Run focused asset checks

```bash
go test ./internal/platform -run 'CRT|Player.*Asset'
```

Expected:

- Required CRT shell, exact flicker declarations, pointer-transparent decorative layers, reveal controller, and consumed-key guard are protected.
- No reduced-motion override is present.
- Existing CSP, safe authored-content construction, and capability-separation checks remain intact.

## 3. Run the focused browser journey

```bash
npm ci --prefix tests/browser
npm test --prefix tests/browser -- crt-rendering.spec.mjs
```

Expected CRT and snapshot coverage:

- The screen, scanlines, and vignette align and overlays do not intercept controls.
- Historical selection, focus, active, and hacking-hover states match approved snapshots.
- Browser animation data exposes the exact six-second flicker checkpoints and hard-step indicators.
- No reduced-motion emulation changes the persistent effects.

The focused journey drives the browser-local fixture routes at
`/__fixture/local/crt/{state}`, where `state` is `content`, `unchanged`,
`replacement`, `waiting`, `hacking`, `hacking-unchanged`,
`hacking-replacement`, or `blocked`. It also arms the next delimiter-pattern
outcome through `POST /__fixture/local/crt/hacking-dud/{position}`, where
`position` is `revealed` or `pending`; this control guarantees dud removal
without changing the authoritative hacking generation.

Expected reveal and keyboard coverage:

- A 25-row list reveals in source order at the historical 40ms cadence and completes within 1.2 seconds.
- Pressing any key mid-reveal shows all remaining rows within 100ms.
- That physical key and its repeats cause zero navigation, activation, paging, back, or hacking actions.
- A later physical key press performs its normal action.
- Replacement cancels stale append work; unchanged updates, pagination, resize, and font fitting do not replay reveal.
- A newly opened content identity starts a reveal independently of a previous skip.
- A new hacking generation initially exposes fewer than all 32 complete code rows, then appends one address-and-code row per 40ms in deterministic DOM source order.
- Only complete revealed hacking rows enter the interaction surface; no hidden or partial code target can receive pointer, focus, or keyboard interaction.
- Same-generation hacking updates, reconnect, viewport changes, and fit work preserve the active or completed row DOM without replay.
- Revealed-row dud removal changes only the authoritative candidate row content, keeps every hacking row DOM node connected at the same index, clears the used pattern highlight, adds the dud-removal log entry, and starts no new reveal.
- Pending-row dud removal updates the queued row descriptor without exposing its cells early; existing rows remain connected and subsequent rows continue at the original 40ms cadence until all 32 are visible.
- A replacement hacking generation disconnects every old row, cancels stale callbacks, and starts exactly one fresh reveal.
- A key pressed during the hacking reveal completes all 32 rows within 100ms and is consumed before typed input, guessing, pattern activation, or navigation; a later key works normally.

### Deterministic dud-removal observation

Run the revealed-row case after all 32 rows are visible. Before activating a
highlighted delimiter pattern, retain the row objects and observe the board:

```js
window.rowsBeforeDud = [...document.querySelectorAll('#hackColumns .hack-row')];
window.removedHackRows = 0;
window.dudObserver = new MutationObserver(records => {
  for (const record of records) {
    window.removedHackRows += [...record.removedNodes]
      .filter(node => node.nodeType === Node.ELEMENT_NODE &&
        (node.matches?.('.hack-row') || node.querySelector?.('.hack-row'))).length;
  }
});
window.dudObserver.observe(document.querySelector('#hackColumns'), {
  childList: true,
  subtree: true,
});
```

Arm `hacking-dud/revealed`, activate the highlighted pattern, and verify the
log contains `Ложное слово удалено.`. Then inspect:

```js
({
  removedHackRows,
  sameRows: [...document.querySelectorAll('#hackColumns .hack-row')]
    .every((row, index) => row === rowsBeforeDud[index]),
})
```

Expected: `removedHackRows` is `0`, `sameRows` is `true`, the affected word is
rendered as dots, the used delimiter no longer highlights, and a later typed
key appears normally in the hacking input preview.

For the pending-row case, start a fresh `hacking` fixture and capture the
currently visible row objects while the count is between 1 and 31. Arm
`hacking-dud/pending`, activate a visible highlighted pattern, and verify:

1. Every captured row stays connected at the same index and zero rows are removed.
2. The affected pending candidate is absent from the DOM until its existing row reaches its original reveal turn.
3. New rows continue arriving at the existing cadence rather than appearing in a replacement burst.
4. The final board reaches 32 rows, contains the dotted candidate, records one pattern activation, and accepts normal later input.

Finally trigger `hacking-replacement` mid-reveal. Unlike dud removal, the new
generation must disconnect all prior row objects, prevent stale rows from
returning, and begin exactly one fresh ordered reveal.

Expected safety coverage:

- Markup-like authored content remains literal text and creates no injected element.
- Simulated audio discovery/playback failure does not interrupt rendering, reveal completion, or input.
- Supported states remain usable without page-level scrolling at 360×640, 768×720, and 1440×900.

Only update snapshots for an intentional, reviewed historical-baseline change:

```bash
npm test --prefix tests/browser -- crt-rendering.spec.mjs --update-snapshots
```

Review every changed image before accepting it.

Approved macOS baselines live in
`tests/browser/crt-rendering.spec.mjs-snapshots/` and use Playwright's
`<viewport>-<state>-darwin.png` naming, covering `focused-character-option`,
`active-character-option`, `selected-terminal-row`, and `hacking-hover` for
the compact, medium, and large viewports.

## 4. Run the complete browser regression suite

```bash
npm test --prefix tests/browser
```

Expected:

- The new CRT journey and every existing player, desktop, and public-access browser journey pass.
- A focused CRT run is development feedback, not the final browser regression gate.

## 5. Run repository checks

```bash
gofmt -l .
go vet ./...
go test ./...
go tool -modfile=tools/wails/go.mod wails3 generate bindings -clean ./...
git diff --exit-code -- frontend/bindings
```

Expected:

- `gofmt -l .` prints no Go source paths.
- Vet and all Go tests succeed.
- Binding generation produces no unexplained drift.

## 6. Exercise the current runtime

```bash
go run ./cmd/build dev
```

Publish content containing a 25-row folder, an empty folder, a multiline record, multiline command output, markup-like authored text, and a hacking-enabled terminal. In the player browser verify:

1. Connection, idle, character selection, assigned waiting, list, record, command, hacking, and blocked states retain one CRT shell.
2. Scanlines and vignette never intercept pointer or keyboard controls.
3. New list, record, and command identities reveal once at the historical cadence.
4. Enter a fresh hacking generation: the 32 complete address-and-code rows appear one at a time at the same 40ms cadence instead of the full grid appearing at once.
5. During that reveal, verify only visible complete rows can be focused or clicked; trigger an unchanged hacking update, reconnect, and resize, and confirm existing rows stay in place while the reveal continues or remains complete.
6. Replace the hacking generation mid-reveal and confirm no old row reappears after the fresh board begins.
7. Press a key during each normal-content and hacking reveal: the full visible page or board appears immediately and that key performs no normal or hacking action.
8. Hold the consumed key: repeats do nothing until release; press a later key and confirm its normal action works.
9. Change pages and resize the viewport: already-opened content does not replay.
10. Confirm flicker, cursor blink, pending pulse, scanlines, and vignette continue after every reveal skip.
11. Confirm there is no player setting or operating-system preference path that disables those persistent effects.

Record unavailable manual checks honestly; do not infer them from automation.

## 7. Build the accepted runtime

```bash
go run ./cmd/build build
```

Expected:

- The player and master production builds complete.
- The unsigned application embeds the verified CRT player assets.

Race, packaging, signing, notarization, and real external-provider checks remain conditional repository gates and are not feature-specific to this browser-only presentation change.

## Validation record — 2026-08-17

- `npm run build --prefix client` passed and produced the production player bundle.
- `go test ./internal/platform -run 'CRT|Player.*Asset'` passed, including the generation-only identity and same-generation row-reconciliation contract.
- `npm test --prefix tests/browser -- crt-rendering.spec.mjs` passed all 25 focused CRT tests. The interactive Playwright journey activated deterministic delimiter patterns for both revealed and pending dud targets, verified stable row identity and cadence, exercised normal later input, and retained the replacement-generation regression.
- `gofmt -l .` produced no paths; `go vet ./...` and `go test ./...` passed.
- `npm test --prefix tests/browser` passed 63 tests; the two real authenticated ngrok checks were skipped because that conditional external provider was not enabled.
- `go tool -modfile=tools/wails/go.mod wails3 generate bindings -clean ./...` completed, and `git diff --exit-code -- frontend/bindings` confirmed zero generated-binding drift.
- `go run ./cmd/build build` passed and produced the unsigned macOS application at `build/bin/Fallout Terminal.app`.

FR-018, FR-019, FR-022, SC-010, and SC-011 are covered by the focused asset and browser checks above. No separate signed, notarized, or real-provider run was performed; those remain conditional repository gates.
