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
import type { CatalogResponse, EntryType } from "../lib/types";
import { SiteFooter } from "./components/SiteFooter";
import { SiteHeader } from "./components/SiteHeader";
import { EMPTY_CATALOG } from "./constants";
import { entriesForType } from "./entryUtils";
import { scrollPageToHash, scrollPageToPosition } from "./navigation";
import { BrowsePage } from "./pages/BrowsePage";
import { CreatorPage } from "./pages/CreatorPage";
import { HomePage } from "./pages/HomePage";
import { NotFoundPage } from "./pages/NotFoundPage";
import { WatchPage } from "./pages/WatchPage";
import { optionalActiveEntryId } from "./propUtils";
import type { AsyncState } from "./types";

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
    let animationFrame = 0;

    const onScroll = () => {
      if (animationFrame !== 0) {
        return;
      }

      animationFrame = window.requestAnimationFrame(() => {
        animationFrame = 0;
        saveCurrentScrollPosition();
      });
    };

    window.addEventListener("scroll", onScroll, { passive: true });

    return () => {
      if (animationFrame !== 0) {
        window.cancelAnimationFrame(animationFrame);
      }
      window.removeEventListener("scroll", onScroll);
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
  const activeWatchEntryId = route.name === "watch" ? route.entryId : undefined;
  const activeWatchEntryType = activeWatchEntryId
    ? entryTypeForId(catalog, activeWatchEntryId)
    : undefined;
  const isHistoryNavigation = navigationState.kind === "pop";

  useRestoreScrollOnHistoryNavigation(navigationState);
  useScrollToHashTarget(navigationState, catalogState.loading);

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
        {route.name === "watch" && (
          <WatchPage
            entryId={route.entryId}
            entryTypeHint={activeWatchEntryType}
            restoreFromHistory={isHistoryNavigation}
            {...optionalEpisodeId(route.episodeId)}
          />
        )}
        {route.name === "not-found" && <NotFoundPage />}
      </main>
      <SiteFooter />
    </div>
  );
}

function entryTypeForId(
  catalog: CatalogResponse,
  entryId: string,
): EntryType | undefined {
  for (const entry of [
    ...catalog.series,
    ...catalog.shorts,
    ...catalog.shots,
    ...catalog.songs,
    ...catalog.songAmvs,
  ]) {
    if (entry.id === entryId) {
      return entry.type;
    }
  }

  return undefined;
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

function optionalEpisodeId(episodeId: string | undefined): {
  episodeId?: string;
} {
  return episodeId ? { episodeId } : {};
}
