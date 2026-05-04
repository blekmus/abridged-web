import type {
  CatalogResponse,
  Entry,
  EntrySummary,
  Episode,
} from "../lib/types";

type WatchPathEntry = Entry | EntrySummary;

export function entryWatchPath(entry: WatchPathEntry): string {
  return `/${entry.creatorSlug}/${entrySlug(entry)}`;
}

export function episodeWatchPath(entry: Entry, episode: Episode): string {
  return `${entryWatchPath(entry)}/${episodeSlug(episode)}`;
}

export function findEntryForWatchPath(
  catalog: CatalogResponse,
  creatorSlug: string,
  rawEntrySlug: string,
): EntrySummary | undefined {
  const entrySlug = decodePathSegment(rawEntrySlug);

  return allCatalogEntries(catalog).find(
    (entry) =>
      entry.creatorSlug === creatorSlug &&
      entrySlugForComparison(entry) === entrySlug,
  );
}

export function entrySlug(entry: WatchPathEntry): string {
  return slugifyPathSegment(entry.entryTitle);
}

export function episodeSlug(episode: Episode): string {
  return slugifyPathSegment(episode.displayTitle || episode.videoTitle);
}

export function slugifyPathSegment(value: string): string {
  const slug = value
    .normalize("NFKD")
    .replace(/\p{Diacritic}/gu, "")
    .toLowerCase()
    .replace(/&/g, " and ")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");

  return slug || "untitled";
}

function entrySlugForComparison(entry: WatchPathEntry): string {
  return slugifyPathSegment(decodePathSegment(entry.entryTitle));
}

function decodePathSegment(segment: string): string {
  try {
    return decodeURIComponent(segment);
  } catch {
    return segment;
  }
}

function allCatalogEntries(catalog: CatalogResponse): EntrySummary[] {
  return [
    ...catalog.series,
    ...catalog.shorts,
    ...catalog.shots,
    ...catalog.songs,
    ...catalog.songAmvs,
  ];
}
