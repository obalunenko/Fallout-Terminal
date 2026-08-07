'use strict';

const { ipcRenderer } = require('electron');

// ── State ─────────────────────────────────────────────────
const state = {
  session:        null,   // { version, name, terminals: [{id,name,hackLevel,introText,root}] }
  filePath:       null,
  liveTerminalId: null,
  editTerminalId: null,
  selectedNodeId: null,
  expanded:       new Set(['root']),
  liveHack:       null,   // last known hack-state of the live terminal, or null
};

let idCounter = 0;
function uid(prefix) {
  idCounter++;
  return `${prefix}_${Date.now().toString(36)}_${idCounter}`;
}

// ── DOM refs ──────────────────────────────────────────────
const startScreen      = document.getElementById('startScreen');
const startStatus      = document.getElementById('startStatus');
const mainLayout        = document.getElementById('mainLayout');
const sessionFileLabel  = document.getElementById('sessionFileLabel');
const serverUrlEl       = document.getElementById('serverUrl');
const clientCountEl     = document.getElementById('clientCount');
const termList          = document.getElementById('termList');
const saveStatus        = document.getElementById('saveStatus');
const editingTermName   = document.getElementById('editingTermName');
const liveFlag          = document.getElementById('liveFlag');
const btnMakeLive       = document.getElementById('btnMakeLive');
const btnPublish        = document.getElementById('btnPublish');
const treeView          = document.getElementById('treeView');
const nodeForm          = document.getElementById('nodeForm');
const toolbarHint       = document.getElementById('toolbarHint');
const btnAddFolder      = document.getElementById('btnAddFolder');
const btnAddCommand     = document.getElementById('btnAddCommand');
const btnAddEntry       = document.getElementById('btnAddEntry');
const btnAddTerminal    = document.getElementById('btnAddTerminal');
const btnStopBroadcast  = document.getElementById('btnStopBroadcast');
const hackStatus        = document.getElementById('hackStatus');
const hackStatusLine    = document.getElementById('hackStatusLine');
const btnHackSuccess    = document.getElementById('btnHackSuccess');
const hackLevelSelect   = document.getElementById('hackLevelSelect');
const introTextArea     = document.getElementById('introTextArea');
const btnApplySettings  = document.getElementById('btnApplySettings');

let serverUrl = null;

// ── Server info / connection count ─────────────────────────
ipcRenderer.on('server-info', (_e, info) => {
  serverUrl = info.url;
  serverUrlEl.textContent = info.url;
});
ipcRenderer.on('client-count', (_e, count) => {
  clientCountEl.textContent = count;
});
ipcRenderer.on('hack-state', (_e, hack) => {
  state.liveHack = hack;
  renderHackStatus();
});
serverUrlEl.addEventListener('click', () => {
  if (serverUrl) ipcRenderer.send('open-url', serverUrl);
});

// ── Start screen: open / new session ───────────────────────
document.getElementById('btnOpenSession').addEventListener('click', async () => {
  const res = await ipcRenderer.invoke('session:open');
  if (!res.ok) {
    if (res.error) startStatus.textContent = 'Ошибка: ' + res.error;
    return;
  }
  loadSession(res.session, res.filePath);
});

document.getElementById('btnNewSession').addEventListener('click', async () => {
  const res = await ipcRenderer.invoke('session:new');
  if (!res.ok) {
    if (res.error) startStatus.textContent = 'Ошибка: ' + res.error;
    return;
  }
  loadSession(res.session, res.filePath);
});

function loadSession(session, filePath) {
  state.session        = session;
  state.filePath        = filePath;
  state.liveTerminalId  = null;
  state.editTerminalId  = (session.terminals[0] && session.terminals[0].id) || null;
  state.selectedNodeId  = null;
  state.expanded         = new Set(['root']);
  state.liveHack         = null;

  sessionFileLabel.textContent = filePath;
  startScreen.style.display = 'none';
  mainLayout.style.display  = 'flex';
  renderAll();
}

// ── Autosave (writes to the currently open session file) ──
async function autosave() {
  if (!state.filePath) return;
  const res = await ipcRenderer.invoke('session:save', state.session);
  if (res.ok) {
    saveStatus.textContent = 'Сохранено ' + new Date().toLocaleTimeString();
    saveStatus.classList.remove('err');
  } else {
    saveStatus.textContent = 'Ошибка сохранения: ' + (res.error || '');
    saveStatus.classList.add('err');
  }
}

// ── Helpers ─────────────────────────────────────────────────
function escHtml(s) {
  return String(s == null ? '' : s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
}
function escAttr(s) {
  return escHtml(s).replace(/"/g, '&quot;');
}

function getEditTerminal() {
  if (!state.session) return null;
  return state.session.terminals.find(t => t.id === state.editTerminalId) || null;
}

function locateNode(root, id) {
  function walk(node, parent) {
    if (node.id === id) return { node, parent };
    if (node.children) {
      for (const child of node.children) {
        const res = walk(child, node);
        if (res) return res;
      }
    }
    return null;
  }
  return walk(root, null);
}

function currentAddTarget() {
  const term = getEditTerminal();
  if (!term) return null;
  if (!state.selectedNodeId) return term.root;
  const loc = locateNode(term.root, state.selectedNodeId);
  if (!loc) return term.root;
  return loc.node.type === 'folder' ? loc.node : (loc.parent || term.root);
}

// ── Render: everything ──────────────────────────────────────
function renderAll() {
  renderTermList();
  renderTreeHeader();
  renderSettingsPanel();
  renderToolbarState();
  renderTree();
  renderNodeForm();
  renderToolbarHint();
  renderHackStatus();
}

// ── Render: terminal list ────────────────────────────────────
function renderTermList() {
  termList.innerHTML = '';
  if (!state.session.terminals.length) {
    termList.innerHTML = '<div class="tree-empty-hint">Нет терминалов</div>';
    return;
  }

  state.session.terminals.forEach(term => {
    const row = document.createElement('div');
    row.className = 'term-row'
      + (term.id === state.editTerminalId ? ' editing' : '')
      + (term.id === state.liveTerminalId ? ' is-live' : '');

    const nameRow = document.createElement('div');
    nameRow.className = 'term-row-name';
    nameRow.textContent = term.name;
    row.appendChild(nameRow);

    const metaRow = document.createElement('div');
    metaRow.className = 'term-row-meta';
    metaRow.textContent = '● В ЭФИРЕ';
    row.appendChild(metaRow);

    const actions = document.createElement('div');
    actions.style.display = 'flex';
    actions.style.gap = '6px';
    actions.style.marginTop = '2px';

    const renameBtn = document.createElement('button');
    renameBtn.className = 'btn btn-mini';
    renameBtn.textContent = 'ПЕРЕИМ.';
    renameBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      startRenameTerminal(term, nameRow);
    });
    actions.appendChild(renameBtn);

    const delBtn = document.createElement('button');
    delBtn.className = 'btn btn-mini btn-danger';
    delBtn.textContent = 'УДАЛИТЬ';
    delBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      if (!window.confirm(`Удалить терминал "${term.name}" целиком?`)) return;
      const idx = state.session.terminals.findIndex(t => t.id === term.id);
      if (idx >= 0) state.session.terminals.splice(idx, 1);
      if (state.liveTerminalId === term.id) {
        ipcRenderer.send('terminal:clear-live');
        state.liveTerminalId = null;
      }
      if (state.editTerminalId === term.id) {
        state.editTerminalId = (state.session.terminals[0] && state.session.terminals[0].id) || null;
        state.selectedNodeId = null;
        state.expanded = new Set(['root']);
      }
      autosave();
      renderAll();
    });
    actions.appendChild(delBtn);

    row.appendChild(actions);

    row.addEventListener('click', () => {
      if (state.editTerminalId === term.id) return;
      state.editTerminalId = term.id;
      state.selectedNodeId = null;
      state.expanded = new Set(['root']);
      renderAll();
    });

    termList.appendChild(row);
  });
}

function startRenameTerminal(term, nameRow) {
  const input = document.createElement('input');
  input.className = 'field-input';
  input.value = term.name;
  nameRow.replaceWith(input);
  input.focus();
  input.select();

  const commit = () => {
    const val = input.value.trim();
    if (val) {
      term.name = val;
      // Renaming does not re-broadcast: doing so via terminal:set-live would
      // regenerate an in-progress hack board. The new name reaches players
      // next time the terminal is (re)made live.
      autosave();
    }
    renderAll();
  };
  input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') input.blur();
    if (e.key === 'Escape') renderAll();
  });
  input.addEventListener('blur', commit);
}

// ── Render: tree header + toolbar state ──────────────────────
function renderTreeHeader() {
  const term = getEditTerminal();
  editingTermName.textContent = term ? term.name : '—';
  const isLive = !!term && term.id === state.liveTerminalId;
  liveFlag.style.display   = isLive ? '' : 'none';
  btnMakeLive.textContent  = isLive ? 'ПЕРЕЗАПУСТИТЬ ТРАНСЛЯЦИЮ' : 'СДЕЛАТЬ АКТИВНЫМ';
  btnMakeLive.disabled     = !term;
  btnPublish.style.display = isLive ? '' : 'none';
}

// ── Render: per-terminal settings (hack level / intro text) ──
function renderSettingsPanel() {
  const term = getEditTerminal();
  hackLevelSelect.disabled = !term;
  introTextArea.disabled   = !term;
  btnApplySettings.disabled = !term;
  hackLevelSelect.value = term ? String(term.hackLevel || 0) : '0';
  introTextArea.value   = term ? (term.introText || '') : '';
}

// ── Render: live hack status (term panel footer) ──────────────
function renderHackStatus() {
  const liveTerm = state.session && state.liveTerminalId
    ? state.session.terminals.find(t => t.id === state.liveTerminalId)
    : null;

  if (!liveTerm || !liveTerm.hackLevel || !state.liveHack) {
    hackStatus.style.display = 'none';
    return;
  }

  hackStatus.style.display = '';
  const h = state.liveHack;
  if (h.solved) {
    hackStatusLine.textContent = 'ВЗЛОМ: ПРОЙДЕН';
  } else if (h.failed) {
    hackStatusLine.textContent = 'ВЗЛОМ: ЗАБЛОКИРОВАН (нужен перезапуск трансляции)';
  } else {
    hackStatusLine.textContent = `ВЗЛОМ: осталось попыток ${h.attemptsLeft}/${h.attemptsMax}`;
  }
  btnHackSuccess.disabled = h.solved || h.failed;
}

function renderToolbarState() {
  const term = getEditTerminal();
  const disabled = !term;
  btnAddFolder.disabled  = disabled;
  btnAddCommand.disabled = disabled;
  btnAddEntry.disabled   = disabled;
}

function renderToolbarHint() {
  const target = currentAddTarget();
  toolbarHint.textContent = target ? `Добавление в: ${target.id === 'root' ? 'ROOT' : target.name}` : '';
}

// ── Render: tree view ─────────────────────────────────────────
function renderTree() {
  const term = getEditTerminal();
  treeView.innerHTML = '';
  if (!term) {
    treeView.innerHTML = '<div class="tree-empty-hint">Нет терминала — создайте его слева</div>';
    return;
  }
  treeView.appendChild(renderNode(term.root, true));
}

function renderNode(node, isRoot) {
  const wrap = document.createElement('div');
  wrap.className = 'tree-node';

  const row = document.createElement('div');
  row.className = 'tree-row' + (state.selectedNodeId === node.id ? ' selected' : '');

  const hasChildren = node.type === 'folder' && node.children && node.children.length > 0;
  const isExpanded   = state.expanded.has(node.id);

  const caret = document.createElement('span');
  caret.className = 'tree-caret';
  if (node.type === 'folder') {
    caret.textContent = hasChildren ? (isExpanded ? '▾' : '▸') : '·';
    caret.addEventListener('click', (e) => {
      e.stopPropagation();
      if (!hasChildren) return;
      if (isExpanded) state.expanded.delete(node.id); else state.expanded.add(node.id);
      renderTree();
    });
  }
  row.appendChild(caret);

  const icon = document.createElement('span');
  icon.className = 'tree-icon ' + node.type;
  icon.textContent = node.type === 'folder' ? '[ПАПКА]' : node.type === 'command' ? '[КОМАНДА]' : '[ЗАПИСЬ]';
  row.appendChild(icon);

  const label = document.createElement('span');
  label.className = 'tree-label';
  label.textContent = isRoot ? 'ROOT' : node.name;
  row.appendChild(label);

  row.addEventListener('click', () => {
    state.selectedNodeId = node.id;
    if (node.type === 'folder') state.expanded.add(node.id);
    renderTree();
    renderNodeForm();
    renderToolbarHint();
  });

  wrap.appendChild(row);

  if (node.type === 'folder' && isExpanded) {
    const childrenWrap = document.createElement('div');
    childrenWrap.className = 'tree-children';
    if (!node.children || node.children.length === 0) {
      const empty = document.createElement('div');
      empty.className = 'tree-empty-hint';
      empty.textContent = '(пусто)';
      childrenWrap.appendChild(empty);
    } else {
      node.children.forEach(child => childrenWrap.appendChild(renderNode(child, false)));
    }
    wrap.appendChild(childrenWrap);
  }

  return wrap;
}

// ── Render: node property form ────────────────────────────────
function renderNodeForm() {
  const term = getEditTerminal();
  if (!term || !state.selectedNodeId) {
    nodeForm.innerHTML = '<div class="node-empty">Выберите узел дерева слева</div>';
    return;
  }
  const loc = locateNode(term.root, state.selectedNodeId);
  if (!loc) {
    nodeForm.innerHTML = '<div class="node-empty">Выберите узел дерева слева</div>';
    return;
  }
  const node = loc.node;

  if (node.id === 'root') {
    nodeForm.innerHTML = `
      <div class="node-type-label">КОРЕНЬ ТЕРМИНАЛА</div>
      <div class="node-empty">Это главный экран терминала. Добавляйте папки, команды и записи через панель инструментов сверху.</div>`;
    return;
  }

  const typeLabel = node.type === 'folder' ? 'ПАПКА' : node.type === 'command' ? 'КОМАНДА' : 'ЗАПИСЬ';
  let html = `<div class="node-type-label">${typeLabel}</div>
    <div class="field-label">НАЗВАНИЕ</div>
    <input class="field-input" id="fldName" value="${escAttr(node.name)}">`;

  if (node.type === 'command') {
    html += `<div class="field-label">ТЕКСТ КОМАНДЫ (появится у игрока внизу экрана)</div>
      <textarea class="field-textarea" id="fldText">${escHtml(node.text || '')}</textarea>`;
  } else if (node.type === 'entry') {
    html += `<div class="field-label">ОПИСАНИЕ ЗАПИСИ</div>
      <textarea class="field-textarea" id="fldText">${escHtml(node.description || '')}</textarea>`;
  } else if (node.type === 'folder') {
    const count = node.children ? node.children.length : 0;
    html += `<div class="field-label">СОДЕРЖИМОЕ</div><div class="node-empty">${count} элемент(ов)</div>`;
  }

  html += `<div class="node-actions">
      <button class="btn btn-primary" id="btnApplyNode">ПРИМЕНИТЬ</button>
      <button class="btn btn-danger" id="btnDeleteNode">УДАЛИТЬ</button>
    </div>`;

  nodeForm.innerHTML = html;

  document.getElementById('btnApplyNode').addEventListener('click', () => {
    const nameEl = document.getElementById('fldName');
    const name = nameEl.value.trim();
    if (!name) { nameEl.focus(); return; }
    node.name = name;
    if (node.type === 'command') node.text = document.getElementById('fldText').value;
    if (node.type === 'entry')   node.description = document.getElementById('fldText').value;
    autosave();
    renderTree();
    renderNodeForm();
    renderToolbarHint();
  });

  document.getElementById('btnDeleteNode').addEventListener('click', () => {
    const childCount = (node.type === 'folder' && node.children) ? node.children.length : 0;
    const msg = childCount > 0
      ? `Удалить "${node.name}" вместе со всем содержимым (${childCount} элемент(ов))?`
      : `Удалить "${node.name}"?`;
    if (!window.confirm(msg)) return;
    const siblings = loc.parent.children;
    const idx = siblings.findIndex(c => c.id === node.id);
    if (idx >= 0) siblings.splice(idx, 1);
    state.selectedNodeId = null;
    autosave();
    renderTree();
    renderNodeForm();
    renderToolbarHint();
  });
}

// ── Toolbar: add nodes ────────────────────────────────────────
function addNode(type) {
  const target = currentAddTarget();
  if (!target) return;

  const node = {
    id:   uid('n'),
    type,
    name: type === 'folder' ? 'Новая папка' : type === 'command' ? 'Новая команда' : 'Новая запись',
  };
  if (type === 'folder')  node.children = [];
  if (type === 'command') node.text = '';
  if (type === 'entry')   node.description = '';

  if (!target.children) target.children = [];
  target.children.push(node);
  state.expanded.add(target.id);
  state.selectedNodeId = node.id;

  autosave();
  renderTree();
  renderNodeForm();
  renderToolbarHint();
}

btnAddFolder.addEventListener('click', () => addNode('folder'));
btnAddCommand.addEventListener('click', () => addNode('command'));
btnAddEntry.addEventListener('click', () => addNode('entry'));

// ── Terminal management ───────────────────────────────────────
btnAddTerminal.addEventListener('click', () => {
  const term = {
    id:        uid('t'),
    name:      `Терминал ${state.session.terminals.length + 1}`,
    hackLevel: 0,
    introText: '',
    root:      { id: 'root', type: 'folder', name: 'ROOT', children: [] },
  };
  state.session.terminals.push(term);
  state.editTerminalId = term.id;
  state.selectedNodeId = null;
  state.expanded = new Set(['root']);
  autosave();
  renderAll();
});

btnApplySettings.addEventListener('click', () => {
  const term = getEditTerminal();
  if (!term) return;
  term.hackLevel = Number(hackLevelSelect.value) || 0;
  term.introText = introTextArea.value;
  autosave();
  if (term.id === state.liveTerminalId) {
    // Intro text can refresh live immediately; hackLevel only takes effect
    // on the next (re)broadcast so it never disrupts an in-progress hack.
    ipcRenderer.send('terminal:update-live', { tree: term.root, introText: term.introText });
  }
});

btnMakeLive.addEventListener('click', () => {
  const term = getEditTerminal();
  if (!term) return;
  state.liveTerminalId = term.id;
  state.liveHack = null;
  ipcRenderer.send('terminal:set-live', {
    terminalId:   term.id,
    terminalName: term.name,
    tree:         term.root,
    hackLevel:    term.hackLevel || 0,
    introText:    term.introText || '',
  });
  renderTermList();
  renderTreeHeader();
  renderHackStatus();
});

btnPublish.addEventListener('click', () => {
  const term = getEditTerminal();
  if (!term || term.id !== state.liveTerminalId) return;
  ipcRenderer.send('terminal:update-live', { tree: term.root, introText: term.introText || '' });
  const original = btnPublish.textContent;
  btnPublish.textContent = 'ОБНОВЛЕНО ✓';
  setTimeout(() => { btnPublish.textContent = original; }, 1200);
});

btnStopBroadcast.addEventListener('click', () => {
  if (!state.liveTerminalId) return;
  ipcRenderer.send('terminal:clear-live');
  state.liveTerminalId = null;
  state.liveHack = null;
  renderTermList();
  renderTreeHeader();
  renderHackStatus();
});

btnHackSuccess.addEventListener('click', () => {
  if (!state.liveHack || state.liveHack.solved || state.liveHack.failed) return;
  ipcRenderer.send('terminal:hack-force-success');
});
