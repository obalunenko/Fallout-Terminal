'use strict';

import { Events } from '@wailsio/runtime';
import * as desktopService from '../bindings/github.com/obalunenko/Fallout-Terminal/desktopservice.js';

const APP_METHODS = Object.freeze({
  getRuntimeStatus: desktopService.GetRuntimeStatus,
  newSession: desktopService.NewSession,
  openSession: desktopService.OpenSession,
  saveSession: desktopService.SaveSession,
  loadReferencedPlayerConfig: desktopService.LoadReferencedPlayerConfig,
  newPlayerConfig: desktopService.NewPlayerConfig,
  openPlayerConfig: desktopService.OpenPlayerConfig,
  requestTerminalActivation: desktopService.RequestTerminalActivation,
  updateLiveTerminal: desktopService.UpdateLiveTerminal,
  requestTerminalClear: desktopService.RequestTerminalClear,
  resolveTerminalSwitch: desktopService.ResolveTerminalSwitch,
  forceHackSuccess: desktopService.ForceHackSuccess,
  resetFailedHack: desktopService.ResetFailedHack,
  addCharacter: desktopService.AddCharacter,
  renameCharacter: desktopService.RenameCharacter,
  deleteCharacter: desktopService.DeleteCharacter,
  renameLogicalSession: desktopService.RenameLogicalSession,
  assignCharacter: desktopService.AssignCharacter,
  releaseCharacter: desktopService.ReleaseCharacter,
  moveCharacter: desktopService.MoveCharacter,
  setActiveController: desktopService.SetActiveController,
  startBroadcast: desktopService.StartBroadcast,
  endBroadcast: desktopService.EndBroadcast,
  openUrl: desktopService.OpenURL,
});

const DISPOSE = Symbol.for('fallout-terminal.desktop-api.dispose');
const subscriptions = new Set();
const eventSubscriptions = new Map();
const requiredEvents = Object.freeze([
  ['server-info', 'serverInfo'],
  ['client-count', 'clientCount'],
  ['hack-state', 'hackState'],
  ['coordination-state', 'coordinationState'],
]);

function invoke(binding, ...args) {
  try {
    if (typeof binding !== 'function') throw new Error('Wails desktop binding is unavailable');
    return Promise.resolve(binding(...args));
  } catch (error) {
    return Promise.reject(error);
  }
}

function command(binding, ...args) {
  return invoke(binding, ...args).catch((error) => ({
    ok: false,
    error: error instanceof Error ? error.message : String(error),
  }));
}

const TERMINAL_SWITCH_STATUSES = new Set(['activated', 'cleared', 'decision-required', 'cancelled']);

function normalizeSwitchCommandResult(result) {
  const value = result && typeof result === 'object' ? result : {};
  const ok = value.ok === true;
  const status = typeof value.status === 'string' && TERMINAL_SWITCH_STATUSES.has(value.status)
    ? value.status
    : '';
  const switchId = typeof value.switchId === 'string' ? value.switchId : '';
  const state = value.state && typeof value.state === 'object' ? value.state : null;
  let error = typeof value.error === 'string' ? value.error : '';
  if (!ok && !error) error = 'Terminal switch command failed';
  return Object.freeze({ ok, error, status, switchId, state });
}

function switchCommand(binding, ...args) {
  return command(binding, ...args).then(normalizeSwitchCommandResult);
}

function normalizePlayerConfigResult(result) {
  const value = result && typeof result === 'object' ? result : {};
  const ok = value.ok === true;
  const canceled = value.canceled === true;
  const config = value.playerConfig && typeof value.playerConfig === 'object' ? value.playerConfig : null;
  const session = value.session && typeof value.session === 'object' ? value.session : null;
  const state = value.state && typeof value.state === 'object' ? value.state : null;
  let error = typeof value.error === 'string' ? value.error : '';
  if (!ok && !canceled && !error) error = 'Player config command failed';
  return Object.freeze({ ok, canceled, error, config, session, state });
}

function playerConfigCommand(binding, ...args) {
  return command(binding, ...args).then(normalizePlayerConfigResult);
}

let latestServerInfo = null;

function normalizeServerInfo(payload) {
  if (!payload || typeof payload !== 'object') return null;
  const url = typeof payload.url === 'string' ? payload.url : '';
  const tunnel = Boolean(payload.tunnel);
  const previousLocalUrl = latestServerInfo?.localUrl
    || (latestServerInfo && !latestServerInfo.tunnel ? latestServerInfo.url : '');
  const suppliedLocalUrl = typeof payload.localUrl === 'string' ? payload.localUrl : '';
  latestServerInfo = Object.freeze({
    ip: typeof payload.ip === 'string' ? payload.ip : '',
    port: Number.isInteger(payload.port) ? payload.port : 0,
    url,
    localUrl: suppliedLocalUrl || (!tunnel ? url : previousLocalUrl),
    tunnel,
    tunnelError: typeof payload.tunnelError === 'string' ? payload.tunnelError : '',
  });
  return latestServerInfo;
}

function unwrapEvent(event) {
  return event && typeof event === 'object' && Object.hasOwn(event, 'data') ? event.data : event;
}

let runtimeStatusPromise = null;

function beginStatusSnapshotWhenReady() {
  if (runtimeStatusPromise || !requiredEvents.every(([name]) => eventSubscriptions.has(name))) return;
  runtimeStatusPromise = command(APP_METHODS.getRuntimeStatus);
  void runtimeStatusPromise.then((status) => {
    if (!status || status.ok === false) return;
    for (const [eventName, field] of requiredEvents) {
      for (const subscription of eventSubscriptions.get(eventName) ?? []) {
        if (!subscription.active || subscription.eventReceived) continue;
        subscription.deliver(status[field]);
      }
    }
  });
}

function subscribe(eventName, statusField, callback, project = (payload) => payload) {
  if (typeof callback !== 'function') throw new TypeError(`${eventName} listener must be a function`);

  const bucket = eventSubscriptions.get(eventName) ?? new Set();
  eventSubscriptions.set(eventName, bucket);
  const subscription = {
    active: true,
    eventReceived: false,
    released: false,
    deliver(payload) {
      if (!this.active) return;
      const projected = project(payload);
      if (statusField === 'serverInfo' && projected == null) return;
      callback(projected);
    },
    releaseRuntime: () => {},
  };
  subscription.releaseRuntime = Events.On(eventName, (event) => {
    if (!subscription.active) return;
    subscription.eventReceived = true;
    subscription.deliver(unwrapEvent(event));
  });
  bucket.add(subscription);

  const unsubscribe = () => {
    if (!subscription.active) return;
    subscription.active = false;
    bucket.delete(subscription);
    subscriptions.delete(unsubscribe);
    if (!subscription.released) {
      subscription.released = true;
      subscription.releaseRuntime();
    }
  };
  subscriptions.add(unsubscribe);
  beginStatusSnapshotWhenReady();
  return unsubscribe;
}

const previousFacade = window.desktopAPI;
if (typeof previousFacade?.[DISPOSE] === 'function') previousFacade[DISPOSE]();

const desktopAPI = {
  onServerInfo: (callback) => subscribe('server-info', 'serverInfo', callback, normalizeServerInfo),
  onClientCount: (callback) => subscribe('client-count', 'clientCount', callback),
  onHackState: (callback) => subscribe('hack-state', 'hackState', callback),
  onCoordinationState: (callback) => subscribe('coordination-state', 'coordinationState', callback),
  getRuntimeStatus: () => {
    beginStatusSnapshotWhenReady();
    return runtimeStatusPromise ?? command(APP_METHODS.getRuntimeStatus);
  },
  openUrl: (url) => command(APP_METHODS.openUrl, url),
  openSession: () => command(APP_METHODS.openSession),
  newSession: () => command(APP_METHODS.newSession),
  saveSession: (session) => command(APP_METHODS.saveSession, session),
  loadReferencedPlayerConfig: () => playerConfigCommand(APP_METHODS.loadReferencedPlayerConfig),
  newPlayerConfig: () => playerConfigCommand(APP_METHODS.newPlayerConfig),
  openPlayerConfig: () => playerConfigCommand(APP_METHODS.openPlayerConfig),
  requestTerminalActivation: (payload) => switchCommand(APP_METHODS.requestTerminalActivation, payload),
  updateLiveTerminal: (payload) => command(APP_METHODS.updateLiveTerminal, payload),
  requestTerminalClear: () => switchCommand(APP_METHODS.requestTerminalClear),
  resolveTerminalSwitch: (payload) => switchCommand(APP_METHODS.resolveTerminalSwitch, payload),
  forceHackSuccess: () => command(APP_METHODS.forceHackSuccess),
  resetFailedHack: (payload) => command(APP_METHODS.resetFailedHack, payload),
  addCharacter: (name) => command(APP_METHODS.addCharacter, name),
  renameCharacter: (payload) => command(APP_METHODS.renameCharacter, payload),
  deleteCharacter: (characterId) => command(APP_METHODS.deleteCharacter, characterId),
  renameLogicalSession: (payload) => command(APP_METHODS.renameLogicalSession, payload),
  assignCharacter: (payload) => command(APP_METHODS.assignCharacter, payload),
  releaseCharacter: (sessionId) => command(APP_METHODS.releaseCharacter, sessionId),
  moveCharacter: (payload) => command(APP_METHODS.moveCharacter, payload),
  setActiveController: (sessionId) => command(APP_METHODS.setActiveController, sessionId),
  startBroadcast: () => command(APP_METHODS.startBroadcast),
  endBroadcast: () => command(APP_METHODS.endBroadcast),
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
