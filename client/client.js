'use strict';

const MODE = { LIST: 'list', ENTRY: 'entry', HACK: 'hack' };
const ROW_WIDTH = 12; // must match server/hack.js
const HACK_DELIMITERS = '()[]{}<>';

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

let pagedView = {
  kind: null,
  key: null,
  text: '',
  container: null,
  pages: [''],
  index: 0,
};
let paginationFrame = null;
let hackFitFrame = null;

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
const termBody   = document.getElementById('termBody');
const termList   = document.getElementById('termList');
const termEntry  = document.getElementById('termEntry');
const entryTitle = document.getElementById('entryTitle');
const entryBody  = document.getElementById('entryBody');

const hackBoard        = document.getElementById('hackBoard');
const hackColumns      = document.getElementById('hackColumns');
const hackLog          = document.getElementById('hackLog');
const hackInputPreview = document.getElementById('hackInputPreview');
const hackBlocked      = document.getElementById('hackBlocked');
const hackLogPanel     = hackLog.closest('.hack-log-panel');
const hackInputLine    = hackInputPreview.closest('.hack-input-line');
const screen           = document.getElementById('screen');

const termOutput  = document.getElementById('termOutput');
const termPrompt  = document.getElementById('termPrompt');
const backBtn     = document.getElementById('backBtn');
const pageNav     = document.getElementById('pageNav');
const pagePrev    = document.getElementById('pagePrev');
const pageNext    = document.getElementById('pageNext');
const pageIndicator = document.getElementById('pageIndicator');
const connOverlay = document.getElementById('connOverlay');
const connText    = document.getElementById('connText');

// ════════════════════════════════════════════════════
// WEBSOCKET
// ════════════════════════════════════════════════════
let ws;
let reconnectTimer = null;
const RECONNECT_DELAY_MS = 3000;

function playerWebSocketURL() {
  const url = new URL('/', window.location.href);
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
  return url.href;
}

function connect() {
  clearTimeout(reconnectTimer);
  reconnectTimer = null;

  const socket = new WebSocket(playerWebSocketURL());
  ws = socket;

  socket.addEventListener('open', () => {
    if (ws !== socket) return;
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
    connOverlay.classList.add('hidden');
  });

  socket.addEventListener('message', ev => {
    if (ws !== socket) return;
    try { dispatch(JSON.parse(ev.data)); }
    catch (e) { console.warn('WS parse error', e); }
  });

  socket.addEventListener('close', () => {
    if (ws !== socket) return;
    connOverlay.classList.remove('hidden');
    connText.textContent = 'СВЯЗЬ ПОТЕРЯНА — ПЕРЕПОДКЛЮЧЕНИЕ...';
    reconnectTimer = setTimeout(connect, RECONNECT_DELAY_MS);
  });

  socket.addEventListener('error', () => socket.close());
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
pagePrev.addEventListener('click', () => changePage(-1));
pageNext.addEventListener('click', () => changePage(1));

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
  hackColumns.querySelectorAll('.hcell.hi').forEach(el => el.classList.remove('hi'));
  hackHoverKey = key;
  hackHoverText = '';
  if (key != null) {
    const els = hackColumns.querySelectorAll(`[data-target="${cssEscape(key)}"]`);
    els.forEach(el => el.classList.add('hi'));
    hackHoverText = Array.from(els).map(el => el.textContent).join('');
  }
  renderHackInputPreview();
}

function patternAtCell(cell) {
  if (!hack || !cell || cell.dataset.row == null || cell.dataset.offset == null) return null;
  const row = Number(cell.dataset.row);
  const offset = Number(cell.dataset.offset);
  const matches = (hack.patterns || []).filter(pattern =>
    pattern.row === row && offset >= pattern.start && offset <= pattern.end
  );
  return matches.find(pattern => pattern.start === offset) ||
    matches.reduce((nearest, pattern) => !nearest || pattern.start > nearest.start ? pattern : nearest, null);
}

function isDelimiterCell(cell) {
  return !!cell && cell.classList.contains('filler') &&
    cell.textContent.length === 1 && HACK_DELIMITERS.includes(cell.textContent);
}

function setHackPatternHover(pattern) {
  hackColumns.querySelectorAll('.hcell.hi').forEach(el => el.classList.remove('hi'));
  hackHoverKey = pattern ? pattern.id : null;
  hackHoverText = '';
  if (pattern && !pattern.used) {
    const cells = hackColumns.querySelectorAll(`[data-row="${pattern.row}"][data-offset]`);
    cells.forEach(cell => {
      const offset = Number(cell.dataset.offset);
      if (offset >= pattern.start && offset <= pattern.end) cell.classList.add('hi');
    });
    hackHoverText = Array.from(cells)
      .filter(cell => {
        const offset = Number(cell.dataset.offset);
        return offset >= pattern.start && offset <= pattern.end;
      })
      .map(cell => cell.textContent)
      .join('');
  }
  renderHackInputPreview();
}

function previewHackCell(cell) {
  if (!cell) return;
  const pattern = patternAtCell(cell);
  if (pattern) {
    if (pattern.used) setHackPatternHover(null);
    else setHackPatternHover(pattern);
    if (!pattern.used) playMultiple();
    return;
  }
  if (isDelimiterCell(cell)) {
    setHackPatternHover(null);
    return;
  }
  const key = cell.dataset.target;
  if (key === hackHoverKey) return;
  setHackHover(key);
  if (cell.classList.contains('word')) playMultiple(); else playSingle();
}

hackColumns.addEventListener('mouseover', (e) => {
  previewHackCell(e.target.closest('.hcell'));
});
hackColumns.addEventListener('mouseout', (e) => {
  const cell = e.target.closest('.hcell');
  if (!cell) return;
  if (patternAtCell(cell)) {
    setHackPatternHover(null);
    return;
  }
  const related = e.relatedTarget && e.relatedTarget.closest ? e.relatedTarget.closest('.hcell') : null;
  if (!related || related.dataset.target !== cell.dataset.target) setHackHover(null);
});
hackColumns.addEventListener('focusin', (e) => {
  previewHackCell(e.target.closest('.hcell'));
});
hackColumns.addEventListener('focusout', (e) => {
  if (e.target.closest('.hcell')) setHackPatternHover(null);
});
hackColumns.addEventListener('click', (e) => {
  const cell = e.target.closest('.hcell');
  if (!cell || !hack || hack.solved || hack.failed) return;
  const pattern = patternAtCell(cell);
  if (pattern && !pattern.used) {
    playEnter();
    send({ type: 'HACK_PATTERN', patternId: pattern.id });
  } else if (pattern || isDelimiterCell(cell)) {
    return;
  } else {
    playEnter();
    send({ type: 'HACK_GUESS', targetId: cell.dataset.target });
  }
});

// ════════════════════════════════════════════════════
// KEYBOARD
// ════════════════════════════════════════════════════
document.addEventListener('keydown', (e) => {
  if (!hasLive) return;

  if (mode === MODE.HACK) {
    if (!hack || hack.solved || hack.failed) return;
    if (e.key === 'Enter') {
      hackTyped = '';
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
    if (e.key === 'ArrowLeft' || e.key === 'PageUp') {
      changePage(-1);
      e.preventDefault();
    } else if (e.key === 'ArrowRight' || e.key === 'PageDown') {
      changePage(1);
      e.preventDefault();
    } else if (e.key === 'Home') {
      changePage(-pagedView.index);
      e.preventDefault();
    } else if (e.key === 'End') {
      changePage(pagedView.pages.length - pagedView.index - 1);
      e.preventDefault();
    } else if (e.key === 'Escape' || e.key === 'Backspace') {
      goBack();
      e.preventDefault();
    }
    return;
  }

  if (pagedView.kind === 'command') {
    if (e.key === 'ArrowLeft' || e.key === 'PageUp') {
      changePage(-1);
      e.preventDefault();
      return;
    }
    if (e.key === 'ArrowRight' || e.key === 'PageDown') {
      changePage(1);
      e.preventDefault();
      return;
    }
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

function replaceWithText(container, text) {
  if (container._revealTimer) {
    clearTimeout(container._revealTimer);
    container._revealTimer = null;
  }
  container.innerHTML = '';
  String(text).split('\n').forEach(line => container.appendChild(lineToDiv(line)));
}

function textFits(container, text) {
  replaceWithText(container, text);
  return container.scrollHeight <= container.clientHeight + 1 &&
    container.scrollWidth <= container.clientWidth + 1;
}

function naturalPageBreak(text, start, fittedEnd) {
  if (fittedEnd >= text.length) return text.length;

  const minimumBreak = start + Math.floor((fittedEnd - start) * .6);
  for (let i = fittedEnd; i > minimumBreak; i--) {
    if (/\s/.test(text[i - 1])) return i;
  }
  return fittedEnd;
}

function paginateText(container, text) {
  const source = String(text == null ? '' : text).replace(/\r\n?/g, '\n');
  if (!source) return [''];
  if (container.clientHeight <= 0 || container.clientWidth <= 0) return [source];

  const pages = [];
  let start = 0;
  while (start < source.length) {
    let low = start + 1;
    let high = source.length;
    let fittedEnd = start;

    while (low <= high) {
      const midpoint = Math.floor((low + high) / 2);
      if (textFits(container, source.slice(start, midpoint))) {
        fittedEnd = midpoint;
        low = midpoint + 1;
      } else {
        high = midpoint - 1;
      }
    }

    if (fittedEnd === start) fittedEnd = start + 1;
    const end = naturalPageBreak(source, start, fittedEnd);
    pages.push(source.slice(start, end));
    start = end;
  }
  return pages;
}

function updatePageControls() {
  if (pagedView.kind === null) {
    pageNav.hidden = true;
    return;
  }

  pageNav.hidden = false;
  pagePrev.hidden = pagedView.index === 0;
  pageNext.hidden = pagedView.index >= pagedView.pages.length - 1;
  pageIndicator.value = `${pagedView.index + 1} / ${pagedView.pages.length}`;
  pageIndicator.textContent = pageIndicator.value;
}

function renderPagedView(animate) {
  if (!pagedView.container) return;
  const page = pagedView.pages[pagedView.index] || '';
  const lines = page.split('\n').map(lineToDiv);
  revealInto(pagedView.container, lines, animate);
  updatePageControls();
}

function recalculatePagination(resetPage, animate) {
  if (pagedView.kind === null || !pagedView.container) return;
  const previousIndex = resetPage ? 0 : pagedView.index;
  pagedView.pages = paginateText(pagedView.container, pagedView.text);
  pagedView.index = Math.min(previousIndex, pagedView.pages.length - 1);
  renderPagedView(animate && pagedView.index === 0);
}

function activatePagination(kind, key, text, container, animate) {
  const source = String(text == null ? '' : text);
  const identityChanged = pagedView.kind !== kind || pagedView.key !== key;
  const contentChanged = pagedView.text !== source || pagedView.container !== container;

  pagedView.kind = kind;
  pagedView.key = key;
  pagedView.text = source;
  pagedView.container = container;

  if (identityChanged || contentChanged) {
    recalculatePagination(identityChanged, animate);
  } else {
    renderPagedView(false);
  }
}

function deactivatePagination() {
  if (paginationFrame !== null) {
    cancelAnimationFrame(paginationFrame);
    paginationFrame = null;
  }
  pagedView = {
    kind: null,
    key: null,
    text: '',
    container: null,
    pages: [''],
    index: 0,
  };
  updatePageControls();
}

function changePage(delta) {
  if (pagedView.kind === null || !delta) return;
  const activeControl = document.activeElement;
  const nextIndex = Math.max(0, Math.min(pagedView.pages.length - 1, pagedView.index + delta));
  if (nextIndex === pagedView.index) return;

  pagedView.index = nextIndex;
  playEnter();
  renderPagedView(false);

  if (activeControl === pagePrev && pagePrev.hidden) {
    (pageNext.hidden ? backBtn : pageNext).focus();
  } else if (activeControl === pageNext && pageNext.hidden) {
    (pagePrev.hidden ? backBtn : pagePrev).focus();
  }
}

function scheduleRepagination() {
  if (pagedView.kind === null) return;
  if (paginationFrame !== null) cancelAnimationFrame(paginationFrame);
  paginationFrame = requestAnimationFrame(() => {
    paginationFrame = null;
    recalculatePagination(false, false);
  });
}

function regionOverflows(region) {
  return region.scrollHeight > region.clientHeight + 1 ||
    region.scrollWidth > region.clientWidth + 1;
}

function regionContains(parent, child) {
  const tolerance = 1;
  const parentBounds = parent.getBoundingClientRect();
  const childBounds = child.getBoundingClientRect();
  return childBounds.top >= parentBounds.top - tolerance &&
    childBounds.left >= parentBounds.left - tolerance &&
    childBounds.right <= parentBounds.right + tolerance &&
    childBounds.bottom <= parentBounds.bottom + tolerance;
}

function hackContentOverflows() {
  const columns = Array.from(hackColumns.children);
  const rows = Array.from(hackColumns.querySelectorAll('.hack-row'));
  const logLines = Array.from(hackLog.children);
  const regions = [hackBoard, hackColumns, hackLogPanel, hackLog, hackInputLine, ...columns, ...rows, ...logLines];
  if (regions.some(regionOverflows)) return true;

  const containedRegions = [
    [screen, hackHeader],
    [screen, hackBoard],
    [hackBoard, hackColumns],
    [hackBoard, hackLogPanel],
    [hackLogPanel, hackLog],
    [hackLogPanel, hackInputLine],
    ...columns.map(column => [hackColumns, column]),
    ...rows.map(row => [hackColumns, row]),
    ...logLines.map(line => [hackLog, line]),
  ];
  return containedRegions.some(([parent, child]) => !regionContains(parent, child));
}

function hackRowsFitColumns() {
  const tolerance = 0.5;
  return Array.from(hackColumns.children).every(column => {
    const columnBounds = column.getBoundingClientRect();
    return Array.from(column.querySelectorAll('.hack-row')).every(row => {
      const addressBounds = row.querySelector('.hack-addr').getBoundingClientRect();
      const cells = row.querySelectorAll('.hcell');
      const finalBounds = cells.length
        ? cells[cells.length - 1].getBoundingClientRect()
        : row.querySelector('.hack-cells').getBoundingClientRect();
      const rowBounds = row.getBoundingClientRect();
      return addressBounds.left >= columnBounds.left - tolerance &&
        finalBounds.right <= columnBounds.right + tolerance &&
        rowBounds.top >= columnBounds.top - tolerance &&
        rowBounds.bottom <= columnBounds.bottom + tolerance;
    });
  });
}

function fitHackRowFont() {
  hackBoard.style.removeProperty('--hack-row-font');
  const rows = Array.from(hackColumns.querySelectorAll('.hack-row'));
  const columns = Array.from(hackColumns.children);
  if (!rows.length || !columns.length) return;

  const baseSize = Number.parseFloat(getComputedStyle(hackBoard).fontSize);
  if (!Number.isFinite(baseSize) || baseSize <= 0) return;

  const applySize = size => hackBoard.style.setProperty('--hack-row-font', `${size}px`);
  const fitsAt = size => {
    applySize(size);
    return hackRowsFitColumns() && !hackContentOverflows();
  };

  if (!fitsAt(baseSize)) {
    applySize(baseSize);
    return;
  }

  let low = baseSize;
  const narrowerColumnWidth = Math.min(...columns.map(column => column.getBoundingClientRect().width));
  let high = Math.max(baseSize * 2, narrowerColumnWidth);
  for (let attempt = 0; attempt < 8 && fitsAt(high); attempt++) {
    low = high;
    high *= 2;
  }

  while (high - low > 0.25) {
    const candidate = (low + high) / 2;
    if (fitsAt(candidate)) low = candidate;
    else high = candidate;
  }
  applySize(low);
}

function fitHackLayout() {
  hackFitFrame = null;
  if (mode !== MODE.HACK || hackBoard.hidden) {
    hackBoard.style.removeProperty('--hack-row-font');
    hackBoard.classList.remove('hack-compact', 'hack-stacked', 'hack-tight');
    return;
  }

  hackBoard.style.removeProperty('--hack-row-font');
  hackBoard.classList.remove('hack-compact', 'hack-stacked', 'hack-tight');
  const preferStacked = hackBoard.clientWidth <= 700 || hackBoard.clientHeight <= 300;
  hackBoard.classList.toggle('hack-stacked', preferStacked);
  hackBoard.classList.toggle('hack-compact', preferStacked || hackContentOverflows());

  if (!preferStacked && hackContentOverflows()) {
    hackBoard.classList.add('hack-compact', 'hack-stacked');
  }
  if (hackContentOverflows()) {
    hackBoard.classList.add('hack-tight');
  }
  fitHackRowFont();
}

function scheduleHackFit() {
  if (hackFitFrame !== null) cancelAnimationFrame(hackFitFrame);
  hackFitFrame = requestAnimationFrame(fitHackLayout);
}

function render() {
  if (!hasLive) {
    deactivatePagination();
    normalHeader.hidden = true;
    hackHeader.hidden   = true;
    termIdle.hidden     = false;
    termList.hidden     = true;
    termEntry.hidden    = true;
    hackBoard.hidden    = true;
    hackBlocked.hidden  = true;
    termOutput.hidden   = true;
    termPrompt.hidden   = true;
    backBtn.hidden      = true;
    return;
  }

  termIdle.hidden = true;

  if (mode === MODE.HACK) {
    renderHackScreen();
  } else {
    renderNormalScreen();
  }
}

function renderNormalScreen() {
  hackHeader.hidden  = true;
  hackBoard.hidden   = true;
  hackBlocked.hidden = true;

  normalHeader.hidden         = false;
  serverLine.textContent     = `-Server ${serverNum}-`;
  introTextEl.textContent    = introText;
  termPrompt.hidden          = false;

  if (mode === MODE.ENTRY) {
    const node = findNodeById(tree, viewEntryId);
    termList.hidden   = true;
    termEntry.hidden  = false;
    termOutput.hidden = true;
    backBtn.hidden    = false;
    entryTitle.textContent  = node ? node.name : '';

    const isNewEntry = viewEntryId !== lastRenderedEntryId;
    lastRenderedEntryId = viewEntryId;
    lastRenderedFolderKey = null;
    lastRenderedCommandKey = null;

    activatePagination('entry', viewEntryId, node ? (node.description || '') : '', entryBody, isNewEntry);
    return;
  }

  // MODE.LIST
  termEntry.hidden = true;
  termList.hidden  = false;
  backBtn.hidden   = navStack.length <= 1;
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
    termOutput.hidden = false;
    const isNewCommand = currentCommandNodeId !== lastRenderedCommandKey;
    lastRenderedCommandKey = currentCommandNodeId;
    activatePagination('command', currentCommandNodeId, commandOutput, termOutput, isNewCommand);
  } else {
    termOutput.hidden = true;
    lastRenderedCommandKey = null;
    deactivatePagination();
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
  deactivatePagination();
  normalHeader.hidden = true;
  termList.hidden     = true;
  termEntry.hidden    = true;
  termOutput.hidden   = true;
  termPrompt.hidden   = true;
  backBtn.hidden      = true;
  hackHeader.hidden   = false;

  if (!hack) {
    hackBoard.hidden   = true;
    hackBlocked.hidden = true;
    return;
  }

  attemptsLine.innerHTML = attemptsLineHtml(hack);

  if (hack.failed) {
    hackBoard.hidden   = true;
    hackBlocked.hidden = false;
    return;
  }

  hackBlocked.hidden = true;
  hackBoard.hidden   = false;
  renderHackColumns();
  renderHackLog();
  renderHackInputPreview();
  scheduleHackFit();
}

function buildColumnHtml(col, colIndex, rowBase) {
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
        cellsHtml += `<span class="hcell word" data-target="${esc(wid)}" tabindex="0">${esc(col.text.slice(i, j))}</span>`;
        i = j;
      } else {
        cellsHtml += `<span class="hcell filler" data-target="${colIndex}:${i}" data-row="${rowBase + r}" data-offset="${i - rowStart}" tabindex="0">${esc(col.text[i])}</span>`;
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
  let rowBase = 0;
  hackColumns.innerHTML = hack.columns.map((col, ci) => {
    const html = buildColumnHtml(col, ci, rowBase);
    rowBase += Math.ceil(col.text.length / ROW_WIDTH);
    return html;
  }).join('');
}

function renderHackLog() {
  hackLog.innerHTML = hack.log.map(line => `<div>${esc(line)}</div>`).join('');
}

function renderHackInputPreview() {
  hackInputPreview.textContent = hackTyped.length ? hackTyped : hackHoverText;
}

// ════════════════════════════════════════════════════
// BOOT
// ════════════════════════════════════════════════════
window.addEventListener('resize', scheduleRepagination);
window.addEventListener('resize', scheduleHackFit);
if ('ResizeObserver' in window) {
  const paginationObserver = new ResizeObserver(scheduleRepagination);
  paginationObserver.observe(termBody);
  const hackFitObserver = new ResizeObserver(scheduleHackFit);
  hackFitObserver.observe(termBody);
}
if (document.fonts && document.fonts.ready) {
  document.fonts.ready.then(scheduleRepagination);
  document.fonts.ready.then(scheduleHackFit);
}
render();
connect();
