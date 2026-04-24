import { copyFile, mkdir, rm } from "node:fs/promises";
import { dirname } from "node:path";
import { build, context } from "esbuild";

const watch = process.argv.includes("--watch");
const distDir = "frontend/dist";
const assetsDir = `${distDir}/assets`;

async function copyStaticAssets() {
  await mkdir(assetsDir, { recursive: true });
  await copyFile("frontend/index.html", `${distDir}/index.html`);
  await copyFile("frontend/src/styles/app.css", `${assetsDir}/app.css`);
}

async function cleanDist() {
  await rm(distDir, { recursive: true, force: true });
}

const buildOptions = {
  entryPoints: ["frontend/src/app/main.tsx"],
  bundle: true,
  format: "esm",
  platform: "browser",
  target: "es2020",
  outfile: `${assetsDir}/app.js`,
  jsx: "automatic",
  jsxImportSource: "preact",
  sourcemap: watch,
  minify: !watch,
  logLevel: "info",
};

if (!watch) {
  await cleanDist();
}

await copyStaticAssets();

if (watch) {
  const ctx = await context(buildOptions);
  await ctx.watch();
  console.log("Watching frontend sources...");
} else {
  await mkdir(dirname(buildOptions.outfile), { recursive: true });
  await build(buildOptions);
}
