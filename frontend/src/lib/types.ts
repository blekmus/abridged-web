export type EntryType = "series" | "short" | "shot" | "song";
export type EntrySubtype = "amv";

export type Episode = {
  id: string;
  entryId: string;
  label: string;
  videoTitle: string;
  displayTitle: string;
  rawStem: string;
  kind?: string;
  number?: string;
  description?: string;
  date?: string;
  durationSeconds: number;
  videoUrl: string;
  thumbnailUrl?: string;
  hasThumbnail: boolean;
};

export type EpisodeCard = {
  id: string;
  durationSeconds: number;
  thumbnailUrl?: string;
  hasThumbnail: boolean;
};

export type Entry = {
  id: string;
  type: EntryType;
  subtype?: EntrySubtype;
  creator: string;
  creatorSlug: string;
  entryTitle: string;
  description?: string;
  defaultEpisodeId: string;
  posterEpisodeId: string;
  episodes: Episode[];
};

export type EntrySummary = {
  id: string;
  type: EntryType;
  subtype?: EntrySubtype;
  creator: string;
  creatorSlug: string;
  entryTitle: string;
  posterEpisodeId: string;
  poster: EpisodeCard;
};

export type CreatorSummary = {
  slug: string;
  name: string;
  entryCount: number;
};

// One card in Recently Added. `episode` is set when a new episode joined an
// existing series; otherwise the card is the entry itself.
export type RecentlyAddedItem = {
  entry: EntrySummary;
  episode?: RecentEpisode;
  addedAt: string;
};

export type RecentEpisode = EpisodeCard & {
  label: string;
  title: string;
  watchPath: string;
};

export type CatalogResponse = {
  series: EntrySummary[];
  shorts: EntrySummary[];
  shots: EntrySummary[];
  songs: EntrySummary[];
  songAmvs: EntrySummary[];
  creators: CreatorSummary[];
  recentlyAdded: RecentlyAddedItem[];
};
