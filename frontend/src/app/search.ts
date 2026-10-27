import type { EntryType } from "../lib/types";

export const MAX_SEARCH_QUERY_LENGTH = 120;

export const SEARCH_TYPE_META: Record<
  EntryType,
  { label: string; className: string }
> = {
  series: {
    label: "series",
    className: "accent-shows",
  },
  shot: {
    label: "shot",
    className: "accent-one-shots",
  },
  short: {
    label: "short",
    className: "accent-shorts",
  },
  song: { label: "song", className: "accent-songs" },
};

export function searchQueryTerms(query: string): string[] {
  return [...new Set(query.trim().toLowerCase().split(/\s+/).filter(Boolean))];
}
