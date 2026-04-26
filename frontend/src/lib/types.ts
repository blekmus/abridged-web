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

export type CatalogResponse = {
  series: Entry[];
  shorts: Entry[];
  shots: Entry[];
  songs: Entry[];
  songAmvs: Entry[];
};
