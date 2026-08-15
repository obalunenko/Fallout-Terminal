# Feature Specification: Embedded ngrok Public Access

**Feature Directory**: `007-embedded-ngrok-access`  
**Created**: 2026-08-15  
**Status**: Approved  
**Purpose**: Let a game master securely publish the existing player experience from the packaged
desktop application through the game master's own ngrok account.

## Clarifications

### Session 2026-08-15

- Q: Which development/test override remains after removal of the external ngrok CLI production
  path? → A: Retain one narrow settings/temporary-credentials injection seam that uses the same
  embedded provider path, is unavailable to packaged UX, and cannot start an external ngrok
  process.
- Q: Which player username is prefilled during initial public-access setup? → A: `players`.
- Q: What policy applies to a manually entered player password? → A: At least 8 characters with
  no character-class composition rules.
- Q: How does the application limit repeated invalid Basic Auth attempts? → A: It adds no
  application-level rate limit or account lockout and leaves any upstream throttling to ngrok.

## User Scenarios & Testing

### User Story 1 - Save public-access settings securely (Priority: P1)

A game master opens public-access settings, enters an ngrok account token once, optionally chooses
a reserved domain, keeps or changes the default `players` username, enters a password or generates
a strong password, and saves the settings. After saving, the application shows only whether each
secret exists and never reveals the stored value.

**Why this priority**: Secure credential setup is required before any public endpoint can be made
available.

**Independent Test**: From a newly installed packaged application, save settings, close and reopen
the application, and confirm that non-secret preferences and secret-presence indicators remain
available while both secret values remain unreadable.

**Acceptance Scenarios**:

1. **Given** no provider token is saved, **When** the game master enters a token and saves, **Then**
   the token field is cleared and the UI reports that a token is present without displaying it.
2. **Given** no player password is saved, **When** the game master enters a password and saves,
   **Then** the password field is cleared and the UI reports that a password is present without
   displaying it.
3. **Given** the game master requests a generated password, **When** the password is created,
   **Then** it is displayed once with an explicit Copy action and cannot be retrieved after that
   creation flow ends.
4. **Given** settings have been saved, **When** the packaged application is relaunched by double
   click, **Then** the optional domain, username, preferences, and secret-presence indicators are
   restored without requiring a Terminal session.
5. **Given** secure credential storage is locked, denied, or unavailable, **When** the game master
   saves or uses a secret, **Then** the operation fails with a clear redacted message and no
   plaintext fallback is created.
6. **Given** a manually entered password shorter than eight characters, **When** the game master
   tries to save it, **Then** the UI rejects it without requiring any specific character classes.

---

### User Story 2 - Start and share protected public access (Priority: P1)

A game master starts public access from the running application's UI without restarting it, follows
visible startup and readiness states, and receives an HTTPS address that can be shared with players.
The game master can copy the address and non-secret login name; a password is copyable only during
its entry or one-time generation flow.

**Why this priority**: Starting and sharing a protected endpoint is the feature's primary user
value.

**Independent Test**: Configure valid credentials, start public access from the UI, observe the
state become ready, and verify that the copied HTTPS address requires the configured player login
before returning any player resource.

**Acceptance Scenarios**:

1. **Given** valid saved credentials and no reserved domain, **When** the game master starts public
   access, **Then** the UI moves through starting to ready and displays a provider-assigned HTTPS
   address.
2. **Given** valid saved credentials and an accessible reserved domain, **When** the game master
   starts public access, **Then** the ready address uses that domain.
3. **Given** a reserved domain that is unavailable or not owned by the account, **When** startup is
   attempted, **Then** the UI shows a clear redacted domain error and does not publish a URL.
4. **Given** startup is still in progress, **When** a request arrives for the prospective external
   host, **Then** no public resource or player request is accepted before authorization is active.
5. **Given** public access is ready, **When** the game master copies sharing information, **Then**
   the address and username are available while an already stored password is not reconstructed.

---

### User Story 3 - Play the full player experience remotely (Priority: P1)

A player opens the public HTTPS address, completes Basic Auth, and uses the same authoritative
player experience available locally, including live streaming, reconnect, character selection,
navigation, hacking, and sound.

**Why this priority**: A public address has no value unless it preserves the complete, existing
player journey.

**Independent Test**: Connect a browser through the public address, authenticate, exercise every
existing player journey while receiving non-empty live updates, interrupt the connection, and
confirm that the browser reconnects to authoritative state.

**Acceptance Scenarios**:

1. **Given** a ready public endpoint, **When** a player requests any static resource or player
   operation without valid Basic Auth, **Then** access is rejected and no protected content is
   returned.
2. **Given** valid Basic Auth, **When** a player opens the public address, **Then** the same player
   interface and authoritative state available locally are shown.
3. **Given** an authenticated non-empty `Subscribe` stream, **When** updates continue over time,
   **Then** they arrive incrementally without waiting for the stream to end.
4. **Given** an authenticated player loses connectivity, **When** connectivity returns, **Then**
   the player reconnects and converges on current authoritative state.
5. **Given** correct credentials but an unknown external `Host`, **When** a request arrives,
   **Then** it is rejected fail-closed.
6. **Given** repeated invalid Basic Auth attempts, **When** a later request supplies the correct
   credentials, **Then** the application accepts it without an application-imposed lockout unless
   the external provider is independently throttling traffic.

---

### User Story 4 - Stop and reconfigure access safely (Priority: P2)

A game master can stop public access, replace or delete the provider token, change the reserved
domain or player login, replace or generate a password, and start again. Changes never leave both
old and new public configurations accepting requests.

**Why this priority**: Credential rotation and predictable shutdown are necessary for safe ongoing
use after the first successful session.

**Independent Test**: Start an endpoint, change each setting while it is active, and verify that the
old address stops accepting requests before a new address is shown and that repeated commands have
the same safe outcome.

**Acceptance Scenarios**:

1. **Given** public access is ready, **When** the game master stops it, **Then** the UI moves through
   stopping to stopped and the old public address no longer accepts requests.
2. **Given** public access is active, **When** the game master applies changed settings, **Then** the
   old public acceptance is disabled before the new configuration starts and the new URL appears
   only after it is fully protected.
3. **Given** a saved token or password, **When** the game master replaces it, **Then** only the new
   secret is used and neither old nor new value is shown afterward.
4. **Given** a saved token, **When** the game master deletes it, **Then** token presence becomes
   false and public access cannot start until another token is saved.
5. **Given** start or stop is requested repeatedly or concurrently, **When** all requests settle,
   **Then** one consistent lifecycle state remains without duplicate public endpoints.

---

### User Story 5 - Keep local play available during public failures (Priority: P2)

The game master and local players continue using the existing local or LAN address without Basic
Auth when the provider, account, domain, network, or secure credential store fails.

**Why this priority**: Optional remote access must never make an in-person game dependent on an
external service.

**Independent Test**: Keep local players active while injecting each public-access failure and
confirm that local navigation, hacking, sound, streaming, and reconnect continue without an
application restart.

**Acceptance Scenarios**:

1. **Given** an invalid or revoked provider token, **When** public startup fails, **Then** the local
   or LAN address remains usable and the UI reports a redacted authentication error.
2. **Given** no network or a startup timeout, **When** the public endpoint cannot become ready,
   **Then** startup ends in an error state while local play remains available.
3. **Given** secure credential storage becomes unavailable, **When** public access needs a secret,
   **Then** the public operation fails closed without affecting the local player experience.
4. **Given** a ready public endpoint fails unexpectedly, **When** the failure is detected, **Then**
   its URL is withdrawn, public acceptance is disabled, and local play continues.
5. **Given** the cause of a public failure is corrected, **When** the game master starts again,
   **Then** a new protected endpoint can become ready without restarting the application.

---

### User Story 6 - Exit the packaged application cleanly (Priority: P2)

The game master can close the packaged application normally or through `Cmd+Q` without leaving a
public endpoint or background resource active.

**Why this priority**: A desktop application must revoke public reachability and release resources
reliably when its owner exits.

**Independent Test**: Exercise quit during stopped, starting, ready, stopping, partially failed, and
reconfiguring states, then confirm cleanup completes within the established shutdown budget.

**Acceptance Scenarios**:

1. **Given** any public-access lifecycle state, **When** the application quits, **Then** public
   acceptance is disabled and all public-access resources are released within five seconds.
2. **Given** shutdown has already begun, **When** another shutdown signal arrives, **Then** cleanup
   remains safe and completes without extending the five-second budget.
3. **Given** startup completed after cancellation or belonged to an older configuration, **When**
   that stale completion arrives, **Then** it is rejected and no URL is published.
4. **Given** the application is launched later by double click, **When** its settings load, **Then**
   no previous endpoint is assumed active and the game master can explicitly start a new one.
5. **Given** the application process terminates unexpectedly, **When** a player uses the previous
   public URL, **Then** no player resource is served and the next application launch begins stopped.

## Edge Cases

- What happens when the token is revoked after an endpoint has already become ready?
- What happens when macOS credential storage is locked, access is denied, or the stored item was
  removed outside the application?
- How does startup end when the network is offline, name resolution fails, or the provider does not
  answer before the time limit?
- How are ownership and availability errors distinguished for a reserved domain without exposing
  account details?
- What final state wins when start, stop, reconfigure, and quit overlap?
- How is a late success from a cancelled or superseded startup prevented from publishing a stale
  URL?
- What happens to requests using the old URL during and after reconfiguration?
- How are missing, malformed, stale, or incorrect Basic Auth credentials rejected for both static
  resources and player operations?
- How is an external `Host` rejected before authorization activation, after stop, and during the
  disable-before-close interval?
- How does a long-lived non-empty stream behave across idle periods, transient network loss, and
  browser reconnect?
- How does local play recover when a public endpoint fails while local players are active?
- How does repeated shutdown handle partial startup without leaking an endpoint, background task,
  listener, or temporary secret material?

## Requirements

### Functional Requirements

- **FR-001**: The packaged `.app` MUST let the game master configure, start, stop, and monitor public
  access entirely from the desktop UI without a Terminal, environment variable, command-line
  argument, or separately installed provider executable.
- **FR-002**: The settings UI MUST accept one account token belonging to the game master's own
  ngrok account.
- **FR-003**: The settings UI MUST support an optional reserved domain whose empty value requests a
  provider-assigned HTTPS address.
- **FR-004**: The settings UI MUST prefill the player username as `players` while allowing a
  non-empty replacement and either a user-entered or newly generated player password.
- **FR-005**: A generated player password MUST contain at least 128 bits of generation entropy.
- **FR-006**: The account token and player password MUST remain distinct credentials with separate
  presence, replacement, deletion, and use semantics.
- **FR-007**: The production application MUST store both secret values only in macOS Keychain.
- **FR-008**: The application MUST persist only versioned non-secret preferences and opaque
  secret-presence indicators in Application Support using atomic replacement and user-only access.
- **FR-009**: Secret values MUST NOT appear in session JSON, player-config JSON, ordinary
  application configuration, process arguments, URLs, public protobuf contracts or responses,
  reusable status or configuration projections, named frontend events, frontend state or storage,
  logs, diagnostics, analytics, test fixtures, or displayed errors.
- **FR-010**: Every shipped artifact MUST be free of shared developer tokens, passwords, and account
  credentials.
- **FR-011**: The UI and its private operations MUST NOT read back a stored account token or player
  password.
- **FR-012**: The UI MUST limit stored-secret operations to presence indication, replacement, and
  deletion.
- **FR-013**: A newly generated player password MUST be displayed through an explicit Copy action
  exactly once during its creation flow, after which it becomes unrecoverable.
- **FR-014**: The application MUST clear transient secret input and result values as soon as their
  save or one-time copy flow finishes.
- **FR-015**: The application MUST restore non-secret settings and secret-presence indicators after
  a packaged-application restart without automatically exposing a public endpoint. The enabled
  preference MAY restore only the UI's public-access preference/presentation; Start remains an
  explicit user action on every process launch.
- **FR-016**: The game master MUST be able to start public access from the running UI without an
  application restart.
- **FR-017**: The UI MUST expose stopped, starting, ready, stopping, and error states with redacted,
  actionable failure information.
- **FR-018**: When no reserved domain is configured, successful startup MUST produce an HTTPS
  address assigned by the provider.
- **FR-019**: When a reserved domain is configured, startup MUST either use that exact domain or
  show a clear redacted ownership or availability error without publishing another URL as though it
  were the requested domain.
- **FR-020**: The UI MUST publish a public URL only after the endpoint and its complete
  authorization policy are ready.
- **FR-021**: The application boundary MUST require the correct Basic Auth username-password pair
  before serving any public static resource or accepting any public ConnectRPC request.
- **FR-022**: The exact external `Host` and its authorization policy MUST become active atomically
  before the public URL is published.
- **FR-023**: The public endpoint MUST reject every unknown external `Host` before startup
  completes, during reconfiguration, during stopping, and after failure.
- **FR-024**: Public authorization MUST preserve incremental delivery for a non-empty long-lived
  `Subscribe` stream without buffering it to completion or converting it into a single response.
- **FR-025**: An authenticated public player MUST retain character selection, navigation, hacking,
  sound, authoritative live updates, and reconnect behavior equivalent to the local player
  experience.
- **FR-026**: Successful public startup MUST expose the application's existing authoritative player
  service on port `3690` without starting a second player service.
- **FR-027**: Local and LAN player access MUST remain available without Basic Auth.
- **FR-028**: Provider, network, account, domain, secure-store, startup, and endpoint failures MUST
  leave local and LAN play available without an application restart.
- **FR-029**: Start and stop operations MUST be idempotent, cancellable, and bounded by explicit
  completion time limits.
- **FR-030**: Applying settings while public access is active MUST disable the old public acceptance
  before starting the replacement configuration.
- **FR-031**: A replacement public URL MUST remain hidden until the replacement endpoint and exact
  external `Host` authorization are ready.
- **FR-032**: Stop, reconfigure, failure, and quit paths MUST make stale public URLs unusable before
  closing or replacing their endpoints.
- **FR-033**: Quit, `Cmd+Q`, repeated shutdown, partial startup, and error cleanup MUST release all
  public endpoints, background work, listeners, and temporary secret material within five seconds.
- **FR-034**: A completion from a cancelled or superseded lifecycle operation MUST NOT change the
  current state or publish a URL.
- **FR-035**: The feature MUST preserve session JSON version 1 and player-config version 1 without
  adding or changing fields.
- **FR-036**: The feature MUST preserve existing local play, broadcast lifecycle, player roles, and
  server-authoritative game state.
- **FR-037**: The legacy production path that requires an external ngrok executable MUST be removed
  after functional and lifecycle parity is proven.
- **FR-038**: The shipped application MUST contain exactly one production public-access mechanism.
- **FR-039**: The feature MUST retain only one narrow development/test injection seam that routes
  non-secret settings and transient user-supplied credentials through the same embedded provider
  path, never places secrets in process arguments, remains unavailable to packaged UX, and cannot
  start an external ngrok process.
- **FR-040**: The required packaged profile MUST remain macOS 13 or later on `arm64` Apple Silicon,
  with Windows and Linux excluded from acceptance.
- **FR-041**: Public distribution, Developer ID credentials, notarization, and provider-plan limits
  MUST remain conditional external gates whose unavailable results are reported rather than claimed.
- **FR-042**: Automated verification MUST include unit tests, race tests, integration tests, and
  browser tests for public and local access.
- **FR-043**: Automated leak checks MUST prove that no secret appears in forbidden persistence,
  transport, UI, event, log, diagnostic, error, URL, argument, fixture, or packaged-artifact
  surfaces.
- **FR-044**: Packaged macOS smoke verification MUST exercise double-click launch, secure settings,
  start, readiness, protected player access, stop, and quit without a Terminal or external provider
  executable.
- **FR-045**: Real external-service verification MUST run only as explicit opt-in with user-supplied
  account credentials, reporting `NOT RUN` when credentials or connectivity are unavailable.
- **FR-046**: Deterministic provider substitutes MUST NOT be reported as proof of a real externally
  reachable endpoint.
- **FR-047**: An unexpected application-process termination MUST make the previous public URL
  unable to serve player resources.
- **FR-048**: The first launch after an unexpected termination MUST begin in the stopped state
  without restoring a stale URL as active.
- **FR-049**: Packaged UX MUST NOT require, download, install, or update an external ngrok CLI.
- **FR-050**: A manually entered player password MUST contain at least 8 characters without any
  character-class composition rules.
- **FR-051**: The application MUST NOT impose its own Basic Auth rate limit or account lockout,
  leaving any upstream throttling to ngrok as an external condition.
- **FR-052**: The application MUST restrict every new secret crossing from the UI to a narrow
  master-only private mutation input with minimum lifetime, redacted errors, and no logging or
  serialization outside that call.
- **FR-053**: The application MUST restrict secret-bearing private protobuf results to the single
  initial return of a newly generated player password for explicit Copy, with no existing-secret
  readback, named-event publication, or persistence in frontend state or storage.

### Out of Scope

- A project-owned relay service or support for tunnel providers other than ngrok.
- A shared developer account, embedded shared token, or automatic ngrok account registration.
- OAuth or another replacement for player Basic Auth.
- Windows or Linux support.
- Changes to session, player-configuration, or other game-data formats.
- Changes to local broadcast semantics, player roles, or authoritative game behavior.
- A public app-store or other public marketplace release.
- Treating Developer ID, notarization, paid provider features, or real external connectivity as
  available when the required external credentials or services are absent.

## Key Entities

- **Public Access Preferences**: Non-secret settings remembered for a game master, including the
  optional reserved domain, player username, and whether public access is preferred; related to
  presence indicators but never containing a secret value.
- **Provider Account Credential**: The game master's private ngrok account token, independently
  replaceable and deletable, whose stored value is never returned to the UI.
- **Player Access Credential**: The username and private password required from public players; the
  password has a lifecycle separate from the provider token and cannot be read back after storage.
- **Credential Presence**: An opaque `present`, `absent`, or `unknown` indication for a provider
  token or player password, without carrying the credential value. `unknown` represents a locked,
  denied, or unavailable secure store and is never treated as absence or permission to overwrite.
- **Public Access Session**: The observable lifecycle of one public endpoint, including its current
  state, current HTTPS address when ready, exact authorized external host, and redacted error when
  applicable.
- **One-Time Password Presentation**: A transient presentation of a newly generated player password
  that supports one explicit copy before it is discarded and cannot represent an existing stored
  password.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Under a responsive network and valid account, at least 95% of public-start attempts
  reach ready or a clear terminal error within 15 seconds, and every attempt ends within 30 seconds.
- **SC-002**: Across automated requests made before readiness, during reconfiguration, during stop,
  and after failure, zero public static resources or player operations succeed without both correct
  Basic Auth and the currently authorized external host.
- **SC-003**: Every existing player browser journey passes through the public address, including a
  non-empty stream sustained for at least 30 minutes and reconnection to current authoritative state
  within 5 seconds after connectivity is restored under test conditions.
- **SC-004**: During every simulated provider, network, token, domain, timeout, and secure-store
  failure, 100% of the existing local/LAN acceptance journey remains usable without restarting the
  application.
- **SC-005**: In 100 repeated normal, repeated, and unexpected shutdown trials spanning every
  lifecycle state, the previous URL serves zero player resources afterward; for graceful shutdown,
  public acceptance is disabled and all owned resources are released within five seconds.
- **SC-006**: In 100 repeated concurrent start, stop, and reconfigure trials, the final state matches
  the latest valid user intent with zero duplicate endpoints and zero published stale URLs.
- **SC-007**: Automated secret-leak scans report zero secret values across every forbidden surface
  and every redacted failure scenario.
- **SC-008**: A packaged macOS application launched by double click completes configuration, start,
  authenticated player access, stop, and quit without a Terminal, environment setup, command-line
  arguments, or separately installed provider executable.
- **SC-009**: After 100 save-and-relaunch trials, non-secret preferences and credential-presence
  indicators are restored in every trial while stored secret values are never displayed or returned.
- **SC-010**: All compatibility checks for existing game-data documents, local broadcast behavior,
  player roles, and authoritative state pass without migration or user action.
- **SC-011**: Release evidence labels every unavailable credential-dependent, network-dependent,
  signing, notarization, or provider-plan check as `NOT RUN` and never substitutes a deterministic
  test double for real external reachability.
- **SC-012**: Password validation rejects 100% of manually entered values shorter than eight
  characters and accepts values of eight or more characters without requiring particular character
  classes.
- **SC-013**: In an acceptance run without external provider throttling, every invalid Basic Auth
  attempt is rejected and the next request with correct credentials succeeds without an
  application-imposed cooldown or lockout.

## Assumptions

- The game master obtains and manages a personal ngrok account and token outside the application.
- The initial player username is `players`; a game master may replace it with another non-empty
  value.
- Public access starts only after an explicit user action on each application run; remembering a
  preference does not automatically publish an endpoint at launch.
- An account without a configured reserved domain can receive a provider-assigned HTTPS address.
- A generated password uses at least 128 bits of generation entropy; a user-entered password must
  contain at least eight characters and has no character-class composition requirements.
- Copying full player credentials is possible only while a new password is being entered or shown
  once after generation; an already stored password is never reconstructed for copying.
- The 15-second readiness target assumes a responsive provider and network; the 30-second terminal
  bound applies even when they are unresponsive.
- The development/test injection seam in FR-039 is available only to automation, uses the same
  embedded provider behavior as production, and provides no external-process fallback.
- Basic Auth attempt throttling, when present, is an external ngrok behavior and is not an
  application acceptance guarantee.
- Real external-service, signing, notarization, and provider-plan validation uses user-supplied
  prerequisites and remains conditional.

## Verbatim Constraints

- Feature identifier: `007-embedded-ngrok-access`.
- Provider name: `ngrok`.
- Initial player username: `players`.
- Player authentication scheme: `Basic Auth`.
- External authority field: `Host`.
- Long-lived player stream: `Subscribe`.
- Authoritative local player port: `3690`.
- Packaged application form: `.app`.
- Quit command: `Cmd+Q`.
- Required platform profile: `macOS 13+ arm64` (`Apple Silicon`).
- Required secure credential store: `Keychain`.
- Existing formats: `session JSON version 1` and `player-config version 1`.
- Unavailable conditional-evidence result: `NOT RUN`.
