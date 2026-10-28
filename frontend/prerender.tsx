import { writeFile } from "node:fs/promises";
import { renderToString } from "preact-render-to-string";
import { App } from "./src/app/App";

// Render the same initial loading state the browser hydrates. Catalog data is
// fetched after mounting, so this artifact stays independent of archive data.
await writeFile(
  new URL("./dist/app-shell.html", import.meta.url),
  renderToString(<App initialPathname="/" />),
);
