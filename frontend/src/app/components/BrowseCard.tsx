import type { Entry } from "../../lib/types";
import { findPosterEpisode } from "../entryUtils";
import { handleInternalLinkClick } from "../navigation";
import { Thumbnail } from "./Thumbnail";

export function BrowseCard({
  entry,
  active,
  animate,
  animationDelay,
  hideDuration = false,
}: {
  entry: Entry;
  active: boolean;
  animate: boolean;
  animationDelay: number;
  hideDuration?: boolean;
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
        <Thumbnail
          episode={poster}
          title={entry.entryTitle}
          hideDuration={hideDuration}
        />
      </a>
      <div class="card-copy">
        <a
          href={`/creator/${entry.creatorSlug}`}
          class="card-meta"
          onClick={(event) => {
            handleInternalLinkClick(event, `/creator/${entry.creatorSlug}`);
          }}
        >
          [{entry.creator}]
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
