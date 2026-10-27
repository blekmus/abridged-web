package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"abridged-web/backend/catalog"
	"github.com/gin-gonic/gin"
)

func TestIsSameOriginSearchRequest(t *testing.T) {
	cases := []struct {
		name    string
		headers map[string]string
		want    bool
	}{
		{name: "same origin", headers: map[string]string{"Sec-Fetch-Site": "same-origin"}, want: true},
		{name: "same site", headers: map[string]string{"Sec-Fetch-Site": "same-site"}, want: true},
		{name: "cross site", headers: map[string]string{"Sec-Fetch-Site": "cross-site"}, want: false},
		{name: "direct navigation", headers: map[string]string{"Sec-Fetch-Site": "none"}, want: false},
		{name: "matching origin", headers: map[string]string{"Origin": "https://example.test"}, want: true},
		{name: "mismatched origin", headers: map[string]string{"Origin": "https://evil.test"}, want: false},
		{name: "matching referer", headers: map[string]string{"Referer": "https://example.test/series"}, want: true},
		{name: "no headers", headers: nil, want: false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/search?q=test", nil)
			request.Host = "example.test"
			for key, value := range testCase.headers {
				request.Header.Set(key, value)
			}
			if got := isSameOriginSearchRequest(request); got != testCase.want {
				t.Fatalf("isSameOriginSearchRequest() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestHandleSearchRejectsDirectAndOversizedRequests(t *testing.T) {
	server := &appServer{library: &catalog.Library{}}

	if response := performSearch(server, "/api/search?q=test", nil); response.Code != http.StatusForbidden {
		t.Fatalf("direct status = %d, want %d", response.Code, http.StatusForbidden)
	}

	sameOrigin := map[string]string{"Sec-Fetch-Site": "same-origin"}

	longQuery := "/api/search?q=" + strings.Repeat("a", maxSearchQueryLength+1)
	if response := performSearch(server, longQuery, sameOrigin); response.Code != http.StatusBadRequest {
		t.Fatalf("long query status = %d, want %d", response.Code, http.StatusBadRequest)
	}

	manyTerms := "/api/search?q=" + strings.Repeat("term+", maxSearchTerms) + "extra"
	if response := performSearch(server, manyTerms, sameOrigin); response.Code != http.StatusBadRequest {
		t.Fatalf("many terms status = %d, want %d", response.Code, http.StatusBadRequest)
	}

	response := performSearch(server, "/api/search?q=test", sameOrigin)
	if response.Code != http.StatusOK {
		t.Fatalf("valid status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Header().Get("Cache-Control") == "" {
		t.Fatal("Cache-Control header was not set")
	}
}

func TestHandleSearchCachesNormalizedQueries(t *testing.T) {
	server := &appServer{library: &catalog.Library{}}
	sameOrigin := map[string]string{"Sec-Fetch-Site": "same-origin"}

	performSearch(server, "/api/search?q=Team+Four", sameOrigin)
	performSearch(server, "/api/search?q=team+++four", sameOrigin)

	if got := len(server.search.items); got != 1 {
		t.Fatalf("cache items = %d, want 1", got)
	}
	if _, ok := server.search.items["team four"]; !ok {
		t.Fatal("cache key was not normalized")
	}
}

func performSearch(server *appServer, target string, headers map[string]string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodGet, target, nil)
	for key, value := range headers {
		context.Request.Header.Set(key, value)
	}
	server.handleSearch(context)
	return response
}
