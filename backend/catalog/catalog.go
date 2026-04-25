package catalog

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type EntryType string

const (
	EntryTypeSeries EntryType = "series"
	EntryTypeShort  EntryType = "short"
	EntryTypeShot   EntryType = "shot"
)

type Entry struct {
	ID               string    `json:"id"`
	Type             EntryType `json:"type"`
	Creator          string    `json:"creator"`
	CreatorSlug      string    `json:"creatorSlug"`
	EntryTitle       string    `json:"entryTitle"`
	Description      string    `json:"description,omitempty"`
	DefaultEpisodeID string    `json:"defaultEpisodeId"`
	PosterEpisodeID  string    `json:"posterEpisodeId"`
	Episodes         []Episode `json:"episodes"`
}

type Episode struct {
	ID              string `json:"id"`
	EntryID         string `json:"entryId"`
	Label           string `json:"label"`
	VideoTitle      string `json:"videoTitle"`
	DisplayTitle    string `json:"displayTitle"`
	RawStem         string `json:"rawStem"`
	Kind            string `json:"kind,omitempty"`
	Number          string `json:"number,omitempty"`
	Description     string `json:"description,omitempty"`
	Date            string `json:"date,omitempty"`
	DurationSeconds int    `json:"durationSeconds"`
	VideoURL        string `json:"videoUrl"`
	ThumbnailURL    string `json:"thumbnailUrl,omitempty"`
	HasThumbnail    bool   `json:"hasThumbnail"`
	VideoPath       string `json:"-"`
	ThumbnailPath   string `json:"-"`
}

type EpisodeAsset struct {
	VideoPath     string
	ThumbnailPath string
}

type videoMetadata struct {
	Description     string
	Date            string
	DurationSeconds int
}

type ffprobeFormatPayload struct {
	Format struct {
		Duration string            `json:"duration"`
		Tags     map[string]string `json:"tags"`
	} `json:"format"`
}

type cachedVideoMetadata struct {
	Size            int64  `json:"size"`
	ModTimeUnixNano int64  `json:"modTimeUnixNano"`
	Description     string `json:"description,omitempty"`
	Date            string `json:"date,omitempty"`
	DurationSeconds int    `json:"durationSeconds"`
}

type videoMetadataCacheFile struct {
	Version int                            `json:"version"`
	Items   map[string]cachedVideoMetadata `json:"items"`
}

type videoMetadataCache struct {
	root  string
	path  string
	items map[string]cachedVideoMetadata
	seen  map[string]struct{}
	dirty bool
}

type CatalogResponse struct {
	Series []Entry `json:"series"`
	Shorts []Entry `json:"shorts"`
	Shots  []Entry `json:"shots"`
}

type Library struct {
	root          string
	metadata      *videoMetadataCache
	all           []Entry
	byType        map[EntryType][]Entry
	byEntryID     map[string]Entry
	byEpisodeID   map[string]EpisodeAsset
	byCreatorSlug map[string][]Entry
}

var (
	entryNamePattern  = regexp.MustCompile(`^\[(?P<creator>[^\]]+)\]\s*(?P<title>.+)$`)
	seriesStemPattern = regexp.MustCompile(`^(Episode|OVA|Movie)\s+(\d+(?:\.\d+)?(?:~\d+(?:\.\d+)?)?)(?:\s*-\s*(.+))?$`)
)

func Load(root string) (*Library, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve archive root: %w", err)
	}

	lib := &Library{
		root:          absRoot,
		metadata:      newVideoMetadataCache(absRoot),
		byType:        map[EntryType][]Entry{},
		byEntryID:     map[string]Entry{},
		byEpisodeID:   map[string]EpisodeAsset{},
		byCreatorSlug: map[string][]Entry{},
	}

	categoryDirs := []struct {
		dir  string
		kind EntryType
	}{
		{dir: "Series", kind: EntryTypeSeries},
		{dir: "Shorts", kind: EntryTypeShort},
		{dir: "Shots", kind: EntryTypeShot},
	}

	for _, category := range categoryDirs {
		entries, scanErr := lib.scanCategory(filepath.Join(absRoot, category.dir), category.kind)
		if scanErr != nil {
			return nil, scanErr
		}
		lib.byType[category.kind] = entries
		lib.all = append(lib.all, entries...)
		for _, entry := range entries {
			lib.byEntryID[entry.ID] = entry
			lib.byCreatorSlug[entry.CreatorSlug] = append(lib.byCreatorSlug[entry.CreatorSlug], entry)
			for _, episode := range entry.Episodes {
				lib.byEpisodeID[episode.ID] = EpisodeAsset{
					VideoPath:     episode.VideoPath,
					ThumbnailPath: episode.ThumbnailPath,
				}
			}
		}
	}

	if lib.metadata != nil {
		lib.metadata.PruneMissing()
		if err := lib.metadata.Save(); err != nil {
			return nil, fmt.Errorf("save video metadata cache: %w", err)
		}
	}

	return lib, nil
}

func (l *Library) CatalogResponse() CatalogResponse {
	return CatalogResponse{
		Series: cloneEntries(l.byType[EntryTypeSeries]),
		Shorts: cloneEntries(l.byType[EntryTypeShort]),
		Shots:  cloneEntries(l.byType[EntryTypeShot]),
	}
}

func (l *Library) EntriesForType(entryType EntryType) []Entry {
	return cloneEntries(l.byType[entryType])
}

func (l *Library) EntriesForCreator(slug string) []Entry {
	return cloneEntries(l.byCreatorSlug[slug])
}

func (l *Library) Entry(entryID string) (Entry, bool) {
	entry, ok := l.byEntryID[entryID]
	return entry, ok
}

func (l *Library) EpisodeAsset(episodeID string) (EpisodeAsset, bool) {
	asset, ok := l.byEpisodeID[episodeID]
	return asset, ok
}

func (l *Library) scanCategory(categoryPath string, entryType EntryType) ([]Entry, error) {
	items, err := os.ReadDir(categoryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read category %s: %w", categoryPath, err)
	}

	var entries []Entry
	for _, item := range items {
		if !item.IsDir() {
			continue
		}

		entryPath := filepath.Join(categoryPath, item.Name())
		entry, ok, scanErr := l.scanEntry(entryPath, item.Name(), entryType)
		if scanErr != nil {
			return nil, scanErr
		}
		if !ok {
			continue
		}

		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Creator != entries[j].Creator {
			return strings.ToLower(entries[i].Creator) < strings.ToLower(entries[j].Creator)
		}
		return strings.ToLower(entries[i].EntryTitle) < strings.ToLower(entries[j].EntryTitle)
	})

	return entries, nil
}

func (l *Library) scanEntry(entryPath, folderName string, entryType EntryType) (Entry, bool, error) {
	creator, title := parseEntryFolder(folderName)
	entryID := stableID(string(entryType), folderName)

	entry := Entry{
		ID:          entryID,
		Type:        entryType,
		Creator:     creator,
		CreatorSlug: slugify(creator),
		EntryTitle:  title,
		Description: readEntryDescription(entryPath),
	}

	var episodes []episodeRecord
	switch entryType {
	case EntryTypeShort, EntryTypeShot:
		record, ok := l.scanSingleRelease(entryPath, entryID, entryType, title)
		if !ok {
			return Entry{}, false, nil
		}
		episodes = append(episodes, record)
	case EntryTypeSeries:
		seriesEpisodes, err := l.scanSeries(entryPath, entryID)
		if err != nil {
			return Entry{}, false, err
		}
		if len(seriesEpisodes) == 0 {
			return Entry{}, false, nil
		}
		episodes = append(episodes, seriesEpisodes...)
	default:
		return Entry{}, false, nil
	}

	sort.Slice(episodes, func(i, j int) bool {
		return compareEpisodeRecords(episodes[i], episodes[j])
	})

	entry.Episodes = make([]Episode, 0, len(episodes))
	for _, record := range episodes {
		entry.Episodes = append(entry.Episodes, record.Episode)
	}

	entry.DefaultEpisodeID = entry.Episodes[0].ID
	entry.PosterEpisodeID = entry.Episodes[0].ID
	for _, episode := range entry.Episodes {
		if episode.HasThumbnail {
			entry.PosterEpisodeID = episode.ID
			break
		}
	}

	return entry, true, nil
}

type episodeRecord struct {
	Episode
	videoPath   string
	thumbPath   string
	kindOrder   int
	numberValue float64
	hasNumber   bool
}

func (l *Library) scanSingleRelease(entryPath, entryID string, entryType EntryType, title string) (episodeRecord, bool) {
	videoPath := filepath.Join(entryPath, "1.mp4")
	if _, err := os.Stat(videoPath); err != nil {
		return episodeRecord{}, false
	}
	metadata := l.readVideoMetadata(videoPath)

	thumbPath := filepath.Join(entryPath, "1.webp")
	thumbnailURL := ""
	if _, err := os.Stat(thumbPath); err == nil {
		thumbnailURL = "/thumb/" + stableID(entryID, "1")
	} else {
		thumbPath = ""
	}

	episodeID := stableID(entryID, "1")
	displayLabel := "WATCH"
	if entryType == EntryTypeShot {
		displayLabel = "SHOT"
	}
	if entryType == EntryTypeShort {
		displayLabel = "SHORT"
	}

	return episodeRecord{
		Episode: Episode{
			ID:              episodeID,
			EntryID:         entryID,
			Label:           displayLabel,
			VideoTitle:      title,
			DisplayTitle:    title,
			RawStem:         "1",
			Description:     metadata.Description,
			Date:            metadata.Date,
			DurationSeconds: metadata.DurationSeconds,
			VideoURL:        "/video/" + episodeID,
			ThumbnailURL:    thumbnailURL,
			HasThumbnail:    thumbnailURL != "",
			VideoPath:       videoPath,
			ThumbnailPath:   thumbPath,
		},
		videoPath: videoPath,
		thumbPath: thumbPath,
	}, true
}

func (l *Library) scanSeries(entryPath, entryID string) ([]episodeRecord, error) {
	items, err := os.ReadDir(entryPath)
	if err != nil {
		return nil, fmt.Errorf("read series entry %s: %w", entryPath, err)
	}

	var episodes []episodeRecord
	for _, item := range items {
		if item.IsDir() || filepath.Ext(item.Name()) != ".mp4" {
			continue
		}

		stem := strings.TrimSuffix(item.Name(), filepath.Ext(item.Name()))
		videoPath := filepath.Join(entryPath, item.Name())
		thumbPath := filepath.Join(entryPath, stem+".webp")
		parsed := parseSeriesStem(stem)
		metadata := l.readVideoMetadata(videoPath)

		episodeID := stableID(entryID, stem)
		thumbnailURL := ""
		if _, err := os.Stat(thumbPath); err == nil {
			thumbnailURL = "/thumb/" + episodeID
		} else {
			thumbPath = ""
		}

		episodes = append(episodes, episodeRecord{
			Episode: Episode{
				ID:              episodeID,
				EntryID:         entryID,
				Label:           parsed.Label(),
				VideoTitle:      parsed.VideoTitle(stem),
				DisplayTitle:    stem,
				RawStem:         stem,
				Kind:            parsed.Kind,
				Number:          parsed.Number,
				Description:     metadata.Description,
				Date:            metadata.Date,
				DurationSeconds: metadata.DurationSeconds,
				VideoURL:        "/video/" + episodeID,
				ThumbnailURL:    thumbnailURL,
				HasThumbnail:    thumbnailURL != "",
				VideoPath:       videoPath,
				ThumbnailPath:   thumbPath,
			},
			videoPath:   videoPath,
			thumbPath:   thumbPath,
			kindOrder:   parsed.KindOrder(),
			numberValue: parsed.FirstNumber(),
			hasNumber:   parsed.Number != "",
		})
	}

	return episodes, nil
}

func compareEpisodeRecords(a, b episodeRecord) bool {
	if a.kindOrder != b.kindOrder {
		return a.kindOrder < b.kindOrder
	}
	if a.hasNumber != b.hasNumber {
		return a.hasNumber
	}
	if a.hasNumber && b.hasNumber && a.numberValue != b.numberValue {
		return a.numberValue < b.numberValue
	}
	return strings.ToLower(a.RawStem) < strings.ToLower(b.RawStem)
}

type parsedSeriesStem struct {
	Kind   string
	Number string
	Title  string
}

func parseSeriesStem(stem string) parsedSeriesStem {
	matches := seriesStemPattern.FindStringSubmatch(stem)
	if len(matches) == 0 {
		return parsedSeriesStem{}
	}

	return parsedSeriesStem{
		Kind:   matches[1],
		Number: matches[2],
		Title:  strings.TrimSpace(matches[3]),
	}
}

func (p parsedSeriesStem) Label() string {
	switch p.Kind {
	case "Episode":
		if p.Number == "" {
			return "EP"
		}
		return "EP " + p.Number
	case "OVA", "Movie":
		if p.Number == "" {
			return strings.ToUpper(p.Kind)
		}
		return strings.ToUpper(p.Kind) + " " + p.Number
	default:
		return "WATCH"
	}
}

func (p parsedSeriesStem) VideoTitle(fallback string) string {
	if strings.TrimSpace(p.Title) != "" {
		return p.Title
	}
	return fallback
}

func (p parsedSeriesStem) KindOrder() int {
	switch p.Kind {
	case "Episode":
		return 0
	case "OVA":
		return 1
	case "Movie":
		return 2
	default:
		return 3
	}
}

func (p parsedSeriesStem) FirstNumber() float64 {
	if p.Number == "" {
		return 0
	}

	first := strings.SplitN(p.Number, "~", 2)[0]
	value, err := strconv.ParseFloat(first, 64)
	if err != nil {
		return 0
	}
	return value
}

func parseEntryFolder(name string) (string, string) {
	matches := entryNamePattern.FindStringSubmatch(name)
	if len(matches) == 0 {
		return "Unknown", strings.TrimSpace(name)
	}
	return strings.TrimSpace(matches[1]), strings.TrimSpace(matches[2])
}

func readEntryDescription(entryPath string) string {
	candidates := []string{
		"description.txt",
		"description.md",
		"description.nfo",
		"info.txt",
		"info.md",
		"readme.txt",
		"readme.md",
	}

	for _, candidate := range candidates {
		fullPath := filepath.Join(entryPath, candidate)
		if data, err := os.ReadFile(fullPath); err == nil {
			return strings.TrimSpace(string(data))
		}
	}

	items, err := os.ReadDir(entryPath)
	if err != nil {
		return ""
	}

	for _, item := range items {
		if item.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(item.Name()))
		if ext != ".txt" && ext != ".md" && ext != ".nfo" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(entryPath, item.Name()))
		if err == nil {
			return strings.TrimSpace(string(data))
		}
	}

	return ""
}

const (
	videoMetadataCacheVersion  = 2
	videoMetadataCacheFilename = ".abridged-video-metadata.json"
)

func newVideoMetadataCache(root string) *videoMetadataCache {
	cache := &videoMetadataCache{
		root:  root,
		path:  filepath.Join(root, videoMetadataCacheFilename),
		items: map[string]cachedVideoMetadata{},
		seen:  map[string]struct{}{},
	}
	cache.load()
	return cache
}

func (c *videoMetadataCache) load() {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return
	}

	var payload videoMetadataCacheFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return
	}
	if payload.Version != videoMetadataCacheVersion || payload.Items == nil {
		return
	}

	c.items = payload.Items
}

func (c *videoMetadataCache) Read(videoPath string) videoMetadata {
	info, err := os.Stat(videoPath)
	if err != nil {
		return videoMetadata{}
	}

	key := c.key(videoPath)
	c.seen[key] = struct{}{}

	if cached, ok := c.items[key]; ok && cached.matches(info) {
		return cached.metadata()
	}

	metadata, ok := probeVideoMetadata(videoPath)
	if !ok {
		return videoMetadata{}
	}

	c.items[key] = cachedVideoMetadata{
		Size:            info.Size(),
		ModTimeUnixNano: info.ModTime().UnixNano(),
		Description:     metadata.Description,
		Date:            metadata.Date,
		DurationSeconds: metadata.DurationSeconds,
	}
	c.dirty = true

	return metadata
}

func (c *videoMetadataCache) PruneMissing() {
	for key := range c.items {
		if _, ok := c.seen[key]; ok {
			continue
		}
		delete(c.items, key)
		c.dirty = true
	}
}

func (c *videoMetadataCache) Save() error {
	if !c.dirty {
		return nil
	}

	data, err := json.MarshalIndent(videoMetadataCacheFile{
		Version: videoMetadataCacheVersion,
		Items:   c.items,
	}, "", "  ")
	if err != nil {
		return err
	}

	tempPath := c.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tempPath, c.path); err != nil {
		_ = os.Remove(tempPath)
		return err
	}

	c.dirty = false
	return nil
}

func (c *videoMetadataCache) key(videoPath string) string {
	relative, err := filepath.Rel(c.root, videoPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return filepath.ToSlash(videoPath)
	}
	return filepath.ToSlash(relative)
}

func (m cachedVideoMetadata) matches(info os.FileInfo) bool {
	return m.Size == info.Size() && m.ModTimeUnixNano == info.ModTime().UnixNano()
}

func (m cachedVideoMetadata) metadata() videoMetadata {
	return videoMetadata{
		Description:     m.Description,
		Date:            m.Date,
		DurationSeconds: m.DurationSeconds,
	}
}

func (l *Library) readVideoMetadata(videoPath string) videoMetadata {
	if l.metadata == nil {
		return videoMetadata{}
	}
	return l.metadata.Read(videoPath)
}

func probeVideoMetadata(videoPath string) (videoMetadata, bool) {
	output, err := exec.Command(
		"ffprobe",
		"-v",
		"error",
		"-print_format",
		"json",
		"-show_format",
		videoPath,
	).Output()
	if err != nil {
		return videoMetadata{}, false
	}

	var payload ffprobeFormatPayload
	if err := json.Unmarshal(output, &payload); err != nil {
		return videoMetadata{}, false
	}

	durationSeconds := 0
	if duration, err := strconv.ParseFloat(payload.Format.Duration, 64); err == nil {
		durationSeconds = int(duration)
	}

	return videoMetadata{
		Description:     firstMetadataTag(payload.Format.Tags, "description", "synopsis"),
		Date:            firstMetadataTag(payload.Format.Tags, "date"),
		DurationSeconds: durationSeconds,
	}, true
}

func firstMetadataTag(tags map[string]string, names ...string) string {
	if len(tags) == 0 {
		return ""
	}

	for _, name := range names {
		value := strings.TrimSpace(tags[name])
		if value != "" {
			return value
		}
	}

	return ""
}

func stableID(parts ...string) string {
	sum := sha1.Sum([]byte(strings.Join(parts, "::")))
	return hex.EncodeToString(sum[:8])
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false

	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteByte('-')
			lastDash = true
		}
	}

	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "unknown"
	}
	return slug
}

func cloneEntries(entries []Entry) []Entry {
	cloned := make([]Entry, len(entries))
	for i, entry := range entries {
		cloned[i] = entry
		cloned[i].Episodes = append([]Episode(nil), entry.Episodes...)
	}
	return cloned
}
