import type { CatalogResponse, EntryType } from "../lib/types";

export const EMPTY_CATALOG: CatalogResponse = {
  series: [],
  shorts: [],
  shots: [],
  songs: [],
  songAmvs: [],
};

export const TYPE_META: Record<
  EntryType,
  { title: string; eyebrow: string; path: string }
> = {
  series: { title: "Shows", eyebrow: "Multi-episode runs", path: "/series" },
  short: {
    title: "Shorts",
    eyebrow: "Titles with a mildy coherent plot",
    path: "/shorts",
  },
  shot: {
    title: "One-Shots",
    eyebrow: "Single entry stories",
    path: "/shots",
  },
  song: {
    title: "Songs",
    eyebrow: "original songs & AMVs",
    path: "/songs",
  },
};

export const CREATOR_SECTIONS: Array<{
  entryType: EntryType;
  title: string;
  titleClassName: string;
}> = [
  { entryType: "series", title: "Shows", titleClassName: "accent-shows" },
  { entryType: "short", title: "Shorts", titleClassName: "accent-shorts" },
  {
    entryType: "shot",
    title: "One-Shots",
    titleClassName: "accent-one-shots",
  },
  { entryType: "song", title: "Songs", titleClassName: "accent-songs" },
];

export const LOADING_CARD_KEYS = [
  "skeleton-1",
  "skeleton-2",
  "skeleton-3",
  "skeleton-4",
] as const;
