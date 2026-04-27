import { useState } from "preact/hooks";
import type { EpisodeCard } from "../../lib/types";
import { formatDurationLabel } from "../entryUtils";

export function Thumbnail({
  episode,
  title,
  hideDuration = false,
}: { episode: EpisodeCard; title: string; hideDuration?: boolean }) {
  const [loadedImageUrl, setLoadedImageUrl] = useState<string | null>(null);
  const imageLoaded =
    Boolean(episode.thumbnailUrl) && loadedImageUrl === episode.thumbnailUrl;

  return (
    <>
      {episode.thumbnailUrl ? (
        <>
          {!imageLoaded ? (
            <div class="thumb-loading skeleton-block" aria-hidden="true" />
          ) : null}
          <img
            class={`thumb ${imageLoaded ? "is-loaded" : ""}`}
            src={episode.thumbnailUrl}
            alt={`${title} thumbnail`}
            loading="lazy"
            decoding="async"
            onLoad={() => setLoadedImageUrl(episode.thumbnailUrl ?? null)}
            onError={() => setLoadedImageUrl(episode.thumbnailUrl ?? null)}
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
