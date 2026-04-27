import type { CatalogResponse, Entry, EntrySummary, EntryType } from "./types";

function typePath(entryType: EntryType): string {
  switch (entryType) {
    case "series":
      return "series";
    case "short":
      return "shorts";
    case "shot":
      return "shots";
    case "song":
      return "songs";
  }
}

async function readJSON<T>(url: string): Promise<T> {
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(
      `Request failed: ${response.status} ${response.statusText}`,
    );
  }
  return response.json() as Promise<T>;
}

export const api = {
  fetchCatalog(): Promise<CatalogResponse> {
    return readJSON<CatalogResponse>("/api/catalog");
  },

  fetchBrowse(entryType: EntryType): Promise<EntrySummary[]> {
    return readJSON<EntrySummary[]>(`/api/${typePath(entryType)}`);
  },

  fetchEntry(entryId: string): Promise<Entry> {
    return readJSON<Entry>(`/api/entry/${entryId}`);
  },

  fetchCreator(creatorSlug: string): Promise<EntrySummary[]> {
    return readJSON<EntrySummary[]>(`/api/creator/${creatorSlug}`);
  },
};
