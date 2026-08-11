import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: '.',
  testMatch: 'hacking-camouflage.spec.mjs',
  fullyParallel: false,
  workers: 1,
  use: {
    baseURL: 'http://127.0.0.1:34119',
    headless: true,
  },
  webServer: {
    command: 'python3 -m http.server 34119 --bind 127.0.0.1 --directory ../../client',
    url: 'http://127.0.0.1:34119',
    reuseExistingServer: false,
  },
});
