import type { EntryType } from "./types";

export const APP_NAVIGATION_EVENT = "abridged:navigate";

export type Route =
  | { name: "home" }
  | { name: "browse"; entryType: EntryType }
  | { name: "creator"; slug: string }
  | { name: "watch"; entryId: string; episodeId?: string }
  | { name: "not-found"; path: string };

export type ScrollPosition = {
  left: number;
  top: number;
};

type AppHistoryState = {
  scroll?: ScrollPosition;
};

export function currentPath(): string {
  return window.location.pathname;
}

export function navigate(to: string): void {
  if (window.location.pathname === to) {
    return;
  }

  saveCurrentScrollPosition();
  window.history.pushState({ scroll: { left: 0, top: 0 } }, "", to);
  window.dispatchEvent(new CustomEvent(APP_NAVIGATION_EVENT));
}

export function saveCurrentScrollPosition(): void {
  const state = readHistoryState();
  window.history.replaceState(
    {
      ...state,
      scroll: {
        left: window.scrollX,
        top: window.scrollY,
      },
    },
    "",
    currentPath(),
  );
}

export function currentHistoryScrollPosition(): ScrollPosition {
  const scroll = readHistoryState().scroll;

  if (scroll && Number.isFinite(scroll.left) && Number.isFinite(scroll.top)) {
    return scroll;
  }

  return { left: 0, top: 0 };
}

function readHistoryState(): AppHistoryState {
  if (typeof window.history.state !== "object" || !window.history.state) {
    return {};
  }

  return window.history.state as AppHistoryState;
}

export function parseRoute(pathname: string): Route {
  const parts = pathname.split("/").filter(Boolean);
  if (parts.length === 0) {
    return { name: "home" };
  }

  if (parts.length === 1) {
    if (parts[0] === "series") {
      return { name: "browse", entryType: "series" };
    }
    if (parts[0] === "shorts") {
      return { name: "browse", entryType: "short" };
    }
    if (parts[0] === "shots") {
      return { name: "browse", entryType: "shot" };
    }
  }

  if (parts[0] === "creator" && parts[1]) {
    return { name: "creator", slug: parts[1] };
  }

  if (parts[0] === "watch" && parts[1]) {
    return parts[2]
      ? { name: "watch", entryId: parts[1], episodeId: parts[2] }
      : { name: "watch", entryId: parts[1] };
  }

  return { name: "not-found", path: pathname };
}
