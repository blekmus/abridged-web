package catalog

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
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

type CreatorSummary struct {
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	EntryCount int    `json:"entryCount"`
}

type CatalogResponse struct {
	Series   []EntrySummary   `json:"series"`
	Shorts   []EntrySummary   `json:"shorts"`
	Shots    []EntrySummary   `json:"shots"`
	Songs    []EntrySummary   `json:"songs"`
	SongAMVs []EntrySummary   `json:"songAmvs"`
	Creators []CreatorSummary `json:"creators"`
}

type Library struct {
	root          string
	metadata      *videoMetadataCache
	all           []Entry
	byType        map[EntryType][]Entry
	byEntryID     map[string]Entry
	byEpisodeID   map[string]EpisodeAsset
	byCreatorSlug map[string][]Entry
	cacheTag      string
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
	slog.Info("catalog load started", "archive_root", absRoot)
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
		slog.Info("catalog category registered", "type", category.kind, "entries", len(entries), "total_episodes", countEpisodes(entries))

		if category.kind == EntryTypeSong {
			amvs, scanErr := lib.scanCategory(filepath.Join(absRoot, category.dir, "AMV"), category.kind, EntrySubtypeAMV)
			if scanErr != nil {
				return nil, scanErr
			}
			lib.registerEntries(category.kind, amvs)
			slog.Info("catalog category registered", "type", category.kind, "subtype", EntrySubtypeAMV, "entries", len(amvs), "total_episodes", countEpisodes(amvs))
		}
	}

	cacheTag, err := catalogueCacheTag(lib.all)
	if err != nil {
		return nil, err
	}
	lib.cacheTag = cacheTag

	slog.Info("catalog load complete", "entries", len(lib.all), "episodes", len(lib.byEpisodeID), "cache_tag", cacheTag)
	metadata.logLookupSummary()
	return lib, nil
}

// CacheTag changes whenever the in-memory catalogue content changes. It is used
// as the ETag for JSON responses. Video and thumbnail content is validated
// separately with file metadata because those URLs are stable across re-encodes.
func (l *Library) CacheTag() string {
	return l.cacheTag
}

// catalogueCacheVersion must be bumped whenever the shape of an API response
// changes. Without it, a client holding a pre-deploy ETag would receive a 304
// and keep using a response that the new frontend can no longer parse.
const catalogueCacheVersion = 2

func catalogueCacheTag(entries []Entry) (string, error) {
	data, err := json.Marshal(entries)
	if err != nil {
		return "", fmt.Errorf("hash catalogue: %w", err)
	}

	hash := sha1.New()
	if _, err := fmt.Fprintf(hash, "catalogue-v%d\n", catalogueCacheVersion); err != nil {
		return "", fmt.Errorf("hash catalogue: %w", err)
	}
	if _, err := hash.Write(data); err != nil {
		return "", fmt.Errorf("hash catalogue: %w", err)
	}

	return `"` + hex.EncodeToString(hash.Sum(nil)) + `"`, nil
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
		Creators: l.creatorSummaries(),
	}
}

func (l *Library) creatorSummaries() []CreatorSummary {
	summaries := make([]CreatorSummary, 0, len(l.byCreatorSlug))
	for slug, entries := range l.byCreatorSlug {
		if len(entries) == 0 {
			continue
		}
		summaries = append(summaries, CreatorSummary{
			Slug:       slug,
			Name:       entries[0].Creator,
			EntryCount: len(entries),
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		left := strings.ToLower(summaries[i].Name)
		right := strings.ToLower(summaries[j].Name)
		if left != right {
			return left < right
		}
		return summaries[i].Slug < summaries[j].Slug
	})

	return summaries
}

func (l *Library) EntriesForType(entryType EntryType) []EntrySummary {
	return entrySummaries(l.byType[entryType])
}

func (l *Library) EntriesForCreator(slug string) []EntrySummary {
	return entrySummaries(l.byCreatorSlug[slug])
}

func (l *Library) HasCreator(slug string) bool {
	return len(l.byCreatorSlug[slug]) > 0
}

func (l *Library) CreatorName(slug string) (string, bool) {
	entries := l.byCreatorSlug[slug]
	if len(entries) == 0 {
		return "", false
	}
	return entries[0].Creator, true
}

// EntryForWatchPath mirrors frontend/src/app/watchPaths.ts so the server can
// distinguish a real watch URL from a 404 before serving the SPA shell.
func (l *Library) EntryForWatchPath(creatorSlug string, entrySlug string) (Entry, bool) {
	for _, entry := range l.byCreatorSlug[creatorSlug] {
		if routeSlug(entry.EntryTitle) == entrySlug {
			return entry, true
		}
	}
	return Entry{}, false
}

func (l *Library) EpisodeForWatchPath(entry Entry, episodeSlug string) (Episode, bool) {
	for _, episode := range entry.Episodes {
		title := episode.DisplayTitle
		if title == "" {
			title = episode.VideoTitle
		}
		if routeSlug(title) == episodeSlug {
			return episode, true
		}
	}
	return Episode{}, false
}

func (l *Library) Entry(entryID string) (Entry, bool) {
	entry, ok := l.byEntryID[entryID]
	return entry, ok
}

// Entries returns a copy of the full catalogue for consumers that need episode
// data rather than the summary shape used by browse responses.
func (l *Library) Entries() []Entry {
	return cloneEntries(l.all)
}

// WatchPath is the canonical entry URL. It mirrors entryWatchPath in
// frontend/src/app/watchPaths.ts.
func (e Entry) WatchPath() string {
	return "/" + url.PathEscape(e.CreatorSlug) + "/" + url.PathEscape(routeSlug(e.EntryTitle))
}

// WatchPath is the canonical episode URL for an entry.
func (e Episode) WatchPath(entry Entry) string {
	title := e.DisplayTitle
	if title == "" {
		title = e.VideoTitle
	}
	return entry.WatchPath() + "/" + url.PathEscape(routeSlug(title))
}

const searchResultLimit = 50

func (l *Library) Search(query string, limit int) []EntrySummary {
	terms := searchTerms(query)
	if len(terms) == 0 {
		return []EntrySummary{}
	}
	if limit <= 0 || limit > searchResultLimit {
		limit = searchResultLimit
	}

	type candidate struct {
		entry Entry
		score int
	}
	matches := make([]candidate, 0)
	for _, entry := range l.all {
		score, ok := entrySearchScore(entry, terms)
		if !ok {
			continue
		}
		matches = append(matches, candidate{entry: entry, score: score})
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		if !strings.EqualFold(matches[i].entry.EntryTitle, matches[j].entry.EntryTitle) {
			return strings.ToLower(matches[i].entry.EntryTitle) < strings.ToLower(matches[j].entry.EntryTitle)
		}
		return strings.ToLower(matches[i].entry.Creator) < strings.ToLower(matches[j].entry.Creator)
	})

	if len(matches) > limit {
		matches = matches[:limit]
	}

	results := make([]EntrySummary, len(matches))
	for index, match := range matches {
		results[index] = entrySummary(match.entry)
	}
	return results
}

func searchTerms(query string) []string {
	fields := strings.Fields(strings.ToLower(query))
	seen := make(map[string]bool, len(fields))
	terms := make([]string, 0, len(fields))
	for _, field := range fields {
		if seen[field] {
			continue
		}
		seen[field] = true
		terms = append(terms, field)
	}
	return terms
}

func entrySearchScore(entry Entry, terms []string) (int, bool) {
	fields := []struct {
		text   string
		weight int
	}{
		{text: strings.ToLower(entry.EntryTitle), weight: 4},
		{text: strings.ToLower(entry.Creator), weight: 3},
		{text: typeSearchKeywords(entry), weight: 2},
		{text: strings.ToLower(entry.Description), weight: 1},
	}

	score := 0
	for _, term := range terms {
		matched := false
		for _, field := range fields {
			if field.text == "" || !strings.Contains(field.text, term) {
				continue
			}
			score += field.weight
			matched = true
		}
		if !matched {
			return 0, false
		}
	}
	return score, true
}

func typeSearchKeywords(entry Entry) string {
	switch entry.Type {
	case EntryTypeSeries:
		return "series shows show"
	case EntryTypeShort:
		return "short shorts"
	case EntryTypeShot:
		return "shot shots one-shot one-shots"
	case EntryTypeSong:
		if entry.Subtype == EntrySubtypeAMV {
			return "song songs amv"
		}
		return "song songs"
	default:
		return ""
	}
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
			slog.Info("catalog category missing", "path", categoryPath, "type", entryType, "subtype", firstSubtype(subtype))
			return nil, nil
		}
		return nil, fmt.Errorf("read category %s: %w", categoryPath, err)
	}
	slog.Info("catalog category scan started", "path", categoryPath, "type", entryType, "subtype", firstSubtype(subtype), "items", len(items))

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
			slog.Warn("catalog failed to stat single-release video", "path", candidate, "err", err)
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
			slog.Warn("catalog failed to read entry description", "path", fullPath, "err", err)
		}
	}

	items, err := os.ReadDir(entryPath)
	if err != nil {
		slog.Warn("catalog failed to read entry directory while searching descriptions", "path", entryPath, "err", err)
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
			slog.Warn("catalog failed to read fallback entry description", "path", filepath.Join(entryPath, item.Name()), "err", err)
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
	slog.Info("metadata cache loading", "root", cache.root, "path", cache.path)
	if err := cache.load(); err != nil {
		return nil, err
	}
	slog.Info("metadata cache ready", "path", cache.path, "normalized_items", len(cache.items))
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
	slog.Info("metadata cache file read", "path", c.path, "bytes", len(data))

	var payload videoMetadataCacheFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("read metadata file %s: %w", c.path, err)
	}
	if payload.Version != videoMetadataCacheVersion || payload.Items == nil {
		return fmt.Errorf("metadata file %s is not a valid cache file", c.path)
	}
	slog.Info("metadata cache file parsed", "path", c.path, "version", payload.Version, "raw_items", len(payload.Items))

	c.items = normalizeVideoMetadataItems(payload.Items)
	slog.Info("metadata cache normalized", "path", c.path, "raw_items", len(payload.Items), "normalized_items", len(c.items))
	return nil
}

func (c *videoMetadataCache) Read(videoPath string) videoMetadata {
	info, err := os.Stat(videoPath)
	if err != nil {
		c.stats.statFailures++
		appendExample(&c.stats.statExamples, fmt.Sprintf("%s: %v", videoPath, err))
		slog.Warn("metadata failed to stat video", "path", videoPath, "err", err)
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
	slog.Info(
		"metadata lookup summary",
		"matched_videos", c.stats.hits,
		"videos_without_metadata", c.stats.misses,
		"videos_with_outdated_metadata", c.stats.stale,
		"video_stat_failures", c.stats.statFailures,
	)
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
	slog.Info(prefix, "count", len(examples), "examples", examples)
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
				slog.Warn("metadata ignored empty normalized key", "original_key", key)
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
		slog.Warn("metadata cache unavailable", "video", videoPath)
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

// routeSlug is the server-side equivalent of slugifyPathSegment in
// frontend/src/app/watchPaths.ts. It must stay in sync with that function
// because the server uses it to decide whether a watch URL is a real 200 or a
// 404 before the SPA loads.
func routeSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "&", " and ")

	var b strings.Builder
	lastDash := false

	for _, r := range norm.NFKD.String(value) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}

		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteByte('-')
			lastDash = true
		}
	}

	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "untitled"
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
