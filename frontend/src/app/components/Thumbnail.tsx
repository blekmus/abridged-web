import type { EpisodeCard } from "../../lib/types";
import { formatDurationLabel } from "../entryUtils";

export function Thumbnail({
  episode,
  title,
  hideDuration = false,
}: {
  episode: EpisodeCard;
  title: string;
  hideDuration?: boolean;
}) {
  const markThumbnailLoaded = (image: HTMLImageElement) => {
    image.parentElement?.classList.add("has-loaded-thumbnail");
  };

  return (
    <>
      {episode.thumbnailUrl ? (
        <>
          <div class="thumb-loading" aria-hidden="true" />
          <img
            class="thumb"
            src={episode.thumbnailUrl}
            // alt={`${title} thumbnail`}
            loading="lazy"
            decoding="async"
            onLoad={(event) => {
              markThumbnailLoaded(event.currentTarget);
            }}
            onError={(event) => {
              markThumbnailLoaded(event.currentTarget);
            }}
          />
        </>
      ) : (
        <div class="thumb thumb-fallback">
          <span>{title}</span>
        </div>
      )}
      {!hideDuration && episode.durationSeconds > 0 ? (
        <span class="thumb-duration">
          {formatDurationLabel(episode.durationSeconds)}
        </span>
      ) : null}
    </>
  );
}
