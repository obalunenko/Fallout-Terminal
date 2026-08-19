import { expect, test } from '@playwright/test';

const FIXTURE = '/__fixture/terminal-navigation';
const MASTER = `${FIXTURE}/master`;

async function openMaster(page) {
  await page.goto(MASTER);
  await page.getByRole('button', { name: 'ОТКРЫТЬ СЕССИЮ' }).click();
  await expect(page.locator('#mainLayout')).toBeVisible();
}

async function selectCommand(page, name) {
  await page.locator('.tree-row', { hasText: name }).first().click();
  await expect(page.locator('#nodeForm')).toContainText('КОМАНДА');
}

async function openPlayer(browser) {
  const context = await browser.newContext();
  await context.addInitScript(() => {
    HTMLMediaElement.prototype.play = () => Promise.resolve();
    try { localStorage.removeItem('fallout-terminal.player-token'); } catch { /* about:blank */ }
  });
  const page = await context.newPage();
  await page.goto('/');
  await expect(page.locator('#connOverlay')).toBeHidden();
  await page.locator('#characterOptions button:not([disabled])').first().click();
  await expect(page.locator('#roleBadge')).toContainText('АКТИВНЫЙ');
  return { context, page };
}

async function openParticipant(browser, token = '') {
  const context = await browser.newContext();
  await context.addInitScript(({ tokenKey, retainedToken }) => {
    HTMLMediaElement.prototype.play = () => Promise.resolve();
    try {
      if (retainedToken) localStorage.setItem(tokenKey, retainedToken);
      else localStorage.removeItem(tokenKey);
    } catch { /* about:blank */ }
  }, { tokenKey: 'fallout-terminal.player-token', retainedToken: token });
  const page = await context.newPage();
  await page.goto('/');
  await expect(page.locator('#connOverlay')).toBeHidden();
  if (await page.locator('#characterSelect').isVisible()) {
    await page.locator('#characterOptions button:not([disabled])').first().click();
  }
  await expect(page.locator('#roleBadge')).toContainText(/АКТИВНЫЙ|НАБЛЮДАТЕЛЬ/);
  return { context, page };
}

async function approveForwardTransition(master, player) {
  await player.locator('.term-row', { hasText: 'ПЕРЕЙТИ В ОХРАНУ' }).click();
  await expectPendingTransitionSurface(player);
  const dialog = master.getByRole('dialog', { name: 'ПЕРЕХОД МЕЖДУ ТЕРМИНАЛАМИ' });
  await expect(dialog).toHaveCount(1);
  await dialog.getByRole('button', { name: 'ОДОБРИТЬ' }).click();
  await expect(player.locator('#hackHeader')).toBeVisible();
  await expect(player.locator('#playerNotice')).toBeHidden();
}

async function decideNavigation(master, decision) {
  const dialog = master.getByRole('dialog', { name: 'ПЕРЕХОД МЕЖДУ ТЕРМИНАЛАМИ' });
  await expect(dialog).toHaveCount(1);
  await dialog.getByRole('button', { name: decision === 'approve' ? 'ОДОБРИТЬ' : 'ОТКЛОНИТЬ' }).click();
  await expect(dialog).toBeHidden();
}

async function finishHack(request, player) {
  expect((await request.post(`${FIXTURE}/force-hack`)).ok()).toBe(true);
  await expect(player.locator('#hackHeader')).toBeHidden({ timeout: 5000 });
  await expect(player.locator('#termList')).toBeVisible();
}

async function expectPendingTransitionSurface(page, timeout = 2000) {
  await expect(page.locator('#termEntry')).toBeVisible({ timeout });
  await page.keyboard.press('Shift');
  await expect(page.locator('#entryBody')).toHaveText('Выполняется запрос', { timeout });
  await expect(page.locator('#termList')).toBeHidden({ timeout });
  await expect(page.locator('#termOutput')).toBeHidden({ timeout });
  await expect(page.locator('#termPrompt')).toBeVisible({ timeout });
  await expect(page.locator('#backBtn')).toBeHidden({ timeout });
  await expect(page.locator('#playerNotice')).toBeHidden({ timeout });
}

test.beforeEach(async ({ request }) => {
  const response = await request.post(`${FIXTURE}/reset`);
  expect(response.ok()).toBe(true);
});

test('transition authoring is mutually exclusive, validates locally, and survives reopen', async ({ page }) => {
  await openMaster(page);
  await selectCommand(page, 'ПЕРЕЙТИ В ОХРАНУ');
  const form = page.locator('#nodeForm');
  const transition = form.getByLabel('ПЕРЕХОД В ДРУГОЙ ТЕРМИНАЛ');
  const stateChange = form.getByLabel('ИЗМЕНЯЕТ СОСТОЯНИЕ');
  await transition.check();
  await expect(stateChange).not.toBeChecked();
  await form.getByLabel('ЦЕЛЕВОЙ ТЕРМИНАЛ').selectOption('security');
  await form.getByRole('button', { name: 'ПРИМЕНИТЬ' }).click();
  await expect.poll(() => page.evaluate(() => __desktopFixture.calls.filter(call => call.method === 'SaveSession').at(-1)?.args?.[0]
    ?.terminals?.[0]?.root?.children?.[0]?.terminalTransition?.targetTerminalId)).toBe('security');
  await expect.poll(() => page.evaluate(() => __desktopFixture.calls.filter(call => call.method === 'SaveSession').at(-1)?.args?.[0]
    ?.terminals?.map(terminal => terminal.id))).toEqual(['residential', 'security', 'vault']);

  await page.reload();
  await openMaster(page);
  await expect(page.locator('.term-row')).toHaveCount(3);
  await selectCommand(page, 'ПЕРЕЙТИ В ОХРАНУ');
  await expect(page.getByLabel('ПЕРЕХОД В ДРУГОЙ ТЕРМИНАЛ')).toBeChecked();
  await expect(page.getByLabel('ЦЕЛЕВОЙ ТЕРМИНАЛ')).toHaveValue('security');

  await selectCommand(page, 'ЗАВЕРШЁННАЯ КОМАНДА');
  await expect(page.getByLabel('ПЕРЕХОД В ДРУГОЙ ТЕРМИНАЛ')).toBeDisabled();
});

test('deleting a referenced terminal is blocked before local mutation', async ({ page }) => {
  await openMaster(page);
	const security = page.locator('.term-row', { hasText: 'Терминал охраны' });
  await security.getByRole('button', { name: 'УДАЛИТЬ' }).click();
	await expect(page.locator('#coordinationError')).toContainText(/ссыла|переход/i);
	await expect(page.locator('.term-row', { hasText: 'Терминал охраны' })).toHaveCount(1);
});

test('one forward request opens one exact master dialog and close rejects it', async ({ page, request }) => {
  await openMaster(page);
  const armed = await request.post(`${FIXTURE}/pending-forward`);
  expect(armed.ok()).toBe(true);
  const dialog = page.getByRole('dialog', { name: 'ПЕРЕХОД МЕЖДУ ТЕРМИНАЛАМИ' });
  await expect(dialog).toHaveCount(1);
  await expect(dialog).toContainText('ИЗ: Жилой терминал');
  await expect(dialog).toContainText('КОМАНДА: ПЕРЕЙТИ В ОХРАНУ');
  await expect(dialog).toContainText('В: Терминал охраны');
  await dialog.press('Escape');
  await expect.poll(() => page.evaluate(() => __desktopFixture.calls.filter(call => call.method === 'ResolveTerminalNavigation'))).toEqual([
    expect.objectContaining({ args: [{ requestId: 'navigation-forward-1', decision: 'reject' }] }),
  ]);
  await expect(dialog).toBeHidden();
});

test('approved first entry opens the destination hack at root without a terminal-switch decision', async ({ browser }) => {
  const masterContext = await browser.newContext();
  const master = await masterContext.newPage();
  await openMaster(master);
  const player = await openPlayer(browser);
  try {
    await approveForwardTransition(master, player.page);
    await expect(master.getByRole('dialog', { name: /СМЕНА ТЕРМИНАЛА/i })).toHaveCount(0);
    await expect(player.page.locator('#attemptsLine')).toContainText('ОСТАЛОСЬ');
  } finally {
    await player.context.close();
    await masterContext.close();
  }
});

test('direct pending replaces every player menu with the inert record surface across reconnect and decisions', async ({ browser, request }) => {
  const runDecision = async decision => {
    const masterContext = await browser.newContext();
    const master = await masterContext.newPage();
    await openMaster(master);
    const controller = await openParticipant(browser);
    const firstObserver = await openParticipant(browser);
    let secondObserver = await openParticipant(browser);
    try {
      await expect(controller.page.locator('#roleBadge')).toContainText('АКТИВНЫЙ');
      await expect(firstObserver.page.locator('#roleBadge')).toContainText('НАБЛЮДАТЕЛЬ');
      await expect(secondObserver.page.locator('#roleBadge')).toContainText('НАБЛЮДАТЕЛЬ');
      const sourceMenu = await controller.page.locator('#termList').textContent();

      await controller.page.locator('.term-row', { hasText: 'ПЕРЕЙТИ В ОХРАНУ' }).first().click();
      await Promise.all([controller, firstObserver, secondObserver].map(participant =>
        expectPendingTransitionSurface(participant.page)));

      const retainedToken = await secondObserver.page.evaluate(() =>
        localStorage.getItem('fallout-terminal.player-token'));
      await secondObserver.context.close();
      secondObserver = await openParticipant(browser, retainedToken);
      await expectPendingTransitionSurface(secondObserver.page);

      for (const participant of [controller, firstObserver, secondObserver]) {
        await participant.page.keyboard.press('Enter');
        await participant.page.keyboard.press('Backspace');
        await expectPendingTransitionSurface(participant.page);
      }

      const dialog = master.getByRole('dialog', { name: 'ПЕРЕХОД МЕЖДУ ТЕРМИНАЛАМИ' });
      await expect(dialog).toBeVisible();
      if (decision === 'close') await dialog.press('Escape');
      else await dialog.getByRole('button', { name: decision === 'approve' ? 'ОДОБРИТЬ' : 'ОТКЛОНИТЬ' }).click();
      await expect(dialog).toBeHidden();

      if (decision === 'approve') {
        await Promise.all([controller, firstObserver, secondObserver].map(participant =>
          expect(participant.page.locator('#hackHeader')).toBeVisible({ timeout: 2000 })));
      } else {
        await Promise.all([controller, firstObserver, secondObserver].map(async participant => {
          await expect(participant.page.locator('#termList')).toBeVisible({ timeout: 2000 });
          await expect(participant.page.locator('#termList')).toHaveText(sourceMenu);
          await expect(participant.page.locator('#termEntry')).toBeHidden();
        }));
      }
    } finally {
      await controller.context.close();
      await firstObserver.context.close();
      await secondObserver.context.close();
      await masterContext.close();
    }
  };

  await runDecision('approve');
  expect((await request.post(FIXTURE + '/reset')).ok()).toBe(true);
  await runDecision('reject');
  expect((await request.post(FIXTURE + '/reset')).ok()).toBe(true);
  await runDecision('close');
});

test('an unfinished destination hack resumes with the exact retained progress', async ({ browser, request }) => {
  const masterContext = await browser.newContext();
  const master = await masterContext.newPage();
  await openMaster(master);
  const player = await openPlayer(browser);
  try {
    await approveForwardTransition(master, player.page);
    const words = player.page.locator('.hcell.word');
    await expect(words.first()).toBeVisible();
    const initialAttempts = await player.page.locator('#attemptsLine').textContent();
    await words.first().click();
    await expect.poll(async () => player.page.locator('#attemptsLine').textContent()).not.toBe(initialAttempts);
    const retainedAttempts = await player.page.locator('#attemptsLine').textContent();
    const retainedLog = await player.page.locator('#hackLog').textContent();

    expect((await request.post(`${FIXTURE}/switch-source`)).ok()).toBe(true);
    await expect(player.page.locator('.term-row', { hasText: 'ПЕРЕЙТИ В ОХРАНУ' })).toBeVisible();
    expect((await request.post(`${FIXTURE}/switch-target`)).ok()).toBe(true);
    await expect(player.page.locator('#hackHeader')).toBeVisible();
    await expect(player.page.locator('#attemptsLine')).toHaveText(retainedAttempts);
    await expect(player.page.locator('#hackLog')).toHaveText(retainedLog);
  } finally {
    await player.context.close();
    await masterContext.close();
  }
});

test('root return stays pending on reject, then restores the nested source menu on approve', async ({ browser, request }) => {
  const masterContext = await browser.newContext();
  const master = await masterContext.newPage();
  await openMaster(master);
  const player = await openPlayer(browser);
  try {
    await player.page.locator('.term-row', { hasText: 'НАВИГАЦИЯ' }).click();
    await player.page.locator('.term-row', { hasText: 'ПЕРЕЙТИ В ОХРАНУ ИЗ ПАПКИ' }).click();
    await decideNavigation(master, 'approve');
    await expect(player.page.locator('#hackHeader')).toBeVisible();
    await finishHack(request, player.page);
		expect((await request.post(`${FIXTURE}/move-source-folder`)).ok()).toBe(true);

    const returnButton = player.page.getByRole('button', { name: /НАЗАД В Жилой терминал/i });
    await expect(returnButton).toBeVisible();
    await returnButton.click();
    await expect(returnButton).toBeDisabled();
    await decideNavigation(master, 'reject');
    await expect(returnButton).toBeEnabled();
    await returnButton.click();
    await decideNavigation(master, 'approve');

    await expect(player.page.locator('.term-row', { hasText: 'ПЕРЕЙТИ В ОХРАНУ ИЗ ПАПКИ' })).toBeVisible();
    await expect(player.page.getByRole('button', { name: /НАЗАД В/i })).toHaveCount(0);
  } finally {
    await player.context.close();
    await masterContext.close();
  }
});

test('deleted source folder falls back to root without losing the terminal route', async ({ browser, request }) => {
  const masterContext = await browser.newContext();
  const master = await masterContext.newPage();
  await openMaster(master);
  const player = await openPlayer(browser);
  try {
    await player.page.locator('.term-row', { hasText: 'НАВИГАЦИЯ' }).click();
    await player.page.locator('.term-row', { hasText: 'ПЕРЕЙТИ В ОХРАНУ ИЗ ПАПКИ' }).click();
    await decideNavigation(master, 'approve');
    await finishHack(request, player.page);
    expect((await request.post(`${FIXTURE}/delete-source-folder`)).ok()).toBe(true);
    await player.page.getByRole('button', { name: /НАЗАД В Жилой терминал/i }).click();
    await decideNavigation(master, 'approve');
    await expect(player.page.locator('.term-row', { hasText: 'ПЕРЕЙТИ В ОХРАНУ' }).first()).toBeVisible();
    await expect(player.page.locator('.term-row', { hasText: 'НАВИГАЦИЯ' })).toHaveCount(0);
  } finally {
    await player.context.close();
    await masterContext.close();
  }
});

test('controller and two observers reconnect during pending and converge before new-broadcast cleanup', async ({ browser, request }) => {
  const masterContext = await browser.newContext();
  const master = await masterContext.newPage();
  await openMaster(master);
  const controller = await openParticipant(browser);
  let observer = await openParticipant(browser);
  const secondObserver = await openParticipant(browser);
  try {
    await expect(controller.page.locator('#roleBadge')).toContainText('АКТИВНЫЙ');
    await expect(observer.page.locator('#roleBadge')).toContainText('НАБЛЮДАТЕЛЬ');
    await expect(secondObserver.page.locator('#roleBadge')).toContainText('НАБЛЮДАТЕЛЬ');
    await controller.page.locator('.term-row', { hasText: 'ПЕРЕЙТИ В ОХРАНУ' }).first().click();
    await Promise.all([controller, observer, secondObserver].map(participant =>
      expectPendingTransitionSurface(participant.page)));
    await Promise.all([controller, observer, secondObserver].map(async participant => {
      await participant.page.keyboard.press('Enter');
      await participant.page.keyboard.press('Backspace');
      await expectPendingTransitionSurface(participant.page);
    }));

    const token = await observer.page.evaluate(() => localStorage.getItem('fallout-terminal.player-token'));
    await observer.context.close();
    observer = await openParticipant(browser, token);
    await expectPendingTransitionSurface(observer.page);
    await decideNavigation(master, 'approve');
    await Promise.all([controller, observer, secondObserver].map(participant =>
      expect(participant.page.locator('#hackHeader')).toBeVisible({ timeout: 2000 })));

    expect((await request.post(`${FIXTURE}/new-broadcast`)).ok()).toBe(true);
    await Promise.all([controller, observer, secondObserver].map(participant =>
      expect(participant.page.locator('#characterSelect')).toBeVisible({ timeout: 2000 })));
    await Promise.all([controller, observer, secondObserver].map(participant =>
      expect(participant.page.getByRole('button', { name: /НАЗАД В/i })).toHaveCount(0)));
  } finally {
    await controller.context.close();
    await observer.context.close();
    await secondObserver.context.close();
    await masterContext.close();
  }
});

test('stale target approval fails safely and keeps the source terminal active', async ({ browser, request }) => {
  const masterContext = await browser.newContext();
  const master = await masterContext.newPage();
  await openMaster(master);
  const player = await openPlayer(browser);
  try {
    await player.page.locator('.term-row', { hasText: 'ПЕРЕЙТИ В ОХРАНУ' }).first().click();
    expect((await request.post(`${FIXTURE}/remove-target`)).ok()).toBe(true);
    await decideNavigation(master, 'approve');
    await expect(master.locator('#coordinationError')).toContainText(/ИЗМЕНИЛАСЬ|НЕ СУЩЕСТВУЕТ|no longer|changed/i);
    await expect(player.page.locator('.term-row', { hasText: 'ПЕРЕЙТИ В ОХРАНУ' }).first()).toBeVisible();
    await expect(player.page.locator('#hackHeader')).toBeHidden();
  } finally {
    await player.context.close();
    await masterContext.close();
  }
});

test('A to B to C returns unwind exactly B then A', async ({ browser, request }) => {
  const masterContext = await browser.newContext();
  const master = await masterContext.newPage();
  await openMaster(master);
  const player = await openPlayer(browser);
  try {
    await player.page.locator('.term-row', { hasText: 'ПЕРЕЙТИ В ОХРАНУ' }).first().click();
    await decideNavigation(master, 'approve');
    await finishHack(request, player.page);
    await player.page.locator('.term-row', { hasText: 'ПЕРЕЙТИ В ХРАНИЛИЩЕ' }).click();
    await decideNavigation(master, 'approve');

    await player.page.getByRole('button', { name: /НАЗАД В Терминал охраны/i }).click();
    await decideNavigation(master, 'approve');
    await expect(player.page.locator('.term-row', { hasText: 'ПЕРЕЙТИ В ХРАНИЛИЩЕ' })).toBeVisible();
    await player.page.getByRole('button', { name: /НАЗАД В Жилой терминал/i }).click();
    await decideNavigation(master, 'approve');
    await expect(player.page.locator('.term-row', { hasText: 'ПЕРЕЙТИ В ОХРАНУ' }).first()).toBeVisible();
    await expect(player.page.getByRole('button', { name: /НАЗАД В/i })).toHaveCount(0);
  } finally {
    await player.context.close();
    await masterContext.close();
  }
});
