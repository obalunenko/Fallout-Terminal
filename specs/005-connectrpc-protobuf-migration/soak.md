# Stream and Reconnect Soak Evidence

Date: 2026-08-13 (Asia/Tbilisi)
Revision under test: working tree for feature 005

## Representative duration workload

The deterministic soak models three hours at one authoritative projection per
simulated second: 10,800 strictly increasing compound updates. It interrupts
and rebuilds the physical stream every five simulated minutes (36 reconnects),
requires each reconnect to begin at the complete current snapshot revision,
and verifies final idempotent cleanup with zero registered streams.

```text
$ GOCACHE=/private/tmp/fallout-go-cache go test ./internal/player \
    -run TestRepresentativeThreeHourStreamReconnectSoak -count=1 -v
--- PASS: TestRepresentativeThreeHourStreamReconnectSoak (0.02s)
PASS
```

Result: 10,800/10,800 ordered deliveries, 36/36 complete-snapshot recoveries,
no stale increment replay, no overflow, and no leaked subscription.

## Local and authenticated-public boundary

The production-shaped browser fixture uses the generated Connect handler and
built static/sound assets. Its protected prefix applies the same fail-closed
Basic Auth semantics used by the ngrok traffic policy before forwarding either
static or RPC capabilities. The focused soak journeys exercised invalid auth,
valid authenticated static plus `SoundManifest`, recognized reconnect, and
three-tab recognition convergence:

```text
$ npm test --prefix tests/browser -- --grep \
    'protected forwarding|recognized reconnect|concurrent clean tabs'
3 passed (1.8s)
```

The tunnel service suite separately verifies the fixed HTTPS ngrok domain,
exact `enforce: true` Basic Auth policy, static/Connect-agnostic forwarding,
credential precedence/redaction, startup timeout, process ownership, and policy
cleanup. Credentials are never recorded in this evidence.

## Interpretation

This is a scheduled, accelerated three-hour-equivalent soak: duration-sensitive
stream ordering/reconnect work is represented by 10,800 sequential intervals,
while origin/auth behavior is exercised through the production handler and the
same fail-closed policy semantics without publishing an internet endpoint from
the test runner. A literal three-to-four-hour public internet run remains a
release-operator smoke (account/domain/network dependent), not a condition that
weakens or bypasses any automated acceptance gate.
