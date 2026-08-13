# Research: Protobuf-First ConnectRPC Migration

## Decision 1: Use the Connect protocol with explicit binary protobuf in the browser

**Decision**: Use `createConnectTransport` from `@connectrpc/connect-web` and explicitly set binary protobuf for all generated browser calls. Serve the generated Connect-Go handlers through the existing same-origin `net/http` listener over HTTP/1.1 or HTTP/2 as available; do not introduce gRPC-Web, CORS, a second backend, or a client development server.

**Rationale**: Connect is designed to carry generated protobuf RPCs over ordinary HTTP, supports browser-portable server streaming, and composes with Go's standard `net/http`. Binary protobuf is smaller than ProtoJSON, preserves unknown fields without JSON-name/string-enum concerns, and makes the 4 KiB message limit directly measurable with deterministic marshaling. Explicitly setting the transport option prevents a library-default change from silently changing the public wire encoding. Official references: [Connect-Go](https://github.com/connectrpc/connect-go), [Connect-ES](https://github.com/connectrpc/connect-es), and the [Connect protocol](https://connectrpc.com/docs/protocol/).

**Alternatives considered**:

- Generated ProtoJSON: supported and curl-friendly, but rejected because binary has less framing/encoding overhead and avoids JSON-specific ambiguity for a tightly bounded embedded client released with its server.
- gRPC-Web: rejected because the Connect protocol already provides portable browser server streaming and simpler ordinary HTTP integration.
- Native gRPC: rejected because browser clients cannot directly provide the required transport semantics without another proxy or protocol.

## Decision 2: Pin a local, reproducible Buf and generator toolchain

**Decision**: Pin Buf CLI `v1.72.0`, Protocol Buffers compiler/toolchain `v35.0`, `protoc-gen-go` and `google.golang.org/protobuf` `v1.36.11`, Connect-Go and `protoc-gen-connect-go` `v1.20.0`, `@bufbuild/protobuf` and `@bufbuild/protoc-gen-es` `2.13.0`, and `@connectrpc/connect` plus `@connectrpc/connect-web` `2.1.2`. Use Go 1.26 tool directives for Buf and Go plugins, an exact client npm lockfile for ECMAScript tooling/runtime, Buf v2 generation templates, checked-in generated outputs, and clean second-generation diff checks.

**Rationale**: The current official releases support generated Go and ECMAScript from one schema while keeping every version reviewable in repository metadata. Connect-ES v2 consumes service descriptors produced by Protobuf-ES, so a separate deprecated `protoc-gen-connect-es` is unnecessary. Local pinned plugins avoid an unpinned latest BSR lookup, while checked-in output makes clean Wails development and packaged builds independent of generation-time network access. Official references: [Buf generation](https://buf.build/docs/generate/), [Buf v2 generation configuration](https://buf.build/docs/configuration/v2/buf-gen-yaml/), [Protobuf-ES manual](https://github.com/bufbuild/protobuf-es/blob/main/MANUAL.md), and [protobuf-go releases](https://github.com/protocolbuffers/protobuf-go/releases).

**Alternatives considered**:

- Unversioned remote plugins: rejected because reproducibility would depend on mutable latest versions.
- Handwritten generated files or adapters that duplicate transport messages: rejected by both schema governance and drift detection.
- Generate during every `wails dev`: rejected because normal startup must remain one command and should not require network or mutate the checkout.

## Decision 3: Split public, private, persistence, and configuration schema packages

**Decision**: Use `fallout.terminal.player.v1` for the public service and player-safe values; `fallout.terminal.private.v1` for Wails requests/results/status/events and private coordination; `fallout.terminal.persistence.v1` for known session-v1/player-config-v1 semantics; and `fallout.terminal.config.v1` for serializable application/listener/queue/timeout/path/tunnel/startup/shutdown values. Generate ECMAScript only from `player.v1`; generate Go from all packages. Public schemas import only public schemas and standard well-known types.

**Rationale**: A physical package and generation-input split makes public/private separation inspectable in descriptors, generated inputs, imports, and the final bundle. Persistence and configuration need protobuf governance but are not public capabilities, so separate private packages prevent their accidental inclusion while keeping their different compatibility and lifecycle rules visible.

**Alternatives considered**:

- One large package: rejected because any public import could drag private/native/configuration meanings into the generated player graph.
- Duplicate player-safe types in private schemas: rejected because private schemas may import public values where the same meaning is intentionally reused; the public graph never imports back.
- Treat persistence JSON or Wails native objects as ungoverned exceptions: rejected because the constitution explicitly makes their structured semantics protobuf-owned.

## Decision 4: Expose one streaming responsibility and five typed unary responsibilities

**Decision**: Define `fallout.terminal.player.v1.PlayerService` with server-streaming `Subscribe` and unary `SelectCharacter`, `Navigate`, `Guess`, `ActivatePattern`, and `SoundManifest`. `SubscriptionMessage` is a `oneof` of `PersonalizedSnapshot` and `CompoundUpdate`; every action request has its own procedure-specific message rather than a command envelope.

**Rationale**: This exactly separates authoritative delivery from browser mutations and sound discovery. Procedure-specific request descriptors provide structural validation, prevent a generic capability dispatcher, and make public enumeration a meaningful security check. The stream needs only one application message family because the snapshot/update variant and complete nested projections replace all same-revision component envelopes.

**Alternatives considered**:

- Generic `Command` RPC: rejected because it recreates the handwritten dispatcher and weakens procedure-level validation and replay fingerprints.
- Separate state streams for player, terminal, navigation, and hacking: rejected because a committed revision could fragment into several messages and violate compound-update semantics.
- Client or bidirectional streaming: rejected because browser request bodies are not portable for those modes and the feature needs only unary mutations.

## Decision 5: Register a bounded stream before capturing its snapshot

**Decision**: Resolve recognition, attach the physical stream, register its non-blocking sink, and capture the personalized snapshot under the same coordinator order. Return snapshot revision R to the handler; the handler writes it first and subsequently accepts only queued updates above R. Each physical stream queue holds `32` complete subscription messages, and overflow cancels that stream without retrying old incremental values.

**Rationale**: Register-before-capture closes the snapshot/publication gap. Buffering updates until the snapshot has physically been written preserves the first-message rule without holding the coordinator mutex on network I/O. Revision filtering removes any impossible-or-defensive duplicate at R or below. Per-stream queues preserve multi-tab equivalence while isolating blocked writers and retaining raw stream count semantics.

**Alternatives considered**:

- Capture then register: rejected because a commit between those operations can be lost.
- Register after sending the snapshot: rejected for the same gap and because network writes are unbounded relative to mutation.
- Keep an application-level incremental replay log: rejected because physical delivery is not acknowledged and recovery is explicitly a new complete snapshot.

## Decision 6: Commit one personalized compound update per affected logical session

**Decision**: Keep `control.Service.commit` as the sole mutation/revision boundary, but replace component-by-component transport effects with one transport-independent compound projection per affected logical session and revision. Components are complete values; message-field absence means unchanged; terminal presentation uses a `oneof` for complete live terminal versus explicit no-live-terminal. Enqueue every logical update before completing an accepted unary result.

**Rationale**: The existing coordinator already supplies atomic copy-on-write mutation and synchronous detached effects, but the current player server serializes several legacy envelopes for one revision. A compound value preserves the existing canonical owner while meeting exactly-once revision and personalized fanout rules. Generated protobuf remains an adapter output, not the canonical aggregate.

**Alternatives considered**:

- Publish one protobuf per changed component: rejected because subscribers could observe several application messages with the same revision.
- Make generated `CompoundUpdate` canonical state: rejected because generated boundary values must not own mutable domain behavior.
- Use ambiguous partial patches: rejected because omission must mean unchanged and every present component must be complete.

## Decision 7: Bound encoded bodies, decompressed messages, and semantic fields independently

**Decision**: Apply `connect.WithReadMaxBytes(4096)` to every public procedure so each uncompressed decoded protobuf request is at most `4 KiB`; wrap the generated RPC handler in `http.MaxBytesHandler` with an `8 KiB` encoded-body ceiling; retain Connect's bounded decompression path and test gzip bodies that expand beyond 4 KiB. Bound recognition handles, request IDs, broadcast IDs, and generation IDs to 128 UTF-8 bytes; terminal, character, node/navigation target, guess target, and `patternId` values to 256 bytes; and sound categories to the eight-value enum with a 32-byte defensive adapter bound. Minted recognition handles remain 32 lowercase hexadecimal characters, accepted handles use the opaque token alphabet `[A-Za-z0-9_-]`, and clients never parse them.

**Rationale**: Connect-Go documents `WithReadMaxBytes` as a per-message limit and recommends `http.MaxBytesHandler` for the total HTTP request stream. The two independent limits stop oversized protobufs, compressed expansion, streaming-frame abuse, and excessive encoded bodies before application adapters. Existing durable terminal/node IDs already use 256-byte validation, while current random IDs and browser UUIDs fit comfortably within 128 bytes. Official reference: [Connect-Go handler options](https://github.com/connectrpc/connect-go/blob/main/option.go).

**Alternatives considered**:

- Only `Content-Length`: rejected because it is optional, describes encoded bytes, and does not bound decompression or streamed reads.
- Only protobuf field validation: rejected because unknown fields and malformed/oversized bodies must be rejected before service invocation.
- Disable compression: rejected because the public boundary must remain safe even for crafted compressed requests and Connect already supplies bounded decompression semantics.

## Decision 8: Fingerprint deterministic protobuf payloads and retain bounded replay only

**Decision**: Keep request-result records per logical session and broadcast at the default limit `256`. Fingerprint the fully qualified procedure plus deterministic binary marshaling of the validated procedure-specific request payload excluding transport-only recognition and the request ID itself. An exact retained identity replays the original result/revision; a retained identity with another procedure or fingerprint returns `duplicate`; deterministic eviction remains internal and post-loss reuse is evaluated as new.

**Rationale**: Generated messages eliminate handwritten field concatenation and deterministic protobuf bytes capture variants and future compatible known fields without inventing a generic command. Procedure inclusion prevents cross-RPC identity reuse. Preserving the bounded cache and its limited guarantee matches existing feature-004 behavior without promising delivery acknowledgement or durable exactly-once semantics.

**Alternatives considered**:

- Hash raw HTTP bytes: rejected because equivalent encodings, compression, and unknown-field order are transport details and may leak raw request material into diagnostics.
- Persist replay data: rejected because sessions and replay guarantees end with process state.
- Promise exactly once after eviction: rejected because no matching record remains to distinguish retry from a new request.

## Decision 9: Preserve Wails native objects and JSON codecs through explicit exhaustive adapters

**Decision**: Define private protobuf messages for every current Wails request/result/status/event, but retain exact named App methods, `desktop-api.js` functions, event names, and native JavaScript object shapes. Add explicit compatibility ↔ generated-private ↔ domain adapters and descriptor-driven tests that require every field and `oneof` variant to be mapped. Define persistence messages for every known version-1 field, but keep the existing custom session JSON and strict player-config JSON encoders/decoders as the only file codecs.

**Rationale**: Wails and user-owned JSON have established transport/representation contracts that protobuf must govern semantically without replacing. Explicit adapters make schema drift fail tests, preserve recursive session unknown fields and strict player-config rejection, and prevent generated values from becoming another mutable model.

**Alternatives considered**:

- Send protobuf binary/Base64/ProtoJSON through Wails: rejected because it changes native object behavior and creates a serialized envelope.
- Replace portable JSON with generic ProtoJSON: rejected because field names, unknown-field behavior, formatting, validation, relative paths, and atomic saves would change.
- Leave compatibility mapping implicit in reflection: rejected because private capability and shape changes must fail exhaustiveness verification.

## Decision 10: Prove a vertical slice, then remove WebSocket completely

**Decision**: Implement generated subscription/snapshot, selection, local same-origin use, authenticated ngrok, Basic Auth failure, and packaged assets as the first reviewed slice. Permit side-by-side transport only on `005-connectrpc-protobuf-migration` while parity tests are constructed. After all procedures and projections pass, remove the WebSocket route/upgrade, browser constructor, JSON decoder/envelopes, direct dependency, fixtures, CSP allowances, sound-list endpoint, and active protocol documentation in one final cutover.

**Rationale**: The vertical slice tests the highest-risk generation, browser streaming, listener, auth, and packaging assumptions before bulk conversion. A bounded branch-only coexistence window provides a parity oracle without leaving two protocols whose authorization and security behavior can drift.

**Alternatives considered**:

- Big-bang replacement before a packaged proof: rejected because generation/bundling and public ngrok behavior would be discovered too late.
- Permanent fallback WebSocket: rejected because the constitution and feature require exactly one public protocol.
- Separate Connect listener: rejected because it would complicate same-origin, Basic Auth, port, tunnel, and shutdown behavior.

