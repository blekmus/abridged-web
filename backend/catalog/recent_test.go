package catalog

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecentlyAddedGroupsNewSeriesAndListsNewEpisodes(t *testing.T) {
	root := t.TempDir()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	day := 24 * time.Hour

	files := map[string]time.Time{
		// Old series that just gained Episode 3: show the episode, not the series.
		"Series/[Creator] Old Show/Episode 1 - One.mp4":   base,
		"Series/[Creator] Old Show/Episode 2 - Two.mp4":   base.Add(day),
		"Series/[Creator] Old Show/Episode 3 - Three.mp4": base.Add(30 * day),
		// Whole series landed together: one card for the series.
		"Series/[Creator] New Show/Episode 1 - A.mp4": base.Add(20 * day),
		"Series/[Creator] New Show/Episode 2 - B.mp4": base.Add(20*day + time.Hour),
		"Shorts/[Creator] A Short/1.mp4":              base.Add(25 * day),
		"Shots/[Creator] A Shot/1.mp4":                base.Add(10 * day),
		// Songs are excluded however new they are.
		"Songs/[Creator] A Song/1.mp4":     base.Add(40 * day),
		"Songs/AMV/[Creator] An AMV/1.mp4": base.Add(40 * day),
	}
	for name, modTime := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("video"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, modTime, modTime); err != nil {
			t.Fatal(err)
		}
	}
	writeMetadataCacheFile(t, root, nil)

	lib, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	type card struct{ entry, episode string }
	want := []card{
		{"Old Show", "Three"},
		{"A Short", ""},
		{"New Show", ""},
		{"A Shot", ""},
		{"Old Show", "Two"},
		{"Old Show", "One"},
	}

	got := lib.CatalogResponse().RecentlyAdded
	if len(got) != len(want) {
		t.Fatalf("len(RecentlyAdded) = %d, want %d: %+v", len(got), len(want), got)
	}
	for index, item := range got {
		episode := ""
		if item.Episode != nil {
			episode = item.Episode.Title
		}
		if (card{item.Entry.EntryTitle, episode}) != want[index] {
			t.Fatalf("RecentlyAdded[%d] = {%s %s}, want %v", index, item.Entry.EntryTitle, episode, want[index])
		}
	}

	newEpisode := got[0].Episode
	if newEpisode.WatchPath != "/creator/old-show/episode-3-three" {
		t.Fatalf("episode WatchPath = %q", newEpisode.WatchPath)
	}
	if newEpisode.Label == "" {
		t.Fatal("episode Label is empty")
	}
	if got[0].AddedAt != base.Add(30*day).Format(time.RFC3339) {
		t.Fatalf("AddedAt = %q", got[0].AddedAt)
	}
}

func TestRecentlyAddedIsCapped(t *testing.T) {
	entries := make([]Entry, 0, recentlyAddedLimit+5)
	for index := range recentlyAddedLimit + 5 {
		entries = append(entries, Entry{
			ID:         stableID("short", string(rune('a'+index))),
			Type:       EntryTypeShort,
			EntryTitle: string(rune('a' + index)),
			Episodes:   []Episode{{AddedAt: time.Unix(int64(index+1), 0)}},
		})
	}
	if got := len(buildRecentlyAdded(entries)); got != recentlyAddedLimit {
		t.Fatalf("len = %d, want %d", got, recentlyAddedLimit)
	}
}
