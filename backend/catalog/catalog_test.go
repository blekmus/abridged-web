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
