// RobCo-style terminal hacking minigame — board generation + shared game state
// mutation. All gameplay logic lives here and runs server-side only; clients
// never receive the secret word, only the rendered board + log + counters.
const { pickWords } = require('./wordbank');

const FILLER_POOL = '!@#$%^&*()_+-=[]{}\\|;:\'",.<>/?~'.split('');
const ROWS = 16;
const ROW_WIDTH = 12;
const COL_CHARS = ROWS * ROW_WIDTH; // 192

function randFiller() {
  return FILLER_POOL[Math.floor(Math.random() * FILLER_POOL.length)];
}

function genAddresses(count) {
  let addr = Math.floor(Math.random() * 0x4000) + 0xC000;
  const step = [0x0C, 0x10, 0x14, 0x18][Math.floor(Math.random() * 4)];
  const list = [];
  for (let i = 0; i < count; i++) {
    list.push('0x' + (addr & 0xFFFF).toString(16).toUpperCase().padStart(4, '0'));
    addr += step;
  }
  return list;
}

// A "column" here is one content sub-column (192 flattened characters,
// rendered by the client as 16 rows of 12 chars). `prefix` keeps word ids
// globally unique across the two columns.
function makeColumnBuilder(prefix) {
  const chars = new Array(COL_CHARS).fill(null);
  const words = []; // {id, start, length, isAdmin}
  let n = 1;

  function place(text, isAdmin, startAt) {
    const len = text.length;
    const GAP = 4; // minimum filler characters required between two words
    let start = startAt;
    if (start == null) {
      let attempts = 0;
      while (attempts < 300) {
        const candidate = Math.floor(Math.random() * (COL_CHARS - len));
        const checkFrom = Math.max(0, candidate - GAP);
        const checkTo = Math.min(COL_CHARS, candidate + len + GAP);
        let free = true;
        for (let i = checkFrom; i < checkTo; i++) {
          if (chars[i] !== null) { free = false; break; }
        }
        if (free) { start = candidate; break; }
        attempts++;
      }
      if (start == null) return null; // grid too full — skip this word
    }
    const id = prefix + (n++);
    for (let i = 0; i < len; i++) chars[start + i] = text[i];
    words.push({ id, start, length: len, isAdmin: !!isAdmin });
    return id;
  }

  function finish() {
    for (let i = 0; i < chars.length; i++) {
      if (chars[i] === null) chars[i] = randFiller();
    }
    return { chars, words };
  }

  return { place, finish };
}

function generateBoard(level) {
  const wordLength = { 1: 4, 2: 5, 3: 6, 4: 7, 5: 8 }[level] || 4;
  const wordCount = 11 + Math.max(1, Math.min(5, level)); // 12..16
  const words = pickWords(wordLength, wordCount);
  const secretWord = words[Math.floor(Math.random() * words.length)];

  const colA = makeColumnBuilder('A');
  const colB = makeColumnBuilder('B');
  const wordsById = {};

  const successId = colB.place('SUCCESS', true, 0);
  if (successId) wordsById[successId] = { text: 'SUCCESS', isAdmin: true };

  words.forEach((text, i) => {
    const builder = i % 2 === 0 ? colA : colB;
    const id = builder.place(text, false);
    if (id) wordsById[id] = { text, isAdmin: false };
  });

  const finishedA = colA.finish();
  const finishedB = colB.finish();

  return {
    level,
    wordLength,
    attemptsMax: 4,
    attemptsLeft: 4,
    secretWord,
    wordsById,
    adminModeUsed: false,
    solved: false,
    failed: false,
    log: [],
    columns: [
      { addresses: genAddresses(ROWS), text: finishedA.chars.join(''), words: finishedA.words },
      { addresses: genAddresses(ROWS), text: finishedB.chars.join(''), words: finishedB.words },
    ],
  };
}

function pushLog(hack, lines) {
  for (const line of lines) hack.log.push('> ' + line);
}

function countMatches(a, b) {
  let n = 0;
  for (let i = 0; i < a.length && i < b.length; i++) if (a[i] === b[i]) n++;
  return n;
}

function findWordLocation(hack, id) {
  for (let colIndex = 0; colIndex < hack.columns.length; colIndex++) {
    const w = hack.columns[colIndex].words.find(w => w.id === id);
    if (w) return { colIndex, start: w.start, length: w.length };
  }
  return null;
}

function applyAdmin(hack) {
  if (hack.solved || hack.failed) return;
  pushLog(hack, ['Режим администратора активирован.']);
  if (hack.adminModeUsed) return;
  hack.adminModeUsed = true;

  const candidateIds = Object.keys(hack.wordsById).filter(id => !hack.wordsById[id].isAdmin);
  const keepId = candidateIds.find(id => hack.wordsById[id].text === hack.secretWord) || candidateIds[0];
  const others = candidateIds.filter(id => id !== keepId);
  const decoyId = others.length ? others[Math.floor(Math.random() * others.length)] : null;
  const toDot = candidateIds.filter(id => id !== keepId && id !== decoyId);

  for (const id of toDot) {
    const loc = findWordLocation(hack, id);
    if (!loc) continue;
    const col = hack.columns[loc.colIndex];
    const dots = '.'.repeat(loc.length);
    col.text = col.text.slice(0, loc.start) + dots + col.text.slice(loc.start + loc.length);
    col.words = col.words.filter(w => w.id !== id);
    delete hack.wordsById[id];
  }
}

function applyGuess(hack, targetId) {
  if (hack.solved || hack.failed) return;

  const entry = hack.wordsById[targetId];
  if (entry && entry.isAdmin) {
    applyAdmin(hack);
    return;
  }

  if (entry) {
    pushLog(hack, [entry.text]);
    const matches = countMatches(entry.text, hack.secretWord);
    if (matches === hack.wordLength) {
      hack.solved = true;
      pushLog(hack, ['Точно!', 'Пожалуйста,', 'подождите', 'входа в систему.']);
    } else {
      hack.attemptsLeft = Math.max(0, hack.attemptsLeft - 1);
      pushLog(hack, ['Отказ в доступе', `${matches}/${hack.wordLength} правильно.`]);
      if (hack.attemptsLeft === 0) hack.failed = true;
    }
    return;
  }

  // Not a known word id — treat as a filler click, targetId = "colIndex:charIndex".
  const m = /^(\d+):(\d+)$/.exec(String(targetId));
  if (!m) return;
  const colIndex = Number(m[1]);
  const charIndex = Number(m[2]);
  const col = hack.columns[colIndex];
  if (!col || charIndex < 0 || charIndex >= col.text.length) return;
  const insideWord = col.words.some(w => charIndex >= w.start && charIndex < w.start + w.length);
  if (insideWord) return; // stale/tampered reference — ignore

  pushLog(hack, [col.text[charIndex]]);
  hack.attemptsLeft = Math.max(0, hack.attemptsLeft - 1);
  pushLog(hack, ['Отказ в доступе', `0/${hack.wordLength} правильно.`]);
  if (hack.attemptsLeft === 0) hack.failed = true;
}

function forceSuccess(hack) {
  if (!hack || hack.solved || hack.failed) return;
  pushLog(hack, [hack.secretWord]);
  hack.solved = true;
  pushLog(hack, ['Точно!', 'Пожалуйста,', 'подождите', 'входа в систему.']);
}

function publicHackState(hack) {
  if (!hack) return null;
  return {
    level: hack.level,
    wordLength: hack.wordLength,
    attemptsMax: hack.attemptsMax,
    attemptsLeft: hack.attemptsLeft,
    solved: hack.solved,
    failed: hack.failed,
    log: hack.log,
    columns: hack.columns,
  };
}

module.exports = { generateBoard, applyGuess, applyAdmin, forceSuccess, publicHackState };
