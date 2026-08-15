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
};

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

function persistPublicAccess() {
  try {
    const serialized = JSON.stringify(state.publicAccess);
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

globalThis.__desktopFixture = {
  calls: state.calls,
  timeline: state.calls,
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
    let active = true;
    return () => {
      if (!active) return;
      active = false;
      listeners.delete(callback);
      state.releases.set(name, (state.releases.get(name) ?? 0) + 1);
    };
  },
};

export function GetRuntimeStatus() {
  state.calls.push({ method: 'GetRuntimeStatus', args: [] });
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
	  persistPublicAccess();
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
export const OpenSession = (...args) => record('OpenSession', args);
export const OpenURL = (...args) => record('OpenURL', args);
export const ReleaseCharacter = (...args) => record('ReleaseCharacter', args);
export const RenameCharacter = (...args) => record('RenameCharacter', args);
export const RenameLogicalSession = (...args) => record('RenameLogicalSession', args);
export const RequestTerminalActivation = (...args) => record('RequestTerminalActivation', args);
export const RequestTerminalClear = (...args) => record('RequestTerminalClear', args);
export const ResetFailedHack = (...args) => record('ResetFailedHack', args);
export const ResolveTerminalSwitch = (...args) => record('ResolveTerminalSwitch', args);
export const SaveSession = (...args) => record('SaveSession', args);
export const SetActiveController = (...args) => record('SetActiveController', args);
export const StartBroadcast = (...args) => record('StartBroadcast', args);
export const UpdateLiveTerminal = (...args) => record('UpdateLiveTerminal', args);
