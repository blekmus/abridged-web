import { useLayoutEffect, useRef, useState } from "preact/hooks";
import type { RecentlyAddedItem } from "../../lib/types";
import { LOADING_CARD_KEYS } from "../constants";
import { BrowseCard } from "./BrowseCard";
import { SectionHeader } from "./SectionHeader";

// One row on multi-column layouts; a fixed count once cards stack vertically.
const STACKED_VISIBLE_COUNT = 6;

export function RecentlyAdded({
  items,
  loading,
  animateCards,
}: {
  items: RecentlyAddedItem[];
  loading: boolean;
  animateCards: boolean;
}) {
  const gridRef = useRef<HTMLDivElement | null>(null);
  const visibleCount = useOneRowCount(gridRef, !loading && items.length > 0);

  if (!loading && items.length === 0) {
    return null;
  }

  return (
    <section class="browse-section recently-added">
      <div class="container">
        <SectionHeader title="Recently Added" className="accent-recent" />
        {loading ? (
          <div class="card-grid">
            {LOADING_CARD_KEYS.map((key) => (
              <div key={key} class="browse-card skeleton-card">
                <div class="card-thumbnail-shell skeleton-block" />
                <div class="card-copy">
                  <div class="skeleton-line short" />
                  <div class="skeleton-line" />
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div class="card-grid" ref={gridRef}>
            {items.slice(0, visibleCount).map((item, index) => (
              <BrowseCard
                key={item.episode?.id ?? item.entry.id}
                entry={item.entry}
                active={false}
                animate={animateCards}
                animationDelay={index}
                hideDuration={item.entry.type === "series" && !item.episode}
                {...(item.episode
                  ? {
                      href: item.episode.watchPath,
                      poster: item.episode,
                    }
                  : {})}
                {...optionalSubtitle(item)}
              />
            ))}
          </div>
        )}
      </div>
    </section>
  );
}

function optionalSubtitle(item: RecentlyAddedItem): { subtitle?: string } {
  if (item.episode) {
    const { label, title } = item.episode;
    return {
      subtitle: label && label !== "WATCH" ? `${label} · ${title}` : title,
    };
  }
  if (item.entry.type === "series") {
    return { subtitle: "New series" };
  }
  return {};
}

/**
 * Counts the grid's auto-fill columns so the row shows exactly one line of
 * cards. Measured before paint, so the clipped cards never flash in.
 */
function useOneRowCount(
  gridRef: { current: HTMLDivElement | null },
  active: boolean,
): number {
  const [count, setCount] = useState(STACKED_VISIBLE_COUNT);

  useLayoutEffect(() => {
    const grid = gridRef.current;
    if (!active || !grid) {
      return;
    }

    const measure = () => {
      const columns = getComputedStyle(grid)
        .gridTemplateColumns.split(/\s+/)
        .filter(Boolean).length;
      setCount(columns > 1 ? columns : STACKED_VISIBLE_COUNT);
    };

    measure();
    if (!("ResizeObserver" in window)) {
      return;
    }
    const observer = new ResizeObserver(measure);
    observer.observe(grid);
    return () => observer.disconnect();
  }, [active, gridRef]);

  return count;
}
