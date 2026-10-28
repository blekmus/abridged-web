package server

import (
	"net/http"
	"strings"
	"testing"
	"testing/fstest"
)

func TestPrerenderedShellRoutes(t *testing.T) {
	const shell = `<div class="page-shell">Generated homepage</div>`
	handler := &spaHandler{staticFS: fstest.MapFS{
		"index.html":     {Data: []byte(`<html><body><div id="app"><!--app-shell--></div></body></html>`)},
		"app-shell.html": {Data: []byte(shell)},
	}}

	for _, path := range []string{"/", "/series", "/shorts", "/creators", "/creator/video", "/index.html"} {
		t.Run(path, func(t *testing.T) {
			response := performSPARequest(t, handler, path, nil)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", response.Code)
			}
			if got := strings.Contains(response.Body.String(), shell); got != (path == "/") {
				t.Fatalf("homepage shell present = %v for %s", got, path)
			}
			if strings.Contains(response.Body.String(), "<!--app-shell-->") {
				t.Fatal("unreplaced shell placeholder")
			}
		})
	}
}

func TestMissingPrerenderedShell(t *testing.T) {
	handler := &spaHandler{staticFS: fstest.MapFS{
		"index.html": {Data: []byte(`<div id="app"><!--app-shell--></div>`)},
	}}
	response := performSPARequest(t, handler, "/", nil)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 for incomplete frontend build", response.Code)
	}
}
