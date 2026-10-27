package server

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRobotsTxt(t *testing.T) {
	library := loadTestLibrary(t)
	router, err := NewRouter(library, Options{
		AppEnv:        "test",
		PublicBaseURL: "https://abridged.example/",
	})
	if err != nil {
		t.Fatal(err)
	}

	response := serveRouter(t, router, "/robots.txt", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Fatalf("Content-Type = %q, want text/plain", got)
	}
	if got := response.Header().Get("Cache-Control"); got != robotsCacheControl {
		t.Fatalf("Cache-Control = %q, want %q", got, robotsCacheControl)
	}
	body := response.Body.String()
	for _, want := range []string{
		"User-agent: *",
		"Allow: /",
		"Disallow: /api/",
		"Disallow: /video/",
		"Sitemap: https://abridged.example/sitemap.xml",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("robots.txt missing %q:\n%s", want, body)
		}
	}

	etag := response.Header().Get("ETag")
	if etag == "" {
		t.Fatal("ETag is empty")
	}
	notModified := serveRouter(t, router, "/robots.txt", map[string]string{"If-None-Match": `W/` + etag})
	if notModified.Code != http.StatusNotModified {
		t.Fatalf("conditional status = %d, want %d", notModified.Code, http.StatusNotModified)
	}
}

func TestSitemapXML(t *testing.T) {
	library := loadTestLibrary(t)
	router, err := NewRouter(library, Options{
		AppEnv:        "test",
		PublicBaseURL: "https://abridged.example",
	})
	if err != nil {
		t.Fatal(err)
	}

	response := serveRouter(t, router, "/sitemap.xml", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/xml") {
		t.Fatalf("Content-Type = %q, want application/xml", got)
	}
	if got := response.Header().Get("Cache-Control"); got != sitemapCacheControl {
		t.Fatalf("Cache-Control = %q, want %q", got, sitemapCacheControl)
	}

	body := response.Body.Bytes()
	for _, want := range []string{
		"<loc>https://abridged.example/</loc>",
		"<loc>https://abridged.example/creators</loc>",
		"<loc>https://abridged.example/series</loc>",
		"<loc>https://abridged.example/creator</loc>",
		"<loc>https://abridged.example/creator/show</loc>",
		"<loc>https://abridged.example/creator/show/episode-2-next</loc>",
	} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("sitemap missing %q:\n%s", want, body)
		}
	}
	for _, unwanted := range []string{"/api/", "/video/", "/thumb/", "/assets/", "/episode-1-start"} {
		if strings.Contains(string(body), unwanted) {
			t.Fatalf("sitemap contains %q:\n%s", unwanted, body)
		}
	}

	var set sitemapURLSet
	if err := xml.Unmarshal(body, &set); err != nil {
		t.Fatal(err)
	}
	if set.Xmlns != "http://www.sitemaps.org/schemas/sitemap/0.9" {
		t.Fatalf("xmlns = %q", set.Xmlns)
	}
	if len(set.URLs) < 9 {
		t.Fatalf("len(URLs) = %d, want at least 9", len(set.URLs))
	}

	etag := response.Header().Get("ETag")
	if etag == "" {
		t.Fatal("ETag is empty")
	}
	notModified := serveRouter(t, router, "/sitemap.xml", map[string]string{"If-None-Match": etag})
	if notModified.Code != http.StatusNotModified {
		t.Fatalf("conditional status = %d, want %d", notModified.Code, http.StatusNotModified)
	}
}

func TestSitemapDerivesBaseURLFromRequest(t *testing.T) {
	library := loadTestLibrary(t)
	router, err := NewRouter(library, Options{AppEnv: "test"})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	request.Host = "abridged.test"
	request.Header.Set("X-Forwarded-Proto", "https")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "https://abridged.test/creator/show") {
		t.Fatalf("derived base URL missing:\n%s", response.Body.String())
	}
}

func TestNormalizePublicBaseURL(t *testing.T) {
	valid := []struct {
		input string
		want  string
	}{
		{input: "", want: ""},
		{input: "https://abridged.example", want: "https://abridged.example"},
		{input: "https://abridged.example/", want: "https://abridged.example"},
		{input: "http://127.0.0.1:8080", want: "http://127.0.0.1:8080"},
	}
	for _, testCase := range valid {
		got, err := normalizePublicBaseURL(testCase.input)
		if err != nil {
			t.Fatalf("normalizePublicBaseURL(%q) returned an error: %v", testCase.input, err)
		}
		if got != testCase.want {
			t.Fatalf("normalizePublicBaseURL(%q) = %q, want %q", testCase.input, got, testCase.want)
		}
	}

	invalid := []string{
		"abridged.example",
		"ftp://abridged.example",
		"https://abridged.example/base",
		"https://user:pass@abridged.example",
		"https://abridged.example/?x=1",
	}
	for _, input := range invalid {
		if _, err := normalizePublicBaseURL(input); err == nil {
			t.Fatalf("normalizePublicBaseURL(%q) returned no error", input)
		}
	}
}

func TestSitemapDate(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{input: "2026-04-27", want: "2026-04-27"},
		{input: "20260427", want: "2026-04-27"},
		{input: "2026-04-27T12:34:56Z", want: "2026-04-27"},
		{input: "", want: ""},
		{input: "not-a-date", want: ""},
	}
	for _, testCase := range cases {
		if got := sitemapDate(testCase.input); got != testCase.want {
			t.Fatalf("sitemapDate(%q) = %q, want %q", testCase.input, got, testCase.want)
		}
	}
}
