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
};

function record(method, args) {
  state.calls.push({ method, args });
  return Promise.resolve({ ok: true, method, args });
}

globalThis.__desktopFixture = {
  calls: state.calls,
  timeline: state.calls,
  emit(name, data) {
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
  releaseCount(name) { return state.releases.get(name) ?? 0; },
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
