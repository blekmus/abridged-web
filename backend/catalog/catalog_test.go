package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSupportsWebMVideos(t *testing.T) {
	root := t.TempDir()
	seriesPath := filepath.Join(root, "Series", "[Creator] Show", "Episode 1 - Start.webm")
	shortPath := filepath.Join(root, "Shorts", "[Creator] Short", "1.webm")
	for _, videoPath := range []string{seriesPath, shortPath} {
		if err := os.MkdirAll(filepath.Dir(videoPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeMetadataCacheFile(t, root, nil)

	lib, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	series := lib.CatalogResponse().Series
	if len(series) != 1 {
		t.Fatalf("len(Series) = %d, want 1", len(series))
	}
	seriesEntry, ok := lib.Entry(series[0].ID)
	if !ok {
		t.Fatal("series entry was not registered")
	}
	if got := seriesEntry.Episodes[0].VideoPath; got != seriesPath {
		t.Fatalf("series VideoPath = %q, want %q", got, seriesPath)
	}

	shorts := lib.CatalogResponse().Shorts
	if len(shorts) != 1 {
		t.Fatalf("len(Shorts) = %d, want 1", len(shorts))
	}
	shortEntry, ok := lib.Entry(shorts[0].ID)
	if !ok {
		t.Fatal("short entry was not registered")
	}
	if got := shortEntry.Episodes[0].VideoPath; got != shortPath {
		t.Fatalf("short VideoPath = %q, want %q", got, shortPath)
	}
}

func TestVideoMetadataCacheReadsHostAbsoluteKeys(t *testing.T) {
	root := t.TempDir()
	videoPath := filepath.Join(root, "Series", "[Creator] Show", "Episode 1 - Start.mp4")
	if err := os.MkdirAll(filepath.Dir(videoPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(videoPath)
	if err != nil {
		t.Fatal(err)
	}

	payload := videoMetadataCacheFile{
		Version: videoMetadataCacheVersion,
		Items: map[string]cachedVideoMetadata{
			"/srv/abridged/archive/Series/[Creator] Show/Episode 1 - Start.mp4": {
				Size:            info.Size(),
				ModTimeUnixNano: info.ModTime().UnixNano(),
				Description:     "host generated description",
				Date:            "2026-04-27",
				DurationSeconds: 42,
			},
		},
	}
	writeMetadataCacheFile(t, root, payload.Items)

	cache, err := newVideoMetadataCache(root)
	if err != nil {
		t.Fatal(err)
	}

	metadata := cache.Read(videoPath)
	if metadata.Description != "host generated description" {
		t.Fatalf("Description = %q, want host generated description", metadata.Description)
	}
	if metadata.DurationSeconds != 42 {
		t.Fatalf("DurationSeconds = %d, want 42", metadata.DurationSeconds)
	}
}

func TestVideoMetadataCacheNormalizesWindowsKeys(t *testing.T) {
	root := t.TempDir()
	videoPath := filepath.Join(root, "Shorts", "[Creator] Short", "1.mp4")
	if err := os.MkdirAll(filepath.Dir(videoPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(videoPath)
	if err != nil {
		t.Fatal(err)
	}

	payload := videoMetadataCacheFile{
		Version: videoMetadataCacheVersion,
		Items: map[string]cachedVideoMetadata{
			`C:\Archive\Shorts\[Creator] Short\1.mp4`: {
				Size:            info.Size(),
				ModTimeUnixNano: info.ModTime().UnixNano(),
				Description:     "windows generated description",
			},
		},
	}
	writeMetadataCacheFile(t, root, payload.Items)

	cache, err := newVideoMetadataCache(root)
	if err != nil {
		t.Fatal(err)
	}

	metadata := cache.Read(videoPath)
	if metadata.Description != "windows generated description" {
		t.Fatalf("Description = %q, want windows generated description", metadata.Description)
	}
}

func TestVideoMetadataCacheAcceptsMatchingSizeWithChangedModTime(t *testing.T) {
	root := t.TempDir()
	videoPath := filepath.Join(root, "Series", "[Creator] Show", "Episode 2 - Changed Time.mp4")
	if err := os.MkdirAll(filepath.Dir(videoPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(videoPath)
	if err != nil {
		t.Fatal(err)
	}

	writeMetadataCacheFile(t, root, map[string]cachedVideoMetadata{
		"Series/[Creator] Show/Episode 2 - Changed Time.mp4": {
			Size:            info.Size(),
			ModTimeUnixNano: info.ModTime().UnixNano() - 1,
			Description:     "description survives copied mtimes",
		},
	})

	cache, err := newVideoMetadataCache(root)
	if err != nil {
		t.Fatal(err)
	}

	metadata := cache.Read(videoPath)
	if metadata.Description != "description survives copied mtimes" {
		t.Fatalf("Description = %q, want description survives copied mtimes", metadata.Description)
	}
}

func writeMetadataCacheFile(t *testing.T, root string, items map[string]cachedVideoMetadata) {
	t.Helper()
	if items == nil {
		items = map[string]cachedVideoMetadata{}
	}
	payload := videoMetadataCacheFile{
		Version: videoMetadataCacheVersion,
		Items:   items,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, videoMetadataCacheFilename), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSearchMatchesCreatorTitleTypeKeywordsAndSubtype(t *testing.T) {
	root := t.TempDir()
	videoFiles := []string{
		"Series/[TeamFourStar] Dragon Ball Z Abridged/Episode 1 - Start.mp4",
		"Shorts/[PurpleEyesWTF] I'm at Soup/1.mp4",
		"Songs/[TeamFourStar] Hype/1.mp4",
		"Songs/AMV/[Something Witty] Sword Art Online - This Is War/1.mp4",
	}
	for _, relativePath := range videoFiles {
		fullPath := filepath.Join(root, relativePath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte("video"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeMetadataCacheFile(t, root, nil)

	lib, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name  string
		query string
		want  int
	}{
		{name: "creator", query: "teamfourstar", want: 2},
		{name: "title", query: "abridged", want: 1},
		{name: "series keyword", query: "shows", want: 1},
		{name: "short keyword", query: "shorts", want: 1},
		{name: "song keyword", query: "songs", want: 2},
		{name: "amv subtype", query: "amv", want: 1},
		{name: "multiple terms", query: "teamfourstar hype", want: 1},
		{name: "all terms required", query: "teamfourstar soup", want: 0},
		{name: "empty", query: "   ", want: 0},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			results := lib.Search(testCase.query, 0)
			if len(results) != testCase.want {
				t.Fatalf("len(Search(%q)) = %d, want %d", testCase.query, len(results), testCase.want)
			}
		})
	}
}

func TestSearchRespectsLimitAndPrioritizesTitleMatches(t *testing.T) {
	root := t.TempDir()
	videoFiles := []string{
		"Series/[Creator A] Hype Show/Episode 1 - Start.mp4",
		"Series/[Creator B] Other Show/Episode 1 - Start.mp4",
		"Series/[Hype] Third Show/Episode 1 - Start.mp4",
	}
	for _, relativePath := range videoFiles {
		fullPath := filepath.Join(root, relativePath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte("video"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeMetadataCacheFile(t, root, nil)

	lib, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	results := lib.Search("hype", 0)
	if len(results) != 2 {
		t.Fatalf("len(Search(hype)) = %d, want 2", len(results))
	}
	if results[0].EntryTitle != "Hype Show" {
		t.Fatalf("first result = %q, want Hype Show", results[0].EntryTitle)
	}

	limited := lib.Search("show", 1)
	if len(limited) != 1 {
		t.Fatalf("len(Search(show, 1)) = %d, want 1", len(limited))
	}
}

func TestCacheTagTracksCatalogueContent(t *testing.T) {
	root := t.TempDir()
	writeMetadataCacheFile(t, root, nil)

	writeShort := func(name string) {
		t.Helper()
		videoPath := filepath.Join(root, "Shorts", "[Creator] "+name, "1.mp4")
		if err := os.MkdirAll(filepath.Dir(videoPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	writeShort("First")

	first, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	firstTag := first.CacheTag()
	if firstTag == "" {
		t.Fatal("CacheTag() is empty")
	}

	again, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := again.CacheTag(); got != firstTag {
		t.Fatalf("CacheTag() changed without a catalogue change: %q != %q", got, firstTag)
	}

	writeShort("Second")

	changed, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := changed.CacheTag(); got == firstTag {
		t.Fatal("CacheTag() did not change after the catalogue changed")
	}
}

func TestRouteSlugMatchesFrontendBehaviour(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{input: "Episode 1 - Start", want: "episode-1-start"},
		{input: "I'm at Soup", want: "i-m-at-soup"},
		{input: "A & B", want: "a-and-b"},
		{input: "Pokémon", want: "pokemon"},
		{input: "  --Hello, World!--  ", want: "hello-world"},
		{input: "Sword Art Online - This Is War", want: "sword-art-online-this-is-war"},
		{input: "", want: "untitled"},
	}

	for _, testCase := range cases {
		t.Run(testCase.input, func(t *testing.T) {
			if got := routeSlug(testCase.input); got != testCase.want {
				t.Fatalf("routeSlug(%q) = %q, want %q", testCase.input, got, testCase.want)
			}
		})
	}
}

func TestCatalogCreatorsAreSortedAndCounted(t *testing.T) {
	root := t.TempDir()
	videoFiles := []string{
		"Shorts/[Beta] B Short/1.mp4",
		"Songs/[Alpha] A Song/1.mp4",
		"Series/[Alpha] A Show/Episode 1 - Start.mp4",
	}
	for _, relativePath := range videoFiles {
		fullPath := filepath.Join(root, relativePath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte("video"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeMetadataCacheFile(t, root, nil)

	lib, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	creators := lib.CatalogResponse().Creators
	if len(creators) != 2 {
		t.Fatalf("len(Creators) = %d, want 2", len(creators))
	}
	if creators[0].Name != "Alpha" || creators[0].Slug != "alpha" || creators[0].EntryCount != 2 {
		t.Fatalf("creators[0] = %+v, want Alpha with 2 entries", creators[0])
	}
	if creators[1].Name != "Beta" || creators[1].Slug != "beta" || creators[1].EntryCount != 1 {
		t.Fatalf("creators[1] = %+v, want Beta with 1 entry", creators[1])
	}
}
