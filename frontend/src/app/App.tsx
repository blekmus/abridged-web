import type { ComponentChildren } from "preact";
import { useEffect, useMemo, useState } from "preact/hooks";
import { api } from "../lib/api";
import { currentPath, navigate, parseRoute } from "../lib/router";
import type { CatalogResponse, Entry, EntryType, Episode } from "../lib/types";

type AsyncState<T> = {
  data: T | null;
  loading: boolean;
  error: string | null;
};

const EMPTY_CATALOG: CatalogResponse = {
  series: [],
  shorts: [],
  shots: [],
};

const TYPE_META: Record<
  EntryType,
  { title: string; eyebrow: string; path: string }
> = {
  series: { title: "Series", eyebrow: "Multi-episode runs", path: "/series" },
  short: { title: "Shorts", eyebrow: "Single-release edits", path: "/shorts" },
  shot: { title: "Shots", eyebrow: "Quick-hit clips", path: "/shots" },
};

const LOADING_CARD_KEYS = [
  "skeleton-1",
  "skeleton-2",
  "skeleton-3",
  "skeleton-4",
] as const;

export function App() {
  const [pathname, setPathname] = useState(currentPath());
  const [catalogState, setCatalogState] = useState<AsyncState<CatalogResponse>>(
    {
      data: null,
      loading: true,
      error: null,
    },
  );

  useEffect(() => {
    const onPopState = () => setPathname(currentPath());
    window.addEventListener("popstate", onPopState);
    return () => window.removeEventListener("popstate", onPopState);
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

  const route = useMemo(() => parseRoute(pathname), [pathname]);
  const catalog = catalogState.data ?? EMPTY_CATALOG;
  const activeWatchEntryId = route.name === "watch" ? route.entryId : undefined;

  return (
    <div class="page-shell">
      <SiteHeader currentPathname={pathname} />
      <main class="site-main">
        {route.name === "home" && (
          <HomePage
            catalog={catalog}
            loading={catalogState.loading}
            error={catalogState.error}
            {...optionalActiveEntryId(activeWatchEntryId)}
          />
        )}
        {route.name === "browse" && (
          <BrowsePage
            entryType={route.entryType}
            entries={entriesForType(catalog, route.entryType)}
            loading={catalogState.loading}
            error={catalogState.error}
            {...optionalActiveEntryId(activeWatchEntryId)}
          />
        )}
        {route.name === "creator" && (
          <CreatorPage
            slug={route.slug}
            {...optionalActiveEntryId(activeWatchEntryId)}
          />
        )}
        {route.name === "watch" && (
          <WatchPage
            entryId={route.entryId}
            {...optionalEpisodeId(route.episodeId)}
          />
        )}
        {route.name === "not-found" && <NotFoundPage />}
      </main>
    </div>
  );
}

function SiteHeader({ currentPathname }: { currentPathname: string }) {
  const links = [
    { href: "/series", label: "Shows", accentClass: "accent-shows" },
    { href: "/shorts", label: "Shorts", accentClass: "accent-shorts" },
    { href: "/shots", label: "One-Shots", accentClass: "accent-one-shots" },
  ];

  return (
    <header class="site-header">
      <div class="container site-header-row">
        <div class="site-header-rail">
          <button type="button" class="site-mark" onClick={() => navigate("/")}>
            The abridged catalogue
          </button>
        </div>
        <nav class="site-nav" aria-label="Primary">
          {links.map((link) => (
            <NavLink
              key={link.href}
              href={link.href}
              active={currentPathname === link.href}
              className={link.accentClass}
            >
              {link.label}
            </NavLink>
          ))}
        </nav>
      </div>
    </header>
  );
}

function HomePage({
  catalog,
  loading,
  error,
  activeEntryId,
}: {
  catalog: CatalogResponse;
  loading: boolean;
  error: string | null;
  activeEntryId?: string;
}) {
  const heroStats = [
    { label: "shows", count: catalog.series.length },
    { label: "shorts", count: catalog.shorts.length },
    { label: "one-shots", count: catalog.shots.length },
  ];

  return (
    <>
      <section class="hero">
        <div class="container hero-grid">
          <div>
            <h1 class="hero-title">
              <span style={{ color: "#52dfff" }}>The</span>
              <br />
              <span style={{ color: "#34dcba" }}>Abridged</span>
              <br />
              <span style={{ color: "#ef3e78" }}>Catalogue</span>
            </h1>
          </div>
          <div class="hero-panel" aria-label="Catalog counts">
            {heroStats.map((stat) => (
              <div class={`hero-stat ${stat.label}-stat`} key={stat.label}>
                <strong>{loading ? "..." : stat.count}</strong>
                <span class="hero-stat-label">{stat.label}</span>
              </div>
            ))}
          </div>

          <p class="hero-copy">
            The browse surface stays quiet. Thumbnails, titles, and creator
            names route directly to the player.
          </p>
        </div>
      </section>

      {error ? <InlineError message={error} /> : null}
      {loading ? <LoadingGrid /> : null}

      {!loading && (
        <>
          <BrowseSection
            title="Shows"
            titleHref="/series"
            titleClassName="accent-shows"
            entries={catalog.series}
            {...optionalActiveEntryId(activeEntryId)}
            limited
          />
          <BrowseSection
            title="Shorts"
            titleHref="/shorts"
            titleClassName="accent-shorts"
            entries={catalog.shorts}
            {...optionalActiveEntryId(activeEntryId)}
            limited
          />
          <BrowseSection
            title="One-Shots"
            titleHref="/shots"
            titleClassName="accent-one-shots"
            entries={catalog.shots}
            {...optionalActiveEntryId(activeEntryId)}
            limited
          />
        </>
      )}
    </>
  );
}

function BrowsePage({
  entryType,
  entries,
  loading,
  error,
  activeEntryId,
}: {
  entryType: EntryType;
  entries: Entry[];
  loading: boolean;
  error: string | null;
  activeEntryId?: string;
}) {
  const meta = TYPE_META[entryType];

  return (
    <section class="browse-page">
      <div class="container">
        <p class="eyebrow">{meta.eyebrow}</p>
        <SectionHeader title={meta.title} />
        {error ? <InlineError message={error} /> : null}
        {loading ? (
          <LoadingGrid />
        ) : (
          <BrowseSection
            title={meta.title}
            entries={entries}
            {...optionalActiveEntryId(activeEntryId)}
            hideHeader
          />
        )}
      </div>
    </section>
  );
}

function CreatorPage({
  slug,
  activeEntryId,
}: {
  slug: string;
  activeEntryId?: string;
}) {
  const [state, setState] = useState<AsyncState<Entry[]>>({
    data: null,
    loading: true,
    error: null,
  });

  useEffect(() => {
    let active = true;
    setState({ data: null, loading: true, error: null });
    api
      .fetchCreator(slug)
      .then((data) => {
        if (!active) return;
        setState({ data, loading: false, error: null });
      })
      .catch((error: Error) => {
        if (!active) return;
        setState({ data: [], loading: false, error: error.message });
      });

    return () => {
      active = false;
    };
  }, [slug]);

  const title = state.data?.[0]?.creator ?? slug.replace(/-/g, " ");

  return (
    <section class="browse-page">
      <div class="container">
        <p class="eyebrow">Creator catalog</p>
        <header class="creator-header">
          <h1 class="creator-title">{title}</h1>
          <p class="creator-count">{state.data?.length ?? 0} titles</p>
        </header>
        {state.error ? <InlineError message={state.error} /> : null}
        {state.loading ? (
          <LoadingGrid />
        ) : (
          <BrowseSection
            title={title}
            entries={state.data ?? []}
            {...optionalActiveEntryId(activeEntryId)}
            hideHeader
          />
        )}
      </div>
    </section>
  );
}

function WatchPage({
  entryId,
  episodeId,
}: {
  entryId: string;
  episodeId?: string;
}) {
  const [state, setState] = useState<AsyncState<Entry>>({
    data: null,
    loading: true,
    error: null,
  });

  useEffect(() => {
    let active = true;
    setState({ data: null, loading: true, error: null });
    api
      .fetchEntry(entryId)
      .then((data) => {
        if (!active) return;
        setState({ data, loading: false, error: null });
      })
      .catch((error: Error) => {
        if (!active) return;
        setState({ data: null, loading: false, error: error.message });
      });

    return () => {
      active = false;
    };
  }, [entryId]);

  if (state.loading) {
    return (
      <section class="watch-page">
        <div class="container">
          <div class="player-shell loading-block" />
          <div class="watch-layout">
            <div class="loading-copy" />
            <div class="loading-copy" />
          </div>
        </div>
      </section>
    );
  }

  if (state.error || !state.data) {
    return <InlineError message={state.error ?? "Entry not found."} />;
  }

  const entry = state.data;
  const episode = chooseEpisode(entry, episodeId);

  return (
    <section class="watch-page">
      <div class="container">
        {/* <div class="watch-breadcrumbs">
          <NavLink href={TYPE_META[entry.type].path} active={false}>
            {TYPE_META[entry.type].title}
          </NavLink>
          <span>/</span>
          <NavLink href={`/creator/${entry.creatorSlug}`} active={false}>
            {entry.creator}
          </NavLink>
        </div> */}

        <div class="player-shell">
          <video
            class="video-player"
            controls
            preload="metadata"
            src={episode.videoUrl}
            poster={episode.thumbnailUrl}
          />
        </div>

        <div class="watch-layout">
          <div class="watch-main">
            <div class="metadata-block">
              <p>
                Produced by {" "}
                <button
                  type="button"
                  class="creator-link card-meta"
                  onClick={() => navigate(`/creator/${entry.creatorSlug}`)}
                >
                  {entry.creator}
                </button>
              </p>
              <p class="video-title">{episode.videoTitle}</p>
              {/* <h1 class="entry-title">{entry.entryTitle}</h1> */}
            </div>

            {entry.description ? (
              <DescriptionBlock text={entry.description} />
            ) : null}
          </div>

          {entry.type === "series" && (
            <aside class="episode-column">
              <SectionHeader title="Episodes" compact />
              <div class="episode-list">
                {entry.episodes.map((item) => (
                  <button
                    key={item.id}
                    type="button"
                    class={`episode-row ${item.id === episode.id ? "is-active" : ""}`}
                    onClick={() => navigate(`/watch/${entry.id}/${item.id}`)}
                  >
                    <Thumbnail episode={item} title={entry.entryTitle} />
                    <span class="episode-copy">
                      <span class="episode-label">{item.label}</span>
                      <span class="episode-title">{item.videoTitle}</span>
                    </span>
                  </button>
                ))}
              </div>
            </aside>
          )}
        </div>
      </div>
    </section>
  );
}

function BrowseSection({
  title,
  titleHref,
  titleClassName,
  entries,
  activeEntryId,
  limited = false,
  hideHeader = false,
}: {
  title: string;
  titleHref?: string;
  titleClassName?: string;
  entries: Entry[];
  activeEntryId?: string;
  limited?: boolean;
  hideHeader?: boolean;
}) {
  const initial = limited ? 4 : 8;
  const increment = limited ? 4 : 8;
  const [visibleCount, setVisibleCount] = useState(initial);
  const visibleEntries = entries.slice(0, visibleCount);

  useEffect(() => {
    setVisibleCount(initial);
  }, [initial]);

  return (
    <section class="browse-section">
      <div class="container">
        {!hideHeader ? (
          <SectionHeader
            title={title}
            href={titleHref}
            className={titleClassName}
          />
        ) : null}
        {entries.length === 0 ? (
          <p class="empty-state">No items found in this section.</p>
        ) : (
          <>
            <div class="card-grid">
              {visibleEntries.map((entry, index) => (
                <BrowseCard
                  key={entry.id}
                  entry={entry}
                  active={activeEntryId === entry.id}
                  animationDelay={index}
                />
              ))}
            </div>
            {visibleCount < entries.length ? (
              <button
                type="button"
                class="show-more"
                onClick={() => setVisibleCount((count) => count + increment)}
              >
                Show more
              </button>
            ) : null}
          </>
        )}
      </div>
    </section>
  );
}

function BrowseCard({
  entry,
  active,
  animationDelay,
}: {
  entry: Entry;
  active: boolean;
  animationDelay: number;
}) {
  const poster = findPosterEpisode(entry);

  return (
    <button
      type="button"
      class={`browse-card ${active ? "is-active" : ""}`}
      style={{ animationDelay: `${animationDelay * 60}ms` }}
      onClick={() => navigate(`/watch/${entry.id}`)}
    >
      <div class="card-thumbnail-shell">
        <Thumbnail episode={poster} title={entry.entryTitle} />
      </div>
      <div class="card-copy">
        <span class="card-meta">{entry.creator}</span>
        <h3 class="card-title">{entry.entryTitle}</h3>
      </div>
    </button>
  );
}

function Thumbnail({ episode, title }: { episode: Episode; title: string }) {
  if (episode.thumbnailUrl) {
    return (
      <img
        class="thumb"
        src={episode.thumbnailUrl}
        alt={`${title} thumbnail`}
        loading="lazy"
      />
    );
  }

  return (
    <div class="thumb thumb-fallback">
      <span>{title}</span>
    </div>
  );
}

function DescriptionBlock({ text }: { text: string }) {
  const [expanded, setExpanded] = useState(false);

  return (
    <section class="description-block">
      <div class={`description-copy ${expanded ? "expanded" : "collapsed"}`}>
        <div>
          <p>{text}</p>
        </div>
      </div>
      <button
        type="button"
        class="description-toggle"
        onClick={() => setExpanded((value) => !value)}
      >
        {expanded ? "Collapse" : "Read more"}
      </button>
    </section>
  );
}

function SectionHeader({
  title,
  href,
  className,
  compact = false,
}: {
  title: string;
  href?: string;
  className?: string;
  compact?: boolean;
}) {
  return (
    <div class={`section-header ${compact ? "is-compact" : ""}`}>
      <h2>
        {href ? (
          <a
            href={href}
            class={`section-title-link ${className ?? ""}`}
            onClick={(event) => {
              event.preventDefault();
              navigate(href);
            }}
          >
            {title}
          </a>
        ) : (
          <span class={`section-title-text ${className ?? ""}`}>{title}</span>
        )}
      </h2>
      <span class="section-rule" />
    </div>
  );
}

function NavLink({
  href,
  active,
  className,
  children,
}: {
  href: string;
  active: boolean;
  className?: string;
  children: ComponentChildren;
}) {
  return (
    <a
      href={href}
      class={`nav-link ${className ?? ""} ${active ? "is-active" : ""}`}
      onClick={(event) => {
        event.preventDefault();
        navigate(href);
      }}
    >
      {children}
    </a>
  );
}

function LoadingGrid() {
  return (
    <div class="container">
      <div class="card-grid">
        {LOADING_CARD_KEYS.map((key) => (
          <div key={key} class="browse-card skeleton-card">
            <div class="card-thumbnail-shell skeleton-block" />
            <div class="card-copy">
              <div class="skeleton-line short" />
              <div class="skeleton-line" />
              <div class="skeleton-line short" />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function InlineError({ message }: { message: string }) {
  return (
    <div class="container">
      <div class="error-banner">{message}</div>
    </div>
  );
}

function NotFoundPage() {
  return (
    <section class="browse-page">
      <div class="container">
        <SectionHeader title="Not found" />
        <p class="empty-state">
          This route does not exist in the current archive.
        </p>
      </div>
    </section>
  );
}

function entriesForType(
  catalog: CatalogResponse,
  entryType: EntryType,
): Entry[] {
  switch (entryType) {
    case "series":
      return catalog.series;
    case "short":
      return catalog.shorts;
    case "shot":
      return catalog.shots;
  }
}

function chooseEpisode(entry: Entry, episodeId?: string): Episode {
  if (episodeId) {
    const matched = entry.episodes.find((episode) => episode.id === episodeId);
    if (matched) {
      return matched;
    }
  }

  return (
    entry.episodes.find((episode) => episode.id === entry.defaultEpisodeId) ??
    entry.episodes[0] ??
    missingEpisode(entry.id)
  );
}

function findPosterEpisode(entry: Entry): Episode {
  return (
    entry.episodes.find((episode) => episode.id === entry.posterEpisodeId) ??
    entry.episodes[0] ??
    missingEpisode(entry.id)
  );
}

function missingEpisode(entryId: string): never {
  throw new Error(`Entry ${entryId} does not have any episodes.`);
}

function optionalActiveEntryId(activeEntryId: string | undefined): {
  activeEntryId?: string;
} {
  return activeEntryId ? { activeEntryId } : {};
}

function optionalEpisodeId(episodeId: string | undefined): {
  episodeId?: string;
} {
  return episodeId ? { episodeId } : {};
}
