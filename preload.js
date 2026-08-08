'use strict';

const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('electronAPI', {
  onServerInfo: (callback) => {
    ipcRenderer.on('server-info', (_event, info) => callback(info));
  },
  onClientCount: (callback) => {
    ipcRenderer.on('client-count', (_event, count) => callback(count));
  },
  onHackState: (callback) => {
    ipcRenderer.on('hack-state', (_event, hack) => callback(hack));
  },
  openUrl: (url) => ipcRenderer.send('open-url', url),
  openSession: () => ipcRenderer.invoke('session:open'),
  newSession: () => ipcRenderer.invoke('session:new'),
  saveSession: (session) => ipcRenderer.invoke('session:save', session),
  setLiveTerminal: (payload) => ipcRenderer.send('terminal:set-live', payload),
  updateLiveTerminal: (payload) => ipcRenderer.send('terminal:update-live', payload),
  clearLiveTerminal: () => ipcRenderer.send('terminal:clear-live'),
  forceHackSuccess: () => ipcRenderer.send('terminal:hack-force-success'),
});
