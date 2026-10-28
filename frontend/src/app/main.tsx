import { hydrate, render } from "preact";
import { App } from "./App";
import "../styles/app.css";

const root = document.getElementById("app");

if (!root) {
  throw new Error("App root not found");
}

// Production's homepage shell is generated from App during the frontend build.
// Other routes and the Vite dev server start with an empty root.
if (root.children.length > 0) {
  hydrate(<App />, root);
} else {
  render(<App />, root);
}
