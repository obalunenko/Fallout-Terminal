import { expect, test } from '@playwright/test';

test.beforeEach(async ({ page }) => {
  await page.goto('/__fixture/desktop-api');
  await expect.poll(() => page.evaluate(() => typeof window.desktopAPI)).toBe('object');
});

test('generated desktop service calls remain explicit and normalized behind the facade', async ({ page }) => {
  const results = await page.evaluate(async () => ({
    open: await desktopAPI.openSession(),
    save: await desktopAPI.saveSession({ version: 1 }),
    url: await desktopAPI.openUrl('https://fallout.example'),
    terminal: await desktopAPI.requestTerminalClear(),
    playerConfig: await desktopAPI.openPlayerConfig(),
  }));
  expect(results.open.ok).toBe(true);
  expect(results.save.ok).toBe(true);
  expect(results.url.ok).toBe(true);
  expect(results.terminal).toEqual(expect.objectContaining({ ok: true, status: '', switchId: '' }));
  expect(results.playerConfig).toEqual(expect.objectContaining({ ok: true, canceled: false }));

  const calls = await page.evaluate(() => __desktopFixture.calls.map(call => call.method));
  expect(calls).toEqual(expect.arrayContaining([
    'OpenSession', 'SaveSession', 'OpenURL', 'RequestTerminalClear', 'OpenPlayerConfig',
  ]));
  expect(calls).not.toContain('Dispatch');
  expect(calls).not.toContain('Call');
});

test('saveSession snapshots the complete cross-terminal document at the Wails boundary', async ({ page }) => {
  const retained = await page.evaluate(async () => {
    const candidate = {
      version: 1,
      name: 'demo boundary candidate',
      terminals: [
        {
          id: 't_demo1', name: 'Source', hackLevel: 0, introText: '',
          root: {
            id: 'root', type: 'folder', name: 'ROOT',
            children: [{
              id: 'go', type: 'command', name: 'GO',
              terminalTransition: { targetTerminalId: 't_demo2' },
            }],
          },
        },
        {
          id: 't_demo2', name: 'Target', hackLevel: 0, introText: '',
          root: { id: 'root', type: 'folder', name: 'ROOT', children: [] },
        },
      ],
    };
    const pending = desktopAPI.saveSession(candidate);
    candidate.terminals.splice(1, 1);
    await pending;
    return __desktopFixture.calls.filter(call => call.method === 'SaveSession').at(-1).args[0];
  });

  expect(retained.terminals.map(terminal => terminal.id)).toEqual(['t_demo1', 't_demo2']);
  expect(retained.terminals[0].root.children[0].terminalTransition.targetTerminalId).toBe('t_demo2');
});

test('all four listeners precede the snapshot and newer wrapped events win per field', async ({ page }) => {
  const result = await page.evaluate(async () => {
    __desktopFixture.deferStatus();
    const observed = { server: [], clients: [], hack: [], coordination: [] };
    const releases = [
      desktopAPI.onServerInfo(value => observed.server.push(value)),
      desktopAPI.onClientCount(value => observed.clients.push(value)),
      desktopAPI.onHackState(value => observed.hack.push(value)),
      desktopAPI.onCoordinationState(value => observed.coordination.push(value)),
    ];
    __desktopFixture.emit('client-count', 7);
    __desktopFixture.emit('server-info', { url: 'https://public.example', tunnel: true });
    __desktopFixture.resolveStatus({
      serverInfo: { url: 'http://127.0.0.1:3690', tunnel: false },
      clientCount: 1,
      hackState: { attemptsLeft: 2 },
      coordinationState: { revision: 4 },
    });
    await Promise.resolve();
    await Promise.resolve();
    return { observed, timeline: __desktopFixture.timeline.map(entry => entry.method) };
  });

  expect(result.timeline.slice(0, 5)).toEqual([
    'event:on:server-info', 'event:on:client-count', 'event:on:hack-state',
    'event:on:coordination-state', 'GetRuntimeStatus',
  ]);
  expect(result.observed.clients).toEqual([7]);
  expect(result.observed.server).toEqual([expect.objectContaining({
    url: 'https://public.example', localUrl: '', tunnel: true,
  })]);
  expect(result.observed.hack).toEqual([{ attemptsLeft: 2 }]);
  expect(result.observed.coordination).toEqual([{ revision: 4 }]);
});

test('release is exact-once, suppresses pending snapshot callbacks, and hot disposal releases old listeners', async ({ page }) => {
  const result = await page.evaluate(async () => {
    __desktopFixture.deferStatus();
    let callbacks = 0;
    const releases = [
      desktopAPI.onServerInfo(() => { callbacks += 1; }),
      desktopAPI.onClientCount(() => { callbacks += 1; }),
      desktopAPI.onHackState(() => { callbacks += 1; }),
      desktopAPI.onCoordinationState(() => { callbacks += 1; }),
    ];
    releases[0]();
    releases[0]();
    __desktopFixture.emit('server-info', { url: 'http://late.example', tunnel: false });
    __desktopFixture.resolveStatus();
    await Promise.resolve();
    await import('/__fixture/desktop-api.js?hot=2');
    await Promise.resolve();
    return {
      callbacks,
      releases: ['server-info', 'client-count', 'hack-state', 'coordination-state']
        .map(name => __desktopFixture.releaseCount(name)),
    };
  });

  expect(result.callbacks).toBe(3);
  expect(result.releases).toEqual([1, 1, 1, 1]);
});

test('public-access facade exposes exactly five methods with secret-free reusable results', async ({ page }) => {
  const result = await page.evaluate(async () => {
    const request = {
      expectedRevision: 0,
      enabledPreference: true,
      reservedDomain: '',
      username: 'players',
      replacementProviderToken: 'synthetic-provider-input',
      replacementPlayerPassword: 'synthetic-player-input',
    };
    const saved = await desktopAPI.savePublicAccessSettings(request);
    const generated = await desktopAPI.generatePlayerPassword({ expectedRevision: saved.snapshot.preferences.revision });
    const afterGenerated = await desktopAPI.getPublicAccess();
    const started = await desktopAPI.startPublicAccess({ expectedRevision: afterGenerated.preferences.revision });
    const stopped = await desktopAPI.stopPublicAccess({ expectedRevision: started.snapshot.preferences.revision });
    return {
      request,
      saved,
      generated,
      afterGenerated,
      started,
      stopped,
      methods: __desktopFixture.calls.map(call => call.method).filter(method => method.includes('PublicAccess') || method.includes('PlayerPassword')),
      retainedCalls: __desktopFixture.calls,
    };
  });

  expect(result.methods).toEqual([
    'SavePublicAccessSettings', 'GeneratePlayerPassword', 'GetPublicAccess',
    'StartPublicAccess', 'StopPublicAccess',
  ]);
  expect(result.request.replacementProviderToken).toBe('');
  expect(result.request.replacementPlayerPassword).toBe('');
  expect(result.saved.snapshot.providerTokenPresence).toBe('present');
  expect(result.saved.snapshot.playerPasswordPresence).toBe('present');
  expect(result.generated.generatedPassword).toBe('synthetic-one-time-generated-value');
  expect(result.afterGenerated.generatedPassword).toBeUndefined();
  expect(JSON.stringify([result.saved, result.afterGenerated, result.started, result.stopped, result.retainedCalls]))
    .not.toContain('synthetic-one-time-generated-value');
  expect(JSON.stringify(result.retainedCalls)).not.toContain('synthetic-provider-input');
  expect(JSON.stringify(result.retainedCalls)).not.toContain('synthetic-player-input');
});

test('public-access event beats an equal or older snapshot and disposal releases exactly once', async ({ page }) => {
  const result = await page.evaluate(async () => {
    __desktopFixture.deferPublicAccess();
    const observed = [];
    const release = desktopAPI.onPublicAccessStatus(value => observed.push(value));
    __desktopFixture.emit('public-access-status', {
      preferences: { version: 1, username: 'players', revision: 3 },
      providerTokenPresence: 'unknown',
      playerPasswordPresence: 'present',
      status: { state: 'stopped', generation: 4, settingsRevision: 3 },
    });
    __desktopFixture.resolvePublicAccess({
      preferences: { version: 1, username: 'players', revision: 2 },
      providerTokenPresence: 'absent',
      playerPasswordPresence: 'absent',
      status: { state: 'stopped', generation: 3, settingsRevision: 2 },
    });
    await Promise.resolve();
    await Promise.resolve();
    release();
    release();
    return {
      observed,
      releaseCount: __desktopFixture.releaseCount('public-access-status'),
      timeline: __desktopFixture.timeline.map(entry => entry.method),
    };
  });

  expect(result.timeline).toEqual(expect.arrayContaining([
    'event:on:public-access-status', 'GetPublicAccess',
  ]));
  expect(result.observed).toHaveLength(1);
  expect(result.observed[0]).toEqual(expect.objectContaining({
    providerTokenPresence: 'unknown',
    status: expect.objectContaining({ generation: 4, settingsRevision: 3 }),
  }));
  expect(result.releaseCount).toBe(1);
});
