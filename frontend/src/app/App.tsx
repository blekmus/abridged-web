import {
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "preact/hooks";
import { api } from "../lib/api";
import {
  APP_NAVIGATION_EVENT,
  currentHistoryScrollPosition,
  currentPath,
  parseRoute,
  saveCurrentScrollPosition,
} from "../lib/router";
import type { CatalogResponse } from "../lib/types";
import { SiteFooter } from "./components/SiteFooter";
import { SiteHeader } from "./components/SiteHeader";
import { EMPTY_CATALOG, TYPE_META } from "./constants";
import { entriesForType } from "./entryUtils";
import { scrollPageToHash, scrollPageToPosition } from "./navigation";
import { setPageTitle } from "./pageTitle";
import { BrowsePage } from "./pages/BrowsePage";
import { CreatorPage } from "./pages/CreatorPage";
import { HomePage } from "./pages/HomePage";
import { NotFoundPage } from "./pages/NotFoundPage";
import { WatchPage } from "./pages/WatchPage";
import { optionalActiveEntryId } from "./propUtils";
import type { AsyncState } from "./types";
import { findEntryForWatchPath } from "./watchPaths";

type NavigationKind = "initial" | "push" | "pop";

type NavigationState = {
  kind: NavigationKind;
  pathname: string;
};

export function App() {
  const [navigationState, setNavigationState] = useState<NavigationState>({
    kind: "initial",
    pathname: currentPath(),
  });
  const [catalogState, setCatalogState] = useState<AsyncState<CatalogResponse>>(
    {
      data: null,
      loading: true,
      error: null,
    },
  );

  useEffect(() => {
    const previousScrollRestoration = window.history.scrollRestoration;
    window.history.scrollRestoration = "manual";

    const onAppNavigation = () => {
      setNavigationState({ kind: "push", pathname: currentPath() });
    };
    const onPopState = () => {
      setNavigationState({ kind: "pop", pathname: currentPath() });
    };

    window.addEventListener(APP_NAVIGATION_EVENT, onAppNavigation);
    window.addEventListener("popstate", onPopState);

    return () => {
      window.history.scrollRestoration = previousScrollRestoration;
      window.removeEventListener(APP_NAVIGATION_EVENT, onAppNavigation);
      window.removeEventListener("popstate", onPopState);
    };
  }, []);

  useEffect(() => {
    let saveTimeout = 0;
    let lastSavedLeft = window.scrollX;
    let lastSavedTop = window.scrollY;

    const saveIfChanged = () => {
      saveTimeout = 0;

      if (window.scrollX === lastSavedLeft && window.scrollY === lastSavedTop) {
        return;
      }

      lastSavedLeft = window.scrollX;
      lastSavedTop = window.scrollY;
      saveCurrentScrollPosition();
    };

    const onScroll = () => {
      if (saveTimeout !== 0) {
        return;
      }

      saveTimeout = window.setTimeout(saveIfChanged, 200);
    };

    const flushScrollPosition = () => {
      if (saveTimeout !== 0) {
        window.clearTimeout(saveTimeout);
      }

      saveIfChanged();
    };

    window.addEventListener("scroll", onScroll, { passive: true });
    window.addEventListener("pagehide", flushScrollPosition);

    return () => {
      if (saveTimeout !== 0) {
        window.clearTimeout(saveTimeout);
      }
      window.removeEventListener("scroll", onScroll);
      window.removeEventListener("pagehide", flushScrollPosition);
    };
  }, []);

  useEffect(() => {
    let active = true;
    setCatalogState((current) => ({ ...current, loading: true, error: null }));
    api
      .fetchCatalog()
      .then((data) => {
        if (!active) return;
        setCatalogState({ data, loading: false, error: null });
      })
      .catch((error: Error) => {
        if (!active) return;
        setCatalogState({
          data: EMPTY_CATALOG,
          loading: false,
          error: error.message,
        });
      });

    return () => {
      active = false;
    };
  }, []);

  const pathname = navigationState.pathname;
  const route = useMemo(() => parseRoute(pathname), [pathname]);
  const catalog = catalogState.data ?? EMPTY_CATALOG;
  const sluggedWatchEntry =
    route.name === "watch-path"
      ? findEntryForWatchPath(catalog, route.creatorSlug, route.entrySlug)
      : undefined;
  const missingWatchEntry =
    route.name === "watch-path" && !catalogState.loading && !sluggedWatchEntry;
  const activeWatchEntryId = sluggedWatchEntry?.id;
  const isHistoryNavigation = navigationState.kind === "pop";

  useRestoreScrollOnHistoryNavigation(navigationState);
  useScrollToHashTarget(navigationState, catalogState.loading);
  useRoutePageTitle(route, missingWatchEntry);

  return (
    <div class="page-shell">
      <SiteHeader currentPathname={pathname} />
      <main class="site-main">
        {route.name === "home" && (
          <HomePage
            catalog={catalog}
            loading={catalogState.loading}
            error={catalogState.error}
            restoreFromHistory={isHistoryNavigation}
            {...optionalActiveEntryId(activeWatchEntryId)}
          />
        )}
        {route.name === "browse" && (
          <BrowsePage
            key={route.entryType}
            entryType={route.entryType}
            entries={entriesForType(catalog, route.entryType)}
            songAmvs={catalog.songAmvs}
            loading={catalogState.loading}
            error={catalogState.error}
            restoreFromHistory={isHistoryNavigation}
            {...optionalActiveEntryId(activeWatchEntryId)}
          />
        )}
        {route.name === "creator" && (
          <CreatorPage
            slug={route.slug}
            restoreFromHistory={isHistoryNavigation}
            {...optionalActiveEntryId(activeWatchEntryId)}
          />
        )}
        {route.name === "watch-path" &&
          (sluggedWatchEntry || catalogState.loading ? (
            <WatchPage
              entryId={sluggedWatchEntry?.id}
              entryTypeHint={sluggedWatchEntry?.type}
              restoreFromHistory={isHistoryNavigation}
              {...optionalEpisodeSlug(route.episodeSlug)}
            />
          ) : (
            <NotFoundPage />
          ))}
        {route.name === "not-found" && <NotFoundPage />}
      </main>
      <SiteFooter />
    </div>
  );
}

function useScrollToHashTarget(
  navigationState: NavigationState,
  loading: boolean,
): void {
  useLayoutEffect(() => {
    if (navigationState.kind === "pop" || loading || !window.location.hash) {
      return;
    }

    scrollPageToHash(window.location.hash);
  }, [navigationState, loading]);
}

function useRoutePageTitle(
  route: ReturnType<typeof parseRoute>,
  missingWatchEntry: boolean,
): void {
  useEffect(() => {
    if (missingWatchEntry) {
      setPageTitle("Not Found");
      return;
    }

    switch (route.name) {
      case "home":
        setPageTitle();
        return;
      case "browse":
        setPageTitle(TYPE_META[route.entryType].title);
        return;
      case "creator":
        setPageTitle(route.slug.replace(/-/g, " "));
        return;
      case "watch-path":
        return;
      case "not-found":
        setPageTitle("Not Found");
        return;
    }
  }, [route, missingWatchEntry]);
}

function useRestoreScrollOnHistoryNavigation(
  navigationState: NavigationState,
): void {
  const restoreToken = useRef(0);

  useLayoutEffect(() => {
    if (navigationState.kind !== "pop") {
      restoreToken.current += 1;
      return;
    }

    const targetPosition = currentHistoryScrollPosition();
    const token = restoreToken.current + 1;
    restoreToken.current = token;
    let frame = 0;
    let attempts = 0;
    const maxAttempts = 30;

    const restore = () => {
      if (restoreToken.current !== token) {
        return;
      }

      attempts += 1;
      scrollPageToPosition(targetPosition);

      const scrollingElement =
        document.scrollingElement ?? document.documentElement;
      const maxTop = Math.max(
        0,
        scrollingElement.scrollHeight - window.innerHeight,
      );
      const contentCanReachTarget = maxTop >= targetPosition.top;
      const restored =
        contentCanReachTarget &&
        Math.abs(window.scrollY - targetPosition.top) <= 1;

      if (restored || attempts >= maxAttempts) {
        return;
      }

      frame = window.requestAnimationFrame(restore);
    };

    frame = window.requestAnimationFrame(restore);

    return () => {
      restoreToken.current += 1;
      if (frame !== 0) {
        window.cancelAnimationFrame(frame);
      }
    };
  }, [navigationState]);
}

function optionalEpisodeSlug(episodeSlug: string | undefined): {
  episodeSlug?: string;
} {
  return episodeSlug ? { episodeSlug } : {};
}
