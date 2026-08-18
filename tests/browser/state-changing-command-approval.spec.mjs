import { expect, test } from '@playwright/test';

const TOKEN_KEY = 'fallout-terminal.player-token';
const FIXTURE = '/__fixture/state-changing-command-approval';
const MASTER_URL = `${FIXTURE}/master`;
const REQUEST_ID = 'approval-request-1';
const COMMAND_NAME = 'Открыть двери';
const CONFIRMATION_TEXT = 'Разрешить доступ в защищённый сектор?';

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
  await journey.player.locator('.term-row', { hasText: COMMAND_NAME }).click();
  await expectFullScreenCommandSurface(journey.player, 'Выполняется запрос');

  const dialogs = journey.master.getByRole('dialog', { name: 'ПОДТВЕРЖДЕНИЕ КОМАНДЫ' });
  await expect(dialogs).toHaveCount(1);
  const dialog = dialogs.first();
  await expect(dialog).toBeVisible();
  await expect(dialog.locator('#commandExecutionDialogStatus')).toHaveText(`КОМАНДА: ${COMMAND_NAME}`);
  await expect(dialog.locator('#commandExecutionDialogDescription')).toHaveText(CONFIRMATION_TEXT);
  await expect(dialog.locator('#commandExecutionDialogDescription')).not.toContainText(COMMAND_NAME);
  return dialog;
}

function collectFixtureNodes(session) {
  const nodes = [];
  const visit = (node, isRoot = false) => {
    if (!isRoot) nodes.push(node);
    for (const child of node.children ?? []) visit(child);
  };
  for (const terminal of session.terminals ?? []) visit(terminal.root, true);
  return nodes;
}

async function expectFullScreenCommandSurface(page, text) {
  await expect(page.locator('#termEntry')).toBeVisible();
  await expect(page.locator('#entryBody')).toContainText(text);
  await expect(page.locator('#termOutput')).toBeHidden();
  await expect(page.locator('#termList')).toBeHidden();
  await expect(page.locator('#termPrompt')).toBeVisible();
}

async function completeVisibleReveal(page) {
  await page.keyboard.press('Shift');
}

async function recordRendererSnapshot(page) {
  return page.evaluate(() => {
    const surface = document.querySelector('#termEntry');
    const body = document.querySelector('#entryBody');
    const surfaceStyle = getComputedStyle(surface);
    const bodyStyle = getComputedStyle(body);
    const surfaceBounds = surface.getBoundingClientRect();
    const bodyBounds = body.getBoundingClientRect();
    const roundedBounds = bounds => ({
      x: Math.round(bounds.x),
      y: Math.round(bounds.y),
      width: Math.round(bounds.width),
      height: Math.round(bounds.height),
    });
    return {
      surface: {
        bounds: roundedBounds(surfaceBounds),
        display: surfaceStyle.display,
        flexDirection: surfaceStyle.flexDirection,
        overflow: surfaceStyle.overflow,
        padding: surfaceStyle.padding,
      },
      body: {
        bounds: roundedBounds(bodyBounds),
        fontFamily: bodyStyle.fontFamily,
        fontSize: bodyStyle.fontSize,
        lineHeight: bodyStyle.lineHeight,
        overflow: bodyStyle.overflow,
        overflowWrap: bodyStyle.overflowWrap,
        whiteSpace: bodyStyle.whiteSpace,
      },
    };
  });
}

async function pageCount(page) {
  const value = await page.locator('#pageIndicator').textContent();
  return Number.parseInt(value.split('/')[1], 10);
}

async function resolveCalls(master) {
  return master.evaluate(() => __desktopFixture.calls
    .filter(call => call.method === 'ResolveCommandExecution'));
}

test.beforeEach(async ({ request }) => {
  await resetApprovalFixture(request);
});

test('canonical approval input has explicit folder, entry, and only initial state-changing commands', async ({ request }) => {
  const response = await request.get(`${FIXTURE}/session`);
  expect(response.ok()).toBe(true);
  const session = await response.json();
  const nodes = collectFixtureNodes(session);
  const commands = nodes.filter(node => node.type === 'command');

  expect(nodes.some(node => node.type === 'folder')).toBe(true);
  expect(nodes.some(node => node.type === 'entry')).toBe(true);
  expect(commands.length).toBeGreaterThan(0);
  expect(session.terminals.flatMap(terminal => Object.keys(terminal.commandStates ?? {}))).toEqual([]);
  for (const command of commands) {
    expect(command.name.trim()).not.toBe('');
    expect(command.text.trim()).not.toBe('');
    expect(command.stateChange?.completedName?.trim()).not.toBe('');
    expect(command.stateChange?.confirmationText?.trim()).not.toBe('');
    expect(command.stateChange.confirmationText).not.toContain(command.name);
  }
});

test('one pending request opens exactly one master dialog and approve publishes a full-screen durable result', async ({ browser, request }) => {
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
    await expectFullScreenCommandSurface(journey.player, 'Доступ в сектор разрешён.');

    await journey.player.keyboard.press('Enter');
    await expect(journey.player.locator('.term-row', { hasText: 'Двери открыты' })).toBeVisible();
    await expect(journey.player.locator('.term-row', { hasText: 'Открыть двери' })).toHaveCount(0);
  } finally {
    await closeApprovalJourney(journey);
  }
});

test('pending full-screen request ignores Enter and Back until the master decides', async ({ browser }) => {
  const journey = await openApprovalJourney(browser);
  try {
    const dialog = await chooseStateChangingCommand(journey);
    await journey.player.keyboard.press('Enter');
    await journey.player.keyboard.press('Backspace');

    await expectFullScreenCommandSurface(journey.player, 'Выполняется запрос');
    await expect(dialog).toBeVisible();
    expect(await resolveCalls(journey.master)).toEqual([]);
    await expect(journey.player.locator('#backBtn')).toBeHidden();
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
    await expectFullScreenCommandSurface(journey.player, 'Ошибка доступа');
    await expect(journey.player.locator('#entryBody')).toHaveText('Ошибка доступа');
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
    await expectFullScreenCommandSurface(journey.player, 'Ошибка доступа');
    await expect(journey.player.locator('#entryBody')).toHaveText('Ошибка доступа');
    await expect(journey.player.locator('#entryBody')).not.toContainText('Выполняется запрос');

    await journey.player.keyboard.press('Enter');
    await expect(journey.player.locator('.term-row', { hasText: 'Открыть двери' })).toBeVisible();
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
    await expect(journey.player.locator('#entryBody')).not.toContainText('Доступ в сектор разрешён.');
    await expect(journey.player.locator('.term-row', { hasText: 'Открыть двери' })).toBeVisible();
    await expect(journey.player.locator('.term-row', { hasText: 'Двери открыты' })).toHaveCount(0);
  } finally {
    await closeApprovalJourney(journey);
  }
});

test('pending, rejected, and completed command states match the selected-record renderer', async ({ browser, request }) => {
  let journey = await openApprovalJourney(browser);
  try {
    await journey.player.locator('.term-row', { hasText: 'ЭТАЛОН РЕНДЕРА' }).click();
    await expect(journey.player.locator('#termEntry')).toBeVisible();
    await completeVisibleReveal(journey.player);
    const referenceWide = await recordRendererSnapshot(journey.player);
    const referenceWidePages = await pageCount(journey.player);
    expect(referenceWidePages).toBeGreaterThan(1);
    await journey.player.locator('#backBtn').click();
    await expect(journey.player.locator('.term-row', { hasText: 'Открыть двери' })).toBeVisible();

    let dialog = await chooseStateChangingCommand(journey);
    await completeVisibleReveal(journey.player);
    expect(await recordRendererSnapshot(journey.player)).toEqual(referenceWide);

    await dialog.getByRole('button', { name: 'ОТКЛОНИТЬ' }).click();
    await expectFullScreenCommandSurface(journey.player, 'Ошибка доступа');
    await completeVisibleReveal(journey.player);
    expect(await recordRendererSnapshot(journey.player)).toEqual(referenceWide);
    await journey.player.keyboard.press('Enter');
    await expect(journey.player.locator('.term-row', { hasText: 'Открыть двери' })).toBeVisible();

    await closeApprovalJourney(journey);
    await resetApprovalFixture(request);
    journey = await openApprovalJourney(browser);

    dialog = await chooseStateChangingCommand(journey);
    await dialog.getByRole('button', { name: 'ОДОБРИТЬ' }).click();
    await expectFullScreenCommandSurface(journey.player, 'Доступ в сектор разрешён.');
    await completeVisibleReveal(journey.player);
    expect(await recordRendererSnapshot(journey.player)).toEqual(referenceWide);
    await expect(journey.player.locator('#pageNext')).toBeVisible();

    await journey.player.setViewportSize({ width: 720, height: 520 });
    await expect.poll(() => pageCount(journey.player)).not.toBe(referenceWidePages);
    const completedNarrow = await recordRendererSnapshot(journey.player);
    const completedNarrowPages = await pageCount(journey.player);
    expect(completedNarrowPages).toBeGreaterThan(1);
    await journey.player.keyboard.press('Enter');

    await journey.player.locator('.term-row', { hasText: 'ЭТАЛОН РЕНДЕРА' }).click();
    await expect(journey.player.locator('#termEntry')).toBeVisible();
    await completeVisibleReveal(journey.player);
    expect(await recordRendererSnapshot(journey.player)).toEqual(completedNarrow);
    expect(await pageCount(journey.player)).toBe(completedNarrowPages);
    await journey.player.locator('#backBtn').click();

    await journey.player.locator('.term-row', { hasText: 'Двери открыты' }).click();
    await expectFullScreenCommandSurface(journey.player, 'Доступ в сектор разрешён.');
    await completeVisibleReveal(journey.player);
    expect(await recordRendererSnapshot(journey.player)).toEqual(completedNarrow);
    expect(await pageCount(journey.player)).toBe(completedNarrowPages);
  } finally {
    await closeApprovalJourney(journey);
  }
});
