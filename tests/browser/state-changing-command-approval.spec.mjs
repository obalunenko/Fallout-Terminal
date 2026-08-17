import { expect, test } from '@playwright/test';

const TOKEN_KEY = 'fallout-terminal.player-token';
const FIXTURE = '/__fixture/state-changing-command-approval';
const MASTER_URL = `${FIXTURE}/master`;
const REQUEST_ID = 'approval-request-1';

async function resetApprovalFixture(request) {
  const response = await request.post(`${FIXTURE}/reset`);
  expect(response.status()).toBe(204);
}

async function installPlayerDiagnostics(context) {
  await context.addInitScript((tokenKey) => {
    window.__webSocketConstructions = 0;
    window.WebSocket = class ForbiddenLegacyPlayerTransport {
      constructor() {
        window.__webSocketConstructions += 1;
        throw new Error('the player must use generated ConnectRPC');
      }
    };
    HTMLMediaElement.prototype.play = () => Promise.resolve();
    localStorage.removeItem(tokenKey);
  }, TOKEN_KEY);
}

async function openApprovalJourney(browser) {
  const masterContext = await browser.newContext();
  const playerContext = await browser.newContext();
  await installPlayerDiagnostics(playerContext);

  const master = await masterContext.newPage();
  await master.goto(MASTER_URL);
  await master.getByRole('button', { name: 'ОТКРЫТЬ СЕССИЮ' }).click();
  await expect(master.locator('#mainLayout')).toBeVisible();

  const player = await playerContext.newPage();
  await player.goto('/');
  await expect(player.locator('#connOverlay')).toBeHidden();
  await expect.poll(() => player.evaluate(() => window.__webSocketConstructions)).toBe(0);
  const character = player.locator('#characterOptions button:not([disabled])').first();
  await expect(character).toBeVisible();
  await character.click();
  await expect(player.locator('#roleBadge')).toContainText('АКТИВНЫЙ');
  await expect(player.locator('.term-row', { hasText: 'Открыть двери' })).toBeVisible();

  return { master, masterContext, player, playerContext };
}

async function closeApprovalJourney(journey) {
  await journey.playerContext.close();
  await journey.masterContext.close();
}

async function chooseStateChangingCommand(journey) {
  await journey.player.locator('.term-row', { hasText: 'Открыть двери' }).click();
  await expect(journey.player.locator('#termOutput')).toContainText('Выполняется запрос');

  const dialogs = journey.master.getByRole('dialog', { name: 'ПОДТВЕРЖДЕНИЕ КОМАНДЫ' });
  await expect(dialogs).toHaveCount(1);
  const dialog = dialogs.first();
  await expect(dialog).toBeVisible();
  await expect(dialog).toContainText('Открыть двери? Это действие нельзя повторить.');
  return dialog;
}

async function resolveCalls(master) {
  return master.evaluate(() => __desktopFixture.calls
    .filter(call => call.method === 'ResolveCommandExecution'));
}

test.beforeEach(async ({ request }) => {
  await resetApprovalFixture(request);
});

test('one pending request opens exactly one master dialog and approve publishes the durable result', async ({ browser, request }) => {
  const journey = await openApprovalJourney(browser);
  try {
    const dialog = await chooseStateChangingCommand(journey);

    const replay = await request.post(`${FIXTURE}/reemit-pending`);
    expect(replay.status()).toBe(204);
    await expect(journey.master.getByRole('dialog', { name: 'ПОДТВЕРЖДЕНИЕ КОМАНДЫ' })).toHaveCount(1);
    expect(await resolveCalls(journey.master)).toEqual([]);

    await dialog.getByRole('button', { name: 'ОДОБРИТЬ' }).click();
    await expect.poll(() => resolveCalls(journey.master)).toEqual([
      expect.objectContaining({
        args: [{ requestId: REQUEST_ID, decision: 'approve' }],
      }),
    ]);
    await expect(dialog).toBeHidden();
    await expect(journey.player.locator('#termOutput')).toContainText('Доступ в сектор разрешён.');

    await journey.player.locator('#backBtn').click();
    await expect(journey.player.locator('.term-row', { hasText: 'Двери открыты' })).toBeVisible();
    await expect(journey.player.locator('.term-row', { hasText: 'Открыть двери' })).toHaveCount(0);
  } finally {
    await closeApprovalJourney(journey);
  }
});

test('reject leaves the command initial and lets the controller return to the same menu', async ({ browser }) => {
  const journey = await openApprovalJourney(browser);
  try {
    const dialog = await chooseStateChangingCommand(journey);
    await dialog.getByRole('button', { name: 'ОТКЛОНИТЬ' }).click();

    await expect.poll(() => resolveCalls(journey.master)).toEqual([
      expect.objectContaining({
        args: [{ requestId: REQUEST_ID, decision: 'reject' }],
      }),
    ]);
    await expect(dialog).toBeHidden();
    await expect(journey.player.locator('#termOutput')).toHaveText('Запрос отклонён');
    await expect(journey.player.locator('#backBtn')).toBeVisible();

    await journey.player.locator('#backBtn').click();
    await expect(journey.player.locator('.term-row', { hasText: 'Открыть двери' })).toBeVisible();
    await expect(journey.player.locator('.term-row', { hasText: 'Двери открыты' })).toHaveCount(0);
  } finally {
    await closeApprovalJourney(journey);
  }
});

test('closing the master dialog is exactly one rejection and never leaves players pending', async ({ browser }) => {
  const journey = await openApprovalJourney(browser);
  try {
    const dialog = await chooseStateChangingCommand(journey);
    await dialog.press('Escape');

    await expect.poll(() => resolveCalls(journey.master)).toEqual([
      expect.objectContaining({
        args: [{ requestId: REQUEST_ID, decision: 'reject' }],
      }),
    ]);
    await expect(dialog).toBeHidden();
    await expect(journey.player.locator('#termOutput')).toHaveText('Запрос отклонён');
    await expect(journey.player.locator('#termOutput')).not.toContainText('Выполняется запрос');
  } finally {
    await closeApprovalJourney(journey);
  }
});

test('approve persistence failure exposes no completed result and reports safe errors', async ({ browser, request }) => {
  const armed = await request.post(`${FIXTURE}/fail-next-save`);
  expect(armed.status()).toBe(204);

  const journey = await openApprovalJourney(browser);
  try {
    const dialog = await chooseStateChangingCommand(journey);
    await dialog.getByRole('button', { name: 'ОДОБРИТЬ' }).click();

    await expect.poll(() => resolveCalls(journey.master)).toEqual([
      expect.objectContaining({
        args: [{ requestId: REQUEST_ID, decision: 'approve' }],
      }),
    ]);
    await expect(dialog).toBeHidden();
    const masterError = journey.master.getByRole('alert').filter({ hasText: /сохран|состояни/i });
    await expect(masterError).toBeVisible();
    await expect(masterError).not.toContainText(/\/private\/|rename|fsync|temporary file/i);

    await expect(journey.player.locator('#playerNotice')).toContainText(/сохран|состояние команды не изменено/i);
    await expect(journey.player.locator('#termOutput')).not.toContainText('Доступ в сектор разрешён.');
    await expect(journey.player.locator('.term-row', { hasText: 'Открыть двери' })).toBeVisible();
    await expect(journey.player.locator('.term-row', { hasText: 'Двери открыты' })).toHaveCount(0);
  } finally {
    await closeApprovalJourney(journey);
  }
});
