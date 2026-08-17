'use strict';

const desktopAPI = window.desktopAPI;

// ── State ─────────────────────────────────────────────────
const state = {
  session:        null,   // { version, name, terminals: [{id,name,hackLevel,introText,root}] }
  filePath:       null,
  liveTerminalId: null,
  editTerminalId: null,
  selectedNodeId: null,
  expanded:       new Set(['root']),
  liveHack:       null,   // last known hack-state of the live terminal, or null
  coordination:   null,   // authoritative roster/session/broadcast projection
};

let idCounter = 0;
function uid(prefix) {
  idCounter++;
  return `${prefix}_${Date.now().toString(36)}_${idCounter}`;
}

// ── DOM refs ──────────────────────────────────────────────
const startScreen      = document.getElementById('startScreen');
const startStatus      = document.getElementById('startStatus');
const btnOpenSession   = document.getElementById('btnOpenSession');
const btnNewSession    = document.getElementById('btnNewSession');
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
const btnResetFailedHack = document.getElementById('btnResetFailedHack');
const hackLevelSelect   = document.getElementById('hackLevelSelect');
const introTextArea     = document.getElementById('introTextArea');
const btnApplySettings  = document.getElementById('btnApplySettings');
const termSettings      = document.getElementById('termSettings');
const broadcastSummary = document.getElementById('broadcastSummary');
const coordinationPanel = document.getElementById('coordinationPanel');
const playerConfigStatus = document.getElementById('playerConfigStatus');
const playerConfigError = document.getElementById('playerConfigError');
const btnOpenPlayerConfig = document.getElementById('btnOpenPlayerConfig');
const btnNewPlayerConfig = document.getElementById('btnNewPlayerConfig');
const characterRoster = document.getElementById('characterRoster');
const characterNameInput = document.getElementById('characterNameInput');
const btnAddCharacter = document.getElementById('btnAddCharacter');
const btnStartBroadcast = document.getElementById('btnStartBroadcast');
const btnEndBroadcast = document.getElementById('btnEndBroadcast');
const endBroadcastDialog = document.getElementById('endBroadcastDialog');
const btnCancelEndBroadcast = document.getElementById('btnCancelEndBroadcast');
const btnConfirmEndBroadcast = document.getElementById('btnConfirmEndBroadcast');
const coordinationStatus = document.getElementById('coordinationStatus');
const coordinationError = document.getElementById('coordinationError');
const logicalSessionList = document.getElementById('logicalSessionList');
const characterRosterRowTemplate = document.getElementById('characterRosterRowTemplate');
const logicalSessionRowTemplate = document.getElementById('logicalSessionRowTemplate');
const terminalSwitchDialog = document.getElementById('terminalSwitchDialog');
const terminalSwitchStatus = document.getElementById('terminalSwitchStatus');
const terminalSwitchError = document.getElementById('terminalSwitchError');
const terminalSwitchButtons = Array.from(document.querySelectorAll('[data-switch-decision]'));
const publicAccessSection = document.getElementById('publicAccessSection');
const publicAccessForm = document.getElementById('publicAccessForm');
const publicAccessEnabledPreference = document.getElementById('publicAccessEnabledPreference');
const publicAccessDomain = document.getElementById('publicAccessDomain');
const publicAccessUsername = document.getElementById('publicAccessUsername');
const publicAccessProviderToken = document.getElementById('publicAccessProviderToken');
const publicAccessPlayerPassword = document.getElementById('publicAccessPlayerPassword');
const publicAccessDeleteProviderToken = document.getElementById('publicAccessDeleteProviderToken');
const publicAccessDeletePlayerPassword = document.getElementById('publicAccessDeletePlayerPassword');
const publicAccessProviderPresence = document.getElementById('publicAccessProviderPresence');
const publicAccessPasswordPresence = document.getElementById('publicAccessPasswordPresence');
const publicAccessStatus = document.getElementById('publicAccessStatus');
const publicAccessError = document.getElementById('publicAccessError');
const publicAccessURL = document.getElementById('publicAccessURL');
const publicAccessCopyStatus = document.getElementById('publicAccessCopyStatus');
const btnSavePublicAccess = document.getElementById('btnSavePublicAccess');
const btnGeneratePlayerPassword = document.getElementById('btnGeneratePlayerPassword');
const btnStartPublicAccess = document.getElementById('btnStartPublicAccess');
const btnStopPublicAccess = document.getElementById('btnStopPublicAccess');
const btnCopyPublicURL = document.getElementById('btnCopyPublicURL');
const btnCopyPublicUsername = document.getElementById('btnCopyPublicUsername');
const btnCopyManualPassword = document.getElementById('btnCopyManualPassword');
const generatedPasswordDialog = document.getElementById('generatedPasswordDialog');
const generatedPasswordValue = document.getElementById('generatedPasswordValue');
const btnCopyGeneratedPassword = document.getElementById('btnCopyGeneratedPassword');
const btnDismissGeneratedPassword = document.getElementById('btnDismissGeneratedPassword');

let serverUrl = null;
let serverUrlTitle = '';
let saveGeneration = 0;
let saveInvocation = 0;
let latestRenderedSave = 0;
let newestDurableRevision = 0;
let coordinationCommandPending = false;
let pendingTerminalSwitch = null;
let startupStatus = null;
let publicAccessSnapshot = null;
let publicAccessCommandPending = false;
let sessionStateCommandPending = false;
let commandExecutionDialogRequestID = null;
let commandExecutionDecisionRequestID = null;
let commandExecutionDialogEpoch = 0;
const resolvedCommandExecutionRequestIDs = new Set();

const commandStateActions = document.createElement('div');
commandStateActions.className = 'settings-row command-state-terminal-actions';
commandStateActions.hidden = true;
commandStateActions.innerHTML = `
  <button class="btn btn-mini btn-danger" id="btnResetTerminalCommandStates" type="button">
    СБРОСИТЬ ВСЕ СОСТОЯНИЯ
  </button>`;
termSettings.appendChild(commandStateActions);
const btnResetTerminalCommandStates = document.getElementById('btnResetTerminalCommandStates');

const commandExecutionDialog = document.createElement('dialog');
commandExecutionDialog.className = 'terminal-switch-dialog command-execution-dialog';
commandExecutionDialog.id = 'commandExecutionDialog';
commandExecutionDialog.hidden = true;
commandExecutionDialog.setAttribute('aria-modal', 'true');
commandExecutionDialog.setAttribute('aria-labelledby', 'commandExecutionDialogTitle');
commandExecutionDialog.setAttribute('aria-describedby', 'commandExecutionDialogDescription commandExecutionDialogStatus commandExecutionDialogError');
commandExecutionDialog.innerHTML = `
  <div class="terminal-switch-dialog-panel">
    <h2 class="terminal-switch-dialog-title" id="commandExecutionDialogTitle">ПОДТВЕРЖДЕНИЕ КОМАНДЫ</h2>
    <p class="terminal-switch-dialog-description" id="commandExecutionDialogDescription"></p>
    <div class="terminal-switch-actions" role="group" aria-label="Решение мастера по выполнению команды" style="grid-template-columns:repeat(2,minmax(0,1fr))">
      <button class="btn btn-primary" id="btnApproveCommandExecution" type="button">ОДОБРИТЬ</button>
      <button class="btn btn-danger" id="btnRejectCommandExecution" type="button">ОТКЛОНИТЬ</button>
    </div>
    <div class="terminal-switch-status" id="commandExecutionDialogStatus" role="status" aria-live="polite" aria-atomic="true"></div>
    <div class="terminal-switch-error" id="commandExecutionDialogError" role="alert" aria-live="assertive" aria-atomic="true" hidden></div>
  </div>`;
document.body.appendChild(commandExecutionDialog);
const commandExecutionDialogDescription = document.getElementById('commandExecutionDialogDescription');
const commandExecutionDialogStatus = document.getElementById('commandExecutionDialogStatus');
const commandExecutionDialogError = document.getElementById('commandExecutionDialogError');
const btnApproveCommandExecution = document.getElementById('btnApproveCommandExecution');
const btnRejectCommandExecution = document.getElementById('btnRejectCommandExecution');

function renderStartupPresentation(status) {
  startupStatus = status && typeof status === 'object' ? status : {};
  const info = startupStatus.serverInfo && typeof startupStatus.serverInfo === 'object'
    ? startupStatus.serverInfo
    : null;
  const startupError = typeof startupStatus.startupError === 'string' ? startupStatus.startupError : '';
  const tunnelError = typeof info?.tunnelError === 'string' ? info.tunnelError : '';
  const fatal = !info && Boolean(startupError);

  btnOpenSession.disabled = fatal;
  btnNewSession.disabled = fatal;
  if (fatal) {
    startStatus.dataset.state = 'failed';
    startStatus.textContent = `ЗАПУСК НЕ ЗАВЕРШЁН: ${startupError}`;
  } else if (info?.tunnel && info.url) {
    startStatus.dataset.state = 'ready-public';
    startStatus.textContent = `ГОТОВО · ПУБЛИЧНЫЙ И ЛОКАЛЬНЫЙ ДОСТУП${info.localUrl ? ` · ${info.localUrl}` : ''}`;
  } else if (info) {
    const warning = tunnelError || startupError;
    startStatus.dataset.state = warning ? 'warning' : 'ready-local';
    startStatus.textContent = warning
      ? `ЛОКАЛЬНЫЙ РЕЖИМ ГОТОВ · ПУБЛИЧНЫЙ ДОСТУП НЕДОСТУПЕН: ${warning}`
      : `ЛОКАЛЬНЫЙ РЕЖИМ ГОТОВ · ${info.localUrl || info.url}`;
  } else {
    startStatus.dataset.state = 'starting';
    startStatus.textContent = 'ЗАПУСК ЛОКАЛЬНОГО СЕРВЕРА…';
  }
}

// ── Server info / connection count ─────────────────────────
desktopAPI.onServerInfo((info) => {
  renderStartupPresentation({ ...(startupStatus || {}), serverInfo: info });
  const publicUrl = info.tunnel && info.url ? info.url : '';
  const localUrl = info.localUrl || (!info.tunnel ? info.url : '');
  const tunnelUnavailable = info.tunnel && !publicUrl;

  serverUrl = publicUrl || localUrl || null;
  serverUrlEl.classList.toggle('server-url-error', Boolean(info.tunnelError) || tunnelUnavailable);

  if (info.tunnelError) {
    serverUrlEl.textContent = localUrl
      ? `NGROK: ОШИБКА · ЛОКАЛЬНО: ${localUrl}`
      : 'NGROK: ОШИБКА';
    serverUrlTitle = localUrl
      ? `${info.tunnelError}\nЛокальная ссылка остаётся доступна (нажмите, чтобы открыть)`
      : info.tunnelError;
  } else if (publicUrl) {
    serverUrlEl.textContent = publicUrl;
    serverUrlTitle = localUrl
      ? `Публичная ссылка (нажмите, чтобы открыть)\nЛокально: ${localUrl}`
      : 'Публичная ссылка (нажмите, чтобы открыть)';
  } else if (localUrl) {
    serverUrlEl.textContent = localUrl;
    serverUrlTitle = tunnelUnavailable
      ? 'Публичная ссылка недоступна; локальная ссылка остаётся доступна'
      : 'Локальная ссылка (нажмите, чтобы открыть)';
  } else {
    serverUrlEl.textContent = '—';
    serverUrlTitle = 'Адрес игроков пока недоступен';
  }
  serverUrlEl.title = serverUrlTitle;
});
desktopAPI.onClientCount((count) => {
  clientCountEl.textContent = count;
});
desktopAPI.onHackState((hack) => {
  state.liveHack = hack;
  renderHackStatus();
});
desktopAPI.onCoordinationState((coordination) => {
  applyCoordinationState(coordination);
  renderCoordination();
  if (state.session) {
    renderTermList();
    renderTreeHeader();
    renderHackStatus();
  }
});
if (typeof desktopAPI.onSessionState === 'function') {
  desktopAPI.onSessionState((event) => {
    const revision = Number(event?.revision);
    if (!state.session || !event?.session || !Number.isSafeInteger(revision) ||
        revision <= newestDurableRevision) return;
    state.session = event.session;
    newestDurableRevision = revision;
    saveStatus.textContent = `СОСТОЯНИЕ СЕССИИ ОБНОВЛЕНО · ревизия ${revision}`;
    saveStatus.dataset.savedRevision = String(revision);
    saveStatus.classList.remove('err');
    renderAll();
  });
}
void desktopAPI.getRuntimeStatus().then(renderStartupPresentation);
serverUrlEl.addEventListener('click', async () => {
  const requestedUrl = serverUrl;
  if (!requestedUrl) return;

  // The frontend never opens a URL directly. The Go command parses the final
  // value and rejects malformed or non-HTTP(S) protocols at the privilege edge.
  const result = await desktopAPI.openUrl(requestedUrl);
  if (requestedUrl === serverUrl && result && result.ok === false) {
    const detail = result.error ? `: ${result.error}` : '';
    serverUrlEl.title = `${serverUrlTitle}\nНе удалось открыть ссылку${detail}`;
  }
});

// ── Public access: trusted settings and explicit lifecycle ──
const publicAccessStateLabels = Object.freeze({
  stopped: 'ОСТАНОВЛЕН',
  starting: 'ЗАПУСК…',
  ready: 'ГОТОВ',
  stopping: 'ОСТАНОВКА…',
  error: 'ОШИБКА',
});

function renderSecretPresence(element, presence) {
  element.dataset.presence = presence;
  element.textContent = presence === 'present'
    ? 'СОХРАНЕН'
    : presence === 'absent' ? 'НЕ СОХРАНЕН' : 'НЕДОСТУПЕН';
}

function renderPublicAccess(snapshot) {
  if (!snapshot || typeof snapshot !== 'object') return;
  if (publicAccessSnapshot) {
    const candidateGeneration = Number(snapshot.status?.generation || 0);
    const currentGeneration = Number(publicAccessSnapshot.status?.generation || 0);
    const candidateRevision = Number(snapshot.status?.settingsRevision || snapshot.preferences?.revision || 0);
    const currentRevision = Number(publicAccessSnapshot.status?.settingsRevision || publicAccessSnapshot.preferences?.revision || 0);
    if (candidateGeneration < currentGeneration ||
      (candidateGeneration === currentGeneration && candidateRevision < currentRevision)) return;
  }
  publicAccessSection.hidden = false;
  publicAccessSnapshot = snapshot;
  const preferences = snapshot.preferences || {};
  const status = snapshot.status || {};
  publicAccessEnabledPreference.checked = preferences.enabledPreference === true;
  publicAccessDomain.value = preferences.reservedDomain || '';
  publicAccessUsername.value = preferences.username || 'players';
  renderSecretPresence(publicAccessProviderPresence, snapshot.providerTokenPresence);
  renderSecretPresence(publicAccessPasswordPresence, snapshot.playerPasswordPresence);
  publicAccessStatus.textContent = publicAccessStateLabels[status.state] || 'ЗАГРУЗКА…';
  publicAccessStatus.dataset.state = status.state || 'loading';
  publicAccessStatus.dataset.generation = String(Number(status.generation || 0));
  publicAccessStatus.dataset.settingsRevision = String(Number(status.settingsRevision || preferences.revision || 0));
  const publicFailure = status.errorMessage || 'ПУБЛИЧНЫЙ ДОСТУП НЕДОСТУПЕН';
  publicAccessError.textContent = status.state === 'error'
    ? `${publicFailure} · ЛОКАЛЬНЫЙ РЕЖИМ ПРОДОЛЖАЕТ РАБОТАТЬ`
    : '';
  publicAccessError.hidden = publicAccessError.textContent === '';
  publicAccessURL.textContent = status.state === 'ready' ? (status.publicUrl || '') : '';
  btnCopyPublicURL.hidden = publicAccessURL.textContent === '';
  const transitioning = status.state === 'starting' || status.state === 'stopping';
  const disabled = publicAccessCommandPending || transitioning;
  for (const control of [
    publicAccessEnabledPreference, publicAccessDomain, publicAccessUsername,
    publicAccessProviderToken, publicAccessPlayerPassword,
    publicAccessDeleteProviderToken, publicAccessDeletePlayerPassword,
    btnSavePublicAccess, btnGeneratePlayerPassword,
  ]) control.disabled = disabled;
  btnStartPublicAccess.disabled = disabled || status.state === 'ready';
  btnStopPublicAccess.disabled = disabled || status.state === 'stopped';
  btnCopyManualPassword.disabled = disabled || publicAccessPlayerPassword.value === '';
}

function publicAccessRevision() {
  return Number(publicAccessSnapshot?.preferences?.revision || 0);
}

async function copyTransientText(value, successMessage) {
  if (!value) return false;
  let copied = false;
  try {
    if (typeof navigator.clipboard?.writeText === 'function') {
      await navigator.clipboard.writeText(value);
      copied = true;
    }
  } catch {
    // Packaged WebViews may deny the browser Clipboard API. The native Wails
    // runtime is the bounded fallback and returns no copy of the value.
  }
  if (!copied) {
    copied = await desktopAPI.writeClipboardText(value);
  }
  if (copied) {
    publicAccessCopyStatus.textContent = successMessage;
    return true;
  }
  publicAccessCopyStatus.textContent = 'НЕ УДАЛОСЬ СКОПИРОВАТЬ';
  return false;
}

function showGeneratedPassword(oneTimeValue) {
  let transientValue = oneTimeValue;
  generatedPasswordValue.textContent = transientValue;
  const clearAndClose = () => {
    transientValue = '';
    generatedPasswordValue.textContent = '';
    btnCopyGeneratedPassword.onclick = null;
    btnDismissGeneratedPassword.onclick = null;
    generatedPasswordDialog.oncancel = null;
    if (generatedPasswordDialog.open) generatedPasswordDialog.close();
    publicAccessPlayerPassword.value = '';
    btnGeneratePlayerPassword.focus();
  };
  btnCopyGeneratedPassword.onclick = async () => {
    await copyTransientText(transientValue, 'ПАРОЛЬ СКОПИРОВАН');
    clearAndClose();
  };
  btnDismissGeneratedPassword.onclick = clearAndClose;
  generatedPasswordDialog.oncancel = (event) => {
    event.preventDefault();
    clearAndClose();
  };
  generatedPasswordDialog.showModal();
  btnCopyGeneratedPassword.focus();
}

// The facade owns exact `public-access-status` event ordering and stale-snapshot suppression.
desktopAPI.onPublicAccessStatus(renderPublicAccess);

publicAccessForm.addEventListener('submit', async (event) => {
  event.preventDefault();
  if (publicAccessCommandPending) return;
  if (publicAccessSnapshot?.status?.state === 'ready' &&
    !window.confirm('ПУБЛИЧНЫЙ ДОСТУП АКТИВЕН. СОХРАНЕНИЕ ОСТАНОВИТ И ПЕРЕЗАПУСТИТ ССЫЛКУ. ПРОДОЛЖИТЬ?')) return;
  const request = {
    expectedRevision: publicAccessRevision(),
    enabledPreference: publicAccessEnabledPreference.checked,
    reservedDomain: publicAccessDomain.value,
    username: publicAccessUsername.value,
    replacementProviderToken: publicAccessProviderToken.value,
    deleteProviderToken: publicAccessDeleteProviderToken.checked,
    replacementPlayerPassword: publicAccessPlayerPassword.value,
    deletePlayerPassword: publicAccessDeletePlayerPassword.checked,
  };
  publicAccessCommandPending = true;
  renderPublicAccess(publicAccessSnapshot);
  const pending = desktopAPI.savePublicAccessSettings(request);
  publicAccessProviderToken.value = '';
  publicAccessPlayerPassword.value = '';
  request.replacementProviderToken = '';
  request.replacementPlayerPassword = '';
  const result = await pending;
  publicAccessCommandPending = false;
  publicAccessDeleteProviderToken.checked = false;
  publicAccessDeletePlayerPassword.checked = false;
  renderPublicAccess(result.snapshot || publicAccessSnapshot);
  if (!result.ok) {
    publicAccessError.textContent = result.error || 'НЕ УДАЛОСЬ СОХРАНИТЬ НАСТРОЙКИ';
    publicAccessError.hidden = false;
  }
});

btnGeneratePlayerPassword.addEventListener('click', async () => {
  if (publicAccessCommandPending) return;
  if (publicAccessSnapshot?.status?.state === 'ready' &&
    !window.confirm('ПУБЛИЧНЫЙ ДОСТУП АКТИВЕН. НОВЫЙ ПАРОЛЬ ОСТАНОВИТ И ПЕРЕЗАПУСТИТ ССЫЛКУ. ПРОДОЛЖИТЬ?')) return;
  publicAccessCommandPending = true;
  renderPublicAccess(publicAccessSnapshot);
  const result = await desktopAPI.generatePlayerPassword({ expectedRevision: publicAccessRevision() });
  publicAccessCommandPending = false;
  if (!result.ok || !result.generatedPassword) {
    renderPublicAccess(publicAccessSnapshot);
    publicAccessError.textContent = result.error || 'НЕ УДАЛОСЬ СОЗДАТЬ ПАРОЛЬ';
    publicAccessError.hidden = false;
    return;
  }
  showGeneratedPassword(result.generatedPassword);
  const refreshed = await desktopAPI.getPublicAccess();
  renderPublicAccess(refreshed);
});

async function runPublicAccessLifecycle(command) {
  if (publicAccessCommandPending) return;
  publicAccessCommandPending = true;
  renderPublicAccess(publicAccessSnapshot);
  const result = await command({ expectedRevision: publicAccessRevision() });
  publicAccessCommandPending = false;
  renderPublicAccess(result.snapshot || publicAccessSnapshot);
  if (!result.ok) {
    publicAccessError.textContent = result.error || 'ОПЕРАЦИЯ ПУБЛИЧНОГО ДОСТУПА НЕ ВЫПОЛНЕНА';
    publicAccessError.hidden = false;
  }
}

btnStartPublicAccess.addEventListener('click', () => runPublicAccessLifecycle(desktopAPI.startPublicAccess));
btnStopPublicAccess.addEventListener('click', () => runPublicAccessLifecycle(desktopAPI.stopPublicAccess));
btnCopyPublicURL.addEventListener('click', () => copyTransientText(publicAccessURL.textContent, 'URL СКОПИРОВАН'));
btnCopyPublicUsername.addEventListener('click', () => copyTransientText(publicAccessUsername.value, 'ИМЯ СКОПИРОВАНО'));
btnCopyManualPassword.addEventListener('click', () => copyTransientText(publicAccessPlayerPassword.value, 'ВВЕДЁННЫЙ ПАРОЛЬ СКОПИРОВАН'));
publicAccessPlayerPassword.addEventListener('input', () => {
  btnCopyManualPassword.disabled = publicAccessCommandPending || publicAccessPlayerPassword.value === '';
});

// ── Start screen: open / new session ───────────────────────
btnOpenSession.addEventListener('click', async () => {
  const res = await desktopAPI.openSession();
  if (!res.ok) {
    if (res.error) startStatus.textContent = 'Ошибка: ' + res.error;
    return;
  }
  await loadSession(res.session, res.filePath);
});

btnNewSession.addEventListener('click', async () => {
  const res = await desktopAPI.newSession();
  if (!res.ok) {
    if (res.error) startStatus.textContent = 'Ошибка: ' + res.error;
    return;
  }
  await loadSession(res.session, res.filePath);
});

async function loadSession(session, filePath) {
  saveGeneration++;
  saveInvocation = 0;
  latestRenderedSave = 0;
  newestDurableRevision = 0;
  delete saveStatus.dataset.savedRevision;
  saveStatus.textContent = '';
  saveStatus.classList.remove('err');
  state.session        = session;
  state.filePath        = filePath;
  state.liveTerminalId  = state.coordination?.broadcast?.activeTerminalId || null;
  state.editTerminalId  = (session.terminals[0] && session.terminals[0].id) || null;
  state.selectedNodeId  = null;
  state.expanded         = new Set(['root']);
  state.liveHack         = null;

  sessionFileLabel.textContent = filePath;
  startScreen.style.display = 'none';
  mainLayout.style.display  = 'flex';
  renderAll();

  if (session.playerConfig) {
    await runPlayerConfigCommand(
      () => desktopAPI.loadReferencedPlayerConfig(),
      'КОНФИГУРАЦИЯ ИГРОКОВ ЗАГРУЖЕНА'
    );
  } else {
    setPlayerConfigError('ВЫБЕРИТЕ ИЛИ СОЗДАЙТЕ КОНФИГУРАЦИЮ ИГРОКОВ');
  }
}

// ── Autosave (writes to the currently open session file) ──
async function autosave() {
  if (!state.filePath) return;
  const generation = saveGeneration;
  const invocation = ++saveInvocation;
  const res = await desktopAPI.saveSession(state.session);

  // A completion from a previously-open session or an older durable revision
  // must never replace status for newer work.
  if (generation !== saveGeneration) return;
  const durableRevision = Number(res.savedRevision || 0);
  if (durableRevision < newestDurableRevision || invocation < latestRenderedSave) return;
  if (!res.ok && invocation < saveInvocation) return;

  latestRenderedSave = invocation;
  newestDurableRevision = Math.max(newestDurableRevision, durableRevision);
  if (res.ok) {
    const revisionLabel = newestDurableRevision > 0
      ? ` · ревизия ${newestDurableRevision}`
      : '';
    saveStatus.textContent = 'Сохранено' + revisionLabel + ' · ' + new Date().toLocaleTimeString();
    saveStatus.dataset.savedRevision = String(newestDurableRevision);
    saveStatus.classList.remove('err');
  } else {
    saveStatus.textContent = 'Ошибка сохранения: ' + (res.error || '');
    saveStatus.dataset.savedRevision = String(newestDurableRevision);
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

function commandExecutionState(term, commandID) {
  const commandStates = term?.commandStates;
  if (!commandStates || typeof commandStates !== 'object') return null;
  const snapshot = commandStates[commandID];
  return snapshot && typeof snapshot === 'object' ? snapshot : null;
}

function effectiveNodeName(term, node) {
  if (node?.type !== 'command') return node?.name || '';
  const completedName = commandExecutionState(term, node.id)?.completedName;
  return typeof completedName === 'string' && completedName ? completedName : node.name;
}

function renderSessionStateResult(result, successMessage) {
  if (!result?.ok || !result.session) {
    saveStatus.textContent = 'Ошибка изменения состояния: ' + (result?.error || 'сессия не обновлена');
    saveStatus.classList.add('err');
    return false;
  }

  state.session = result.session;
  const revision = Number(result.revision || result.savedRevision || 0);
  newestDurableRevision = Math.max(newestDurableRevision, revision);
  saveStatus.textContent = successMessage + (revision > 0 ? ` · ревизия ${revision}` : '');
  saveStatus.dataset.savedRevision = String(newestDurableRevision);
  saveStatus.classList.remove('err');
  renderAll();
  return true;
}

async function runSessionStateCommand(command, successMessage) {
  if (sessionStateCommandPending) return;
  sessionStateCommandPending = true;
  renderSettingsPanel();
  renderNodeForm();
  try {
    const result = await command();
    renderSessionStateResult(result, successMessage);
  } catch (error) {
    saveStatus.textContent = 'Ошибка изменения состояния: '
      + (error instanceof Error ? error.message : String(error));
    saveStatus.classList.add('err');
  } finally {
    sessionStateCommandPending = false;
    renderSettingsPanel();
    renderNodeForm();
  }
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
  renderCoordination();
}

// ── Render: authoritative roster and broadcast state ────────
function renderCoordination() {
  const coordination = state.coordination;
  const roster = Array.isArray(coordination?.roster) ? coordination.roster : [];
  const sessions = Array.isArray(coordination?.sessions) ? coordination.sessions : [];
  const broadcast = coordination?.broadcast || null;
  const playerConfig = coordination?.playerConfig || null;
  const availableCharacters = roster.filter(character => !character.claimedBySessionId);
  const unassignedSessions = sessions.filter(session => !session.character);

  broadcastSummary.textContent = broadcast
    ? (broadcast.activeTerminalId
      ? `ТРАНСЛЯЦИЯ АКТИВНА · ТЕРМИНАЛ ${broadcast.activeTerminalId}`
      : `ТРАНСЛЯЦИЯ АКТИВНА · ОЖИДАНИЕ ТЕРМИНАЛА · ${broadcast.id}`)
    : 'ТРАНСЛЯЦИЯ НЕ ЗАПУЩЕНА';
  broadcastSummary.classList.toggle('is-live', Boolean(broadcast));
  coordinationPanel.dataset.playerConfigActive = String(Boolean(playerConfig));
  playerConfigStatus.dataset.active = String(Boolean(playerConfig));
  playerConfigStatus.textContent = playerConfig
    ? `${playerConfig.name} · ${playerConfig.filePath}`
    : 'НЕ ВЫБРАНА · СОЗДАЙТЕ ИЛИ ВЫБЕРИТЕ ФАЙЛ';
  btnOpenPlayerConfig.disabled = coordinationCommandPending || Boolean(broadcast);
  btnNewPlayerConfig.disabled = coordinationCommandPending || Boolean(broadcast);
  btnStartBroadcast.disabled = coordinationCommandPending || Boolean(broadcast) || !playerConfig;
  btnEndBroadcast.hidden = !broadcast;
  btnEndBroadcast.disabled = coordinationCommandPending || !broadcast;
  btnAddCharacter.disabled = coordinationCommandPending || !playerConfig;
  characterNameInput.disabled = coordinationCommandPending || !playerConfig;

  characterRoster.replaceChildren();
  if (!roster.length) {
    const empty = document.createElement('div');
    empty.className = 'roster-empty';
    empty.textContent = 'ПЕРСОНАЖИ НЕ ЗАДАНЫ';
    characterRoster.appendChild(empty);
  } else {
    for (const character of roster) {
      const fragment = characterRosterRowTemplate.content.cloneNode(true);
      const row = fragment.querySelector('.roster-row');
      const claimed = Boolean(character.claimedBySessionId);
      row.dataset.characterId = character.id;
      row.dataset.claimed = String(claimed);
      row.querySelector('.roster-name').textContent = character.name || '—';
      const claimStatus = row.querySelector('.roster-claim-status');
      claimStatus.dataset.claimState = claimed ? 'claimed' : 'available';
      claimStatus.textContent = claimed ? 'ЗАНЯТ' : 'СВОБОДЕН';

      const nameInput = row.querySelector('.roster-name-input');
      nameInput.value = character.name || '';
      const renameButton = row.querySelector('.roster-rename');
      const deleteButton = row.querySelector('.roster-delete');
      const moveControls = row.querySelector('.roster-move-controls');
      const moveSelect = row.querySelector('.roster-move-session-select');
      const moveButton = row.querySelector('.roster-move');
      for (const control of row.querySelectorAll('input, select, button')) {
        control.disabled = coordinationCommandPending || !playerConfig;
      }
      renameButton.addEventListener('click', () => {
        const name = nameInput.value.trim();
        if (!name) return setCoordinationStatus('УКАЖИТЕ ИМЯ ПЕРСОНАЖА', true);
        runCoordinationCommand(
          () => desktopAPI.renameCharacter({ characterId: character.id, name }),
          'ПЕРСОНАЖ ПЕРЕИМЕНОВАН',
          'ПЕРЕИМЕНОВАНИЕ ПЕРСОНАЖА...'
        );
      });
      deleteButton.addEventListener('click', () => runCoordinationCommand(
        () => desktopAPI.deleteCharacter(character.id),
        'ПЕРСОНАЖ УДАЛЁН',
        claimed ? 'ПРОВЕРКА АКТИВНОГО НАЗНАЧЕНИЯ...' : 'УДАЛЕНИЕ ПЕРСОНАЖА...'
      ));

      moveControls.hidden = !claimed;
      fillSelect(moveSelect, unassignedSessions, session => session.id, session => sessionLabel(session), 'НЕТ СВОБОДНЫХ СЕССИЙ');
      moveButton.disabled = coordinationCommandPending || !claimed || !moveSelect.value;
      moveButton.addEventListener('click', () => runCoordinationCommand(
        () => desktopAPI.moveCharacter({ characterId: character.id, toSessionId: moveSelect.value }),
        'НАЗНАЧЕНИЕ ПЕРЕМЕЩЕНО',
        'ПЕРЕМЕЩЕНИЕ НАЗНАЧЕНИЯ...'
      ));
      characterRoster.appendChild(fragment);
    }
  }

  logicalSessionList.replaceChildren();
  if (!sessions.length) {
    const empty = document.createElement('div');
    empty.className = 'session-empty';
    empty.setAttribute('role', 'listitem');
    empty.textContent = 'СЕССИИ НЕ ПОДКЛЮЧЕНЫ';
    logicalSessionList.appendChild(empty);
    return;
  }

  for (const session of sessions) {
    const fragment = logicalSessionRowTemplate.content.cloneNode(true);
    const row = fragment.querySelector('.session-row');
    const assigned = Boolean(session.character);
    const role = session.role || 'unassigned';
    row.dataset.sessionId = session.id;
    row.dataset.connected = String(Boolean(session.connected));
    row.dataset.role = role;
    row.querySelector('.session-primary-name').textContent = assigned
      ? session.character.name
      : session.fallbackName;
    const presence = row.querySelector('.session-presence');
    presence.dataset.presence = session.connected ? 'connected' : 'disconnected';
    presence.textContent = session.connected ? 'ПОДКЛЮЧЕН' : 'ОТКЛЮЧЕН';
    const roleLabel = row.querySelector('.session-role');
    roleLabel.dataset.sessionRole = role;
    roleLabel.textContent = role === 'active'
      ? (session.connected ? 'УПРАВЛЯЮЩИЙ' : 'УПРАВЛЯЮЩИЙ · НЕТ СВЯЗИ')
      : role === 'observer' ? 'НАБЛЮДАТЕЛЬ' : 'БЕЗ РОЛИ';
    row.querySelector('.session-character-name').textContent = assigned
      ? `ПЕРСОНАЖ: ${session.character.name}`
      : 'ПЕРСОНАЖ НЕ НАЗНАЧЕН';
    row.querySelector('.session-fallback-label').textContent = `СЕССИЯ: ${session.fallbackName}`;

    const nameInput = row.querySelector('.session-name-input');
    nameInput.value = session.fallbackName || '';
    const renameButton = row.querySelector('.session-rename');
    const assignmentControls = row.querySelector('.session-assignment-controls');
    const characterSelect = row.querySelector('.session-character-select');
    const assignButton = row.querySelector('.session-assign');
    const claimedControls = row.querySelector('.session-claimed-controls');
    const releaseButton = row.querySelector('.session-release');
    const controllerButton = row.querySelector('.session-controller');
    for (const control of row.querySelectorAll('input, select, button')) {
      control.disabled = coordinationCommandPending;
    }
    renameButton.addEventListener('click', () => {
      const fallbackName = nameInput.value.trim();
      if (!fallbackName) return setCoordinationStatus('УКАЖИТЕ МЕТКУ СЕССИИ', true);
      runCoordinationCommand(
        () => desktopAPI.renameLogicalSession({ sessionId: session.id, fallbackName }),
        'МЕТКА СЕССИИ ОБНОВЛЕНА',
        'ПЕРЕИМЕНОВАНИЕ СЕССИИ...'
      );
    });

    assignmentControls.hidden = assigned || !broadcast;
    fillSelect(characterSelect, availableCharacters, character => character.id, character => character.name, 'НЕТ ДОСТУПНЫХ ПЕРСОНАЖЕЙ');
    assignButton.disabled = coordinationCommandPending || assigned || !broadcast || !characterSelect.value;
    assignButton.addEventListener('click', () => runCoordinationCommand(
      () => desktopAPI.assignCharacter({ sessionId: session.id, characterId: characterSelect.value }),
      'ПЕРСОНАЖ НАЗНАЧЕН',
      'НАЗНАЧЕНИЕ ПЕРСОНАЖА...'
    ));

    claimedControls.hidden = !assigned;
    releaseButton.disabled = coordinationCommandPending || !assigned;
    releaseButton.addEventListener('click', () => runCoordinationCommand(
      () => desktopAPI.releaseCharacter(session.id),
      'ПЕРСОНАЖ ОСВОБОЖДЁН',
      'ОСВОБОЖДЕНИЕ ПЕРСОНАЖА...'
    ));
    controllerButton.disabled = true;
    controllerButton.hidden = role === 'active';
    if (assigned && session.connected && role !== 'active') {
      controllerButton.disabled = coordinationCommandPending;
      controllerButton.addEventListener('click', () => runCoordinationCommand(
        () => desktopAPI.setActiveController(session.id),
        'УПРАВЛЕНИЕ ПЕРЕДАНО',
        'ПЕРЕДАЧА УПРАВЛЕНИЯ...'
      ));
    }
    logicalSessionList.appendChild(fragment);
  }
}

function setPlayerConfigError(message = '') {
  playerConfigError.textContent = message;
  playerConfigError.hidden = !message;
}

async function runPlayerConfigCommand(command, successMessage) {
  if (coordinationCommandPending || state.coordination?.broadcast) return null;
  coordinationCommandPending = true;
  setPlayerConfigError('');
  setCoordinationStatus('ЗАГРУЗКА КОНФИГУРАЦИИ ИГРОКОВ...');
  renderCoordination();
  let result;
  try {
    result = await command();
  } catch (error) {
    result = { ok: false, error: error instanceof Error ? error.message : String(error) };
  }
  coordinationCommandPending = false;
  if (result?.canceled) {
    setCoordinationStatus('ВЫБОР КОНФИГУРАЦИИ ОТМЕНЁН');
    setPlayerConfigError('');
    renderCoordination();
    return result;
  }
  if (!result?.ok) {
    const message = result?.error || 'НЕ УДАЛОСЬ ЗАГРУЗИТЬ КОНФИГУРАЦИЮ ИГРОКОВ';
    setCoordinationStatus(message, true);
    setPlayerConfigError(message);
    renderCoordination();
    return result;
  }
  if (result.session) state.session = result.session;
  applyCoordinationState(result.state || state.coordination);
  setCoordinationStatus(successMessage);
  setPlayerConfigError('');
  renderAll();
  return result;
}

btnOpenPlayerConfig.addEventListener('click', () => runPlayerConfigCommand(
  () => desktopAPI.openPlayerConfig(),
  'КОНФИГУРАЦИЯ ИГРОКОВ ВЫБРАНА'
));

btnNewPlayerConfig.addEventListener('click', () => runPlayerConfigCommand(
  () => desktopAPI.newPlayerConfig(),
  'КОНФИГУРАЦИЯ ИГРОКОВ СОЗДАНА'
));

function coordinationRevision(coordination) {
  const revision = Number(coordination?.revision || 0);
  return Number.isSafeInteger(revision) && revision >= 0 ? revision : 0;
}

function rememberResolvedCommandExecution(requestID) {
  resolvedCommandExecutionRequestIDs.add(requestID);
  if (resolvedCommandExecutionRequestIDs.size <= 128) return;
  const oldest = resolvedCommandExecutionRequestIDs.values().next().value;
  resolvedCommandExecutionRequestIDs.delete(oldest);
}

function hideCommandExecutionDialog() {
  // Any authoritative close invalidates the promise currently resolving this
  // dialog. Its eventual callback must not overwrite a newer lifecycle state
  // or dismiss a different request shown in the meantime.
  commandExecutionDialogEpoch += 1;
  commandExecutionDecisionRequestID = null;
  commandExecutionDialogRequestID = null;
  commandExecutionDialog.hidden = true;
  if (typeof commandExecutionDialog.close === 'function' && commandExecutionDialog.open) {
    commandExecutionDialog.close();
  } else {
    commandExecutionDialog.removeAttribute('open');
  }
  btnApproveCommandExecution.disabled = false;
  btnRejectCommandExecution.disabled = false;
  commandExecutionDialogStatus.textContent = '';
  commandExecutionDialogError.textContent = '';
  commandExecutionDialogError.hidden = true;
}

function showCommandExecutionDialog(pending) {
  commandExecutionDialogEpoch += 1;
  commandExecutionDecisionRequestID = null;
  commandExecutionDialogRequestID = pending.requestId;
  commandExecutionDialogDescription.textContent = pending.confirmationText;
  commandExecutionDialogStatus.textContent = pending.commandName
    ? `КОМАНДА: ${pending.commandName}`
    : 'КОМАНДА ОЖИДАЕТ РЕШЕНИЯ';
  commandExecutionDialogError.textContent = '';
  commandExecutionDialogError.hidden = true;
  btnApproveCommandExecution.disabled = false;
  btnRejectCommandExecution.disabled = false;
  commandExecutionDialog.hidden = false;
  if (typeof commandExecutionDialog.showModal === 'function' && !commandExecutionDialog.open) {
    commandExecutionDialog.showModal();
  } else {
    commandExecutionDialog.setAttribute('open', '');
  }
  btnApproveCommandExecution.focus();
}

function syncCommandExecutionDialog(coordination) {
  const pending = coordination?.pendingCommandExecution;
  const requestID = typeof pending?.requestId === 'string' ? pending.requestId : '';
  if (!requestID) {
    if (commandExecutionDialogRequestID) hideCommandExecutionDialog();
    return;
  }
  if (requestID === commandExecutionDecisionRequestID || resolvedCommandExecutionRequestIDs.has(requestID)) {
    return;
  }
  if (requestID === commandExecutionDialogRequestID) return;
  if (commandExecutionDialogRequestID) hideCommandExecutionDialog();
  showCommandExecutionDialog(pending);
}

function applyCoordinationState(coordination) {
  if (coordination && state.coordination &&
      coordinationRevision(coordination) <= coordinationRevision(state.coordination)) {
    return false;
  }
  state.coordination = coordination || null;
  state.liveTerminalId = coordination?.broadcast?.activeTerminalId || null;
  syncCommandExecutionDialog(coordination);
  return true;
}

async function resolveCommandExecution(decision) {
  const requestID = commandExecutionDialogRequestID;
  if (!requestID || commandExecutionDecisionRequestID) return null;

  commandExecutionDecisionRequestID = requestID;
  const epoch = commandExecutionDialogEpoch;
  const startingRevision = coordinationRevision(state.coordination);
  btnApproveCommandExecution.disabled = true;
  btnRejectCommandExecution.disabled = true;
  commandExecutionDialogStatus.textContent = decision === 'approve'
    ? 'СОХРАНЕНИЕ И ВЫПОЛНЕНИЕ...'
    : 'ОТКЛОНЕНИЕ ЗАПРОСА...';
  commandExecutionDialogError.textContent = '';
  commandExecutionDialogError.hidden = true;

  let result;
  try {
    result = await desktopAPI.resolveCommandExecution({ requestId: requestID, decision });
  } catch (error) {
    result = { ok: false, error: error instanceof Error ? error.message : String(error) };
  }

  if (epoch !== commandExecutionDialogEpoch || commandExecutionDecisionRequestID !== requestID) {
    return result;
  }
  commandExecutionDecisionRequestID = null;
  const resultRevision = coordinationRevision(result?.state);
  if (resultRevision > 0 && resultRevision < coordinationRevision(state.coordination)) {
    return result;
  }
  if (!result?.state && coordinationRevision(state.coordination) > startingRevision &&
      state.coordination?.pendingCommandExecution?.requestId !== requestID) {
    return result;
  }

  rememberResolvedCommandExecution(requestID);
  if (result?.state) applyCoordinationState(result.state);
  if (commandExecutionDialogRequestID === requestID) hideCommandExecutionDialog();

  if (!result?.ok) {
    setCoordinationStatus(result?.error || 'СОСТОЯНИЕ КОМАНДЫ НЕ УДАЛОСЬ СОХРАНИТЬ', true);
  } else if (decision === 'approve') {
    setCoordinationStatus('КОМАНДА ВЫПОЛНЕНА И СОХРАНЕНА');
  } else {
    setCoordinationStatus('ЗАПРОС ОТКЛОНЁН');
  }
  renderCoordination();
  return result;
}

btnApproveCommandExecution.addEventListener('click', () => {
  void resolveCommandExecution('approve');
});
btnRejectCommandExecution.addEventListener('click', () => {
  void resolveCommandExecution('reject');
});
commandExecutionDialog.addEventListener('cancel', (event) => {
  event.preventDefault();
  void resolveCommandExecution('reject');
});

function setCoordinationStatus(message, isError = false) {
  coordinationStatus.textContent = isError ? '' : (message || '');
  coordinationStatus.classList.remove('err');
  coordinationError.textContent = isError ? (message || '') : '';
  coordinationError.hidden = !isError || !message;
}

function fillSelect(select, values, valueOf, labelOf, emptyLabel) {
  select.replaceChildren();
  for (const value of values) {
    const option = document.createElement('option');
    option.value = valueOf(value);
    option.textContent = labelOf(value);
    select.appendChild(option);
  }
  if (!values.length) {
    const option = document.createElement('option');
    option.value = '';
    option.textContent = emptyLabel;
    select.appendChild(option);
  }
}

function sessionLabel(session) {
  const character = session.character ? ` · ${session.character.name}` : '';
  return `${session.fallbackName}${character}`;
}

async function runCoordinationCommand(command, successMessage, pendingMessage) {
  if (coordinationCommandPending) return null;
  coordinationCommandPending = true;
  setCoordinationStatus(pendingMessage || 'ВЫПОЛНЕНИЕ ОПЕРАЦИИ...');
  renderCoordination();
  renderHackStatus();
  let result;
  try {
    result = await command();
  } catch (error) {
    result = { ok: false, error: error instanceof Error ? error.message : String(error) };
  }
  coordinationCommandPending = false;
  renderHackStatus();
  if (!result?.ok) {
    if (result?.state) applyCoordinationState(result.state);
    setCoordinationStatus(result?.error || 'ОПЕРАЦИЯ ОТКЛОНЕНА', true);
    renderCoordination();
    return result;
  }
  if (result.state) applyCoordinationState(result.state);
  setCoordinationStatus(successMessage || 'ОПЕРАЦИЯ ВЫПОЛНЕНА');
  renderCoordination();
  return result;
}

function showTerminalSwitchDecision(result) {
  pendingTerminalSwitch = result?.switchId || null;
  if (!pendingTerminalSwitch) return;
  terminalSwitchStatus.textContent = 'ИСХОДНЫЙ ТЕРМИНАЛ ОСТАЁТСЯ АКТИВНЫМ ДО ВЫБОРА';
  terminalSwitchError.textContent = '';
  terminalSwitchError.hidden = true;
  terminalSwitchDialog.hidden = false;
  if (typeof terminalSwitchDialog.showModal === 'function' && !terminalSwitchDialog.open) {
    terminalSwitchDialog.showModal();
  } else {
    terminalSwitchDialog.setAttribute('open', '');
  }
  terminalSwitchButtons[0]?.focus();
}

function hideTerminalSwitchDecision() {
  pendingTerminalSwitch = null;
  terminalSwitchDialog.hidden = true;
  if (typeof terminalSwitchDialog.close === 'function' && terminalSwitchDialog.open) {
    terminalSwitchDialog.close();
  } else {
    terminalSwitchDialog.removeAttribute('open');
  }
}

function showEndBroadcastConfirmation() {
  endBroadcastDialog.hidden = false;
  if (typeof endBroadcastDialog.showModal === 'function' && !endBroadcastDialog.open) {
    endBroadcastDialog.showModal();
  } else {
    endBroadcastDialog.setAttribute('open', '');
  }
  btnCancelEndBroadcast.focus();
}

function hideEndBroadcastConfirmation({ restoreFocus = true } = {}) {
  endBroadcastDialog.hidden = true;
  if (typeof endBroadcastDialog.close === 'function' && endBroadcastDialog.open) {
    endBroadcastDialog.close();
  } else {
    endBroadcastDialog.removeAttribute('open');
  }
  btnCancelEndBroadcast.disabled = false;
  btnConfirmEndBroadcast.disabled = false;
  if (restoreFocus && !btnEndBroadcast.hidden) btnEndBroadcast.focus();
}

async function runTerminalSwitchRequest(command, completedMessage, pendingMessage) {
  const result = await runCoordinationCommand(command, completedMessage, pendingMessage);
  if (result?.ok && result.status === 'decision-required' && result.switchId) {
    showTerminalSwitchDecision(result);
  }
  return result;
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
      if (state.liveTerminalId === term.id || state.coordination?.pendingSwitch?.sourceTerminalId === term.id) {
        setCoordinationStatus('АКТИВНЫЙ ИЛИ СОХРАНЁННЫЙ ТЕРМИНАЛ НЕЛЬЗЯ УДАЛИТЬ', true);
        return;
      }
      if (!window.confirm(`Удалить терминал "${term.name}" целиком?`)) return;
      const idx = state.session.terminals.findIndex(t => t.id === term.id);
      if (idx >= 0) state.session.terminals.splice(idx, 1);
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
  const broadcastActive = Boolean(state.coordination?.broadcast);
  liveFlag.style.display   = isLive ? '' : 'none';
  btnMakeLive.textContent  = isLive ? 'ОБНОВИТЬ АКТИВНЫЙ' : 'СДЕЛАТЬ АКТИВНЫМ';
  btnMakeLive.disabled     = !term || !broadcastActive || coordinationCommandPending;
  btnPublish.style.display = isLive ? '' : 'none';
  btnPublish.disabled = !isLive || coordinationCommandPending;
  btnStopBroadcast.disabled = !broadcastActive || !state.liveTerminalId || coordinationCommandPending;
}

// ── Render: per-terminal settings (hack level / intro text) ──
function renderSettingsPanel() {
  const term = getEditTerminal();
  hackLevelSelect.disabled = !term;
  introTextArea.disabled   = !term;
  btnApplySettings.disabled = !term;
  hackLevelSelect.value = term ? String(term.hackLevel || 0) : '0';
  introTextArea.value   = term ? (term.introText || '') : '';
  const completedCount = term?.commandStates && typeof term.commandStates === 'object'
    ? Object.keys(term.commandStates).length
    : 0;
  commandStateActions.hidden = !term;
  btnResetTerminalCommandStates.disabled = !term || completedCount === 0 || sessionStateCommandPending;
}

btnResetTerminalCommandStates.addEventListener('click', () => {
  const term = getEditTerminal();
  if (!term || sessionStateCommandPending) return;
  if (!window.confirm(`Сбросить все выполненные состояния команд терминала "${term.name}"?`)) return;
  runSessionStateCommand(
    () => desktopAPI.resetTerminalCommandStates({ terminalId: term.id }),
    'СОСТОЯНИЯ КОМАНД ТЕРМИНАЛА СБРОШЕНЫ'
  );
});

// ── Render: live hack status (term panel footer) ──────────────
function renderHackStatus() {
  const liveTerm = state.session && state.liveTerminalId
    ? state.session.terminals.find(t => t.id === state.liveTerminalId)
    : null;

  if (!liveTerm || !liveTerm.hackLevel || !state.liveHack) {
    hackStatus.style.display = 'none';
    btnResetFailedHack.hidden = true;
    return;
  }

  hackStatus.style.display = '';
  const h = state.liveHack;
  if (h.solved) {
    hackStatusLine.textContent = 'ВЗЛОМ: ПРОЙДЕН';
  } else if (h.failed) {
    hackStatusLine.textContent = 'ВЗЛОМ: ЗАБЛОКИРОВАН';
  } else {
    hackStatusLine.textContent = `ВЗЛОМ: осталось попыток ${h.attemptsLeft}/${h.attemptsMax}`;
  }
  btnHackSuccess.disabled = h.solved || h.failed;
  btnHackSuccess.hidden = h.failed;
  btnResetFailedHack.hidden = !h.failed;
  btnResetFailedHack.disabled = !h.failed || coordinationCommandPending;
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
  const term = getEditTerminal();
  const wrap = document.createElement('div');
  wrap.className = 'tree-node';

  const row = document.createElement('div');
  const completed = commandExecutionState(term, node.id);
  row.className = 'tree-row'
    + (state.selectedNodeId === node.id ? ' selected' : '')
    + (completed ? ' command-completed' : '');

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
  label.textContent = isRoot ? 'ROOT' : effectiveNodeName(term, node);
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
  const snapshot = commandExecutionState(term, node.id);
  let html = `<div class="node-type-label">${typeLabel}</div>
    <label class="field-label" for="fldName">${node.type === 'command' ? 'ИСХОДНОЕ НАЗВАНИЕ' : 'НАЗВАНИЕ'}</label>
    <input class="field-input" id="fldName" value="${escAttr(node.name)}">`;

  if (node.type === 'command') {
    html += `
      <label class="state-change-toggle" for="fldStateChangeEnabled">
        <input id="fldStateChangeEnabled" type="checkbox"${node.stateChange ? ' checked' : ''}${snapshot ? ' disabled' : ''}>
        <span>ИЗМЕНЯЕТ СОСТОЯНИЕ</span>
      </label>
      ${snapshot ? '<div class="state-change-toggle-hint">Сначала сбросьте выполненное состояние, чтобы отключить настройку.</div>' : ''}
      <div class="state-change-fields" id="stateChangeFields"${node.stateChange ? '' : ' hidden'}>
        <label class="field-label" for="fldCompletedName">НАЗВАНИЕ ПОСЛЕ ВЫПОЛНЕНИЯ</label>
        <input class="field-input" id="fldCompletedName" value="${escAttr(node.stateChange?.completedName || '')}">
        <label class="field-label" for="fldConfirmationText">ТЕКСТ ЗАПРОСА ПОДТВЕРЖДЕНИЯ</label>
        <textarea class="field-textarea state-change-textarea" id="fldConfirmationText">${escHtml(node.stateChange?.confirmationText || '')}</textarea>
      </div>
      <label class="field-label" for="fldText">ТЕКСТ УСПЕШНОГО ВЫПОЛНЕНИЯ</label>
      <textarea class="field-textarea" id="fldText">${escHtml(node.text || '')}</textarea>`;
    if (snapshot) {
      html += `
        <div class="command-execution-snapshot" role="status" aria-label="СОХРАНЁННОЕ СОСТОЯНИЕ КОМАНДЫ">
          <div class="command-execution-heading">ВЫПОЛНЕНО</div>
          <div class="command-execution-label">ЗАФИКСИРОВАННОЕ НАЗВАНИЕ</div>
          <div class="command-execution-value">${escHtml(snapshot.completedName || '')}</div>
          <div class="command-execution-label">ЗАФИКСИРОВАННЫЙ РЕЗУЛЬТАТ</div>
          <div class="command-execution-value command-execution-result">${escHtml(snapshot.resultText || '')}</div>
        </div>`;
    }
  } else if (node.type === 'entry') {
    html += `<label class="field-label" for="fldText">ОПИСАНИЕ ЗАПИСИ</label>
      <textarea class="field-textarea" id="fldText">${escHtml(node.description || '')}</textarea>`;
  } else if (node.type === 'folder') {
    const count = node.children ? node.children.length : 0;
    html += `<div class="field-label">СОДЕРЖИМОЕ</div><div class="node-empty">${count} элемент(ов)</div>`;
  }

  html += '<div class="node-validation-error" id="nodeValidationError" role="alert" hidden></div>';
  html += `<div class="node-actions">
      <button class="btn btn-primary" id="btnApplyNode">ПРИМЕНИТЬ</button>
      ${snapshot ? '<button class="btn btn-secondary" id="btnResetCommandState" type="button">СБРОСИТЬ СОСТОЯНИЕ</button>' : ''}
      <button class="btn btn-danger" id="btnDeleteNode">УДАЛИТЬ</button>
    </div>`;

  nodeForm.innerHTML = html;

  const validationError = document.getElementById('nodeValidationError');
  const showValidationError = (message, field) => {
    validationError.textContent = message;
    validationError.hidden = false;
    field?.focus();
  };

  if (node.type === 'command') {
    const enabled = document.getElementById('fldStateChangeEnabled');
    const fields = document.getElementById('stateChangeFields');
    enabled.addEventListener('change', () => {
      fields.hidden = !enabled.checked;
      validationError.hidden = true;
      validationError.textContent = '';
    });
  }

  document.getElementById('btnApplyNode').addEventListener('click', () => {
    const nameEl = document.getElementById('fldName');
    const name = nameEl.value.trim();
    if (!name) {
      showValidationError(
        node.type === 'command' ? 'УКАЖИТЕ ИСХОДНОЕ НАЗВАНИЕ КОМАНДЫ' : 'УКАЖИТЕ НАЗВАНИЕ',
        nameEl
      );
      return;
    }

    if (node.type === 'command') {
      const stateChangeEnabled = document.getElementById('fldStateChangeEnabled').checked;
      const textEl = document.getElementById('fldText');
      if (stateChangeEnabled) {
        const completedNameEl = document.getElementById('fldCompletedName');
        const confirmationTextEl = document.getElementById('fldConfirmationText');
        if (!completedNameEl.value.trim()) {
          showValidationError('УКАЖИТЕ НАЗВАНИЕ ПОСЛЕ ВЫПОЛНЕНИЯ', completedNameEl);
          return;
        }
        if (!confirmationTextEl.value.trim()) {
          showValidationError('УКАЖИТЕ ТЕКСТ ЗАПРОСА ПОДТВЕРЖДЕНИЯ', confirmationTextEl);
          return;
        }
        if (!textEl.value.trim()) {
          showValidationError('УКАЖИТЕ ТЕКСТ УСПЕШНОГО РЕЗУЛЬТАТА', textEl);
          return;
        }
        node.stateChange = {
          completedName: completedNameEl.value,
          confirmationText: confirmationTextEl.value,
        };
      } else {
        delete node.stateChange;
      }
      node.text = textEl.value;
    }

    node.name = name;
    if (node.type === 'entry')   node.description = document.getElementById('fldText').value;
    autosave();
    renderTree();
    renderNodeForm();
    renderToolbarHint();
  });

  const btnResetCommandState = document.getElementById('btnResetCommandState');
  if (btnResetCommandState) {
    btnResetCommandState.disabled = sessionStateCommandPending;
    btnResetCommandState.addEventListener('click', () => {
      if (sessionStateCommandPending) return;
      const displayedName = snapshot?.completedName || node.name;
      if (!window.confirm(`Сбросить выполненное состояние команды "${displayedName}"?`)) return;
      runSessionStateCommand(
        () => desktopAPI.resetCommandState({ terminalId: term.id, commandId: node.id }),
        'СОСТОЯНИЕ КОМАНДЫ СБРОШЕНО'
      );
    });
  }

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

// ── Player roster and broadcast management ──────────────────
btnAddCharacter.addEventListener('click', async () => {
  if (coordinationCommandPending || !state.coordination?.playerConfig) return;
  const name = characterNameInput.value.trim();
  if (!name) {
    setCoordinationStatus('УКАЖИТЕ ИМЯ ПЕРСОНАЖА', true);
    return;
  }

  coordinationCommandPending = true;
  setCoordinationStatus('ДОБАВЛЕНИЕ ПЕРСОНАЖА...');
  renderCoordination();
  const result = await desktopAPI.addCharacter(name);
  coordinationCommandPending = false;
  if (!result?.ok) {
    setCoordinationStatus(result?.error || 'НЕ УДАЛОСЬ ДОБАВИТЬ ПЕРСОНАЖА', true);
    renderCoordination();
    return;
  }

  state.coordination = result.state || state.coordination;
  characterNameInput.value = '';
  setCoordinationStatus('ПЕРСОНАЖ ДОБАВЛЕН');
  renderCoordination();
});

characterNameInput.addEventListener('keydown', (event) => {
  if (event.key === 'Enter') {
    event.preventDefault();
    btnAddCharacter.click();
  }
});

btnStartBroadcast.addEventListener('click', async () => {
  if (coordinationCommandPending || state.coordination?.broadcast || !state.coordination?.playerConfig) return;
  coordinationCommandPending = true;
  setCoordinationStatus('ЗАПУСК ТРАНСЛЯЦИИ...');
  renderCoordination();
  const result = await desktopAPI.startBroadcast();
  coordinationCommandPending = false;
  if (!result?.ok) {
    setCoordinationStatus(result?.error || 'НЕ УДАЛОСЬ ЗАПУСТИТЬ ТРАНСЛЯЦИЮ', true);
    renderCoordination();
    return;
  }

  state.coordination = result.state || state.coordination;
  setCoordinationStatus('ТРАНСЛЯЦИЯ ЗАПУЩЕНА');
  renderCoordination();
  renderTreeHeader();
});

btnEndBroadcast.addEventListener('click', () => {
  if (coordinationCommandPending || !state.coordination?.broadcast) return;
  showEndBroadcastConfirmation();
});

btnCancelEndBroadcast.addEventListener('click', () => hideEndBroadcastConfirmation());
endBroadcastDialog.addEventListener('cancel', (event) => {
  event.preventDefault();
  if (!coordinationCommandPending) hideEndBroadcastConfirmation();
});

btnConfirmEndBroadcast.addEventListener('click', async () => {
  if (coordinationCommandPending || !state.coordination?.broadcast) return;
  btnCancelEndBroadcast.disabled = true;
  btnConfirmEndBroadcast.disabled = true;
  const result = await runCoordinationCommand(
    () => desktopAPI.endBroadcast(),
    'ТРАНСЛЯЦИЯ ЗАВЕРШЕНА · СЕССИИ И ПЕРСОНАЖИ СОХРАНЕНЫ',
    'ЗАВЕРШЕНИЕ ТРАНСЛЯЦИИ...'
  );
  if (!result?.ok) {
    hideEndBroadcastConfirmation();
    return;
  }
  if (!result.state || result.state.broadcast) {
    setCoordinationStatus('ЗАВЕРШЕНИЕ НЕ ПОДТВЕРЖДЕНО АВТОРИТЕТНЫМ СОСТОЯНИЕМ', true);
    renderCoordination();
    hideEndBroadcastConfirmation();
    return;
  }
  hideEndBroadcastConfirmation({ restoreFocus: false });
  hideTerminalSwitchDecision();
  state.liveHack = null;
  renderAll();
  btnStartBroadcast.focus();
});

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

btnApplySettings.addEventListener('click', async () => {
  const term = getEditTerminal();
  if (!term) return;
  term.hackLevel = Number(hackLevelSelect.value) || 0;
  term.introText = introTextArea.value;
  autosave();
  if (term.id === state.liveTerminalId) {
    // Intro text can refresh live immediately; hackLevel only takes effect
    // on the next (re)broadcast so it never disrupts an in-progress hack.
    await runCoordinationCommand(
      () => desktopAPI.updateLiveTerminal({ tree: term.root, introText: term.introText }),
      'АКТИВНЫЙ ТЕРМИНАЛ ОБНОВЛЁН',
      'ОБНОВЛЕНИЕ АКТИВНОГО ТЕРМИНАЛА...'
    );
  }
});

btnMakeLive.addEventListener('click', async () => {
  const term = getEditTerminal();
  if (!term || !state.coordination?.broadcast) return;
  const result = await runTerminalSwitchRequest(
    () => desktopAPI.requestTerminalActivation({
      terminalId: term.id,
      terminalName: term.name,
      tree: term.root,
      hackLevel: term.hackLevel || 0,
      introText: term.introText || '',
    }),
    'АКТИВНЫЙ ТЕРМИНАЛ ВЫБРАН',
    'ПЕРЕКЛЮЧЕНИЕ АКТИВНОГО ТЕРМИНАЛА...'
  );
  if (result?.ok && result.status === 'activated') state.liveHack = null;
  renderTermList();
  renderTreeHeader();
  renderHackStatus();
});

btnPublish.addEventListener('click', async () => {
  const term = getEditTerminal();
  if (!term || term.id !== state.liveTerminalId) return;
  const result = await runTerminalSwitchRequest(
    () => desktopAPI.updateLiveTerminal({ tree: term.root, introText: term.introText || '' }),
    'АКТИВНЫЙ ТЕРМИНАЛ ОБНОВЛЁН',
    'ПУБЛИКАЦИЯ ОБНОВЛЕНИЯ...'
  );
  if (!result?.ok) return;
  const original = btnPublish.textContent;
  btnPublish.textContent = 'ОБНОВЛЕНО ✓';
  setTimeout(() => { btnPublish.textContent = original; }, 1200);
});

btnStopBroadcast.addEventListener('click', async () => {
  if (!state.coordination?.broadcast || !state.liveTerminalId) return;
  const result = await runCoordinationCommand(
    () => desktopAPI.requestTerminalClear(),
    'АКТИВНЫЙ ТЕРМИНАЛ УБРАН · ТРАНСЛЯЦИЯ ПРОДОЛЖАЕТСЯ',
    'ОЧИСТКА АКТИВНОГО ТЕРМИНАЛА...'
  );
  if (result?.ok && result.status === 'cleared') state.liveHack = null;
  renderTermList();
  renderTreeHeader();
  renderHackStatus();
});

for (const button of terminalSwitchButtons) {
  button.addEventListener('click', async () => {
    if (!pendingTerminalSwitch || coordinationCommandPending) return;
    const decision = button.dataset.switchDecision;
    terminalSwitchButtons.forEach(control => { control.disabled = true; });
    terminalSwitchStatus.textContent = 'ПРИМЕНЕНИЕ РЕШЕНИЯ...';
    terminalSwitchError.hidden = true;
    const result = await runCoordinationCommand(
      () => desktopAPI.resolveTerminalSwitch({ switchId: pendingTerminalSwitch, decision }),
      decision === 'cancel' ? 'ПЕРЕКЛЮЧЕНИЕ ОТМЕНЕНО' : 'РЕШЕНИЕ ПРИМЕНЕНО',
      'ПРИМЕНЕНИЕ РЕШЕНИЯ...'
    );
    terminalSwitchButtons.forEach(control => { control.disabled = false; });
    if (!result?.ok) {
      terminalSwitchError.textContent = result?.error || 'РЕШЕНИЕ ОТКЛОНЕНО';
      terminalSwitchError.hidden = false;
      terminalSwitchStatus.textContent = 'ИСХОДНЫЙ ТЕРМИНАЛ ОСТАЁТСЯ АКТИВНЫМ';
      return;
    }
    if (result.status === 'activated' || result.status === 'cleared') state.liveHack = null;
    hideTerminalSwitchDecision();
    renderAll();
  });
}

btnHackSuccess.addEventListener('click', () => {
  if (!state.liveHack || state.liveHack.solved || state.liveHack.failed) return;
  desktopAPI.forceHackSuccess();
});

btnResetFailedHack.addEventListener('click', async () => {
  const term = state.session && state.liveTerminalId
    ? state.session.terminals.find(candidate => candidate.id === state.liveTerminalId)
    : null;
  if (!term || !state.liveHack?.failed || coordinationCommandPending) return;
  const result = await runCoordinationCommand(
    () => desktopAPI.resetFailedHack({
      terminalId: term.id,
      terminalName: term.name,
      tree: term.root,
      hackLevel: term.hackLevel || 0,
      introText: term.introText || '',
    }),
    'СОЗДАНА НОВАЯ ГОЛОВОЛОМКА',
    'ПОДГОТОВКА НОВОЙ ГОЛОВОЛОМКИ...'
  );
  if (!result?.ok) renderHackStatus();
});
