import type { CreatorSummary } from "../../lib/types";
import { handleInternalLinkClick } from "../navigation";

const CREATOR_SKELETON_KEYS = [
  "creator-1",
  "creator-2",
  "creator-3",
  "creator-4",
  "creator-5",
  "creator-6",
  "creator-7",
  "creator-8",
] as const;

export function CreatorBilling({
  creators,
  compact = false,
}: {
  creators: CreatorSummary[];
  compact?: boolean;
}) {
  return (
    <ul
      class={`creator-billing ${compact ? "is-compact" : ""}`}
      aria-label="Creators"
    >
      {creators.map((creator) => {
        const creatorPath = `/${creator.slug}`;

        return (
          <li key={creator.slug}>
            <a
              class="creator-billing-entry"
              href={creatorPath}
              onClick={(event) => {
                handleInternalLinkClick(event, creatorPath, {
                  scrollToTop: true,
                });
              }}
            >
              <span class="creator-billing-name">{creator.name}</span>
              <span class="creator-billing-count">
                {creator.entryCount}{" "}
                {creator.entryCount === 1 ? "entry" : "entries"}
              </span>
            </a>
          </li>
        );
      })}
    </ul>
  );
}

export function CreatorLoading({ compact = false }: { compact?: boolean }) {
  return (
    <div
      class={`creator-loading-grid ${compact ? "is-compact" : ""}`}
      aria-label="Loading creators"
      aria-busy="true"
    >
      {CREATOR_SKELETON_KEYS.map((key) => (
        <div key={key} class="creator-loading-row">
          <span class="skeleton-line" />
          <span class="skeleton-line short" />
        </div>
      ))}
    </div>
  );
}
