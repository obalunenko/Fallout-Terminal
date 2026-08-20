import { cpSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [{
    name: 'copy-embedded-sound-assets',
    closeBundle() {
      cpSync(resolve('sounds'), resolve('dist/sounds'), { recursive: true });
      writeFileSync(resolve('dist/.keep'), '');
    },
  }],
});
