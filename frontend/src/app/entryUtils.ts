import type {
  CatalogResponse,
  Entry,
  EntrySummary,
  EntryType,
  Episode,
} from "../lib/types";
import { episodeSlug } from "./watchPaths";

export function entriesForType(
  catalog: CatalogResponse,
  entryType: EntryType,
): EntrySummary[] {
  switch (entryType) {
    case "series":
      return sortedEntriesByTitle(catalog.series);
    case "short":
      return sortedEntriesByTitle(catalog.shorts);
    case "shot":
      return sortedEntriesByTitle(catalog.shots);
    case "song":
      return sortedEntriesByTitle(catalog.songs);
  }
}

export function sortedEntriesByTitle<T extends EntrySummary>(
  entries: T[],
): T[] {
  return [...entries].sort((leftEntry, rightEntry) =>
    leftEntry.entryTitle.localeCompare(rightEntry.entryTitle, undefined, {
      sensitivity: "base",
    }),
  );
}

export function shuffledEntries<T>(entries: T[]): T[] {
  const shuffled = [...entries];

  for (let index = shuffled.length - 1; index > 0; index -= 1) {
    const randomIndex = Math.floor(Math.random() * (index + 1));
    const currentEntry = shuffled[index];
    const randomEntry = shuffled[randomIndex];

    if (!currentEntry || !randomEntry) {
      continue;
    }

    shuffled[index] = randomEntry;
    shuffled[randomIndex] = currentEntry;
  }

  return shuffled;
}

export function groupEntriesByType(
  entries: EntrySummary[],
): Record<EntryType, EntrySummary[]> {
  return {
    series: entries.filter((entry) => entry.type === "series"),
    short: entries.filter((entry) => entry.type === "short"),
    shot: entries.filter((entry) => entry.type === "shot"),
    song: entries.filter((entry) => entry.type === "song"),
  };
}

export function watchTypeLabel(entry: Entry): string {
  if (entry.subtype === "amv") {
    return "Song / AMV";
  }

  switch (entry.type) {
    case "series":
      return "Show";
    case "short":
      return "Short";
    case "shot":
      return "One-Shot";
    case "song":
      return "Song";
  }
}

export function accentClassForEntryType(entryType: EntryType): string {
  switch (entryType) {
    case "series":
      return "accent-shows";
    case "short":
      return "accent-shorts";
    case "shot":
      return "accent-one-shots";
    case "song":
      return "accent-songs";
  }
}

export function formatEpisodeDate(rawDate?: string): string | undefined {
  const trimmed = rawDate?.trim();
  if (!trimmed) {
    return undefined;
  }

  const dateMatch =
    /^(\d{4})-(\d{2})-(\d{2})$/.exec(trimmed) ??
    /^(\d{4})(\d{2})(\d{2})$/.exec(trimmed);
  if (!dateMatch) {
    return trimmed;
  }

  const [, year, month, day] = dateMatch;
  const utcDate = new Date(`${year}-${month}-${day}T00:00:00Z`);
  if (Number.isNaN(utcDate.getTime())) {
    return trimmed;
  }

  return new Intl.DateTimeFormat("en-US", {
    month: "long",
    day: "2-digit",
    year: "numeric",
    timeZone: "UTC",
  }).format(utcDate);
}

export function formatDurationLabel(durationSeconds: number): string {
  const totalSeconds = Math.max(0, Math.floor(durationSeconds));
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  if (hours > 0) {
    return `${hours}:${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
  }

  return `${minutes}:${String(seconds).padStart(2, "0")}`;
}

export function chooseEpisode(
  entry: Entry,
  requestedEpisodeSlug?: string,
): Episode {
  if (requestedEpisodeSlug) {
    const matched = entry.episodes.find(
      (episode) => episodeSlug(episode) === requestedEpisodeSlug,
    );
    if (matched) {
      return matched;
    }
  }

  return (
    entry.episodes.find((episode) => episode.id === entry.defaultEpisodeId) ??
    entry.episodes[0] ??
    missingEpisode(entry.id)
  );
}

export function findPosterEpisode(entry: Entry): Episode {
  return (
    entry.episodes.find((episode) => episode.id === entry.posterEpisodeId) ??
    entry.episodes[0] ??
    missingEpisode(entry.id)
  );
}

function missingEpisode(entryId: string): never {
  throw new Error(`Entry ${entryId} does not have any episodes.`);
}
