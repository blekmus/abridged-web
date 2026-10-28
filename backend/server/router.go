package server

import (
	"bytes"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"abridged-web/backend/catalog"
	"abridged-web/frontend"
	"github.com/gin-gonic/gin"
)

type Options struct {
	AppEnv        string
	ServeFromDisk bool
	PublicBaseURL string
}

type appServer struct {
	library *catalog.Library
	search  searchCache
	baseURL string
}

func NewRouter(library *catalog.Library, options Options) (*gin.Engine, error) {
	if options.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	baseURL, err := normalizePublicBaseURL(options.PublicBaseURL)
	if err != nil {
		return nil, err
	}

	server := &appServer{library: library, baseURL: baseURL}
	router := gin.New()
	router.Use(securityHeaders(), requestIDMiddleware(), requestLogger(), recoveryMiddleware(), limitRequestSize())

	router.GET("/healthz", server.handleHealth)
	router.GET("/robots.txt", server.handleRobots)
	router.GET("/sitemap.xml", server.handleSitemap)
	router.GET("/index.html", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/")
	})
	router.GET("/api/catalog", server.handleCatalog)
	router.GET("/api/series", server.handleSeries)
	router.GET("/api/shorts", server.handleShorts)
	router.GET("/api/shots", server.handleShots)
	router.GET("/api/songs", server.handleSongs)
	router.GET("/api/creator/:slug", server.handleCreator)
	router.GET("/api/entry/:entryID", server.handleEntry)
	router.GET("/api/search", server.handleSearch)
	router.GET("/video/:assetID", server.handleVideo)
	router.GET("/thumb/:assetID", server.handleThumb)

	appHandler, err := newSPAHandler(options.ServeFromDisk)
	if err != nil {
		return nil, err
	}
	appHandler.metaBuilder = server.pageMetadata

	router.NoRoute(func(c *gin.Context) {
		if c.Request.URL.Path == "/api" || strings.HasPrefix(c.Request.URL.Path, "/api/") {
			setNoStore(c)
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		appHandler.serveHTTP(
			c.Writer,
			c.Request,
			resolveRouteStatus(server.library, c.Request.URL.Path),
		)
	})

	return router, nil
}

func (s *appServer) handleHealth(c *gin.Context) {
	setNoStore(c)
	c.Status(http.StatusNoContent)
}

func (s *appServer) handleCatalog(c *gin.Context) {
	writeJSONCached(c, s.library.CacheTag(), http.StatusOK, s.library.CatalogResponse())
}

func (s *appServer) handleSeries(c *gin.Context) {
	writeJSONCached(c, s.library.CacheTag(), http.StatusOK, s.library.EntriesForType(catalog.EntryTypeSeries))
}

func (s *appServer) handleShorts(c *gin.Context) {
	writeJSONCached(c, s.library.CacheTag(), http.StatusOK, s.library.EntriesForType(catalog.EntryTypeShort))
}

func (s *appServer) handleShots(c *gin.Context) {
	writeJSONCached(c, s.library.CacheTag(), http.StatusOK, s.library.EntriesForType(catalog.EntryTypeShot))
}

func (s *appServer) handleSongs(c *gin.Context) {
	writeJSONCached(c, s.library.CacheTag(), http.StatusOK, s.library.CatalogResponse().Songs)
}

func (s *appServer) handleCreator(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		setNoStore(c)
		c.Status(http.StatusNotFound)
		return
	}

	entries := s.library.EntriesForCreator(slug)
	if len(entries) == 0 {
		setNoStore(c)
		c.Status(http.StatusNotFound)
		return
	}

	writeJSONCached(c, s.library.CacheTag(), http.StatusOK, entries)
}

func (s *appServer) handleEntry(c *gin.Context) {
	entryID := c.Param("entryID")
	if entryID == "" {
		setNoStore(c)
		c.Status(http.StatusNotFound)
		return
	}

	entry, ok := s.library.Entry(entryID)
	if !ok {
		setNoStore(c)
		c.Status(http.StatusNotFound)
		return
	}

	writeJSONCached(c, s.library.CacheTag(), http.StatusOK, entry)
}

func (s *appServer) handleSearch(c *gin.Context) {
	if !isSameOriginSearchRequest(c.Request) {
		setNoStore(c)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	query := strings.TrimSpace(c.Query("q"))
	if utf8.RuneCountInString(query) > maxSearchQueryLength ||
		len(strings.Fields(query)) > maxSearchTerms {
		setNoStore(c)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	key := normalizeSearchKey(query)
	results, ok := s.search.get(key)
	if !ok {
		results = s.library.Search(query, 0)
		s.search.set(key, results)
	}

	c.Header("Cache-Control", "private, max-age=60")
	c.JSON(http.StatusOK, results)
}

func (s *appServer) handleVideo(c *gin.Context) {
	assetID := c.Param("assetID")
	asset, ok := s.library.EpisodeAsset(assetID)
	if !ok || asset.VideoPath == "" {
		setNoStore(c)
		c.Status(http.StatusNotFound)
		return
	}

	file, err := os.Open(asset.VideoPath)
	if err != nil {
		slog.Error("open video failed", "request_id", requestID(c), "asset_id", assetID, "err", err)
		abortRequest(c, http.StatusInternalServerError, "video unavailable")
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		slog.Error("stat video failed", "request_id", requestID(c), "asset_id", assetID, "err", err)
		abortRequest(c, http.StatusInternalServerError, "video unavailable")
		return
	}

	if contentType := mime.TypeByExtension(filepath.Ext(asset.VideoPath)); contentType != "" {
		c.Header("Content-Type", contentType)
	}
	c.Header("Cache-Control", videoCacheControl)
	c.Header("Etag", fileETag(info))
	http.ServeContent(c.Writer, c.Request, filepath.Base(asset.VideoPath), info.ModTime(), file)
}

func (s *appServer) handleThumb(c *gin.Context) {
	assetID := c.Param("assetID")
	asset, ok := s.library.EpisodeAsset(assetID)
	if !ok || asset.ThumbnailPath == "" {
		setNoStore(c)
		c.Status(http.StatusNotFound)
		return
	}

	file, err := os.Open(asset.ThumbnailPath)
	if err != nil {
		slog.Error("open thumbnail failed", "request_id", requestID(c), "asset_id", assetID, "err", err)
		abortRequest(c, http.StatusInternalServerError, "thumbnail unavailable")
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		slog.Error("stat thumbnail failed", "request_id", requestID(c), "asset_id", assetID, "err", err)
		abortRequest(c, http.StatusInternalServerError, "thumbnail unavailable")
		return
	}

	contentType := mime.TypeByExtension(filepath.Ext(asset.ThumbnailPath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.Header("Cache-Control", thumbnailCacheControl)
	c.Header("Content-Type", contentType)
	c.Header("Etag", fileETag(info))
	http.ServeContent(c.Writer, c.Request, filepath.Base(asset.ThumbnailPath), info.ModTime(), file)
}

type spaHandler struct {
	staticFS      fs.FS
	handler       http.Handler
	etags         map[string]string
	serveFromDisk bool
	metaBuilder   func(*http.Request, int) pageMeta
}

func newSPAHandler(serveFromDisk bool) (*spaHandler, error) {
	var assets fs.FS
	if serveFromDisk {
		assets = os.DirFS("frontend/dist")
	} else {
		sub, err := fs.Sub(frontendassets.DistFS, "dist")
		if err != nil {
			return nil, err
		}
		assets = sub
	}

	etags, err := buildAssetETags(assets)
	if err != nil {
		return nil, err
	}

	return &spaHandler{
		staticFS:      assets,
		handler:       http.FileServer(http.FS(assets)),
		etags:         etags,
		serveFromDisk: serveFromDisk,
	}, nil
}

func buildAssetETags(assets fs.FS) (map[string]string, error) {
	etags := make(map[string]string)
	err := fs.WalkDir(assets, ".", func(assetPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		data, err := fs.ReadFile(assets, assetPath)
		if err != nil {
			return err
		}
		etags[assetPath] = bytesETag(data)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return etags, nil
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.serveHTTP(w, r, http.StatusOK)
}

func (h *spaHandler) serveHTTP(w http.ResponseWriter, r *http.Request, status int) {
	cleanPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if cleanPath == "." || cleanPath == "" || cleanPath == "index.html" {
		h.serveIndex(w, r, status)
		return
	}

	file, err := h.staticFS.Open(cleanPath)
	if err == nil {
		defer file.Close()
		info, statErr := file.Stat()
		if statErr == nil && !info.IsDir() {
			h.setFileCacheHeaders(w, cleanPath)
			h.handler.ServeHTTP(w, r)
			return
		}
	}

	// A missing file with an extension is a missing static asset, not an SPA
	// route. Falling back to index.html here would serve HTML as a favicon or
	// script and cache a 200 for a path that does not exist.
	if strings.HasPrefix(cleanPath, "assets/") || path.Ext(cleanPath) != "" {
		w.Header().Set("Cache-Control", noStoreCacheControl)
		http.NotFound(w, r)
		return
	}

	h.serveIndex(w, r, status)
}

func (h *spaHandler) setFileCacheHeaders(w http.ResponseWriter, assetPath string) {
	if strings.HasPrefix(assetPath, "assets/") {
		w.Header().Set("Cache-Control", immutableAssetCacheControl)
		return
	}

	if h.serveFromDisk {
		w.Header().Set("Cache-Control", indexCacheControl)
		return
	}

	w.Header().Set("Cache-Control", staticAssetCacheControl)
	if etag, ok := h.etags[assetPath]; ok {
		w.Header().Set("Etag", etag)
	}
}

func (h *spaHandler) serveIndex(w http.ResponseWriter, r *http.Request, status int) {
	data, err := fs.ReadFile(h.staticFS, "index.html")
	if err != nil {
		slog.Error("read index failed", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	meta := defaultPageMeta()
	if h.metaBuilder != nil {
		meta = h.metaBuilder(r, status)
	}

	shell, err := appShellForPath(h.staticFS, r.URL.Path)
	if err != nil {
		slog.Error("read app shell failed", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	indexHTML := strings.Replace(string(data), "<!--app-shell-->", shell, 1)
	body := []byte(injectHead(indexHTML, renderMeta(meta)))

	if status != http.StatusOK {
		w.Header().Set("Cache-Control", noStoreCacheControl)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		_, _ = w.Write(body)
		return
	}

	w.Header().Set("Cache-Control", indexCacheControl)
	w.Header().Set("Etag", bytesETag(body))
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(body))
}
