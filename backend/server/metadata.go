package server

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"

	"abridged-web/backend/catalog"
)

const (
	siteTitle          = "Abridged Catalogue"
	siteName           = "The Abridged Catalogue"
	defaultDescription = "A catalogue of my favourite abridged anime now in web form. This is more of an awareness effort than a full blown archive."
	defaultImagePath   = "/images/icon.jpg"
)

type pageMeta struct {
	Title       string
	Description string
	Canonical   string
	Image       string
	ImageAlt    string
	OGType      string
	NoIndex     bool
}

type browsePageMeta struct {
	Title       string
	Description string
	Path        string
}

var browsePageMetaByType = map[catalog.EntryType]browsePageMeta{
	catalog.EntryTypeSeries: {
		Title:       "Shows",
		Description: "Multi-episode abridged anime series.",
		Path:        "/series",
	},
	catalog.EntryTypeShort: {
		Title:       "Shorts",
		Description: "Short abridged anime and comedy clips.",
		Path:        "/shorts",
	},
	catalog.EntryTypeShot: {
		Title:       "One-Shots",
		Description: "Single-entry abridged stories.",
		Path:        "/shots",
	},
	catalog.EntryTypeSong: {
		Title:       "Songs",
		Description: "Original abridged songs and AMVs.",
		Path:        "/songs",
	},
}

func defaultPageMeta() pageMeta {
	return pageMeta{Description: defaultDescription, OGType: "website"}
}

func (s *appServer) pageMetadata(r *http.Request, status int) pageMeta {
	base := s.publicBaseURL(r)
	route := resolveRoute(s.library, r.URL.Path)

	if status != http.StatusOK || route.kind == routeKindNotFound {
		return notFoundPageMeta(base)
	}

	switch route.kind {
	case routeKindHome, routeKindStatic:
		return homePageMeta(base)
	case routeKindBrowse:
		return s.browsePageMetadata(base, route.entryType)
	case routeKindCreators:
		return creatorsPageMetadata(base)
	case routeKindCreator:
		return s.creatorPageMetadata(base, route.creatorSlug)
	case routeKindEntry:
		if route.entry == nil {
			return notFoundPageMeta(base)
		}
		return entryPageMetadata(base, *route.entry)
	case routeKindEpisode:
		if route.entry == nil || route.episode == nil {
			return notFoundPageMeta(base)
		}
		return episodePageMetadata(base, *route.entry, *route.episode)
	default:
		return homePageMeta(base)
	}
}

func homePageMeta(base string) pageMeta {
	return pageMeta{
		Description: defaultDescription,
		Canonical:   absoluteURL(base, "/"),
		Image:       absoluteURL(base, defaultImagePath),
		ImageAlt:    siteName,
		OGType:      "website",
	}
}

func (s *appServer) browsePageMetadata(base string, entryType catalog.EntryType) pageMeta {
	meta, ok := browsePageMetaByType[entryType]
	if !ok {
		return homePageMeta(base)
	}

	description := meta.Description
	if count := len(s.library.EntriesForType(entryType)); count > 0 {
		noun := "titles"
		if count == 1 {
			noun = "title"
		}
		description = fmt.Sprintf("%s %d %s in the catalogue.", description, count, noun)
	}

	return pageMeta{
		Title:       meta.Title,
		Description: description,
		Canonical:   absoluteURL(base, meta.Path),
		Image:       absoluteURL(base, defaultImagePath),
		ImageAlt:    meta.Title,
		OGType:      "website",
	}
}

func creatorsPageMetadata(base string) pageMeta {
	return pageMeta{
		Title:       "Creators",
		Description: "Every creator in the abridged catalogue.",
		Canonical:   absoluteURL(base, "/creators"),
		Image:       absoluteURL(base, defaultImagePath),
		ImageAlt:    "Creators",
		OGType:      "website",
	}
}

func (s *appServer) creatorPageMetadata(base string, slug string) pageMeta {
	name, ok := s.library.CreatorName(slug)
	if !ok {
		name = humanizeSlug(slug)
	}

	entries := s.library.EntriesForCreator(slug)
	image := defaultImagePath
	for _, entry := range entries {
		if entry.Poster.ThumbnailURL != "" {
			image = entry.Poster.ThumbnailURL
			break
		}
	}

	description := fmt.Sprintf("Abridged works by %s.", name)
	if len(entries) > 0 {
		noun := "entries"
		if len(entries) == 1 {
			noun = "entry"
		}
		description = fmt.Sprintf("%d abridged %s by %s.", len(entries), noun, name)
	}

	return pageMeta{
		Title:       name,
		Description: description,
		Canonical:   absoluteURL(base, "/"+url.PathEscape(slug)),
		Image:       absoluteURL(base, image),
		ImageAlt:    name,
		OGType:      "website",
	}
}

func entryPageMetadata(base string, entry catalog.Entry) pageMeta {
	description := strings.TrimSpace(entry.Description)
	if description == "" {
		description = fmt.Sprintf(
			"Watch %s by %s, an abridged %s in the catalogue.",
			entry.EntryTitle,
			entry.Creator,
			entryTypeDescription(entry),
		)
	}

	image := posterPath(entry)
	if image == "" {
		image = defaultImagePath
	}

	return pageMeta{
		Title:       entry.EntryTitle,
		Description: summarize(description, 300),
		Canonical:   absoluteURL(base, entry.WatchPath()),
		Image:       absoluteURL(base, image),
		ImageAlt:    entry.EntryTitle,
		OGType:      "video.other",
	}
}

func episodePageMetadata(base string, entry catalog.Entry, episode catalog.Episode) pageMeta {
	episodeTitle := episode.DisplayTitle
	if episodeTitle == "" {
		episodeTitle = episode.VideoTitle
	}

	title := entry.EntryTitle + " - " + episodeTitle
	description := strings.TrimSpace(episode.Description)
	if description == "" {
		description = strings.TrimSpace(entry.Description)
	}
	if description == "" {
		description = fmt.Sprintf(
			"Watch %s from %s by %s.",
			episodeTitle,
			entry.EntryTitle,
			entry.Creator,
		)
	}

	image := episode.ThumbnailURL
	if image == "" {
		image = posterPath(entry)
	}
	if image == "" {
		image = defaultImagePath
	}

	canonicalPath := episode.WatchPath(entry)
	if episode.ID == entry.DefaultEpisodeID {
		canonicalPath = entry.WatchPath()
	}

	return pageMeta{
		Title:       title,
		Description: summarize(description, 300),
		Canonical:   absoluteURL(base, canonicalPath),
		Image:       absoluteURL(base, image),
		ImageAlt:    title,
		OGType:      "video.episode",
	}
}

func notFoundPageMeta(base string) pageMeta {
	return pageMeta{
		Title:       "Not Found",
		Description: "This page is not part of the abridged catalogue.",
		Image:       absoluteURL(base, defaultImagePath),
		ImageAlt:    siteName,
		OGType:      "website",
		NoIndex:     true,
	}
}

func posterPath(entry catalog.Entry) string {
	for _, episode := range entry.Episodes {
		if episode.ID == entry.PosterEpisodeID && episode.ThumbnailURL != "" {
			return episode.ThumbnailURL
		}
	}
	for _, episode := range entry.Episodes {
		if episode.ThumbnailURL != "" {
			return episode.ThumbnailURL
		}
	}
	return ""
}

func entryTypeDescription(entry catalog.Entry) string {
	if entry.Subtype == catalog.EntrySubtypeAMV {
		return "AMV"
	}

	switch entry.Type {
	case catalog.EntryTypeSeries:
		return "series"
	case catalog.EntryTypeShort:
		return "short"
	case catalog.EntryTypeShot:
		return "one-shot"
	case catalog.EntryTypeSong:
		return "song"
	default:
		return "entry"
	}
}

func humanizeSlug(slug string) string {
	slug = strings.ReplaceAll(slug, "-", " ")
	if slug == "" {
		return siteName
	}
	return slug
}

func summarize(text string, limit int) string {
	text = strings.Join(strings.Fields(text), " ")
	if text == "" {
		return ""
	}

	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}

	cut := string(runes[:limit])
	if index := strings.LastIndexByte(cut, ' '); index > 0 {
		cut = cut[:index]
	}
	return strings.TrimSpace(cut) + "…"
}

func absoluteURL(base string, requestPath string) string {
	if requestPath == "" {
		return ""
	}
	if strings.HasPrefix(requestPath, "http://") || strings.HasPrefix(requestPath, "https://") {
		return requestPath
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(requestPath, "/")
}

func fullTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return siteTitle
	}
	return title + " | " + siteTitle
}

func socialTitle(meta pageMeta) string {
	if title := strings.TrimSpace(meta.Title); title != "" {
		return title
	}
	return siteName
}

func renderMeta(meta pageMeta) string {
	var b strings.Builder

	b.WriteString("<title>")
	b.WriteString(html.EscapeString(fullTitle(meta.Title)))
	b.WriteString("</title>\n")

	description := strings.TrimSpace(meta.Description)
	if description == "" {
		description = defaultDescription
	}
	b.WriteString(`<meta name="description" content="`)
	b.WriteString(html.EscapeString(description))
	b.WriteString(`">` + "\n")

	if meta.NoIndex {
		b.WriteString(`<meta name="robots" content="noindex, nofollow">` + "\n")
	}
	if meta.Canonical != "" {
		b.WriteString(`<link rel="canonical" href="`)
		b.WriteString(html.EscapeString(meta.Canonical))
		b.WriteString(`">` + "\n")
	}

	ogType := meta.OGType
	if ogType == "" {
		ogType = "website"
	}
	title := socialTitle(meta)
	imageAlt := meta.ImageAlt
	if imageAlt == "" {
		imageAlt = title
	}

	b.WriteString(`<meta property="og:site_name" content="` + html.EscapeString(siteName) + `">` + "\n")
	b.WriteString(`<meta property="og:locale" content="en_US">` + "\n")
	b.WriteString(`<meta property="og:type" content="` + html.EscapeString(ogType) + `">` + "\n")
	b.WriteString(`<meta property="og:title" content="` + html.EscapeString(title) + `">` + "\n")
	b.WriteString(`<meta property="og:description" content="` + html.EscapeString(description) + `">` + "\n")
	if meta.Canonical != "" {
		b.WriteString(`<meta property="og:url" content="` + html.EscapeString(meta.Canonical) + `">` + "\n")
	}
	if meta.Image != "" {
		b.WriteString(`<meta property="og:image" content="` + html.EscapeString(meta.Image) + `">` + "\n")
		b.WriteString(`<meta property="og:image:alt" content="` + html.EscapeString(imageAlt) + `">` + "\n")
	}

	b.WriteString(`<meta name="twitter:card" content="summary_large_image">` + "\n")
	b.WriteString(`<meta name="twitter:title" content="` + html.EscapeString(title) + `">` + "\n")
	b.WriteString(`<meta name="twitter:description" content="` + html.EscapeString(description) + `">` + "\n")
	if meta.Image != "" {
		b.WriteString(`<meta name="twitter:image" content="` + html.EscapeString(meta.Image) + `">` + "\n")
		b.WriteString(`<meta name="twitter:image:alt" content="` + html.EscapeString(imageAlt) + `">` + "\n")
	}

	return b.String()
}

func injectHead(indexHTML string, metaHTML string) string {
	const titleOpen = "<title>"
	if start := strings.Index(indexHTML, titleOpen); start >= 0 {
		if endOffset := strings.Index(indexHTML[start:], "</title>"); endOffset >= 0 {
			end := start + endOffset + len("</title>")
			return indexHTML[:start] + metaHTML + indexHTML[end:]
		}
	}

	if strings.Contains(indexHTML, "<!--app-meta-->") {
		return strings.Replace(indexHTML, "<!--app-meta-->", metaHTML, 1)
	}

	if end := strings.Index(indexHTML, "</head>"); end >= 0 {
		return indexHTML[:end] + metaHTML + indexHTML[end:]
	}

	return indexHTML
}
