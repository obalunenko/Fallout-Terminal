'use strict';

// ════════════════════════════════════════════════════
// SOUND SYSTEM — Web Audio playback + folder-based random pick.
// Folders with a single expected file (ambient, hack-good, hack-bad,
// menu-focus) just play files[0]. Folders with many files (single,
// multiple, enter) pick a random one each time.
// ════════════════════════════════════════════════════

const folderFiles = {};   // folder -> [filenames]
const rawBufs     = new Map(); // url -> ArrayBuffer
const decodedBufs = new Map(); // url -> AudioBuffer
const folderLoads = new Map(); // folder -> Promise<[filenames]>
const rawLoads    = new Map(); // url -> Promise<void>
const oneShotFolders = ['single', 'multiple', 'enter', 'hack-good', 'hack-bad', 'menu-focus', 'charscroll'];
let audioCtx = null;
let webAudioEligible = false;
let webAudioReady = null;

function reportPlayback(url) {
  try {
    if (typeof window.__falloutTerminalSoundObserver === 'function') {
      window.__falloutTerminalSoundObserver(url);
    }
  } catch {
    // Test/diagnostic observation is optional and cannot affect playback.
  }
}

function getCtx() {
  const AudioContextClass = window.AudioContext || window.webkitAudioContext;
  if (!AudioContextClass) return null;
  try {
    if (!audioCtx) audioCtx = new AudioContextClass();
  } catch {
    return null;
  }
  return audioCtx;
}

function enableWebAudio() {
  if (webAudioReady) return webAudioReady;
  const context = getCtx();
  if (!context) return null;

  webAudioReady = (async () => {
    try {
      if (context.state === 'suspended') await context.resume();
      if (context.state !== 'running') return false;

      await Promise.all(oneShotFolders.map(loadFolder));
      await Promise.all(Array.from(rawBufs.entries()).map(async ([url, raw]) => {
        if (decodedBufs.has(url)) return;
        const buffer = await context.decodeAudioData(raw.slice(0));
        decodedBufs.set(url, buffer);
      }));
      if (context.state === 'running') {
        webAudioEligible = true;
        return true;
      }
    } catch {
      // A document without an eligible Web Audio context remains silent.
    }
    return false;
  })();
  return webAudioReady;
}

async function prefetch(url) {
  if (rawBufs.has(url) || decodedBufs.has(url)) return;
  if (rawLoads.has(url)) return rawLoads.get(url);
  const loading = (async () => {
    try {
      const res = await fetch(url);
      if (res.ok) rawBufs.set(url, await res.arrayBuffer());
    } catch { /* sound file missing — silently skip */ }
  })();
  rawLoads.set(url, loading);
  try {
    await loading;
  } finally {
    rawLoads.delete(url);
  }
}

async function playBuf(url, volume) {
  try {
    if (!webAudioReady || !await webAudioReady || !webAudioEligible) {
      return;
    }
    let buffer = decodedBufs.get(url);
    if (!buffer) {
      if (!rawBufs.has(url)) await prefetch(url);
      const raw = rawBufs.get(url);
      if (!raw) return;
      const c = getCtx();
      if (!c) return;
      buffer = await c.decodeAudioData(raw.slice(0));
      decodedBufs.set(url, buffer);
    }
    const c = getCtx();
    if (!c) return;
    const src = c.createBufferSource();
    const gain = c.createGain();
    src.buffer = buffer;
    gain.gain.value = volume;
    src.connect(gain);
    gain.connect(c.destination);
    src.start();
    reportPlayback(url);
  } catch {
    // Audio is optional: decode, autoplay, and device failures are non-fatal.
  }
}

async function loadFolder(name) {
  if (Object.prototype.hasOwnProperty.call(folderFiles, name)) return folderFiles[name];
  if (folderLoads.has(name)) return folderLoads.get(name);
  const loading = (async () => {
    try {
      const res = await fetch(`/api/sounds/${name}`);
      if (!res.ok) return [];
      const files = await res.json();
      if (!Array.isArray(files)) return [];
      const supported = files.filter(file => typeof file === 'string');
      folderFiles[name] = supported;
      await Promise.all(supported.map(file =>
        prefetch(`/sounds/${name}/${encodeURIComponent(file)}`)
      ));
      return supported;
    } catch {
      return [];
    }
  })();
  folderLoads.set(name, loading);
  try {
    return await loading;
  } finally {
    folderLoads.delete(name);
  }
}

async function playFromFolder(name, volume) {
  const files = folderFiles[name] || await loadFolder(name);
  if (!files || !files.length) return;
  const f = files[Math.floor(Math.random() * files.length)];
  await playBuf(`/sounds/${name}/${encodeURIComponent(f)}`, volume);
}

async function playFirst(name, volume) {
  const files = folderFiles[name] || await loadFolder(name);
  if (!files || !files.length) return;
  await playBuf(`/sounds/${name}/${encodeURIComponent(files[0])}`, volume);
}

// ── Public one-shot sounds ─────────────────────────────
function playSingle()    { playFromFolder('single', 0.55); }
function playMultiple()  { playFromFolder('multiple', 0.55); }
function playEnter()     { playFromFolder('enter', 0.65); }
function playMenuFocus() { playFirst('menu-focus', 0.5); }
function playHackGood()  { playFirst('hack-good', 0.8); }
function playHackBad()   { playFirst('hack-bad', 0.7); }
function playCharScroll() { playFromFolder('charscroll', 0.4); }

// ── Ambient loop (needs a user gesture before autoplay is allowed) ───
let ambientAudio = null;
let ambientReady = false;
let userGestured = false;

async function setupAmbient() {
  try {
    await loadFolder('ambient');
    const files = folderFiles.ambient;
    if (!files || !files.length || typeof window.Audio !== 'function') return;
    ambientAudio = new window.Audio(`/sounds/ambient/${encodeURIComponent(files[0])}`);
    ambientAudio.loop = true;
    ambientAudio.volume = 0.25;
    ambientReady = true;
    tryStartAmbient();
  } catch {
    ambientAudio = null;
    ambientReady = false;
  }
}

function tryStartAmbient() {
  if (ambientReady && userGestured && ambientAudio && ambientAudio.paused) {
    try {
      const playing = ambientAudio.play();
      if (playing && typeof playing.catch === 'function') playing.catch(() => {});
    } catch { /* autoplay or device failure — silently skip */ }
  }
}

function stopAmbient() {
  try {
    if (ambientAudio) ambientAudio.pause();
  } catch { /* optional audio must not interrupt terminal state */ }
}

document.addEventListener('click', () => {
  if (!userGestured) {
    userGestured = true;
    enableWebAudio();
    tryStartAmbient();
  }
});

// ── Boot: prefetch everything up front ─────────────────
oneShotFolders.forEach(loadFolder);
setupAmbient().catch(() => {});
