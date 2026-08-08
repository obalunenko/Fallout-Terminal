'use strict';

const { spawn } = require('child_process');
const fs = require('fs');
const os = require('os');
const path = require('path');

let tunnelProcess = null;
const DEFAULT_DOMAIN = 'fallout-terminal.ngrok.app';

function getBasicAuthCredential(options) {
  if (options.basicAuth !== undefined) return options.basicAuth;

  const username = options.username || process.env.NGROK_USERNAME;
  const password = options.password || process.env.NGROK_PASSWORD;
  if (username || password) {
    if (!username || !password) {
      throw new Error('для ngrok нужно задать и NGROK_USERNAME, и NGROK_PASSWORD');
    }
    return `${username}:${password}`;
  }

  return process.env.NGROK_BASIC_AUTH;
}

function validateBasicAuthCredential(credential) {
  if (typeof credential !== 'string' || !credential.includes(':')) {
    throw new Error('задайте доступ игроков через NGROK_USERNAME и NGROK_PASSWORD');
  }

  const separator = credential.indexOf(':');
  const username = credential.slice(0, separator);
  const password = credential.slice(separator + 1);
  if (!username || username.includes('\n') || username.includes('\r')) {
    throw new Error('логин ngrok не должен быть пустым или содержать перенос строки');
  }
  if (password.length < 8 || password.length > 128) {
    throw new Error('пароль ngrok должен содержать от 8 до 128 символов');
  }
  if (password.includes('\n') || password.includes('\r')) {
    throw new Error('пароль ngrok не должен содержать перенос строки');
  }

  return credential;
}

function createTrafficPolicy(credential) {
  return JSON.stringify({
    on_http_request: [{
      actions: [{
        type: 'basic-auth',
        config: {
          realm: 'Fallout Terminal Players',
          credentials: [credential],
          enforce: true,
        },
      }],
    }],
  });
}

function writeTemporaryPolicy(credential) {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'fallout-terminal-ngrok-'));
  const file = path.join(directory, 'traffic-policy.json');
  fs.writeFileSync(file, createTrafficPolicy(credential), { mode: 0o600 });

  return {
    file,
    cleanup() {
      try { fs.rmSync(directory, { recursive: true, force: true }); } catch { /* best effort */ }
    },
  };
}

function findPublicUrl(line) {
  try {
    const entry = JSON.parse(line);
    if (entry.msg === 'started tunnel'
        && typeof entry.url === 'string'
        && entry.url.startsWith('https://')) {
      return entry.url;
    }
  } catch {
    // Keep the fallback below compatible with slightly different ngrok logs.
  }

  const match = /started tunnel/i.test(line)
    ? line.match(/https:\/\/[^\s"']+/)
    : null;
  return match ? match[0] : null;
}

function stopNgrok() {
  if (!tunnelProcess) return;
  tunnelProcess.kill();
  tunnelProcess = null;
}

function startNgrok(port, options = {}) {
  if (tunnelProcess) {
    return Promise.reject(new Error('туннель ngrok уже запущен'));
  }

  const binary = options.binary || process.env.NGROK_BIN || 'ngrok';
  const domain = options.domain || process.env.NGROK_DOMAIN || DEFAULT_DOMAIN;
  const timeoutMs = options.timeoutMs || 20000;
  const endpointUrl = /^https?:\/\//i.test(domain) ? domain : `https://${domain}`;
  let credential;
  try {
    credential = validateBasicAuthCredential(getBasicAuthCredential(options));
  } catch (error) {
    return Promise.reject(error);
  }

  return new Promise((resolve, reject) => {
    let policy;
    try {
      policy = writeTemporaryPolicy(credential);
    } catch (error) {
      reject(new Error(`не удалось подготовить защиту ngrok: ${error.message}`));
      return;
    }

    const child = spawn(binary, [
      'http',
      String(port),
      '--url', endpointUrl,
      '--traffic-policy-file', policy.file,
      '--log', 'stdout',
      '--log-format', 'json',
    ], {
      env: process.env,
      stdio: ['ignore', 'pipe', 'pipe'],
      windowsHide: true,
    });

    tunnelProcess = child;
    let settled = false;
    let stdoutBuffer = '';
    let errorOutput = '';

    const fail = (error) => {
      if (settled) return;
      settled = true;
      clearTimeout(timer);
      policy.cleanup();
      if (tunnelProcess === child) tunnelProcess = null;
      if (!child.killed) child.kill();
      reject(error);
    };

    const inspectLine = (line) => {
      const url = findPublicUrl(line);
      if (!url || settled) return;
      settled = true;
      clearTimeout(timer);
      policy.cleanup();
      resolve(url);
    };

    child.stdout.on('data', (chunk) => {
      stdoutBuffer += chunk.toString();
      const lines = stdoutBuffer.split(/\r?\n/);
      stdoutBuffer = lines.pop() || '';
      lines.forEach(inspectLine);
    });

    child.stderr.on('data', (chunk) => {
      errorOutput = (errorOutput + chunk.toString()).slice(-4000);
    });

    child.once('error', (error) => {
      if (error.code === 'ENOENT') {
        fail(new Error(`бинарник ngrok не найден (${binary})`));
      } else {
        fail(error);
      }
    });

    child.once('exit', (code) => {
      if (tunnelProcess === child) tunnelProcess = null;
      if (!settled) {
        const details = errorOutput.trim();
        fail(new Error(details || `ngrok завершился с кодом ${code}`));
      } else if (code !== 0 && code !== null) {
        console.error(`[ngrok] tunnel stopped with code ${code}`);
      }
    });

    const timer = setTimeout(() => {
      fail(new Error('ngrok не выдал публичный адрес за 20 секунд'));
    }, timeoutMs);
  });
}

module.exports = {
  startNgrok,
  stopNgrok,
  findPublicUrl,
  validateBasicAuthCredential,
  createTrafficPolicy,
  DEFAULT_DOMAIN,
};
