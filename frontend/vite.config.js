import { defineConfig } from "vite";
import { writeFileSync } from "node:fs";
import wails from "@wailsio/runtime/plugins/vite";

export default defineConfig({
  root: "src",
  base: "./",
  plugins: [
    wails("./bindings"),
    {
      name: "preserve-go-embed-marker",
      closeBundle() {
        writeFileSync(new URL("./dist/.keep", import.meta.url), "");
      },
    },
  ],
  build: {
    outDir: "../dist",
    emptyOutDir: true,
  },
});
