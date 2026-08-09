import { defineConfig } from "vite";
import { writeFileSync } from "node:fs";

export default defineConfig({
  root: "src",
  base: "./",
  plugins: [
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
