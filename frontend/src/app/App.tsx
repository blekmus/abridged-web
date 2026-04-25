import type { ComponentChildren, JSX } from "preact";
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
  navigate,
  parseRoute,
  saveCurrentScrollPosition,
} from "../lib/router";
import type { CatalogResponse, Entry, EntryType, Episode } from "../lib/types";

type AsyncState<T> = {
  data: T | null;
  loading: boolean;
  error: string | null;
};

type NavigationKind = "initial" | "push" | "pop";

type NavigationState = {
  kind: NavigationKind;
  pathname: string;
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
  short: {
    title: "Shorts",
    eyebrow: "Edits with practically no plot",
    path: "/shorts",
  },
  shot: {
    title: "Shots",
    eyebrow: "Single entry stories",
    path: "/shots",
  },
};

const CREATOR_SECTIONS: Array<{
  entryType: EntryType;
  title: string;
  titleClassName: string;
}> = [
  { entryType: "series", title: "Shows", titleClassName: "accent-shows" },
  { entryType: "short", title: "Shorts", titleClassName: "accent-shorts" },
  {
    entryType: "shot",
    title: "One-Shots",
    titleClassName: "accent-one-shots",
  },
];

const LOADING_CARD_KEYS = [
  "skeleton-1",
  "skeleton-2",
  "skeleton-3",
  "skeleton-4",
] as const;

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
  const isHistoryNavigation = navigationState.kind === "pop";

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
            key={route.entryType}
            entryType={route.entryType}
            entries={entriesForType(catalog, route.entryType)}
            loading={catalogState.loading}
            error={catalogState.error}
            restoreFromHistory={isHistoryNavigation}
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
      <SiteFooter />
    </div>
  );
}

function SiteFooter() {
  return (
    <footer class="site-footer">
      <div class="container footer-simple">
        <span class="footer-rule" aria-hidden="true" />
        <p class="footer-credit">
          Made with love by{" "}
          <a href="https://garden.dinil.dev" target="_blank" rel="noreferrer">
            blekmus
          </a>
        </p>
      </div>
    </footer>
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
          <a
            href="/"
            class="site-mark"
            onClick={(event) => {
              handleInternalLinkClick(event, "/");
            }}
          >
            The abridged catalogue
          </a>
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
            A heavily curated catalogue of my favourite abridged anime. Most of
            them are from youtube, where they have been banished to the aether.
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
          <HomeQuestions />
        </>
      )}
    </>
  );
}

function HomeQuestions() {
  const [openQuestion, setOpenQuestion] = useState<string | null>(
    "Why does this exist?",
  );
  const questions: Array<{ question: string; answer: ComponentChildren }> = [
    {
      question: "Why does this exist?",
      answer: (
        <p>
          I love going through this collection about once per year and I've done
          everything I can to make that as seamless as possible. Maybe I'm
          overengineering this a whole lot more than it needs to be, but I enjoy
          it. I've already built a{" "}
          <a
            href="https://github.com/blekmus/abridged-cli"
            target="_blank"
            rel="noreferrer"
          >
            CLI
          </a>{" "}
          and a{" "}
          <a href="https://github.com/blekmus/raycast-abridged">
            Raycast Plugin
          </a>{" "}
          so a website is sort of the natural evolution.
        </p>
      ),
    },
    {
      question: "Where can I find more?",
      answer: (
        <p>
          Once upon a time I made a{" "}
          <a
            href="https://nyaa.si/view/1979033"
            target="_blank"
            rel="noreferrer"
          >
            torrent
          </a>{" "}
          with a bunch of the entries here. Other than that, there's always the{" "}
          <a
            target="_blank"
            href="https://abridgedseries.fandom.com/wiki/Abridged_Archive"
            rel="noreferrer"
          >
            abridged archive.
          </a>
        </p>
      ),
    },
    {
      question: "Why are some classics missing?",
      answer: (
        <p>
          As much as I want to include every abridgedment out there, I'm limited
          by how much storage I have on my server. So I decided to not host the
          big titles that are already on youtube.
        </p>
      ),
    },
    {
      question: "How are shows, shorts, and one-shots split?",
      answer: (
        <>
          <p>
            <strong>Shows: </strong> A set of entries that run episodically or
            are part of a series of works by a creator or group of creators.
          </p>
          <p>
            <strong>One-shots: </strong> Single videos that are not part of a
            continuous series but share an overarching story. These are usually
            longer than 5 minutes.
          </p>
          <p>
            <strong>Shorts: </strong>Similar to Shots, but shorter in length and
            may not have a well-defined plot.
          </p>
        </>
      ),
    },
  ];

  return (
    <section class="qa-section">
      <div class="container qa-container">
        <div class="qa-intro">
          <h2 class="qa-title">Q&A</h2>
        </div>
        <div class="qa-list">
          {questions.map((item) => (
            <details
              class="qa-item"
              key={item.question}
              open={openQuestion === item.question}
            >
              <summary
                onClick={(event) => {
                  event.preventDefault();
                  setOpenQuestion((current) =>
                    current === item.question ? null : item.question,
                  );
                }}
                onKeyDown={(event) => {
                  if (event.key !== "Enter" && event.key !== " ") {
                    return;
                  }

                  event.preventDefault();
                  setOpenQuestion((current) =>
                    current === item.question ? null : item.question,
                  );
                }}
              >
                {item.question}
              </summary>
              <div class="qa-answer">{item.answer}</div>
            </details>
          ))}
        </div>
      </div>
    </section>
  );
}

function BrowsePage({
  entryType,
  entries,
  loading,
  error,
  activeEntryId,
  restoreFromHistory,
}: {
  entryType: EntryType;
  entries: Entry[];
  loading: boolean;
  error: string | null;
  activeEntryId?: string;
  restoreFromHistory: boolean;
}) {
  const meta = TYPE_META[entryType];

  useLayoutEffect(() => {
    if (!restoreFromHistory || loading) {
      return;
    }

    scrollPageToPosition(currentHistoryScrollPosition());
  }, [loading, restoreFromHistory]);

  return (
    <section class="browse-page">
      <div class="container">
        <p class="eyebrow">{meta.eyebrow}</p>
        <SectionHeader
          title={meta.title}
          className={accentClassForEntryType(entryType)}
        />
        {error ? <InlineError message={error} /> : null}
        {loading ? (
          <LoadingGrid />
        ) : (
          <BrowseSection
            key={entryType}
            title={meta.title}
            entries={entries}
            {...optionalActiveEntryId(activeEntryId)}
            hideHeader
            infiniteScroll
            showAllEntries={restoreFromHistory}
            animateCards={!restoreFromHistory}
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

  const entries = state.data ?? [];
  const title = entries[0]?.creator ?? slug.replace(/-/g, " ");
  const groupedEntries = groupEntriesByType(entries);

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
        ) : entries.length > 0 ? (
          <div class="creator-sections">
            {CREATOR_SECTIONS.map((section) => {
              const sectionEntries = groupedEntries[section.entryType];
              if (sectionEntries.length === 0) {
                return null;
              }

              return (
                <BrowseSection
                  key={section.entryType}
                  title={section.title}
                  titleClassName={section.titleClassName}
                  entries={sectionEntries}
                  {...optionalActiveEntryId(activeEntryId)}
                  showAllEntries
                  compactHeader
                />
              );
            })}
          </div>
        ) : (
          <p class="empty-state">No items found for this creator.</p>
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
        <div class="container watch-container">
          <div class="watch-intro">
            <div class="loading-line short" />
            <div class="loading-line title" />
            <div class="loading-line" />
          </div>
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
  const description = episode.description || entry.description;
  const episodeDate = formatEpisodeDate(episode.date);

  return (
    <section class={`watch-page watch-page-${entry.type}`}>
      <div class="container watch-container">
        {entry.type === "series" ? (
          <div class="watch-intro">
            <h1 class="watch-entry-title">{entry.entryTitle}</h1>
          </div>
        ) : null}

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
              <p class="video-title">
                {episode.displayTitle || episode.videoTitle}
              </p>
              <div class="watch-credit">
                <p>
                  <a
                    href={TYPE_META[entry.type].path}
                    class={`watch-type-tag ${accentClassForEntryType(entry.type)}`}
                    onClick={(event) => {
                      handleInternalLinkClick(
                        event,
                        TYPE_META[entry.type].path,
                      );
                    }}
                  >
                    {watchTypeLabel(entry.type)}
                  </a>
                </p>
                <p>
                  Produced by{" "}
                  <a
                    href={`/creator/${entry.creatorSlug}`}
                    class="watch-credit-link"
                    onClick={(event) => {
                      handleInternalLinkClick(
                        event,
                        `/creator/${entry.creatorSlug}`,
                      );
                    }}
                  >
                    {entry.creator}
                  </a>
                </p>
                {episodeDate ? <p>Uploaded on {episodeDate}</p> : null}
              </div>
            </div>

            {description ? (
              <DescriptionBlock key={description} text={description} />
            ) : null}

            {entry.type === "series" ? (
              <section class="episode-section">
                <h2 class="watch-section-title">
                  <span>Episodes</span>
                </h2>
                <div class="watch-metadata-grid card-grid">
                  {entry.episodes.map((item) => (
                    <article
                      key={item.id}
                      class={`browse-card watch-episode-card ${item.id === episode.id ? "is-active" : ""}`}
                    >
                      <a
                        href={`/watch/${entry.id}/${item.id}`}
                        class="card-thumbnail-shell"
                        onClick={(event) => {
                          handleInternalLinkClick(
                            event,
                            `/watch/${entry.id}/${item.id}`,
                            { scrollToTop: true },
                          );
                        }}
                      >
                        <Thumbnail episode={item} title={entry.entryTitle} />
                      </a>
                      <div class="card-copy">
                        <a
                          href={`/watch/${entry.id}/${item.id}`}
                          class="card-title-link"
                          onClick={(event) => {
                            handleInternalLinkClick(
                              event,
                              `/watch/${entry.id}/${item.id}`,
                              { scrollToTop: true },
                            );
                          }}
                        >
                          <h3 class="card-title">
                            {item.displayTitle || item.videoTitle}
                          </h3>
                          {item.date ? (
                            <p class="card-submeta">
                              {formatEpisodeDate(item.date)}
                            </p>
                          ) : null}
                        </a>
                      </div>
                    </article>
                  ))}
                </div>
              </section>
            ) : null}
          </div>
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
  compactHeader = false,
  infiniteScroll = false,
  showAllEntries = false,
  animateCards = true,
}: {
  title: string;
  titleHref?: string;
  titleClassName?: string;
  entries: Entry[];
  activeEntryId?: string;
  limited?: boolean;
  hideHeader?: boolean;
  compactHeader?: boolean;
  infiniteScroll?: boolean;
  showAllEntries?: boolean;
  animateCards?: boolean;
}) {
  const initial = showAllEntries ? entries.length : limited ? 4 : 8;
  const increment = limited ? 4 : 8;
  const [visibleCount, setVisibleCount] = useState(initial);
  const loadMoreRef = useRef<HTMLDivElement | null>(null);
  const visibleEntries = entries.slice(0, visibleCount);
  const hasMoreEntries = visibleCount < entries.length;

  useEffect(() => {
    setVisibleCount(initial);
  }, [initial]);

  useEffect(() => {
    if (!infiniteScroll || visibleCount >= entries.length) {
      return;
    }

    const loadMoreTarget = loadMoreRef.current;
    if (!loadMoreTarget) {
      return;
    }

    if (!("IntersectionObserver" in window)) {
      setVisibleCount(entries.length);
      return;
    }

    const observer = new IntersectionObserver(
      (items) => {
        if (!items.some((item) => item.isIntersecting)) {
          return;
        }

        setVisibleCount(entries.length);
      },
      { rootMargin: "1200px 0px" },
    );

    observer.observe(loadMoreTarget);

    return () => {
      observer.disconnect();
    };
  }, [entries.length, infiniteScroll, visibleCount]);

  return (
    <section class="browse-section">
      <div class="container">
        {!hideHeader ? (
          <SectionHeader
            title={title}
            href={titleHref}
            className={titleClassName}
            compact={compactHeader}
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
                  animate={animateCards}
                  animationDelay={index}
                />
              ))}
            </div>
            {infiniteScroll && hasMoreEntries ? (
              <div
                ref={loadMoreRef}
                class="infinite-scroll-sentinel"
                aria-hidden="true"
              />
            ) : null}
            {!infiniteScroll && hasMoreEntries ? (
              titleHref ? (
                <a
                  href={titleHref}
                  class="show-more"
                  onClick={(event) => {
                    handleInternalLinkClick(event, titleHref, {
                      scrollToTop: true,
                    });
                  }}
                >
                  Show more
                </a>
              ) : (
                <button
                  type="button"
                  class="show-more"
                  onClick={() => setVisibleCount((count) => count + increment)}
                >
                  Show more
                </button>
              )
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
  animate,
  animationDelay,
}: {
  entry: Entry;
  active: boolean;
  animate: boolean;
  animationDelay: number;
}) {
  const poster = findPosterEpisode(entry);

  return (
    <article
      class={`browse-card ${active ? "is-active" : ""} ${animate ? "" : "skip-enter-animation"}`}
      style={animate ? { animationDelay: `${animationDelay * 60}ms` } : {}}
    >
      <a
        href={`/watch/${entry.id}`}
        class="card-thumbnail-shell"
        onClick={(event) => {
          handleInternalLinkClick(event, `/watch/${entry.id}`, {
            scrollToTop: true,
          });
        }}
      >
        <Thumbnail episode={poster} title={entry.entryTitle} />
      </a>
      <div class="card-copy">
        <a
          href={`/creator/${entry.creatorSlug}`}
          class="card-meta"
          onClick={(event) => {
            handleInternalLinkClick(event, `/creator/${entry.creatorSlug}`);
          }}
        >
          {entry.creator}
        </a>
        <a
          href={`/watch/${entry.id}`}
          class="card-title-link"
          onClick={(event) => {
            handleInternalLinkClick(event, `/watch/${entry.id}`, {
              scrollToTop: true,
            });
          }}
        >
          <h3 class="card-title">{entry.entryTitle}</h3>
        </a>
      </div>
    </article>
  );
}

function Thumbnail({ episode, title }: { episode: Episode; title: string }) {
  return (
    <>
      {episode.thumbnailUrl ? (
        <img
          class="thumb"
          src={episode.thumbnailUrl}
          alt={`${title} thumbnail`}
          loading="lazy"
          decoding="async"
        />
      ) : (
        <div class="thumb thumb-fallback">
          <span>{title}</span>
        </div>
      )}
      {episode.durationSeconds > 0 ? (
        <span class="thumb-duration">
          {formatDurationLabel(episode.durationSeconds)}
        </span>
      ) : null}
    </>
  );
}

function DescriptionBlock({ text }: { text: string }) {
  const [expanded, setExpanded] = useState(false);
  const [canExpand, setCanExpand] = useState(false);
  const copyRef = useRef<HTMLDivElement | null>(null);
  const paragraphs = text
    .split(/\n{2,}/)
    .map((paragraph) => paragraph.trim())
    .filter(Boolean);
  const isCollapsed = canExpand && !expanded;

  useLayoutEffect(() => {
    setExpanded(false);
  }, []);

  useLayoutEffect(() => {
    const copy = copyRef.current;
    if (!copy) {
      return;
    }

    const updateCanExpand = () => {
      const paragraph = copy.querySelector("p");
      const lineHeight = Number.parseFloat(
        window.getComputedStyle(paragraph ?? copy).lineHeight,
      );
      const collapsedHeight = lineHeight * 6;

      setCanExpand(copy.scrollHeight > collapsedHeight + 1);
    };

    updateCanExpand();

    if (typeof ResizeObserver === "undefined") {
      window.addEventListener("resize", updateCanExpand);
      return () => {
        window.removeEventListener("resize", updateCanExpand);
      };
    }

    const observer = new ResizeObserver(updateCanExpand);
    observer.observe(copy);

    return () => {
      observer.disconnect();
    };
  }, []);

  return (
    <section class="description-section">
      <h2 class="watch-section-title">
        <span>Youtube Description</span>
      </h2>
      <div class="description-block">
        <div
          class={`description-copy ${isCollapsed ? "collapsed" : "expanded"}`}
        >
          <div ref={copyRef}>
            {paragraphs.map((paragraph) => (
              <p key={paragraph}>{paragraph}</p>
            ))}
          </div>
        </div>
        {canExpand ? (
          <button
            type="button"
            class="description-toggle"
            onClick={() => setExpanded((value) => !value)}
          >
            {expanded ? "Collapse" : "Read more"}
          </button>
        ) : null}
      </div>
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
  href?: string | undefined;
  className?: string | undefined;
  compact?: boolean | undefined;
}) {
  return (
    <div class={`section-header ${compact ? "is-compact" : ""}`}>
      <h2>
        {href ? (
          <a
            href={href}
            class={`section-title-link ${className ?? ""}`}
            onClick={(event) => {
              handleInternalLinkClick(event, href);
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
        handleInternalLinkClick(event, href);
      }}
    >
      {children}
    </a>
  );
}

function handleInternalLinkClick(
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

function scrollPageToTop(): void {
  scrollPageToPosition({ top: 0, left: 0 });
}

function scrollPageToPosition(position: { left: number; top: number }): void {
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

function groupEntriesByType(entries: Entry[]): Record<EntryType, Entry[]> {
  return {
    series: entries.filter((entry) => entry.type === "series"),
    short: entries.filter((entry) => entry.type === "short"),
    shot: entries.filter((entry) => entry.type === "shot"),
  };
}

function watchTypeLabel(entryType: EntryType): string {
  switch (entryType) {
    case "series":
      return "Show";
    case "short":
      return "Short";
    case "shot":
      return "One-Shot";
  }
}

function accentClassForEntryType(entryType: EntryType): string {
  switch (entryType) {
    case "series":
      return "accent-shows";
    case "short":
      return "accent-shorts";
    case "shot":
      return "accent-one-shots";
  }
}

function formatEpisodeDate(rawDate?: string): string | undefined {
  const trimmed = rawDate?.trim();
  if (!trimmed) {
    return undefined;
  }

  const dateMatch =
    /^(\d{4})-(\d{2})-(\d{2})$/.exec(trimmed) ??
    /^(\d{4})(\d{2})(\d{2})$/.exec(trimmed);
  if (!dateMatch) {
    return trimmed;
  }

  const [, year, month, day] = dateMatch;
  const utcDate = new Date(`${year}-${month}-${day}T00:00:00Z`);
  if (Number.isNaN(utcDate.getTime())) {
    return trimmed;
  }

  return new Intl.DateTimeFormat("en-US", {
    month: "long",
    day: "2-digit",
    year: "numeric",
    timeZone: "UTC",
  }).format(utcDate);
}

function formatDurationLabel(durationSeconds: number): string {
  const totalSeconds = Math.max(0, Math.floor(durationSeconds));
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  if (hours > 0) {
    return `${hours}:${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
  }

  return `${minutes}:${String(seconds).padStart(2, "0")}`;
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
