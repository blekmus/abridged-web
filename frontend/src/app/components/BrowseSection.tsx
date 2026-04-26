import { useEffect, useRef, useState } from "preact/hooks";
import type { Entry } from "../../lib/types";
import { handleInternalLinkClick } from "../navigation";
import { BrowseCard } from "./BrowseCard";
import { SectionHeader } from "./SectionHeader";

export function BrowseSection({
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
  hideSeriesDurations?: boolean;
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
