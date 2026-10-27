package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// Vite fingerprints everything under assets/, so those URLs can be cached
	// forever. index.html is served with no-cache and points at the new hashes
	// after each deploy.
	immutableAssetCacheControl = "public, max-age=31536000, immutable"

	// Files copied from public/ (fonts, images, audio) are not fingerprinted.
	// ETags still let browsers revalidate, but a bounded max-age avoids a
	// revalidation round trip for every icon/font on every page load.
	staticAssetCacheControl = "public, max-age=86400, must-revalidate"

	// Media URLs are stable across re-encodes and thumbnail regeneration. Do
	// not use immutable here: replacing a file at the same URL must eventually
	// be visible. One hour bounds that staleness while still letting the player
	// cache chunks during playback.
	videoCacheControl     = "public, max-age=3600, must-revalidate"
	thumbnailCacheControl = "public, max-age=3600, must-revalidate"

	// JSON responses revalidate on every request. They are small and a 304 is
	// much cheaper than serving data that changed after a restart.
	jsonCacheControl  = "public, no-cache"
	indexCacheControl = "no-cache"

	noStoreCacheControl = "no-store"
)

func setNoStore(c *gin.Context) {
	c.Header("Cache-Control", noStoreCacheControl)
}

func writeJSONCached(c *gin.Context, cacheTag string, status int, payload any) {
	if cacheTag != "" {
		c.Header("Etag", cacheTag)
	}
	c.Header("Cache-Control", jsonCacheControl)

	if cacheTag != "" && ifNoneMatch(c.GetHeader("If-None-Match"), cacheTag) {
		c.AbortWithStatus(http.StatusNotModified)
		return
	}

	c.JSON(status, payload)
}

// ifNoneMatch implements the weak comparison required for If-None-Match.
// nginx weakens upstream ETags when it compresses a response, so a browser may
// send back W/"..." even though we emitted a strong ETag.
func ifNoneMatch(header string, etag string) bool {
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if candidate == "*" {
			return true
		}
		candidate = strings.TrimPrefix(candidate, "W/")
		candidate = strings.TrimPrefix(candidate, "w/")
		if candidate == etag {
			return true
		}
	}

	return false
}

func fileETag(info os.FileInfo) string {
	return fmt.Sprintf(`"%x-%x"`, info.Size(), info.ModTime().UnixNano())
}

func bytesETag(data []byte) string {
	sum := sha256.Sum256(data)
	return `"` + hex.EncodeToString(sum[:16]) + `"`
}
