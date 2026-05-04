import type { Entry, EntrySummary, EpisodeCard } from "../../lib/types";
import { findPosterEpisode } from "../entryUtils";
import { handleInternalLinkClick } from "../navigation";
import { entryWatchPath } from "../watchPaths";
import { Thumbnail } from "./Thumbnail";

type BrowseCardEntry = EntrySummary | Entry;

export function BrowseCard({
  entry,
  active,
  animate,
  animationDelay,
  hideDuration = false,
}: {
  entry: BrowseCardEntry;
  active: boolean;
  animate: boolean;
  animationDelay: number;
  hideDuration?: boolean;
}) {
  const poster = posterForEntry(entry);
  const watchPath = entryWatchPath(entry);
  const creatorPath = `/${entry.creatorSlug}`;

  return (
    <article
      class={`browse-card ${active ? "is-active" : ""} ${animate ? "" : "skip-enter-animation"}`}
      style={animate ? { animationDelay: `${animationDelay * 60}ms` } : {}}
    >
      <a
        href={watchPath}
        class="card-thumbnail-shell"
        onClick={(event) => {
          handleInternalLinkClick(event, watchPath, {
            scrollToTop: true,
          });
        }}
      >
        <Thumbnail
          episode={poster}
          title={entry.entryTitle}
          hideDuration={hideDuration}
        />
      </a>
      <div class="card-copy">
        <a
          href={creatorPath}
          class="card-meta"
          onClick={(event) => {
            handleInternalLinkClick(event, creatorPath);
          }}
        >
          [{entry.creator}]
        </a>
        <a
          href={watchPath}
          class="card-title-link"
          onClick={(event) => {
            handleInternalLinkClick(event, watchPath, {
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

function posterForEntry(entry: BrowseCardEntry): EpisodeCard {
  if ("poster" in entry) {
    return entry.poster;
  }

  return findPosterEpisode(entry);
}
