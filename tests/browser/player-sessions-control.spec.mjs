import { expect, test } from '@playwright/test';
import { readFile } from 'node:fs/promises';

const TOKEN_KEY = 'fallout-terminal.player-token';

async function openPlayer(page, storedToken = null) {
  await page.addInitScript(({ tokenKey, token }) => {
    window.__outboundMessages = [];
    window.__xssExecuted = false;
    if (token) localStorage.setItem(tokenKey, token);
    else localStorage.removeItem(tokenKey);

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
  }, { tokenKey: TOKEN_KEY, token: storedToken });

  await page.goto('/');
  await page.waitForFunction(() => window.__playerSocket);
}

async function emit(page, message) {
  await page.evaluate(value => {
    window.__playerSocket.emit('message', { data: JSON.stringify(value) });
  }, message);
}

function selectingState(revision = 1) {
  return {
    revision,
    sessionId: 'session-1',
    fallbackName: 'PLAYER 1',
    character: null,
    role: 'unassigned',
    phase: 'selecting',
    broadcastId: 'broadcast-1',
    activeTerminalId: null,
    roster: [
      { id: 'character-1', name: 'Mara', status: 'available' },
      { id: 'character-2', name: 'Boone', status: 'claimed' },
      { id: 'character-3', name: '<img src=x onerror="window.__xssExecuted=true">', status: 'available' },
    ],
  };
}

function assignedState(role = 'active', revision = 2) {
  return {
    revision,
    sessionId: 'session-1',
    fallbackName: 'PLAYER 1',
    character: { id: 'character-1', name: 'Mara' },
    role,
    phase: role === 'active' ? 'controlling' : 'observing',
    broadcastId: 'broadcast-1',
    activeTerminalId: 'terminal-1',
    roster: [
      { id: 'character-1', name: 'Mara', status: 'claimed' },
      { id: 'character-2', name: 'Boone', status: 'claimed' },
    ],
  };
}

function sharedTree() {
  return {
    id: 'root',
    type: 'folder',
    name: 'ROOT',
    children: [
      {
        id: 'docs',
        type: 'folder',
        name: 'DOCS',
        children: [
          {
            id: 'report',
            type: 'entry',
            name: 'REPORT',
            description: Array.from({ length: 300 }, (_, index) => `REPORT LINE ${index + 1}`).join('\n'),
          },
          { id: 'unlock', type: 'command', name: 'UNLOCK', text: 'ACCESS GRANTED' },
        ],
      },
      { id: 'status', type: 'entry', name: 'STATUS', description: 'SYSTEM NOMINAL' },
    ],
  };
}

function terminalLive(overrides = {}) {
  return {
    type: 'TERMINAL_LIVE',
    revision: 2,
    terminalId: 'terminal-1',
    terminalName: 'Overseer',
    tree: sharedTree(),
    hackLevel: 0,
    introText: 'WELCOME',
    hack: null,
    nav: { path: ['root'], mode: 'list', viewEntryId: null, commandNodeId: null },
    ...overrides,
  };
}

async function openAssignedPlayer(page, role = 'active') {
  await openPlayer(page);
  await emit(page, {
    type: 'SESSION_WELCOME',
    browserToken: 'token-1',
    state: assignedState(role),
  });
  await emit(page, terminalLive());
  await page.evaluate(() => { window.__outboundMessages.length = 0; });
}

async function sharedOutbound(page) {
  return page.evaluate(() => window.__outboundMessages.filter(message =>
    ['NAV_ACTION', 'HACK_GUESS', 'HACK_PATTERN'].includes(message.type)
  ));
}

async function installRealSocketRecorder(context) {
  await context.addInitScript(() => {
    window.__outboundMessages = [];
    HTMLMediaElement.prototype.play = () => Promise.resolve();
  });
}

function recordNativeWebSocketFrames(page) {
  const messages = [];
  page.on('websocket', socket => {
    socket.on('framesent', event => {
      try {
        messages.push(JSON.parse(event.payload));
      } catch {
        // The player protocol is JSON; retain native behavior for other frames.
      }
    });
  });
  return messages;
}

async function resetSharedRecording(page) {
  await page.evaluate(() => {
    window.__outboundMessages.length = 0;
    window.__pendingTransitions = [document.querySelector('#screen').classList.contains('shared-input-pending')];
    window.__pendingObserver?.disconnect();
    window.__pendingObserver = new MutationObserver(() => {
      window.__pendingTransitions.push(document.querySelector('#screen').classList.contains('shared-input-pending'));
    });
    window.__pendingObserver.observe(document.querySelector('#screen'), { attributes: true, attributeFilter: ['class'] });
  });
}

async function findHackingFiller(page, patternOpening) {
  const fillers = page.locator('.hcell.filler');
  const total = await fillers.count();
  for (let index = 0; index < total; index += 1) {
    const candidate = fillers.nth(index);
    await candidate.hover();
    const highlighted = await page.locator('.hcell.hi').count();
    if ((patternOpening && highlighted > 1) || (!patternOpening && highlighted === 1)) return candidate;
  }
  throw new Error(patternOpening ? 'fixture has no unused pattern opening' : 'fixture has no ordinary filler target');
}

async function waitForAuthoritativeCompletion(page) {
  await expect.poll(() => page.evaluate(() => window.__pendingTransitions)).toContain(true);
  await expect(page.locator('#screen')).not.toHaveClass(/shared-input-pending/);
  await expect.poll(() => page.evaluate(() => window.__pendingTransitions.at(-1))).toBe(false);
}

async function hackingSurface(page) {
  return page.evaluate(() => ({
    attempts: document.querySelector('#attemptsLine').textContent,
    log: document.querySelector('#hackLog').textContent,
    board: document.querySelector('#hackColumns').textContent,
  }));
}

async function disconnectAndAwaitReconnect(page) {
  await page.evaluate(() => {
    window.__disconnectedPlayerSocket = window.__playerSocket;
    window.__playerSocket.close();
  });
  await expect(page.locator('#connOverlay')).toBeVisible();
  await page.waitForFunction(() =>
    window.__playerSocket && window.__playerSocket !== window.__disconnectedPlayerSocket,
  null, { timeout: 5000 });
}

async function installContinuityFixture(context, { disableLocks = false, seedToken = null } = {}) {
  await context.addInitScript(({ tokenKey, locksDisabled, initialToken }) => {
    window.__outboundMessages = [];
    if (initialToken !== null && localStorage.getItem(tokenKey) === null) {
      localStorage.setItem(tokenKey, initialToken);
    }
    if (locksDisabled) {
      Object.defineProperty(navigator, 'locks', { configurable: true, value: undefined });
    }

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
  }, { tokenKey: TOKEN_KEY, locksDisabled: disableLocks, initialToken: seedToken });
}

async function gotoContinuityPage(page) {
  await page.goto('/');
  await page.waitForFunction(() => window.__playerSocket);
}

async function outboundMessages(page) {
  return page.evaluate(() => window.__outboundMessages || []);
}

async function welcomeSession(page, browserToken, sessionId, overrides = {}) {
  await emit(page, {
    type: 'SESSION_WELCOME',
    browserToken,
    state: {
      ...assignedState('active', 2),
      sessionId,
      ...overrides,
    },
  });
}

test('handshake keeps connection gating active, reuses only the opaque token, and stores the replacement welcome token', async ({ page }) => {
  await openPlayer(page, 'recognized-token');

  await expect.poll(() => page.evaluate(() => window.__outboundMessages)).toEqual([
    { type: 'SESSION_HELLO', browserToken: 'recognized-token' },
  ]);
  await expect(page.locator('#connOverlay')).toBeVisible();

  await emit(page, {
    type: 'SESSION_WELCOME',
    browserToken: 'replacement-token',
    state: selectingState(),
  });

  await expect(page.locator('#connOverlay')).toBeHidden();
  await expect(page.locator('#characterSelect')).toBeVisible();
  expect(await page.evaluate(key => localStorage.getItem(key), TOKEN_KEY)).toBe('replacement-token');
  const storedValues = await page.evaluate(() => Object.values(localStorage));
  expect(storedValues).toEqual(['replacement-token']);
});

test('selection renders escaped availability, permits one pending request, and recovers from an authoritative conflict', async ({ page }) => {
  await openPlayer(page);
  await emit(page, { type: 'SESSION_WELCOME', browserToken: 'token-1', state: selectingState() });

  await expect(page.locator('#characterSelect')).toBeVisible();
  await expect(page.locator('[data-character-id="character-1"]')).toContainText('Mara');
  await expect(page.locator('[data-character-id="character-2"]')).toBeDisabled();
  await expect(page.locator('[data-character-id="character-3"]')).toContainText('<img');
  await expect(page.locator('#characterSelect img')).toHaveCount(0);
  expect(await page.evaluate(() => window.__xssExecuted)).toBe(false);

  await page.locator('[data-character-id="character-1"]').click();
  await expect(page.locator('#characterSelect')).toHaveClass(/pending/);
  await page.locator('[data-character-id="character-3"]').click({ force: true });

  const requests = await page.evaluate(() => window.__outboundMessages.filter(message => message.type === 'CHARACTER_SELECT'));
  expect(requests).toHaveLength(1);
  expect(requests[0]).toMatchObject({
    type: 'CHARACTER_SELECT',
    broadcastId: 'broadcast-1',
    characterId: 'character-1',
  });
  expect(requests[0].requestId).toEqual(expect.any(String));
  expect(requests[0].requestId.length).toBeGreaterThan(0);

  await emit(page, {
    type: 'ACTION_RESULT',
    requestId: requests[0].requestId,
    accepted: false,
    reason: 'conflict',
    revision: 1,
  });
  await expect(page.locator('#characterSelect')).not.toHaveClass(/pending/);
  await expect(page.locator('#playerNotice')).toContainText(/занят|conflict/i);
  await expect(page.locator('[data-character-id="character-3"]')).toBeEnabled();
});

test('accepted assignment advances only after authoritative state and shows character-primary assigned waiting identity', async ({ page }) => {
  await openPlayer(page);
  await emit(page, { type: 'SESSION_WELCOME', browserToken: 'token-1', state: selectingState() });
  await page.locator('[data-character-id="character-1"]').click();
  const request = await page.evaluate(() => window.__outboundMessages.find(message => message.type === 'CHARACTER_SELECT'));

  await emit(page, {
    type: 'ACTION_RESULT',
    requestId: request.requestId,
    accepted: true,
    reason: 'accepted',
    revision: 2,
  });
  await expect(page.locator('#characterSelect')).toBeVisible();
  await expect(page.locator('#characterSelect')).toHaveClass(/pending/);

  await emit(page, {
    type: 'PLAYER_STATE',
    state: {
      ...selectingState(2),
      character: { id: 'character-1', name: 'Mara' },
      role: 'active',
      phase: 'waiting',
    },
  });

  await expect(page.locator('#characterSelect')).toBeHidden();
  await expect(page.locator('#assignedWaiting')).toBeVisible();
  await expect(page.locator('#playerIdentity')).toContainText('Mara');
  await expect(page.locator('#playerIdentity')).toContainText('PLAYER 1');
  await expect(page.locator('#roleBadge')).toContainText(/актив|active/i);
});

test('observer is visibly read-only while hover, keyboard selection, paging, and back feedback stay local', async ({ page }) => {
  await openAssignedPlayer(page, 'observer');

  await expect(page.locator('#roleBadge')).toContainText(/наблюд|observer/i);
  await expect(page.locator('#roleBadge')).toHaveAttribute('data-role', 'observer');
  await expect(page.locator('#screen')).toHaveClass(/observer-read-only/);

  const firstRow = page.locator('.term-row').filter({ hasText: 'DOCS' });
  const secondRow = page.locator('.term-row').filter({ hasText: 'STATUS' });
  await expect(secondRow).toBeVisible();
  await firstRow.hover();
  await expect(firstRow).toHaveClass(/sel/);
  await expect(firstRow).toHaveCSS('cursor', 'not-allowed');

  await page.keyboard.press('ArrowDown');
  await expect(secondRow).toHaveClass(/sel/);
  await page.keyboard.press('Enter');
  await firstRow.click();

  await emit(page, {
    type: 'NAV_STATE',
    revision: 3,
    terminalId: 'terminal-1',
    nav: { path: ['root', 'docs'], mode: 'list', viewEntryId: null, commandNodeId: null },
  });
  await expect(page.locator('#backBtn')).toBeVisible();
  await page.locator('#backBtn').click();
  await page.keyboard.press('Backspace');

  await emit(page, {
    type: 'NAV_STATE',
    revision: 4,
    terminalId: 'terminal-1',
    nav: { path: ['root', 'docs'], mode: 'entry', viewEntryId: 'report', commandNodeId: null },
  });
  await expect(page.locator('#pageNext')).toBeVisible();
  const firstPage = await page.locator('#pageIndicator').textContent();
  await page.locator('#pageNext').click();
  await expect(page.locator('#pageIndicator')).not.toHaveText(firstPage);

  expect(await sharedOutbound(page)).toEqual([]);
});

test('observer hacking hover, focus, and typed preview stay local while every hack target is read-only', async ({ page }) => {
  await openAssignedPlayer(page, 'observer');
  await emit(page, terminalLive({
    revision: 5,
    hackLevel: 1,
    hack: {
      level: 1,
      wordLength: 5,
      attemptsMax: 4,
      attemptsLeft: 4,
      solved: false,
      failed: false,
      log: [],
      columns: [{
        addresses: ['0xC000'],
        text: '[!!]VAULT...',
        words: [{ id: 'A1', start: 4, length: 5 }],
      }],
      patterns: [{ id: 'pattern-1', row: 0, start: 0, end: 3, used: false }],
    },
  }));
  await page.evaluate(() => { window.__outboundMessages.length = 0; });

  const opening = page.locator('[data-row="0"][data-offset="0"]');
  await opening.hover();
  await expect(page.locator('.hcell.hi')).toHaveCount(4);
  await expect(page.locator('#hackInputPreview')).toHaveText('[!!]');

  const word = page.locator('[data-target="A1"]');
  await word.focus();
  await expect(page.locator('#hackInputPreview')).toHaveText('VAULT');
  await page.keyboard.press('x');
  await expect(page.locator('#hackInputPreview')).toHaveText('x');

  await opening.click();
  await word.click();
  await page.locator('[data-target="0:9"]').click();

  expect(await sharedOutbound(page)).toEqual([]);
});

test('active controller can select password, filler, and pattern targets with authoritative pending completion', async ({ page }) => {
  await openAssignedPlayer(page, 'active');
  const initialHack = {
    level: 1,
    wordLength: 5,
    attemptsMax: 4,
    attemptsLeft: 4,
    solved: false,
    failed: false,
    log: [],
    columns: [{
      addresses: ['0xC000'],
      text: '[!!]VAULT...',
      words: [{ id: 'A1', start: 4, length: 5 }],
    }],
    patterns: [{ id: 'pattern-1', row: 0, start: 0, end: 3, used: false }],
  };
  await emit(page, terminalLive({ revision: 5, hackLevel: 1, hack: initialHack }));
  await page.evaluate(() => { window.__outboundMessages.length = 0; });

  const assertLatestRequest = async expected => {
    const requests = await sharedOutbound(page);
    const request = requests.at(-1);
    expect(request).toEqual({
      ...expected,
      requestId: expect.any(String),
      broadcastId: 'broadcast-1',
      terminalId: 'terminal-1',
    });
    await expect(page.locator('#screen')).toHaveClass(/shared-input-pending/);
    return request;
  };

  await page.locator('[data-target="A1"]').click();
  const password = await assertLatestRequest({ type: 'HACK_GUESS', targetId: 'A1' });
  await emit(page, {
    type: 'ACTION_RESULT', requestId: password.requestId, accepted: false,
    reason: 'invalid-action', revision: 5,
  });
  await expect(page.locator('#screen')).not.toHaveClass(/shared-input-pending/);

  await page.locator('[data-target="0:9"]').click();
  const filler = await assertLatestRequest({ type: 'HACK_GUESS', targetId: '0:9' });
  await emit(page, {
    type: 'ACTION_RESULT', requestId: filler.requestId, accepted: true,
    reason: 'accepted', revision: 6,
  });
  await expect(page.locator('#screen')).toHaveClass(/shared-input-pending/);
  await emit(page, { type: 'HACK_STATE', revision: 6, terminalId: 'terminal-1', hack: initialHack });
  await expect(page.locator('#screen')).not.toHaveClass(/shared-input-pending/);

  await page.locator('[data-row="0"][data-offset="0"]').click();
  const pattern = await assertLatestRequest({ type: 'HACK_PATTERN', patternId: 'pattern-1' });
  const usedHack = structuredClone(initialHack);
  usedHack.patterns[0].used = true;
  await emit(page, {
    type: 'ACTION_RESULT', requestId: pattern.requestId, accepted: true,
    reason: 'accepted', revision: 7,
  });
  await emit(page, { type: 'HACK_STATE', revision: 7, terminalId: 'terminal-1', hack: usedHack });
  await expect(page.locator('#screen')).not.toHaveClass(/shared-input-pending/);
  expect(await sharedOutbound(page)).toHaveLength(3);
});

test('BUG-003 composed server lets the active controller mutate every hacking target category while observers stay silent', async ({ browser }, testInfo) => {
  const contextOptions = { baseURL: testInfo.project.use.baseURL };
  const activeContext = await browser.newContext(contextOptions);
  const observerContext = await browser.newContext(contextOptions);
  try {
    await installRealSocketRecorder(activeContext);
    await installRealSocketRecorder(observerContext);
    const active = await activeContext.newPage();
    const observer = await observerContext.newPage();
    const activeFrames = recordNativeWebSocketFrames(active);
    const observerFrames = recordNativeWebSocketFrames(observer);

    await active.goto('/');
    await expect(active.locator('#characterSelect')).toBeVisible();
    await active.locator('.character-option:not([disabled])').first().click();
    await expect(active.locator('#roleBadge')).toHaveAttribute('data-role', 'active');
    await expect(active.locator('#hackBoard')).toBeVisible();

    await observer.goto('/');
    await expect(observer.locator('#characterSelect')).toBeVisible();
    await observer.locator('.character-option:not([disabled])').first().click();
    await expect(observer.locator('#roleBadge')).toHaveAttribute('data-role', 'observer');
    await expect(observer.locator('#hackBoard')).toBeVisible();
    expect(activeFrames.map(message => message.type)).toContain('SESSION_HELLO');
    expect(activeFrames.map(message => message.type)).toContain('CHARACTER_SELECT');

    const assertConverged = async () => {
      await expect.poll(async () => ({
        active: await hackingSurface(active),
        observer: await hackingSurface(observer),
      })).toEqual({ active: await hackingSurface(active), observer: await hackingSurface(active) });
    };

    await resetSharedRecording(active);
    activeFrames.length = 0;
    const filler = await findHackingFiller(active, false);
    const fillerTarget = await filler.getAttribute('data-target');
    const beforeFiller = await hackingSurface(active);
    await filler.click();
    await expect(active.locator('#screen')).toHaveClass(/shared-input-pending/);
    await expect.poll(() => activeFrames.filter(message => message.type.startsWith('HACK_'))).toHaveLength(1);
    expect(activeFrames.find(message => message.type.startsWith('HACK_'))).toMatchObject({
      type: 'HACK_GUESS', requestId: expect.any(String), broadcastId: expect.any(String),
      terminalId: 'terminal-1', targetId: fillerTarget,
    });
    await waitForAuthoritativeCompletion(active);
    await assertConverged();
    expect(await hackingSurface(active)).not.toEqual(beforeFiller);

    await resetSharedRecording(active);
    activeFrames.length = 0;
    const opening = await findHackingFiller(active, true);
    const beforePattern = await hackingSurface(active);
    await opening.click();
    await expect.poll(() => activeFrames.filter(message => message.type.startsWith('HACK_'))).toHaveLength(1);
    expect(activeFrames.find(message => message.type.startsWith('HACK_'))).toMatchObject({
      type: 'HACK_PATTERN', requestId: expect.any(String), broadcastId: expect.any(String),
      terminalId: 'terminal-1', patternId: expect.any(String),
    });
    await waitForAuthoritativeCompletion(active);
    await assertConverged();
    expect(await hackingSurface(active)).not.toEqual(beforePattern);

    await resetSharedRecording(active);
    activeFrames.length = 0;
    const password = active.locator('.hcell.word').first();
    const passwordTarget = await password.getAttribute('data-target');
    const beforePassword = await hackingSurface(active);
    await password.click();
    await expect.poll(() => activeFrames.filter(message => message.type.startsWith('HACK_'))).toHaveLength(1);
    expect(activeFrames.find(message => message.type.startsWith('HACK_'))).toMatchObject({
      type: 'HACK_GUESS', requestId: expect.any(String), broadcastId: expect.any(String),
      terminalId: 'terminal-1', targetId: passwordTarget,
    });
    await waitForAuthoritativeCompletion(active);
    await assertConverged();
    expect(await hackingSurface(active)).not.toEqual(beforePassword);

    await resetSharedRecording(observer);
    observerFrames.length = 0;
    const canonicalBeforeObserverClick = await hackingSurface(active);
    const observerTarget = observer.locator('.hcell').first();
    await observerTarget.hover();
    await observerTarget.click();
    await observer.waitForTimeout(100);
    expect(observerFrames.filter(message => message.type.startsWith('HACK_'))).toEqual([]);
    expect(await hackingSurface(active)).toEqual(canonicalBeforeObserverClick);
    await assertConverged();

    const canonicalResume = await hackingSurface(active);
    const token = await active.evaluate(key => localStorage.getItem(key), TOKEN_KEY);
    expect(token).toEqual(expect.any(String));
    expect(token.length).toBeGreaterThan(0);

    const sibling = await activeContext.newPage();
    const siblingFrames = recordNativeWebSocketFrames(sibling);
    await sibling.goto('/');
    await expect(sibling.locator('#roleBadge')).toHaveAttribute('data-role', 'active');
    await expect(sibling.locator('#hackBoard')).toBeVisible();
    await expect.poll(() => hackingSurface(sibling)).toEqual(canonicalResume);
    await expect.poll(() => siblingFrames.find(message => message.type === 'SESSION_HELLO')).toMatchObject({
      type: 'SESSION_HELLO', browserToken: token,
    });

    await active.close();
    await sibling.close();
    await new Promise(resolve => setTimeout(resolve, 100));

    const reconnected = await activeContext.newPage();
    const reconnectFrames = recordNativeWebSocketFrames(reconnected);
    await reconnected.goto('/');
    await expect(reconnected.locator('#roleBadge')).toHaveAttribute('data-role', 'active');
    await expect(reconnected.locator('#hackBoard')).toBeVisible();
    await expect.poll(() => hackingSurface(reconnected)).toEqual(canonicalResume);
    await expect.poll(() => reconnectFrames.find(message => message.type === 'SESSION_HELLO')).toMatchObject({
      type: 'SESSION_HELLO', browserToken: token,
    });
  } finally {
    await activeContext.close();
    await observerContext.close();
  }
});

test('controller action stays pending and canonical navigation waits for accepted state at its revision', async ({ page }) => {
  await openAssignedPlayer(page, 'active');

  const docs = page.locator('.term-row').filter({ hasText: 'DOCS' });
  await docs.click();
  await expect(docs).toBeVisible();

  const requests = await sharedOutbound(page);
  expect(requests).toHaveLength(1);
  expect(requests[0]).toMatchObject({
    type: 'NAV_ACTION',
    requestId: expect.any(String),
    broadcastId: 'broadcast-1',
    terminalId: 'terminal-1',
    action: 'enter',
    nodeId: 'docs',
  });
  await expect(page.locator('#screen')).toHaveClass(/shared-input-pending/);

  await page.keyboard.press('Enter');
  await docs.click({ force: true });
  expect(await sharedOutbound(page)).toHaveLength(1);

  await emit(page, {
    type: 'ACTION_RESULT',
    requestId: requests[0].requestId,
    accepted: true,
    reason: 'accepted',
    revision: 3,
  });
  await expect(page.locator('#screen')).toHaveClass(/shared-input-pending/);
  await expect(docs).toBeVisible();

  await emit(page, {
    type: 'NAV_STATE',
    revision: 3,
    terminalId: 'terminal-1',
    nav: { path: ['root', 'docs'], mode: 'list', viewEntryId: null, commandNodeId: null },
  });
  await expect(page.locator('#screen')).not.toHaveClass(/shared-input-pending/);
  await expect(page.locator('.term-row').filter({ hasText: 'REPORT' })).toBeVisible();
});

test('rejected controller action clears pending immediately and leaves the canonical view unchanged', async ({ page }) => {
  await openAssignedPlayer(page, 'active');

  const status = page.locator('.term-row').filter({ hasText: 'STATUS' });
  await status.click();
  const requests = await sharedOutbound(page);
  expect(requests).toHaveLength(1);
  await expect(page.locator('#screen')).toHaveClass(/shared-input-pending/);
  await expect(status).toBeVisible();
  await expect(page.locator('#termEntry')).toBeHidden();

  await emit(page, {
    type: 'ACTION_RESULT',
    requestId: requests[0].requestId,
    accepted: false,
    reason: 'not-controller',
    revision: 2,
  });
  await expect(page.locator('#screen')).not.toHaveClass(/shared-input-pending/);
  await expect(page.locator('#playerNotice')).toContainText(/not-controller|контрол/i);
  await expect(status).toBeVisible();
  await expect(page.locator('#termEntry')).toBeHidden();

  await status.click();
  expect(await sharedOutbound(page)).toHaveLength(2);
});

test('US3 recognized identity survives reload and reopening in the same browser context', async ({ context, page }) => {
  await installContinuityFixture(context);
  await gotoContinuityPage(page);
  await expect.poll(() => outboundMessages(page)).toEqual([{ type: 'SESSION_HELLO' }]);

  await welcomeSession(page, 'profile-token', 'session-profile');
  await expect(page.locator('#playerIdentity')).toContainText('Mara');

  await page.reload();
  await page.waitForFunction(() => window.__playerSocket);
  await expect.poll(() => outboundMessages(page)).toEqual([
    { type: 'SESSION_HELLO', browserToken: 'profile-token' },
  ]);
  await welcomeSession(page, 'profile-token', 'session-profile');
  await expect(page.locator('#playerIdentity')).toContainText('Mara');

  await page.close();
  const reopened = await context.newPage();
  await gotoContinuityPage(reopened);
  await expect.poll(() => outboundMessages(reopened)).toEqual([
    { type: 'SESSION_HELLO', browserToken: 'profile-token' },
  ]);
  await welcomeSession(reopened, 'profile-token', 'session-profile');
  await expect(reopened.locator('#playerIdentity')).toContainText('Mara');
  await expect(reopened.locator('#roleBadge')).toContainText(/актив|active/i);
});

test('US3 three same-context pages share identity while cleared and private-equivalent contexts stay isolated', async ({ browser, context, page }, testInfo) => {
  await installContinuityFixture(context);
  await gotoContinuityPage(page);
  await welcomeSession(page, 'shared-profile-token', 'session-shared');

  const sameContextPages = [page, await context.newPage(), await context.newPage()];
  for (const sibling of sameContextPages.slice(1)) {
    await gotoContinuityPage(sibling);
    await expect.poll(() => outboundMessages(sibling)).toEqual([
      { type: 'SESSION_HELLO', browserToken: 'shared-profile-token' },
    ]);
    await welcomeSession(sibling, 'shared-profile-token', 'session-shared');
  }
  for (const sibling of sameContextPages) {
    await expect(sibling.locator('#playerIdentity')).toContainText('Mara');
    expect(await sibling.evaluate(key => localStorage.getItem(key), TOKEN_KEY)).toBe('shared-profile-token');
  }

  const contextOptions = { baseURL: testInfo.project.use.baseURL };
  const otherContext = await browser.newContext(contextOptions);
  const privateEquivalentContext = await browser.newContext(contextOptions);
  try {
    await installContinuityFixture(otherContext);
    const otherPage = await otherContext.newPage();
    await gotoContinuityPage(otherPage);
    await expect.poll(() => outboundMessages(otherPage)).toEqual([{ type: 'SESSION_HELLO' }]);
    await welcomeSession(otherPage, 'other-profile-token', 'session-other', {
      fallbackName: 'PLAYER 2',
      character: null,
      role: 'unassigned',
      phase: 'selecting',
      activeTerminalId: null,
    });
    expect(await otherPage.evaluate(key => localStorage.getItem(key), TOKEN_KEY)).toBe('other-profile-token');

    await otherPage.evaluate(() => localStorage.clear());
    await otherPage.reload();
    await otherPage.waitForFunction(() => window.__playerSocket);
    await expect.poll(() => outboundMessages(otherPage)).toEqual([{ type: 'SESSION_HELLO' }]);
    await welcomeSession(otherPage, 'cleared-profile-token', 'session-cleared', {
      fallbackName: 'PLAYER 3',
      character: null,
      role: 'unassigned',
      phase: 'selecting',
      activeTerminalId: null,
      roster: [
        { id: 'character-1', name: 'Mara', status: 'claimed' },
        { id: 'character-2', name: 'Boone', status: 'available' },
      ],
    });
    await expect(otherPage.locator('[data-character-id="character-1"]')).toBeDisabled();
    expect(await otherPage.evaluate(key => localStorage.getItem(key), TOKEN_KEY)).toBe('cleared-profile-token');

    await installContinuityFixture(privateEquivalentContext);
    const privatePage = await privateEquivalentContext.newPage();
    await gotoContinuityPage(privatePage);
    await expect.poll(() => outboundMessages(privatePage)).toEqual([{ type: 'SESSION_HELLO' }]);
    await welcomeSession(privatePage, 'private-profile-token', 'session-private', {
      fallbackName: 'PLAYER 4',
      character: null,
      role: 'unassigned',
      phase: 'selecting',
      activeTerminalId: null,
    });
    expect(await privatePage.evaluate(key => localStorage.getItem(key), TOKEN_KEY)).toBe('private-profile-token');

    for (const sibling of sameContextPages) {
      await expect(sibling.locator('#playerIdentity')).toContainText('Mara');
      expect(await sibling.evaluate(key => localStorage.getItem(key), TOKEN_KEY)).toBe('shared-profile-token');
    }
  } finally {
    await otherContext.close();
    await privateEquivalentContext.close();
  }
});

test('US3 stale recognition is replaced and only the replacement is reused', async ({ context, page }) => {
  await installContinuityFixture(context, { seedToken: 'stale-process-token' });
  await gotoContinuityPage(page);
  await expect.poll(() => outboundMessages(page)).toEqual([
    { type: 'SESSION_HELLO', browserToken: 'stale-process-token' },
  ]);

  await welcomeSession(page, 'replacement-token', 'fresh-session', {
    fallbackName: 'PLAYER 9',
    character: null,
    role: 'unassigned',
    phase: 'selecting',
    activeTerminalId: null,
  });
  expect(await page.evaluate(key => localStorage.getItem(key), TOKEN_KEY)).toBe('replacement-token');
  expect(await page.evaluate(() => Object.values(localStorage))).toEqual(['replacement-token']);

  await page.reload();
  await page.waitForFunction(() => window.__playerSocket);
  await expect.poll(() => outboundMessages(page)).toEqual([
    { type: 'SESSION_HELLO', browserToken: 'replacement-token' },
  ]);
});

for (const handshakeMode of [
  { name: 'Web Locks', disableLocks: false },
  { name: 'storage-event fallback', disableLocks: true },
]) {
  test(`US3 first-use tabs serialize token issuance with ${handshakeMode.name}`, async ({ context, page }) => {
    await installContinuityFixture(context, { disableLocks: handshakeMode.disableLocks });
    const pages = [page, await context.newPage(), await context.newPage()];
    await Promise.all(pages.map(candidate => candidate.goto('/')));

    await expect.poll(async () => {
      const batches = await Promise.all(pages.map(candidate => outboundMessages(candidate)));
      return batches.flat().filter(message => message.type === 'SESSION_HELLO').length;
    }).toBe(1);

    const batches = await Promise.all(pages.map(candidate => outboundMessages(candidate)));
    const leaderIndex = batches.findIndex(messages => messages.some(message => message.type === 'SESSION_HELLO'));
    expect(leaderIndex).toBeGreaterThanOrEqual(0);
    expect(batches[leaderIndex]).toEqual([{ type: 'SESSION_HELLO' }]);
    await welcomeSession(pages[leaderIndex], 'serialized-token', 'serialized-session');

    await expect.poll(async () => {
      const hellos = (await Promise.all(pages.map(candidate => outboundMessages(candidate))))
        .flat()
        .filter(message => message.type === 'SESSION_HELLO');
      return {
        total: hellos.length,
        initial: hellos.filter(message => !('browserToken' in message)).length,
        recognized: hellos.filter(message => message.browserToken === 'serialized-token').length,
      };
    }).toEqual({ total: 3, initial: 1, recognized: 2 });

    await expect.poll(async () => Promise.all(
      pages.map(candidate => candidate.evaluate(key => localStorage.getItem(key), TOKEN_KEY))
    )).toEqual(['serialized-token', 'serialized-token', 'serialized-token']);
  });
}

test('US4 authoritative roster renames and claim availability update by stable character ID', async ({ page }) => {
  await openPlayer(page);
  await emit(page, { type: 'SESSION_WELCOME', browserToken: 'token-1', state: selectingState() });

  await expect(page.locator('[data-character-id="character-1"]')).toContainText('Mara');
  await expect(page.locator('[data-character-id="character-1"]')).toBeEnabled();
  await expect(page.locator('[data-character-id="character-2"]')).toBeDisabled();

  await emit(page, {
    type: 'PLAYER_STATE',
    state: {
      ...selectingState(2),
      roster: [
        { id: 'character-1', name: 'Mara Voss', status: 'claimed' },
        { id: 'character-2', name: 'Craig Boone', status: 'available' },
        { id: 'character-3', name: 'Arcade', status: 'available' },
      ],
    },
  });

  await expect(page.locator('[data-character-id="character-1"]')).toContainText('Mara Voss');
  await expect(page.locator('[data-character-id="character-1"]')).toBeDisabled();
  await expect(page.locator('[data-character-id="character-2"]')).toContainText('Craig Boone');
  await expect(page.locator('[data-character-id="character-2"]')).toBeEnabled();
  expect(await page.locator('[data-character-id]').evaluateAll(nodes => nodes.map(node => node.dataset.characterId)))
    .toEqual(['character-1', 'character-2', 'character-3']);
});

test('US4 assignment rename keeps character primary, fallback secondary, and canonical terminal unchanged', async ({ page }) => {
  await openAssignedPlayer(page, 'active');
  await expect(page.locator('.term-row').filter({ hasText: 'STATUS' })).toBeVisible();
  const canonicalBefore = {
    intro: await page.locator('#introTextEl').textContent(),
    rows: await page.locator('#termList').textContent(),
  };

  await emit(page, {
    type: 'PLAYER_STATE',
    state: {
      ...assignedState('active', 3),
      fallbackName: 'TABLET LEFT',
      character: { id: 'character-1', name: 'Mara Voss' },
      roster: [
        { id: 'character-1', name: 'Mara Voss', status: 'claimed' },
        { id: 'character-2', name: 'Boone', status: 'claimed' },
      ],
    },
  });

  await expect(page.locator('#playerIdentity')).toContainText('Mara Voss');
  await expect(page.locator('#playerIdentity')).toContainText('TABLET LEFT');
  const identityText = await page.locator('#playerIdentity').textContent();
  expect(identityText.indexOf('Mara Voss')).toBeLessThan(identityText.indexOf('TABLET LEFT'));
  await expect(page.locator('.term-row').filter({ hasText: 'DOCS' })).toBeVisible();
  expect(await page.locator('#introTextEl').textContent()).toBe(canonicalBefore.intro);
  expect(await page.locator('#termList').textContent()).toBe(canonicalBefore.rows);
  expect(await sharedOutbound(page)).toEqual([]);
});

test('US4 release returns every tab state to selection and later transfer reuses the canonical terminal mirror', async ({ page }) => {
  await openAssignedPlayer(page, 'active');
  await expect(page.locator('.term-row').filter({ hasText: 'DOCS' })).toBeVisible();

  await emit(page, {
    type: 'PLAYER_STATE',
    state: {
      ...selectingState(3),
      roster: [
        { id: 'character-1', name: 'Mara', status: 'available' },
        { id: 'character-2', name: 'Boone', status: 'claimed' },
      ],
    },
  });
  await expect(page.locator('#characterSelect')).toBeVisible();
  await expect(page.locator('[data-character-id="character-1"]')).toBeEnabled();
  await expect(page.locator('#termList')).toBeHidden();

  await emit(page, {
    type: 'PLAYER_STATE',
    state: {
      ...assignedState('observer', 4),
      character: { id: 'character-2', name: 'Boone' },
      roster: [
        { id: 'character-1', name: 'Mara', status: 'available' },
        { id: 'character-2', name: 'Boone', status: 'claimed' },
      ],
    },
  });

  await expect(page.locator('#characterSelect')).toBeHidden();
  await expect(page.locator('#playerIdentity')).toContainText('Boone');
  await expect(page.locator('#roleBadge')).toHaveAttribute('data-role', 'observer');
  await expect(page.locator('.term-row').filter({ hasText: 'DOCS' })).toBeVisible();
  expect(await sharedOutbound(page)).toEqual([]);
});

test('US5 live controller reassignment converges every tab without changing identity or puzzle state', async ({ context, page }) => {
  const formerControllerTabs = [page, await context.newPage()];
  const newControllerTabs = [await context.newPage(), await context.newPage()];
  const roster = [
    { id: 'character-1', name: 'Mara', status: 'claimed' },
    { id: 'character-2', name: 'Boone', status: 'claimed' },
  ];
  const roleState = ({ revision, sessionId, fallbackName, character, role }) => ({
    revision,
    sessionId,
    fallbackName,
    character,
    role,
    phase: role === 'active' ? 'controlling' : 'observing',
    broadcastId: 'broadcast-1',
    activeTerminalId: 'terminal-1',
    roster,
  });
  const hackSnapshot = terminalLive({
    revision: 5,
    hackLevel: 1,
    hack: {
      level: 1,
      wordLength: 5,
      attemptsMax: 4,
      attemptsLeft: 3,
      solved: false,
      failed: false,
      log: ['> VAULT', 'СОВПАДЕНИЙ: 2/5'],
      columns: [{
        addresses: ['0xC000'],
        text: '[!!]VAULT...',
        words: [{ id: 'A1', start: 4, length: 5 }],
      }],
      patterns: [{ id: 'pattern-1', row: 0, start: 0, end: 3, used: false }],
    },
  });
  const surfaceSnapshot = candidate => candidate.evaluate(() => ({
    character: document.querySelector('#playerCharacterName').textContent,
    fallback: document.querySelector('#playerFallbackName').textContent,
    attempts: document.querySelector('#attemptsLine').textContent,
    board: Array.from(document.querySelectorAll('#hackColumns .hcell'), cell => ({
      target: cell.dataset.target,
      row: cell.dataset.row || null,
      offset: cell.dataset.offset || null,
      text: cell.textContent,
    })),
    log: document.querySelector('#hackLog').textContent,
  }));

  for (const candidate of formerControllerTabs) {
    await openPlayer(candidate);
    await emit(candidate, {
      type: 'SESSION_WELCOME',
      browserToken: 'former-controller-token',
      state: roleState({
        revision: 4,
        sessionId: 'session-former',
        fallbackName: 'PLAYER 1',
        character: { id: 'character-1', name: 'Mara' },
        role: 'active',
      }),
    });
    await emit(candidate, hackSnapshot);
    await candidate.evaluate(() => { window.__outboundMessages.length = 0; });
  }
  for (const candidate of newControllerTabs) {
    await openPlayer(candidate);
    await emit(candidate, {
      type: 'SESSION_WELCOME',
      browserToken: 'new-controller-token',
      state: roleState({
        revision: 4,
        sessionId: 'session-new',
        fallbackName: 'PLAYER 2',
        character: { id: 'character-2', name: 'Boone' },
        role: 'observer',
      }),
    });
    await emit(candidate, hackSnapshot);
    await candidate.evaluate(() => { window.__outboundMessages.length = 0; });
  }

  const canonicalBefore = await Promise.all(
    [...formerControllerTabs, ...newControllerTabs].map(surfaceSnapshot)
  );
  const formerLocalWord = formerControllerTabs[0].locator('[data-target="A1"]');
  await formerLocalWord.focus();
  await expect(formerControllerTabs[0].locator('#hackInputPreview')).toHaveText('VAULT');
  await expect(formerLocalWord).toHaveClass(/hi/);

  for (const candidate of formerControllerTabs) {
    await emit(candidate, {
      type: 'PLAYER_STATE',
      state: roleState({
        revision: 6,
        sessionId: 'session-former',
        fallbackName: 'PLAYER 1',
        character: { id: 'character-1', name: 'Mara' },
        role: 'observer',
      }),
    });
  }
  for (const candidate of newControllerTabs) {
    await emit(candidate, {
      type: 'PLAYER_STATE',
      state: roleState({
        revision: 6,
        sessionId: 'session-new',
        fallbackName: 'PLAYER 2',
        character: { id: 'character-2', name: 'Boone' },
        role: 'active',
      }),
    });
  }

  for (const candidate of formerControllerTabs) {
    await expect(candidate.locator('#roleBadge')).toHaveAttribute('data-role', 'observer');
    await expect(candidate.locator('#screen')).toHaveClass(/observer-read-only/);
    await expect(candidate.locator('#playerCharacterName')).toHaveText('Mara');
  }
  for (const candidate of newControllerTabs) {
    await expect(candidate.locator('#roleBadge')).toHaveAttribute('data-role', 'active');
    await expect(candidate.locator('#screen')).not.toHaveClass(/observer-read-only/);
    await expect(candidate.locator('#playerCharacterName')).toHaveText('Boone');
  }
  expect(await Promise.all(
    [...formerControllerTabs, ...newControllerTabs].map(surfaceSnapshot)
  )).toEqual(canonicalBefore);

  await expect(formerControllerTabs[0].locator('#hackInputPreview')).toHaveText('VAULT');
  await expect(formerControllerTabs[0].locator('[data-target="A1"]')).toHaveClass(/hi/);
  await expect(formerControllerTabs[0].locator('[data-target="A1"]')).toBeFocused();

  await formerControllerTabs[0].locator('[data-target="A1"]').click();
  await formerControllerTabs[0].keyboard.press('Enter');
  expect(await sharedOutbound(formerControllerTabs[0])).toEqual([]);

  await newControllerTabs[0].locator('[data-target="A1"]').click();
  const promotedRequests = await sharedOutbound(newControllerTabs[0]);
  expect(promotedRequests).toHaveLength(1);
  expect(promotedRequests[0]).toMatchObject({
    type: 'HACK_GUESS',
    requestId: expect.any(String),
    broadcastId: 'broadcast-1',
    terminalId: 'terminal-1',
    targetId: 'A1',
  });

  for (const candidate of formerControllerTabs) {
    await emit(candidate, {
      type: 'PLAYER_STATE',
      state: roleState({
        revision: 5,
        sessionId: 'session-former',
        fallbackName: 'PLAYER 1',
        character: { id: 'character-1', name: 'Mara' },
        role: 'active',
      }),
    });
    await expect(candidate.locator('#roleBadge')).toHaveAttribute('data-role', 'observer');
  }
  for (const candidate of newControllerTabs) {
    await emit(candidate, {
      type: 'PLAYER_STATE',
      state: roleState({
        revision: 5,
        sessionId: 'session-new',
        fallbackName: 'PLAYER 2',
        character: { id: 'character-2', name: 'Boone' },
        role: 'observer',
      }),
    });
    await expect(candidate.locator('#roleBadge')).toHaveAttribute('data-role', 'active');
  }
});

for (const reconnectCase of [
  { name: 'unchanged controller', role: 'active', maySend: true },
  { name: 'reassigned former controller', role: 'observer', maySend: false },
]) {
  test(`US6 reconnect restores the welcomed ${reconnectCase.name} role without selection or canonical reset`, async ({ page }) => {
    await openAssignedPlayer(page, 'active');
    await emit(page, terminalLive({
      revision: 5,
      hackLevel: 1,
      hack: {
        level: 1,
        wordLength: 5,
        attemptsMax: 4,
        attemptsLeft: 3,
        solved: false,
        failed: false,
        log: ['> VAULT', 'СОВПАДЕНИЙ: 2/5'],
        columns: [{
          addresses: ['0xC000'],
          text: '[!!]VAULT...',
          words: [{ id: 'A1', start: 4, length: 5 }],
        }],
        patterns: [{ id: 'pattern-1', row: 0, start: 0, end: 3, used: false }],
      },
    }));
    await page.evaluate(() => { window.__outboundMessages.length = 0; });

    const word = page.locator('[data-target="A1"]');
    await word.focus();
    await page.keyboard.press('x');
    await expect(page.locator('#hackInputPreview')).toHaveText('x');
    const surfaceBefore = await page.evaluate(() => ({
      character: document.querySelector('#playerCharacterName').textContent,
      fallback: document.querySelector('#playerFallbackName').textContent,
      attempts: document.querySelector('#attemptsLine').textContent,
      board: document.querySelector('#hackColumns').innerHTML,
      log: document.querySelector('#hackLog').textContent,
    }));

    await disconnectAndAwaitReconnect(page);
    await expect(page.locator('#connOverlay')).toBeVisible();
    await expect.poll(() => page.evaluate(() => window.__outboundMessages)).toEqual([
      { type: 'SESSION_HELLO', browserToken: 'token-1' },
    ]);
    await expect(page.locator('#characterSelect')).toBeHidden();
    expect(await page.evaluate(() => ({
      character: document.querySelector('#playerCharacterName').textContent,
      fallback: document.querySelector('#playerFallbackName').textContent,
      attempts: document.querySelector('#attemptsLine').textContent,
      board: document.querySelector('#hackColumns').innerHTML,
      log: document.querySelector('#hackLog').textContent,
    }))).toEqual(surfaceBefore);

    await emit(page, {
      type: 'SESSION_WELCOME',
      browserToken: 'token-1',
      state: assignedState(reconnectCase.role, 7),
    });

    await expect(page.locator('#connOverlay')).toBeHidden();
    await expect(page.locator('#characterSelect')).toBeHidden();
    await expect(page.locator('#playerCharacterName')).toHaveText('Mara');
    await expect(page.locator('#roleBadge')).toHaveAttribute('data-role', reconnectCase.role);
    await expect(page.locator('#hackInputPreview')).toHaveText('x');
    await expect(page.locator('[data-target="A1"]')).toBeFocused();
    expect(await page.evaluate(() => ({
      character: document.querySelector('#playerCharacterName').textContent,
      fallback: document.querySelector('#playerFallbackName').textContent,
      attempts: document.querySelector('#attemptsLine').textContent,
      board: document.querySelector('#hackColumns').innerHTML,
      log: document.querySelector('#hackLog').textContent,
    }))).toEqual(surfaceBefore);

    await page.evaluate(() => { window.__outboundMessages.length = 0; });
    await page.locator('[data-target="A1"]').click();
    const requests = await sharedOutbound(page);
    expect(requests).toHaveLength(reconnectCase.maySend ? 1 : 0);
    if (reconnectCase.maySend) {
      expect(requests[0]).toMatchObject({
        type: 'HACK_GUESS',
        requestId: expect.any(String),
        broadcastId: 'broadcast-1',
        terminalId: 'terminal-1',
        targetId: 'A1',
      });
    } else {
      await expect(page.locator('#screen')).toHaveClass(/observer-read-only/);
    }
  });
}

test('US7 assigned players wait with no terminal, follow ten revealed switches, and a late assignee joins the current terminal', async ({ context, page }) => {
  const activePage = page;
  const observerPage = await context.newPage();
  const assignedPages = [activePage, observerPage];
  const switchedTree = index => ({
    id: 'root',
    type: 'folder',
    name: 'ROOT',
    children: [1, 2, 3].map(row => ({
      id: `terminal-${index}-entry-${row}`,
      type: 'entry',
      name: `TERMINAL ${index} ENTRY ${row}`,
      description: `TERMINAL ${index} CONTENT ${row}`,
    })),
  });
  const assignedTerminalState = (role, revision, terminalId, overrides = {}) => ({
    ...assignedState(role, revision),
    activeTerminalId: terminalId,
    phase: terminalId === null ? 'waiting' : (role === 'active' ? 'controlling' : 'observing'),
    ...overrides,
  });
  const assertStableIdentity = async (candidate, character, fallback, role) => {
    await expect(candidate.locator('#characterSelect')).toBeHidden();
    await expect(candidate.locator('#playerCharacterName')).toHaveText(character);
    await expect(candidate.locator('#playerFallbackName')).toHaveText(fallback);
    await expect(candidate.locator('#roleBadge')).toHaveAttribute('data-role', role);
  };

  await openAssignedPlayer(activePage, 'active');
  await openPlayer(observerPage);
  await emit(observerPage, {
    type: 'SESSION_WELCOME',
    browserToken: 'observer-token',
    state: assignedTerminalState('observer', 2, 'terminal-1', {
      sessionId: 'session-observer',
      fallbackName: 'PLAYER 2',
      character: { id: 'character-2', name: 'Boone' },
    }),
  });
  await emit(observerPage, terminalLive());
  await observerPage.evaluate(() => { window.__outboundMessages.length = 0; });

  for (const [candidate, role, sessionId, fallbackName, character] of [
    [activePage, 'active', 'session-1', 'PLAYER 1', { id: 'character-1', name: 'Mara' }],
    [observerPage, 'observer', 'session-observer', 'PLAYER 2', { id: 'character-2', name: 'Boone' }],
  ]) {
    await emit(candidate, {
      type: 'PLAYER_STATE',
      state: assignedTerminalState(role, 3, null, { sessionId, fallbackName, character }),
    });
    await emit(candidate, { type: 'TERMINAL_CLEAR', revision: 3 });
    await expect(candidate.locator('#assignedWaiting')).toBeVisible();
    await assertStableIdentity(candidate, character.name, fallbackName, role);
  }

  let currentTerminal = null;
  let currentTree = null;
  for (let index = 1; index <= 10; index++) {
    const revision = index + 3;
    currentTerminal = `terminal-${index + 1}`;
    currentTree = switchedTree(index + 1);

    for (const [candidate, role, sessionId, fallbackName, character] of [
      [activePage, 'active', 'session-1', 'PLAYER 1', { id: 'character-1', name: 'Mara' }],
      [observerPage, 'observer', 'session-observer', 'PLAYER 2', { id: 'character-2', name: 'Boone' }],
    ]) {
      await emit(candidate, {
        type: 'PLAYER_STATE',
        state: assignedTerminalState(role, revision, currentTerminal, {
          sessionId,
          fallbackName,
          character,
        }),
      });
      await expect(candidate.locator('#assignedWaiting')).toBeHidden();
      await expect(candidate.locator('#termIdle')).toBeVisible();
      await assertStableIdentity(candidate, character.name, fallbackName, role);

      await candidate.evaluate(() => {
        window.__terminalRevealCounts = [];
        window.__terminalRevealObserver?.disconnect();
        window.__terminalRevealObserver = new MutationObserver(() => {
          window.__terminalRevealCounts.push(document.querySelector('#termList').children.length);
        });
        window.__terminalRevealObserver.observe(document.querySelector('#termList'), { childList: true });
      });
      await emit(candidate, terminalLive({
        revision,
        terminalId: currentTerminal,
        terminalName: `TERMINAL ${index + 1}`,
        introText: `SWITCH ${index + 1}`,
        tree: currentTree,
      }));

      await expect(candidate.locator('#introTextEl')).toHaveText(`SWITCH ${index + 1}`);
      await expect(candidate.locator('.term-row')).toHaveCount(3);
      await expect(candidate.locator('.term-row').last()).toContainText(`TERMINAL ${index + 1} ENTRY 3`);
      await expect.poll(() => candidate.evaluate(() => window.__terminalRevealCounts)).toEqual([1, 2, 3]);
      await assertStableIdentity(candidate, character.name, fallbackName, role);
    }
  }

  expect(await sharedOutbound(activePage)).toEqual([]);
  expect(await sharedOutbound(observerPage)).toEqual([]);

  const latePage = await context.newPage();
  await openPlayer(latePage);
  await emit(latePage, {
    type: 'SESSION_WELCOME',
    browserToken: 'late-token',
    state: {
      ...selectingState(14),
      sessionId: 'session-late',
      fallbackName: 'PLAYER 3',
      activeTerminalId: currentTerminal,
      roster: [
        { id: 'character-1', name: 'Mara', status: 'claimed' },
        { id: 'character-2', name: 'Boone', status: 'claimed' },
        { id: 'character-3', name: 'Arcade', status: 'available' },
      ],
    },
  });
  await latePage.locator('[data-character-id="character-3"]').click();
  const selection = await latePage.evaluate(() =>
    window.__outboundMessages.find(message => message.type === 'CHARACTER_SELECT')
  );
  await emit(latePage, {
    type: 'ACTION_RESULT',
    requestId: selection.requestId,
    accepted: true,
    reason: 'accepted',
    revision: 15,
  });
  await emit(latePage, {
    type: 'PLAYER_STATE',
    state: assignedTerminalState('observer', 15, currentTerminal, {
      sessionId: 'session-late',
      fallbackName: 'PLAYER 3',
      character: { id: 'character-3', name: 'Arcade' },
      roster: [
        { id: 'character-1', name: 'Mara', status: 'claimed' },
        { id: 'character-2', name: 'Boone', status: 'claimed' },
        { id: 'character-3', name: 'Arcade', status: 'claimed' },
      ],
    }),
  });

  await expect(latePage.locator('#characterSelect')).toBeHidden();
  await expect(latePage.locator('#termIdle')).toBeVisible();
  await assertStableIdentity(latePage, 'Arcade', 'PLAYER 3', 'observer');
  await emit(latePage, terminalLive({
    revision: 15,
    terminalId: currentTerminal,
    terminalName: 'CURRENT TERMINAL',
    introText: 'CURRENT SWITCH',
    tree: currentTree,
  }));
  await expect(latePage.locator('#introTextEl')).toHaveText('CURRENT SWITCH');
  await expect(latePage.locator('.term-row')).toHaveCount(3);
  await expect(latePage.locator('.term-row').last()).toContainText('ENTRY 3');
  await assertStableIdentity(latePage, 'Arcade', 'PLAYER 3', 'observer');
});

test('US8 pending and cancelled switches keep the source, preserve restores it, and discard returns a fresh puzzle', async ({ page }) => {
  const sourceHack = {
    level: 1,
    wordLength: 5,
    attemptsMax: 4,
    attemptsLeft: 3,
    solved: false,
    failed: false,
    log: ['> VAULT', 'СОВПАДЕНИЙ: 2/5'],
    columns: [{
      addresses: ['0xC000'],
      text: '[!!]VAULT...',
      words: [{ id: 'A1', start: 4, length: 5 }],
    }],
    patterns: [{ id: 'source-pattern', row: 0, start: 0, end: 3, used: true }],
  };
  const targetHack = {
    level: 1,
    wordLength: 5,
    attemptsMax: 4,
    attemptsLeft: 2,
    solved: false,
    failed: false,
    log: ['> ROBOT', 'СОВПАДЕНИЙ: 1/5'],
    columns: [{
      addresses: ['0xD000'],
      text: '()()ROBOT...',
      words: [{ id: 'B1', start: 4, length: 5 }],
    }],
    patterns: [{ id: 'target-pattern', row: 0, start: 0, end: 1, used: false }],
  };
  const freshSourceHack = {
    level: 1,
    wordLength: 5,
    attemptsMax: 4,
    attemptsLeft: 4,
    solved: false,
    failed: false,
    log: [],
    columns: [{
      addresses: ['0xE000'],
      text: '{}{}LASER...',
      words: [{ id: 'C1', start: 4, length: 5 }],
    }],
    patterns: [{ id: 'fresh-pattern', row: 0, start: 0, end: 1, used: false }],
  };
  const terminalState = (revision, terminalId) => ({
    ...assignedState('active', revision),
    activeTerminalId: terminalId,
    phase: 'controlling',
  });
  const livePuzzle = (revision, terminalId, introText, hack) => terminalLive({
    revision,
    terminalId,
    terminalName: terminalId === 'terminal-1' ? 'SOURCE' : 'TARGET',
    introText,
    hackLevel: 1,
    hack,
  });
  const puzzleSurface = candidate => candidate.evaluate(() => ({
    attempts: document.querySelector('#attemptsLine').textContent,
    cells: Array.from(document.querySelectorAll('#hackColumns .hcell'), cell => ({
      className: cell.className,
      target: cell.dataset.target,
      row: cell.dataset.row || null,
      offset: cell.dataset.offset || null,
      text: cell.textContent,
    })),
    log: document.querySelector('#hackLog').textContent,
    blocked: !document.querySelector('#hackBlocked').hidden,
  }));
  const expectStablePlayer = async () => {
    await expect(page.locator('#characterSelect')).toBeHidden();
    await expect(page.locator('#playerCharacterName')).toHaveText('Mara');
    await expect(page.locator('#playerFallbackName')).toHaveText('PLAYER 1');
    await expect(page.locator('#roleBadge')).toHaveAttribute('data-role', 'active');
  };

  await openAssignedPlayer(page, 'active');
  await emit(page, livePuzzle(3, 'terminal-1', 'SOURCE PUZZLE', sourceHack));
  await expect(page.locator('#hackBoard')).toBeVisible();
  const sourceBeforeSwitch = await puzzleSurface(page);
  await expectStablePlayer();

  await test.step('a decision-required request and cancellation leave the source authoritative', async () => {
    await emit(page, livePuzzle(4, 'terminal-2', 'PREMATURE TARGET', targetHack));
    expect(await puzzleSurface(page)).toEqual(sourceBeforeSwitch);
    await expectStablePlayer();

    await emit(page, {
      type: 'PLAYER_STATE',
      state: terminalState(5, 'terminal-1'),
    });
    expect(await puzzleSurface(page)).toEqual(sourceBeforeSwitch);
    await expectStablePlayer();
  });

  await test.step('preserve switches away and restores the exact public source puzzle', async () => {
    await emit(page, {
      type: 'PLAYER_STATE',
      state: terminalState(6, 'terminal-2'),
    });
    await emit(page, livePuzzle(6, 'terminal-2', 'TARGET PUZZLE', targetHack));
    const targetSurface = await puzzleSurface(page);
    expect(targetSurface).not.toEqual(sourceBeforeSwitch);
    await expectStablePlayer();

    await emit(page, {
      type: 'PLAYER_STATE',
      state: terminalState(7, 'terminal-1'),
    });
    await emit(page, livePuzzle(7, 'terminal-1', 'SOURCE PUZZLE', sourceHack));
    expect(await puzzleSurface(page)).toEqual(sourceBeforeSwitch);
    await expectStablePlayer();
  });

  await test.step('discard switches away and later creates a visibly fresh source puzzle', async () => {
    await emit(page, {
      type: 'PLAYER_STATE',
      state: terminalState(8, 'terminal-2'),
    });
    await emit(page, livePuzzle(8, 'terminal-2', 'TARGET PUZZLE', targetHack));
    await emit(page, {
      type: 'PLAYER_STATE',
      state: terminalState(9, 'terminal-1'),
    });
    await emit(page, livePuzzle(9, 'terminal-1', 'FRESH SOURCE PUZZLE', freshSourceHack));

    const freshSurface = await puzzleSurface(page);
    expect(freshSurface).not.toEqual(sourceBeforeSwitch);
    expect(freshSurface).toMatchObject({
      attempts: expect.stringContaining('4'),
      log: '',
      blocked: false,
    });
    expect(freshSurface.cells.some(cell => cell.target === 'C1' && cell.text === 'LASER')).toBe(true);
    expect(freshSurface.cells.some(cell => cell.target === 'A1' || cell.text === 'VAULT')).toBe(false);
    await expectStablePlayer();
  });

  expect(await sharedOutbound(page)).toEqual([]);
});

test('US9 broadcast end clears pending terminal ownership, requires reselection, and restart welcomes a fresh session', async ({ page }) => {
  const roster = [
    { id: 'character-1', name: 'Mara', status: 'available' },
    { id: 'character-2', name: 'Boone', status: 'available' },
  ];
  const lifecycleState = (overrides = {}) => ({
    revision: 1,
    sessionId: 'session-1',
    fallbackName: 'TABLET LEFT',
    character: null,
    role: 'unassigned',
    phase: 'no-broadcast',
    broadcastId: null,
    activeTerminalId: null,
    roster,
    ...overrides,
  });

  await openPlayer(page);
  await emit(page, {
    type: 'SESSION_WELCOME',
    browserToken: 'token-before-restart',
    state: lifecycleState({
      revision: 2,
      character: { id: 'character-1', name: 'Mara' },
      role: 'active',
      phase: 'controlling',
      broadcastId: 'broadcast-1',
      activeTerminalId: 'terminal-1',
      roster: roster.map(entry => ({ ...entry, status: entry.id === 'character-1' ? 'claimed' : entry.status })),
    }),
  });
  await emit(page, terminalLive({ revision: 2 }));
  await expect(page.locator('.term-row').filter({ hasText: 'DOCS' })).toBeVisible();
  await page.evaluate(() => { window.__outboundMessages.length = 0; });

  await page.locator('.term-row').filter({ hasText: 'DOCS' }).click();
  const oldPendingRequest = await page.evaluate(() =>
    window.__outboundMessages.find(message => message.type === 'NAV_ACTION')
  );
  expect(oldPendingRequest).toMatchObject({
    broadcastId: 'broadcast-1',
    terminalId: 'terminal-1',
    requestId: expect.any(String),
  });
  await expect(page.locator('#screen')).toHaveClass(/shared-input-pending/);

  await test.step('ending the broadcast immediately returns to the unassigned waiting surface', async () => {
    await emit(page, {
      type: 'PLAYER_STATE',
      state: lifecycleState({ revision: 3 }),
    });

    await expect(page.locator('#screen')).not.toHaveClass(/shared-input-pending/);
    await expect(page.locator('#termIdle')).toBeVisible();
    await expect(page.locator('#termList')).toBeHidden();
    await expect(page.locator('#hackBoard')).toBeHidden();
    await expect(page.locator('#characterSelect')).toBeHidden();
    await expect(page.locator('#assignedWaiting')).toBeHidden();
    await expect(page.locator('#playerCharacterName')).toHaveText('TABLET LEFT');
    await expect(page.locator('#playerFallbackName')).toHaveText('');
    await expect(page.locator('#roleBadge')).toHaveAttribute('data-role', 'unassigned');
  });

  await test.step('a second broadcast requires a fresh character selection', async () => {
    await emit(page, {
      type: 'PLAYER_STATE',
      state: lifecycleState({
        revision: 4,
        phase: 'selecting',
        broadcastId: 'broadcast-2',
      }),
    });

    await expect(page.locator('#characterSelect')).toBeVisible();
    await expect(page.locator('#assignedWaiting')).toBeHidden();
    await expect(page.locator('[data-character-id="character-1"]')).toBeEnabled();
    await expect(page.locator('[data-character-id="character-2"]')).toBeEnabled();
    await expect(page.locator('#termList')).toBeHidden();

    await emit(page, {
      type: 'ACTION_RESULT',
      requestId: oldPendingRequest.requestId,
      accepted: true,
      reason: 'accepted',
      revision: 99,
    });
    await emit(page, {
      type: 'NAV_STATE',
      revision: 99,
      terminalId: 'terminal-1',
      nav: { path: ['root', 'docs'], mode: 'list', viewEntryId: null, commandNodeId: null },
    });
    await emit(page, terminalLive({ revision: 99, introText: 'STALE FIRST BROADCAST' }));

    await expect(page.locator('#screen')).not.toHaveClass(/shared-input-pending/);
    await expect(page.locator('#characterSelect')).toBeVisible();
    await expect(page.locator('#introTextEl')).not.toHaveText('STALE FIRST BROADCAST');
    await expect(page.locator('#termList')).toBeHidden();

    await page.locator('[data-character-id="character-2"]').click();
    const secondBroadcastSelection = await page.evaluate(() =>
      window.__outboundMessages.filter(message => message.type === 'CHARACTER_SELECT').at(-1)
    );
    expect(secondBroadcastSelection).toMatchObject({
      type: 'CHARACTER_SELECT',
      requestId: expect.any(String),
      broadcastId: 'broadcast-2',
      characterId: 'character-2',
    });
    expect(secondBroadcastSelection.requestId).not.toBe(oldPendingRequest.requestId);

    await emit(page, {
      type: 'ACTION_RESULT',
      requestId: secondBroadcastSelection.requestId,
      accepted: true,
      reason: 'accepted',
      revision: 5,
    });
    await emit(page, {
      type: 'PLAYER_STATE',
      state: lifecycleState({
        revision: 5,
        character: { id: 'character-2', name: 'Boone' },
        role: 'active',
        phase: 'waiting',
        broadcastId: 'broadcast-2',
        roster: roster.map(entry => ({ ...entry, status: entry.id === 'character-2' ? 'claimed' : entry.status })),
      }),
    });

    await expect(page.locator('#characterSelect')).toBeHidden();
    await expect(page.locator('#assignedWaiting')).toBeVisible();
    await expect(page.locator('#playerCharacterName')).toHaveText('Boone');
    await expect(page.locator('#playerFallbackName')).toHaveText('TABLET LEFT');
    await expect(page.locator('#roleBadge')).toHaveAttribute('data-role', 'active');
    await expect(page.locator('#screen')).not.toHaveClass(/shared-input-pending/);
  });

  await test.step('an old token receives a clean authoritative welcome after process restart', async () => {
    await page.evaluate(() => { window.__outboundMessages.length = 0; });
    await disconnectAndAwaitReconnect(page);
    await expect.poll(() => page.evaluate(() => window.__outboundMessages)).toEqual([
      { type: 'SESSION_HELLO', browserToken: 'token-before-restart' },
    ]);

    await emit(page, {
      type: 'SESSION_WELCOME',
      browserToken: 'token-after-restart',
      state: {
        revision: 1,
        sessionId: 'fresh-session-after-restart',
        fallbackName: 'PLAYER 1',
        character: null,
        role: 'unassigned',
        phase: 'selecting',
        broadcastId: 'broadcast-after-restart',
        activeTerminalId: null,
        roster: [{ id: 'fresh-character', name: 'Arcade', status: 'available' }],
      },
    });

    await expect(page.locator('#connOverlay')).toBeHidden();
    await expect(page.locator('#characterSelect')).toBeVisible();
    await expect(page.locator('[data-character-id="fresh-character"]')).toHaveText('Arcade');
    await expect(page.locator('[data-character-id="character-1"]')).toHaveCount(0);
    await expect(page.locator('#playerCharacterName')).toHaveText('PLAYER 1');
    await expect(page.locator('#playerFallbackName')).toHaveText('');
    await expect(page.locator('#roleBadge')).toHaveAttribute('data-role', 'unassigned');
    await expect(page.locator('#assignedWaiting')).toBeHidden();
    await expect(page.locator('#termList')).toBeHidden();
    expect(await page.evaluate(key => localStorage.getItem(key), TOKEN_KEY)).toBe('token-after-restart');
  });
});

test('BUG-001/BUG-002 master workflow accepts an active player config with an empty roster', async () => {
  const [markup, script] = await Promise.all([
    readFile(new URL('../../frontend/src/index.html', import.meta.url), 'utf8'),
    readFile(new URL('../../frontend/src/master.js', import.meta.url), 'utf8'),
  ]);

  for (const id of ['playerConfigStatus', 'btnOpenPlayerConfig', 'btnNewPlayerConfig', 'playerConfigError']) {
    expect(markup).toContain(`id="${id}"`);
  }
  expect(script).toContain('loadReferencedPlayerConfig');
  expect(script).toContain('openPlayerConfig');
  expect(script).toContain('newPlayerConfig');
  expect(script).toContain('if (result.session) state.session = result.session;');
  expect(script).toContain('applyCoordinationState(result.state || state.coordination);');
  expect(script).toContain('const playerConfig = coordination?.playerConfig || null;');
  expect(script).toContain('const roster = Array.isArray(coordination?.roster) ? coordination.roster : [];');
  expect(script).toMatch(/btnStartBroadcast\.disabled[^;]*playerConfig/);
  expect(script).toMatch(/btnAddCharacter\.disabled[^;]*playerConfig/);
  expect(script).not.toMatch(/(?:btnStartBroadcast|btnAddCharacter)\.disabled[^;]*roster\.length/);
});

test('US4 player-config cancellation stays informational while editing remains available', async ({ page }) => {
  const [rawMarkup, script] = await Promise.all([
    readFile(new URL('../../frontend/src/index.html', import.meta.url), 'utf8'),
    readFile(new URL('../../frontend/src/master.js', import.meta.url), 'utf8'),
  ]);
  const markup = rawMarkup
    .replace(/<meta[^>]+http-equiv="Content-Security-Policy"[^>]*>/i, '')
    .replace(/<script type="module" src="\.\/(?:desktop-api|master)\.js"><\/script>/g, '');

  await page.setContent(markup);
  await page.evaluate(() => {
    const canceled = async () => ({ ok: false, canceled: true, error: '', state: null });
    const session = {
      version: 1,
      name: 'Cancellation Fixture',
      terminals: [{
        id: 'terminal-1',
        name: 'Overseer',
        hackLevel: 0,
        introText: '',
        root: { id: 'root', type: 'folder', name: 'ROOT', children: [] },
      }],
    };
    window.desktopAPI = new Proxy({
      newSession: async () => ({ ok: true, filePath: '/tmp/cancellation.json', session }),
      newPlayerConfig: canceled,
      openPlayerConfig: canceled,
    }, {
      get(target, property) {
        if (property in target) return target[property];
        if (String(property).startsWith('on')) return () => () => {};
        return async () => ({ ok: true });
      },
    });
  });
  await page.addScriptTag({ content: script });

  await page.locator('#btnNewSession').click();
  await expect(page.locator('#mainLayout')).toBeVisible();
  await expect(page.locator('#playerConfigError')).toBeVisible();

  for (const button of ['#btnNewPlayerConfig', '#btnOpenPlayerConfig']) {
    await page.locator(button).click();
    await expect(page.locator('#coordinationStatus')).toContainText('ОТМЕНЁН');
    await expect(page.locator('#playerConfigError')).toBeHidden();
    await expect(page.locator('#playerConfigError')).toHaveText('');
    await expect(page.locator('#btnAddTerminal')).toBeEnabled();
    await expect(page.locator('#btnAddCharacter')).toBeDisabled();
    await expect(page.locator('#characterNameInput')).toBeDisabled();
    await expect(page.locator('#btnStartBroadcast')).toBeDisabled();
    await expect(page.locator('#characterRoster .roster-row')).toHaveCount(0);
  }
});

test('BUG-004 master end-broadcast control confirms in-app and invokes the authoritative command once', async ({ page }) => {
  const [rawMarkup, script] = await Promise.all([
    readFile(new URL('../../frontend/src/index.html', import.meta.url), 'utf8'),
    readFile(new URL('../../frontend/src/master.js', import.meta.url), 'utf8'),
  ]);
  const markup = rawMarkup
    .replace(/<meta[^>]+http-equiv="Content-Security-Policy"[^>]*>/i, '')
    .replace(/<script type="module" src="\.\/(?:desktop-api|master)\.js"><\/script>/g, '');
  const activeState = {
    revision: 17,
    playerConfig: { name: 'BUG-004', filePath: '/tmp/bug-004-players.json' },
    roster: [{ id: 'character-1', name: 'Mara', claimedBySessionId: 'session-1' }],
    sessions: [{
      id: 'session-1',
      fallbackName: 'TABLET LEFT',
      connected: true,
      character: { id: 'character-1', name: 'Mara' },
      role: 'active',
    }],
    broadcast: { id: 'broadcast-1', activeTerminalId: 'terminal-1' },
    pendingSwitch: null,
  };

  await page.setContent(markup);
  await page.evaluate((coordination) => {
    window.__endBroadcastCalls = 0;
    const session = {
      version: 1,
      name: 'BUG-004 Fixture',
      terminals: [{
        id: 'terminal-1',
        name: 'Overseer',
        hackLevel: 0,
        introText: '',
        root: { id: 'root', type: 'folder', name: 'ROOT', children: [] },
      }],
    };
    window.desktopAPI = new Proxy({
      newSession: async () => ({ ok: true, filePath: '/tmp/bug-004.json', session }),
      endBroadcast: async () => {
        window.__endBroadcastCalls += 1;
        return {
          ok: true,
          state: {
            ...coordination,
            revision: coordination.revision + 1,
            roster: coordination.roster.map(character => ({
              id: character.id,
              name: character.name,
              claimedBySessionId: null,
            })),
            sessions: coordination.sessions.map(logicalSession => ({
              ...logicalSession,
              character: null,
              role: 'unassigned',
            })),
            broadcast: null,
          },
        };
      },
      onCoordinationState: (callback) => {
        queueMicrotask(() => callback(structuredClone(coordination)));
        return () => {};
      },
    }, {
      get(target, property) {
        if (property in target) return target[property];
        if (String(property).startsWith('on')) return () => () => {};
        return async () => ({ ok: true });
      },
    });
  }, activeState);
  await page.addScriptTag({ content: script });
  await page.locator('#btnNewSession').click();

  const endBroadcast = page.locator('#btnEndBroadcast');
  await expect(endBroadcast).toBeVisible();
  await expect(endBroadcast).toBeEnabled();
  await endBroadcast.click();
  await expect(page.locator('#endBroadcastDialog')).toBeVisible();
  await page.locator('#btnConfirmEndBroadcast').click();

  await expect.poll(() => page.evaluate(() => window.__endBroadcastCalls)).toBe(1);
  await expect(page.locator('#broadcastSummary')).toHaveText('ТРАНСЛЯЦИЯ НЕ ЗАПУЩЕНА');
  await expect(endBroadcast).toBeHidden();
  await expect(page.locator('#coordinationStatus')).toContainText('ТРАНСЛЯЦИЯ ЗАВЕРШЕНА');
});

test('BUG-005 master retries one failed puzzle and active and observer converge without identity drift', async ({ context, page }) => {
  const active = await context.newPage();
  const observer = await context.newPage();
  await openAssignedPlayer(active, 'active');
  await openAssignedPlayer(observer, 'observer');

  const hackAt = attemptsLeft => ({
    level: 1,
    wordLength: 5,
    attemptsMax: 4,
    attemptsLeft,
    solved: false,
    failed: attemptsLeft === 0,
    log: Array.from({ length: 4 - attemptsLeft }, () => '> Отказ в доступе'),
    columns: [{ addresses: ['0xC000'], text: '............', words: [] }],
    patterns: [],
  });
  const initial = terminalLive({ revision: 3, hackLevel: 1, hack: hackAt(4) });
  await emit(active, initial);
  await emit(observer, initial);
  await active.evaluate(() => { window.__outboundMessages.length = 0; });
  await observer.evaluate(() => { window.__outboundMessages.length = 0; });

  let revision = 3;
  for (let attempt = 1; attempt <= 4; attempt++) {
    await active.locator('[data-target="0:0"]').click();
    const request = (await sharedOutbound(active)).at(-1);
    revision += 1;
    await emit(active, {
      type: 'ACTION_RESULT', requestId: request.requestId, accepted: true,
      reason: 'accepted', revision,
    });
    const update = { type: 'HACK_STATE', revision, terminalId: 'terminal-1', hack: hackAt(4 - attempt) };
    await emit(active, update);
    await emit(observer, update);
  }
  await expect(active.locator('#hackBlocked')).toBeVisible();
  await expect(observer.locator('#hackBlocked')).toBeVisible();
  expect(await sharedOutbound(active)).toHaveLength(4);
  expect(await sharedOutbound(observer)).toEqual([]);

  const [rawMarkup, script] = await Promise.all([
    readFile(new URL('../../frontend/src/index.html', import.meta.url), 'utf8'),
    readFile(new URL('../../frontend/src/master.js', import.meta.url), 'utf8'),
  ]);
  const markup = rawMarkup
    .replace(/<meta[^>]+http-equiv="Content-Security-Policy"[^>]*>/i, '')
    .replace(/<script type="module" src="\.\/(?:desktop-api|master)\.js"><\/script>/g, '');
  const coordination = {
    revision,
    playerConfig: { name: 'BUG-005', filePath: '/tmp/bug-005-players.json' },
    roster: [
      { id: 'character-1', name: 'Mara', claimedBySessionId: 'session-active' },
      { id: 'character-2', name: 'Boone', claimedBySessionId: 'session-observer' },
    ],
    sessions: [
      { id: 'session-active', fallbackName: 'PLAYER 1', connected: true, character: { id: 'character-1', name: 'Mara' }, role: 'active' },
      { id: 'session-observer', fallbackName: 'PLAYER 2', connected: true, character: { id: 'character-2', name: 'Boone' }, role: 'observer' },
    ],
    broadcast: { id: 'broadcast-1', activeTerminalId: 'terminal-1', controllerSessionId: 'session-active' },
    pendingSwitch: null,
  };
  await page.setContent(markup);
  await page.evaluate(({ coordinationState, failedHack }) => {
    window.__resetFailedHackCalls = [];
    let hackCallback = () => {};
    const session = {
      version: 1,
      name: 'BUG-005 Fixture',
      terminals: [{
        id: 'terminal-1', name: 'Overseer', hackLevel: 2, introText: 'LATEST',
        root: { id: 'root', type: 'folder', name: 'ROOT', children: [] },
      }],
    };
    window.desktopAPI = new Proxy({
      newSession: async () => ({ ok: true, filePath: '/tmp/bug-005.json', session }),
      resetFailedHack: async payload => {
        window.__resetFailedHackCalls.push(structuredClone(payload));
        return { ok: true, state: { ...coordinationState, revision: coordinationState.revision + 1 } };
      },
      onCoordinationState: callback => {
        queueMicrotask(() => callback(structuredClone(coordinationState)));
        return () => {};
      },
      onHackState: callback => {
        hackCallback = callback;
        queueMicrotask(() => callback(structuredClone(failedHack)));
        return () => {};
      },
    }, {
      get(target, property) {
        if (property in target) return target[property];
        if (String(property).startsWith('on')) return () => () => {};
        return async () => ({ ok: true });
      },
    });
    window.__publishMasterHack = value => hackCallback(structuredClone(value));
  }, { coordinationState: coordination, failedHack: hackAt(0) });
  await page.addScriptTag({ content: script });
  await page.locator('#btnNewSession').click();

  await page.evaluate(value => window.__publishMasterHack(value), hackAt(0));

  await expect(page.locator('#hackStatusLine')).toContainText('ВЗЛОМ: ЗАБЛОКИРОВАН');
  const retry = page.locator('#btnResetFailedHack');
  await expect(retry).toBeVisible();
  await expect(retry).toBeEnabled();
  await retry.click();
  await expect.poll(() => page.evaluate(() => window.__resetFailedHackCalls.length)).toBe(1);
  expect(await page.evaluate(() => window.__resetFailedHackCalls[0])).toMatchObject({
    terminalId: 'terminal-1', terminalName: 'Overseer', hackLevel: 2, introText: 'LATEST',
  });

  const freshHack = {
    ...hackAt(4),
    level: 2,
    columns: [{ addresses: ['0xD000'], text: '....LASER...', words: [{ id: 'retry:A1', start: 4, length: 5 }] }],
  };
  const freshLive = terminalLive({
    revision: revision + 1, terminalName: 'Overseer', hackLevel: 2, introText: 'LATEST', hack: freshHack,
  });
  await emit(active, freshLive);
  await emit(observer, freshLive);
  await page.evaluate(value => window.__publishMasterHack(value), freshHack);

  await expect(active.locator('#hackBlocked')).toBeHidden();
  await expect(observer.locator('#hackBlocked')).toBeHidden();
  await expect(active.locator('[data-target="retry:A1"]')).toHaveText('LASER');
  await expect(observer.locator('[data-target="retry:A1"]')).toHaveText('LASER');
  await expect(active.locator('#roleBadge')).toHaveAttribute('data-role', 'active');
  await expect(observer.locator('#roleBadge')).toHaveAttribute('data-role', 'observer');
  await expect(page.locator('#hackStatusLine')).toContainText('осталось попыток 4/4');
});
