import assert from "node:assert/strict";
import { test } from "node:test";
import {
  entryWatchPath,
  episodeWatchPath,
  findEntryForWatchPath,
} from "../app/watchPaths";
import { api } from "./api";
import { parseRoute } from "./router";
import type { CatalogResponse, Entry } from "./types";

const entry: Entry = {
  id: "kanji-entry",
  type: "series",
  creator: "山田太郎",
  creatorSlug: "山田太郎",
  entryTitle: "Example Show",
  defaultEpisodeId: "episode-1",
  posterEpisodeId: "episode-1",
  episodes: [
    {
      id: "episode-1",
      entryId: "kanji-entry",
      label: "1",
      videoTitle: "First Episode",
      displayTitle: "First Episode",
      rawStem: "first-episode",
      durationSeconds: 60,
      videoUrl: "/video/episode-1",
      hasThumbnail: false,
    },
  ],
};
const summary = {
  ...entry,
  poster: { id: "episode-1", durationSeconds: 60, hasThumbnail: false },
};
const catalog: CatalogResponse = {
  series: [summary],
  shorts: [],
  shots: [],
  songs: [],
  songAmvs: [],
  creators: [],
};

for (const creatorSlug of ["山田太郎", "studio-山田", "example-creator"]) {
  test(`creator route decodes the browser URL for ${creatorSlug}`, () => {
    const pathname = new URL(`/${creatorSlug}`, "https://example.com").pathname;
    assert.deepEqual(parseRoute(pathname), {
      name: "creator",
      slug: creatorSlug,
    });
    assert.deepEqual(parseRoute(`/${creatorSlug}`), {
      name: "creator",
      slug: creatorSlug,
    });
  });
}

test("Japanese creator watch links resolve to the catalog entry", () => {
  const episode = entry.episodes[0];
  assert.ok(episode);
  for (const path of [
    entryWatchPath(entry),
    episodeWatchPath(entry, episode),
  ]) {
    const pathname = new URL(path, "https://example.com").pathname;
    const route = parseRoute(pathname);
    assert.equal(route.name, "watch-path");
    if (route.name !== "watch-path") throw new Error("Expected a watch route");
    assert.equal(route.creatorSlug, entry.creatorSlug);
    assert.equal(
      findEntryForWatchPath(catalog, route.creatorSlug, route.entrySlug)?.id,
      entry.id,
    );
    if (path === episodeWatchPath(entry, episode)) {
      assert.equal(route.episodeSlug, "first-episode");
    }
  }
});

test("creator slug decoding happens once and tolerates malformed escapes", () => {
  assert.deepEqual(parseRoute("/literal%2520name"), {
    name: "creator",
    slug: "literal%20name",
  });
  assert.deepEqual(parseRoute("/bad%escape"), {
    name: "creator",
    slug: "bad%escape",
  });
});

test("creator API encodes the decoded slug as a single URL segment", async () => {
  const originalFetch = globalThis.fetch;
  let requestedURL: unknown;
  globalThis.fetch = (async (url) => {
    requestedURL = url;
    return new Response("[]", {
      headers: { "Content-Type": "application/json" },
    });
  }) as typeof fetch;
  try {
    await api.fetchCreator("山田%太郎");
    assert.equal(
      requestedURL,
      `/api/creator/${encodeURIComponent("山田%太郎")}`,
    );
  } finally {
    globalThis.fetch = originalFetch;
  }
});
