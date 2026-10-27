package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(requestIDMiddleware())
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, requestID(c))
	})

	generated := performRequest(t, router, "/", nil)
	generatedID := generated.Header().Get(requestIDHeader)
	if !validRequestID(generatedID) {
		t.Fatalf("generated request ID %q is invalid", generatedID)
	}
	if generated.Body.String() != generatedID {
		t.Fatalf("handler request ID = %q, response header = %q", generated.Body.String(), generatedID)
	}

	provided := performRequest(t, router, "/", map[string]string{requestIDHeader: "req-123.test"})
	if got := provided.Header().Get(requestIDHeader); got != "req-123.test" {
		t.Fatalf("provided request ID = %q, want req-123.test", got)
	}

	invalid := performRequest(t, router, "/", map[string]string{requestIDHeader: "bad id"})
	if got := invalid.Header().Get(requestIDHeader); !validRequestID(got) || got == "bad id" {
		t.Fatalf("invalid request ID was not replaced: %q", got)
	}
}

func TestLimitRequestSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(limitRequestSize())
	router.NoRoute(func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	tooLongURI := "/" + strings.Repeat("a", maxRequestURIBytes)
	uriResponse := performRequest(t, router, tooLongURI, nil)
	if uriResponse.Code != http.StatusRequestURITooLong {
		t.Fatalf("long URI status = %d, want %d", uriResponse.Code, http.StatusRequestURITooLong)
	}
	if got := uriResponse.Header().Get("Cache-Control"); got != noStoreCacheControl {
		t.Fatalf("long URI Cache-Control = %q, want %q", got, noStoreCacheControl)
	}

	apiResponse := performRequest(t, router, "/api/"+strings.Repeat("a", maxRequestURIBytes), nil)
	if apiResponse.Code != http.StatusRequestURITooLong {
		t.Fatalf("API long URI status = %d, want %d", apiResponse.Code, http.StatusRequestURITooLong)
	}
	if got := apiResponse.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("API long URI Content-Type = %q, want application/json", got)
	}

	body := strings.Repeat("a", maxRequestBodyBytes+1)
	bodyRequest := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	bodyResponse := httptest.NewRecorder()
	router.ServeHTTP(bodyResponse, bodyRequest)
	if bodyResponse.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("large body status = %d, want %d", bodyResponse.Code, http.StatusRequestEntityTooLarge)
	}
	if got := bodyResponse.Header().Get("Cache-Control"); got != noStoreCacheControl {
		t.Fatalf("large body Cache-Control = %q, want %q", got, noStoreCacheControl)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(recoveryMiddleware())
	router.GET("/panic", func(_ *gin.Context) {
		panic("secret detail")
	})
	router.GET("/api/panic", func(_ *gin.Context) {
		panic("secret detail")
	})

	page := performRequest(t, router, "/panic", nil)
	if page.Code != http.StatusInternalServerError {
		t.Fatalf("page panic status = %d, want %d", page.Code, http.StatusInternalServerError)
	}
	if page.Body.String() != "internal server error" {
		t.Fatalf("page panic body = %q, want generic error", page.Body.String())
	}
	if strings.Contains(page.Body.String(), "secret detail") {
		t.Fatal("page panic body leaked the panic value")
	}

	api := performRequest(t, router, "/api/panic", nil)
	if api.Code != http.StatusInternalServerError {
		t.Fatalf("API panic status = %d, want %d", api.Code, http.StatusInternalServerError)
	}
	if got := api.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("API panic Content-Type = %q, want application/json", got)
	}
	if strings.Contains(api.Body.String(), "secret detail") {
		t.Fatal("API panic body leaked the panic value")
	}
}

func TestMediaErrorsDoNotLeakPaths(t *testing.T) {
	library := loadTestLibrary(t)
	router, err := NewRouter(library, Options{AppEnv: "test"})
	if err != nil {
		t.Fatal(err)
	}

	series := library.CatalogResponse().Series
	entry, ok := library.Entry(series[0].ID)
	if !ok {
		t.Fatal("entry was not registered")
	}
	episode := entry.Episodes[0]

	if err := os.Remove(episode.VideoPath); err != nil {
		t.Fatal(err)
	}
	video := serveRouter(t, router, "/video/"+episode.ID, nil)
	if video.Code != http.StatusInternalServerError {
		t.Fatalf("video status = %d, want %d", video.Code, http.StatusInternalServerError)
	}
	if video.Body.String() != "video unavailable" {
		t.Fatalf("video body = %q, want generic error", video.Body.String())
	}
	if strings.Contains(video.Body.String(), episode.VideoPath) {
		t.Fatal("video body leaked the filesystem path")
	}

	if err := os.Remove(episode.ThumbnailPath); err != nil {
		t.Fatal(err)
	}
	thumbnail := serveRouter(t, router, "/thumb/"+episode.ID, nil)
	if thumbnail.Code != http.StatusInternalServerError {
		t.Fatalf("thumbnail status = %d, want %d", thumbnail.Code, http.StatusInternalServerError)
	}
	if thumbnail.Body.String() != "thumbnail unavailable" {
		t.Fatalf("thumbnail body = %q, want generic error", thumbnail.Body.String())
	}
	if strings.Contains(thumbnail.Body.String(), episode.ThumbnailPath) {
		t.Fatal("thumbnail body leaked the filesystem path")
	}
}

func performRequest(t *testing.T, handler http.Handler, target string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, target, nil)
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
