import { useEffect, useRef, useState } from "preact/hooks";
import type { EntrySummary } from "../../lib/types";
import { LOADING_CARD_KEYS } from "../constants";
import { handleInternalLinkClick } from "../navigation";
import { BrowseCard } from "./BrowseCard";
import { SectionHeader } from "./SectionHeader";

const DEFAULT_VISIBLE_COUNT = 8;
const LIMITED_VISIBLE_COUNT = 4;
const INFINITE_SCROLL_BATCH_SIZE = 24;

export function BrowseSection({
  sectionId,
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
  hideSeriesDurations = false,
  loading = false,
}: {
  sectionId?: string;
  title: string;
  titleHref?: string;
  titleClassName?: string;
  entries: EntrySummary[];
  activeEntryId?: string;
  limited?: boolean;
  hideHeader?: boolean;
  compactHeader?: boolean;
  infiniteScroll?: boolean;
  showAllEntries?: boolean;
  animateCards?: boolean;
  hideSeriesDurations?: boolean;
  loading?: boolean;
}) {
  const initial = showAllEntries
    ? entries.length
    : limited
      ? LIMITED_VISIBLE_COUNT
      : infiniteScroll
        ? INFINITE_SCROLL_BATCH_SIZE
        : DEFAULT_VISIBLE_COUNT;
  const increment = limited
    ? LIMITED_VISIBLE_COUNT
    : infiniteScroll
      ? INFINITE_SCROLL_BATCH_SIZE
      : DEFAULT_VISIBLE_COUNT;
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

        setVisibleCount((count) => Math.min(count + increment, entries.length));
      },
      { rootMargin: "1200px 0px" },
    );

    observer.observe(loadMoreTarget);

    return () => {
      observer.disconnect();
    };
  }, [entries.length, increment, infiniteScroll, visibleCount]);

  return (
    <section id={sectionId} class="browse-section">
      <div class="container">
        {!hideHeader ? (
          <SectionHeader
            title={title}
            href={titleHref}
            className={titleClassName}
            compact={compactHeader}
          />
        ) : null}
        {loading ? (
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
        ) : entries.length === 0 ? (
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
                  animationDelay={index % increment}
                  hideDuration={hideSeriesDurations && entry.type === "series"}
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
