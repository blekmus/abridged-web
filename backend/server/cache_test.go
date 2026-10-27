package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"abridged-web/backend/catalog"
	"github.com/gin-gonic/gin"
)

func TestIfNoneMatch(t *testing.T) {
	const tag = `"abc123"`

	cases := []struct {
		name   string
		header string
		tag    string
		want   bool
	}{
		{name: "empty", header: "", tag: tag, want: false},
		{name: "exact", header: tag, tag: tag, want: true},
		{name: "weak", header: `W/"abc123"`, tag: tag, want: true},
		{name: "lowercase weak", header: `w/"abc123"`, tag: tag, want: true},
		{name: "list", header: `"other", W/"abc123"`, tag: tag, want: true},
		{name: "wildcard", header: "*", tag: tag, want: true},
		{name: "mismatch", header: `"other"`, tag: tag, want: false},
		{name: "malformed", header: `abc123`, tag: tag, want: false},
		{name: "empty tag", header: tag, tag: "", want: false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := ifNoneMatch(testCase.header, testCase.tag); got != testCase.want {
				t.Fatalf("ifNoneMatch(%q, %q) = %v, want %v", testCase.header, testCase.tag, got, testCase.want)
			}
		})
	}
}

func TestWriteJSONCached(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const tag = `"abc123"`

	first := performCachedJSON(t, nil, tag)
	if first.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", first.Code, http.StatusOK)
	}
	if got := first.Header().Get("ETag"); got != tag {
		t.Fatalf("ETag = %q, want %q", got, tag)
	}
	if got := first.Header().Get("Cache-Control"); got != jsonCacheControl {
		t.Fatalf("Cache-Control = %q, want %q", got, jsonCacheControl)
	}
	if first.Body.Len() == 0 {
		t.Fatal("response body is empty")
	}

	second := performCachedJSON(t, map[string]string{"If-None-Match": `W/"abc123"`}, tag)
	if second.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want %d", second.Code, http.StatusNotModified)
	}
	if second.Body.Len() != 0 {
		t.Fatalf("304 body = %q, want empty", second.Body.String())
	}
	if got := second.Header().Get("Cache-Control"); got != jsonCacheControl {
		t.Fatalf("304 Cache-Control = %q, want %q", got, jsonCacheControl)
	}

	third := performCachedJSON(t, map[string]string{"If-None-Match": `"other", W/"abc123"`}, tag)
	if third.Code != http.StatusNotModified {
		t.Fatalf("list status = %d, want %d", third.Code, http.StatusNotModified)
	}

	fourth := performCachedJSON(t, map[string]string{"If-None-Match": `"other"`}, tag)
	if fourth.Code != http.StatusOK {
		t.Fatalf("mismatch status = %d, want %d", fourth.Code, http.StatusOK)
	}
}

func TestSPAHandlerCacheHeaders(t *testing.T) {
	assets := fstest.MapFS{
		"index.html": {
			Data: []byte(`<html><body><div id="app"><!--app-shell--></div></body></html>`),
		},
		"assets/app-123.js": {
			Data: []byte(`console.log("app")`),
		},
		"images/icon.jpg": {
			Data: []byte("icon"),
		},
	}

	etags, err := buildAssetETags(assets)
	if err != nil {
		t.Fatal(err)
	}
	handler := &spaHandler{
		staticFS: assets,
		handler:  http.FileServer(http.FS(assets)),
		etags:    etags,
	}

	asset := performSPARequest(t, handler, "/assets/app-123.js", nil)
	if asset.Code != http.StatusOK {
		t.Fatalf("asset status = %d, want %d", asset.Code, http.StatusOK)
	}
	if got := asset.Header().Get("Cache-Control"); got != immutableAssetCacheControl {
		t.Fatalf("asset Cache-Control = %q, want %q", got, immutableAssetCacheControl)
	}

	image := performSPARequest(t, handler, "/images/icon.jpg", nil)
	if image.Code != http.StatusOK {
		t.Fatalf("image status = %d, want %d", image.Code, http.StatusOK)
	}
	if got := image.Header().Get("Cache-Control"); got != staticAssetCacheControl {
		t.Fatalf("image Cache-Control = %q, want %q", got, staticAssetCacheControl)
	}
	if image.Header().Get("ETag") == "" {
		t.Fatal("image ETag is empty")
	}

	index := performSPARequest(t, handler, "/", nil)
	if index.Code != http.StatusOK {
		t.Fatalf("index status = %d, want %d", index.Code, http.StatusOK)
	}
	if got := index.Header().Get("Cache-Control"); got != indexCacheControl {
		t.Fatalf("index Cache-Control = %q, want %q", got, indexCacheControl)
	}
	indexTag := index.Header().Get("ETag")
	if indexTag == "" {
		t.Fatal("index ETag is empty")
	}
	if !strings.Contains(index.Body.String(), "page-shell") {
		t.Fatal("index body does not contain the injected shell")
	}

	notModified := performSPARequest(t, handler, "/", map[string]string{"If-None-Match": `W/` + indexTag})
	if notModified.Code != http.StatusNotModified {
		t.Fatalf("index conditional status = %d, want %d", notModified.Code, http.StatusNotModified)
	}
	if notModified.Body.Len() != 0 {
		t.Fatalf("index 304 body = %q, want empty", notModified.Body.String())
	}

	directIndex := performSPARequest(t, handler, "/index.html", nil)
	if got := directIndex.Header().Get("Cache-Control"); got != indexCacheControl {
		t.Fatalf("direct index Cache-Control = %q, want %q", got, indexCacheControl)
	}

	missing := performSPARequest(t, handler, "/assets/missing.js", nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing asset status = %d, want %d", missing.Code, http.StatusNotFound)
	}
	if got := missing.Header().Get("Cache-Control"); got != noStoreCacheControl {
		t.Fatalf("missing asset Cache-Control = %q, want %q", got, noStoreCacheControl)
	}

	missingIcon := performSPARequest(t, handler, "/favicon.ico", nil)
	if missingIcon.Code != http.StatusNotFound {
		t.Fatalf("missing icon status = %d, want %d", missingIcon.Code, http.StatusNotFound)
	}
	if got := missingIcon.Header().Get("Cache-Control"); got != noStoreCacheControl {
		t.Fatalf("missing icon Cache-Control = %q, want %q", got, noStoreCacheControl)
	}
}

func TestAPIAndMediaCacheHeaders(t *testing.T) {
	library := loadTestLibrary(t)
	router, err := NewRouter(library, Options{AppEnv: "test"})
	if err != nil {
		t.Fatal(err)
	}

	catalogResponse := serveRouter(t, router, "/api/catalog", nil)
	if catalogResponse.Code != http.StatusOK {
		t.Fatalf("catalog status = %d, want %d", catalogResponse.Code, http.StatusOK)
	}
	catalogTag := catalogResponse.Header().Get("ETag")
	if catalogTag == "" {
		t.Fatal("catalog ETag is empty")
	}
	if got := catalogResponse.Header().Get("Cache-Control"); got != jsonCacheControl {
		t.Fatalf("catalog Cache-Control = %q, want %q", got, jsonCacheControl)
	}

	notModified := serveRouter(t, router, "/api/catalog", map[string]string{"If-None-Match": `W/` + catalogTag})
	if notModified.Code != http.StatusNotModified {
		t.Fatalf("catalog conditional status = %d, want %d", notModified.Code, http.StatusNotModified)
	}

	series := library.CatalogResponse().Series
	if len(series) != 1 {
		t.Fatalf("len(Series) = %d, want 1", len(series))
	}
	entry, ok := library.Entry(series[0].ID)
	if !ok {
		t.Fatal("entry was not registered")
	}
	episodeID := entry.Episodes[0].ID

	video := serveRouter(t, router, "/video/"+episodeID, nil)
	if video.Code != http.StatusOK {
		t.Fatalf("video status = %d, want %d", video.Code, http.StatusOK)
	}
	if got := video.Header().Get("Cache-Control"); got != videoCacheControl {
		t.Fatalf("video Cache-Control = %q, want %q", got, videoCacheControl)
	}
	videoTag := video.Header().Get("ETag")
	if videoTag == "" {
		t.Fatal("video ETag is empty")
	}
	if got := video.Header().Get("Accept-Ranges"); got != "bytes" {
		t.Fatalf("video Accept-Ranges = %q, want bytes", got)
	}

	videoNotModified := serveRouter(t, router, "/video/"+episodeID, map[string]string{"If-None-Match": videoTag})
	if videoNotModified.Code != http.StatusNotModified {
		t.Fatalf("video conditional status = %d, want %d", videoNotModified.Code, http.StatusNotModified)
	}

	rangeResponse := serveRouter(t, router, "/video/"+episodeID, map[string]string{"Range": "bytes=0-3"})
	if rangeResponse.Code != http.StatusPartialContent {
		t.Fatalf("video range status = %d, want %d", rangeResponse.Code, http.StatusPartialContent)
	}
	if rangeResponse.Header().Get("Content-Range") == "" {
		t.Fatal("video range Content-Range is empty")
	}

	thumbnail := serveRouter(t, router, "/thumb/"+episodeID, nil)
	if thumbnail.Code != http.StatusOK {
		t.Fatalf("thumbnail status = %d, want %d", thumbnail.Code, http.StatusOK)
	}
	thumbnailCacheControlHeader := thumbnail.Header().Get("Cache-Control")
	if thumbnailCacheControlHeader != thumbnailCacheControl {
		t.Fatalf("thumbnail Cache-Control = %q, want %q", thumbnailCacheControlHeader, thumbnailCacheControl)
	}
	if strings.Contains(thumbnailCacheControlHeader, "immutable") {
		t.Fatalf("thumbnail Cache-Control = %q, must not be immutable", thumbnailCacheControlHeader)
	}
	if thumbnail.Header().Get("ETag") == "" {
		t.Fatal("thumbnail ETag is empty")
	}
}

func TestUnknownAPIPathReturnsNotFound(t *testing.T) {
	library := loadTestLibrary(t)
	router, err := NewRouter(library, Options{AppEnv: "test"})
	if err != nil {
		t.Fatal(err)
	}

	response := serveRouter(t, router, "/api/missing", nil)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if got := response.Header().Get("Cache-Control"); got != noStoreCacheControl {
		t.Fatalf("Cache-Control = %q, want %q", got, noStoreCacheControl)
	}
	if !strings.Contains(response.Body.String(), "not found") {
		t.Fatalf("body = %q, want JSON error", response.Body.String())
	}
}

func performCachedJSON(t *testing.T, headers map[string]string, tag string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/test", nil)
	for key, value := range headers {
		context.Request.Header.Set(key, value)
	}
	writeJSONCached(context, tag, http.StatusOK, map[string]string{"status": "ok"})
	return response
}

func performSPARequest(t *testing.T, handler http.Handler, target string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, target, nil)
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func serveRouter(t *testing.T, handler http.Handler, target string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	return performSPARequest(t, handler, target, headers)
}

func loadTestLibrary(t *testing.T) *catalog.Library {
	t.Helper()

	root := t.TempDir()
	entryPath := filepath.Join(root, "Series", "[Creator] Show")
	if err := os.MkdirAll(entryPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(entryPath, "Episode 1 - Start.mp4"), []byte("video-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(entryPath, "Episode 1 - Start.webp"), []byte("thumb-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(entryPath, "Episode 2 - Next.mp4"), []byte("video-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(entryPath, "Episode 2 - Next.webp"), []byte("thumb-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	metadata, err := json.Marshal(map[string]any{
		"version": 2,
		"items":   map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".abridged-video-metadata.json"), metadata, 0o644); err != nil {
		t.Fatal(err)
	}

	library, err := catalog.Load(root)
	if err != nil {
		t.Fatal(err)
	}

	return library
}
