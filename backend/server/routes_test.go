package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestResolveRouteStatus(t *testing.T) {
	library := loadTestLibrary(t)

	cases := []struct {
		path string
		want int
	}{
		{path: "/", want: http.StatusOK},
		{path: "/series", want: http.StatusOK},
		{path: "/shorts", want: http.StatusOK},
		{path: "/shots", want: http.StatusOK},
		{path: "/songs", want: http.StatusOK},
		{path: "/creators", want: http.StatusOK},
		{path: "/series/", want: http.StatusOK},
		{path: "/creator", want: http.StatusOK},
		{path: "/creator/show", want: http.StatusOK},
		{path: "/creator/show/episode-1-start", want: http.StatusOK},
		{path: "/assets/app-123.js", want: http.StatusOK},
		{path: "/favicon.ico", want: http.StatusOK},
		{path: "/missing", want: http.StatusNotFound},
		{path: "/creator/missing", want: http.StatusNotFound},
		{path: "/creator/show/missing", want: http.StatusNotFound},
		{path: "/creator/show/episode-1-start/extra", want: http.StatusNotFound},
	}

	for _, testCase := range cases {
		t.Run(testCase.path, func(t *testing.T) {
			if got := resolveRouteStatus(library, testCase.path); got != testCase.want {
				t.Fatalf("resolveRouteStatus(%q) = %d, want %d", testCase.path, got, testCase.want)
			}
		})
	}
}

func TestRouterReturnsRealStatuses(t *testing.T) {
	library := loadTestLibrary(t)
	router, err := NewRouter(library, Options{AppEnv: "test"})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		path        string
		want        int
		contentType string
	}{
		{path: "/", want: http.StatusOK, contentType: "text/html"},
		{path: "/series", want: http.StatusOK, contentType: "text/html"},
		{path: "/creators", want: http.StatusOK, contentType: "text/html"},
		{path: "/creator", want: http.StatusOK, contentType: "text/html"},
		{path: "/creator/show", want: http.StatusOK, contentType: "text/html"},
		{path: "/creator/show/episode-1-start", want: http.StatusOK, contentType: "text/html"},
		{path: "/missing", want: http.StatusNotFound, contentType: "text/html"},
		{path: "/creator/missing", want: http.StatusNotFound, contentType: "text/html"},
		{path: "/creator/show/missing", want: http.StatusNotFound, contentType: "text/html"},
	}

	for _, testCase := range cases {
		t.Run(testCase.path, func(t *testing.T) {
			response := serveRouter(t, router, testCase.path, nil)
			if response.Code != testCase.want {
				t.Fatalf("status = %d, want %d", response.Code, testCase.want)
			}
			if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, testCase.contentType) {
				t.Fatalf("Content-Type = %q, want prefix %q", got, testCase.contentType)
			}
			if !strings.Contains(response.Body.String(), `id="app"`) {
				t.Fatalf("body does not contain the SPA root: %q", response.Body.String())
			}

			if testCase.want == http.StatusNotFound {
				if got := response.Header().Get("Cache-Control"); got != noStoreCacheControl {
					t.Fatalf("404 Cache-Control = %q, want %q", got, noStoreCacheControl)
				}
				return
			}

			if got := response.Header().Get("Cache-Control"); got != indexCacheControl {
				t.Fatalf("200 Cache-Control = %q, want %q", got, indexCacheControl)
			}
			if response.Header().Get("ETag") == "" {
				t.Fatal("200 ETag is empty")
			}
		})
	}
}
