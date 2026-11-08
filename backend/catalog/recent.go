package catalog

import (
	"os"
	"sort"
	"strings"
	"time"
)

const (
	// recentlyAddedLimit covers the widest one-row desktop grid; the frontend
	// trims it to the visible column count (or 6 on mobile).
	recentlyAddedLimit = 12

	// newSeriesWindow is how close together every episode of a series must have
	// landed for the whole series to count as one new addition rather than a
	// run of new episodes on an existing series.
	newSeriesWindow = 48 * time.Hour
)

// RecentlyAddedItem is one card in the home page's Recently Added row. Episode
// is set when a new episode joined an existing series; otherwise the card is
// the entry itself (a short, a one-shot, or a series added all at once).
type RecentlyAddedItem struct {
	Entry   EntrySummary   `json:"entry"`
	Episode *RecentEpisode `json:"episode,omitempty"`
	AddedAt string         `json:"addedAt"`
}

type RecentEpisode struct {
	EpisodeCard
	Label     string `json:"label"`
	Title     string `json:"title"`
	WatchPath string `json:"watchPath"`
}

// "Added" is the video file's modification time. Copies that preserve mtime
// (rsync -a, cp -p) keep the source file's time and will rank as old.
func fileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

type recentCandidate struct {
	item    RecentlyAddedItem
	addedAt time.Time
	order   int
}

func buildRecentlyAdded(entries []Entry) []RecentlyAddedItem {
	var candidates []recentCandidate
	add := func(entry Entry, episode *Episode, addedAt time.Time, order int) {
		if addedAt.IsZero() {
			return
		}
		item := RecentlyAddedItem{
			Entry:   entrySummary(entry),
			AddedAt: addedAt.UTC().Format(time.RFC3339),
		}
		if episode != nil {
			item.Episode = &RecentEpisode{
				EpisodeCard: episodeCard(*episode),
				Label:       episode.Label,
				Title:       episode.VideoTitle,
				WatchPath:   episode.WatchPath(entry),
			}
		}
		candidates = append(candidates, recentCandidate{item: item, addedAt: addedAt, order: order})
	}

	for _, entry := range entries {
		switch entry.Type {
		case EntryTypeShort, EntryTypeShot:
			if len(entry.Episodes) > 0 {
				add(entry, nil, entry.Episodes[0].AddedAt, 0)
			}
		case EntryTypeSeries:
			oldest, newest := addedRange(entry.Episodes)
			if !oldest.IsZero() && newest.Sub(oldest) <= newSeriesWindow {
				add(entry, nil, newest, 0)
				continue
			}
			for index := range entry.Episodes {
				add(entry, &entry.Episodes[index], entry.Episodes[index].AddedAt, index)
			}
		}
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if !a.addedAt.Equal(b.addedAt) {
			return a.addedAt.After(b.addedAt)
		}
		if a.item.Entry.ID != b.item.Entry.ID {
			return strings.ToLower(a.item.Entry.EntryTitle) < strings.ToLower(b.item.Entry.EntryTitle)
		}
		return a.order > b.order
	})

	limit := min(len(candidates), recentlyAddedLimit)
	items := make([]RecentlyAddedItem, 0, limit)
	for _, candidate := range candidates[:limit] {
		items = append(items, candidate.item)
	}
	return items
}

func addedRange(episodes []Episode) (oldest time.Time, newest time.Time) {
	for _, episode := range episodes {
		if episode.AddedAt.IsZero() {
			return time.Time{}, time.Time{}
		}
		if oldest.IsZero() || episode.AddedAt.Before(oldest) {
			oldest = episode.AddedAt
		}
		if episode.AddedAt.After(newest) {
			newest = episode.AddedAt
		}
	}
	return oldest, newest
}
