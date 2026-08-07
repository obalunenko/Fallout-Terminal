'use strict';

const MODE = { LIST: 'list', ENTRY: 'entry', HACK: 'hack' };
const ROW_WIDTH = 12; // must match server/hack.js

// ── State ─────────────────────────────────────────────────
// navStack / mode(list|entry) / viewEntryId / commandOutput are mirrors of
// the server's shared nav state — every connected player sees the same
// position. Only selIndex (the keyboard highlight) stays local per client.
let hasLive       = false;
let terminalName  = '';
let introText     = '';
let serverNum     = 1;
let tree          = null;   // root node {id:'root', type:'folder', name, children:[]}
let navStack      = ['root'];
let selIndex      = 0;
let mode          = MODE.LIST;
let viewEntryId   = null;
let commandOutput = null;
let currentCommandNodeId = null;

// Typewriter reveal: only replay when the shown content actually changed.
let lastRenderedFolderKey  = null;
let lastRenderedEntryId    = null;
let lastRenderedCommandKey = null;

let hackLevel        = 0;
let hack              = null;  // public hack state from server, or null
let hackSolvedTimer   = null;
let hackTyped         = '';
let hackHoverKey      = null;
let hackHoverText     = '';
let hackWasSolved     = false;
let lastAttemptsLeft  = null;

// ── DOM refs ──────────────────────────────────────────────
const normalHeader = document.getElementById('normalHeader');
const introTextEl  = document.getElementById('introTextEl');
const serverLine   = document.getElementById('serverLine');
const hackHeader   = document.getElementById('hackHeader');
const attemptsLine = document.getElementById('attemptsLine');

const termIdle   = document.getElementById('termIdle');
const termList   = document.getElementById('termList');
const termEntry  = document.getElementById('termEntry');
const entryTitle = document.getElementById('entryTitle');
const entryBody  = document.getElementById('entryBody');

const hackBoard        = document.getElementById('hackBoard');
const hackColumns      = document.getElementById('hackColumns');
const hackLog          = document.getElementById('hackLog');
const hackInputPreview = document.getElementById('hackInputPreview');
const hackBlocked      = document.getElementById('hackBlocked');

const termOutput  = document.getElementById('termOutput');
const termPrompt  = document.getElementById('termPrompt');
const backBtn     = document.getElementById('backBtn');
const connOverlay = document.getElementById('connOverlay');
const connText    = document.getElementById('connText');

// ════════════════════════════════════════════════════
// WEBSOCKET
// ════════════════════════════════════════════════════
let ws;
let reconnectTimer;

function connect() {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  ws = new WebSocket(`${proto}//${location.host}`);

  ws.addEventListener('open', () => {
    clearTimeout(reconnectTimer);
    connOverlay.classList.add('hidden');
  });

  ws.addEventListener('message', ev => {
    try { dispatch(JSON.parse(ev.data)); }
    catch (e) { console.warn('WS parse error', e); }
  });

  ws.addEventListener('close', () => {
    connOverlay.classList.remove('hidden');
    connText.textContent = 'СВЯЗЬ ПОТЕРЯНА — ПЕРЕПОДКЛЮЧЕНИЕ...';
    reconnectTimer = setTimeout(connect, 3000);
  });

  ws.addEventListener('error', () => ws.close());
}

function send(msg) {
  if (ws && ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify(msg));
}

// Applies a nav position pushed by the server. Never touches MODE.HACK —
// the hack gate always wins until it resolves.
function applyNavFromServer(nav) {
  navStack      = nav.path.slice();
  viewEntryId   = nav.viewEntryId || null;
  currentCommandNodeId = nav.commandNodeId || null;
  commandOutput = nav.commandNodeId ? ((findNodeById(tree, nav.commandNodeId) || {}).text ?? null) : null;
  selIndex      = 0;
  if (mode !== MODE.HACK) {
    mode = nav.mode === 'entry' ? MODE.ENTRY : MODE.LIST;
  }
}

function dispatch(msg) {
  if (msg.type === 'TERMINAL_LIVE') {
    hasLive       = true;
    terminalName  = msg.terminalName || '';
    introText     = msg.introText || '';
    tree          = msg.tree;
    hackLevel     = msg.hackLevel || 0;
    hack          = msg.hack || null;
    serverNum     = 1 + Math.floor(Math.random() * 9);
    hackTyped     = '';
    hackHoverKey  = null;
    hackHoverText = '';
    hackWasSolved = !!(hack && hack.solved);
    lastAttemptsLeft = hack ? hack.attemptsLeft : null;
    clearTimeout(hackSolvedTimer);
    hackSolvedTimer = null;

    const nav = msg.nav || { path: ['root'], mode: 'list', viewEntryId: null, commandNodeId: null };
    navStack      = nav.path.slice();
    viewEntryId   = nav.viewEntryId || null;
    currentCommandNodeId = nav.commandNodeId || null;
    commandOutput = nav.commandNodeId ? ((findNodeById(tree, nav.commandNodeId) || {}).text ?? null) : null;
    selIndex      = 0;
    lastRenderedFolderKey  = null;
    lastRenderedEntryId    = null;
    lastRenderedCommandKey = null;

    mode = (hackLevel > 0 && hack && !hack.solved) ? MODE.HACK : (nav.mode === 'entry' ? MODE.ENTRY : MODE.LIST);

    tryStartAmbient();
    render();
  } else if (msg.type === 'TERMINAL_UPDATE') {
    tree = msg.tree;
    if (msg.introText != null) introText = msg.introText;
    if (msg.nav) applyNavFromServer(msg.nav);
    render();
  } else if (msg.type === 'NAV_STATE') {
    applyNavFromServer(msg.nav);
    render();
  } else if (msg.type === 'HACK_STATE') {
    hack = msg.hack;
    if (mode !== MODE.HACK || !hack) return;

    if (!hack.solved && lastAttemptsLeft != null && hack.attemptsLeft < lastAttemptsLeft) {
      playHackBad();
    }
    lastAttemptsLeft = hack.attemptsLeft;

    if (hack.solved && !hackWasSolved) {
      hackWasSolved = true;
      playHackGood();
    }

    if (hack.solved && !hackSolvedTimer) {
      hackSolvedTimer = setTimeout(() => {
        hackSolvedTimer = null;
        mode = MODE.LIST;
        render();
      }, 2600);
    }
    render();
  } else if (msg.type === 'TERMINAL_CLEAR') {
    hasLive = false;
    tree = null;
    hack = null;
    clearTimeout(hackSolvedTimer);
    hackSolvedTimer = null;
    stopAmbient();
    render();
  }
}

// ════════════════════════════════════════════════════
// TREE HELPERS
// ════════════════════════════════════════════════════
function findNodeById(root, id) {
  if (!root) return null;
  if (root.id === id) return root;
  if (root.children) {
    for (const c of root.children) {
      const found = findNodeById(c, id);
      if (found) return found;
    }
  }
  return null;
}

function currentFolderNode() {
  let cur = tree;
  for (let i = 1; i < navStack.length; i++) {
    cur = cur.children.find(c => c.id === navStack[i]);
    if (!cur) { cur = tree; break; }
  }
  return cur;
}

// ════════════════════════════════════════════════════
// NORMAL-SCREEN NAVIGATION ACTIONS — sent to the server, the shared
// position only actually changes once NAV_STATE / TERMINAL_UPDATE echoes back.
// ════════════════════════════════════════════════════
function activateRow(node) {
  if (node.type === 'folder') {
    send({ type: 'NAV_ACTION', action: 'enter', nodeId: node.id });
  } else if (node.type === 'command') {
    send({ type: 'NAV_ACTION', action: 'command', nodeId: node.id });
  } else if (node.type === 'entry') {
    send({ type: 'NAV_ACTION', action: 'entry', nodeId: node.id });
  }
}

function goBack() {
  if (mode === MODE.HACK) return;
  send({ type: 'NAV_ACTION', action: 'back' });
}

backBtn.addEventListener('click', goBack);

// ════════════════════════════════════════════════════
// MENU HOVER: highlight the entry under the cursor + focus sound
// ════════════════════════════════════════════════════
let lastMenuHoverIdx = null;
termList.addEventListener('mouseover', (e) => {
  const row = e.target.closest('.term-row');
  if (!row || row.dataset.idx == null) return;
  if (row.dataset.idx === lastMenuHoverIdx) return;
  lastMenuHoverIdx = row.dataset.idx;
  playMenuFocus();
  selIndex = Number(row.dataset.idx);
  termList.querySelectorAll('.term-row.sel').forEach(el => el.classList.remove('sel'));
  row.classList.add('sel');
});
termList.addEventListener('mouseleave', () => { lastMenuHoverIdx = null; });

// ════════════════════════════════════════════════════
// HACKING MINIGAME — hover/click on the board
// ════════════════════════════════════════════════════
function cssEscape(s) {
  return (window.CSS && CSS.escape) ? CSS.escape(s) : String(s).replace(/[^a-zA-Z0-9_-]/g, '\\$&');
}

function setHackHover(key) {
  if (hackHoverKey === key) return;
  if (hackHoverKey != null) {
    hackColumns.querySelectorAll(`[data-target="${cssEscape(hackHoverKey)}"]`).forEach(el => el.classList.remove('hi'));
  }
  hackHoverKey = key;
  hackHoverText = '';
  if (key != null) {
    const els = hackColumns.querySelectorAll(`[data-target="${cssEscape(key)}"]`);
    els.forEach(el => el.classList.add('hi'));
    hackHoverText = Array.from(els).map(el => el.textContent).join('');
  }
  renderHackInputPreview();
}

hackColumns.addEventListener('mouseover', (e) => {
  const cell = e.target.closest('.hcell');
  if (!cell) return;
  const key = cell.dataset.target;
  if (key === hackHoverKey) return;
  setHackHover(key);
  if (cell.classList.contains('word')) playMultiple(); else playSingle();
});
hackColumns.addEventListener('mouseout', (e) => {
  const cell = e.target.closest('.hcell');
  if (!cell) return;
  const related = e.relatedTarget && e.relatedTarget.closest ? e.relatedTarget.closest('.hcell') : null;
  if (!related || related.dataset.target !== cell.dataset.target) setHackHover(null);
});
hackColumns.addEventListener('click', (e) => {
  const cell = e.target.closest('.hcell');
  if (!cell || !hack || hack.solved || hack.failed) return;
  playEnter();
  send({ type: 'HACK_GUESS', targetId: cell.dataset.target });
});

// ════════════════════════════════════════════════════
// KEYBOARD
// ════════════════════════════════════════════════════
document.addEventListener('keydown', (e) => {
  if (!hasLive) return;

  if (mode === MODE.HACK) {
    if (!hack || hack.solved || hack.failed) return;
    if (e.key === 'Enter') {
      const val = hackTyped.trim();
      hackTyped = '';
      if (val === '1') send({ type: 'HACK_ADMIN' });
      renderHackInputPreview();
      e.preventDefault();
    } else if (e.key === 'Backspace') {
      hackTyped = hackTyped.slice(0, -1);
      renderHackInputPreview();
      e.preventDefault();
    } else if (e.key === 'Escape') {
      hackTyped = '';
      renderHackInputPreview();
      e.preventDefault();
    } else if (e.key.length === 1 && hackTyped.length < 24) {
      hackTyped += e.key;
      renderHackInputPreview();
      e.preventDefault();
    }
    return;
  }

  if (mode === MODE.ENTRY) {
    if (e.key === 'Escape' || e.key === 'Backspace' || e.key === 'Enter') { goBack(); e.preventDefault(); }
    return;
  }

  const kids = (currentFolderNode().children || []);
  if (e.key === 'ArrowDown') {
    if (kids.length) {
      const next = Math.min(kids.length - 1, selIndex + 1);
      if (next !== selIndex) { selIndex = next; playMultiple(); }
    }
    render();
    e.preventDefault();
  } else if (e.key === 'ArrowUp') {
    if (kids.length) {
      const next = Math.max(0, selIndex - 1);
      if (next !== selIndex) { selIndex = next; playMultiple(); }
    }
    render();
    e.preventDefault();
  } else if (e.key === 'Enter') {
    if (kids[selIndex]) activateRow(kids[selIndex]);
    e.preventDefault();
  } else if (e.key === 'Escape' || e.key === 'Backspace') {
    goBack();
    e.preventDefault();
  }
});

// ════════════════════════════════════════════════════
// RENDER
// ════════════════════════════════════════════════════
function esc(s) {
  return String(s == null ? '' : s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
}

// ── Typewriter reveal: append elements one at a time (fast), with a
// charscroll sound per new line. `animate=false` just shows everything
// instantly — used when re-rendering content that hasn't actually changed.
const REVEAL_DELAY_MS = 40;

function revealInto(container, elements, animate) {
  if (container._revealTimer) {
    clearTimeout(container._revealTimer);
    container._revealTimer = null;
  }
  container.innerHTML = '';
  if (!animate) {
    elements.forEach(el => container.appendChild(el));
    return;
  }
  let i = 0;
  function next() {
    if (i >= elements.length) return;
    container.appendChild(elements[i]);
    playCharScroll();
    i++;
    if (i < elements.length) container._revealTimer = setTimeout(next, REVEAL_DELAY_MS);
  }
  next();
}

function lineToDiv(text) {
  const d = document.createElement('div');
  d.textContent = text;
  return d;
}

function render() {
  if (!hasLive) {
    normalHeader.style.display = 'none';
    hackHeader.style.display   = 'none';
    termIdle.style.display     = '';
    termList.style.display     = 'none';
    termEntry.style.display    = 'none';
    hackBoard.style.display    = 'none';
    hackBlocked.style.display  = 'none';
    termOutput.style.display   = 'none';
    termPrompt.style.display   = 'none';
    backBtn.style.display      = 'none';
    return;
  }

  termIdle.style.display = 'none';

  if (mode === MODE.HACK) {
    renderHackScreen();
  } else {
    renderNormalScreen();
  }
}

function renderNormalScreen() {
  hackHeader.style.display  = 'none';
  hackBoard.style.display   = 'none';
  hackBlocked.style.display = 'none';

  normalHeader.style.display = '';
  serverLine.textContent     = `-Server ${serverNum}-`;
  introTextEl.textContent    = introText;
  termPrompt.style.display   = '';

  if (mode === MODE.ENTRY) {
    const node = findNodeById(tree, viewEntryId);
    termList.style.display  = 'none';
    termEntry.style.display = '';
    termOutput.style.display = 'none';
    backBtn.style.display   = '';
    entryTitle.textContent  = node ? node.name : '';

    const isNewEntry = viewEntryId !== lastRenderedEntryId;
    lastRenderedEntryId = viewEntryId;
    lastRenderedFolderKey = null;
    lastRenderedCommandKey = null;

    const lines = (node ? (node.description || '') : '').split('\n').map(lineToDiv);
    revealInto(entryBody, lines, isNewEntry);
    return;
  }

  // MODE.LIST
  termEntry.style.display = 'none';
  termList.style.display  = '';
  backBtn.style.display   = navStack.length > 1 ? '' : 'none';
  lastRenderedEntryId = null;
  lastMenuHoverIdx = null;

  const folder = currentFolderNode();
  const kids = folder.children || [];

  const folderKey = navStack.join('/');
  const isNewFolder = folderKey !== lastRenderedFolderKey;
  lastRenderedFolderKey = folderKey;

  let rows;
  if (!kids.length) {
    const empty = document.createElement('div');
    empty.className = 'term-empty';
    empty.textContent = '[ ДИРЕКТОРИЯ ПУСТА ]';
    rows = [empty];
  } else {
    rows = kids.map((node, i) => {
      const row = document.createElement('div');
      row.className = 'term-row' + (i === selIndex ? ' sel' : '');
      row.dataset.idx = String(i);
      row.textContent = '> ' + node.name;
      row.addEventListener('click', () => {
        selIndex = i;
        activateRow(node);
      });
      return row;
    });
  }
  revealInto(termList, rows, isNewFolder);

  if (commandOutput !== null) {
    termOutput.style.display = '';
    const isNewCommand = currentCommandNodeId !== lastRenderedCommandKey;
    lastRenderedCommandKey = currentCommandNodeId;
    const outLines = String(commandOutput).split('\n').map(lineToDiv);
    revealInto(termOutput, outLines, isNewCommand);
  } else {
    termOutput.style.display = 'none';
    lastRenderedCommandKey = null;
  }
}

// ── Hacking screen ─────────────────────────────────────────
function pluralAttempts(n) {
  const mod10 = n % 10, mod100 = n % 100;
  if (mod10 === 1 && mod100 !== 11) return 'ПОПЫТКА';
  if ([2, 3, 4].includes(mod10) && ![12, 13, 14].includes(mod100)) return 'ПОПЫТКИ';
  return 'ПОПЫТОК';
}

function attemptsLineHtml(h) {
  const squares = Array.from({ length: h.attemptsMax }, (_, i) =>
    `<span class="atsq ${i < h.attemptsLeft ? 'full' : 'empty'}">■</span>`
  ).join(' ');
  return `${h.attemptsLeft} ${pluralAttempts(h.attemptsLeft)} ОСТАЛОСЬ: ${squares}`;
}

function renderHackScreen() {
  normalHeader.style.display = 'none';
  termList.style.display     = 'none';
  termEntry.style.display    = 'none';
  termOutput.style.display   = 'none';
  termPrompt.style.display   = 'none';
  backBtn.style.display      = 'none';
  hackHeader.style.display   = '';

  if (!hack) {
    hackBoard.style.display   = 'none';
    hackBlocked.style.display = 'none';
    return;
  }

  attemptsLine.innerHTML = attemptsLineHtml(hack);

  if (hack.failed) {
    hackBoard.style.display   = 'none';
    hackBlocked.style.display = '';
    return;
  }

  hackBlocked.style.display = 'none';
  hackBoard.style.display   = '';
  renderHackColumns();
  renderHackLog();
  renderHackInputPreview();
}

function buildColumnHtml(col, colIndex) {
  const wordAt = new Array(col.text.length).fill(null);
  col.words.forEach(w => { for (let i = w.start; i < w.start + w.length; i++) wordAt[i] = w.id; });

  const rows = Math.ceil(col.text.length / ROW_WIDTH);
  let rowsHtml = '';
  for (let r = 0; r < rows; r++) {
    const rowStart = r * ROW_WIDTH;
    const rowEnd = Math.min(rowStart + ROW_WIDTH, col.text.length);
    let cellsHtml = '';
    let i = rowStart;
    while (i < rowEnd) {
      const wid = wordAt[i];
      if (wid) {
        let j = i;
        while (j < rowEnd && wordAt[j] === wid) j++;
        cellsHtml += `<span class="hcell word" data-target="${esc(wid)}">${esc(col.text.slice(i, j))}</span>`;
        i = j;
      } else {
        cellsHtml += `<span class="hcell filler" data-target="${colIndex}:${i}">${esc(col.text[i])}</span>`;
        i++;
      }
    }
    const addr = col.addresses[r] || '';
    rowsHtml += `<div class="hack-row"><span class="hack-addr">${esc(addr)}</span><span class="hack-cells">${cellsHtml}</span></div>`;
  }
  return `<div class="hack-col">${rowsHtml}</div>`;
}

function renderHackColumns() {
  hackHoverKey = null;
  hackHoverText = '';
  hackColumns.innerHTML = hack.columns.map((col, ci) => buildColumnHtml(col, ci)).join('');
}

function renderHackLog() {
  hackLog.innerHTML = hack.log.map(line => `<div>${esc(line)}</div>`).join('');
  hackLog.scrollTop = hackLog.scrollHeight;
}

function renderHackInputPreview() {
  hackInputPreview.textContent = hackTyped.length ? hackTyped : hackHoverText;
}

// ════════════════════════════════════════════════════
// BOOT
// ════════════════════════════════════════════════════
render();
connect();
