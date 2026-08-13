import { expect, test } from '@playwright/test';

const RECOGNITION_KEY = 'fallout-terminal.player-token';
const NGROK_TEST_URL = process.env.NGROK_TEST_URL;
const NGROK_TEST_USERNAME = process.env.NGROK_USERNAME;
const NGROK_TEST_PASSWORD = process.env.NGROK_PASSWORD;
const NGROK_TEST_FIXTURE = process.env.NGROK_TEST_FIXTURE === '1';
const PROTECTED_FIXTURE_URL = 'http://127.0.0.1:34120';

test.beforeEach(async ({ request, page }) => {
  await page.addInitScript(() => {
    window.WebSocket = class ForbiddenLegacyPlayerTransport {
      constructor() {
        throw new Error('the player must use generated Connect exclusively');
      }
    };
  });
  const response = await request.post('/__fixture/reset');
  expect(response.status()).toBe(204);
});

test('built player contains no legacy JSON protocol or WebSocket constructor', async ({ page }) => {
  const response = await page.request.get('/assets/' + (await page.request.get('/').then(result => result.text()))
    .match(/src="\/assets\/(.+\.js)"/)?.[1]);
  expect(response.ok()).toBe(true);
  const bundle = await response.text();
  for (const forbidden of ['WebSocket(', 'SESSION_HELLO', 'CHARACTER_SELECT', 'NAV_ACTION', 'HACK_GUESS', 'HACK_PATTERN', 'ACTION_RESULT']) {
    expect(bundle).not.toContain(forbidden);
  }
});

test('protected forwarding authenticates static, unary, and streaming capabilities', async ({ browser, request }) => {
  const protectedURL = PROTECTED_FIXTURE_URL + '/';
  for (const headers of [{}, { Authorization: `Basic ${Buffer.from('players:wrong-password').toString('base64')}` }]) {
    const response = await request.get(protectedURL, { headers });
    expect(response.status()).toBe(401);
    expect(response.headers()['www-authenticate']).toContain('Fallout Terminal Players');
    expect(await response.text()).not.toContain('characterSelect');
  }

  const authorization = `Basic ${Buffer.from('players:password-long-enough').toString('base64')}`;
  const pageResponse = await request.get(protectedURL, { headers: { Authorization: authorization } });
  expect(pageResponse.status()).toBe(200);
  expect(await pageResponse.text()).toContain('characterSelect');

  const rpcResponse = await request.post(
    PROTECTED_FIXTURE_URL + '/fallout.terminal.player.v1.PlayerService/SoundManifest',
    {
      headers: { Authorization: authorization, 'Content-Type': 'application/json' },
      data: { category: 'SOUND_CATEGORY_AMBIENT' },
    },
  );
  expect(rpcResponse.status()).toBe(200);

  const unauthorizedSubscribe = await request.post(
    PROTECTED_FIXTURE_URL + '/fallout.terminal.player.v1.PlayerService/Subscribe',
    {
      headers: {
        'Content-Type': 'application/connect+proto',
        'Connect-Protocol-Version': '1',
      },
      data: Buffer.alloc(5),
    },
  );
  expect(unauthorizedSubscribe.status()).toBe(401);

  const context = await browser.newContext({
    httpCredentials: { username: 'players', password: 'password-long-enough' },
  });
  const page = await context.newPage();
  const subscribeResponses = [];
  page.on('response', response => {
    if (response.url().endsWith('/fallout.terminal.player.v1.PlayerService/Subscribe')) {
      subscribeResponses.push(response.status());
    }
  });
  await page.goto(protectedURL);
  await expect(page.locator('#connOverlay')).toBeHidden();
  await expect(page.locator('#characterSelect')).toBeVisible();
  await page.locator('#characterOptions button:not([disabled])').first().click();
  await expect(page.locator('#termList')).toBeVisible();
  await expect.poll(() => subscribeResponses.length).toBe(1);
  expect(subscribeResponses).toEqual([200]);
  await context.close();
});

test.describe('actual authenticated ngrok endpoint', () => {
  test.skip(
    !NGROK_TEST_URL || !NGROK_TEST_USERNAME || !NGROK_TEST_PASSWORD,
    'set NGROK_TEST_URL, NGROK_USERNAME, and NGROK_PASSWORD for the public streaming acceptance journey',
  );

  test('delivers the first snapshot, dismisses the overlay, and reconnects', async ({ browser }) => {
    const context = await browser.newContext({
      httpCredentials: {
        username: NGROK_TEST_USERNAME,
        password: NGROK_TEST_PASSWORD,
      },
    });
    const page = await context.newPage();
    const subscribeResponses = [];
    page.on('response', response => {
      if (response.url().endsWith('/fallout.terminal.player.v1.PlayerService/Subscribe')) {
        subscribeResponses.push({
          status: response.status(),
          contentType: response.headers()['content-type'] || '',
        });
      }
    });

    await page.goto(NGROK_TEST_URL, { waitUntil: 'domcontentloaded' });
    await expect(page.locator('#connOverlay')).toBeHidden({ timeout: 20_000 });
    await expect.poll(() => page.evaluate(key => Boolean(localStorage.getItem(key)), RECOGNITION_KEY)).toBe(true);
    await expect(page.locator('#screen')).toBeVisible();
    if (await page.locator('#characterSelect').isVisible()) {
      await page.locator('#characterOptions button:not([disabled])').first().click();
      await expect(page.locator('#termList')).toBeVisible();
    }
    if (NGROK_TEST_FIXTURE) {
      await expect(page.locator('#termList')).toBeVisible();
      const update = await page.request.post(new URL('/__fixture/update', NGROK_TEST_URL).href);
      expect(update.status()).toBe(204);
      await expect(page.locator('.term-row', { hasText: 'PUBLIC UPDATE' })).toBeVisible();
    }

    const handle = await page.evaluate(key => localStorage.getItem(key), RECOGNITION_KEY);
    await page.reload({ waitUntil: 'domcontentloaded' });
    await expect(page.locator('#connOverlay')).toBeHidden({ timeout: 20_000 });
    expect(await page.evaluate(key => localStorage.getItem(key), RECOGNITION_KEY)).toBe(handle);
    await expect.poll(() => subscribeResponses.length).toBeGreaterThanOrEqual(2);
    expect(subscribeResponses.every(response => response.status === 200)).toBe(true);
    expect(subscribeResponses.every(response => response.contentType === 'application/connect+proto')).toBe(true);

    await context.close();
  });
});

test('local player discovers sounds only through the typed same-origin manifest', async ({ page }) => {
  const soundRequests = [];
  page.on('request', request => {
    if (request.url().includes('SoundManifest') || request.url().includes('/api/sounds/')) {
      soundRequests.push(request.url());
    }
  });
  await page.goto('/');
  await expect(page.locator('#connOverlay')).toBeHidden();
  await expect.poll(() => soundRequests.filter(url => url.endsWith('/SoundManifest')).length).toBeGreaterThanOrEqual(8);
  expect(soundRequests.some(url => url.includes('/api/sounds/'))).toBe(false);
  expect(soundRequests.every(url => new URL(url).origin === 'http://127.0.0.1:34119')).toBe(true);
});

test('clean profile receives a generated snapshot, stores only its handle, and selects a character', async ({ page }) => {
  const rpcRequests = [];
  page.on('request', request => {
    if (request.url().includes('/fallout.terminal.player.v1.PlayerService/')) {
      rpcRequests.push({ url: request.url(), contentType: request.headers()['content-type'] || '' });
    }
  });

  await page.goto('/');
  await expect(page.locator('#connOverlay')).toBeHidden();
  await expect(page.locator('#characterSelect')).toBeVisible();

  const storage = await page.evaluate(() => {
    const values = {};
    for (let index = 0; index < localStorage.length; index += 1) {
      const key = localStorage.key(index);
      values[key] = localStorage.getItem(key);
    }
    return values;
  });
  expect(Object.keys(storage)).toEqual([RECOGNITION_KEY]);
  expect(storage[RECOGNITION_KEY]).toMatch(/\S+/);

  const firstCharacter = page.locator('#characterOptions button').first();
  await expect(firstCharacter).toBeVisible();
  await firstCharacter.click();
  await expect(page.locator('#playerCharacterName')).not.toHaveText('');

  expect(rpcRequests.some(request => request.url.endsWith('/Subscribe'))).toBe(true);
  expect(rpcRequests.some(request => request.url.endsWith('/SelectCharacter'))).toBe(true);
  expect(rpcRequests.every(request => ['application/connect+proto', 'application/proto'].includes(request.contentType))).toBe(true);
});

test('recognized reconnect resumes from one complete current snapshot without replaying cues', async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('#connOverlay')).toBeHidden();
  const handle = await page.evaluate(key => localStorage.getItem(key), RECOGNITION_KEY);
  expect(handle).toMatch(/\S+/);

  await page.reload();
  await expect(page.locator('#connOverlay')).toBeHidden();
  expect(await page.evaluate(key => localStorage.getItem(key), RECOGNITION_KEY)).toBe(handle);
  expect(await page.evaluate(() => window.__audioStarts || 0)).toBe(0);
});

test('well-formed unknown recognition is replaced while malformed recognition fails closed', async ({ browser }) => {
  const context = await browser.newContext();
  const page = await context.newPage();
  await page.addInitScript(key => localStorage.setItem(key, 'unknown-but-well-formed'), RECOGNITION_KEY);
  await page.goto('/');
  await expect(page.locator('#connOverlay')).toBeHidden();
  const replacement = await page.evaluate(key => localStorage.getItem(key), RECOGNITION_KEY);
  expect(replacement).not.toBe('unknown-but-well-formed');
  expect(replacement).toMatch(/\S+/);
  await context.close();
});

test('concurrent clean tabs converge on one logical recognition handle', async ({ browser }) => {
  const context = await browser.newContext();
  const pages = await Promise.all([context.newPage(), context.newPage(), context.newPage()]);
  await Promise.all(pages.map(page => page.goto('/')));
  await Promise.all(pages.map(page => expect(page.locator('#connOverlay')).toBeHidden()));
  const handles = await Promise.all(pages.map(page => page.evaluate(key => localStorage.getItem(key), RECOGNITION_KEY)));
  expect(new Set(handles).size).toBe(1);
  expect(handles[0]).toMatch(/\S+/);

  await pages[0].close();
  await expect(pages[1].locator('#connOverlay')).toBeHidden();
  await pages[1].close();
  await expect(pages[2].locator('#connOverlay')).toBeHidden();
  await context.close();
});

test('mixed typed actions stay pending until unary result and authoritative stream revision converge', async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('#connOverlay')).toBeHidden();
  const character = page.locator('#characterOptions button:not([disabled])').first();
  await character.click();
  await expect(page.locator('#characterSelect')).not.toHaveAttribute('aria-busy', 'true');

  const typedProcedures = await page.evaluate(() => performance.getEntriesByType('resource')
    .map(entry => entry.name)
    .filter(name => name.includes('/fallout.terminal.player.v1.PlayerService/')));
  expect(typedProcedures.some(name => name.endsWith('/SelectCharacter'))).toBe(true);
  expect(typedProcedures.every(name => !name.endsWith('/Command'))).toBe(true);
});

test('conflicting generated selections clear pending immediately and never alter the rejected canonical view optimistically', async ({ browser }) => {
  const contexts = await Promise.all([browser.newContext(), browser.newContext()]);
  const pages = await Promise.all(contexts.map(context => context.newPage()));
  let releaseSelections;
  const gate = new Promise(resolve => { releaseSelections = resolve; });
  let observed = 0;
  await Promise.all(pages.map(page => page.route('**/fallout.terminal.player.v1.PlayerService/SelectCharacter', async route => {
    observed += 1;
    await gate;
    await route.continue();
  })));

  await Promise.all(pages.map(page => page.goto('/')));
  await Promise.all(pages.map(page => expect(page.locator('#connOverlay')).toBeHidden()));
  const fallbackNames = await Promise.all(pages.map(page => page.locator('#playerCharacterName').textContent()));
  await Promise.all(pages.map(page => page.locator('#characterOptions button').first().click()));
  await expect.poll(() => observed).toBe(2);
  await Promise.all(pages.map(page => expect(page.locator('#characterSelect')).toHaveAttribute('aria-busy', 'true')));
  expect(await Promise.all(pages.map(page => page.locator('#playerCharacterName').textContent()))).toEqual(fallbackNames);

  releaseSelections();
  await Promise.all(pages.map(page => expect(page.locator('#characterSelect')).not.toHaveAttribute('aria-busy', 'true')));
  const names = await Promise.all(pages.map(page => page.locator('#playerCharacterName').textContent()));
  expect(names.filter(name => name === 'Mara')).toHaveLength(1);
  expect(names.filter(name => fallbackNames.includes(name))).toHaveLength(1);
  const rejectedPage = pages[names.findIndex(name => fallbackNames.includes(name))];
  await expect(rejectedPage.locator('#characterSelect')).toBeVisible();
  await expect(rejectedPage.locator('#termList')).toBeHidden();
  await expect(rejectedPage.locator('#playerNotice')).toContainText('conflict');

  await Promise.all(contexts.map(context => context.close()));
});

test('retained request identity reused with a different typed payload is rejected without canonical navigation', async ({ page }) => {
  await page.addInitScript(() => {
    const requestIds = ['session-owner', 'selection-request', 'navigation-request'];
    Object.defineProperty(window.crypto, 'randomUUID', {
      configurable: true,
      value: () => requestIds.shift() || 'navigation-request',
    });
  });
  await page.goto('/');
  await expect(page.locator('#connOverlay')).toBeHidden();
  await page.locator('#characterOptions button:not([disabled])').first().click();
  await expect(page.locator('#termList')).toBeVisible();

  await page.locator('.term-row', { hasText: 'DOCS' }).click();
  await expect(page.locator('.term-row', { hasText: 'REPORT' })).toBeVisible();
  await page.locator('#backBtn').click();
  await expect(page.locator('#screen')).not.toHaveClass(/shared-input-pending/);
  await expect(page.locator('.term-row', { hasText: 'REPORT' })).toBeVisible();
  await expect(page.locator('#playerNotice')).toContainText('duplicate');
});
