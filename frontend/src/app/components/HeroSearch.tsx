import type { JSX } from "preact";
import {
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "preact/hooks";
import { api } from "../../lib/api";
import { navigate } from "../../lib/router";
import type { CatalogResponse, EntrySummary } from "../../lib/types";
import { handleInternalLinkClick, scrollPageToTop } from "../navigation";
import {
  MAX_SEARCH_QUERY_LENGTH,
  SEARCH_TYPE_META,
  searchQueryTerms,
} from "../search";
import { entryWatchPath } from "../watchPaths";
import { SearchEntrySurprise } from "./SearchEntrySurprise";

export function HeroSearch({
  catalog,
  loading,
}: { catalog: CatalogResponse; loading: boolean }) {
  const [query, setQuery] = useState("");
  const [expanded, setExpanded] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [panelHeight, setPanelHeight] = useState(0);
  const [results, setResults] = useState<EntrySummary[]>([]);
  const [resultsLoading, setResultsLoading] = useState(false);
  const [surprisePunchline, setSurprisePunchline] = useState(false);
  const [shortcutModifier, setShortcutModifier] = useState("Ctrl");
  const rootRef = useRef<HTMLDivElement>(null);
  const fieldRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const terms = useMemo(() => searchQueryTerms(query), [query]);
  const hasQuery = terms.length > 0;
  const showResults = expanded && terms.length > 0;

  useEffect(() => {
    if (!expanded || hasQuery) setSurprisePunchline(false);
  }, [expanded, hasQuery]);

  useEffect(() => {
    if (!hasQuery) {
      setResults([]);
      setResultsLoading(false);
      return;
    }

    const controller = new AbortController();
    setResultsLoading(true);
    const timeout = window.setTimeout(() => {
      api
        .search(query, controller.signal)
        .then((entries) => {
          if (controller.signal.aborted) return;
          setResults(entries);
          setSelectedIndex(0);
        })
        .catch(() => {
          if (controller.signal.aborted) return;
          setResults([]);
        })
        .finally(() => {
          if (!controller.signal.aborted) setResultsLoading(false);
        });
    }, 300);

    return () => {
      window.clearTimeout(timeout);
      controller.abort();
    };
  }, [query, hasQuery]);

  const selected = results[selectedIndex];

  const focusSearch = () => {
    setExpanded(true);
    inputRef.current?.focus({ preventScroll: true });
  };
  const dismiss = () => {
    setExpanded(false);
    inputRef.current?.focus({ preventScroll: true });
  };
  const clearSearch = () => {
    setQuery("");
    setSelectedIndex(0);
    focusSearch();
  };
  const selectRelative = (direction: number) => {
    setExpanded(true);
    setSelectedIndex((index) =>
      results.length
        ? (index + direction + results.length) % results.length
        : 0,
    );
  };
  const openSelected = () => {
    if (!selected) return;
    navigate(entryWatchPath(selected));
    scrollPageToTop();
  };

  useEffect(() => {
    setShortcutModifier(
      /Mac|iPhone|iPad/.test(navigator.platform) ? "⌘" : "Ctrl",
    );
    const onKeyDown = (event: KeyboardEvent) => {
      if (
        (event.metaKey || event.ctrlKey) &&
        !event.altKey &&
        event.key.toLowerCase() === "k"
      ) {
        event.preventDefault();
        setExpanded(true);
        inputRef.current?.focus();
      }
    };
    const onPointerDown = (event: PointerEvent) => {
      if (
        event.target instanceof Node &&
        !rootRef.current?.contains(event.target)
      )
        setExpanded(false);
    };
    document.addEventListener("keydown", onKeyDown);
    document.addEventListener("pointerdown", onPointerDown);
    return () => {
      document.removeEventListener("keydown", onKeyDown);
      document.removeEventListener("pointerdown", onPointerDown);
    };
  }, []);

  useLayoutEffect(() => {
    if (!expanded) return;
    const measure = () => {
      const bottom = fieldRef.current?.getBoundingClientRect().bottom ?? 0;
      const viewport = window.visualViewport;
      const viewportBottom = viewport
        ? viewport.offsetTop + viewport.height
        : window.innerHeight;
      setPanelHeight(Math.max(0, viewportBottom - bottom - 16));
    };
    measure();
    const observer = new ResizeObserver(measure);
    if (fieldRef.current) observer.observe(fieldRef.current);
    window.addEventListener("resize", measure);
    window.addEventListener("scroll", measure, { passive: true });
    window.visualViewport?.addEventListener("resize", measure);
    window.visualViewport?.addEventListener("scroll", measure);
    return () => {
      observer.disconnect();
      window.removeEventListener("resize", measure);
      window.removeEventListener("scroll", measure);
      window.visualViewport?.removeEventListener("resize", measure);
      window.visualViewport?.removeEventListener("scroll", measure);
    };
  }, [expanded]);

  useLayoutEffect(() => {
    if (!showResults || !results.length || panelHeight === 0) return;
    const list = listRef.current;
    const option = list?.children[selectedIndex] as HTMLElement | undefined;
    if (!list || !option) return;
    if (option.offsetTop < list.scrollTop) list.scrollTop = option.offsetTop;
    else if (
      option.offsetTop + option.offsetHeight >
      list.scrollTop + list.clientHeight
    )
      list.scrollTop =
        option.offsetTop + option.offsetHeight - list.clientHeight;
  }, [showResults, selectedIndex, results, panelHeight]);

  const onInputKeyDown = (
    event: JSX.TargetedKeyboardEvent<HTMLInputElement>,
  ) => {
    if (event.isComposing) return;
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      if (!showResults) {
        setExpanded(true);
        setSelectedIndex(
          event.key === "ArrowUp" ? Math.max(0, results.length - 1) : 0,
        );
      } else selectRelative(event.key === "ArrowDown" ? 1 : -1);
    } else if (event.key === "Enter" && showResults && selected) {
      event.preventDefault();
      openSelected();
    }
  };

  const stats = [
    {
      label: "Series",
      href: "/series",
      count: catalog.series.length,
      className: "accent-shows",
    },
    {
      label: "Shots",
      href: "/shots",
      count: catalog.shots.length,
      className: "accent-one-shots",
    },
    {
      label: "Shorts",
      href: "/shorts",
      count: catalog.shorts.length,
      className: "accent-shorts",
    },
    {
      label: "Songs",
      href: "/songs",
      count: catalog.songs.length + catalog.songAmvs.length,
      className: "accent-songs",
    },
  ];

  return (
    <div
      ref={rootRef}
      class={`hero-search ${expanded ? "is-open" : ""}`}
      onKeyDown={(event) => {
        if (
          event.key === "Escape" &&
          (expanded || query.length > 0) &&
          !event.isComposing
        ) {
          event.preventDefault();
          setQuery("");
          setSelectedIndex(0);
          setExpanded(false);
          inputRef.current?.blur();
        }
      }}
    >
      <div ref={fieldRef} class="hero-search-field">
        <svg
          class="search-icon"
          width="21"
          height="21"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.6"
          aria-hidden="true"
        >
          <circle cx="10.8" cy="10.8" r="7.3" />
          <path d="m16.2 16.2 4.3 4.3" />
        </svg>
        <input
          ref={inputRef}
          id="catalogue-search"
          type="text"
          role="combobox"
          aria-label="Search the catalogue"
          aria-autocomplete="list"
          aria-expanded={expanded}
          aria-controls={expanded ? "catalogue-search-results" : undefined}
          aria-describedby={
            expanded && !hasQuery ? "search-welcome-hint" : undefined
          }
          aria-activedescendant={
            showResults && selected ? `search-option-${selected.id}` : undefined
          }
          autoComplete="off"
          spellcheck={false}
          maxLength={MAX_SEARCH_QUERY_LENGTH}
          placeholder="Search the catalogue…"
          value={query}
          onFocus={(event) => {
            if (
              !(event.relatedTarget instanceof Node) ||
              !rootRef.current?.contains(event.relatedTarget)
            )
              setExpanded(true);
          }}
          onPointerDown={() => setExpanded(true)}
          onInput={(event) => {
            setQuery(event.currentTarget.value);
            setSelectedIndex(0);
            setExpanded(true);
            if (listRef.current) listRef.current.scrollTop = 0;
          }}
          onKeyDown={onInputKeyDown}
        />
        {query ? (
          <button
            type="button"
            class="search-clear"
            aria-label="Clear search"
            onClick={clearSearch}
          >
            Clear
          </button>
        ) : null}
        {expanded ? (
          <button
            type="button"
            class="search-dismiss"
            aria-label="Close search"
            onClick={dismiss}
          >
            <svg
              width="16"
              height="16"
              viewBox="0 0 20 20"
              fill="none"
              stroke="currentColor"
              stroke-width="1.5"
              aria-hidden="true"
            >
              <path d="m5 5 10 10M15 5 5 15" />
            </svg>
          </button>
        ) : null}
        <button
          type="button"
          class="search-shortcut"
          aria-label="Focus search"
          onClick={focusSearch}
        >
          <kbd>{shortcutModifier}</kbd>
          <kbd>K</kbd>
        </button>
      </div>

      <nav
        class="search-stats"
        aria-label="Catalogue counts by format"
        inert={expanded}
      >
        {stats.map(({ label, href, count, className }) => (
          <a
            key={href}
            href={href}
            class={className}
            onClick={(event) =>
              handleInternalLinkClick(event, href, { scrollToTop: true })
            }
          >
            {label}: <span>{loading ? "…" : count.toLocaleString()}</span>
          </a>
        ))}
      </nav>

      {expanded ? (
        <div
          class="search-panel"
          style={{ height: `${panelHeight}px` }}
          onPointerDown={(event) => {
            if (
              results.length === 0 &&
              event.button === 0 &&
              event.target instanceof Element &&
              !event.target.closest("button, a, input")
            ) {
              event.preventDefault();
              focusSearch();
            }
          }}
        >
          <div
            ref={listRef}
            id="catalogue-search-results"
            class="search-results"
            // biome-ignore lint/a11y/useSemanticElements: Rich combobox suggestions require thumbnails and links instead of a native select.
            role="listbox"
            tabIndex={-1}
            aria-label="Search suggestions"
          >
            {results.map((entry, index) => (
              <a
                key={entry.id}
                href={entryWatchPath(entry)}
                tabIndex={-1}
                id={`search-option-${entry.id}`}
                // biome-ignore lint/a11y/useSemanticElements: Rich options are navigated through the combobox aria-activedescendant.
                role="option"
                aria-selected={selectedIndex === index}
                class={`search-option ${selectedIndex === index ? "is-selected" : ""}`}
                onPointerMove={() => setSelectedIndex(index)}
                onMouseDown={(event) => {
                  if (event.button === 0) event.preventDefault();
                }}
                onClick={(event) =>
                  handleInternalLinkClick(event, entryWatchPath(entry), {
                    scrollToTop: true,
                  })
                }
              >
                <span
                  class={`search-thumbnail ${SEARCH_TYPE_META[entry.type].className}`}
                >
                  {entry.poster.thumbnailUrl ? (
                    <img
                      src={entry.poster.thumbnailUrl}
                      alt=""
                      loading="lazy"
                      onError={(event) => {
                        event.currentTarget.hidden = true;
                      }}
                    />
                  ) : null}
                  <svg
                    width="20"
                    height="20"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.3"
                    aria-hidden="true"
                  >
                    <rect x="2" y="5" width="20" height="14" rx="2" />
                    <path d="m10 9 5 3-5 3Z" />
                  </svg>
                </span>
                <span class="search-entry-copy">
                  <span class="search-creator">
                    [<SearchHighlight text={entry.creator} terms={terms} />]
                  </span>
                  <span class="search-title">
                    <SearchHighlight text={entry.entryTitle} terms={terms} />
                  </span>
                </span>
                <span
                  class={`search-type ${SEARCH_TYPE_META[entry.type].className}`}
                >
                  <SearchHighlight
                    text={
                      entry.subtype === "amv"
                        ? "AMV"
                        : SEARCH_TYPE_META[entry.type].label
                    }
                    terms={terms}
                  />
                </span>
              </a>
            ))}
          </div>
          {!hasQuery ? (
            <SearchWelcome
              onBegin={() => setSurprisePunchline(false)}
              onCheerStart={() => setSurprisePunchline(true)}
            />
          ) : !resultsLoading && results.length === 0 ? (
            <div class="search-empty">
              <p>No entries found for “{query.trim()}”</p>
              <span>Try a title, creator, or format.</span>
              <button type="button" onClick={clearSearch}>
                Clear search
              </button>
            </div>
          ) : null}
          <div class="search-actions">
            <output class="search-result-count" aria-live="polite">
              {hasQuery ? (
                resultsLoading ? (
                  "Searching…"
                ) : (
                  `${results.length} ${results.length === 1 ? "result" : "results"}`
                )
              ) : (
                <span
                  class="search-status"
                  data-punchline={surprisePunchline ? "true" : "false"}
                >
                  <span
                    class="search-status-label search-status-ready"
                    aria-hidden={surprisePunchline}
                  >
                    Ready when you are
                  </span>
                  <span
                    class="search-status-label search-status-punchline"
                    aria-hidden={!surprisePunchline}
                  >
                    Woah! Absolutely nothing happened!!
                  </span>
                </span>
              )}
            </output>
            <span class="search-select-actions">
              <button
                type="button"
                aria-label="Select previous result"
                disabled={!results.length}
                onClick={() => selectRelative(-1)}
              >
                <kbd>↑</kbd>
              </button>
              <button
                type="button"
                aria-label="Select next result"
                disabled={!results.length}
                onClick={() => selectRelative(1)}
              >
                <kbd>↓</kbd>
              </button>
              <span>Select</span>
            </span>
            <button
              type="button"
              class="search-open-action"
              aria-label="Open"
              disabled={!selected}
              onClick={openSelected}
            >
              <kbd>↵</kbd>
              <span>Open</span>
            </button>
            <button
              type="button"
              class="search-close-action"
              aria-label="Close"
              onClick={dismiss}
            >
              <kbd>esc</kbd>
              <span>Close</span>
            </button>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function SearchWelcome({
  onBegin,
  onCheerStart,
}: {
  onBegin: () => void;
  onCheerStart: () => void;
}) {
  return (
    <div class="search-welcome">
      <div class="search-welcome-copy">
        <h2>Welcome to the catalogue.</h2>
        <p id="search-welcome-hint">
          Type a title, creator, or format to start searching.
        </p>
      </div>
      <SearchEntrySurprise onBegin={onBegin} onCheerStart={onCheerStart} />
    </div>
  );
}

function SearchHighlight({ text, terms }: { text: string; terms: string[] }) {
  if (terms.length === 0) return <>{text}</>;
  const expression = new RegExp(
    `(${[...terms]
      .sort((a, b) => b.length - a.length)
      .map((term) => term.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"))
      .join("|")})`,
    "gi",
  );
  let offset = 0;
  return (
    <>
      {text
        .split(expression)
        .filter(Boolean)
        .map((part) => {
          const key = offset;
          offset += part.length;
          return terms.includes(part.toLowerCase()) ? (
            <mark key={key}>{part}</mark>
          ) : (
            <span key={key}>{part}</span>
          );
        })}
    </>
  );
}
