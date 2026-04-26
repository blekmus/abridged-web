import { useEffect, useState } from "preact/hooks";
import { api } from "../../lib/api";
import type { Entry } from "../../lib/types";
import { DescriptionBlock } from "../components/DescriptionBlock";
import { InlineError } from "../components/InlineError";
import { Thumbnail } from "../components/Thumbnail";
import { TYPE_META } from "../constants";
import {
  accentClassForEntryType,
  chooseEpisode,
  formatEpisodeDate,
  watchTypeLabel,
} from "../entryUtils";
import { handleInternalLinkClick } from "../navigation";
import type { AsyncState } from "../types";

export function WatchPage({
  entryId,
  episodeId,
  restoreFromHistory,
}: {
  entryId: string;
  episodeId?: string;
  restoreFromHistory: boolean;
}) {
  const [state, setState] = useState<AsyncState<Entry>>({
    data: null,
    loading: true,
    error: null,
  });

  useEffect(() => {
    let active = true;
    setState({ data: null, loading: true, error: null });
    api
      .fetchEntry(entryId)
      .then((data) => {
        if (!active) return;
        setState({ data, loading: false, error: null });
      })
      .catch((error: Error) => {
        if (!active) return;
        setState({ data: null, loading: false, error: error.message });
      });

    return () => {
      active = false;
    };
  }, [entryId]);

  if (state.loading) {
    return (
      <section class="watch-page">
        <div class="container watch-container">
          <div class="watch-intro">
            <div class="loading-line short" />
            <div class="loading-line title" />
            <div class="loading-line" />
          </div>
          <div class="player-shell loading-block" />
          <div class="watch-layout">
            <div class="loading-copy" />
            <div class="loading-copy" />
          </div>
        </div>
      </section>
    );
  }

  if (state.error || !state.data) {
    return <InlineError message={state.error ?? "Entry not found."} />;
  }

  const entry = state.data;
  const episode = chooseEpisode(entry, episodeId);
  const description = episode.description || entry.description;
  const episodeDate = formatEpisodeDate(episode.date);

  return (
    <section class={`watch-page watch-page-${entry.type}`}>
      <div class="container watch-container">
        <div class="watch-intro">
          <h1 class="watch-entry-title">{entry.entryTitle}</h1>
        </div>

        <div class="player-shell">
          <video
            class="video-player"
            controls
            preload="metadata"
            src={episode.videoUrl}
            poster={episode.thumbnailUrl}
          />
        </div>

        <div class="watch-layout">
          <div class="watch-main">
            <div class="metadata-block">
              {entry.type === "series" ? (
                <p class="video-title">
                  {episode.displayTitle || episode.videoTitle}
                </p>
              ) : (
                ""
              )}
              <div class="watch-credit">
                <p>
                  <a
                    href={TYPE_META[entry.type].path}
                    class={`watch-type-tag ${accentClassForEntryType(entry.type)}`}
                    onClick={(event) => {
                      handleInternalLinkClick(
                        event,
                        TYPE_META[entry.type].path,
                      );
                    }}
                  >
                    {watchTypeLabel(entry.type)}
                  </a>
                </p>
                <p>
                  Produced by{" "}
                  <a
                    href={`/creator/${entry.creatorSlug}`}
                    class="watch-credit-link"
                    onClick={(event) => {
                      handleInternalLinkClick(
                        event,
                        `/creator/${entry.creatorSlug}`,
                      );
                    }}
                  >
                    {entry.creator}
                  </a>
                </p>
                {episodeDate ? <p>Uploaded on {episodeDate}</p> : null}
              </div>
            </div>

            {description ? (
              <DescriptionBlock key={description} text={description} />
            ) : null}

            {entry.type === "series" ? (
              <section class="episode-section">
                <h2 class="watch-section-title">
                  <span>Episodes</span>
                </h2>
                <div class="watch-metadata-grid card-grid">
                  {entry.episodes.map((item) => (
                    <article
                      key={item.id}
                      class={`browse-card watch-episode-card ${item.id === episode.id ? "is-active" : ""} ${restoreFromHistory ? "skip-enter-animation" : ""}`}
                    >
                      <a
                        href={`/watch/${entry.id}/${item.id}`}
                        class="card-thumbnail-shell"
                        onClick={(event) => {
                          handleInternalLinkClick(
                            event,
                            `/watch/${entry.id}/${item.id}`,
                            { scrollToTop: true },
                          );
                        }}
                      >
                        <Thumbnail episode={item} title={entry.entryTitle} />
                      </a>
                      <div class="card-copy">
                        <a
                          href={`/watch/${entry.id}/${item.id}`}
                          class="card-title-link"
                          onClick={(event) => {
                            handleInternalLinkClick(
                              event,
                              `/watch/${entry.id}/${item.id}`,
                              { scrollToTop: true },
                            );
                          }}
                        >
                          <h3 class="card-title">
                            {item.displayTitle || item.videoTitle}
                          </h3>
                          {item.date ? (
                            <p class="card-submeta">
                              {formatEpisodeDate(item.date)}
                            </p>
                          ) : null}
                        </a>
                      </div>
                    </article>
                  ))}
                </div>
              </section>
            ) : null}
          </div>
        </div>
      </div>
    </section>
  );
}
