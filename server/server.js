const express = require('express');
const http = require('http');
const WebSocket = require('ws');
const path = require('path');
const os = require('os');
const fs = require('fs');
const { generateBoard, applyGuess, applyAdmin, forceSuccess, publicHackState } = require('./hack');
const { defaultNav, applyNavAction, revalidateNav } = require('./nav');

const SOUND_EXTS = new Set(['.mp3', '.wav', '.ogg', '.m4a', '.webm']);
const SOUND_FOLDERS = new Set(['ambient', 'hack-good', 'hack-bad', 'menu-focus', 'single', 'multiple', 'enter', 'charscroll']);

const clients = new Set();
let live = null; // { terminalId, terminalName, tree, hackLevel, introText, hack, nav } | null
let clientCountCallback = null;
let hackStateCallback = null;

function getLocalIP() {
  const interfaces = os.networkInterfaces();
  for (const name of Object.keys(interfaces)) {
    for (const iface of interfaces[name]) {
      if (iface.family === 'IPv4' && !iface.internal) {
        return iface.address;
      }
    }
  }
  return 'localhost';
}

function notifyClientCount() {
  if (clientCountCallback) clientCountCallback(clients.size);
}

function notifyHackChange() {
  if (hackStateCallback) hackStateCallback(live && live.hack ? publicHackState(live.hack) : null);
}

function broadcast(data) {
  const msg = JSON.stringify(data);
  for (const ws of clients) {
    if (ws.readyState === WebSocket.OPEN) {
      ws.send(msg);
    }
  }
}

function publicLiveState() {
  if (!live) return null;
  return {
    terminalId:   live.terminalId,
    terminalName: live.terminalName,
    tree:         live.tree,
    hackLevel:    live.hackLevel || 0,
    introText:    live.introText || '',
    hack:         publicHackState(live.hack),
    nav:          live.nav,
  };
}

// A terminal became live (or a different one was selected, or the GM restarted
// the broadcast): full reset for clients, fresh hack board if applicable.
function setLiveTerminal(payload) {
  const hackLevel = payload.hackLevel || 0;
  live = {
    terminalId:   payload.terminalId,
    terminalName: payload.terminalName,
    tree:         payload.tree,
    hackLevel,
    introText:    payload.introText || '',
    hack:         hackLevel > 0 ? generateBoard(hackLevel) : null,
    nav:          defaultNav(),
  };
  broadcast({ type: 'TERMINAL_LIVE', ...publicLiveState() });
  notifyHackChange();
}

// Edits pushed to the currently-live terminal: shared nav position is
// re-validated against the new tree and re-synced to every player.
function updateLiveTerminal(tree, introText) {
  if (!live) return;
  live.tree = tree;
  if (introText != null) live.introText = introText;
  live.nav = revalidateNav(live.nav, live.tree);
  broadcast({ type: 'TERMINAL_UPDATE', tree: live.tree, introText: live.introText, nav: live.nav });
}

function clearLiveTerminal() {
  live = null;
  broadcast({ type: 'TERMINAL_CLEAR' });
  notifyHackChange();
}

function forceHackSuccess() {
  if (!live || !live.hack) return;
  forceSuccess(live.hack);
  broadcast({ type: 'HACK_STATE', hack: publicHackState(live.hack) });
  notifyHackChange();
}

function handleClientMessage(raw) {
  if (!live) return;
  let msg;
  try { msg = JSON.parse(raw); } catch { return; }

  if (msg.type === 'HACK_GUESS' && live.hack && typeof msg.targetId === 'string') {
    applyGuess(live.hack, msg.targetId);
    broadcast({ type: 'HACK_STATE', hack: publicHackState(live.hack) });
    notifyHackChange();
  } else if (msg.type === 'HACK_ADMIN' && live.hack) {
    applyAdmin(live.hack);
    broadcast({ type: 'HACK_STATE', hack: publicHackState(live.hack) });
    notifyHackChange();
  } else if (msg.type === 'NAV_ACTION' && typeof msg.action === 'string') {
    applyNavAction(live.nav, live.tree, msg);
    broadcast({ type: 'NAV_STATE', nav: live.nav });
  }
}

function onClientCountChange(cb) {
  clientCountCallback = cb;
}

function onHackStateChange(cb) {
  hackStateCallback = cb;
}

async function startServer(port = 3690) {
  const app = express();
  const server = http.createServer(app);

  app.use(express.static(path.join(__dirname, '../client')));

  // Generic sound-folder listing: client picks a random (or the only) file.
  // Folder name is allowlisted so req.params can never escape client/sounds.
  app.get('/api/sounds/:folder', (req, res) => {
    if (!SOUND_FOLDERS.has(req.params.folder)) return res.json([]);
    try {
      const dir = path.join(__dirname, '../client/sounds', req.params.folder);
      const files = fs.readdirSync(dir).filter(f => SOUND_EXTS.has(path.extname(f).toLowerCase()));
      res.json(files);
    } catch {
      res.json([]);
    }
  });

  const wss = new WebSocket.Server({ server });

  wss.on('connection', (ws) => {
    clients.add(ws);
    notifyClientCount();

    if (live) {
      ws.send(JSON.stringify({ type: 'TERMINAL_LIVE', ...publicLiveState() }));
    }

    ws.on('message', (raw) => handleClientMessage(raw));

    ws.on('close', () => {
      clients.delete(ws);
      notifyClientCount();
    });

    ws.on('error', () => {
      clients.delete(ws);
      notifyClientCount();
    });
  });

  return new Promise((resolve, reject) => {
    server.listen(port, '0.0.0.0', () => {
      const ip = getLocalIP();
      resolve({ ip, port, url: `http://${ip}:${port}` });
    });
    server.on('error', reject);
  });
}

module.exports = {
  startServer,
  setLiveTerminal,
  updateLiveTerminal,
  clearLiveTerminal,
  forceHackSuccess,
  onClientCountChange,
  onHackStateChange,
};
