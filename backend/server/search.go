package server

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"abridged-web/backend/catalog"
)

const (
	maxSearchQueryLength = 120
	maxSearchTerms       = 8
	searchCacheTTL       = 5 * time.Minute
	searchCacheMaxItems  = 256
)

type cachedSearch struct {
	results []catalog.EntrySummary
	expires time.Time
}

type searchCache struct {
	mu    sync.Mutex
	items map[string]cachedSearch
}

func (c *searchCache) get(key string) ([]catalog.EntrySummary, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, ok := c.items[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(item.expires) {
		delete(c.items, key)
		return nil, false
	}
	return item.results, true
}

func (c *searchCache) set(key string, results []catalog.EntrySummary) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.items == nil {
		c.items = make(map[string]cachedSearch)
	}
	if len(c.items) >= searchCacheMaxItems {
		now := time.Now()
		for cachedKey, item := range c.items {
			if now.After(item.expires) {
				delete(c.items, cachedKey)
			}
		}
		for cachedKey := range c.items {
			if len(c.items) < searchCacheMaxItems {
				break
			}
			delete(c.items, cachedKey)
		}
	}
	c.items[key] = cachedSearch{
		results: results,
		expires: time.Now().Add(searchCacheTTL),
	}
}

func normalizeSearchKey(query string) string {
	return strings.ToLower(strings.Join(strings.Fields(query), " "))
}

// isSameOriginSearchRequest rejects direct and cross-site calls made from a
// browser. Non-browser clients can still spoof these headers, so this is not an
// authentication boundary.
func isSameOriginSearchRequest(r *http.Request) bool {
	switch r.Header.Get("Sec-Fetch-Site") {
	case "same-origin", "same-site":
		return true
	case "":
		break
	default:
		return false
	}

	if origin := r.Header.Get("Origin"); origin != "" {
		return hasSameHost(origin, r.Host)
	}
	if referer := r.Header.Get("Referer"); referer != "" {
		return hasSameHost(referer, r.Host)
	}
	return false
}

func hasSameHost(rawURL string, host string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return strings.EqualFold(parsed.Host, host)
}
