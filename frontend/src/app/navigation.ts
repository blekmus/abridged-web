import type { JSX } from "preact";
import { navigate } from "../lib/router";

export function handleInternalLinkClick(
  event: JSX.TargetedMouseEvent<HTMLAnchorElement>,
  href: string,
  options: { scrollToTop?: boolean } = {},
): void {
  if (shouldUseBrowserNavigation(event)) {
    return;
  }

  event.preventDefault();
  navigate(href);

  if (options.scrollToTop) {
    requestAnimationFrame(() => {
      scrollPageToTop();
    });
  }
}

export function scrollPageToTop(): void {
  scrollPageToPosition({ top: 0, left: 0 });
}

export function scrollPageToPosition(position: {
  left: number;
  top: number;
}): void {
  const scrollingElement =
    document.scrollingElement ?? document.documentElement;

  scrollingElement.scrollTo({ ...position, behavior: "auto" });
  window.scrollTo({ ...position, behavior: "auto" });
}

function shouldUseBrowserNavigation(
  event: JSX.TargetedMouseEvent<HTMLAnchorElement>,
): boolean {
  return (
    event.defaultPrevented ||
    event.button !== 0 ||
    event.metaKey ||
    event.ctrlKey ||
    event.shiftKey ||
    event.altKey ||
    (event.currentTarget.target !== "" &&
      event.currentTarget.target !== "_self")
  );
}
