import { expect, test } from '@playwright/test';

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
      addresses: ['0xC000', '0xC00C'],
      text: '[!!]!]......(DUST)......',
      words: [{ id: 'A1', start: 13, length: 4 }],
    },
  ],
  patterns: [{ id: 'pattern-initial', row: 0, start: 0, end: 3, used: false }],
};

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
  await page.evaluate(hack => {
    window.__playerSocket.emit('message', {
      data: JSON.stringify({
        type: 'TERMINAL_LIVE',
        terminalName: 'Controlled terminal',
        tree: { id: 'root', type: 'folder', name: 'Root', children: [] },
        hackLevel: 1,
        hack,
        nav: { path: ['root'], mode: 'list', viewEntryId: null, commandNodeId: null },
      }),
    });
  }, initialHack);
  await expect(page.locator('#hackBoard')).toBeVisible();
}

test('valid patterns interact while standalone delimiters remain inert and visually identical', async ({ page }) => {
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
  await expect.poll(() => page.evaluate(() => window.__outboundMessages)).toEqual([
    { type: 'HACK_PATTERN', patternId: 'pattern-initial' },
  ]);
  await expect(page.locator('#hackColumns')).toHaveText(boardBefore);
  await expect(page.locator('#attemptsLine')).toHaveText(attemptsBefore);

  await page.evaluate(() => { window.__outboundMessages.length = 0; });
  await standaloneDecoy.hover();
  await expect(page.locator('.hcell.hi')).toHaveCount(0);
  await expect(page.locator('#hackInputPreview')).toHaveText('');
  await standaloneDecoy.focus();
  await expect(page.locator('.hcell.hi')).toHaveCount(0);
  await standaloneDecoy.click();
  expect(await page.evaluate(() => window.__outboundMessages)).toEqual([]);
  await expect(page.locator('#hackColumns')).toHaveText(boardBefore);
  await expect(page.locator('#attemptsLine')).toHaveText(attemptsBefore);
});

test('word-interrupted spans stay ordinary until a server snapshot publishes a new pattern', async ({ page }) => {
  await openControlledPuzzle(page);

  const interruptedOpening = page.locator('[data-row="1"][data-offset="0"]');
  await interruptedOpening.click();
  expect(await page.evaluate(() => window.__outboundMessages)).toEqual([]);

  await page.locator('.hcell.word[data-target="A1"]').click();
  await expect.poll(() => page.evaluate(() => window.__outboundMessages)).toEqual([
    { type: 'HACK_GUESS', targetId: 'A1' },
  ]);

  const postDudHack = structuredClone(initialHack);
  postDudHack.columns[0].text = '[!!]!]......(....)......';
  postDudHack.columns[0].words = [];
  postDudHack.patterns.push({ id: 'pattern-after-dud', row: 1, start: 0, end: 5, used: false });
  await page.evaluate(hack => {
    window.__outboundMessages.length = 0;
    window.__playerSocket.emit('message', {
      data: JSON.stringify({ type: 'HACK_STATE', hack }),
    });
  }, postDudHack);

  const publishedOpening = page.locator('[data-row="1"][data-offset="0"]');
  await publishedOpening.hover();
  await expect(page.locator('.hcell.hi')).toHaveCount(6);
  await publishedOpening.click();
  await expect.poll(() => page.evaluate(() => window.__outboundMessages)).toEqual([
    { type: 'HACK_PATTERN', patternId: 'pattern-after-dud' },
  ]);
});
