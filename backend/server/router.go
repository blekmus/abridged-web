package server

import (
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"abridged-web/backend/catalog"
	"abridged-web/frontend"
	"github.com/gin-gonic/gin"
)

type Options struct {
	AppEnv        string
	ServeFromDisk bool
}

type appServer struct {
	library *catalog.Library
}

func NewRouter(library *catalog.Library, options Options) (*gin.Engine, error) {
	if options.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	server := &appServer{library: library}
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	router.GET("/api/catalog", server.handleCatalog)
	router.GET("/api/series", server.handleSeries)
	router.GET("/api/shorts", server.handleShorts)
	router.GET("/api/shots", server.handleShots)
	router.GET("/api/songs", server.handleSongs)
	router.GET("/api/creator/:slug", server.handleCreator)
	router.GET("/api/entry/:entryID", server.handleEntry)
	router.GET("/video/:assetID", server.handleVideo)
	router.GET("/thumb/:assetID", server.handleThumb)

	appHandler, err := newSPAHandler(options.ServeFromDisk)
	if err != nil {
		return nil, err
	}
	router.NoRoute(gin.WrapH(appHandler))

	return router, nil
}

func (s *appServer) handleCatalog(c *gin.Context) {
	c.JSON(http.StatusOK, s.library.CatalogResponse())
}

func (s *appServer) handleSeries(c *gin.Context) {
	c.JSON(http.StatusOK, s.library.EntriesForType(catalog.EntryTypeSeries))
}

func (s *appServer) handleShorts(c *gin.Context) {
	c.JSON(http.StatusOK, s.library.EntriesForType(catalog.EntryTypeShort))
}

func (s *appServer) handleShots(c *gin.Context) {
	c.JSON(http.StatusOK, s.library.EntriesForType(catalog.EntryTypeShot))
}

func (s *appServer) handleSongs(c *gin.Context) {
	c.JSON(http.StatusOK, s.library.CatalogResponse().Songs)
}

func (s *appServer) handleCreator(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.Status(http.StatusNotFound)
		return
	}

	entries := s.library.EntriesForCreator(slug)
	if len(entries) == 0 {
		c.Status(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, entries)
}

func (s *appServer) handleEntry(c *gin.Context) {
	entryID := c.Param("entryID")
	if entryID == "" {
		c.Status(http.StatusNotFound)
		return
	}

	entry, ok := s.library.Entry(entryID)
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, entry)
}

func (s *appServer) handleVideo(c *gin.Context) {
	assetID := c.Param("assetID")
	asset, ok := s.library.EpisodeAsset(assetID)
	if !ok || asset.VideoPath == "" {
		c.Status(http.StatusNotFound)
		return
	}

	file, err := os.Open(asset.VideoPath)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	if contentType := mime.TypeByExtension(filepath.Ext(asset.VideoPath)); contentType != "" {
		c.Header("Content-Type", contentType)
	}
	http.ServeContent(c.Writer, c.Request, filepath.Base(asset.VideoPath), info.ModTime(), file)
}

func (s *appServer) handleThumb(c *gin.Context) {
	assetID := c.Param("assetID")
	asset, ok := s.library.EpisodeAsset(assetID)
	if !ok || asset.ThumbnailPath == "" {
		c.Status(http.StatusNotFound)
		return
	}

	file, err := os.Open(asset.ThumbnailPath)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	contentType := mime.TypeByExtension(filepath.Ext(asset.ThumbnailPath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("Content-Type", contentType)
	http.ServeContent(c.Writer, c.Request, filepath.Base(asset.ThumbnailPath), info.ModTime(), file)
}

type spaHandler struct {
	staticFS fs.FS
	handler  http.Handler
}

func newSPAHandler(serveFromDisk bool) (http.Handler, error) {
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

	return &spaHandler{
		staticFS: assets,
		handler:  http.FileServer(http.FS(assets)),
	}, nil
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cleanPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if cleanPath == "." || cleanPath == "" {
		h.serveIndex(w, r)
		return
	}

	file, err := h.staticFS.Open(cleanPath)
	if err == nil {
		defer file.Close()
		info, statErr := file.Stat()
		if statErr == nil && !info.IsDir() {
			h.handler.ServeHTTP(w, r)
			return
		}
	}

	h.serveIndex(w, r)
}

func (h *spaHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	data, err := fs.ReadFile(h.staticFS, "index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}
