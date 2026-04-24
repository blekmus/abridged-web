import preact from "@preact/preset-vite";
import { defineConfig } from "vite";

export default defineConfig({
  root: "frontend",
  plugins: [preact()],
  server: {
    proxy: {
      "/api": "http://localhost:8080",
      "/thumb": "http://localhost:8080",
      "/video": "http://localhost:8080",
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
});
