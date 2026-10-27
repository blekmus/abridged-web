package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestInjectedMetadata(t *testing.T) {
	library := loadTestLibrary(t)
	router, err := NewRouter(library, Options{
		AppEnv:        "test",
		PublicBaseURL: "https://abridged.example",
	})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name      string
		path      string
		title     string
		canonical string
		ogType    string
		status    int
		noIndex   bool
	}{
		{
			name:      "home",
			path:      "/",
			title:     "Abridged Catalogue",
			canonical: "https://abridged.example/",
			ogType:    "website",
			status:    http.StatusOK,
		},
		{
			name:      "browse",
			path:      "/series",
			title:     "Shows | Abridged Catalogue",
			canonical: "https://abridged.example/series",
			ogType:    "website",
			status:    http.StatusOK,
		},
		{
			name:      "creators",
			path:      "/creators",
			title:     "Creators | Abridged Catalogue",
			canonical: "https://abridged.example/creators",
			ogType:    "website",
			status:    http.StatusOK,
		},
		{
			name:      "creator",
			path:      "/creator",
			title:     "Creator | Abridged Catalogue",
			canonical: "https://abridged.example/creator",
			ogType:    "website",
			status:    http.StatusOK,
		},
		{
			name:      "entry",
			path:      "/creator/show",
			title:     "Show | Abridged Catalogue",
			canonical: "https://abridged.example/creator/show",
			ogType:    "video.other",
			status:    http.StatusOK,
		},
		{
			name:      "default episode",
			path:      "/creator/show/episode-1-start",
			title:     "Show - Episode 1 - Start | Abridged Catalogue",
			canonical: "https://abridged.example/creator/show",
			ogType:    "video.episode",
			status:    http.StatusOK,
		},
		{
			name:      "episode",
			path:      "/creator/show/episode-2-next",
			title:     "Show - Episode 2 - Next | Abridged Catalogue",
			canonical: "https://abridged.example/creator/show/episode-2-next",
			ogType:    "video.episode",
			status:    http.StatusOK,
		},
		{
			name:    "not found",
			path:    "/missing",
			title:   "Not Found | Abridged Catalogue",
			ogType:  "website",
			status:  http.StatusNotFound,
			noIndex: true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			response := serveRouter(t, router, testCase.path, nil)
			if response.Code != testCase.status {
				t.Fatalf("status = %d, want %d", response.Code, testCase.status)
			}

			body := response.Body.String()
			if count := strings.Count(body, "<title>"); count != 1 {
				t.Fatalf("<title> count = %d, want 1:\n%s", count, body)
			}
			if !strings.Contains(body, "<title>"+testCase.title+"</title>") {
				t.Fatalf("missing title %q:\n%s", testCase.title, body)
			}
			if !strings.Contains(body, `<meta name="description" content="`) {
				t.Fatalf("missing description:\n%s", body)
			}

			if testCase.canonical == "" {
				if strings.Contains(body, `rel="canonical"`) {
					t.Fatalf("unexpected canonical URL:\n%s", body)
				}
			} else {
				if !strings.Contains(body, `<link rel="canonical" href="`+testCase.canonical+`">`) {
					t.Fatalf("missing canonical %q:\n%s", testCase.canonical, body)
				}
				if !strings.Contains(body, `<meta property="og:url" content="`+testCase.canonical+`">`) {
					t.Fatalf("missing og:url %q:\n%s", testCase.canonical, body)
				}
			}

			if !strings.Contains(body, `<meta property="og:type" content="`+testCase.ogType+`">`) {
				t.Fatalf("missing og:type %q:\n%s", testCase.ogType, body)
			}
			if !strings.Contains(body, `<meta name="twitter:card" content="summary_large_image">`) {
				t.Fatalf("missing twitter card:\n%s", body)
			}
			if !strings.Contains(body, `<meta property="og:image" content="https://abridged.example/`) {
				t.Fatalf("missing absolute og:image:\n%s", body)
			}

			hasNoIndex := strings.Contains(body, `<meta name="robots" content="noindex, nofollow">`)
			if hasNoIndex != testCase.noIndex {
				t.Fatalf("noindex = %v, want %v:\n%s", hasNoIndex, testCase.noIndex, body)
			}
		})
	}
}

func TestMetadataUsesPosterImageAndEscapesValues(t *testing.T) {
	library := loadTestLibrary(t)
	router, err := NewRouter(library, Options{
		AppEnv:        "test",
		PublicBaseURL: "https://abridged.example",
	})
	if err != nil {
		t.Fatal(err)
	}

	entryResponse := serveRouter(t, router, "/creator/show", nil)
	if !strings.Contains(entryResponse.Body.String(), `content="https://abridged.example/thumb/`) {
		t.Fatalf("entry og:image is not the poster:\n%s", entryResponse.Body.String())
	}

	rendered := renderMeta(pageMeta{
		Title:       `A "quote" & <tag>`,
		Description: `Desc & <b>`,
		Canonical:   `https://example.com/?a=1&b=2`,
		Image:       `https://example.com/i.png`,
		ImageAlt:    `alt "x"`,
	})
	if strings.Contains(rendered, "<tag>") || strings.Contains(rendered, "<b>") {
		t.Fatalf("meta was not HTML-escaped:\n%s", rendered)
	}
	for _, want := range []string{"&#34;", "&amp;", "&lt;"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("meta missing escaped value %q:\n%s", want, rendered)
		}
	}
}

func TestInjectHead(t *testing.T) {
	injected := injectHead("<head><title>Old</title></head>", "<title>New</title>\n")
	if strings.Contains(injected, "Old") {
		t.Fatalf("old title was not replaced: %s", injected)
	}
	if strings.Count(injected, "<title>") != 1 || !strings.Contains(injected, "<title>New</title>") {
		t.Fatalf("new title was not injected: %s", injected)
	}

	placeholder := injectHead("<head><!--app-meta--></head>", "<title>New</title>\n")
	if !strings.Contains(placeholder, "<title>New</title>") {
		t.Fatalf("placeholder was not replaced: %s", placeholder)
	}

	fallback := injectHead("<head></head>", "<title>New</title>\n")
	if !strings.Contains(fallback, "<title>New</title>") {
		t.Fatalf("head fallback did not inject meta: %s", fallback)
	}
}

func TestIndexHTMLRedirectsToHome(t *testing.T) {
	library := loadTestLibrary(t)
	router, err := NewRouter(library, Options{AppEnv: "test"})
	if err != nil {
		t.Fatal(err)
	}

	response := serveRouter(t, router, "/index.html", nil)
	if response.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMovedPermanently)
	}
	if got := response.Header().Get("Location"); got != "/" {
		t.Fatalf("Location = %q, want /", got)
	}
}
