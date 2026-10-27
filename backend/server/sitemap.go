package server

import (
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"abridged-web/backend/catalog"
	"github.com/gin-gonic/gin"
)

const (
	robotsCacheControl  = "public, max-age=86400"
	sitemapCacheControl = "public, no-cache"
	robotsContentType   = "text/plain; charset=utf-8"
	sitemapContentType  = "application/xml; charset=utf-8"
)

var reservedCreatorSlugs = map[string]struct{}{
	"api":      {},
	"creators": {},
	"healthz":  {},
	"series":   {},
	"shorts":   {},
	"shots":    {},
	"songs":    {},
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

func normalizePublicBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("PUBLIC_BASE_URL: %w", err)
	}
	if scheme := strings.ToLower(parsed.Scheme); scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("PUBLIC_BASE_URL must use http or https")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("PUBLIC_BASE_URL must include a host")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("PUBLIC_BASE_URL must not include credentials, query, or fragment")
	}
	if strings.Trim(parsed.Path, "/") != "" {
		return "", fmt.Errorf("PUBLIC_BASE_URL must not include a path")
	}

	parsed.Path = ""
	parsed.RawPath = ""
	return parsed.String(), nil
}

func (s *appServer) publicBaseURL(r *http.Request) string {
	if s.baseURL != "" {
		return s.baseURL
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := forwardedProto(r.Header.Get("X-Forwarded-Proto")); proto != "" {
		scheme = proto
	}

	host := r.Host
	if host == "" {
		host = "localhost"
	}
	return scheme + "://" + host
}

func forwardedProto(value string) string {
	if index := strings.IndexByte(value, ','); index >= 0 {
		value = value[:index]
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "http":
		return "http"
	case "https":
		return "https"
	default:
		return ""
	}
}

func (s *appServer) handleRobots(c *gin.Context) {
	base := s.publicBaseURL(c.Request)
	body := []byte(robotsTxt(base))
	writeCachedBytes(c, robotsCacheControl, bytesETag(body), robotsContentType, body)
}

func (s *appServer) handleSitemap(c *gin.Context) {
	base := s.publicBaseURL(c.Request)
	body, err := s.buildSitemap(base)
	if err != nil {
		slog.Error("build sitemap failed", "request_id", requestID(c), "err", err)
		abortRequest(c, http.StatusInternalServerError, "sitemap unavailable")
		return
	}

	writeCachedBytes(c, sitemapCacheControl, bytesETag(body), sitemapContentType, body)
}

func robotsTxt(base string) string {
	lines := []string{
		"User-agent: *",
		"Allow: /",
		"Disallow: /api/",
		"Disallow: /assets/",
		"Disallow: /audio/",
		"Disallow: /fonts/",
		"Disallow: /healthz",
		"Disallow: /images/",
		"Disallow: /index.html",
		"Disallow: /thumb/",
		"Disallow: /video/",
		"",
		"Sitemap: " + base + "/sitemap.xml",
		"",
	}
	return strings.Join(lines, "\n")
}

func (s *appServer) buildSitemap(base string) ([]byte, error) {
	entries := s.library.Entries()
	urls := make([]sitemapURL, 0, len(entries)*2+8)
	seen := make(map[string]struct{}, len(entries)*2+8)

	add := func(loc string, lastMod string) {
		if loc == "" {
			return
		}
		if _, ok := seen[loc]; ok {
			return
		}
		seen[loc] = struct{}{}
		urls = append(urls, sitemapURL{Loc: loc, LastMod: lastMod})
	}

	add(base+"/", "")

	for _, browsePath := range []string{"/creators", "/series", "/shorts", "/shots", "/songs"} {
		add(base+browsePath, "")
	}

	creatorDates := make(map[string]string, len(entries))
	for _, entry := range entries {
		if _, ok := creatorDates[entry.CreatorSlug]; !ok {
			creatorDates[entry.CreatorSlug] = ""
		}
		if date := latestEntryDate(entry); date > creatorDates[entry.CreatorSlug] {
			creatorDates[entry.CreatorSlug] = date
		}
	}

	creatorSlugs := make([]string, 0, len(creatorDates))
	for slug := range creatorDates {
		if _, reserved := reservedCreatorSlugs[slug]; reserved {
			continue
		}
		creatorSlugs = append(creatorSlugs, slug)
	}
	sort.Strings(creatorSlugs)
	for _, slug := range creatorSlugs {
		add(base+"/"+url.PathEscape(slug), creatorDates[slug])
	}

	for _, entry := range entries {
		add(base+entry.WatchPath(), latestEntryDate(entry))
		if entry.Type != catalog.EntryTypeSeries {
			continue
		}
		for _, episode := range entry.Episodes {
			// The entry URL is canonical for its default episode. List only the
			// other episode URLs so the sitemap matches the canonical tags.
			if episode.ID == entry.DefaultEpisodeID {
				continue
			}
			add(base+episode.WatchPath(entry), sitemapDate(episode.Date))
		}
	}

	set := sitemapURLSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}
	data, err := xml.MarshalIndent(set, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal sitemap: %w", err)
	}

	return append([]byte(xml.Header), data...), nil
}

func latestEntryDate(entry catalog.Entry) string {
	latest := ""
	for _, episode := range entry.Episodes {
		if date := sitemapDate(episode.Date); date > latest {
			latest = date
		}
	}
	return latest
}

func sitemapDate(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if len(raw) == 8 {
		if parsed, err := time.Parse("20060102", raw); err == nil {
			return parsed.Format("2006-01-02")
		}
	}

	if parsed, err := time.Parse("2006-01-02", raw); err == nil {
		return parsed.Format("2006-01-02")
	}

	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.UTC().Format("2006-01-02")
	}

	return ""
}

func writeCachedBytes(c *gin.Context, cacheControl string, etag string, contentType string, body []byte) {
	c.Header("Cache-Control", cacheControl)
	c.Header("Etag", etag)
	if ifNoneMatch(c.GetHeader("If-None-Match"), etag) {
		c.AbortWithStatus(http.StatusNotModified)
		return
	}
	c.Data(http.StatusOK, contentType, body)
}
