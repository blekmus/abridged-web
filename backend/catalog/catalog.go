package catalog

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
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
	EntryTypeSong   EntryType = "song"
)

type EntrySubtype string

const (
	EntrySubtypeAMV EntrySubtype = "amv"
)

type Entry struct {
	ID               string       `json:"id"`
	Type             EntryType    `json:"type"`
	Subtype          EntrySubtype `json:"subtype,omitempty"`
	Creator          string       `json:"creator"`
	CreatorSlug      string       `json:"creatorSlug"`
	EntryTitle       string       `json:"entryTitle"`
	Description      string       `json:"description,omitempty"`
	DefaultEpisodeID string       `json:"defaultEpisodeId"`
	PosterEpisodeID  string       `json:"posterEpisodeId"`
	Episodes         []Episode    `json:"episodes"`
}

type EntrySummary struct {
	ID              string       `json:"id"`
	Type            EntryType    `json:"type"`
	Subtype         EntrySubtype `json:"subtype,omitempty"`
	Creator         string       `json:"creator"`
	CreatorSlug     string       `json:"creatorSlug"`
	EntryTitle      string       `json:"entryTitle"`
	PosterEpisodeID string       `json:"posterEpisodeId"`
	Poster          EpisodeCard  `json:"poster"`
}

type EpisodeCard struct {
	ID              string `json:"id"`
	DurationSeconds int    `json:"durationSeconds"`
	ThumbnailURL    string `json:"thumbnailUrl,omitempty"`
	HasThumbnail    bool   `json:"hasThumbnail"`
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
	stats videoMetadataCacheStats
}

type videoMetadataCacheStats struct {
	hits          int
	misses        int
	stale         int
	statFailures  int
	missExamples  []string
	staleExamples []string
	statExamples  []string
}

type CatalogResponse struct {
	Series   []EntrySummary `json:"series"`
	Shorts   []EntrySummary `json:"shorts"`
	Shots    []EntrySummary `json:"shots"`
	Songs    []EntrySummary `json:"songs"`
	SongAMVs []EntrySummary `json:"songAmvs"`
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

var supportedVideoExtensions = []string{".mp4", ".webm"}

func Load(root string) (*Library, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve archive root: %w", err)
	}
	log.Printf("catalog: load started archive_root=%q", absRoot)
	if err := validateArchiveRoot(absRoot); err != nil {
		return nil, err
	}

	metadata, err := newVideoMetadataCache(absRoot)
	if err != nil {
		return nil, err
	}

	lib := &Library{
		root:          absRoot,
		metadata:      metadata,
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
		{dir: "Songs", kind: EntryTypeSong},
	}

	for _, category := range categoryDirs {
		entries, scanErr := lib.scanCategory(filepath.Join(absRoot, category.dir), category.kind)
		if scanErr != nil {
			return nil, scanErr
		}
		lib.registerEntries(category.kind, entries)
		log.Printf("catalog: registered category type=%s entries=%d total_episodes=%d", category.kind, len(entries), countEpisodes(entries))

		if category.kind == EntryTypeSong {
			amvs, scanErr := lib.scanCategory(filepath.Join(absRoot, category.dir, "AMV"), category.kind, EntrySubtypeAMV)
			if scanErr != nil {
				return nil, scanErr
			}
			lib.registerEntries(category.kind, amvs)
			log.Printf("catalog: registered category type=%s subtype=%s entries=%d total_episodes=%d", category.kind, EntrySubtypeAMV, len(amvs), countEpisodes(amvs))
		}
	}

	log.Printf("catalog: load complete entries=%d episodes=%d", len(lib.all), len(lib.byEpisodeID))
	metadata.logLookupSummary()
	return lib, nil
}

func validateArchiveRoot(root string) error {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("archive path does not exist: %s", root)
		}
		return fmt.Errorf("read archive path %s: %w", root, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("archive path is not a directory: %s", root)
	}
	return nil
}

func (l *Library) CatalogResponse() CatalogResponse {
	return CatalogResponse{
		Series:   entrySummaries(l.byType[EntryTypeSeries]),
		Shorts:   entrySummaries(l.byType[EntryTypeShort]),
		Shots:    entrySummaries(l.byType[EntryTypeShot]),
		Songs:    entrySummaries(entriesWithSubtype(l.byType[EntryTypeSong], "")),
		SongAMVs: entrySummaries(entriesWithSubtype(l.byType[EntryTypeSong], EntrySubtypeAMV)),
	}
}

func (l *Library) EntriesForType(entryType EntryType) []EntrySummary {
	return entrySummaries(l.byType[entryType])
}

func (l *Library) EntriesForCreator(slug string) []EntrySummary {
	return entrySummaries(l.byCreatorSlug[slug])
}

func (l *Library) Entry(entryID string) (Entry, bool) {
	entry, ok := l.byEntryID[entryID]
	return entry, ok
}

func (l *Library) EpisodeAsset(episodeID string) (EpisodeAsset, bool) {
	asset, ok := l.byEpisodeID[episodeID]
	return asset, ok
}

func (l *Library) registerEntries(entryType EntryType, entries []Entry) {
	l.byType[entryType] = append(l.byType[entryType], entries...)
	l.all = append(l.all, entries...)
	for _, entry := range entries {
		l.byEntryID[entry.ID] = entry
		l.byCreatorSlug[entry.CreatorSlug] = append(l.byCreatorSlug[entry.CreatorSlug], entry)
		for _, episode := range entry.Episodes {
			l.byEpisodeID[episode.ID] = EpisodeAsset{
				VideoPath:     episode.VideoPath,
				ThumbnailPath: episode.ThumbnailPath,
			}
		}
	}
}

func (l *Library) scanCategory(categoryPath string, entryType EntryType, subtype ...EntrySubtype) ([]Entry, error) {
	items, err := os.ReadDir(categoryPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("catalog: category missing path=%q type=%s subtype=%s", categoryPath, entryType, firstSubtype(subtype))
			return nil, nil
		}
		return nil, fmt.Errorf("read category %s: %w", categoryPath, err)
	}
	log.Printf("catalog: category scan started path=%q type=%s subtype=%s items=%d", categoryPath, entryType, firstSubtype(subtype), len(items))

	var entries []Entry
	entrySubtype := firstSubtype(subtype)
	for _, item := range items {
		if !item.IsDir() {
			continue
		}
		if entryType == EntryTypeSong && entrySubtype == "" && strings.EqualFold(item.Name(), "AMV") {
			continue
		}

		entryPath := filepath.Join(categoryPath, item.Name())
		entry, ok, scanErr := l.scanEntry(entryPath, item.Name(), entryType, entrySubtype)
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

func (l *Library) scanEntry(entryPath, folderName string, entryType EntryType, subtype EntrySubtype) (Entry, bool, error) {
	creator, title := parseEntryFolder(folderName)
	idParts := []string{string(entryType)}
	if subtype != "" {
		idParts = append(idParts, string(subtype))
	}
	idParts = append(idParts, folderName)
	entryID := stableID(idParts...)

	entry := Entry{
		ID:          entryID,
		Type:        entryType,
		Subtype:     subtype,
		Creator:     creator,
		CreatorSlug: slugify(creator),
		EntryTitle:  title,
		Description: readEntryDescription(entryPath),
	}
	var episodes []episodeRecord
	switch entryType {
	case EntryTypeShort, EntryTypeShot, EntryTypeSong:
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
	videoPath := ""
	for _, ext := range supportedVideoExtensions {
		candidate := filepath.Join(entryPath, "1"+ext)
		if _, err := os.Stat(candidate); err == nil {
			videoPath = candidate
			break
		} else if !os.IsNotExist(err) {
			log.Printf("catalog: failed to stat single-release video candidate path=%q err=%v", candidate, err)
		}
	}
	if videoPath == "" {
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
	if entryType == EntryTypeSong {
		displayLabel = "SONG"
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
		if item.IsDir() || !isSupportedVideoFile(item.Name()) {
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

func isSupportedVideoFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	for _, supported := range supportedVideoExtensions {
		if ext == supported {
			return true
		}
	}
	return false
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
		} else if !os.IsNotExist(err) {
			log.Printf("catalog: failed to read entry description candidate path=%q err=%v", fullPath, err)
		}
	}

	items, err := os.ReadDir(entryPath)
	if err != nil {
		log.Printf("catalog: failed to read entry directory while searching descriptions path=%q err=%v", entryPath, err)
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
		} else {
			log.Printf("catalog: failed to read fallback entry description path=%q err=%v", filepath.Join(entryPath, item.Name()), err)
		}
	}

	return ""
}

const (
	videoMetadataCacheVersion  = 2
	videoMetadataCacheFilename = ".abridged-video-metadata.json"
)

func newVideoMetadataCache(root string) (*videoMetadataCache, error) {
	cache := &videoMetadataCache{
		root:  root,
		path:  filepath.Join(root, videoMetadataCacheFilename),
		items: map[string]cachedVideoMetadata{},
	}
	log.Printf("metadata: loading cache root=%q path=%q", cache.root, cache.path)
	if err := cache.load(); err != nil {
		return nil, err
	}
	log.Printf("metadata: cache ready path=%q normalized_items=%d", cache.path, len(cache.items))
	return cache, nil
}

func (c *videoMetadataCache) load() error {
	data, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("metadata file with the following path does not exist: %s", c.path)
		}
		return fmt.Errorf("read metadata file %s: %w", c.path, err)
	}
	log.Printf("metadata: cache file read path=%q bytes=%d", c.path, len(data))

	var payload videoMetadataCacheFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("read metadata file %s: %w", c.path, err)
	}
	if payload.Version != videoMetadataCacheVersion || payload.Items == nil {
		return fmt.Errorf("metadata file %s is not a valid cache file", c.path)
	}
	log.Printf("metadata: cache file parsed path=%q version=%d raw_items=%d", c.path, payload.Version, len(payload.Items))

	c.items = normalizeVideoMetadataItems(payload.Items)
	log.Printf("metadata: cache normalized path=%q raw_items=%d normalized_items=%d", c.path, len(payload.Items), len(c.items))
	return nil
}

func (c *videoMetadataCache) Read(videoPath string) videoMetadata {
	info, err := os.Stat(videoPath)
	if err != nil {
		c.stats.statFailures++
		appendExample(&c.stats.statExamples, fmt.Sprintf("%s: %v", videoPath, err))
		log.Printf("metadata: failed to stat video for metadata lookup path=%q err=%v", videoPath, err)
		return videoMetadata{}
	}

	keys := c.keys(videoPath)
	foundKey := false
	for _, key := range keys {
		cached, ok := c.items[key]
		if !ok {
			continue
		}
		foundKey = true
		if cached.matches(info) {
			c.stats.hits++
			return cached.metadata()
		}
		c.stats.stale++
		appendExample(&c.stats.staleExamples, fmt.Sprintf("%s key=%s video_size=%d cached_size=%d video_mod_time=%d cached_mod_time=%d", videoPath, key, info.Size(), cached.Size, info.ModTime().UnixNano(), cached.ModTimeUnixNano))
	}

	if !foundKey {
		c.stats.misses++
		appendExample(&c.stats.missExamples, fmt.Sprintf("%s keys=%s", videoPath, strings.Join(keys, ",")))
	}
	return videoMetadata{}
}

func (c *videoMetadataCache) logLookupSummary() {
	log.Printf("metadata: lookup summary matched_videos=%d videos_without_metadata=%d videos_with_outdated_metadata=%d video_stat_failures=%d", c.stats.hits, c.stats.misses, c.stats.stale, c.stats.statFailures)
	logExamples("metadata: videos without metadata entries", c.stats.missExamples)
	logExamples("metadata: videos with outdated metadata entries", c.stats.staleExamples)
	logExamples("metadata: stat failure examples", c.stats.statExamples)
}

func appendExample(examples *[]string, value string) {
	if len(*examples) >= 5 {
		return
	}
	*examples = append(*examples, value)
}

func logExamples(prefix string, examples []string) {
	if len(examples) == 0 {
		return
	}
	log.Printf("%s count=%d examples=%q", prefix, len(examples), examples)
}

func (c *videoMetadataCache) keys(videoPath string) []string {
	keys := []string{metadataRelativeKey(c.root, videoPath), filepath.ToSlash(videoPath)}
	seen := make(map[string]bool, len(keys))
	unique := keys[:0]
	for _, key := range keys {
		key = normalizeMetadataKey(key)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, key)
	}
	return unique
}

func metadataRelativeKey(root, videoPath string) string {
	relative, err := filepath.Rel(root, videoPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return filepath.ToSlash(videoPath)
	}
	return filepath.ToSlash(relative)
}

func normalizeVideoMetadataItems(items map[string]cachedVideoMetadata) map[string]cachedVideoMetadata {
	normalized := make(map[string]cachedVideoMetadata, len(items))
	for key, metadata := range items {
		for _, normalizedKey := range metadataKeyAliases(key) {
			if normalizedKey == "" {
				log.Printf("metadata: ignored empty normalized key original_key=%q", key)
				continue
			}
			normalized[normalizedKey] = metadata
		}
	}
	return normalized
}

func metadataKeyAliases(key string) []string {
	normalized := normalizeMetadataKey(key)
	aliases := []string{normalized}
	for _, marker := range []string{"/Series/", "/Shorts/", "/Shots/", "/Songs/"} {
		if index := strings.Index(normalized, marker); index >= 0 {
			aliases = append(aliases, strings.TrimPrefix(normalized[index:], "/"))
			break
		}
	}
	return aliases
}

func normalizeMetadataKey(key string) string {
	return strings.TrimLeft(filepath.ToSlash(strings.ReplaceAll(strings.TrimSpace(key), "\\", "/")), "/")
}

func (m cachedVideoMetadata) matches(info os.FileInfo) bool {
	return m.Size == info.Size()
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
		log.Printf("metadata: metadata cache unavailable video=%q", videoPath)
		return videoMetadata{}
	}
	return l.metadata.Read(videoPath)
}

func countEpisodes(entries []Entry) int {
	total := 0
	for _, entry := range entries {
		total += len(entry.Episodes)
	}
	return total
}

func firstSubtype(subtypes []EntrySubtype) EntrySubtype {
	if len(subtypes) == 0 {
		return ""
	}
	return subtypes[0]
}

func entriesWithSubtype(entries []Entry, subtype EntrySubtype) []Entry {
	filtered := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.Subtype == subtype {
			filtered = append(filtered, entry)
		}
	}
	return filtered
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

func entrySummaries(entries []Entry) []EntrySummary {
	summaries := make([]EntrySummary, 0, len(entries))
	for _, entry := range entries {
		summaries = append(summaries, entrySummary(entry))
	}
	return summaries
}

func entrySummary(entry Entry) EntrySummary {
	poster := EpisodeCard{}
	for _, episode := range entry.Episodes {
		if episode.ID != entry.PosterEpisodeID {
			continue
		}
		poster = episodeCard(episode)
		break
	}
	if poster.ID == "" && len(entry.Episodes) > 0 {
		poster = episodeCard(entry.Episodes[0])
	}

	return EntrySummary{
		ID:              entry.ID,
		Type:            entry.Type,
		Subtype:         entry.Subtype,
		Creator:         entry.Creator,
		CreatorSlug:     entry.CreatorSlug,
		EntryTitle:      entry.EntryTitle,
		PosterEpisodeID: entry.PosterEpisodeID,
		Poster:          poster,
	}
}

func episodeCard(episode Episode) EpisodeCard {
	return EpisodeCard{
		ID:              episode.ID,
		DurationSeconds: episode.DurationSeconds,
		ThumbnailURL:    episode.ThumbnailURL,
		HasThumbnail:    episode.HasThumbnail,
	}
}
