# Contract: Private Desktop Events and Readiness

## Event Inventory

Exactly four application events cross the Wails bridge. Names and JavaScript-visible payloads remain unchanged.

| Event name | Go producer | Facade | Exact native payload | Private semantic message |
|---|---|---|---|---|
| `server-info` | lifecycle/player/tunnel status adapter | `onServerInfo(callback)` | Safe server object with established `ip`, `port`, `url`, `localUrl`, `tunnel`, and `tunnelError` meanings | `ServerInformationEvent` |
| `client-count` | player active-stream callback | `onClientCount(callback)` | Nonnegative integer active public-stream count | `ClientCountEvent` |
| `hack-state` | canonical live/hack publication | `onHackState(callback)` | Detached public hacking projection or `null` | `HackStateEvent` |
| `coordination-state` | canonical coordination publication | `onCoordinationState(callback)` | Detached private master coordination projection or `null` | `CoordinationStateEvent` |

The existing private protobuf adapters construct the semantic event and return the current native value. No protobuf bytes, ProtoJSON, generic event envelope, credentials, secret hacking fields, or future outcomes cross Wails.

## Registration and Emission

- Declare/register the four typed event names where Wails v3 supports generator-visible typed registration.
- `internal/platform` owns the injected event sink over `application.App.Event.Emit`.
- No event adapter retains a bounded startup context. Emission uses the application-lifetime capability.
- Initial local `server-info` publication remains a startup gate; later update/replay emission behavior retains current safe failure handling.
- The frontend imports the exact pinned `@wailsio/runtime` event API only inside `desktop-api.js`.
- Wails v3 `Events.On` delivers an event object. The facade extracts `.data` and passes only the established payload to authored callbacks.

## Race-Safe Initialization Protocol

Initialization is one coordinated barrier, not four independently racing bootstraps:

1. Resolve the generated desktop service and Wails event runtime. Failure is actionable initialization failure.
2. Register all four underlying listeners before starting or resolving the status snapshot.
3. Start/reuse one cached `GetRuntimeStatus` promise.
4. Track `eventReceived` independently for `serverInfo`, `clientCount`, `hackState`, and `coordinationState`.
5. An event marks only its own field newer and is delivered immediately while active.
6. When the snapshot resolves, apply each field only when that field has not observed a newer event.
7. Preserve current server normalization: ignore an unusable null server value, normalize keys, and retain a known local URL when public/tunnel status replaces the visible URL.
8. Render the unchanged optional `startupError` as actionable master state. Do not invent an event name or serialized phase.
9. Mark desktop readiness only after generated calls are usable, all four listener registrations succeeded, and the snapshot has either applied or yielded to newer per-field events.

This ordering must be explicit even though JavaScript promise callbacks normally run after the current module turn. A test controls promise resolution synchronously/asynchronously and proves all four registrations precede snapshot application.

## Window-Ready Semantics

A Wails v3 window-ready signal is optional. If implementation uses it:

- it gates when bindings/events may safely be used;
- it does not replace listener-first registration plus `GetRuntimeStatus`;
- it does not cause a second DOM-ready-style replay;
- tests prove ready plus snapshot/events produce no duplicate master effects.

If bindings and event registration are usable directly during module initialization, omit the extra signal. The removed v2 DomReady context callback is not recreated as a state-delivery dependency.

## Subscribe/Unsubscribe Contract

For every facade subscription:

- Reject a non-function callback with `TypeError` before registering anything.
- Return one release function.
- Keep an `active` guard. The first release sets inactive, removes its facade tracking entry, and calls the underlying Wails unsubscribe exactly once.
- Repeated release is a no-op.
- Release during a pending snapshot suppresses that snapshot callback.
- Release during callback execution prevents every later callback and remains safe.
- A newer event suppresses only the older snapshot field for that subscription/field.
- Hot replacement uses the existing disposal symbol, releases every listener from the previous facade exactly once, and does not release new-facade listeners.
- No callback is delivered after release, even if the runtime invokes a late queued handler.

## Snapshot and Event Ordering Examples

| Sequence | Required outcome |
|---|---|
| listeners → snapshot | current value delivered once per present field |
| listeners → newer server event → older snapshot | event value wins; snapshot server field is ignored; unrelated snapshot fields may initialize |
| listener → release → snapshot resolves | no callback; underlying release called once |
| listener → event callback calls release → queued event | first callback completes; queued callback is suppressed |
| hot replacement → old queued event | old facade callbacks suppressed and old underlying listeners released once |
| public tunnel event after local status | public URL shown and previous local URL retained |
| tunnel failure after local status | local URL remains usable and redacted `tunnelError` is actionable |

## Failure Behavior

- Missing generated bindings or event runtime must not silently degrade a production build to optional globals or polling.
- A transport rejection from `GetRuntimeStatus` becomes a safe failed command/status state and must not mark the UI ready with empty data.
- Application-owned startup failures remain present in runtime status and master presentation; they do not become framework-only exits.
- Event callback failures are contained so one subscriber cannot corrupt lifecycle ownership or other listener releases.

## Verification

Add direct JavaScript behavioral tests with injected generated-service/runtime doubles for:

- exact four names and exact `.data` payload unwrapping;
- all listeners registered before snapshot application;
- event-before-snapshot per-field precedence;
- null server behavior and local URL retention;
- initial client/hack/coordination values including `null`;
- unsubscribe pending, in callback, after callback, and repeated;
- prior-facade disposal and no late callbacks;
- actionable startup-error status and readiness failure;
- optional window-ready de-duplication if used.

Retain Go tests for event names, payload types, order/detachment, replay/status parity, and safe credential-free server information. Source/bundle scans reject v2 event globals and any fifth application event.
