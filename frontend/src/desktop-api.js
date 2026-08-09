'use strict';

// Prefer Wails' generated App module when it exists. The glob deliberately
// tolerates a pre-generation checkout so `npm run build` remains usable while
// Wails still replaces the fallback with generated bindings in real builds.
const generatedAppModules = typeof import.meta.glob === 'function'
  ? import.meta.glob('../wailsjs/go/main/App.js', { eager: true })
  : {};
const generatedAppBindings = Object.values(generatedAppModules)[0];

const APP_METHODS = Object.freeze({
  getRuntimeStatus: 'GetRuntimeStatus',
  newSession: 'NewSession',
  openSession: 'OpenSession',
  saveSession: 'SaveSession',
  setLiveTerminal: 'SetLiveTerminal',
  updateLiveTerminal: 'UpdateLiveTerminal',
  clearLiveTerminal: 'ClearLiveTerminal',
  forceHackSuccess: 'ForceHackSuccess',
  openUrl: 'OpenURL',
});

const DISPOSE = Symbol.for('fallout-terminal.desktop-api.dispose');
const subscriptions = new Set();

function appBindings() {
  // Generated methods are thin wrappers around this exact narrow namespace;
  // the fallback exists only until the generator has populated frontend/wailsjs.
  const bindings = generatedAppBindings ?? window.go?.main?.App;
  if (!bindings) {
    throw new Error('Wails App bindings are unavailable');
  }
  return bindings;
}

function invoke(method, ...args) {
  try {
    const binding = appBindings()[method];
    if (typeof binding !== 'function') {
      throw new Error(`Wails App.${method} binding is unavailable`);
    }
    return Promise.resolve(binding(...args));
  } catch (error) {
    return Promise.reject(error);
  }
}

function command(method, ...args) {
  return invoke(method, ...args).catch((error) => ({
    ok: false,
    error: error instanceof Error ? error.message : String(error),
  }));
}

let runtimeStatusPromise;
function runtimeStatus() {
  runtimeStatusPromise ??= command(APP_METHODS.getRuntimeStatus);
  return runtimeStatusPromise;
}

let latestServerInfo = null;

function normalizeServerInfo(payload) {
  if (!payload || typeof payload !== 'object') return null;

  const url = typeof payload.url === 'string' ? payload.url : '';
  const tunnel = Boolean(payload.tunnel);
  const previousLocalUrl = latestServerInfo?.localUrl
    || (latestServerInfo && !latestServerInfo.tunnel ? latestServerInfo.url : '');
  const suppliedLocalUrl = typeof payload.localUrl === 'string' ? payload.localUrl : '';
  const localUrl = suppliedLocalUrl || (!tunnel ? url : previousLocalUrl);

  latestServerInfo = Object.freeze({
    ip: typeof payload.ip === 'string' ? payload.ip : '',
    port: Number.isInteger(payload.port) ? payload.port : 0,
    url,
    localUrl,
    tunnel,
    tunnelError: typeof payload.tunnelError === 'string' ? payload.tunnelError : '',
  });
  return latestServerInfo;
}

function subscribe(eventName, statusField, callback, project = (payload) => payload) {
  if (typeof callback !== 'function') {
    throw new TypeError(`${eventName} listener must be a function`);
  }

  let active = true;
  let eventReceived = false;
  let releaseRuntime = () => {};

  const listener = (payload) => {
    if (!active) return;
    const projected = project(payload);
    if (statusField === 'serverInfo' && projected == null) return;
    eventReceived = true;
    callback(projected);
  };

  const eventsOn = window.runtime?.EventsOn;
  if (typeof eventsOn === 'function') {
    const release = eventsOn(eventName, listener);
    if (typeof release === 'function') {
      releaseRuntime = release;
    }
  }

  // domReady can emit before this module is evaluated. Replaying the status
  // snapshot fills that gap, but never overwrites a newer event.
  void runtimeStatus().then((status) => {
    if (!active || eventReceived || !status || status.ok === false) return;
    const payload = status[statusField];
    if (statusField === 'serverInfo' && payload == null) return;
    const projected = project(payload);
    if (statusField === 'serverInfo' && projected == null) return;
    callback(projected);
  });

  const unsubscribe = () => {
    if (!active) return;
    active = false;
    subscriptions.delete(unsubscribe);
    releaseRuntime();
  };
  subscriptions.add(unsubscribe);
  return unsubscribe;
}

const previousFacade = window.desktopAPI;
if (typeof previousFacade?.[DISPOSE] === 'function') {
  previousFacade[DISPOSE]();
}

const desktopAPI = {
  onServerInfo: (callback) => subscribe('server-info', 'serverInfo', callback, normalizeServerInfo),
  onClientCount: (callback) => subscribe('client-count', 'clientCount', callback),
  onHackState: (callback) => subscribe('hack-state', 'hackState', callback),
  // Deliberately perform no privileged browser operation here. App.OpenURL
  // parses and validates the final HTTP(S) URL immediately before opening it.
  openUrl: (url) => command(APP_METHODS.openUrl, url),
  openSession: () => command(APP_METHODS.openSession),
  newSession: () => command(APP_METHODS.newSession),
  saveSession: (session) => command(APP_METHODS.saveSession, session),
  setLiveTerminal: (payload) => command(APP_METHODS.setLiveTerminal, payload),
  updateLiveTerminal: (payload) => command(APP_METHODS.updateLiveTerminal, payload),
  clearLiveTerminal: () => command(APP_METHODS.clearLiveTerminal),
  forceHackSuccess: () => command(APP_METHODS.forceHackSuccess),
};

Object.defineProperty(desktopAPI, DISPOSE, {
  value: () => {
    for (const unsubscribe of [...subscriptions]) unsubscribe();
  },
});

Object.defineProperty(window, 'desktopAPI', {
  value: Object.freeze(desktopAPI),
  configurable: true,
});
