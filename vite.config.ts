import preact from "@preact/preset-vite";
import { threeMinifier } from "@yushijinhun/three-minifier-rollup";
import { type Plugin, defineConfig } from "vite";

export default defineConfig({
  root: "frontend",
  plugins: [
    // Tree-shake Three's source modules and minify its embedded GLSL shaders.
    {
      ...(threeMinifier() as Plugin),
      name: "three-minifier",
      enforce: "pre",
      apply: "build",
    },
    preact(),
  ],
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
    rollupOptions: {
      output: {
        manualChunks(id) {
          // Keep the renderer cached independently of the lazy-loaded effect.
          // Rollup includes its shared Three dependencies in this chunk too.
          if (id.includes("/node_modules/three/src/renderers/")) {
            return "three-renderer";
          }
          return undefined;
        },
      },
    },
  },
});
