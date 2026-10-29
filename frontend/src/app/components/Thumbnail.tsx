import type { EpisodeCard } from "../../lib/types";
import { formatDurationLabel } from "../entryUtils";

export function Thumbnail({
  episode,
  title,
  hideDuration = false,
  nowPlaying = false,
}: {
  episode: EpisodeCard;
  title: string;
  hideDuration?: boolean;
  nowPlaying?: boolean;
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
            aria-label={`${title} thumbnail`}
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
      {nowPlaying ? (
        <span class="thumb-duration thumb-duration-now-playing">
          <span class="episode-card-eq" aria-hidden="true">
            <span />
            <span />
            <span />
          </span>
          Now playing
        </span>
      ) : !hideDuration && episode.durationSeconds > 0 ? (
        <span class="thumb-duration">
          {formatDurationLabel(episode.durationSeconds)}
        </span>
      ) : null}
    </>
  );
}
