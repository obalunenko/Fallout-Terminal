const { app, BrowserWindow, ipcMain, shell, dialog } = require('electron');
const path = require('path');
const fs   = require('fs');

let mainWindow;
let serverModule;
let currentSessionPath = null;

function defaultSession(name) {
  return {
    version: 1,
    name,
    terminals: [
      {
        id: 't' + Date.now(),
        name: 'Терминал 1',
        hackLevel: 0,
        introText: '',
        root: { id: 'root', type: 'folder', name: 'ROOT', children: [] },
      },
    ],
  };
}

function readSessionFile(filePath) {
  const raw = fs.readFileSync(filePath, 'utf-8');
  const data = JSON.parse(raw);
  if (!data || !Array.isArray(data.terminals)) {
    throw new Error('Неверный формат файла сессии: отсутствует terminals[]');
  }
  return data;
}

function writeSessionFile(filePath, session) {
  fs.writeFileSync(filePath, JSON.stringify(session, null, 2), 'utf-8');
}

// Sessions bundled with the app (e.g. the sample demo.json) — resolves
// correctly both in dev and inside a packaged/portable build, since
// __dirname always points at the real app files either way.
function bundledSessionsDir() {
  return path.join(__dirname, 'sessions');
}

// Where new sessions should be saved/opened from by default: next to the
// portable .exe once packaged (so they travel with it), or the project's
// own sessions/ folder in dev.
function writableSessionsDir() {
  const dir = app.isPackaged
    ? path.join(path.dirname(app.getPath('exe')), 'sessions')
    : path.join(__dirname, 'sessions');
  try { fs.mkdirSync(dir, { recursive: true }); } catch { /* already exists */ }
  return dir;
}

// First run of a packaged build: seed the writable sessions folder with the
// bundled demo so there's something to open right away, without ever
// overwriting anything the user has already saved there.
function seedDemoSessionIfEmpty(dir) {
  try {
    if (fs.readdirSync(dir).length > 0) return;
    const demoSrc = path.join(bundledSessionsDir(), 'demo.json');
    if (fs.existsSync(demoSrc)) fs.copyFileSync(demoSrc, path.join(dir, 'demo.json'));
  } catch { /* non-fatal — just no seed file */ }
}

async function createWindow() {
  seedDemoSessionIfEmpty(writableSessionsDir());

  serverModule = require('./server/server');
  const serverInfo = await serverModule.startServer(3690);

  mainWindow = new BrowserWindow({
    width: 1200,
    height: 780,
    minWidth: 900,
    minHeight: 600,
    title: 'Fallout Terminal — Master Control',
    backgroundColor: '#0b0d0a',
    webPreferences: {
      nodeIntegration: true,
      contextIsolation: false,
    },
  });

  mainWindow.loadFile(path.join(__dirname, 'master/index.html'));
  mainWindow.setMenuBarVisibility(false);

  mainWindow.webContents.on('did-finish-load', () => {
    mainWindow.webContents.send('server-info', serverInfo);
  });

  mainWindow.webContents.on('console-message', (_e, _level, message) => {
    console.log('[master]', message);
  });
  mainWindow.webContents.on('render-process-gone', (_e, details) => {
    console.error('[master] renderer crashed:', details.reason);
  });

  serverModule.onClientCountChange((count) => {
    if (mainWindow && !mainWindow.isDestroyed()) {
      mainWindow.webContents.send('client-count', count);
    }
  });

  serverModule.onHackStateChange((hack) => {
    if (mainWindow && !mainWindow.isDestroyed()) {
      mainWindow.webContents.send('hack-state', hack);
    }
  });
}

app.whenReady().then(createWindow);

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') app.quit();
});

// ── Session: new ──────────────────────────────────────────
ipcMain.handle('session:new', async () => {
  const { filePath, canceled } = await dialog.showSaveDialog(mainWindow, {
    title: 'Новая сессия',
    defaultPath: path.join(writableSessionsDir(), 'session.json'),
    filters: [{ name: 'Fallout Terminal Session', extensions: ['json'] }],
  });
  if (canceled || !filePath) return { ok: false };

  try {
    const name = path.basename(filePath, path.extname(filePath));
    const session = defaultSession(name);
    writeSessionFile(filePath, session);
    currentSessionPath = filePath;
    return { ok: true, filePath, session };
  } catch (e) {
    return { ok: false, error: e.message };
  }
});

// ── Session: open ─────────────────────────────────────────
ipcMain.handle('session:open', async () => {
  const { filePaths, canceled } = await dialog.showOpenDialog(mainWindow, {
    title: 'Открыть сессию',
    defaultPath: writableSessionsDir(),
    filters: [{ name: 'Fallout Terminal Session', extensions: ['json'] }],
    properties: ['openFile'],
  });
  if (canceled || !filePaths.length) return { ok: false };

  try {
    const session = readSessionFile(filePaths[0]);
    currentSessionPath = filePaths[0];
    return { ok: true, filePath: filePaths[0], session };
  } catch (e) {
    return { ok: false, error: e.message };
  }
});

// ── Session: save (autosave to currently open file) ───────
ipcMain.handle('session:save', async (_event, session) => {
  if (!currentSessionPath) return { ok: false, error: 'Нет открытого файла сессии' };
  try {
    writeSessionFile(currentSessionPath, session);
    return { ok: true };
  } catch (e) {
    return { ok: false, error: e.message };
  }
});

// ── Broadcast controls ─────────────────────────────────────
ipcMain.on('terminal:set-live', (event, payload) => {
  serverModule.setLiveTerminal(payload);
});

ipcMain.on('terminal:update-live', (event, payload) => {
  serverModule.updateLiveTerminal(payload.tree, payload.introText);
});

ipcMain.on('terminal:clear-live', () => {
  serverModule.clearLiveTerminal();
});

ipcMain.on('terminal:hack-force-success', () => {
  serverModule.forceHackSuccess();
});

ipcMain.on('open-url', (event, url) => {
  shell.openExternal(url);
});
