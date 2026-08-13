import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: '.',
  testMatch: '*.spec.mjs',
  fullyParallel: false,
  workers: 1,
  use: {
    baseURL: 'http://127.0.0.1:34119',
    headless: true,
  },
  webServer: {
    command: 'npm run build --prefix ../../client && GOCACHE=/private/tmp/fallout-browser-fixture-cache go run ./fixture-server',
    url: 'http://127.0.0.1:34119',
    reuseExistingServer: false,
  },
});
