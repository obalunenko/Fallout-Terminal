import { expect, test } from '@playwright/test';
import { readFile } from 'node:fs/promises';

const FIXTURE_URL = '/__fixture/state-changing-command-authoring';
const TERMINAL_ID = 'terminal-stateful';
const BUNDLED_DEMO_URL = new URL('../../sessions/demo.json', import.meta.url);

async function openAuthoringFixture(page) {
  await page.goto(FIXTURE_URL);
  await page.getByRole('button', { name: 'ОТКРЫТЬ СЕССИЮ' }).click();
  await expect(page.locator('#mainLayout')).toBeVisible();
  await expect(page.locator('#editingTermName')).toHaveText('Терминал охраны');
}

async function selectCommand(page, displayedName) {
  const row = page.locator('.tree-row', { hasText: displayedName });
  await expect(row).toHaveCount(1);
  await row.click();
  await expect(page.locator('#nodeForm')).toContainText('КОМАНДА');
}

async function desktopCallCount(page, method) {
  return page.evaluate(name => __desktopFixture.calls.filter(call => call.method === name).length, method);
}

async function lastDesktopCall(page, method) {
  return page.evaluate(name => __desktopFixture.calls.filter(call => call.method === name).at(-1) ?? null, method);
}

async function commandFromLastSave(page, commandID) {
  return page.evaluate(({ id }) => {
    const call = __desktopFixture.calls.filter(candidate => candidate.method === 'SaveSession').at(-1);
    const session = call?.args?.[0];
    const visit = (node) => {
      if (!node || typeof node !== 'object') return null;
      if (node.id === id) return node;
      for (const child of node.children ?? []) {
        const found = visit(child);
        if (found) return found;
      }
      return null;
    };
    return visit(session?.terminals?.[0]?.root);
  }, { id: commandID });
}

async function authoringDurableState(page) {
  return page.evaluate(() => __desktopFixture.authoringDurableState());
}

test.beforeEach(async ({ page }) => {
  const reset = await page.request.post(`${FIXTURE_URL}/reset`);
  expect(reset.ok()).toBe(true);
  await openAuthoringFixture(page);
});

test('bundled read-only demo exposes state-changing and terminal-transition examples in their initial state', async () => {
  const demo = JSON.parse(await readFile(BUNDLED_DEMO_URL, 'utf8'));
  const terminalIDs = new Set(demo.terminals.map(terminal => terminal.id));
  expect([...terminalIDs]).toEqual(['t_demo1', 't_demo2']);
  const nodes = demo.terminals.flatMap((terminal) => {
    const collected = [];
    const visit = (node, isRoot = false) => {
      if (!isRoot) collected.push(node);
      for (const child of node.children ?? []) visit(child);
    };
    visit(terminal.root, true);
    return collected;
  });
  const commands = nodes.filter(node => node.type === 'command');
  const stateChangingCommands = commands.filter(command => command.stateChange);
  const terminalTransitionCommands = commands.filter(command => command.terminalTransition);
  const completed = demo.terminals.flatMap(terminal => Object.keys(terminal.commandStates ?? {}));

  expect(nodes.some(node => node.type === 'folder')).toBe(true);
  expect(nodes.some(node => node.type === 'entry')).toBe(true);
  expect(stateChangingCommands.length).toBeGreaterThan(0);
  for (const command of stateChangingCommands) {
    expect(command.name.trim()).not.toBe('');
    expect(command.text.trim()).not.toBe('');
    expect(command.stateChange?.completedName?.trim()).not.toBe('');
    expect(command.stateChange?.confirmationText?.trim()).not.toBe('');
    expect(command.stateChange.confirmationText).not.toContain(command.name);
    expect(command.terminalTransition).toBeUndefined();
  }
  expect(terminalTransitionCommands.length).toBeGreaterThan(0);
  for (const command of terminalTransitionCommands) {
    expect(command.name.trim()).not.toBe('');
    expect(command.stateChange).toBeUndefined();
    expect(command.terminalTransition.targetTerminalId.trim()).not.toBe('');
    expect(terminalIDs.has(command.terminalTransition.targetTerminalId)).toBe(true);
  }
  expect(completed).toEqual([]);
});

test('state-change toggle requires all four authored texts and persists an optional config', async ({ page }) => {
  await selectCommand(page, 'Включить аварийный свет');

  const form = page.locator('#nodeForm');
  const enabled = form.getByLabel('ИЗМЕНЯЕТ СОСТОЯНИЕ');
  const initialName = form.getByLabel('ИСХОДНОЕ НАЗВАНИЕ');
  const completedName = form.getByLabel('НАЗВАНИЕ ПОСЛЕ ВЫПОЛНЕНИЯ');
  const confirmationText = form.getByLabel('ТЕКСТ ЗАПРОСА ПОДТВЕРЖДЕНИЯ');
  const successText = form.getByLabel('ТЕКСТ УСПЕШНОГО ВЫПОЛНЕНИЯ');

  await expect(enabled).not.toBeChecked();
  await expect(completedName).toBeHidden();
  await expect(confirmationText).toBeHidden();
  await enabled.check();
  await expect(completedName).toBeVisible();
  await expect(confirmationText).toBeVisible();

  const authored = [
    { field: initialName, value: 'Включить красный свет', error: /исходн.*назван/i },
    { field: completedName, value: 'Красный свет включён', error: /назван.*после/i },
    { field: confirmationText, value: 'Включить аварийное освещение?', error: /подтвержден|запрос/i },
    { field: successText, value: 'Аварийное освещение включено.', error: /успешн|результат/i },
  ];
  for (const item of authored) await item.field.fill(item.value);

  const saveCountBeforeValidation = await desktopCallCount(page, 'SaveSession');
  for (const item of authored) {
    await item.field.fill(' \t ');
    await form.getByRole('button', { name: 'ПРИМЕНИТЬ' }).click();
    await expect(form.getByRole('alert')).toContainText(item.error);
    await expect.poll(() => desktopCallCount(page, 'SaveSession')).toBe(saveCountBeforeValidation);
    await item.field.fill(item.value);
  }

  await form.getByRole('button', { name: 'ПРИМЕНИТЬ' }).click();
  await expect.poll(() => desktopCallCount(page, 'SaveSession')).toBe(saveCountBeforeValidation + 1);
  await expect.poll(() => commandFromLastSave(page, 'emergency-lights')).toEqual(expect.objectContaining({
    id: 'emergency-lights',
    type: 'command',
    name: 'Включить красный свет',
    text: 'Аварийное освещение включено.',
    stateChange: {
      completedName: 'Красный свет включён',
      confirmationText: 'Включить аварийное освещение?',
    },
  }));

  await enabled.uncheck();
  await expect(completedName).toBeHidden();
  await expect(confirmationText).toBeHidden();
  await form.getByRole('button', { name: 'ПРИМЕНИТЬ' }).click();
  const ordinary = await commandFromLastSave(page, 'emergency-lights');
  expect(ordinary).toMatchObject({
    id: 'emergency-lights',
    name: 'Включить красный свет',
    text: 'Аварийное освещение включено.',
  });
  expect(ordinary).not.toHaveProperty('stateChange');
});

test('completed command displays its frozen snapshot while authored fields remain editable', async ({ page }) => {
  await selectCommand(page, 'Двери открыты');

  const form = page.locator('#nodeForm');
  await expect(form.getByLabel('ИЗМЕНЯЕТ СОСТОЯНИЕ')).toBeChecked();
  await expect(form.getByLabel('ИСХОДНОЕ НАЗВАНИЕ')).toHaveValue('Открыть двери');
  await expect(form.getByLabel('НАЗВАНИЕ ПОСЛЕ ВЫПОЛНЕНИЯ')).toHaveValue('Двери разблокированы');
  await expect(form.getByLabel('ТЕКСТ УСПЕШНОГО ВЫПОЛНЕНИЯ')).toHaveValue('Новая редакция результата открытия.');

  const frozen = form.getByRole('status', { name: 'СОХРАНЁННОЕ СОСТОЯНИЕ КОМАНДЫ' });
  await expect(frozen).toContainText('ВЫПОЛНЕНО');
  await expect(frozen).toContainText('Двери открыты');
  await expect(frozen).toContainText('Доступ в сектор разрешён.');

  await form.getByLabel('НАЗВАНИЕ ПОСЛЕ ВЫПОЛНЕНИЯ').fill('Новый заголовок для следующего выполнения');
  await form.getByLabel('ТЕКСТ УСПЕШНОГО ВЫПОЛНЕНИЯ').fill('Новый результат для следующего выполнения.');
  await form.getByRole('button', { name: 'ПРИМЕНИТЬ' }).click();

  await expect(page.locator('.tree-row', { hasText: 'Двери открыты' })).toHaveCount(1);
  await expect(frozen).toContainText('Двери открыты');
  await expect(frozen).toContainText('Доступ в сектор разрешён.');
  const save = await lastDesktopCall(page, 'SaveSession');
  expect(save.args[0].terminals[0].commandStates.doors).toEqual({
    completedName: 'Двери открыты',
    resultText: 'Доступ в сектор разрешён.',
  });
});

test('individual and terminal resets require confirmation and update only the intended snapshots', async ({ page }) => {
  await selectCommand(page, 'Двери открыты');
  const form = page.locator('#nodeForm');
  const resetOne = form.getByRole('button', { name: 'СБРОСИТЬ СОСТОЯНИЕ' });
  const initialDurable = await authoringDurableState(page);
  expect(Object.keys(initialDurable.commandStates).sort()).toEqual(['alarm', 'doors']);

  await resetOne.click();
  const resetConfirmation = page.getByRole('dialog', { name: 'ПОДТВЕРЖДЕНИЕ СБРОСА' });
  await expect(resetConfirmation).toContainText(/сбросить.*двер/i);
  await resetConfirmation.getByRole('button', { name: 'ОТМЕНИТЬ' }).click();
  await expect.poll(() => desktopCallCount(page, 'ResetCommandState')).toBe(0);
  await expect.poll(() => authoringDurableState(page)).toEqual(initialDurable);
  await expect(page.locator('.tree-row', { hasText: 'Двери открыты' })).toHaveCount(1);
  await expect(page.locator('.tree-row', { hasText: 'Тревога включена' })).toHaveCount(1);

  await resetOne.click();
  await expect(resetConfirmation).toContainText(/сбросить.*двер/i);
  await resetConfirmation.getByRole('button', { name: 'ПОДТВЕРДИТЬ' }).click();
  await expect.poll(() => desktopCallCount(page, 'ResetCommandState')).toBe(1);
  expect(await lastDesktopCall(page, 'ResetCommandState')).toMatchObject({
    args: [{ terminalId: TERMINAL_ID, commandId: 'doors' }],
  });
  await expect.poll(() => authoringDurableState(page)).toEqual({
    revision: initialDurable.revision + 1,
    commandStates: { alarm: initialDurable.commandStates.alarm },
  });
  await expect(page.locator('.tree-row', { hasText: 'Открыть двери' })).toHaveCount(1);
  await expect(page.locator('.tree-row', { hasText: 'Тревога включена' })).toHaveCount(1);

  const resetAll = page.getByRole('button', { name: 'СБРОСИТЬ ВСЕ СОСТОЯНИЯ' });
  await resetAll.click();
  await expect(resetConfirmation).toContainText(/сбросить.*все.*терминал/i);
  await resetConfirmation.getByRole('button', { name: 'ОТМЕНИТЬ' }).click();
  await expect.poll(() => desktopCallCount(page, 'ResetTerminalCommandStates')).toBe(0);
  await expect.poll(() => authoringDurableState(page)).toEqual({
    revision: initialDurable.revision + 1,
    commandStates: { alarm: initialDurable.commandStates.alarm },
  });
  await expect(page.locator('.tree-row', { hasText: 'Тревога включена' })).toHaveCount(1);

  await resetAll.click();
  await expect(resetConfirmation).toContainText(/сбросить.*все.*терминал/i);
  await resetConfirmation.getByRole('button', { name: 'ПОДТВЕРДИТЬ' }).click();
  await expect.poll(() => desktopCallCount(page, 'ResetTerminalCommandStates')).toBe(1);
  expect(await lastDesktopCall(page, 'ResetTerminalCommandStates')).toMatchObject({
    args: [{ terminalId: TERMINAL_ID }],
  });
  await expect.poll(() => authoringDurableState(page)).toEqual({
    revision: initialDurable.revision + 2,
    commandStates: {},
  });
  await expect(page.locator('.tree-row', { hasText: 'Включить тревогу' })).toHaveCount(1);
  await expect(page.locator('.tree-row', { hasText: 'Тревога включена' })).toHaveCount(0);

  await page.reload();
  await openAuthoringFixture(page);
  await expect(page.locator('.tree-row', { hasText: 'Открыть двери' })).toHaveCount(1);
  await expect(page.locator('.tree-row', { hasText: 'Включить тревогу' })).toHaveCount(1);
  await expect.poll(() => authoringDurableState(page)).toEqual({
    revision: initialDurable.revision + 2,
    commandStates: {},
  });
});

test('terminal reset rejects a stale backend success even when its session looks reset', async ({ page }) => {
  const durableBefore = await authoringDurableState(page);
  const snapshotResponse = await page.request.get(`${FIXTURE_URL}/session`);
  expect(snapshotResponse.ok()).toBe(true);
  const stale = await snapshotResponse.json();
  stale.revision = 0;
  stale.session.terminals.find(terminal => terminal.id === TERMINAL_ID).commandStates = {};

  await page.route(`**${FIXTURE_URL}/reset-terminal`, route => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify(stale),
  }));
  await page.getByRole('button', { name: 'СБРОСИТЬ ВСЕ СОСТОЯНИЯ' }).click();
  await page.getByRole('dialog', { name: 'ПОДТВЕРЖДЕНИЕ СБРОСА' })
    .getByRole('button', { name: 'ПОДТВЕРДИТЬ' }).click();

  await expect(page.locator('#saveStatus')).toHaveClass(/err/);
  await expect(page.locator('#saveStatus')).toContainText(/не подтвердил канонический сброс/i);
  await expect(page.locator('.tree-row', { hasText: 'Двери открыты' })).toHaveCount(1);
  await expect(page.locator('.tree-row', { hasText: 'Тревога включена' })).toHaveCount(1);
  await expect.poll(() => authoringDurableState(page)).toEqual(durableBefore);
});

test('terminal reset rejects a newer backend result that still contains completed snapshots', async ({ page }) => {
  const durableBefore = await authoringDurableState(page);
  const snapshotResponse = await page.request.get(`${FIXTURE_URL}/session`);
  expect(snapshotResponse.ok()).toBe(true);
  const nonCanonical = await snapshotResponse.json();
  nonCanonical.revision = durableBefore.revision + 1;

  await page.route(`**${FIXTURE_URL}/reset-terminal`, route => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify(nonCanonical),
  }));
  await page.getByRole('button', { name: 'СБРОСИТЬ ВСЕ СОСТОЯНИЯ' }).click();
  await page.getByRole('dialog', { name: 'ПОДТВЕРЖДЕНИЕ СБРОСА' })
    .getByRole('button', { name: 'ПОДТВЕРДИТЬ' }).click();

  await expect(page.locator('#saveStatus')).toHaveClass(/err/);
  await expect(page.locator('#saveStatus')).toContainText(/не подтвердил канонический сброс/i);
  await expect(page.locator('.tree-row', { hasText: 'Двери открыты' })).toHaveCount(1);
  await expect(page.locator('.tree-row', { hasText: 'Тревога включена' })).toHaveCount(1);
  await expect.poll(() => authoringDurableState(page)).toEqual(durableBefore);
});
