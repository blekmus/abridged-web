import { useEffect, useState } from "preact/hooks";
import { api } from "../../lib/api";
import type { Entry, EntryType } from "../../lib/types";
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

const PLAYER_VOLUME_STORAGE_KEY = "abridged-player-volume";
const PLAYER_MUTED_STORAGE_KEY = "abridged-player-muted";
const syncedPlayers = new WeakSet<HTMLVideoElement>();

export function WatchPage({
  entryId,
  episodeId,
  entryTypeHint,
  restoreFromHistory,
}: {
  entryId: string;
  episodeId?: string;
  entryTypeHint: EntryType | undefined;
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
    return <WatchPageSkeleton entryType={entryTypeHint} />;
  }

  if (state.error || !state.data) {
    return <InlineError message={state.error ?? "Entry not found."} />;
  }

  const entry = state.data;
  const episode = chooseEpisode(entry, episodeId);
  const description = episode.description || entry.description;
  const episodeDate = formatEpisodeDate(episode.date);
  const browsePath =
    entry.subtype === "amv" ? "/songs#amvs" : TYPE_META[entry.type].path;

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
            ref={syncPlayerVolume}
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
                <p class="watch-credit-row">
                  <span class="watch-credit-label">Type</span>
                  <span class="watch-credit-value">
                    <a
                      href={browsePath}
                      class={`watch-type-tag ${accentClassForEntryType(entry.type)}`}
                      onClick={(event) => {
                        handleInternalLinkClick(event, browsePath, {
                          scrollToTop: true,
                        });
                      }}
                    >
                      {watchTypeLabel(entry)}
                    </a>
                  </span>
                </p>
                <p class="watch-credit-row">
                  <span class="watch-credit-label">Produced by</span>
                  <span class="watch-credit-value">
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
                  </span>
                </p>
                {episodeDate ? (
                  <p class="watch-credit-row">
                    <span class="watch-credit-label">Uploaded on</span>
                    <span class="watch-credit-value">{episodeDate}</span>
                  </p>
                ) : null}
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

function syncPlayerVolume(player: HTMLVideoElement | null): void {
  if (!player) {
    return;
  }
  if (syncedPlayers.has(player)) {
    return;
  }
  syncedPlayers.add(player);

  const storedVolume = readStoredVolume();
  const storedMuted = readStoredMuted();
  if (storedVolume !== undefined) {
    player.volume = storedVolume;
  }
  if (storedMuted !== undefined) {
    player.muted = storedMuted;
  }

  const savePlayerVolume = () => {
    localStorage.setItem(PLAYER_VOLUME_STORAGE_KEY, String(player.volume));
    localStorage.setItem(PLAYER_MUTED_STORAGE_KEY, String(player.muted));
  };

  player.addEventListener("volumechange", savePlayerVolume);
}

function readStoredVolume(): number | undefined {
  const storedValue = localStorage.getItem(PLAYER_VOLUME_STORAGE_KEY);
  if (storedValue === null) {
    return undefined;
  }

  const volume = Number(storedValue);
  if (!Number.isFinite(volume)) {
    return undefined;
  }

  return Math.min(1, Math.max(0, volume));
}

function readStoredMuted(): boolean | undefined {
  const storedValue = localStorage.getItem(PLAYER_MUTED_STORAGE_KEY);
  if (storedValue === null) {
    return undefined;
  }

  return storedValue === "true";
}

function WatchPageSkeleton({
  entryType,
}: { entryType: EntryType | undefined }) {
  const isSeries = entryType === "series";

  return (
    <section
      class={`watch-page watch-page-skeleton ${entryType ? `watch-page-${entryType}` : ""}`}
      aria-busy="true"
    >
      <div class="container watch-container">
        <div class="watch-intro">
          <div class="loading-line watch-skeleton-entry-title" />
        </div>

        <div class="player-shell loading-block" />

        <div class="watch-layout">
          <div class="watch-main">
            <div class="metadata-block watch-skeleton-metadata">
              {isSeries ? (
                <div class="loading-line watch-skeleton-video-title" />
              ) : null}
              <div class="watch-credit watch-skeleton-credit">
                <div class="loading-line watch-skeleton-tag" />
                <div class="loading-line watch-skeleton-credit-line" />
                <div class="loading-line watch-skeleton-date-line" />
              </div>
            </div>

            <div class="description-block watch-skeleton-description">
              <div class="loading-line" />
              <div class="loading-line" />
              <div class="loading-line short" />
            </div>

            {isSeries ? (
              <section class="episode-section">
                <div class="loading-line watch-skeleton-section-title" />
                <div class="watch-metadata-grid card-grid">
                  {["episode-1", "episode-2", "episode-3"].map((key) => (
                    <article
                      key={key}
                      class="browse-card skeleton-card watch-episode-card"
                    >
                      <div class="card-thumbnail-shell skeleton-block" />
                      <div class="card-copy">
                        <div class="skeleton-line" />
                        <div class="skeleton-line short" />
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
