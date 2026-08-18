function stateChangingAuthoringSession() {
  return {
    version: 1,
    name: 'State-changing authoring fixture',
    terminals: [{
      id: 'terminal-stateful',
      name: 'Терминал охраны',
      hackLevel: 0,
      introText: '',
      root: {
        id: 'root',
        type: 'folder',
        name: 'ROOT',
        children: [
          {
            id: 'emergency-lights',
            type: 'command',
            name: 'Включить аварийный свет',
            text: 'Аварийное освещение включено.',
          },
          {
            id: 'doors',
            type: 'command',
            name: 'Открыть двери',
            text: 'Новая редакция результата открытия.',
            stateChange: {
              completedName: 'Двери разблокированы',
              confirmationText: 'Открыть двери?',
            },
          },
          {
            id: 'alarm',
            type: 'command',
            name: 'Включить тревогу',
            text: 'Сигнал тревоги активирован.',
            stateChange: {
              completedName: 'Сигнал тревоги активен',
              confirmationText: 'Включить тревогу?',
            },
          },
        ],
      },
      commandStates: {
        doors: {
          completedName: 'Двери открыты',
          resultText: 'Доступ в сектор разрешён.',
        },
        alarm: {
          completedName: 'Тревога включена',
          resultText: 'Охрана сектора предупреждена.',
        },
      },
    }],
  };
}

const state = globalThis.__desktopFixtureState ??= {
  calls: [],
  listeners: new Map(),
  releases: new Map(),
  status: {
    serverInfo: { url: 'http://127.0.0.1:3690', localUrl: '', tunnel: false, port: 3690 },
    clientCount: 1,
    hackState: null,
    coordinationState: null,
  },
  statusPromise: null,
  resolveStatus: null,
  publicAccess: {
    preferences: { version: 1, enabledPreference: false, reservedDomain: '', username: 'players', revision: 0 },
    providerTokenPresence: 'absent',
    playerPasswordPresence: 'absent',
    status: { state: 'disabled', generation: 0, settingsRevision: 0 },
  },
  publicAccessPromise: null,
  resolvePublicAccess: null,
  savePublicAccessPromise: null,
  resolveSavePublicAccess: null,
  pendingSavePublicAccess: null,
  clipboardText: '',
  authoringSession: stateChangingAuthoringSession(),
  authoringRevision: 1,
};
if (!state.authoringSession) state.authoringSession = stateChangingAuthoringSession();
if (!Number.isSafeInteger(state.authoringRevision)) state.authoringRevision = 1;
try {
  const durableAuthoring = JSON.parse(globalThis.localStorage?.getItem('fallout-fixture-authoring-session') ?? 'null');
  if (durableAuthoring?.session && Number.isSafeInteger(durableAuthoring.revision)) {
    state.authoringSession = durableAuthoring.session;
    state.authoringRevision = durableAuthoring.revision;
  }
} catch {
  // A fresh fixture remains available when browser storage is unavailable.
}

function persistAuthoringState() {
  try {
    globalThis.localStorage?.setItem('fallout-fixture-authoring-session', JSON.stringify({
      session: state.authoringSession,
      revision: state.authoringRevision,
    }));
  } catch {
    // The in-memory authoring fixture remains authoritative for this page.
  }
}

const durablePublicAccess = (() => {
  try {
    if (globalThis.name?.startsWith('fallout-fixture-public-access:')) {
      return JSON.parse(globalThis.name.slice('fallout-fixture-public-access:'.length));
    }
    const raw = globalThis.localStorage?.getItem('fallout-fixture-public-access');
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
})();
if (durablePublicAccess?.preferences) state.publicAccess = durablePublicAccess;

function persistPublicAccess({ preserveVisiblePreferences = false } = {}) {
  try {
    const durable = structuredClone(state.publicAccess);
    if (preserveVisiblePreferences) {
      const priorRaw = globalThis.localStorage?.getItem('fallout-fixture-public-access');
      const prior = priorRaw ? JSON.parse(priorRaw) : null;
      durable.preferences.reservedDomain = prior?.preferences?.reservedDomain ?? '';
      durable.preferences.username = prior?.preferences?.username ?? 'players';
    }
    const serialized = JSON.stringify(durable);
    globalThis.name = `fallout-fixture-public-access:${serialized}`;
    globalThis.localStorage?.setItem('fallout-fixture-public-access', serialized);
  } catch {
    // The fixture remains usable when browser storage is unavailable.
  }
}

function record(method, args) {
  state.calls.push({ method, args });
  return Promise.resolve({ ok: true, method, args });
}

function authoringFixtureActive() {
  return globalThis.location?.pathname === '/__fixture/state-changing-command-authoring';
}

const authoringFixtureBase = '/__fixture/state-changing-command-authoring';

async function authoringFixtureCommand(path, payload) {
  const response = await fetch(`${authoringFixtureBase}/${path}`, {
    method: payload === undefined ? 'GET' : 'POST',
    headers: payload === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: payload === undefined ? undefined : JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(`authoring fixture ${path} failed`);
  const result = await response.json();
  if (result?.session) {
    state.authoringSession = structuredClone(result.session);
    state.authoringRevision = Number(result.revision || state.authoringRevision);
  }
  return result;
}

function approvalFixtureActive() {
  return globalThis.location?.pathname === '/__fixture/state-changing-command-approval/master';
}

function syncFixtureActive() {
  return globalThis.location?.pathname === '/__fixture/state-changing-command-sync/master';
}

function stateChangingLifecycleBase() {
  if (approvalFixtureActive()) return '/__fixture/state-changing-command-approval';
  if (syncFixtureActive()) return '/__fixture/state-changing-command-sync';
  return '';
}

async function stateChangingCoordinationState() {
  const response = await fetch(`${stateChangingLifecycleBase()}/state`);
  if (!response.ok) throw new Error('state-changing coordination fixture is unavailable');
  return response.json();
}

function emitFixtureEvent(name, data) {
  for (const callback of state.listeners.get(name) ?? []) callback({ data: structuredClone(data) });
}

function authoringSessionResult() {
  return {
    ok: true,
    revision: state.authoringRevision,
    session: structuredClone(state.authoringSession),
  };
}

globalThis.__desktopFixture = {
  calls: state.calls,
  timeline: state.calls,
  async authoringDurableState() {
    if (authoringFixtureActive()) await authoringFixtureCommand('session');
    return {
      revision: state.authoringRevision,
      commandStates: structuredClone(state.authoringSession.terminals[0].commandStates ?? {}),
    };
  },
  emit(name, data) {
	if (name === 'public-access-status' && data?.preferences && data?.status) {
	  state.publicAccess = structuredClone(data);
	}
    for (const callback of state.listeners.get(name) ?? []) callback({ data });
  },
  deferStatus() {
    state.statusPromise = new Promise(resolve => { state.resolveStatus = resolve; });
  },
  resolveStatus(status = state.status) {
    state.resolveStatus?.(status);
    state.resolveStatus = null;
  },
  setStatus(status) { state.status = status; },
  deferPublicAccess() {
    state.publicAccessPromise = new Promise(resolve => { state.resolvePublicAccess = resolve; });
  },
  resolvePublicAccess(snapshot = state.publicAccess) {
    state.resolvePublicAccess?.(snapshot);
    state.resolvePublicAccess = null;
    state.publicAccessPromise = null;
  },
  releaseCount(name) { return state.releases.get(name) ?? 0; },
  takeClipboardText() {
    const value = state.clipboardText;
    state.clipboardText = '';
    return value;
  },
  deferSavePublicAccess() {
    state.savePublicAccessPromise = new Promise(resolve => { state.resolveSavePublicAccess = resolve; });
  },
  resolveSavePublicAccess(result = state.pendingSavePublicAccess) {
    if (result?.snapshot) {
      state.publicAccess = structuredClone(result.snapshot);
      persistPublicAccess();
    }
    state.resolveSavePublicAccess?.(result);
    state.resolveSavePublicAccess = null;
    state.savePublicAccessPromise = null;
    state.pendingSavePublicAccess = null;
  },
};

export const Events = {
  On(name, callback) {
    state.calls.push({ method: `event:on:${name}`, args: [] });
    const listeners = state.listeners.get(name) ?? new Set();
    listeners.add(callback);
    state.listeners.set(name, listeners);
    let coordinationPoll = null;
    if (stateChangingLifecycleBase() && name === 'coordination-state') {
      let lastProjection = '';
      const poll = async () => {
        try {
          const coordination = await stateChangingCoordinationState();
          const projection = JSON.stringify(coordination);
          if (projection === lastProjection) return;
          lastProjection = projection;
          callback({ data: coordination });
        } catch {
          // The next poll retries while the deterministic fixture is running.
        }
      };
      void poll();
      coordinationPoll = setInterval(() => { void poll(); }, 25);
    }
    let active = true;
    return () => {
      if (!active) return;
      active = false;
      if (coordinationPoll !== null) clearInterval(coordinationPoll);
      listeners.delete(callback);
      state.releases.set(name, (state.releases.get(name) ?? 0) + 1);
    };
  },
};

export const Clipboard = {
  SetText(value) {
    state.clipboardText = typeof value === 'string' ? value : '';
    return Promise.resolve();
  },
};

export function GetRuntimeStatus() {
  state.calls.push({ method: 'GetRuntimeStatus', args: [] });
  if (stateChangingLifecycleBase()) {
    return stateChangingCoordinationState().then(coordinationState => ({
      ...state.status,
      coordinationState,
    }));
  }
  return state.statusPromise ?? Promise.resolve(state.status);
}

function snapshot() {
  return structuredClone(state.publicAccess);
}

export function GetPublicAccess() {
  state.calls.push({ method: 'GetPublicAccess', args: [] });
  return state.publicAccessPromise ?? Promise.resolve(snapshot());
}

export function SavePublicAccessSettings(request) {
  const proposed = request && typeof request === 'object' ? request : {};
  const providerReplacement = proposed.replacementProviderToken;
  const passwordReplacement = proposed.replacementPlayerPassword;
  const retained = {
    expectedRevision: proposed.expectedRevision,
    enabledPreference: proposed.enabledPreference,
    reservedDomain: proposed.reservedDomain,
    username: proposed.username,
    replacementProviderToken: '',
    deleteProviderToken: proposed.deleteProviderToken === true,
    replacementPlayerPassword: '',
    deletePlayerPassword: proposed.deletePlayerPassword === true,
  };
  state.calls.push({ method: 'SavePublicAccessSettings', args: [retained] });
  const revision = state.publicAccess.preferences.revision + 1;
  const nextPublicAccess = {
    preferences: {
      version: 1,
      enabledPreference: proposed.enabledPreference === true,
      reservedDomain: typeof proposed.reservedDomain === 'string' ? proposed.reservedDomain : '',
      username: typeof proposed.username === 'string' && proposed.username ? proposed.username : 'players',
      revision,
    },
    providerTokenPresence: proposed.deleteProviderToken ? 'absent' : (providerReplacement ? 'present' : state.publicAccess.providerTokenPresence),
    playerPasswordPresence: proposed.deletePlayerPassword ? 'absent' : (passwordReplacement ? 'present' : state.publicAccess.playerPasswordPresence),
    status: { state: 'disabled', generation: state.publicAccess.status.generation + 1, settingsRevision: revision },
  };
  const result = { ok: true, snapshot: structuredClone(nextPublicAccess) };
  if (state.savePublicAccessPromise) {
    state.pendingSavePublicAccess = result;
    return state.savePublicAccessPromise;
  }
  state.publicAccess = nextPublicAccess;
  persistPublicAccess();
  return Promise.resolve(result);
}

export function GeneratePlayerPassword(request) {
  state.calls.push({ method: 'GeneratePlayerPassword', args: [{ expectedRevision: request?.expectedRevision ?? 0 }] });
  const revision = state.publicAccess.preferences.revision + 1;
  state.publicAccess.preferences.revision = revision;
  state.publicAccess.playerPasswordPresence = 'present';
  state.publicAccess.status = { state: 'disabled', generation: state.publicAccess.status.generation + 1, settingsRevision: revision };
  persistPublicAccess({ preserveVisiblePreferences: true });
  return Promise.resolve({ ok: true, generatedPassword: 'synthetic-one-time-generated-value', settingsRevision: revision });
}

export function StartPublicAccess(request) {
  state.calls.push({ method: 'StartPublicAccess', args: [{ expectedRevision: request?.expectedRevision ?? 0 }] });
  state.publicAccess.status = {
    state: 'ready', generation: state.publicAccess.status.generation + 1,
    settingsRevision: state.publicAccess.preferences.revision, publicUrl: 'https://fixture.example',
  };
  return Promise.resolve({ ok: true, snapshot: snapshot() });
}

export function StopPublicAccess(request) {
  state.calls.push({ method: 'StopPublicAccess', args: [{ expectedRevision: request?.expectedRevision ?? 0 }] });
  state.publicAccess.status = {
    state: 'disabled', generation: state.publicAccess.status.generation + 1,
    settingsRevision: state.publicAccess.preferences.revision,
  };
  return Promise.resolve({ ok: true, snapshot: snapshot() });
}

export const AddCharacter = (...args) => record('AddCharacter', args);
export const AssignCharacter = (...args) => record('AssignCharacter', args);
export const CopyDemo = (...args) => record('CopyDemo', args);
export const DeleteCharacter = (...args) => record('DeleteCharacter', args);
export const EndBroadcast = (...args) => record('EndBroadcast', args);
export const ForceHackSuccess = (...args) => record('ForceHackSuccess', args);
export const LoadReferencedPlayerConfig = (...args) => record('LoadReferencedPlayerConfig', args);
export const MoveCharacter = (...args) => record('MoveCharacter', args);
export const NewPlayerConfig = (...args) => record('NewPlayerConfig', args);
export const NewSession = (...args) => record('NewSession', args);
export const OpenPlayerConfig = (...args) => record('OpenPlayerConfig', args);
export async function OpenSession(...args) {
  if (!authoringFixtureActive() && !stateChangingLifecycleBase()) return record('OpenSession', args);
  state.calls.push({ method: 'OpenSession', args: [] });
  if (syncFixtureActive()) {
    return fetch('/__fixture/state-changing-command-sync/session').then(async response => ({
      ok: response.ok,
      error: response.ok ? '' : 'synchronization session fixture is unavailable',
      filePath: '/private/tmp/fallout-state-changing-sync.json',
      session: response.ok ? await response.json() : null,
    }));
  }
  if (approvalFixtureActive()) {
    return fetch('/__fixture/state-changing-command-approval/session').then(async response => ({
      ok: response.ok,
      error: response.ok ? '' : 'approval session fixture is unavailable',
      filePath: '/private/tmp/fallout-state-changing-approval.json',
      session: response.ok ? await response.json() : null,
    }));
  }
  const result = await authoringFixtureCommand('session');
  return {
    ok: true,
    filePath: '/private/tmp/fallout-state-changing-authoring.json',
    session: structuredClone(result.session),
  };
}
export const OpenURL = (...args) => record('OpenURL', args);
export const ReleaseCharacter = (...args) => record('ReleaseCharacter', args);
export const RenameCharacter = (...args) => record('RenameCharacter', args);
export const RenameLogicalSession = (...args) => record('RenameLogicalSession', args);
export const RequestTerminalActivation = (...args) => record('RequestTerminalActivation', args);
export const RequestTerminalClear = (...args) => record('RequestTerminalClear', args);
export const ResetFailedHack = (...args) => record('ResetFailedHack', args);
export async function ResolveCommandExecution(payload) {
  const fixtureBase = stateChangingLifecycleBase();
  if (!fixtureBase) return record('ResolveCommandExecution', [payload]);
  const retained = structuredClone(payload ?? {});
  state.calls.push({ method: 'ResolveCommandExecution', args: [retained] });
  const response = await fetch(`${fixtureBase}/resolve`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(retained),
  });
  const result = await response.json();
  return result;
}
export async function ResetCommandState(payload) {
  if (!authoringFixtureActive()) return record('ResetCommandState', [payload]);
  const retained = structuredClone(payload ?? {});
  state.calls.push({ method: 'ResetCommandState', args: [retained] });
  const result = await authoringFixtureCommand('reset-command', retained);
  emitFixtureEvent('session-state', { revision: result.revision, session: result.session });
  return result;
}
export async function ResetTerminalCommandStates(payload) {
  if (!authoringFixtureActive()) return record('ResetTerminalCommandStates', [payload]);
  const retained = structuredClone(payload ?? {});
  state.calls.push({ method: 'ResetTerminalCommandStates', args: [retained] });
  const result = await authoringFixtureCommand('reset-terminal', retained);
  emitFixtureEvent('session-state', { revision: result.revision, session: result.session });
  return result;
}
export const ResolveTerminalSwitch = (...args) => record('ResolveTerminalSwitch', args);
export async function SaveSession(session) {
  if (!authoringFixtureActive()) return record('SaveSession', [session]);
  const retained = structuredClone(session);
  state.calls.push({ method: 'SaveSession', args: [retained] });
  const result = await authoringFixtureCommand('save', retained);
  return { ok: result.ok === true, error: result.error || '', savedRevision: result.revision };
}
export const SetActiveController = (...args) => record('SetActiveController', args);
export const StartBroadcast = (...args) => record('StartBroadcast', args);
export const UpdateLiveTerminal = (...args) => record('UpdateLiveTerminal', args);
