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
let audioCtx = null;

function getCtx() {
  if (!audioCtx) audioCtx = new (window.AudioContext || window.webkitAudioContext)();
  if (audioCtx.state === 'suspended') audioCtx.resume().catch(() => {});
  return audioCtx;
}

async function prefetch(url) {
  if (rawBufs.has(url) || decodedBufs.has(url)) return;
  try {
    const res = await fetch(url);
    if (res.ok) rawBufs.set(url, await res.arrayBuffer());
  } catch { /* sound file missing — silently skip */ }
}

async function playBuf(url, volume) {
  let buffer = decodedBufs.get(url);
  if (!buffer) {
    const raw = rawBufs.get(url);
    if (!raw) return;
    try {
      const c = getCtx();
      buffer = await c.decodeAudioData(raw.slice(0));
      decodedBufs.set(url, buffer);
    } catch { return; }
  }
  const c = getCtx();
  const src = c.createBufferSource();
  const gain = c.createGain();
  src.buffer = buffer;
  gain.gain.value = volume;
  src.connect(gain);
  gain.connect(c.destination);
  src.start();
}

async function loadFolder(name) {
  try {
    const res = await fetch(`/api/sounds/${name}`);
    if (!res.ok) return;
    const files = await res.json();
    folderFiles[name] = files;
    files.forEach(f => prefetch(`/sounds/${name}/${encodeURIComponent(f)}`));
  } catch { /* sound folder missing — silently skip */ }
}

function playFromFolder(name, volume) {
  const files = folderFiles[name];
  if (!files || !files.length) return;
  const f = files[Math.floor(Math.random() * files.length)];
  playBuf(`/sounds/${name}/${encodeURIComponent(f)}`, volume);
}

function playFirst(name, volume) {
  const files = folderFiles[name];
  if (!files || !files.length) return;
  playBuf(`/sounds/${name}/${encodeURIComponent(files[0])}`, volume);
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
  await loadFolder('ambient');
  const files = folderFiles.ambient;
  if (!files || !files.length) return;
  ambientAudio = new Audio(`/sounds/ambient/${encodeURIComponent(files[0])}`);
  ambientAudio.loop = true;
  ambientAudio.volume = 0.25;
  ambientReady = true;
  tryStartAmbient();
}

function tryStartAmbient() {
  if (ambientReady && userGestured && ambientAudio && ambientAudio.paused) {
    ambientAudio.play().catch(() => {});
  }
}

function stopAmbient() {
  if (ambientAudio) ambientAudio.pause();
}

document.addEventListener('click', () => {
  if (!userGestured) {
    userGestured = true;
    tryStartAmbient();
  }
});

// ── Boot: prefetch everything up front ─────────────────
['single', 'multiple', 'enter', 'hack-good', 'hack-bad', 'menu-focus', 'charscroll'].forEach(loadFolder);
setupAmbient();
