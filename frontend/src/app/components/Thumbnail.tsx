import type { Episode } from "../../lib/types";
import { formatDurationLabel } from "../entryUtils";

export function Thumbnail({
  episode,
  title,
  hideDuration = false,
}: { episode: Episode; title: string; hideDuration?: boolean }) {
  return (
    <>
      {episode.thumbnailUrl ? (
        <img
          class="thumb"
          src={episode.thumbnailUrl}
          alt={`${title} thumbnail`}
          loading="lazy"
          decoding="async"
        />
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
