import { expect, test } from '@playwright/test';

const TOKEN_KEY = 'fallout-terminal.player-token';
const PLAYER_SERVICE = '/fallout.terminal.player.v1.PlayerService/';

test.beforeEach(async ({ request }) => {
  const response = await request.post('/__fixture/reset');
  expect(response.status()).toBe(204);
});

async function installPlayerDiagnostics(target, storedToken = null, { audioFailure = false } = {}) {
  await target.addInitScript(({ tokenKey, token, failAudio }) => {
    window.__webSocketConstructions = 0;
    window.WebSocket = class ForbiddenLegacyPlayerTransport {
      constructor() {
        window.__webSocketConstructions += 1;
        throw new Error('the player must not construct a WebSocket');
      }
    };
    if (token !== null && localStorage.getItem(tokenKey) === null) {
      localStorage.setItem(tokenKey, token);
    }

    if (failAudio) {
      window.AudioContext = class UnavailableAudioContext {
        constructor() { throw new Error('simulated audio-device failure'); }
      };
      HTMLMediaElement.prototype.play = () => Promise.reject(new DOMException('blocked', 'NotAllowedError'));
    } else {
      HTMLMediaElement.prototype.play = () => Promise.resolve();
    }
  }, { tokenKey: TOKEN_KEY, token: storedToken, failAudio: audioFailure });
}

async function openPlayer(page, storedToken = null, options = {}) {
  await installPlayerDiagnostics(page, storedToken, options);
  await page.goto('/');
  await expect(page.locator('#connOverlay')).toBeHidden();
  await expect.poll(() => page.evaluate(() => window.__webSocketConstructions)).toBe(0);
}

async function selectFirstAvailable(page) {
  const character = page.locator('#characterOptions button:not([disabled])').first();
  await expect(character).toBeVisible();
  const name = await character.textContent();
  await character.click();
  await expect(page.locator('#characterSelect')).not.toHaveAttribute('aria-busy', 'true');
  await expect(page.locator('#playerCharacterName')).toHaveText(name.trim());
  return name.trim();
}

function typedPlayerRequests(page) {
  const requests = [];
  page.on('request', request => {
    if (!request.url().includes(PLAYER_SERVICE)) return;
    requests.push({
      procedure: new URL(request.url()).pathname.split('/').at(-1),
      contentType: request.headers()['content-type'] || '',
    });
  });
  return requests;
}

test('selection uses a generated unary procedure and remains pending until its typed result converges with the stream', async ({ page }) => {
  const requests = typedPlayerRequests(page);
  let releaseSelection;
  const selectionGate = new Promise(resolve => { releaseSelection = resolve; });
  let selectionObserved = false;
  await page.route(`**${PLAYER_SERVICE}SelectCharacter`, async route => {
    selectionObserved = true;
    await selectionGate;
    await route.continue();
  });

  await openPlayer(page);
  const character = page.locator('#characterOptions button:not([disabled])').first();
  const expectedName = (await character.textContent()).trim();
  await character.click();
  await expect.poll(() => selectionObserved).toBe(true);
  await expect(page.locator('#characterSelect')).toHaveAttribute('aria-busy', 'true');
  await expect(page.locator('#playerCharacterName')).not.toHaveText(expectedName);

  releaseSelection();
  await expect(page.locator('#characterSelect')).not.toHaveAttribute('aria-busy', 'true');
  await expect(page.locator('#playerCharacterName')).toHaveText(expectedName);
  expect(requests.map(request => request.procedure)).toContain('Subscribe');
  expect(requests.map(request => request.procedure)).toContain('SelectCharacter');
  expect(requests.every(request => ['application/connect+proto', 'application/proto'].includes(request.contentType))).toBe(true);
});

test('active navigation applies the authoritative compound update while awaiting the delayed unary result', async ({ page }) => {
  const requests = typedPlayerRequests(page);
  await openPlayer(page);
  await selectFirstAvailable(page);
  await expect(page.locator('#termList')).toBeVisible();

  let releaseNavigate;
  const navigateGate = new Promise(resolve => { releaseNavigate = resolve; });
  let navigateObserved = false;
  await page.route(`**${PLAYER_SERVICE}Navigate`, async route => {
    navigateObserved = true;
    await navigateGate;
    await route.continue();
  });

  await page.locator('.term-row', { hasText: 'DOCS' }).click();
  await expect.poll(() => navigateObserved).toBe(true);
  await expect(page.locator('#screen')).toHaveClass(/shared-input-pending/);
  releaseNavigate();

  await expect(page.locator('.term-row', { hasText: 'REPORT' })).toBeVisible();
  await expect(page.locator('#screen')).not.toHaveClass(/shared-input-pending/);
  expect(requests.map(request => request.procedure)).toContain('Navigate');
});

test('three tabs share one recognition handle and converge after one generated selection', async ({ browser }) => {
  const context = await browser.newContext();
  await installPlayerDiagnostics(context);
  const pages = await Promise.all([context.newPage(), context.newPage(), context.newPage()]);
  await Promise.all(pages.map(page => page.goto('/')));
  await Promise.all(pages.map(page => expect(page.locator('#connOverlay')).toBeHidden()));

  const handles = await Promise.all(pages.map(page => page.evaluate(key => localStorage.getItem(key), TOKEN_KEY)));
  expect(new Set(handles).size).toBe(1);
  const characterName = await selectFirstAvailable(pages[0]);
  await Promise.all(pages.map(page => expect(page.locator('#playerCharacterName')).toHaveText(characterName)));

  await pages[0].close();
  await expect(pages[1].locator('#termList')).toBeVisible();
  await pages[1].close();
  await expect(pages[2].locator('#termList')).toBeVisible();
  await context.close();
});

test('four through seven generated players converge across mixed navigation, reconnect, replay, and sound-safe state', async ({ browser, request }) => {
  let acceptedActions = 0;
  let reconnects = 0;

  for (let playerCount = 4; playerCount <= 7; playerCount += 1) {
    const reset = await request.post('/__fixture/reset');
    expect(reset.status()).toBe(204);
    const contexts = await Promise.all(Array.from({ length: playerCount }, () => browser.newContext()));
    await Promise.all(contexts.map(context => installPlayerDiagnostics(context)));
    const pages = await Promise.all(contexts.map(context => context.newPage()));
    await Promise.all(pages.map(page => page.goto('/')));
    await Promise.all(pages.map(page => expect(page.locator('#connOverlay')).toBeHidden()));

    const handles = await Promise.all(pages.map(page => page.evaluate(key => localStorage.getItem(key), TOKEN_KEY)));
    expect(new Set(handles).size).toBe(playerCount);
    const characters = [];
    for (const page of pages) characters.push(await selectFirstAvailable(page));
    expect(new Set(characters).size).toBe(playerCount);
    await expect(pages[0].locator('#roleBadge')).toContainText('АКТИВНЫЙ');
    await Promise.all(pages.slice(1).map(page => expect(page.locator('#roleBadge')).toContainText('НАБЛЮДАТЕЛЬ')));
    const observerRequests = typedPlayerRequests(pages[1]);

    for (let round = 0; round < 4; round += 1) {
      await pages[0].locator('.term-row', { hasText: 'DOCS' }).click();
      await Promise.all(pages.map(page => expect(page.locator('.term-row', { hasText: 'REPORT' })).toBeVisible()));
      acceptedActions += 1;

      const observerRequestCount = observerRequests.length;
      await pages[1].locator('#backBtn').click({ force: true });
      expect(observerRequests.length).toBe(observerRequestCount);
      await pages[0].locator('#backBtn').click();
      await Promise.all(pages.map(page => expect(page.locator('.term-row', { hasText: 'DOCS' })).toBeVisible()));
      acceptedActions += 1;
    }

    if (reconnects < 3) {
      const page = pages[reconnects % playerCount];
      const handle = await page.evaluate(key => localStorage.getItem(key), TOKEN_KEY);
      await page.reload();
      await expect(page.locator('#connOverlay')).toBeHidden();
      expect(await page.evaluate(key => localStorage.getItem(key), TOKEN_KEY)).toBe(handle);
      expect(await page.evaluate(() => window.__audioStarts || 0)).toBe(0);
      reconnects += 1;
    }

    for (const page of pages) {
      const storage = await page.evaluate(() => Object.fromEntries(
        Array.from({ length: localStorage.length }, (_, index) => {
          const key = localStorage.key(index);
          return [key, localStorage.getItem(key)];
        }),
      ));
      expect(Object.keys(storage)).toEqual([TOKEN_KEY]);
    }
    await Promise.all(contexts.map(context => context.close()));
  }

  expect(acceptedActions).toBeGreaterThanOrEqual(25);
  expect(reconnects).toBe(3);
});

test('recognized reload retains identity while an unknown opaque handle receives a safe replacement', async ({ browser, page }) => {
  await openPlayer(page);
  const handle = await page.evaluate(key => localStorage.getItem(key), TOKEN_KEY);
  expect(handle).toMatch(/\S+/);

  await page.reload();
  await expect(page.locator('#connOverlay')).toBeHidden();
  expect(await page.evaluate(key => localStorage.getItem(key), TOKEN_KEY)).toBe(handle);

  const staleContext = await browser.newContext();
  await installPlayerDiagnostics(staleContext, 'unknown-but-well-formed');
  const stalePage = await staleContext.newPage();
  await stalePage.goto('/');
  await expect(stalePage.locator('#connOverlay')).toBeHidden();
  const replacement = await stalePage.evaluate(key => localStorage.getItem(key), TOKEN_KEY);
  expect(replacement).not.toBe('unknown-but-well-formed');
  expect(replacement).toMatch(/\S+/);
  await staleContext.close();
});

test('observer projection is visibly read-only and emits no typed mutation', async ({ browser }) => {
  const controllerContext = await browser.newContext();
  await installPlayerDiagnostics(controllerContext);
  const controller = await controllerContext.newPage();
  await controller.goto('/');
  await expect(controller.locator('#connOverlay')).toBeHidden();
  await selectFirstAvailable(controller);

  const observerContext = await browser.newContext();
  await installPlayerDiagnostics(observerContext);
  const observer = await observerContext.newPage();
  const observerRequests = typedPlayerRequests(observer);
  await observer.goto('/');
  await expect(observer.locator('#connOverlay')).toBeHidden();
  await selectFirstAvailable(observer);
  await expect(observer.locator('#roleBadge')).toContainText('НАБЛЮДАТЕЛЬ');
  await expect(observer.locator('#screen')).toHaveAttribute('aria-readonly', 'true');

  const before = observerRequests.length;
  await observer.locator('.term-row', { hasText: 'DOCS' }).click();
  await observer.waitForTimeout(100);
  expect(observerRequests.slice(before).map(request => request.procedure)).not.toContain('Navigate');
  await expect(observer.locator('.term-row', { hasText: 'DOCS' })).toBeVisible();

  await observerContext.close();
  await controllerContext.close();
});

test('optional audio failures never block typed selection or navigation', async ({ page }) => {
  await openPlayer(page, null, { audioFailure: true });
  await selectFirstAvailable(page);
  await page.locator('.term-row', { hasText: 'DOCS' }).click();
  await expect(page.locator('.term-row', { hasText: 'REPORT' })).toBeVisible();
  await expect(page.locator('#connOverlay')).toBeHidden();
  await expect.poll(() => page.evaluate(() => window.__webSocketConstructions)).toBe(0);
});

test('player persistence contains only the opaque recognition handle', async ({ page }) => {
  await openPlayer(page);
  const storage = await page.evaluate(() => {
    const values = {};
    for (let index = 0; index < localStorage.length; index += 1) {
      const key = localStorage.key(index);
      values[key] = localStorage.getItem(key);
    }
    return values;
  });
  expect(Object.keys(storage)).toEqual([TOKEN_KEY]);
  expect(storage[TOKEN_KEY]).toMatch(/\S+/);
});
