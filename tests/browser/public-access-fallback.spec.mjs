import { expect, test } from '@playwright/test';

const PLAYER_SERVICE = '/fallout.terminal.player.v1.PlayerService/';

async function installLocalDiagnostics(context) {
  await context.addInitScript(() => {
    window.__fallbackAudioPlays = 0;
    window.__fallbackLegacySockets = 0;
    window.WebSocket = class ForbiddenLegacyPlayerTransport {
      constructor() {
        window.__fallbackLegacySockets += 1;
        throw new Error('local fallback must keep generated Connect transport');
      }
    };
    HTMLMediaElement.prototype.play = () => {
      window.__fallbackAudioPlays += 1;
      return Promise.resolve();
    };
  });
}

test.beforeEach(async ({ request }) => {
  const response = await request.post('/__fixture/reset');
  expect(response.status()).toBe(204);
});

test('all public failures leave local gameplay live and a later public generation recovers without restart', async ({ browser, request }) => {
  const playerContext = await browser.newContext();
  await installLocalDiagnostics(playerContext);
  const player = await playerContext.newPage();
  let subscribeCount = 0;
  let soundManifestCount = 0;
  player.on('request', observed => {
    if (!observed.url().includes(PLAYER_SERVICE)) return;
    if (observed.url().endsWith('/Subscribe')) subscribeCount += 1;
    if (observed.url().endsWith('/SoundManifest')) soundManifestCount += 1;
  });
  await player.goto('/');
  await expect(player.locator('#connOverlay')).toBeHidden();
  const character = player.locator('#characterOptions button:not([disabled])').first();
  const characterName = (await character.textContent()).trim();
  await character.click();
  await expect(player.locator('#playerCharacterName')).toHaveText(characterName);
  await expect(player.locator('#termList')).toBeVisible();

  const master = await browser.newPage();
  await master.goto('/__fixture/public-access-settings');
  await expect(master.locator('#publicAccessSection')).toBeVisible();

  const failures = [
    'invalid-token', 'revoked-token', 'no-network', 'dns-timeout', 'domain-conflict',
    'keychain-locked', 'keychain-denied', 'keychain-unavailable', 'policy-failure',
    'provider-failure', 'unexpected-done', 'close-failure', 'stale-completion',
  ];
  for (const [index, failure] of failures.entries()) {
    const response = await request.post(`/__fixture/public-access/failure/${failure}`);
    expect(response.status(), failure).toBe(200);
    const snapshot = await response.json();
    await master.evaluate(value => __desktopFixture.emit('public-access-status', value), snapshot);
    await expect(master.locator('#publicAccessStatus')).toHaveText('ОШИБКА');
    await expect(master.locator('#publicAccessURL')).toHaveText('');
    await expect(master.locator('#publicAccessError')).toContainText('ЛОКАЛЬНЫЙ РЕЖИМ ПРОДОЛЖАЕТ РАБОТАТЬ');

    await expect(player.locator('#connOverlay')).toBeHidden();
    await expect(player.locator('#playerCharacterName')).toHaveText(characterName);
    if (index % 2 === 0) {
      await player.locator('.term-row', { hasText: 'DOCS' }).click();
      await expect(player.locator('.term-row', { hasText: 'REPORT' })).toBeVisible();
      await expect(player.locator('#screen')).not.toHaveClass(/shared-input-pending/);
    } else {
      await player.locator('#backBtn').click();
      await expect(player.locator('.term-row', { hasText: 'DOCS' })).toBeVisible();
    }
  }

  const hacking = await request.post('/__fixture/local/hacking');
  expect(hacking.status()).toBe(204);
  await expect(player.locator('#hackBoard')).toBeVisible();
  const guess = player.locator('#hackColumns [data-target]:not([data-target=""])').first();
  await expect(guess).toBeVisible();
  await guess.click();
  await expect(player.locator('#hackLog')).not.toHaveText('');

  const disconnect = await request.post('/__fixture/local/disconnect');
  expect(disconnect.status()).toBe(204);
  await expect(player.locator('#connOverlay')).toBeHidden({ timeout: 5_000 });
  await expect.poll(() => subscribeCount, { timeout: 5_000 }).toBeGreaterThanOrEqual(2);
  await expect.poll(() => soundManifestCount).toBeGreaterThan(0);
  expect(await player.evaluate(() => window.__fallbackLegacySockets)).toBe(0);

  const recoveryResponse = await request.post('/__fixture/public-access/recover');
  expect(recoveryResponse.status()).toBe(200);
  const recovered = await recoveryResponse.json();
  await master.evaluate(value => __desktopFixture.emit('public-access-status', value), recovered);
  await expect(master.locator('#publicAccessStatus')).toHaveText('ГОТОВ');
  await expect(master.locator('#publicAccessURL')).toHaveText('https://recovered.example');
  await expect(player.locator('#connOverlay')).toBeHidden();
  await expect(player.locator('#playerCharacterName')).toHaveText(characterName);

  await master.close();
  await playerContext.close();
});
