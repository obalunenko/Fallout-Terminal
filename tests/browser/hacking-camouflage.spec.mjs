import { expect, test } from '@playwright/test';

const BROADCAST_ID = 'broadcast-1';
const TERMINAL_ID = 'terminal-1';

const initialHack = {
  level: 1,
  wordLength: 4,
  attemptsMax: 4,
  attemptsLeft: 4,
  solved: false,
  failed: false,
  log: [],
  columns: [
    {
      addresses: ['0xC000', '0xC00C', '0xC018', '0xC024', '0xC030'],
      text: '[!!]!]......(DUST)......[)!!!!!!!!!!<!!>>!!!!!!!)!!!!!!!!!!!',
      words: [{ id: 'A1', start: 13, length: 4 }],
    },
  ],
  patterns: [
    { id: 'pattern-initial', row: 0, start: 0, end: 3, used: false },
    { id: 'pattern-first-closer', row: 3, start: 0, end: 3, used: false },
  ],
};

function controllingState() {
  return {
    revision: 1,
    sessionId: 'session-1',
    fallbackName: 'PLAYER 1',
    character: { id: 'character-1', name: 'Mara' },
    role: 'active',
    phase: 'controlling',
    broadcastId: BROADCAST_ID,
    activeTerminalId: TERMINAL_ID,
    roster: [{ id: 'character-1', name: 'Mara', status: 'claimed' }],
  };
}

async function emit(page, message) {
  await page.evaluate(value => {
    window.__playerSocket.emit('message', { data: JSON.stringify(value) });
  }, message);
}

async function expectSharedRequest(page, expected) {
  const requests = await page.evaluate(() => window.__outboundMessages.filter(message =>
    ['HACK_GUESS', 'HACK_PATTERN'].includes(message.type)
  ));
  expect(requests).toHaveLength(1);
  expect(requests[0]).toEqual({
    ...expected,
    requestId: expect.any(String),
    broadcastId: BROADCAST_ID,
    terminalId: TERMINAL_ID,
  });
  expect(requests[0].requestId.length).toBeGreaterThan(0);
  return requests[0];
}

async function acceptHackRequest(page, request, revision, hack) {
  await emit(page, {
    type: 'ACTION_RESULT',
    requestId: request.requestId,
    accepted: true,
    reason: 'accepted',
    revision,
  });
  await expect(page.locator('#screen')).toHaveClass(/shared-input-pending/);

  await emit(page, {
    type: 'HACK_STATE',
    revision,
    terminalId: TERMINAL_ID,
    hack,
  });
  await expect(page.locator('#screen')).not.toHaveClass(/shared-input-pending/);
  await page.evaluate(() => { window.__outboundMessages.length = 0; });
}

async function openControlledPuzzle(page) {
  await page.addInitScript(() => {
    window.__outboundMessages = [];
    class FakeWebSocket {
      static OPEN = 1;
      static CLOSED = 3;

      constructor(url) {
        this.url = url;
        this.readyState = FakeWebSocket.OPEN;
        this.listeners = new Map();
        window.__playerSocket = this;
        queueMicrotask(() => this.emit('open', {}));
      }

      addEventListener(type, listener) {
        const listeners = this.listeners.get(type) || [];
        listeners.push(listener);
        this.listeners.set(type, listeners);
      }

      emit(type, event) {
        for (const listener of this.listeners.get(type) || []) listener(event);
      }

      send(raw) {
        window.__outboundMessages.push(JSON.parse(raw));
      }

      close() {
        this.readyState = FakeWebSocket.CLOSED;
        this.emit('close', {});
      }
    }
    window.WebSocket = FakeWebSocket;
    HTMLMediaElement.prototype.play = () => Promise.resolve();
  });

  await page.goto('/');
  await page.waitForFunction(() => window.__playerSocket);
  await expect.poll(() => page.evaluate(() => window.__outboundMessages)).toEqual([
    { type: 'SESSION_HELLO' },
  ]);

  await emit(page, {
    type: 'SESSION_WELCOME',
    browserToken: 'token-1',
    state: controllingState(),
  });
  await emit(page, {
    type: 'TERMINAL_LIVE',
    revision: 2,
    terminalId: TERMINAL_ID,
    terminalName: 'Controlled terminal',
    tree: { id: 'root', type: 'folder', name: 'Root', children: [] },
    hackLevel: 1,
    hack: initialHack,
    nav: { path: ['root'], mode: 'list', viewEntryId: null, commandNodeId: null },
  });
  await expect(page.locator('#hackBoard')).toBeVisible();
  await page.evaluate(() => { window.__outboundMessages.length = 0; });
}

test('only a valid opening activates its whole pattern while all other symbols stay individually selectable', async ({ page }) => {
  await openControlledPuzzle(page);

  const validOpening = page.locator('[data-row="0"][data-offset="0"]');
  const standaloneDecoy = page.locator('[data-row="0"][data-offset="5"]');
  const styleProperties = ['color', 'filter', 'font-family', 'font-size', 'opacity', 'text-shadow'];
  const styles = await Promise.all([validOpening, standaloneDecoy].map(locator =>
    locator.evaluate((element, properties) => {
      const computed = getComputedStyle(element);
      return Object.fromEntries(properties.map(property => [property, computed.getPropertyValue(property)]));
    }, styleProperties),
  ));
  expect(styles[0]).toEqual(styles[1]);

  await validOpening.hover();
  await expect(page.locator('.hcell.hi')).toHaveCount(4);
  await expect(page.locator('#hackInputPreview')).toHaveText('[!!]');
  await validOpening.focus();
  await expect(page.locator('.hcell.hi')).toHaveCount(4);

  const boardBefore = await page.locator('#hackColumns').textContent();
  const attemptsBefore = await page.locator('#attemptsLine').textContent();
  await validOpening.click();
  const patternRequest = await expectSharedRequest(page, {
    type: 'HACK_PATTERN',
    patternId: 'pattern-initial',
  });
  await expect(page.locator('#screen')).toHaveClass(/shared-input-pending/);
  await expect(page.locator('#hackColumns')).toHaveText(boardBefore);
  await expect(page.locator('#attemptsLine')).toHaveText(attemptsBefore);

  const usedHack = structuredClone(initialHack);
  usedHack.patterns[0].used = true;
  await acceptHackRequest(page, patternRequest, 3, usedHack);

  let revision = 4;
  for (const target of [
    { offset: 1, glyph: '!', targetId: '0:1' },
    { offset: 2, glyph: '!', targetId: '0:2' },
    { offset: 3, glyph: ']', targetId: '0:3' },
    { offset: 5, glyph: ']', targetId: '0:5' },
  ]) {
    await page.evaluate(() => { window.__outboundMessages.length = 0; });
    const cell = page.locator(`[data-row="0"][data-offset="${target.offset}"]`);
    await cell.hover();
    await expect(page.locator('.hcell.hi')).toHaveCount(1);
    await expect(page.locator('#hackInputPreview')).toHaveText(target.glyph);
    await cell.focus();
    await expect(page.locator('.hcell.hi')).toHaveCount(1);
    await cell.click();
    const guessRequest = await expectSharedRequest(page, {
      type: 'HACK_GUESS',
      targetId: target.targetId,
    });
    await expect(page.locator('#screen')).toHaveClass(/shared-input-pending/);
    await expect(page.locator('#hackColumns')).toHaveText(boardBefore);
    await expect(page.locator('#attemptsLine')).toHaveText(attemptsBefore);
    await acceptHackRequest(page, guessRequest, revision, usedHack);
    revision += 1;
  }

  await validOpening.hover();
  await expect(page.locator('.hcell.hi')).toHaveCount(0);
  await validOpening.click();
  expect(await page.evaluate(() => window.__outboundMessages)).toEqual([]);
});

test('word-interrupted spans stay ordinary until a server snapshot publishes a new pattern', async ({ page }) => {
  await openControlledPuzzle(page);

  const boardBefore = await page.locator('#hackColumns').textContent();
  const attemptsBefore = await page.locator('#attemptsLine').textContent();
  const interruptedOpening = page.locator('[data-row="1"][data-offset="0"]');
  await interruptedOpening.click();
  const interruptedRequest = await expectSharedRequest(page, {
    type: 'HACK_GUESS',
    targetId: '0:12',
  });
  await expect(page.locator('#hackColumns')).toHaveText(boardBefore);
  await expect(page.locator('#attemptsLine')).toHaveText(attemptsBefore);
  await acceptHackRequest(page, interruptedRequest, 3, initialHack);

  await page.locator('.hcell.word[data-target="A1"]').click();
  const wordRequest = await expectSharedRequest(page, {
    type: 'HACK_GUESS',
    targetId: 'A1',
  });
  await expect(page.locator('#hackColumns')).toHaveText(boardBefore);
  await expect(page.locator('#attemptsLine')).toHaveText(attemptsBefore);
  await acceptHackRequest(page, wordRequest, 4, initialHack);

  const postDudHack = structuredClone(initialHack);
  postDudHack.columns[0].text = postDudHack.columns[0].text.replace('(DUST)', '(....)');
  postDudHack.columns[0].words = [];
  postDudHack.patterns.push({ id: 'pattern-after-dud', row: 1, start: 0, end: 5, used: false });
  await emit(page, {
    type: 'HACK_STATE',
    revision: 5,
    terminalId: TERMINAL_ID,
    hack: postDudHack,
  });

  const publishedOpening = page.locator('[data-row="1"][data-offset="0"]');
  await publishedOpening.hover();
  await expect(page.locator('.hcell.hi')).toHaveCount(6);
  await publishedOpening.click();
  const publishedPatternRequest = await expectSharedRequest(page, {
    type: 'HACK_PATTERN',
    patternId: 'pattern-after-dud',
  });
  const postPatternHack = structuredClone(postDudHack);
  postPatternHack.patterns.at(-1).used = true;
  await acceptHackRequest(page, publishedPatternRequest, 6, postPatternHack);

  const publishedClosing = page.locator('[data-row="1"][data-offset="5"]');
  await publishedClosing.hover();
  await expect(page.locator('.hcell.hi')).toHaveCount(1);
  await expect(page.locator('#hackInputPreview')).toHaveText(')');
  await publishedClosing.click();
  const closingRequest = await expectSharedRequest(page, {
    type: 'HACK_GUESS',
    targetId: '0:17',
  });
  await acceptHackRequest(page, closingRequest, 7, postPatternHack);
});

test('invalid delimiter categories remain ordinary individual targets', async ({ page }) => {
  await openControlledPuzzle(page);

  const boardBefore = await page.locator('#hackColumns').textContent();
  const attemptsBefore = await page.locator('#attemptsLine').textContent();
  let revision = 3;
  for (const target of [
    { name: 'unmatched opening', row: 2, offset: 0, glyph: '[', targetId: '0:24' },
    { name: 'mismatched closer', row: 2, offset: 1, glyph: ')', targetId: '0:25' },
    { name: 'later compatible closer', row: 3, offset: 4, glyph: '>', targetId: '0:40' },
    { name: 'unmatched closer', row: 4, offset: 0, glyph: ')', targetId: '0:48' },
  ]) {
    await page.evaluate(() => { window.__outboundMessages.length = 0; });
    const cell = page.locator(`[data-row="${target.row}"][data-offset="${target.offset}"]`);

    await test.step(target.name, async () => {
      await cell.hover();
      await expect(page.locator('.hcell.hi')).toHaveCount(1);
      await expect(page.locator('#hackInputPreview')).toHaveText(target.glyph);
      await cell.focus();
      await expect(page.locator('.hcell.hi')).toHaveCount(1);
      await cell.click();
      const request = await expectSharedRequest(page, {
        type: 'HACK_GUESS',
        targetId: target.targetId,
      });
      await expect(page.locator('#screen')).toHaveClass(/shared-input-pending/);
      await expect(page.locator('#hackColumns')).toHaveText(boardBefore);
      await expect(page.locator('#attemptsLine')).toHaveText(attemptsBefore);
      await acceptHackRequest(page, request, revision, initialHack);
      revision += 1;
    });
  }
});
