import { expect, test } from '@playwright/test';

const TOKEN_KEY = 'fallout-terminal.player-token';
const FIXTURE = '/__fixture/state-changing-command-sync';
const MASTER_URL = `${FIXTURE}/master`;

async function postLifecycle(request, action) {
  const response = await request.post(`${FIXTURE}/${action}`);
  expect(response.status()).toBe(204);
}

async function installPlayerDiagnostics(target, storedToken = null) {
  await target.addInitScript(({ tokenKey, token }) => {
    window.__webSocketConstructions = 0;
    window.WebSocket = class ForbiddenLegacyPlayerTransport {
      constructor() {
        window.__webSocketConstructions += 1;
        throw new Error('the player must use generated ConnectRPC');
      }
    };
    HTMLMediaElement.prototype.play = () => Promise.resolve();
    if (token === null) localStorage.removeItem(tokenKey);
    else localStorage.setItem(tokenKey, token);
  }, { tokenKey: TOKEN_KEY, token: storedToken });
}

async function openPlayer(browser, storedToken = null) {
  const context = await browser.newContext();
  await installPlayerDiagnostics(context, storedToken);
  const page = await context.newPage();
  await page.goto('/');
  await expect(page.locator('#connOverlay')).toBeHidden();
  await expect.poll(() => page.evaluate(() => window.__webSocketConstructions)).toBe(0);
  return { context, page };
}

async function openMaster(browser) {
  const context = await browser.newContext();
  const page = await context.newPage();
  await page.goto(MASTER_URL);
  await page.getByRole('button', { name: 'ОТКРЫТЬ СЕССИЮ' }).click();
  await expect(page.locator('#mainLayout')).toBeVisible();
  return { context, page };
}

async function selectFirstAvailable(page) {
  const option = page.locator('#characterOptions button:not([disabled])').first();
  await expect(option).toBeVisible();
  await option.click();
  await expect(page.locator('#characterSelect')).not.toHaveAttribute('aria-busy', 'true');
}

async function assignControllerAndObservers(players, expectedMenuTitle = 'Открыть двери') {
  for (const player of players) await selectFirstAvailable(player.page);
  await expect(players[0].page.locator('#roleBadge')).toContainText('АКТИВНЫЙ');
  await Promise.all(players.slice(1).map(player =>
    expect(player.page.locator('#roleBadge')).toContainText('НАБЛЮДАТЕЛЬ')));
  await Promise.all(players.map(player =>
    expect(player.page.locator('.term-row', { hasText: expectedMenuTitle })).toBeVisible()));
}

async function openThreePlayerJourney(browser) {
  const master = await openMaster(browser);
  const players = [];
  for (let index = 0; index < 3; index += 1) players.push(await openPlayer(browser));
  await assignControllerAndObservers(players);
  return { master, players };
}

async function closeJourney(journey) {
  for (const player of journey.players) {
    await player.context.close().catch(() => {});
  }
  await journey.master.context.close().catch(() => {});
}

async function chooseStateChangingCommand(journey) {
  await journey.players[0].page.locator('.term-row', { hasText: 'Открыть двери' }).click();
  await Promise.all(journey.players.map(player =>
    expect(player.page.locator('#termOutput')).toHaveText('Выполняется запрос')));

  const dialog = journey.master.page.getByRole('dialog', { name: 'ПОДТВЕРЖДЕНИЕ КОМАНДЫ' });
  await expect(dialog).toBeVisible();
  await expect(dialog).toContainText('Открыть двери? Это действие нельзя повторить.');
  return dialog;
}

async function audit(request) {
  const response = await request.get(`${FIXTURE}/audit`);
  expect(response.ok()).toBe(true);
  return response.json();
}

test.beforeEach(async ({ request }) => {
  await postLifecycle(request, 'reset');
});

test('controller and two observers converge on completed result and title within one second', async ({ browser, request }) => {
  const journey = await openThreePlayerJourney(browser);
  try {
    const dialog = await chooseStateChangingCommand(journey);
    const startedAt = Date.now();
    await dialog.getByRole('button', { name: 'ОДОБРИТЬ' }).click();
    await Promise.all(journey.players.map(player =>
      expect(player.page.locator('#termOutput')).toHaveText('Доступ в сектор разрешён.', { timeout: 1000 })));

    await journey.players[0].page.locator('#backBtn').click();
    const remaining = Math.max(1, 1000 - (Date.now() - startedAt));
    await Promise.all(journey.players.map(player =>
      expect(player.page.locator('.term-row', { hasText: 'Двери открыты' })).toBeVisible({ timeout: remaining })));
    expect(Date.now() - startedAt).toBeLessThanOrEqual(1000);
    expect(await audit(request)).toMatchObject({ executeWrites: 1, pendingRequests: 0, completed: true });
  } finally {
    await closeJourney(journey);
  }
});

test('controller disconnect keeps one pending request and durable completion survives every shared lifecycle', async ({ browser, request }) => {
  const journey = await openThreePlayerJourney(browser);
  try {
    const dialog = await chooseStateChangingCommand(journey);
    const controllerToken = await journey.players[0].page.evaluate(key => localStorage.getItem(key), TOKEN_KEY);
    expect(controllerToken).toMatch(/\S+/);

    await journey.players[0].context.close();
    await Promise.all(journey.players.slice(1).map(player =>
      expect(player.page.locator('#termOutput')).toHaveText('Выполняется запрос')));
    await expect(dialog).toBeVisible();
    await expect.poll(async () => (await audit(request)).pendingRequests).toBe(1);

    await dialog.getByRole('button', { name: 'ОДОБРИТЬ' }).click();
    await expect(dialog).toBeHidden();
    await Promise.all(journey.players.slice(1).map(player =>
      expect(player.page.locator('#termOutput')).toHaveText('Доступ в сектор разрешён.')));

    const reconnected = await openPlayer(browser, controllerToken);
    journey.players[0] = reconnected;
    await expect(reconnected.page.locator('#roleBadge')).toContainText('АКТИВНЫЙ');
    await expect(reconnected.page.locator('#termOutput')).toHaveText('Доступ в сектор разрешён.');
    expect(await reconnected.page.evaluate(key => localStorage.getItem(key), TOKEN_KEY)).toBe(controllerToken);

    await reconnected.page.locator('#backBtn').click();
    await Promise.all(journey.players.map(player =>
      expect(player.page.locator('.term-row', { hasText: 'Двери открыты' })).toBeVisible()));
    await expect(reconnected.page.locator('.term-row', { hasText: 'Открыть двери' })).toHaveCount(0);

    for (let cycle = 0; cycle < 10; cycle += 1) {
      await reconnected.page.locator('.term-row', { hasText: 'АРХИВ' }).click();
      await Promise.all(journey.players.map(player =>
        expect(player.page.locator('.term-row', { hasText: 'ЖУРНАЛ' })).toBeVisible()));
      await reconnected.page.locator('#backBtn').click();
      await Promise.all(journey.players.map(player =>
        expect(player.page.locator('.term-row', { hasText: 'Двери открыты' })).toBeVisible()));
    }

    await postLifecycle(request, 'switch-away');
    await Promise.all(journey.players.map(player =>
      expect(player.page.locator('.term-row', { hasText: 'РЕЗЕРВНЫЙ СТАТУС' })).toBeVisible()));
    await postLifecycle(request, 'switch-back');
    await Promise.all(journey.players.map(player =>
      expect(player.page.locator('.term-row', { hasText: 'Двери открыты' })).toBeVisible()));

    await postLifecycle(request, 'restart-broadcast');
    await Promise.all(journey.players.map(player => expect(player.page.locator('#characterSelect')).toBeVisible()));
    await assignControllerAndObservers(journey.players, 'Двери открыты');
    await Promise.all(journey.players.map(player =>
      expect(player.page.locator('.term-row', { hasText: 'Двери открыты' })).toBeVisible()));

    await postLifecycle(request, 'reopen-session');
    await Promise.all(journey.players.map(player => expect(player.page.locator('#characterSelect')).toBeVisible()));
    await assignControllerAndObservers(journey.players, 'Двери открыты');
    await Promise.all(journey.players.map(player =>
      expect(player.page.locator('.term-row', { hasText: 'Двери открыты' })).toBeVisible()));

    const state = await audit(request);
    expect(state).toMatchObject({ executeWrites: 1, pendingRequests: 0, completed: true });
    for (const player of journey.players) {
      const keys = await player.page.evaluate(() =>
        Array.from({ length: localStorage.length }, (_, index) => localStorage.key(index)));
      expect(keys).toEqual([TOKEN_KEY]);
    }
  } finally {
    await closeJourney(journey);
  }
});

test('terminal switch, broadcast restart, and reopen cancel transient pending or rejected state without a write', async ({ browser, request }) => {
  const journey = await openThreePlayerJourney(browser);
  try {
    let dialog = await chooseStateChangingCommand(journey);
    await postLifecycle(request, 'switch-away');
    await expect(dialog).toBeHidden();
    await Promise.all(journey.players.map(player =>
      expect(player.page.locator('.term-row', { hasText: 'РЕЗЕРВНЫЙ СТАТУС' })).toBeVisible()));
    expect(await audit(request)).toMatchObject({ executeWrites: 0, pendingRequests: 0, completed: false });

    await postLifecycle(request, 'switch-back');
    await Promise.all(journey.players.map(player =>
      expect(player.page.locator('.term-row', { hasText: 'Открыть двери' })).toBeVisible()));

    dialog = await chooseStateChangingCommand(journey);
    await dialog.getByRole('button', { name: 'ОТКЛОНИТЬ' }).click();
    await expect(dialog).toBeHidden();
    await Promise.all(journey.players.map(player =>
      expect(player.page.locator('#termOutput')).toHaveText('Запрос отклонён')));

    await postLifecycle(request, 'restart-broadcast');
    await Promise.all(journey.players.map(player => expect(player.page.locator('#characterSelect')).toBeVisible()));
    await assignControllerAndObservers(journey.players);
    await Promise.all(journey.players.map(player =>
      expect(player.page.locator('.term-row', { hasText: 'Открыть двери' })).toBeVisible()));
    await expect(journey.players[0].page.locator('#termOutput')).not.toContainText('Запрос отклонён');

    dialog = await chooseStateChangingCommand(journey);
    await postLifecycle(request, 'reopen-session');
    await expect(dialog).toBeHidden();
    await Promise.all(journey.players.map(player => expect(player.page.locator('#characterSelect')).toBeVisible()));
    await assignControllerAndObservers(journey.players);
    await Promise.all(journey.players.map(player =>
      expect(player.page.locator('.term-row', { hasText: 'Открыть двери' })).toBeVisible()));

    expect(await audit(request)).toMatchObject({ executeWrites: 0, pendingRequests: 0, completed: false });
  } finally {
    await closeJourney(journey);
  }
});
