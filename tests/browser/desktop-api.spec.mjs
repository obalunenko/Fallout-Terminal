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
